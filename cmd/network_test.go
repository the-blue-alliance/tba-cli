package cmd

import (
	"net/http"
	"strings"
	"testing"
	"time"
)

func TestRootHelpDocumentsTheNetworkFlags(t *testing.T) {
	out, _, err := runCmd(t, nil, "--help")
	requireNoError(t, err, "")

	requireContains(t, out, "--timeout duration")
	requireContains(t, out, "Per-request timeout (e.g. 10s, 1m)")
	requireContains(t, out, "(default 10s)")
	requireContains(t, out, "--retries int")
	requireContains(t, out, "Retry attempts for 429/5xx/network errors; 0 disables")
	requireContains(t, out, "(default 3)")
}

func TestRetriesZeroMakesASingleAttempt(t *testing.T) {
	srv := newFakeTBA(t, map[string]any{"/status": apiStatusJSON})
	setStatus(t, srv, "/status", http.StatusServiceUnavailable)

	_, _, err := runCmd(t, srv, "status", "--retries", "0")
	if err == nil {
		t.Fatal("want an error for a 503")
	}
	if !strings.Contains(err.Error(), "API error 503:") {
		t.Errorf("error = %v, want the plain single-attempt wording", err)
	}
	if n := len(requestsTo(t, srv)); n != 1 {
		t.Errorf("made %d requests with --retries 0, want 1", n)
	}
}

func TestRetriesFlagIsPassedToTheClient(t *testing.T) {
	srv := newFakeTBA(t, map[string]any{"/status": apiStatusJSON})
	setStatus(t, srv, "/status", http.StatusServiceUnavailable)

	_, _, err := runCmd(t, srv, "status", "--retries", "1")
	if err == nil {
		t.Fatal("want an error for a 503")
	}
	if !strings.Contains(err.Error(), "after 2 attempts") {
		t.Errorf("error = %v, want it to mention 2 attempts", err)
	}
	if n := len(requestsTo(t, srv)); n != 2 {
		t.Errorf("made %d requests with --retries 1, want 2", n)
	}
}

func TestNotFoundIsNotRetried(t *testing.T) {
	srv := newFakeTBA(t, map[string]any{})

	_, _, err := runCmd(t, srv, "team", "view", "177")
	if err == nil {
		t.Fatal("want an error for an unknown path")
	}
	if n := len(requestsTo(t, srv)); n != 1 {
		t.Errorf("made %d requests for a 404, want 1", n)
	}
}

func TestTimeoutFlagBoundsTheRequest(t *testing.T) {
	srv := newFakeTBA(t, map[string]any{"/status": apiStatusJSON})
	setDelay(t, srv, "/status", 30*time.Second)

	start := time.Now()
	_, _, err := runCmd(t, srv, "status", "--timeout", "50ms", "--retries", "0")
	if err == nil {
		t.Fatal("want a timeout error")
	}
	if !strings.Contains(err.Error(), "request failed") {
		t.Errorf("error = %v", err)
	}
	if elapsed := time.Since(start); elapsed > 5*time.Second {
		t.Errorf("--timeout did not bound the request: took %v", elapsed)
	}
}

func TestInvalidTimeoutIsAFlagError(t *testing.T) {
	srv := newFakeTBA(t, map[string]any{"/status": apiStatusJSON})
	_, _, err := runCmd(t, srv, "status", "--timeout", "soon")
	if err == nil {
		t.Fatal("want an error for a non-duration --timeout")
	}
	if !strings.Contains(err.Error(), "timeout") {
		t.Errorf("error = %v", err)
	}
}

func TestErrorsGoToStderrNotStdout(t *testing.T) {
	srv := newFakeTBA(t, map[string]any{})
	stdout, _, err := runCmd(t, srv, "team", "view", "177")
	if err == nil {
		t.Fatal("want an error")
	}
	if stdout != "" {
		t.Errorf("stdout should stay empty on failure, got %q", stdout)
	}
}
