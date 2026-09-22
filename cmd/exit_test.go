package cmd

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/the-blue-alliance/tba-cli/internal/clierr"
)

// requireExitCode runs a command and asserts the exit code main would use.
func requireExitCode(t *testing.T, want int, srv *httptest.Server, args ...string) error {
	t.Helper()
	_, _, err := runCmd(t, srv, args...)
	if got := clierr.ExitCode(err); got != want {
		t.Fatalf("exit code for %v = %d (err %v), want %d", args, got, err, want)
	}
	return err
}

func TestExitCodeZeroOnSuccess(t *testing.T) {
	srv := newFakeTBA(t, map[string]any{"/status": apiStatusJSON})
	_ = requireExitCode(t, clierr.ExitOK, srv, "status")
}

func TestExitCodeTwoForUsageMistakes(t *testing.T) {
	srv := newFakeTBA(t, map[string]any{"/status": apiStatusJSON})
	cases := [][]string{
		{"status", "--frmat", "json"},
		{"status", "--format", "xml"},
		{"team", "list", "--year", "not-a-year"},
		{"teem", "view", "177"},
		{"team", "view", "177", "1073"},
		{"event", "view", "not-a-key"},
		{"match", "view", "not-a-key"},
		{"district", "list", "--jq", ".[].key", "--format", "csv"},
	}
	for _, args := range cases {
		t.Run(args[0]+" "+args[1], func(t *testing.T) {
			_ = requireExitCode(t, clierr.ExitUsage, srv, args...)
		})
	}
}

func TestExitCodeFourWhenNoKeyIsConfigured(t *testing.T) {
	t.Setenv("TBA_AUTH_KEY", "")
	t.Setenv("TBA_CONFIG_DIR", t.TempDir())
	srv := newFakeTBA(t, map[string]any{"/status": apiStatusJSON})
	_ = requireExitCode(t, clierr.ExitAuth, srv, "status")
}

func TestExitCodeFourOnHTTP401(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"Error":"Invalid X-TBA-Auth-Key"}`))
	}))
	t.Cleanup(srv.Close)

	t.Setenv("TBA_AUTH_KEY", "bogus")
	t.Setenv("TBA_CACHE_DIR", t.TempDir())
	t.Setenv("TBA_CONFIG_DIR", t.TempDir())

	root := NewRootCmd()
	root.SetOut(nopWriter{})
	root.SetErr(nopWriter{})
	root.SetArgs([]string{"--base-url", srv.URL, "status"})
	err := Run(context.Background(), root)
	if got := clierr.ExitCode(err); got != clierr.ExitAuth {
		t.Fatalf("exit code = %d (err %v), want %d", got, err, clierr.ExitAuth)
	}
	requireErrorContains(t, err, "not authenticated for "+srv.URL)
	requireErrorContains(t, err, "HTTP 401")
	requireErrorContains(t, err, "tba auth login")
}

func TestExitCodeFiveOnHTTP404(t *testing.T) {
	srv := newFakeTBA(t, map[string]any{})
	err := requireExitCode(t, clierr.ExitNotFound, srv, "team", "view", "99999")
	// The API's own sentence, not the JSON document it arrived in, and said
	// once: "not found: /team/frc99999 not found" was the news twice over.
	if got, want := err.Error(), "/team/frc99999 not found"; got != want {
		t.Errorf("error = %q, want %q", got, want)
	}
	if strings.Contains(err.Error(), `{"Error"`) {
		t.Errorf("the raw body leaked into the message: %v", err)
	}
}

func TestExitCodeOneOnServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("boom"))
	}))
	t.Cleanup(srv.Close)

	t.Setenv("TBA_CACHE_DIR", t.TempDir())
	t.Setenv("TBA_CONFIG_DIR", t.TempDir())
	t.Setenv("TBA_AUTH_KEY", "test-key")

	root := NewRootCmd()
	root.SetOut(nopWriter{})
	root.SetErr(nopWriter{})
	root.SetArgs([]string{"--base-url", srv.URL, "status"})
	err := Run(context.Background(), root)
	if got := clierr.ExitCode(err); got != clierr.ExitFailure {
		t.Fatalf("exit code = %d (err %v), want %d", got, err, clierr.ExitFailure)
	}
}

type nopWriter struct{}

func (nopWriter) Write(p []byte) (int, error) { return len(p), nil }

func TestUsageErrorsPrintUsage(t *testing.T) {
	cases := []struct {
		name    string
		args    []string
		command string
	}{
		{"unknown flag", []string{"status", "--frmat", "json"}, "tba status"},
		{"bad format", []string{"status", "--format", "xml"}, "tba status"},
		{"too many args", []string{"team", "view", "177", "1073"}, "tba team view"},
		{"malformed event key", []string{"event", "view", "not-a-key"}, "tba event view"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			srv := newFakeTBA(t, map[string]any{"/status": apiStatusJSON})
			stdout, stderr, err := runCmd(t, srv, tc.args...)
			if err == nil {
				t.Fatalf("want an error for %v", tc.args)
			}
			if got := clierr.ExitCode(err); got != clierr.ExitUsage {
				t.Fatalf("exit code = %d, want %d", got, clierr.ExitUsage)
			}
			requireContains(t, stderr, "Usage:")
			requireContains(t, stderr, "Run '"+tc.command+" --help' for usage.")
			if stdout != "" {
				t.Errorf("usage must not go to stdout, got:\n%s", stdout)
			}
			// What went wrong comes first. It used to be printed below the
			// call shape and the "run --help" line, so on a small terminal
			// the one line worth reading was the one that scrolled away.
			got := lines(stderr)
			if len(got) == 0 || !strings.HasPrefix(got[0], "Error: ") {
				t.Errorf("stderr should open with the error line:\n%s", stderr)
			}
			if !strings.Contains(got[0], err.Error()) {
				t.Errorf("first line = %q, want it to carry %q", got[0], err.Error())
			}
			// Then the call shape and where to find the rest — not the whole
			// usage block with every global flag in it.
			if n := len(got); n > 4 {
				t.Errorf("the error and its hint are %d lines, want at most 4:\n%s", n, stderr)
			}
			if strings.Contains(stderr, "Global Flags:") || strings.Contains(stderr, "--no-cache") {
				t.Errorf("the flag listing belongs in --help, not in the error:\n%s", stderr)
			}
		})
	}
}

func TestRuntimeErrorsPrintNoUsage(t *testing.T) {
	srv := newFakeTBA(t, map[string]any{})
	stdout, stderr, err := runCmd(t, srv, "team", "view", "99999")
	if err == nil {
		t.Fatal("want a 404 error")
	}
	if got := clierr.ExitCode(err); got != clierr.ExitNotFound {
		t.Fatalf("exit code = %d, want %d", got, clierr.ExitNotFound)
	}
	if strings.Contains(stderr, "Usage:") {
		t.Errorf("a 404 should not print usage:\n%s", stderr)
	}
	if stdout != "" {
		t.Errorf("nothing should reach stdout, got:\n%s", stdout)
	}
	// Reporting moved out of main and into Run, so it has to happen here —
	// and exactly once, not once in each.
	if n := strings.Count(stderr, "Error: "); n != 1 {
		t.Errorf("stderr carries %d error lines, want 1:\n%s", n, stderr)
	}
	requireContains(t, stderr, err.Error())
}

func TestHelpIsNotAnError(t *testing.T) {
	stdout, _, err := runCmd(t, nil, "team", "view", "--help")
	requireNoError(t, err, "")
	requireContains(t, stdout, "Usage:")
}
