package cmd

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/the-blue-alliance/tba-cli/internal/cache"
	"github.com/the-blue-alliance/tba-cli/internal/clierr"
)

// seedEntry writes a cache entry with explicit timestamps, so the age-based
// behaviour of info, list and prune can be pinned down exactly.
func seedEntry(t *testing.T, url string, fetched, validated time.Time) {
	t.Helper()
	c, err := cache.New()
	if err != nil {
		t.Fatalf("cache.New: %v", err)
	}
	e := &cache.Entry{
		URL:         url,
		ETag:        `"etag-` + filepath.Base(url) + `"`,
		FetchedAt:   fetched,
		ValidatedAt: validated,
		Body:        json.RawMessage(`{"key":"` + filepath.Base(url) + `"}`),
	}
	if err := c.WriteEntry(e); err != nil {
		t.Fatalf("WriteEntry: %v", err)
	}
}

const fakeBase = "https://www.thebluealliance.com/api/v3"

func TestCacheInfoReportsAgesAndStaleCount(t *testing.T) {
	t.Setenv("TBA_CACHE_DIR", t.TempDir())
	now := time.Now()
	seedEntry(t, fakeBase+"/status", now.Add(-12*time.Minute), now.Add(-12*time.Minute))
	seedEntry(t, fakeBase+"/team/frc177", now.Add(-40*24*time.Hour), now.Add(-40*24*time.Hour))
	seedEntry(t, fakeBase+"/team/frc254", now.Add(-90*24*time.Hour), now.Add(-90*24*time.Hour))

	out, _, err := runCmd(t, nil, "cache", "info", "--format", "table")
	requireNoError(t, err, "")
	requireContains(t, out, "Entries:      3")
	requireContains(t, out, "Oldest:       90d ago")
	requireContains(t, out, "Newest:       12m ago")
	requireContains(t, out, "Stale (>30d): 2")
}

func TestCacheInfoJSONCarriesTheAgeFields(t *testing.T) {
	t.Setenv("TBA_CACHE_DIR", t.TempDir())
	now := time.Now().UTC().Truncate(time.Second)
	oldest := now.Add(-40 * 24 * time.Hour)
	newest := now.Add(-time.Hour)
	seedEntry(t, fakeBase+"/status", newest, newest)
	seedEntry(t, fakeBase+"/team/frc177", oldest, oldest)

	out, _, err := runCmd(t, nil, "cache", "info", "--format", "json")
	requireNoError(t, err, "")
	obj, ok := decodeJSON(t, out).(map[string]any)
	if !ok {
		t.Fatalf("want a JSON object, got:\n%s", out)
	}
	for _, key := range []string{"directory", "entries", "bytes", "size", "oldest_fetched_at", "newest_fetched_at", "stale_count"} {
		if _, present := obj[key]; !present {
			t.Errorf("%s is missing from:\n%s", key, out)
		}
	}
	gotOldest, err := time.Parse(time.RFC3339Nano, obj["oldest_fetched_at"].(string))
	if err != nil {
		t.Fatalf("oldest_fetched_at: %v", err)
	}
	if !gotOldest.Equal(oldest) {
		t.Errorf("oldest_fetched_at = %v, want %v", gotOldest, oldest)
	}
	gotNewest, err := time.Parse(time.RFC3339Nano, obj["newest_fetched_at"].(string))
	if err != nil {
		t.Fatalf("newest_fetched_at: %v", err)
	}
	if !gotNewest.Equal(newest) {
		t.Errorf("newest_fetched_at = %v, want %v", gotNewest, newest)
	}
	if obj["stale_count"] != float64(1) {
		t.Errorf("stale_count = %v, want 1", obj["stale_count"])
	}
}

func TestCacheInfoOnAnEmptyCacheHasNoAges(t *testing.T) {
	t.Setenv("TBA_CACHE_DIR", t.TempDir())

	out, _, err := runCmd(t, nil, "cache", "info", "--format", "table")
	requireNoError(t, err, "")
	requireContains(t, out, "Oldest:       -")
	requireContains(t, out, "Newest:       -")
	requireContains(t, out, "Stale (>30d): 0")

	out, _, err = runCmd(t, nil, "cache", "info", "--format", "json")
	requireNoError(t, err, "")
	obj := decodeJSON(t, out).(map[string]any)
	if obj["oldest_fetched_at"] != nil || obj["newest_fetched_at"] != nil {
		t.Errorf("an empty cache has no ages: %v", out)
	}
}

func TestCacheList(t *testing.T) {
	t.Setenv("TBA_CACHE_DIR", t.TempDir())
	now := time.Now()
	seedEntry(t, fakeBase+"/team/frc177", now.Add(-2*time.Hour), now.Add(-30*time.Minute))
	seedEntry(t, fakeBase+"/status", now.Add(-5*time.Minute), now.Add(-5*time.Minute))

	out, _, err := runCmd(t, nil, "cache", "list", "--format", "table")
	requireNoError(t, err, "")

	// A header, its underline, then one line per entry.
	got := lines(out)
	if len(got) != 4 {
		t.Fatalf("want a header and 2 rows, got:\n%s", out)
	}
	for _, want := range []string{"Path", "Fetched", "Validated", "Size", "ETag"} {
		if !strings.Contains(got[0], want) {
			t.Errorf("header is missing %q: %q", want, got[0])
		}
	}
	// Sorted by path: /status comes before /team/frc177.
	if !strings.HasPrefix(got[2], "/status") {
		t.Errorf("row 1 = %q, want /status first", got[2])
	}
	if !strings.HasPrefix(got[3], "/team/frc177") {
		t.Errorf("row 2 = %q, want /team/frc177", got[3])
	}
	requireContains(t, got[3], "2h ago")
	requireContains(t, got[3], "30m ago")
	requireContains(t, got[3], `"etag-frc177"`)
}

// An entry stamped later than now is a clock that moved, not a response from
// tomorrow, so its age is clamped rather than counted forwards: "in 3h"
// dressed up as "3h ago" would be a lie in both directions.
func TestCacheListClampsAnEntryFromTheFuture(t *testing.T) {
	t.Setenv("TBA_CACHE_DIR", t.TempDir())
	future := time.Now().Add(3 * time.Hour)
	seedEntry(t, fakeBase+"/status", future, future)

	out, _, err := runCmd(t, nil, "cache", "list", "--format", "csv", "--no-headers")
	requireNoError(t, err, "")
	requireContains(t, out, "0s ago")
	if strings.Contains(out, "3h ago") {
		t.Errorf("a future timestamp must not read as an age:\n%s", out)
	}
}

func TestCacheListJSON(t *testing.T) {
	t.Setenv("TBA_CACHE_DIR", t.TempDir())
	now := time.Now()
	seedEntry(t, fakeBase+"/status", now.Add(-time.Minute), now.Add(-time.Minute))

	out, _, err := runCmd(t, nil, "cache", "list", "--format", "json")
	requireNoError(t, err, "")
	rows, ok := decodeJSON(t, out).([]any)
	if !ok {
		t.Fatalf("want a JSON array, got:\n%s", out)
	}
	if len(rows) != 1 {
		t.Fatalf("want 1 row, got %d:\n%s", len(rows), out)
	}
	row := rows[0].(map[string]any)
	if row["path"] != "/status" {
		t.Errorf("path = %v, want /status", row["path"])
	}
	if row["url"] != fakeBase+"/status" {
		t.Errorf("url = %v", row["url"])
	}
	if b, _ := row["bytes"].(float64); b <= 0 {
		t.Errorf("bytes = %v, want a positive size", row["bytes"])
	}
}

func TestCacheListOnAnEmptyCacheIsAnEmptyArray(t *testing.T) {
	t.Setenv("TBA_CACHE_DIR", t.TempDir())
	out, _, err := runCmd(t, nil, "cache", "list", "--format", "json")
	requireNoError(t, err, "")
	rows, ok := decodeJSON(t, out).([]any)
	if !ok || len(rows) != 0 {
		t.Errorf("want [], got:\n%s", out)
	}
}

func TestCacheListHonoursColumnsAndSort(t *testing.T) {
	t.Setenv("TBA_CACHE_DIR", t.TempDir())
	now := time.Now()
	seedEntry(t, fakeBase+"/status", now.Add(-time.Minute), now.Add(-time.Minute))
	seedEntry(t, fakeBase+"/team/frc177", now.Add(-time.Hour), now.Add(-time.Hour))

	out, _, err := runCmd(t, nil, "cache", "list", "--format", "csv", "--columns", "path", "--sort=-path")
	requireNoError(t, err, "")
	want := []string{"Path", "/team/frc177", "/status"}
	if got := lines(out); len(got) != 3 || got[0] != want[0] || got[1] != want[1] || got[2] != want[2] {
		t.Errorf("csv = %q, want %q", lines(out), want)
	}
}

func TestCachePruneRemovesOldEntries(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("TBA_CACHE_DIR", dir)
	now := time.Now()
	seedEntry(t, fakeBase+"/team/frc177", now.Add(-40*24*time.Hour), now.Add(-40*24*time.Hour))
	seedEntry(t, fakeBase+"/status", now.Add(-time.Hour), now.Add(-time.Hour))

	out, errOut, err := runCmd(t, nil, "cache", "prune", "--format", "table")
	requireNoError(t, err, errOut)
	if out != "" {
		t.Errorf("stdout should stay clean in table mode, got:\n%s", out)
	}
	requireContains(t, errOut, "Removed 1 entries (")

	c, err := cache.New()
	if err != nil {
		t.Fatalf("cache.New: %v", err)
	}
	if c.Get(fakeBase+"/team/frc177") != nil {
		t.Error("the 40-day-old entry survived")
	}
	if c.Get(fakeBase+"/status") == nil {
		t.Error("prune took a fresh entry")
	}
}

func TestCachePruneJSON(t *testing.T) {
	t.Setenv("TBA_CACHE_DIR", t.TempDir())
	now := time.Now()
	seedEntry(t, fakeBase+"/team/frc177", now.Add(-40*24*time.Hour), now.Add(-40*24*time.Hour))

	out, errOut, err := runCmd(t, nil, "cache", "prune", "--format", "json")
	requireNoError(t, err, errOut)
	if errOut != "" {
		t.Errorf("json mode should not also write to stderr:\n%s", errOut)
	}
	obj, ok := decodeJSON(t, out).(map[string]any)
	if !ok {
		t.Fatalf("want a JSON object, got:\n%s", out)
	}
	if obj["removed"] != float64(1) {
		t.Errorf("removed = %v, want 1", obj["removed"])
	}
	if b, _ := obj["bytes"].(float64); b <= 0 {
		t.Errorf("bytes = %v, want a positive total", obj["bytes"])
	}
	if obj["dry_run"] != false {
		t.Errorf("dry_run = %v, want false", obj["dry_run"])
	}
	paths, _ := obj["paths"].([]any)
	if len(paths) != 1 || paths[0] != "/team/frc177" {
		t.Errorf("paths = %v, want [/team/frc177]", obj["paths"])
	}
}

func TestCachePruneOlderThanAcceptsDaysWeeksAndGoDurations(t *testing.T) {
	cases := []struct {
		spec string
		age  time.Duration
	}{
		{"12h", 12 * time.Hour},
		{"30d", 30 * 24 * time.Hour},
		{"2w", 14 * 24 * time.Hour},
		{"90m", 90 * time.Minute},
	}
	for _, c := range cases {
		t.Run(c.spec, func(t *testing.T) {
			t.Setenv("TBA_CACHE_DIR", t.TempDir())
			now := time.Now()
			// One entry either side of the cutoff the spec describes.
			seedEntry(t, fakeBase+"/old", now.Add(-c.age-time.Hour), now.Add(-c.age-time.Hour))
			seedEntry(t, fakeBase+"/new", now.Add(-c.age+time.Hour), now.Add(-c.age+time.Hour))

			out, _, err := runCmd(t, nil, "cache", "prune", "--older-than", c.spec, "--format", "json")
			requireNoError(t, err, "")
			obj := decodeJSON(t, out).(map[string]any)
			if obj["removed"] != float64(1) {
				t.Errorf("--older-than %s removed %v, want 1", c.spec, obj["removed"])
			}
			paths, _ := obj["paths"].([]any)
			if len(paths) != 1 || paths[0] != "/old" {
				t.Errorf("paths = %v, want [/old]", obj["paths"])
			}
		})
	}
}

func TestCachePruneRejectsAnUnparseableAge(t *testing.T) {
	t.Setenv("TBA_CACHE_DIR", t.TempDir())
	for _, spec := range []string{"yesterday", "30 days", "d", "-5d", "-1h"} {
		t.Run(spec, func(t *testing.T) {
			_, _, err := runCmd(t, nil, "cache", "prune", "--older-than", spec)
			if clierr.ExitCode(err) != clierr.ExitUsage {
				t.Fatalf("--older-than %q: exit code %d (err %v), want %d",
					spec, clierr.ExitCode(err), err, clierr.ExitUsage)
			}
			requireErrorContains(t, err, "--older-than")
		})
	}
}

func TestCachePruneDryRunListsWithoutRemoving(t *testing.T) {
	t.Setenv("TBA_CACHE_DIR", t.TempDir())
	now := time.Now()
	seedEntry(t, fakeBase+"/team/frc177", now.Add(-40*24*time.Hour), now.Add(-40*24*time.Hour))

	out, errOut, err := runCmd(t, nil, "cache", "prune", "--dry-run", "--format", "table")
	requireNoError(t, err, errOut)
	if out != "" {
		t.Errorf("stdout should stay clean, got:\n%s", out)
	}
	requireContains(t, errOut, "would remove /team/frc177")
	requireContains(t, errOut, "Would remove 1 entries (")

	c, err := cache.New()
	if err != nil {
		t.Fatalf("cache.New: %v", err)
	}
	if c.Get(fakeBase+"/team/frc177") == nil {
		t.Error("a dry run deleted the entry")
	}

	out, _, err = runCmd(t, nil, "cache", "prune", "--dry-run", "--format", "json")
	requireNoError(t, err, "")
	if obj := decodeJSON(t, out).(map[string]any); obj["dry_run"] != true {
		t.Errorf("dry_run = %v, want true", obj["dry_run"])
	}
}

func TestCachePruneSweepsStrayTempFiles(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("TBA_CACHE_DIR", dir)
	now := time.Now()
	seedEntry(t, fakeBase+"/status", now.Add(-time.Minute), now.Add(-time.Minute))
	stray := filepath.Join(dir, "v1", "tba-cache-42.tmp")
	if err := os.WriteFile(stray, []byte("half a write"), 0600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	out, _, err := runCmd(t, nil, "cache", "prune", "--format", "json")
	requireNoError(t, err, "")
	if obj := decodeJSON(t, out).(map[string]any); obj["removed"] != float64(1) {
		t.Errorf("removed = %v, want 1 (the temp file)", obj["removed"])
	}
	if _, err := os.Stat(stray); !os.IsNotExist(err) {
		t.Errorf("the stray temp file survived: %v", err)
	}
}

func TestCachePruneOnAnEmptyCache(t *testing.T) {
	t.Setenv("TBA_CACHE_DIR", t.TempDir())
	out, errOut, err := runCmd(t, nil, "cache", "prune", "--format", "table")
	requireNoError(t, err, errOut)
	requireContains(t, errOut, "Removed 0 entries (0 B)")
	if out != "" {
		t.Errorf("stdout should stay clean, got:\n%s", out)
	}
}

func TestRequestPathStripsTheAPIPrefix(t *testing.T) {
	cases := map[string]string{
		fakeBase + "/team/frc177":         "/team/frc177",
		fakeBase + "/status":              "/status",
		"http://127.0.0.1:8080/status":    "/status",
		fakeBase + "/teams/2024/0?page=1": "/teams/2024/0?page=1",
		"https://www.thebluealliance.com": "/",
		"://not a url":                    "://not a url",
	}
	for in, want := range cases {
		if got := requestPath(in); got != want {
			t.Errorf("requestPath(%q) = %q, want %q", in, got, want)
		}
	}
}
