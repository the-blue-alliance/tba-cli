// Package season remembers which FRC season The Blue Alliance considers
// current, so that commands whose --year defaults to "this season" do not pay
// for an extra request every time they run.
package season

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"

	"github.com/the-blue-alliance/tba-cli/internal/cache"
	"github.com/the-blue-alliance/tba-cli/internal/fsutil"
)

// TTL is how long a remembered season is trusted. A day is short enough that
// the January changeover is noticed the same day and long enough that the
// lookup costs nothing in practice.
const TTL = 24 * time.Hour

// fileName is the record's name inside the cache directory. It sits beside
// the response cache rather than inside it, so clearing cached responses does
// not turn into a `cache clear` that this file has to survive.
const fileName = "season.json"

// A Store is the remembered season on disk.
type Store struct {
	path string
	now  func() time.Time
}

type record struct {
	CurrentSeason int       `json:"current_season"`
	FetchedAt     time.Time `json:"fetched_at"`
}

// New returns the store in the usual cache directory.
func New() (*Store, error) {
	c, err := cache.New()
	if err != nil {
		return nil, err
	}
	return &Store{path: filepath.Join(c.Dir(), fileName), now: time.Now}, nil
}

// NewAt returns a store over an explicit file with an explicit clock.
func NewAt(path string, now func() time.Time) *Store {
	if now == nil {
		now = time.Now
	}
	return &Store{path: path, now: now}
}

// Path is where the record is kept.
func (s *Store) Path() string { return s.path }

// Get returns the remembered season, and whether it is still fresh. Anything
// unreadable, unparseable or too old reads as "not remembered", because the
// answer is only ever a shortcut for asking the API.
func (s *Store) Get() (int, bool) {
	b, err := os.ReadFile(s.path)
	if err != nil {
		return 0, false
	}
	var r record
	if err := json.Unmarshal(b, &r); err != nil {
		return 0, false
	}
	if r.CurrentSeason <= 0 || r.FetchedAt.IsZero() {
		return 0, false
	}
	age := s.now().Sub(r.FetchedAt)
	// A record from the future is a clock that moved; treat it as stale.
	if age < 0 || age >= TTL {
		return 0, false
	}
	return r.CurrentSeason, true
}

// Put remembers a season as of now.
func (s *Store) Put(year int) error {
	b, err := json.Marshal(record{CurrentSeason: year, FetchedAt: s.now().UTC()})
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(s.path), 0700); err != nil {
		return err
	}
	return fsutil.WriteFileAtomic(s.path, b, 0600)
}
