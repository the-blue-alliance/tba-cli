package season

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

// clock is a settable stand-in for time.Now.
type clock struct{ t time.Time }

func (c *clock) now() time.Time { return c.t }

func newTestStore(t *testing.T) (*Store, *clock) {
	t.Helper()
	c := &clock{t: time.Date(2025, 3, 1, 12, 0, 0, 0, time.UTC)}
	return NewAt(filepath.Join(t.TempDir(), fileName), c.now), c
}

func TestGetWithNoRecord(t *testing.T) {
	s, _ := newTestStore(t)
	if year, ok := s.Get(); ok {
		t.Errorf("Get() = %d, true; want no remembered season", year)
	}
}

func TestPutThenGet(t *testing.T) {
	s, _ := newTestStore(t)
	if err := s.Put(2025); err != nil {
		t.Fatalf("Put: %v", err)
	}
	year, ok := s.Get()
	if !ok || year != 2025 {
		t.Errorf("Get() = %d, %v; want 2025, true", year, ok)
	}
}

func TestARecordStaysFreshForADay(t *testing.T) {
	s, c := newTestStore(t)
	if err := s.Put(2025); err != nil {
		t.Fatalf("Put: %v", err)
	}

	c.t = c.t.Add(23 * time.Hour)
	if _, ok := s.Get(); !ok {
		t.Error("a 23-hour-old record should still be used")
	}
	c.t = c.t.Add(2 * time.Hour)
	if _, ok := s.Get(); ok {
		t.Error("a 25-hour-old record should have expired")
	}
}

func TestARecordFromTheFutureIsIgnored(t *testing.T) {
	s, c := newTestStore(t)
	if err := s.Put(2025); err != nil {
		t.Fatalf("Put: %v", err)
	}
	c.t = c.t.Add(-time.Hour)
	if _, ok := s.Get(); ok {
		t.Error("a record written after 'now' should not be trusted")
	}
}

func TestPutOverwritesTheRecord(t *testing.T) {
	s, _ := newTestStore(t)
	if err := s.Put(2024); err != nil {
		t.Fatalf("Put: %v", err)
	}
	if err := s.Put(2025); err != nil {
		t.Fatalf("Put: %v", err)
	}
	if year, _ := s.Get(); year != 2025 {
		t.Errorf("Get() = %d, want 2025", year)
	}
}

func TestACorruptRecordReadsAsMissing(t *testing.T) {
	s, _ := newTestStore(t)
	if err := os.WriteFile(s.Path(), []byte("not json"), 0600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	if _, ok := s.Get(); ok {
		t.Error("a corrupt record should read as a miss")
	}
}

func TestAnEmptySeasonReadsAsMissing(t *testing.T) {
	s, _ := newTestStore(t)
	if err := os.WriteFile(s.Path(), []byte(`{"current_season":0,"fetched_at":"2025-03-01T12:00:00Z"}`), 0600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	if _, ok := s.Get(); ok {
		t.Error("a zero season should read as a miss")
	}
}

func TestPutCreatesTheDirectory(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "nested", "cache")
	s := NewAt(filepath.Join(dir, fileName), nil)
	if err := s.Put(2025); err != nil {
		t.Fatalf("Put: %v", err)
	}
	if _, err := os.Stat(s.Path()); err != nil {
		t.Fatalf("record was not written: %v", err)
	}
}

func TestPutLeavesNoTempFiles(t *testing.T) {
	dir := t.TempDir()
	s := NewAt(filepath.Join(dir, fileName), nil)
	if err := s.Put(2025); err != nil {
		t.Fatalf("Put: %v", err)
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("ReadDir: %v", err)
	}
	if len(entries) != 1 || entries[0].Name() != fileName {
		t.Errorf("cache dir holds %d files, want just %s", len(entries), fileName)
	}
}

func TestNewFollowsTheCacheDirectory(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("TBA_CACHE_DIR", dir)

	s, err := New()
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if want := filepath.Join(dir, fileName); s.Path() != want {
		t.Errorf("Path() = %q, want %q", s.Path(), want)
	}
}
