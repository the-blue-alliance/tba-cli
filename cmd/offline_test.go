package cmd

import (
	"net/http"
	"os"
	"path/filepath"
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
	// The data is identical to a live run's, so the age of the copy is the
	// only thing that can tell the user which one they are reading.
	if !strings.Contains(errOut, "note: offline: /team/frc177 from cache (") {
		t.Errorf("stderr should date the cached copy:\n%s", errOut)
	}
	if !strings.Contains(errOut, "ago)") {
		t.Errorf("the offline note should carry an age:\n%s", errOut)
	}
	if strings.Contains(out, "offline") {
		t.Errorf("the note belongs on stderr:\n%s", out)
	}
}

// Reading one's own cache is not a request to anyone, so it needs no
// credentials. `tba --offline ...` over a warm cache on a machine with no key
// exited 4 "not authenticated", which is the one situation where offline is
// most worth having.
func TestOfflineNeedsNoAPIKey(t *testing.T) {
	sharedCacheDir(t)
	t.Setenv("TBA_CONFIG_DIR", t.TempDir())
	t.Setenv("TBA_AUTH_KEY", "test-key")
	srv := newFakeTBA(t, map[string]any{"/team/frc177": teamFRC177JSON})

	first, stderr, err := runCmd(t, srv, "team", "view", "177", "--format", "json")
	requireNoError(t, err, stderr)

	// Same cache and config directory, no key anywhere.
	t.Setenv("TBA_AUTH_KEY", "")
	out, stderr, err := runCmd(t, srv, "team", "view", "177", "--format", "json", "--offline")
	requireNoError(t, err, stderr)
	if out != first {
		t.Errorf("offline output differs:\n%s\n---\n%s", first, out)
	}

	// Without --offline the same run still asks for a key, because then it
	// really is about to make a request.
	_, _, err = runCmd(t, srv, "team", "view", "177")
	if got := clierr.ExitCode(err); got != clierr.ExitAuth {
		t.Errorf("exit code = %d (err %v), want %d", got, err, clierr.ExitAuth)
	}
}

func TestOfflineFailsOnAnUncachedPath(t *testing.T) {
	sharedCacheDir(t)
	srv := newFakeTBA(t, map[string]any{"/team/frc177": teamFRC177JSON})

	out, _, err := runCmd(t, srv, "team", "view", "177", "--offline")
	if err == nil {
		t.Fatal("want an error for a path that was never fetched")
	}
	// "no copy of that here" is the answer a 404 gives, and a script can act
	// on it the same way; exit 1 said the run itself had gone wrong.
	if got := clierr.ExitCode(err); got != clierr.ExitNotFound {
		t.Errorf("exit code = %d, want %d", got, clierr.ExitNotFound)
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

// A cache directory that cannot be opened has to say so. Running cacheless
// after a silent failure makes --offline report "not cached" for everything,
// which sends the user looking for the wrong problem.
func TestABrokenCacheDirectoryIsReportedRatherThanIgnored(t *testing.T) {
	blocked := filepath.Join(t.TempDir(), "not-a-directory")
	if err := os.WriteFile(blocked, []byte("in the way\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("TBA_CACHE_DIR", blocked)
	srv := newFakeTBA(t, map[string]any{"/team/frc177": teamFRC177JSON})

	_, _, err := runCmd(t, srv, "team", "view", "177", "--offline")
	requireErrorContains(t, err, "opening the response cache")
	if strings.Contains(err.Error(), "not cached") {
		t.Errorf("a broken cache dir should not read as a cache miss: %v", err)
	}
}

func TestOfflineIsAPersistentFlag(t *testing.T) {
	root := NewRootCmd()
	if root.PersistentFlags().Lookup("offline") == nil {
		t.Fatal("--offline should be available to every command")
	}
}
