package cmd

import (
	"fmt"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/the-blue-alliance/tba-cli/internal/clierr"
)

// otherSeason is a season that is not the calendar year, so that a test can
// tell "the API told us" apart from "we guessed from the clock".
func otherSeason() int { return currentYear() - 1 }

// statusWithSeason is a /status body naming a season.
func statusWithSeason(year int) string {
	return fmt.Sprintf(`{"current_season":%d,"max_season":%d,"is_datafeed_down":false}`, year, year)
}

// eventsFake serves /status naming season, plus an empty event list for that
// season, for the calendar year and for any other year the test asks about.
func eventsFake(t *testing.T, season int, years ...int) *httptest.Server {
	t.Helper()
	routes := map[string]any{"/status": statusWithSeason(season)}
	for _, y := range append([]int{season, currentYear()}, years...) {
		routes[fmt.Sprintf("/events/%d", y)] = "[]"
	}
	return newFakeTBA(t, routes)
}

func TestYearDefaultsToTheSeasonTheAPIReports(t *testing.T) {
	season := otherSeason()
	srv := eventsFake(t, season)

	_, _, err := runCmd(t, srv, "event", "list")
	requireNoError(t, err, "")

	want := fmt.Sprintf("/events/%d", season)
	got := requestPaths(t, srv)
	if !contains(got, want) {
		t.Errorf("requested %v, want %s (the season /status reported)", got, want)
	}
	if contains(got, fmt.Sprintf("/events/%d", currentYear())) {
		t.Error("the calendar year was used even though /status answered")
	}
}

func TestYearFallsBackToTheCalendarWhenStatusFails(t *testing.T) {
	season := otherSeason()
	srv := eventsFake(t, season)
	setStatus(t, srv, "/status", 500)

	_, _, err := runCmd(t, srv, "event", "list", "--retries", "0")
	requireNoError(t, err, "")

	want := fmt.Sprintf("/events/%d", currentYear())
	if got := requestPaths(t, srv); !contains(got, want) {
		t.Errorf("requested %v, want the calendar year %s", got, want)
	}
}

func TestYearFallsBackToTheCalendarWhenTheServerIsUnreachable(t *testing.T) {
	// A closed listener: nothing will answer, which is what being offline
	// looks like from here.
	srv := newFakeTBA(t, map[string]any{})
	url := srv.URL
	srv.Close()

	t.Setenv("TBA_CACHE_DIR", t.TempDir())
	t.Setenv("TBA_CONFIG_DIR", t.TempDir())
	_, _, err := runCmd(t, nil, "event", "list", "--base-url", url, "--retries", "0")
	// The command itself fails (there is no server), but it failed asking for
	// the calendar year rather than failing to work out a year at all.
	requireErrorContains(t, err, fmt.Sprintf("%d", currentYear()))
}

func TestYearComesFromTheEnvironment(t *testing.T) {
	season := otherSeason()
	srv := eventsFake(t, season, 2019)
	t.Setenv("TBA_YEAR", "2019")

	_, _, err := runCmd(t, srv, "event", "list")
	requireNoError(t, err, "")

	got := requestPaths(t, srv)
	if !contains(got, "/events/2019") {
		t.Errorf("requested %v, want /events/2019 from TBA_YEAR", got)
	}
	if contains(got, "/status") {
		t.Error("TBA_YEAR was set, so no season lookup should have been made")
	}
}

func TestYearComesFromTheConfigFile(t *testing.T) {
	season := otherSeason()
	srv := eventsFake(t, season, 2018)
	writeConfig(t, "year: 2018\n")

	_, _, err := runCmd(t, srv, "event", "list")
	requireNoError(t, err, "")

	got := requestPaths(t, srv)
	if !contains(got, "/events/2018") {
		t.Errorf("requested %v, want /events/2018 from the config file", got)
	}
	if contains(got, "/status") {
		t.Error("the config file named a year, so no season lookup should have been made")
	}
}

func TestYearFlagBeatsTheEnvironmentAndTheConfigFile(t *testing.T) {
	season := otherSeason()
	srv := eventsFake(t, season, 2017)
	writeConfig(t, "year: 2018\n")
	t.Setenv("TBA_YEAR", "2019")

	_, _, err := runCmd(t, srv, "event", "list", "--year", "2017")
	requireNoError(t, err, "")

	if got := requestPaths(t, srv); !contains(got, "/events/2017") {
		t.Errorf("requested %v, want /events/2017 from --year", got)
	}
}

func TestYearRejectsANonsensicalValue(t *testing.T) {
	srv := eventsFake(t, otherSeason())
	_, _, err := runCmd(t, srv, "event", "list", "--year", "0")
	requireErrorContains(t, err, "--year")
	if got := clierr.ExitCode(err); got != clierr.ExitUsage {
		t.Errorf("exit code = %d, want %d", got, clierr.ExitUsage)
	}
}

func TestTheSeasonIsLookedUpOnceADay(t *testing.T) {
	season := otherSeason()
	srv := eventsFake(t, season)
	cacheDir := t.TempDir()
	t.Setenv("TBA_CACHE_DIR", cacheDir)
	t.Setenv("TBA_CONFIG_DIR", t.TempDir())

	for i := 0; i < 3; i++ {
		if _, _, err := runCmd(t, srv, "event", "list"); err != nil {
			t.Fatalf("event list: %v", err)
		}
	}

	if n := countPath(requestPaths(t, srv), "/status"); n != 1 {
		t.Errorf("asked for the season %d times in a row, want 1", n)
	}
	if _, err := os.Stat(filepath.Join(cacheDir, "season.json")); err != nil {
		t.Errorf("the season was not remembered on disk: %v", err)
	}
}

func TestNoCacheAsksForTheSeasonAgain(t *testing.T) {
	season := otherSeason()
	srv := eventsFake(t, season)
	t.Setenv("TBA_CACHE_DIR", t.TempDir())
	t.Setenv("TBA_CONFIG_DIR", t.TempDir())

	for i := 0; i < 2; i++ {
		if _, _, err := runCmd(t, srv, "event", "list", "--no-cache"); err != nil {
			t.Fatalf("event list: %v", err)
		}
	}

	if n := countPath(requestPaths(t, srv), "/status"); n != 2 {
		t.Errorf("asked for the season %d times with --no-cache, want 2", n)
	}
}

func TestAStaleSeasonRecordIsRefreshed(t *testing.T) {
	season := otherSeason()
	srv := eventsFake(t, season)
	cacheDir := t.TempDir()
	t.Setenv("TBA_CACHE_DIR", cacheDir)
	t.Setenv("TBA_CONFIG_DIR", t.TempDir())

	// A record from two days ago is past its day.
	stale := `{"current_season":1999,"fetched_at":"2001-01-01T00:00:00Z"}`
	if err := os.WriteFile(filepath.Join(cacheDir, "season.json"), []byte(stale), 0600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	_, _, err := runCmd(t, srv, "event", "list")
	requireNoError(t, err, "")

	got := requestPaths(t, srv)
	if !contains(got, "/status") {
		t.Errorf("requested %v, want a fresh season lookup", got)
	}
	if contains(got, "/events/1999") {
		t.Error("a year-old season record was still used")
	}
}

func TestEverySeasonYearFlagSaysItDefaultsToTheCurrentSeason(t *testing.T) {
	cases := [][]string{
		{"event", "list"},
		{"team", "list"},
		{"team", "events"},
		{"team", "matches"},
		{"team", "media"},
		{"district", "list"},
		{"insight", "leaderboards"},
		{"insight", "notables"},
	}
	for _, args := range cases {
		t.Run(strings.Join(args, " "), func(t *testing.T) {
			out, _, err := runCmd(t, nil, append(args, "--help")...)
			requireNoError(t, err, "")
			requireContains(t, out, "Season year (default: current season)")
		})
	}
}

func TestTeamAwardsStillDefaultsToEveryYear(t *testing.T) {
	srv := newFakeTBA(t, map[string]any{
		"/status":             statusWithSeason(otherSeason()),
		"/team/frc177/awards": "[]",
	})

	out, _, err := runCmd(t, nil, "team", "awards", "--help")
	requireNoError(t, err, "")
	requireContains(t, out, "(default: all years)")

	_, _, err = runCmd(t, srv, "team", "awards", "177")
	requireNoError(t, err, "")
	got := requestPaths(t, srv)
	if !contains(got, "/team/frc177/awards") {
		t.Errorf("requested %v, want the all-years awards path", got)
	}
	if contains(got, "/status") {
		t.Error("team awards asked for the current season although its default is every year")
	}
}

// countPath counts how often a path was requested.
func countPath(paths []string, want string) int {
	n := 0
	for _, p := range paths {
		if p == want {
			n++
		}
	}
	return n
}
