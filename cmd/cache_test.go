package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCacheInfoOnAnEmptyCache(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("TBA_CACHE_DIR", dir)

	out, _, err := runCmd(t, nil, "cache", "info", "--format", "table")
	requireNoError(t, err, "")

	requireContains(t, out, "Directory:    "+dir)
	requireContains(t, out, "Entries:      0")
	requireContains(t, out, "Size:         0 B")
}

func TestCacheInfoJSON(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("TBA_CACHE_DIR", dir)

	srv := newFakeTBA(t, map[string]any{"/status": apiStatusJSON})
	if _, _, err := runCmd(t, srv, "status"); err != nil {
		t.Fatalf("status: %v", err)
	}

	out, _, err := runCmd(t, nil, "cache", "info", "--format", "json")
	requireNoError(t, err, "")

	obj, ok := decodeJSON(t, out).(map[string]any)
	if !ok {
		t.Fatalf("want a JSON object, got:\n%s", out)
	}
	if obj["directory"] != dir {
		t.Errorf("directory = %v, want %v", obj["directory"], dir)
	}
	if obj["entries"] != float64(1) {
		t.Errorf("entries = %v, want 1", obj["entries"])
	}
	if b, _ := obj["bytes"].(float64); b <= 0 {
		t.Errorf("bytes = %v, want a positive total", obj["bytes"])
	}
	if _, ok := obj["size"].(string); !ok {
		t.Errorf("size = %v, want a human-readable string", obj["size"])
	}
}

func TestCacheInfoDefaultsToJSONOffATTY(t *testing.T) {
	t.Setenv("TBA_CACHE_DIR", t.TempDir())
	out, _, err := runCmd(t, nil, "cache", "info")
	requireNoError(t, err, "")
	if _, ok := decodeJSON(t, out).(map[string]any); !ok {
		t.Errorf("want a JSON object, got:\n%s", out)
	}
}

func TestCacheInfoCountsEntriesWrittenByRequests(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("TBA_CACHE_DIR", dir)

	srv := newFakeTBA(t, map[string]any{"/status": apiStatusJSON})
	if _, _, err := runCmd(t, srv, "status"); err != nil {
		t.Fatalf("status: %v", err)
	}

	out, _, err := runCmd(t, nil, "cache", "info", "--format", "table")
	requireNoError(t, err, "")
	requireContains(t, out, "Entries:      1")
	if strings.Contains(out, "Size:         0 B") {
		t.Errorf("cached entry should have a non-zero size:\n%s", out)
	}
}

func TestCacheClear(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("TBA_CACHE_DIR", dir)

	srv := newFakeTBA(t, map[string]any{
		"/status":      apiStatusJSON,
		"/team/frc177": teamFRC177JSON,
	})
	if _, _, err := runCmd(t, srv, "status"); err != nil {
		t.Fatalf("status: %v", err)
	}
	if _, _, err := runCmd(t, srv, "team", "view", "177"); err != nil {
		t.Fatalf("team view: %v", err)
	}

	out, _, err := runCmd(t, nil, "cache", "clear", "--format", "table")
	requireNoError(t, err, "")
	requireContains(t, out, "Removed 2 cache entries from "+dir)

	entries, err := os.ReadDir(filepath.Join(dir, "v1"))
	if err != nil {
		t.Fatalf("read cache dir: %v", err)
	}
	for _, e := range entries {
		if filepath.Ext(e.Name()) == ".json" {
			t.Errorf("cache entry %s survived clear", e.Name())
		}
	}
}

func TestCacheClearJSON(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("TBA_CACHE_DIR", dir)

	srv := newFakeTBA(t, map[string]any{"/status": apiStatusJSON})
	if _, _, err := runCmd(t, srv, "status"); err != nil {
		t.Fatalf("status: %v", err)
	}

	out, _, err := runCmd(t, nil, "cache", "clear", "--format", "json")
	requireNoError(t, err, "")

	obj, ok := decodeJSON(t, out).(map[string]any)
	if !ok {
		t.Fatalf("want a JSON object, got:\n%s", out)
	}
	if obj["removed"] != float64(1) {
		t.Errorf("removed = %v, want 1", obj["removed"])
	}
	if obj["directory"] != dir {
		t.Errorf("directory = %v, want %v", obj["directory"], dir)
	}
}

func TestCacheClearOnAnEmptyCache(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("TBA_CACHE_DIR", dir)

	out, _, err := runCmd(t, nil, "cache", "clear", "--format", "table")
	requireNoError(t, err, "")
	requireContains(t, out, "Removed 0 cache entries")
}

func TestCacheRevalidatesWithIfNoneMatch(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("TBA_CACHE_DIR", dir)

	srv := newFakeTBA(t, map[string]any{"/status": apiStatusJSON})
	setETag(t, srv, "/status", `"etag-1"`)

	first, _, err := runCmd(t, srv, "status")
	requireNoError(t, err, "")
	second, _, err := runCmd(t, srv, "status")
	requireNoError(t, err, "")

	if first != second {
		t.Errorf("revalidated output differs:\n%s\n---\n%s", first, second)
	}

	reqs := requestsTo(t, srv)
	if len(reqs) != 2 {
		t.Fatalf("want 2 requests, got %d", len(reqs))
	}
	if got := reqs[0].Headers.Get("If-None-Match"); got != "" {
		t.Errorf("first request should not send If-None-Match, got %q", got)
	}
	if got := reqs[1].Headers.Get("If-None-Match"); got != `"etag-1"` {
		t.Errorf("second request If-None-Match = %q", got)
	}
}

func TestCacheSubcommandsAreRegistered(t *testing.T) {
	got := subcommandNames(t, "cache")
	for _, want := range []string{"info", "list", "prune", "clear"} {
		if !contains(got, want) {
			t.Errorf("cache %s is not registered (have %v)", want, got)
		}
	}
}
