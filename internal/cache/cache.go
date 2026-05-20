// Package cache implements a small on-disk HTTP response cache keyed by
// the request URL. It supports conditional GET revalidation via ETag /
// Last-Modified.
package cache

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

type Cache struct {
	dir string
}

type Entry struct {
	URL          string          `json:"url"`
	ETag         string          `json:"etag,omitempty"`
	LastModified string          `json:"last_modified,omitempty"`
	FetchedAt    time.Time       `json:"fetched_at"`
	Body         json.RawMessage `json:"body"`
}

func defaultDir() string {
	if d := os.Getenv("TBA_CACHE_DIR"); d != "" {
		return d
	}
	if d := os.Getenv("XDG_CACHE_HOME"); d != "" {
		return filepath.Join(d, "tba")
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".cache", "tba")
}

func New() (*Cache, error) {
	dir := defaultDir()
	if err := os.MkdirAll(dir, 0700); err != nil {
		return nil, err
	}
	return &Cache{dir: dir}, nil
}

func (c *Cache) Dir() string { return c.dir }

func keyFor(url string) string {
	sum := sha256.Sum256([]byte(url))
	return hex.EncodeToString(sum[:])
}

func (c *Cache) path(url string) string {
	return filepath.Join(c.dir, keyFor(url)+".json")
}

// Get returns the cached entry for url, or nil if there is no entry.
// Errors reading or parsing a cache file are treated as cache misses.
func (c *Cache) Get(url string) *Entry {
	b, err := os.ReadFile(c.path(url))
	if err != nil {
		return nil
	}
	var e Entry
	if err := json.Unmarshal(b, &e); err != nil {
		return nil
	}
	return &e
}

// Put atomically writes an entry to the cache.
func (c *Cache) Put(url string, etag, lastModified string, body []byte) error {
	e := Entry{
		URL:          url,
		ETag:         etag,
		LastModified: lastModified,
		FetchedAt:    time.Now().UTC(),
		Body:         json.RawMessage(body),
	}
	b, err := json.MarshalIndent(&e, "", "  ")
	if err != nil {
		return err
	}
	final := c.path(url)
	tmp, err := os.CreateTemp(c.dir, "tba-cache-*.tmp")
	if err != nil {
		return err
	}
	if _, err := tmp.Write(b); err != nil {
		tmp.Close()
		os.Remove(tmp.Name())
		return err
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmp.Name())
		return err
	}
	return os.Rename(tmp.Name(), final)
}

// Clear removes all cache entries. The directory itself is kept.
func (c *Cache) Clear() (removed int, err error) {
	entries, err := os.ReadDir(c.dir)
	if err != nil {
		return 0, err
	}
	for _, e := range entries {
		if filepath.Ext(e.Name()) != ".json" {
			continue
		}
		if err := os.Remove(filepath.Join(c.dir, e.Name())); err == nil {
			removed++
		}
	}
	return removed, nil
}

// Stats reports the number of cached entries and total bytes on disk.
func (c *Cache) Stats() (count int, bytes int64, err error) {
	entries, err := os.ReadDir(c.dir)
	if err != nil {
		return 0, 0, err
	}
	for _, e := range entries {
		if filepath.Ext(e.Name()) != ".json" {
			continue
		}
		fi, err := e.Info()
		if err != nil {
			continue
		}
		count++
		bytes += fi.Size()
	}
	return count, bytes, nil
}

// FormatSize returns a short human-readable representation of n bytes.
func FormatSize(n int64) string {
	const k = 1024
	if n < k {
		return fmt.Sprintf("%d B", n)
	}
	if n < k*k {
		return fmt.Sprintf("%.1f KB", float64(n)/k)
	}
	return fmt.Sprintf("%.1f MB", float64(n)/(k*k))
}
