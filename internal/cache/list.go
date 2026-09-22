package cache

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"
)

// EntryInfo describes one cache entry without carrying its body.
type EntryInfo struct {
	URL         string
	ETag        string
	FetchedAt   time.Time
	ValidatedAt time.Time
	Size        int64

	// file is the entry's path on disk, so that Prune removes exactly the
	// file the metadata came from rather than re-deriving it from the URL.
	file string
}

// LastTouched is the more recent of the fetch and the last revalidation. An
// entry that keeps coming back 304 is still in use, so pruning goes by this
// rather than by FetchedAt alone.
func (i EntryInfo) LastTouched() time.Time {
	if i.ValidatedAt.After(i.FetchedAt) {
		return i.ValidatedAt
	}
	return i.FetchedAt
}

// List returns metadata for every readable cache entry, sorted by URL.
// Unreadable or corrupt files are skipped: they cannot be served, and Clear
// already knows how to get rid of them.
func (c *Cache) List() ([]EntryInfo, error) {
	files, err := os.ReadDir(c.entriesDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	out := make([]EntryInfo, 0, len(files))
	for _, f := range files {
		if f.IsDir() || filepath.Ext(f.Name()) != ".json" {
			continue
		}
		path := filepath.Join(c.entriesDir, f.Name())
		b, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		var e Entry
		if err := json.Unmarshal(b, &e); err != nil {
			continue
		}
		out = append(out, EntryInfo{
			URL:         e.URL,
			ETag:        e.ETag,
			FetchedAt:   e.FetchedAt,
			ValidatedAt: e.ValidatedAt,
			Size:        int64(len(b)),
			file:        path,
		})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].URL < out[j].URL })
	return out, nil
}

// PruneResult reports what a prune removed, or would have removed.
type PruneResult struct {
	// Removed counts entry files plus any stray temporary files swept up.
	Removed int
	Bytes   int64
	// Entries lists the cache entries involved, sorted by URL. Stray
	// temporary files are counted but have no metadata worth listing.
	Entries []EntryInfo
}

// Prune removes every entry last touched before cutoff, along with any stray
// temporary file left behind by an interrupted write — those can never be
// served, so their age does not matter. With dryRun the result describes what
// would go without touching the disk.
func (c *Cache) Prune(cutoff time.Time, dryRun bool) (PruneResult, error) {
	var res PruneResult
	entries, err := c.List()
	if err != nil {
		return res, err
	}
	for _, e := range entries {
		if !e.LastTouched().Before(cutoff) {
			continue
		}
		if !dryRun {
			if err := os.Remove(e.file); err != nil {
				continue
			}
		}
		res.Entries = append(res.Entries, e)
		res.Removed++
		res.Bytes += e.Size
	}

	files, err := os.ReadDir(c.entriesDir)
	if err != nil {
		if os.IsNotExist(err) {
			return res, nil
		}
		return res, err
	}
	for _, f := range files {
		if f.IsDir() || filepath.Ext(f.Name()) != ".tmp" {
			continue
		}
		var size int64
		if fi, err := f.Info(); err == nil {
			size = fi.Size()
		}
		if !dryRun {
			if err := os.Remove(filepath.Join(c.entriesDir, f.Name())); err != nil {
				continue
			}
		}
		res.Removed++
		res.Bytes += size
	}
	return res, nil
}

// WriteEntry stores e verbatim, keyed by e.URL. It is the low-level form of
// Put, for callers that need to control the timestamps themselves.
func (c *Cache) WriteEntry(e *Entry) error {
	return c.write(e.URL, e)
}

// FormatAge renders a duration as a short age such as "45s", "12m", "3h" or
// "8d". Anything under a second, or in the future, reads as "0s".
func FormatAge(d time.Duration) string {
	switch {
	case d < time.Second:
		return "0s"
	case d < time.Minute:
		return fmt.Sprintf("%ds", int(d.Seconds()))
	case d < time.Hour:
		return fmt.Sprintf("%dm", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("%dh", int(d.Hours()))
	default:
		return fmt.Sprintf("%dd", int(d.Hours()/24))
	}
}
