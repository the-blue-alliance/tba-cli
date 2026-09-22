package cache

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// putAt writes an entry with explicit timestamps, which is how the age-based
// behaviour is pinned down without waiting for the clock.
func putAt(t *testing.T, c *Cache, url string, fetched, validated time.Time) {
	t.Helper()
	e := &Entry{
		URL:         url,
		ETag:        `"e-` + url + `"`,
		FetchedAt:   fetched,
		ValidatedAt: validated,
		Body:        json.RawMessage(`{"ok":true}`),
	}
	if err := c.WriteEntry(e); err != nil {
		t.Fatalf("WriteEntry(%s): %v", url, err)
	}
}

func TestWriteEntryPreservesTimestamps(t *testing.T) {
	c, _ := newTestCache(t)
	fetched := time.Date(2024, 5, 1, 12, 0, 0, 0, time.UTC)
	validated := fetched.Add(2 * time.Hour)
	putAt(t, c, "https://example.test/a", fetched, validated)

	got := c.Get("https://example.test/a")
	if got == nil {
		t.Fatal("entry is missing")
	}
	if !got.FetchedAt.Equal(fetched) {
		t.Errorf("FetchedAt = %v, want %v", got.FetchedAt, fetched)
	}
	if !got.ValidatedAt.Equal(validated) {
		t.Errorf("ValidatedAt = %v, want %v", got.ValidatedAt, validated)
	}
}

func TestListSortsByURLAndReportsSizes(t *testing.T) {
	c, _ := newTestCache(t)
	now := time.Now().UTC()
	putAt(t, c, "https://example.test/c", now, now)
	putAt(t, c, "https://example.test/a", now, now)
	putAt(t, c, "https://example.test/b", now, now)

	got, err := c.List()
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(got) != 3 {
		t.Fatalf("want 3 entries, got %d", len(got))
	}
	want := []string{"https://example.test/a", "https://example.test/b", "https://example.test/c"}
	for i, w := range want {
		if got[i].URL != w {
			t.Errorf("entry %d URL = %q, want %q", i, got[i].URL, w)
		}
		if got[i].Size <= 0 {
			t.Errorf("entry %d Size = %d, want a positive size", i, got[i].Size)
		}
		if got[i].ETag == "" {
			t.Errorf("entry %d lost its ETag", i)
		}
	}
}

func TestListSkipsCorruptAndForeignFiles(t *testing.T) {
	c, dir := newTestCache(t)
	now := time.Now().UTC()
	putAt(t, c, "https://example.test/good", now, now)
	if err := os.WriteFile(filepath.Join(dir, "deadbeef.json"), []byte("not json"), 0600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "notes.txt"), []byte("hi"), 0600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	got, err := c.List()
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(got) != 1 || got[0].URL != "https://example.test/good" {
		t.Fatalf("List = %+v, want only the readable entry", got)
	}
}

func TestListToleratesAMissingEntriesDirectory(t *testing.T) {
	c, _ := newTestCacheWithRoot(t)
	if err := os.RemoveAll(c.EntriesDir()); err != nil {
		t.Fatalf("RemoveAll: %v", err)
	}
	got, err := c.List()
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("List = %+v, want nothing", got)
	}
}

func TestLastTouchedPrefersTheLaterOfFetchAndValidation(t *testing.T) {
	fetched := time.Date(2024, 5, 1, 0, 0, 0, 0, time.UTC)
	later := fetched.Add(time.Hour)
	if got := (EntryInfo{FetchedAt: fetched, ValidatedAt: later}).LastTouched(); !got.Equal(later) {
		t.Errorf("LastTouched = %v, want %v", got, later)
	}
	// An entry written before ValidatedAt existed has a zero validation time.
	if got := (EntryInfo{FetchedAt: fetched}).LastTouched(); !got.Equal(fetched) {
		t.Errorf("LastTouched = %v, want %v", got, fetched)
	}
}

func TestPruneRemovesOnlyEntriesOlderThanTheCutoff(t *testing.T) {
	c, _ := newTestCache(t)
	now := time.Now().UTC()
	old := now.Add(-40 * 24 * time.Hour)
	fresh := now.Add(-time.Hour)
	putAt(t, c, "https://example.test/old", old, old)
	putAt(t, c, "https://example.test/fresh", fresh, fresh)

	res, err := c.Prune(now.Add(-30*24*time.Hour), false)
	if err != nil {
		t.Fatalf("Prune: %v", err)
	}
	if res.Removed != 1 {
		t.Errorf("Removed = %d, want 1", res.Removed)
	}
	if res.Bytes <= 0 {
		t.Errorf("Bytes = %d, want a positive total", res.Bytes)
	}
	if len(res.Entries) != 1 || res.Entries[0].URL != "https://example.test/old" {
		t.Fatalf("Entries = %+v, want the old one", res.Entries)
	}
	if c.Get("https://example.test/old") != nil {
		t.Error("the old entry survived the prune")
	}
	if c.Get("https://example.test/fresh") == nil {
		t.Error("the fresh entry was pruned")
	}
}

func TestPruneKeepsAnOldEntryThatWasRecentlyRevalidated(t *testing.T) {
	c, _ := newTestCache(t)
	now := time.Now().UTC()
	putAt(t, c, "https://example.test/revalidated", now.Add(-90*24*time.Hour), now.Add(-time.Minute))

	res, err := c.Prune(now.Add(-30*24*time.Hour), false)
	if err != nil {
		t.Fatalf("Prune: %v", err)
	}
	if res.Removed != 0 {
		t.Errorf("Removed = %d, want 0", res.Removed)
	}
	if c.Get("https://example.test/revalidated") == nil {
		t.Error("a revalidated entry should not be pruned")
	}
}

func TestPruneDryRunTouchesNothing(t *testing.T) {
	c, _ := newTestCache(t)
	now := time.Now().UTC()
	old := now.Add(-40 * 24 * time.Hour)
	putAt(t, c, "https://example.test/old", old, old)

	res, err := c.Prune(now.Add(-30*24*time.Hour), true)
	if err != nil {
		t.Fatalf("Prune: %v", err)
	}
	if res.Removed != 1 || len(res.Entries) != 1 {
		t.Errorf("dry run reported %d removed / %d listed, want 1 / 1", res.Removed, len(res.Entries))
	}
	if c.Get("https://example.test/old") == nil {
		t.Error("a dry run deleted the entry")
	}
}

func TestPruneSweepsStrayTempFiles(t *testing.T) {
	c, dir := newTestCache(t)
	now := time.Now().UTC()
	putAt(t, c, "https://example.test/fresh", now, now)
	stray := filepath.Join(dir, "tba-cache-123.tmp")
	if err := os.WriteFile(stray, []byte("interrupted write"), 0600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	res, err := c.Prune(now.Add(-30*24*time.Hour), false)
	if err != nil {
		t.Fatalf("Prune: %v", err)
	}
	if res.Removed != 1 {
		t.Errorf("Removed = %d, want 1 (the temp file)", res.Removed)
	}
	if _, err := os.Stat(stray); !os.IsNotExist(err) {
		t.Errorf("the stray temp file survived: %v", err)
	}
	if c.Get("https://example.test/fresh") == nil {
		t.Error("prune removed a fresh entry while sweeping temp files")
	}
}

func TestPruneOnAnEmptyCache(t *testing.T) {
	c, _ := newTestCacheWithRoot(t)
	if err := os.RemoveAll(c.EntriesDir()); err != nil {
		t.Fatalf("RemoveAll: %v", err)
	}
	res, err := c.Prune(time.Now(), false)
	if err != nil {
		t.Fatalf("Prune: %v", err)
	}
	if res.Removed != 0 || res.Bytes != 0 {
		t.Errorf("Prune = %+v, want an empty result", res)
	}
}

func TestFormatAge(t *testing.T) {
	cases := []struct {
		in   time.Duration
		want string
	}{
		{0, "0s"},
		{-time.Hour, "0s"},
		{900 * time.Millisecond, "0s"},
		{45 * time.Second, "45s"},
		{12 * time.Minute, "12m"},
		{90 * time.Minute, "1h"},
		{23 * time.Hour, "23h"},
		{25 * time.Hour, "1d"},
		{40 * 24 * time.Hour, "40d"},
	}
	for _, c := range cases {
		if got := FormatAge(c.in); got != c.want {
			t.Errorf("FormatAge(%v) = %q, want %q", c.in, got, c.want)
		}
	}
}
