package cmd

import (
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/the-blue-alliance/tba-cli/internal/clierr"
	"github.com/the-blue-alliance/tba-cli/internal/frc"
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
	srv := during2024ctharServer(t)
	out, _, err := runCmdAt(t, srv, time.Date(2024, 3, 23, 9, 0, 0, 0, time.Local), "team", "next", "177", "--year", "2024", "--format", "table")
	requireNoError(t, err, "")

	want := "Event:      NE District Hartford Event (2024cthar)\n" +
		"Match:      SF 13 (2024cthar_sf13m1)\n" +
		"Alliance:   red\n" +
		"Station:    R1\n" +
		"Partners:   1073, 5507\n" +
		"Opponents:  230, 195, 558\n" +
		// The match is tomorrow, so the time carries its date: "Sun 13:00" on
		// a Saturday is a weekday the reader has to work out for themselves.
		"Time:       " + localDateTime(1711299600, time.Date(2024, 3, 23, 9, 0, 0, 0, time.Local)) + " (predicted)\n"
	if !strings.HasPrefix(out, want) {
		t.Errorf("team next =\n%s\nwant it to start with\n%s", out, want)
	}
}

func TestTeamNextCountsDownToTheMatch(t *testing.T) {
	srv := during2024ctharServer(t)
	out, _, err := runCmdAt(t, srv, time.Unix(1711299600-1080, 0), "team", "next", "177", "--year", "2024", "--format", "table")
	requireNoError(t, err, "")
	requireContains(t, out, "Starts in:  18m")
}

// The weekday alone is right for the one case it was written for: a match
// today, read by someone standing at the event, who knows what day it is.
func TestTeamNextLeavesTheDateOffAMatchToday(t *testing.T) {
	srv := during2024ctharServer(t)
	out, _, err := runCmdAt(t, srv, time.Unix(1711299600-1080, 0), "team", "next", "177", "--year", "2024", "--format", "table")
	requireNoError(t, err, "")
	requireContains(t, out, "Time:       "+localTime(1711299600)+" (predicted)")
}

// A match in another season carries its year as well. "Sun 13:00" on a match
// from two seasons ago names one of a hundred Sundays.
func TestTeamNextDatesAMatchFromAnotherSeason(t *testing.T) {
	srv := newFakeTBA(t, map[string]any{
		"/event/2024cthar":                     event2024ctharJSON,
		"/team/frc177/event/2024cthar/matches": teamMatches177At2024ctharJSON,
	})
	out, _, err := runCmdAt(t, srv, time.Date(2026, 5, 1, 12, 0, 0, 0, time.Local), "team", "next", "177", "2024cthar", "--format", "table")
	requireNoError(t, err, "")

	want := time.Unix(1711299600, 0).In(time.Local).Format(frc.YearTimeLayout)
	requireContains(t, out, want+" (predicted)")
}

// A match whose time has come and gone is overdue, not "in -12m".
func TestTeamNextReportsAnOverdueMatch(t *testing.T) {
	srv := during2024ctharServer(t)
	out, _, err := runCmdAt(t, srv, time.Unix(1711299600+720, 0), "team", "next", "177", "--year", "2024", "--format", "table")
	requireNoError(t, err, "")
	requireContains(t, out, "Overdue by:  12m")
	if strings.Contains(out, "Starts in") {
		t.Errorf("an overdue match should not say when it starts:\n%s", out)
	}
}

// A named event skips looking the season up.
func TestTeamNextWithAnExplicitEvent(t *testing.T) {
	srv := newFakeTBA(t, map[string]any{
		"/event/2024cthar":                     event2024ctharJSON,
		"/team/frc177/event/2024cthar/matches": teamMatches177At2024ctharJSON,
	})
	out, _, err := runCmdAt(t, srv, time.Unix(1711299600-600, 0), "team", "next", "177", "2024cthar", "--format", "table")
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

// Out of season there is nothing to be next. That is an answer, not a failure:
// exit 0, the reason on stderr, and nothing on stdout to be piped anywhere.
func TestTeamNextWithNoCurrentOrUpcomingEvent(t *testing.T) {
	srv := newFakeTBA(t, map[string]any{
		"/team/frc177/events/2024": teamEvents177In2024JSON,
	})
	out, errOut, err := runCmdAt(t, srv, time.Date(2024, 7, 1, 12, 0, 0, 0, time.Local), "team", "next", "177", "--year", "2024", "--format", "table")
	requireNoError(t, err, errOut)

	want := "note: no current or upcoming event for team 177 in 2024; " +
		"see 'tba team events 177 --year 2024'\n"
	if errOut != want {
		t.Errorf("stderr = %q, want %q", errOut, want)
	}
	if out != "" {
		t.Errorf("stdout = %q, want nothing", out)
	}
}

// A JSON reader asked for a match and gets the JSON for "there is none".
func TestTeamNextWithNoEventPrintsNullInJSON(t *testing.T) {
	srv := newFakeTBA(t, map[string]any{
		"/team/frc177/events/2024": teamEvents177In2024JSON,
	})
	out, errOut, err := runCmdAt(t, srv, time.Date(2024, 7, 1, 12, 0, 0, 0, time.Local), "team", "next", "177", "--year", "2024", "--json")
	requireNoError(t, err, errOut)
	if strings.TrimSpace(out) != "null" {
		t.Errorf("stdout = %q, want null", out)
	}
	requireContains(t, errOut, "no current or upcoming event")
}

// Asked in the autumn, the season being searched is over, so the answer points
// at the one that is not -- and, because the next season has no schedule yet,
// at the season that does have something to show.
func TestTeamNextInTheOffseasonSuggestsTheNextSeason(t *testing.T) {
	srv := newFakeTBA(t, map[string]any{
		"/team/frc177/events/2024": teamEvents177In2024JSON,
	})
	_, errOut, err := runCmdAt(t, srv, time.Date(2024, 9, 15, 12, 0, 0, 0, time.Local), "team", "next", "177", "--year", "2024", "--format", "table")
	requireNoError(t, err, errOut)
	want := "note: no current or upcoming event for team 177 in 2024; try --year 2025; " +
		"see 'tba team events 177 --year 2024'\n"
	if errOut != want {
		t.Errorf("stderr = %q, want %q", errOut, want)
	}
}

// A season the team was never in is not a season to suggest listing, and the
// next one is a guess rather than a hint, so neither is offered.
func TestTeamNextInASeasonTheTeamSatOutSuggestsNothing(t *testing.T) {
	srv := newFakeTBA(t, map[string]any{
		"/team/frc177/events/2024": "[]",
	})
	_, errOut, err := runCmdAt(t, srv, time.Date(2024, 9, 15, 12, 0, 0, 0, time.Local), "team", "next", "177", "--year", "2024", "--format", "table")
	requireNoError(t, err, errOut)
	if want := "note: no current or upcoming event for team 177 in 2024\n"; errOut != want {
		t.Errorf("stderr = %q, want %q", errOut, want)
	}
}

// In July the next season has no schedule either, so it is not suggested; the
// season just played still is.
func TestTeamNextInJulySuggestsNoOtherSeason(t *testing.T) {
	srv := newFakeTBA(t, map[string]any{
		"/team/frc177/events/2024": teamEvents177In2024JSON,
	})
	_, errOut, err := runCmdAt(t, srv, time.Date(2024, 7, 1, 12, 0, 0, 0, time.Local), "team", "next", "177", "--year", "2024", "--format", "table")
	requireNoError(t, err, errOut)
	if strings.Contains(errOut, "try --year") {
		t.Errorf("stderr = %q, want no suggestion mid-year", errOut)
	}
	requireContains(t, errOut, "see 'tba team events 177 --year 2024'")
}

// Between events, the next one the team is going to is the answer.
func TestTeamNextPicksTheUpcomingEvent(t *testing.T) {
	srv := during2024ctharServer(t)
	out, _, err := runCmdAt(t, srv, time.Date(2024, 3, 15, 12, 0, 0, 0, time.Local), "team", "next", "177", "--year", "2024", "--format", "table")
	requireNoError(t, err, "")
	requireContains(t, out, "Event:      NE District Hartford Event (2024cthar)")
}

// Mid-event, with every match so far played, the queue simply has not posted
// the next one yet.
func TestTeamNextWhenEveryMatchIsPlayed(t *testing.T) {
	srv := newFakeTBA(t, map[string]any{
		"/team/frc177/events/2024":             teamEvents177In2024JSON,
		"/event/2024ctwat":                     event2024ctwatJSON,
		"/team/frc177/event/2024ctwat/matches": teamMatches177FinishedJSON,
	})
	out, errOut, err := runCmdAt(t, srv, time.Date(2024, 3, 9, 12, 0, 0, 0, time.Local), "team", "next", "177", "--year", "2024", "--format", "table")
	requireNoError(t, err, errOut)

	if want := "note: no upcoming match for team 177 at 2024ctwat\n"; errOut != want {
		t.Errorf("stderr = %q, want %q", errOut, want)
	}
	if out != "" {
		t.Errorf("stdout = %q, want nothing", out)
	}
	// The event is still running, so there is no reason to ask how it ended.
	for _, p := range requestPaths(t, srv) {
		if strings.HasSuffix(p, "/status") {
			t.Errorf("a running event should not be asked for its outcome, but requested %s", p)
		}
	}
}

// After the event, "no upcoming match" would read like a schedule gap. Say the
// event is over, and how it ended for this team.
func TestTeamNextAfterTheEventEnded(t *testing.T) {
	srv := newFakeTBA(t, map[string]any{
		"/team/frc177/events/2024":             teamEvents177In2024JSON,
		"/event/2024ctwat":                     event2024ctwatJSON,
		"/team/frc177/event/2024ctwat/matches": teamMatches177FinishedJSON,
		"/team/frc177/event/2024ctwat/status":  teamStatus177At2024ctharJSON,
	})
	_, errOut, err := runCmdAt(t, srv, time.Date(2024, 3, 12, 9, 0, 0, 0, time.Local), "team", "next", "177", "2024ctwat", "--format", "table")
	requireNoError(t, err, errOut)

	want := "note: 2024ctwat ended 2024-03-10; no matches left for 177; 177 won the event\n"
	if errOut != want {
		t.Errorf("stderr = %q, want %q", errOut, want)
	}
}

// A team knocked out is told which round it went out in.
func TestTeamNextAfterTheEventEndedForAnEliminatedTeam(t *testing.T) {
	srv := newFakeTBA(t, map[string]any{
		"/event/2024ctwat":                     event2024ctwatJSON,
		"/team/frc177/event/2024ctwat/matches": teamMatches177FinishedJSON,
		"/team/frc177/event/2024ctwat/status":  teamStatusBackupJSON,
	})
	_, errOut, err := runCmdAt(t, srv, time.Date(2024, 3, 12, 9, 0, 0, 0, time.Local), "team", "next", "177", "2024ctwat", "--format", "table")
	requireNoError(t, err, errOut)

	want := "note: 2024ctwat ended 2024-03-10; no matches left for 177; eliminated in SF\n"
	if errOut != want {
		t.Errorf("stderr = %q, want %q", errOut, want)
	}
}

// The outcome is worth one extra request, not the answer: without it the rest
// of the sentence still stands.
func TestTeamNextAfterTheEventEndedWithoutAStatus(t *testing.T) {
	srv := newFakeTBA(t, map[string]any{
		"/event/2024ctwat":                     event2024ctwatJSON,
		"/team/frc177/event/2024ctwat/matches": teamMatches177FinishedJSON,
	})
	_, errOut, err := runCmdAt(t, srv, time.Date(2024, 3, 12, 9, 0, 0, 0, time.Local), "team", "next", "177", "2024ctwat", "--format", "table")
	requireNoError(t, err, errOut)

	want := "note: 2024ctwat ended 2024-03-10; no matches left for 177\n"
	if errOut != want {
		t.Errorf("stderr = %q, want %q", errOut, want)
	}
}

// A team that never went to the event has no matches there, which used to read
// as "the schedule is not out yet". The event's roster says otherwise.
func TestTeamNextForATeamNotAtTheEvent(t *testing.T) {
	srv := newFakeTBA(t, map[string]any{
		"/event/2024cthar":                     event2024ctharJSON,
		"/team/frc254/event/2024cthar/matches": "[]",
		"/event/2024cthar/teams/keys":          `["frc177", "frc1073", "frc5507"]`,
	})
	out, _, err := runCmdAt(t, srv, time.Date(2024, 3, 23, 9, 0, 0, 0, time.Local), "team", "next", "254", "2024cthar", "--format", "table")
	requireErrorContains(t, err, "team 254 was not at 2024cthar")
	if got := clierr.ExitCode(err); got != clierr.ExitNotFound {
		t.Errorf("exit code = %d, want %d (not found)", got, clierr.ExitNotFound)
	}
	if out != "" {
		t.Errorf("stdout = %q, want nothing", out)
	}
}

// A team that is on the roster but has no schedule yet is a different answer,
// and still a successful one.
func TestTeamNextForATeamAtTheEventWithNoScheduleYet(t *testing.T) {
	srv := newFakeTBA(t, map[string]any{
		"/event/2024cthar":                     event2024ctharJSON,
		"/team/frc177/event/2024cthar/matches": "[]",
		"/event/2024cthar/teams/keys":          `["frc177", "frc1073", "frc5507"]`,
	})
	_, errOut, err := runCmdAt(t, srv, time.Date(2024, 3, 23, 9, 0, 0, 0, time.Local), "team", "next", "177", "2024cthar", "--format", "table")
	requireNoError(t, err, errOut)
	if want := "note: no upcoming match for team 177 at 2024cthar\n"; errOut != want {
		t.Errorf("stderr = %q, want %q", errOut, want)
	}
}

// The roster costs a request, so it is only fetched when the match list came
// back empty.
func TestTeamNextSkipsTheRosterWhenThereAreMatches(t *testing.T) {
	srv := during2024ctharServer(t)
	_, errOut, err := runCmdAt(t, srv, time.Date(2024, 3, 23, 9, 0, 0, 0, time.Local), "team", "next", "177", "2024cthar", "--format", "table")
	requireNoError(t, err, errOut)
	for _, p := range requestPaths(t, srv) {
		if strings.HasSuffix(p, "/teams/keys") {
			t.Errorf("a team with matches should not need the roster, but requested %s", p)
		}
	}
}

// --all takes the same answer: an empty table for a team that was never there
// would be a listing of nothing rather than a correction.
func TestTeamNextAllForATeamNotAtTheEvent(t *testing.T) {
	srv := newFakeTBA(t, map[string]any{
		"/event/2024cthar":                     event2024ctharJSON,
		"/team/frc254/event/2024cthar/matches": "[]",
		"/event/2024cthar/teams/keys":          `["frc177"]`,
	})
	_, _, err := runCmdAt(t, srv, time.Date(2024, 3, 23, 9, 0, 0, 0, time.Local), "team", "next", "254", "2024cthar", "--all", "--format", "csv")
	if got := clierr.ExitCode(err); got != clierr.ExitNotFound {
		t.Errorf("exit code = %d, want %d (not found)", got, clierr.ExitNotFound)
	}
}

// --all answers "what is left" rather than "what is now", as the same table
// every other match listing uses.
func TestTeamNextAll(t *testing.T) {
	srv := during2024ctharServer(t)
	out, _, err := runCmdAt(t, srv, time.Date(2024, 3, 23, 9, 0, 0, 0, time.Local), "team", "next", "177", "--year", "2024", "--all", "--format", "csv")
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
	srv := newFakeTBA(t, map[string]any{
		"/team/frc177/events/2024":             teamEvents177In2024JSON,
		"/event/2024ctwat":                     event2019ctwatJSON,
		"/team/frc177/event/2024ctwat/matches": teamMatches177FinishedJSON,
	})
	out, _, err := runCmdAt(t, srv, time.Date(2024, 3, 9, 12, 0, 0, 0, time.Local), "team", "next", "177", "--year", "2024", "--all", "--format", "csv")
	requireNoError(t, err, "")
	if got := csvColumn(t, out, 1); len(got) != 0 {
		t.Errorf("matches = %v, want none", got)
	}
}

// The JSON form is the match itself, so it can be piped into anything that
// already understands a TBA match.
func TestTeamNextJSON(t *testing.T) {
	srv := during2024ctharServer(t)
	out, _, err := runCmdAt(t, srv, time.Date(2024, 3, 23, 9, 0, 0, 0, time.Local), "team", "next", "177", "--year", "2024", "--json")
	requireNoError(t, err, "")

	obj := decodeJSON(t, out).(map[string]any)
	if obj["key"] != "2024cthar_sf13m1" {
		t.Errorf("key = %v", obj["key"])
	}
}

func TestTeamNextAcceptsBothTeamSpellings(t *testing.T) {
	for _, arg := range []string{"177", "frc177"} {
		t.Run(arg, func(t *testing.T) {
			srv := during2024ctharServer(t)
			_, _, err := runCmdAt(t, srv, time.Date(2024, 3, 23, 9, 0, 0, 0, time.Local), "team", "next", arg, "--year", "2024")
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
	srv := during2024ctharServer(t)
	out, _, err := runCmdAt(t, srv, time.Date(2024, 3, 23, 9, 0, 0, 0, time.Local), "team", "next", "177", "--year", "2024", "--format", "table", "--color", "always")
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
