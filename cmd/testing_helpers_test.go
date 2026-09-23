package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"
)

// recordedRequest captures what a command actually sent to the API.
type recordedRequest struct {
	Method  string
	Path    string
	Query   string
	Headers http.Header
}

// fakeState holds the mutable bookkeeping for one fake server. It is kept in a
// package-level registry so that newFakeTBA can return a plain
// *httptest.Server, matching how tests want to use it.
type fakeState struct {
	mu       sync.Mutex
	bodies   map[string][]byte
	etags    map[string]string
	statuses map[string]int
	delays   map[string]time.Duration
	requests []recordedRequest
}

var (
	fakeRegistryMu sync.Mutex
	fakeRegistry   = map[string]*fakeState{}
)

// newFakeTBA starts a stand-in for the TBA API.
//
// routes maps a request path (the part after the base URL, e.g. "/team/frc177")
// to the response body. Values may be a string or []byte containing raw JSON,
// or any value that will be marshalled to JSON. Unknown paths get a 404 with a
// TBA-shaped error body. Every request is recorded so tests can assert on
// headers such as X-TBA-Auth-Key, User-Agent and If-None-Match.
func newFakeTBA(t *testing.T, routes map[string]any) *httptest.Server {
	t.Helper()

	st := &fakeState{
		bodies:   make(map[string][]byte, len(routes)),
		etags:    make(map[string]string),
		statuses: make(map[string]int),
		delays:   make(map[string]time.Duration),
	}
	for path, v := range routes {
		st.bodies[path] = toJSONBytes(t, v)
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		st.mu.Lock()
		st.requests = append(st.requests, recordedRequest{
			Method:  r.Method,
			Path:    r.URL.Path,
			Query:   r.URL.RawQuery,
			Headers: r.Header.Clone(),
		})
		body, ok := st.bodies[r.URL.Path]
		etag := st.etags[r.URL.Path]
		status := st.statuses[r.URL.Path]
		delay := st.delays[r.URL.Path]
		st.mu.Unlock()

		if delay > 0 {
			select {
			case <-time.After(delay):
			case <-r.Context().Done():
				return
			}
		}

		w.Header().Set("Content-Type", "application/json")
		if status != 0 {
			w.WriteHeader(status)
			_, _ = io.WriteString(w, `{"Error":"status `+strconv.Itoa(status)+`"}`)
			return
		}
		if !ok {
			w.WriteHeader(http.StatusNotFound)
			_, _ = io.WriteString(w, `{"Error":"`+r.URL.Path+` not found"}`)
			return
		}
		if etag != "" {
			w.Header().Set("ETag", etag)
			if r.Header.Get("If-None-Match") == etag {
				w.WriteHeader(http.StatusNotModified)
				return
			}
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(body)
	}))
	t.Cleanup(srv.Close)

	fakeRegistryMu.Lock()
	fakeRegistry[srv.URL] = st
	fakeRegistryMu.Unlock()
	t.Cleanup(func() {
		fakeRegistryMu.Lock()
		delete(fakeRegistry, srv.URL)
		fakeRegistryMu.Unlock()
	})

	return srv
}

func toJSONBytes(t *testing.T, v any) []byte {
	t.Helper()
	switch b := v.(type) {
	case nil:
		return []byte("null")
	case string:
		return []byte(b)
	case []byte:
		return b
	case json.RawMessage:
		return b
	default:
		out, err := json.Marshal(v)
		if err != nil {
			t.Fatalf("marshalling fixture: %v", err)
		}
		return out
	}
}

func stateFor(t *testing.T, srv *httptest.Server) *fakeState {
	t.Helper()
	fakeRegistryMu.Lock()
	defer fakeRegistryMu.Unlock()
	st, ok := fakeRegistry[srv.URL]
	if !ok {
		t.Fatalf("server %s was not created by newFakeTBA", srv.URL)
	}
	return st
}

// setETag makes the fake serve an ETag for path and answer 304 when the client
// sends a matching If-None-Match.
func setETag(t *testing.T, srv *httptest.Server, path, etag string) {
	t.Helper()
	st := stateFor(t, srv)
	st.mu.Lock()
	defer st.mu.Unlock()
	st.etags[path] = etag
}

// setStatus makes the fake answer path with an HTTP status instead of a body,
// so tests can exercise the retry and error paths.
func setStatus(t *testing.T, srv *httptest.Server, path string, code int) {
	t.Helper()
	st := stateFor(t, srv)
	st.mu.Lock()
	defer st.mu.Unlock()
	st.statuses[path] = code
}

// setDelay makes the fake hold a request for d before answering, or until the
// client gives up, which is how the --timeout tests trip the per-request
// deadline.
func setDelay(t *testing.T, srv *httptest.Server, path string, d time.Duration) {
	t.Helper()
	st := stateFor(t, srv)
	st.mu.Lock()
	defer st.mu.Unlock()
	st.delays[path] = d
}

// requestsTo returns every request the fake has received so far.
func requestsTo(t *testing.T, srv *httptest.Server) []recordedRequest {
	t.Helper()
	st := stateFor(t, srv)
	st.mu.Lock()
	defer st.mu.Unlock()
	out := make([]recordedRequest, len(st.requests))
	copy(out, st.requests)
	return out
}

// requestPaths returns the paths of every request the fake has received.
func requestPaths(t *testing.T, srv *httptest.Server) []string {
	t.Helper()
	reqs := requestsTo(t, srv)
	paths := make([]string, len(reqs))
	for i, r := range reqs {
		paths[i] = r.Path
	}
	return paths
}

// runCmd builds a fresh command tree, points it at srv and runs it with args.
// Auth, cache and config all live in per-test temporary state.
func runCmd(t *testing.T, srv *httptest.Server, args ...string) (stdout, stderr string, err error) {
	t.Helper()
	return runCmdStdin(t, srv, "", args...)
}

// fixedClock stops the clock at now, which is what makes a countdown, a
// relative time or an "is this event over" the same on every run.
//
// Its sleep returns at once: a test that pins the clock and then waits for
// real time would wait forever, since nothing is going to move.
func fixedClock(now time.Time) clock {
	return clock{
		now:   func() time.Time { return now },
		sleep: func(ctx context.Context, _ time.Duration) error { return ctx.Err() },
	}
}

// runCmdAt is runCmd with the clock stopped at now.
//
// The clock belongs to the tree this one run builds, so two tests pinning
// different moments cannot see each other's, and `go test -race` has nothing
// to say about it.
func runCmdAt(t *testing.T, srv *httptest.Server, now time.Time, args ...string) (stdout, stderr string, err error) {
	t.Helper()
	return runCmdWith(t, srv, fixedClock(now), &bytes.Buffer{}, args...)
}

// runCmdWith runs against a clock of the caller's own making, which is how the
// watch tests drive a two-hour poll loop in microseconds.
func runCmdWith(t *testing.T, srv *httptest.Server, clk clock, out interface {
	io.Writer
	String() string
}, args ...string) (stdout, stderr string, err error) {
	t.Helper()
	return runCmdOnWith(t, srv, out, "", clk, args...)
}

// runCmdStdin is runCmd with a canned stdin, for commands that prompt.
func runCmdStdin(t *testing.T, srv *httptest.Server, stdin string, args ...string) (stdout, stderr string, err error) {
	t.Helper()
	return runCmdOn(t, srv, &bytes.Buffer{}, stdin, args...)
}

// terminalBuffer is a buffer that claims to be a terminal, so tests can walk
// the code paths a real user at a terminal gets: table by default, color on.
type terminalBuffer struct{ bytes.Buffer }

func (*terminalBuffer) IsTerminal() bool { return true }

// runCmdTTY is runCmd with stdout pretending to be a terminal.
func runCmdTTY(t *testing.T, srv *httptest.Server, args ...string) (stdout, stderr string, err error) {
	t.Helper()
	return runCmdOn(t, srv, &terminalBuffer{}, "", args...)
}

// runCmdOn runs a fresh command tree with out as its stdout, against the real
// clock.
func runCmdOn(t *testing.T, srv *httptest.Server, out interface {
	io.Writer
	String() string
}, stdin string, args ...string) (stdout, stderr string, err error) {
	t.Helper()
	return runCmdOnWith(t, srv, out, stdin, systemClock(), args...)
}

// runCmdOnWith is runCmdOn against a given clock.
func runCmdOnWith(t *testing.T, srv *httptest.Server, out interface {
	io.Writer
	String() string
}, stdin string, clk clock, args ...string) (stdout, stderr string, err error) {
	t.Helper()

	// Each run gets throwaway auth/cache/config state. A test that needs the
	// same directory across two runs (e.g. cache revalidation) can set the
	// variable itself beforehand and it is left alone.
	setEnvUnlessSet(t, "TBA_AUTH_KEY", "test-key")
	setEnvUnlessSet(t, "TBA_CACHE_DIR", t.TempDir())
	setEnvUnlessSet(t, "TBA_CONFIG_DIR", t.TempDir())

	root := newRootCmdWithClock(clk)
	var errBuf bytes.Buffer
	root.SetOut(out)
	root.SetErr(&errBuf)
	root.SetIn(strings.NewReader(stdin))

	full := args
	if srv != nil {
		full = append([]string{"--base-url", srv.URL}, args...)
	}
	root.SetArgs(full)

	err = Run(context.Background(), root)
	return out.String(), errBuf.String(), err
}

func setEnvUnlessSet(t *testing.T, key, value string) {
	t.Helper()
	if _, ok := os.LookupEnv(key); ok {
		return
	}
	t.Setenv(key, value)
}

// decodeJSON unmarshals stdout, failing the test if it is not valid JSON.
func decodeJSON(t *testing.T, s string) any {
	t.Helper()
	var v any
	if err := json.Unmarshal([]byte(s), &v); err != nil {
		t.Fatalf("output is not valid JSON: %v\n---\n%s", err, s)
	}
	return v
}

// lines splits output into lines, dropping the trailing empty element.
func lines(s string) []string {
	return strings.Split(strings.TrimRight(s, "\n"), "\n")
}

func requireNoError(t *testing.T, err error, stderr string) {
	t.Helper()
	if err != nil {
		t.Fatalf("command failed: %v\nstderr: %s", err, stderr)
	}
}

func contains(haystack []string, needle string) bool {
	for _, s := range haystack {
		if s == needle {
			return true
		}
	}
	return false
}

// subcommandNames returns the names registered under a top-level command.
func subcommandNames(t *testing.T, parent string) []string {
	t.Helper()
	for _, c := range NewRootCmd().Commands() {
		if c.Name() != parent {
			continue
		}
		names := make([]string, 0, len(c.Commands()))
		for _, sub := range c.Commands() {
			names = append(names, sub.Name())
		}
		return names
	}
	t.Fatalf("no top-level command named %q", parent)
	return nil
}

// requireErrorContains checks an error message without pinning its wording.
func requireErrorContains(t *testing.T, err error, want string) {
	t.Helper()
	if err == nil {
		t.Fatalf("want an error mentioning %q, got nil", want)
	}
	if !strings.Contains(err.Error(), want) {
		t.Errorf("error %q does not mention %q", err.Error(), want)
	}
}

func requireContains(t *testing.T, got, want string) {
	t.Helper()
	if !strings.Contains(got, want) {
		t.Errorf("output missing %q\n---\n%s", want, got)
	}
}

// thisYear is the calendar year now, which is what a command's default --year
// resolves to when nothing tells it otherwise.
func thisYear() int { return currentYear(time.Now()) }
