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
	requireErrorContains(t, err, "404")
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
			// The hint is the call shape and where to find the rest, not the
			// whole usage block with every global flag in it.
			if n := len(lines(stderr)); n > 3 {
				t.Errorf("usage hint is %d lines, want at most 3:\n%s", n, stderr)
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
}

func TestHelpIsNotAnError(t *testing.T) {
	stdout, _, err := runCmd(t, nil, "team", "view", "--help")
	requireNoError(t, err, "")
	requireContains(t, stdout, "Usage:")
}
