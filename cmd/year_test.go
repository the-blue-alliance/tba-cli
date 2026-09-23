package cmd

import (
	"context"
	"errors"
	"fmt"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/spf13/cobra"

	"github.com/the-blue-alliance/tba-cli/internal/api"
	"github.com/the-blue-alliance/tba-cli/internal/clierr"
)

// otherSeason is a season that is not the calendar year, so that a test can
// tell "the API told us" apart from "we guessed from the clock".
func otherSeason() int { return thisYear() - 1 }

// statusWithSeason is a /status body naming a season.
func statusWithSeason(year int) string {
	return fmt.Sprintf(`{"current_season":%d,"max_season":%d,"is_datafeed_down":false}`, year, year)
}

// eventsFake serves /status naming season, plus an empty event list for that
// season, for the calendar year and for any other year the test asks about.
func eventsFake(t *testing.T, season int, years ...int) *httptest.Server {
	t.Helper()
	routes := map[string]any{"/status": statusWithSeason(season)}
	for _, y := range append([]int{season, thisYear()}, years...) {
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
	if contains(got, fmt.Sprintf("/events/%d", thisYear())) {
		t.Error("the calendar year was used even though /status answered")
	}
}

func TestYearFallsBackToTheCalendarWhenStatusFails(t *testing.T) {
	season := otherSeason()
	srv := eventsFake(t, season)
	setStatus(t, srv, "/status", 500)

	_, _, err := runCmd(t, srv, "event", "list", "--retries", "0")
	requireNoError(t, err, "")

	want := fmt.Sprintf("/events/%d", thisYear())
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
	requireErrorContains(t, err, fmt.Sprintf("%d", thisYear()))
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

// A year outside the range FRC seasons are named in is a typo, a number that
// was meant to be something else, or an event key that lost its letters.
// Turning it into a request produces an empty list and no explanation.
func TestYearRejectsAYearThatIsNotASeason(t *testing.T) {
	cases := []struct{ name, year string }{
		{"before the first season", "1800"},
		{"the year before the first season", "1991"},
		{"far in the future", "3000"},
		{"two years out", strconv.Itoa(thisYear() + 2)},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			srv := eventsFake(t, otherSeason())
			_, _, err := runCmd(t, srv, "event", "list", "--year", c.year)

			want := fmt.Sprintf("--year %s is not an FRC season (1992-%d)", c.year, thisYear()+1)
			if err == nil || err.Error() != want {
				t.Errorf("error = %v, want %q", err, want)
			}
			if got := clierr.ExitCode(err); got != clierr.ExitUsage {
				t.Errorf("exit code = %d, want %d", got, clierr.ExitUsage)
			}
			if got := requestPaths(t, srv); len(got) != 0 {
				t.Errorf("a bad --year should not reach the API, got %v", got)
			}
		})
	}
}

// The season after this calendar year is a real thing to ask about: a season
// is named after the year it ends in, so from kickoff in January the coming
// season already has events in it.
func TestYearAcceptsNextSeason(t *testing.T) {
	next := thisYear() + 1
	srv := eventsFake(t, otherSeason(), next)

	_, _, err := runCmd(t, srv, "event", "list", "--year", strconv.Itoa(next))
	requireNoError(t, err, "")
	if got := requestPaths(t, srv); !contains(got, fmt.Sprintf("/events/%d", next)) {
		t.Errorf("requested %v, want next season", got)
	}
}

func TestYearRejectsANonSeasonFromTheEnvironment(t *testing.T) {
	srv := eventsFake(t, otherSeason())
	t.Setenv("TBA_YEAR", "1800")

	_, _, err := runCmd(t, srv, "event", "list")
	requireErrorContains(t, err, "TBA_YEAR 1800 is not an FRC season")
	if got := clierr.ExitCode(err); got != clierr.ExitUsage {
		t.Errorf("exit code = %d, want %d", got, clierr.ExitUsage)
	}
}

// Ctrl-C during the season lookup used to be swallowed with every other
// failure, so the command guessed a year and made another request before
// giving up, instead of stopping when it was told to.
func TestYearSurfacesACancelledSeasonLookup(t *testing.T) {
	srv := eventsFake(t, otherSeason())
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, _, err := runCtx(t, ctx, srv.URL, "event", "list")
	if err == nil {
		t.Fatal("want an error when the context is cancelled")
	}
	requireErrorContains(t, err, "looking up the current season")
	if !errors.Is(err, context.Canceled) {
		t.Errorf("error = %v, want one wrapping context.Canceled", err)
	}
	if got := clierr.ExitCode(err); got != clierr.ExitInterrupt {
		t.Errorf("exit code = %d, want %d", got, clierr.ExitInterrupt)
	}
	for _, p := range requestPaths(t, srv) {
		if strings.HasPrefix(p, "/events/") {
			t.Errorf("a cancelled lookup should not be followed by %s", p)
		}
	}
}

// A command that already has a client can lend it, so the season lookup is
// paced by the same rate limiter as everything else the command does rather
// than by one of its own.
func TestResolveYearUsesALentClient(t *testing.T) {
	season := otherSeason()
	srv := newFakeTBA(t, map[string]any{"/status": statusWithSeason(season)})
	t.Setenv("TBA_AUTH_KEY", "test-key")
	t.Setenv("TBA_CACHE_DIR", t.TempDir())
	t.Setenv("TBA_CONFIG_DIR", t.TempDir())

	client, err := api.NewClient(srv.URL, api.WithUserAgent("lent-client"))
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	cmd := &cobra.Command{Use: "test"}
	addYearFlag(cmd)
	cmd.SetContext(context.Background())

	got, err := resolveYear(cmd, client)
	requireNoError(t, err, "")
	if got != season {
		t.Errorf("year = %d, want %d", got, season)
	}
	reqs := requestsTo(t, srv)
	if len(reqs) != 1 {
		t.Fatalf("%d requests, want 1", len(reqs))
	}
	if ua := reqs[0].Headers.Get("User-Agent"); ua != "lent-client" {
		t.Errorf("User-Agent = %q, want the lent client's: a second client was built", ua)
	}
}

// Every command that resolves the default season hands the lookup the client
// it already built, so the extra /status request is paced by the same rate
// limiter as the rest of the command instead of by a second client's.
//
// What a command-level test can see is the request stream: /status goes out
// first, before the command's own request, and every request carries the same
// User-Agent. That the lookup really borrows the client rather than building
// an identical one is pinned by TestResolveYearUsesALentClient above, which
// gives the lent client a User-Agent of its own.
func TestCommandsLendTheirClientToTheSeasonLookup(t *testing.T) {
	season := otherSeason()
	cases := []struct {
		name string
		path string
		args []string
	}{
		{"district list", fmt.Sprintf("/districts/%d", season), []string{"district", "list"}},
		{"event list", fmt.Sprintf("/events/%d", season), []string{"event", "list"}},
		{"team events", fmt.Sprintf("/team/frc177/events/%d", season), []string{"team", "events", "177"}},
		{"team media", fmt.Sprintf("/team/frc177/media/%d", season), []string{"team", "media", "177"}},
		{"team matches", fmt.Sprintf("/team/frc177/matches/%d", season), []string{"team", "matches", "177"}},
		{"team search", fmt.Sprintf("/teams/%d/0", season), []string{"team", "search", "bobcat"}},
		{"insight leaderboards", fmt.Sprintf("/insights/leaderboards/%d", season), []string{"insight", "leaderboards"}},
		{"insight notables", fmt.Sprintf("/insights/notables/%d", season), []string{"insight", "notables"}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			srv := newFakeTBA(t, map[string]any{
				"/status": statusWithSeason(season),
				c.path:    "[]",
			})
			_, errOut, err := runCmd(t, srv, append(c.args, "--no-cache")...)
			requireNoError(t, err, errOut)

			reqs := requestsTo(t, srv)
			if len(reqs) < 2 {
				t.Fatalf("%d requests, want the season lookup and the command's own", len(reqs))
			}
			if reqs[0].Path != "/status" {
				t.Errorf("first request = %s, want /status", reqs[0].Path)
			}
			if reqs[1].Path != c.path {
				t.Errorf("second request = %s, want %s", reqs[1].Path, c.path)
			}
			ua := reqs[0].Headers.Get("User-Agent")
			if ua == "" {
				t.Fatal("the season lookup sent no User-Agent")
			}
			for _, r := range reqs[1:] {
				if got := r.Headers.Get("User-Agent"); got != ua {
					t.Errorf("%s sent User-Agent %q, want the same client's %q", r.Path, got, ua)
				}
			}
		})
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
