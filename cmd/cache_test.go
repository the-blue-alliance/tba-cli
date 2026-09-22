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

	out, _, err := runCmd(t, nil, "cache", "info")
	requireNoError(t, err, "")

	requireContains(t, out, "Directory: "+dir)
	requireContains(t, out, "Entries:   0")
	requireContains(t, out, "Size:      0 B")
}

func TestCacheInfoCountsEntriesWrittenByRequests(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("TBA_CACHE_DIR", dir)

	srv := newFakeTBA(t, map[string]any{"/status": apiStatusJSON})
	if _, _, err := runCmd(t, srv, "status"); err != nil {
		t.Fatalf("status: %v", err)
	}

	out, _, err := runCmd(t, nil, "cache", "info")
	requireNoError(t, err, "")
	requireContains(t, out, "Entries:   1")
	if strings.Contains(out, "Size:      0 B") {
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

	out, _, err := runCmd(t, nil, "cache", "clear")
	requireNoError(t, err, "")
	requireContains(t, out, "Removed 2 cache entries from "+dir)

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("read cache dir: %v", err)
	}
	for _, e := range entries {
		if filepath.Ext(e.Name()) == ".json" {
			t.Errorf("cache entry %s survived clear", e.Name())
		}
	}
}

func TestCacheClearOnAnEmptyCache(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("TBA_CACHE_DIR", dir)

	out, _, err := runCmd(t, nil, "cache", "clear")
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
	for _, want := range []string{"info", "clear"} {
		if !contains(got, want) {
			t.Errorf("cache %s is not registered (have %v)", want, got)
		}
	}
}
