package cmd

import (
	"net/http"
	"strings"
	"testing"

	"github.com/the-blue-alliance/tba-cli/internal/clierr"
)

// sharedCacheDir pins one cache directory for every run in a test, so that a
// second invocation sees what the first one cached.
func sharedCacheDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("TBA_CACHE_DIR", dir)
	return dir
}

func TestStaleFallbackNoteGoesToStderrAndStdoutStaysJSON(t *testing.T) {
	sharedCacheDir(t)
	srv := newFakeTBA(t, map[string]any{"/team/frc177": teamFRC177JSON})

	first, _, err := runCmd(t, srv, "team", "view", "177", "--format", "json")
	requireNoError(t, err, "")

	setStatus(t, srv, "/team/frc177", http.StatusServiceUnavailable)
	out, errOut, err := runCmd(t, srv, "team", "view", "177", "--format", "json", "--retries", "0")
	requireNoError(t, err, errOut)

	if out != first {
		t.Errorf("stdout changed when falling back:\n%s\n---\n%s", first, out)
	}
	if _, ok := decodeJSON(t, out).(map[string]any); !ok {
		t.Errorf("stdout should be clean JSON:\n%s", out)
	}
	if !strings.Contains(errOut, "note: /team/frc177 unavailable (HTTP 503); using cached copy from ") {
		t.Errorf("stderr note missing or reworded:\n%s", errOut)
	}
	if !strings.HasSuffix(strings.TrimRight(errOut, "\n"), "ago") {
		t.Errorf("the note should end with an age:\n%s", errOut)
	}
}

func TestStaleFallbackNeedsACachedCopy(t *testing.T) {
	sharedCacheDir(t)
	srv := newFakeTBA(t, map[string]any{"/team/frc177": teamFRC177JSON})
	setStatus(t, srv, "/team/frc177", http.StatusServiceUnavailable)

	out, errOut, err := runCmd(t, srv, "team", "view", "177", "--retries", "0")
	if err == nil {
		t.Fatal("want an error when there is nothing cached to fall back on")
	}
	if clierr.ExitCode(err) != clierr.ExitFailure {
		t.Errorf("exit code = %d, want %d", clierr.ExitCode(err), clierr.ExitFailure)
	}
	if out != "" {
		t.Errorf("stdout should be empty on failure:\n%s", out)
	}
	if strings.Contains(errOut, "cached copy") {
		t.Errorf("no cached copy exists, so nothing should claim one:\n%s", errOut)
	}
}

func TestNoStaleFallbackOnA404(t *testing.T) {
	sharedCacheDir(t)
	srv := newFakeTBA(t, map[string]any{"/team/frc177": teamFRC177JSON})
	if _, _, err := runCmd(t, srv, "team", "view", "177"); err != nil {
		t.Fatalf("priming the cache: %v", err)
	}

	setStatus(t, srv, "/team/frc177", http.StatusNotFound)
	out, errOut, err := runCmd(t, srv, "team", "view", "177")
	if err == nil {
		t.Fatal("a 404 must surface, not be masked by the cache")
	}
	if clierr.ExitCode(err) != clierr.ExitNotFound {
		t.Errorf("exit code = %d, want %d", clierr.ExitCode(err), clierr.ExitNotFound)
	}
	if out != "" {
		t.Errorf("stdout should be empty:\n%s", out)
	}
	if strings.Contains(errOut, "cached copy") {
		t.Errorf("stderr should not mention a cached copy:\n%s", errOut)
	}
}

func TestOfflineAnswersFromTheCacheWithoutAnyRequest(t *testing.T) {
	sharedCacheDir(t)
	srv := newFakeTBA(t, map[string]any{"/team/frc177": teamFRC177JSON})
	setETag(t, srv, "/team/frc177", `"etag-1"`)

	first, _, err := runCmd(t, srv, "team", "view", "177", "--format", "json")
	requireNoError(t, err, "")
	if n := len(requestPaths(t, srv)); n != 1 {
		t.Fatalf("priming made %d requests, want 1", n)
	}

	out, errOut, err := runCmd(t, srv, "team", "view", "177", "--format", "json", "--offline")
	requireNoError(t, err, errOut)
	if out != first {
		t.Errorf("offline output differs:\n%s\n---\n%s", first, out)
	}
	if n := len(requestPaths(t, srv)); n != 1 {
		t.Errorf("--offline reached the network: %v", requestPaths(t, srv))
	}
}

func TestOfflineFailsOnAnUncachedPath(t *testing.T) {
	sharedCacheDir(t)
	srv := newFakeTBA(t, map[string]any{"/team/frc177": teamFRC177JSON})

	out, _, err := runCmd(t, srv, "team", "view", "177", "--offline")
	if err == nil {
		t.Fatal("want an error for a path that was never fetched")
	}
	if clierr.ExitCode(err) != clierr.ExitFailure {
		t.Errorf("exit code = %d, want %d", clierr.ExitCode(err), clierr.ExitFailure)
	}
	requireErrorContains(t, err, "not cached: /team/frc177 (run without --offline to fetch)")
	if out != "" {
		t.Errorf("stdout should be empty:\n%s", out)
	}
	if n := len(requestPaths(t, srv)); n != 0 {
		t.Errorf("--offline reached the network: %v", requestPaths(t, srv))
	}
}

func TestOfflineWithNoCacheIsAUsageError(t *testing.T) {
	sharedCacheDir(t)
	srv := newFakeTBA(t, map[string]any{"/team/frc177": teamFRC177JSON})

	err := requireExitCode(t, clierr.ExitUsage, srv, "team", "view", "177", "--offline", "--no-cache")
	requireErrorContains(t, err, "--offline and --no-cache")
	if n := len(requestPaths(t, srv)); n != 0 {
		t.Errorf("a usage error should not make a request: %v", requestPaths(t, srv))
	}
}

func TestOfflineIsAPersistentFlag(t *testing.T) {
	root := NewRootCmd()
	if root.PersistentFlags().Lookup("offline") == nil {
		t.Fatal("--offline should be available to every command")
	}
}
