package cache

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// compact normalises JSON so tests compare values rather than whitespace.
func compact(t *testing.T, b []byte) string {
	t.Helper()
	var buf bytes.Buffer
	if err := json.Compact(&buf, b); err != nil {
		t.Fatalf("compacting %s: %v", b, err)
	}
	return buf.String()
}

func newTestCache(t *testing.T) (*Cache, string) {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("TBA_CACHE_DIR", dir)
	c, err := New()
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	return c, dir
}

func TestTBACacheDirOverridesEverything(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CACHE_HOME", t.TempDir())
	t.Setenv("TBA_CACHE_DIR", dir)

	c, err := New()
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if c.Dir() != dir {
		t.Errorf("Dir() = %q, want %q", c.Dir(), dir)
	}
}

func TestXDGCacheHomeIsUsedWhenTBACacheDirIsUnset(t *testing.T) {
	xdg := t.TempDir()
	t.Setenv("TBA_CACHE_DIR", "")
	t.Setenv("XDG_CACHE_HOME", xdg)

	c, err := New()
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if want := filepath.Join(xdg, "tba"); c.Dir() != want {
		t.Errorf("Dir() = %q, want %q", c.Dir(), want)
	}
}

func TestNewCreatesTheDirectory(t *testing.T) {
	base := t.TempDir()
	dir := filepath.Join(base, "nested", "cache")
	t.Setenv("TBA_CACHE_DIR", dir)

	if _, err := New(); err != nil {
		t.Fatalf("New: %v", err)
	}
	fi, err := os.Stat(dir)
	if err != nil {
		t.Fatalf("cache dir was not created: %v", err)
	}
	if !fi.IsDir() {
		t.Error("cache path is not a directory")
	}
}

func TestPutGetRoundTrip(t *testing.T) {
	c, _ := newTestCache(t)

	url := "https://www.thebluealliance.com/api/v3/team/frc177"
	body := []byte(`{"key":"frc177","nickname":"Bobcat Robotics"}`)
	if err := c.Put(url, `"etag-1"`, "Wed, 01 May 2024 12:00:00 GMT", body); err != nil {
		t.Fatalf("Put: %v", err)
	}

	got := c.Get(url)
	if got == nil {
		t.Fatal("Get returned nil for a freshly written entry")
	}
	if got.URL != url {
		t.Errorf("URL = %q", got.URL)
	}
	if got.ETag != `"etag-1"` {
		t.Errorf("ETag = %q", got.ETag)
	}
	if got.LastModified != "Wed, 01 May 2024 12:00:00 GMT" {
		t.Errorf("LastModified = %q", got.LastModified)
	}
	// The entry file is stored pretty-printed, so compare the JSON values.
	if compact(t, got.Body) != compact(t, body) {
		t.Errorf("Body = %s, want %s", got.Body, body)
	}
	if got.FetchedAt.IsZero() {
		t.Error("FetchedAt was not recorded")
	}
}

func TestPutOverwritesAnExistingEntry(t *testing.T) {
	c, dir := newTestCache(t)
	url := "https://example.test/status"

	if err := c.Put(url, `"v1"`, "", []byte(`{"n":1}`)); err != nil {
		t.Fatalf("Put: %v", err)
	}
	if err := c.Put(url, `"v2"`, "", []byte(`{"n":2}`)); err != nil {
		t.Fatalf("Put: %v", err)
	}

	got := c.Get(url)
	if got == nil || got.ETag != `"v2"` || compact(t, got.Body) != `{"n":2}` {
		t.Fatalf("second Put did not replace the entry: %+v", got)
	}
	count, _, err := c.Stats()
	if err != nil {
		t.Fatalf("Stats: %v", err)
	}
	if count != 1 {
		t.Errorf("want 1 entry in %s, got %d", dir, count)
	}
}

func TestDifferentURLsGetDifferentEntries(t *testing.T) {
	c, _ := newTestCache(t)
	if err := c.Put("https://example.test/a", "", "", []byte(`1`)); err != nil {
		t.Fatalf("Put: %v", err)
	}
	if err := c.Put("https://example.test/b", "", "", []byte(`2`)); err != nil {
		t.Fatalf("Put: %v", err)
	}

	if compact(t, c.Get("https://example.test/a").Body) != "1" {
		t.Error("entry a was clobbered")
	}
	if compact(t, c.Get("https://example.test/b").Body) != "2" {
		t.Error("entry b was clobbered")
	}
}

func TestGetMissingEntryReturnsNil(t *testing.T) {
	c, _ := newTestCache(t)
	if got := c.Get("https://example.test/never-fetched"); got != nil {
		t.Errorf("want nil, got %+v", got)
	}
}

func TestGetCorruptEntryReturnsNil(t *testing.T) {
	c, dir := newTestCache(t)
	url := "https://example.test/corrupt"

	if err := c.Put(url, "", "", []byte(`{"ok":true}`)); err != nil {
		t.Fatalf("Put: %v", err)
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("ReadDir: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("want 1 cache file, got %d", len(entries))
	}
	if err := os.WriteFile(filepath.Join(dir, entries[0].Name()), []byte("not json at all"), 0600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	if got := c.Get(url); got != nil {
		t.Errorf("a corrupt file should read as a miss, got %+v", got)
	}
}

func TestPutIsAtomicAndLeavesNoTempFiles(t *testing.T) {
	c, dir := newTestCache(t)
	for _, u := range []string{"https://example.test/1", "https://example.test/2", "https://example.test/3"} {
		if err := c.Put(u, "", "", []byte(`{}`)); err != nil {
			t.Fatalf("Put: %v", err)
		}
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("ReadDir: %v", err)
	}
	if len(entries) != 3 {
		t.Errorf("want exactly 3 files, got %d", len(entries))
	}
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".tmp") || strings.HasPrefix(e.Name(), "tba-cache-") {
			t.Errorf("temporary file %s was left behind", e.Name())
		}
		if filepath.Ext(e.Name()) != ".json" {
			t.Errorf("unexpected file %s", e.Name())
		}
	}
}

func TestEntryFileIsValidJSON(t *testing.T) {
	c, dir := newTestCache(t)
	if err := c.Put("https://example.test/x", `"e"`, "", []byte(`{"a":1}`)); err != nil {
		t.Fatalf("Put: %v", err)
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("ReadDir: %v", err)
	}
	b, err := os.ReadFile(filepath.Join(dir, entries[0].Name()))
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	var e Entry
	if err := json.Unmarshal(b, &e); err != nil {
		t.Fatalf("cache file is not valid JSON: %v", err)
	}
	if e.URL != "https://example.test/x" {
		t.Errorf("URL = %q", e.URL)
	}
}

func TestClearCountsAndRemovesEntries(t *testing.T) {
	c, dir := newTestCache(t)
	for _, u := range []string{"a", "b", "c"} {
		if err := c.Put("https://example.test/"+u, "", "", []byte(`{}`)); err != nil {
			t.Fatalf("Put: %v", err)
		}
	}
	// A non-cache file must survive and must not be counted.
	other := filepath.Join(dir, "README.txt")
	if err := os.WriteFile(other, []byte("keep me"), 0600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	removed, err := c.Clear()
	if err != nil {
		t.Fatalf("Clear: %v", err)
	}
	if removed != 3 {
		t.Errorf("Clear removed %d, want 3", removed)
	}
	if _, err := os.Stat(other); err != nil {
		t.Errorf("Clear deleted a non-cache file: %v", err)
	}
	if _, err := os.Stat(dir); err != nil {
		t.Errorf("Clear removed the cache directory: %v", err)
	}
}

func TestClearOnAnEmptyCache(t *testing.T) {
	c, _ := newTestCache(t)
	removed, err := c.Clear()
	if err != nil {
		t.Fatalf("Clear: %v", err)
	}
	if removed != 0 {
		t.Errorf("Clear removed %d, want 0", removed)
	}
}

func TestStats(t *testing.T) {
	c, dir := newTestCache(t)

	count, bytes, err := c.Stats()
	if err != nil {
		t.Fatalf("Stats: %v", err)
	}
	if count != 0 || bytes != 0 {
		t.Errorf("empty cache Stats = (%d, %d), want (0, 0)", count, bytes)
	}

	if err := c.Put("https://example.test/a", "", "", []byte(`{"a":1}`)); err != nil {
		t.Fatalf("Put: %v", err)
	}
	if err := c.Put("https://example.test/b", "", "", []byte(`{"b":2}`)); err != nil {
		t.Fatalf("Put: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "notes.txt"), []byte("ignored"), 0600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	count, bytes, err = c.Stats()
	if err != nil {
		t.Fatalf("Stats: %v", err)
	}
	if count != 2 {
		t.Errorf("count = %d, want 2", count)
	}
	if bytes <= 0 {
		t.Errorf("bytes = %d, want a positive total", bytes)
	}
}

func TestFormatSize(t *testing.T) {
	cases := []struct {
		in   int64
		want string
	}{
		{0, "0 B"},
		{512, "512 B"},
		{1023, "1023 B"},
		{1024, "1.0 KB"},
		{1536, "1.5 KB"},
		{1024 * 1024, "1.0 MB"},
		{3 * 1024 * 1024, "3.0 MB"},
	}
	for _, c := range cases {
		if got := FormatSize(c.in); got != c.want {
			t.Errorf("FormatSize(%d) = %q, want %q", c.in, got, c.want)
		}
	}
}
