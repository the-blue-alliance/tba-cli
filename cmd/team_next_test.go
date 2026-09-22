package cmd

import (
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/the-blue-alliance/tba-cli/internal/clierr"
)

// during2024ctharServer serves team 177's season and its matches at the event
// it is competing at.
func during2024ctharServer(t *testing.T) *httptest.Server {
	t.Helper()
	return newFakeTBA(t, map[string]any{
		"/team/frc177/events/2024":             teamEvents177In2024JSON,
		"/event/2024cthar":                     event2024ctharJSON,
		"/team/frc177/event/2024cthar/matches": teamMatches177At2024ctharJSON,
	})
}

// With no event named, the team's event for today is the one that matters.
func TestTeamNextAutoDetectsTheCurrentEvent(t *testing.T) {
	withNow(t, time.Date(2024, 3, 23, 9, 0, 0, 0, time.Local))
	srv := during2024ctharServer(t)
	out, _, err := runCmd(t, srv, "team", "next", "177", "--year", "2024", "--format", "table")
	requireNoError(t, err, "")

	want := "Event:      NE District Hartford Event (2024cthar)\n" +
		"Match:      SF 13 (2024cthar_sf13m1)\n" +
		"Alliance:   red\n" +
		"Station:    R1\n" +
		"Partners:   1073, 5507\n" +
		"Opponents:  230, 195, 558\n" +
		"Time:       " + localTime(1711299600) + " (predicted)\n"
	if !strings.HasPrefix(out, want) {
		t.Errorf("team next =\n%s\nwant it to start with\n%s", out, want)
	}
}

func TestTeamNextCountsDownToTheMatch(t *testing.T) {
	withNow(t, time.Unix(1711299600-1080, 0))
	srv := during2024ctharServer(t)
	out, _, err := runCmd(t, srv, "team", "next", "177", "--year", "2024", "--format", "table")
	requireNoError(t, err, "")
	requireContains(t, out, "Starts in:  18m")
}

// A match whose time has come and gone is overdue, not "in -12m".
func TestTeamNextReportsAnOverdueMatch(t *testing.T) {
	withNow(t, time.Unix(1711299600+720, 0))
	srv := during2024ctharServer(t)
	out, _, err := runCmd(t, srv, "team", "next", "177", "--year", "2024", "--format", "table")
	requireNoError(t, err, "")
	requireContains(t, out, "Overdue by:  12m")
	if strings.Contains(out, "Starts in") {
		t.Errorf("an overdue match should not say when it starts:\n%s", out)
	}
}

// A named event skips looking the season up.
func TestTeamNextWithAnExplicitEvent(t *testing.T) {
	withNow(t, time.Unix(1711299600-600, 0))
	srv := newFakeTBA(t, map[string]any{
		"/event/2024cthar":                     event2024ctharJSON,
		"/team/frc177/event/2024cthar/matches": teamMatches177At2024ctharJSON,
	})
	out, _, err := runCmd(t, srv, "team", "next", "177", "2024cthar", "--format", "table")
	requireNoError(t, err, "")

	for _, p := range requestPaths(t, srv) {
		if strings.Contains(p, "/events/") {
			t.Errorf("a named event should not need the season listing, but requested %s", p)
		}
	}
	requireContains(t, out, "Match:      SF 13 (2024cthar_sf13m1)")
}

func TestTeamNextRejectsAMalformedEventKey(t *testing.T) {
	srv := newFakeTBA(t, map[string]any{})
	_, _, err := runCmd(t, srv, "team", "next", "177", "not-a-key")
	requireErrorContains(t, err, "is not a valid event key")
	if got := requestPaths(t, srv); len(got) != 0 {
		t.Errorf("a malformed key should not reach the API, got %v", got)
	}
}

// Out of season there is nothing to be next.
func TestTeamNextWithNoCurrentOrUpcomingEvent(t *testing.T) {
	withNow(t, time.Date(2024, 7, 1, 12, 0, 0, 0, time.Local))
	srv := newFakeTBA(t, map[string]any{
		"/team/frc177/events/2024": teamEvents177In2024JSON,
	})
	_, _, err := runCmd(t, srv, "team", "next", "177", "--year", "2024")
	requireErrorContains(t, err, "no current or upcoming event for team 177 in 2024")
	if got := clierr.ExitCode(err); got != clierr.ExitFailure {
		t.Errorf("exit code = %d, want %d", got, clierr.ExitFailure)
	}
}

// Between events, the next one the team is going to is the answer.
func TestTeamNextPicksTheUpcomingEvent(t *testing.T) {
	withNow(t, time.Date(2024, 3, 15, 12, 0, 0, 0, time.Local))
	srv := during2024ctharServer(t)
	out, _, err := runCmd(t, srv, "team", "next", "177", "--year", "2024", "--format", "table")
	requireNoError(t, err, "")
	requireContains(t, out, "Event:      NE District Hartford Event (2024cthar)")
}

func TestTeamNextWhenEveryMatchIsPlayed(t *testing.T) {
	withNow(t, time.Date(2024, 3, 9, 12, 0, 0, 0, time.Local))
	srv := newFakeTBA(t, map[string]any{
		"/team/frc177/events/2024":             teamEvents177In2024JSON,
		"/event/2024ctwat":                     event2019ctwatJSON,
		"/team/frc177/event/2024ctwat/matches": teamMatches177FinishedJSON,
	})
	_, _, err := runCmd(t, srv, "team", "next", "177", "--year", "2024")
	requireErrorContains(t, err, "no upcoming match for team 177 at 2024ctwat")
	if got := clierr.ExitCode(err); got != clierr.ExitFailure {
		t.Errorf("exit code = %d, want %d", got, clierr.ExitFailure)
	}
}

// --all answers "what is left" rather than "what is now", as the same table
// every other match listing uses.
func TestTeamNextAll(t *testing.T) {
	withNow(t, time.Date(2024, 3, 23, 9, 0, 0, 0, time.Local))
	srv := during2024ctharServer(t)
	out, _, err := runCmd(t, srv, "team", "next", "177", "--year", "2024", "--all", "--format", "csv")
	requireNoError(t, err, "")

	records := parseCSV(t, out)
	if records[0][0] != "Match" || records[0][1] != "Key" {
		t.Errorf("header = %v, want the shared match table", records[0])
	}
	if got := csvColumn(t, out, 1); !equalStrings(got, []string{"2024cthar_sf13m1"}) {
		t.Errorf("matches = %v, want only the unplayed one", got)
	}
}

// With nothing left to play, --all is an empty listing rather than an error:
// "no rows" is a perfectly good answer to "what is left".
func TestTeamNextAllWithNothingLeft(t *testing.T) {
	withNow(t, time.Date(2024, 3, 9, 12, 0, 0, 0, time.Local))
	srv := newFakeTBA(t, map[string]any{
		"/team/frc177/events/2024":             teamEvents177In2024JSON,
		"/event/2024ctwat":                     event2019ctwatJSON,
		"/team/frc177/event/2024ctwat/matches": teamMatches177FinishedJSON,
	})
	out, _, err := runCmd(t, srv, "team", "next", "177", "--year", "2024", "--all", "--format", "csv")
	requireNoError(t, err, "")
	if got := csvColumn(t, out, 1); len(got) != 0 {
		t.Errorf("matches = %v, want none", got)
	}
}

// The JSON form is the match itself, so it can be piped into anything that
// already understands a TBA match.
func TestTeamNextJSON(t *testing.T) {
	withNow(t, time.Date(2024, 3, 23, 9, 0, 0, 0, time.Local))
	srv := during2024ctharServer(t)
	out, _, err := runCmd(t, srv, "team", "next", "177", "--year", "2024", "--json")
	requireNoError(t, err, "")

	obj := decodeJSON(t, out).(map[string]any)
	if obj["key"] != "2024cthar_sf13m1" {
		t.Errorf("key = %v", obj["key"])
	}
}

func TestTeamNextAcceptsBothTeamSpellings(t *testing.T) {
	withNow(t, time.Date(2024, 3, 23, 9, 0, 0, 0, time.Local))
	for _, arg := range []string{"177", "frc177"} {
		t.Run(arg, func(t *testing.T) {
			srv := during2024ctharServer(t)
			_, _, err := runCmd(t, srv, "team", "next", arg, "--year", "2024")
			requireNoError(t, err, "")
			for _, p := range requestPaths(t, srv) {
				if strings.Contains(p, "frcfrc") {
					t.Errorf("request path was double-prefixed: %s", p)
				}
			}
		})
	}
}

func TestTeamNextColorsTheAlliance(t *testing.T) {
	withNow(t, time.Date(2024, 3, 23, 9, 0, 0, 0, time.Local))
	srv := during2024ctharServer(t)
	out, _, err := runCmd(t, srv, "team", "next", "177", "--year", "2024", "--format", "table", "--color", "always")
	requireNoError(t, err, "")
	if !strings.Contains(out, "\x1b[31mred\x1b[0m") {
		t.Errorf("the alliance is not colored:\n%q", out)
	}
}

// Without --year, the current season is asked for, not year zero.
func TestTeamNextDefaultsToTheCurrentSeason(t *testing.T) {
	srv := newFakeTBA(t, map[string]any{})
	_, _, _ = runCmd(t, srv, "team", "next", "177")
	paths := requestPaths(t, srv)
	if len(paths) == 0 {
		t.Fatal("no request was made")
	}
	if strings.HasSuffix(paths[0], "/0") {
		t.Errorf("requested %q, want the current season", paths[0])
	}
}
