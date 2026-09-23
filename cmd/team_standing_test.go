package cmd

import (
	"strings"
	"testing"
	"time"

	"github.com/the-blue-alliance/tba-cli/internal/clierr"
)

func TestTeamStanding(t *testing.T) {
	srv := newFakeTBA(t, map[string]any{
		"/team/frc177/event/2024cthar/status": teamStatus177At2024ctharJSON,
	})
	out, _, err := runCmd(t, srv, "team", "standing", "177", "--event", "2024cthar", "--format", "table")
	requireNoError(t, err, "")

	want := "Team:           177\n" +
		"Event:          2024cthar\n" +
		"Rank:           1 of 40\n" +
		"Record:         10-2-0\n" +
		"Played:         12\n" +
		"Ranking Score:  2.50\n" +
		"Avg Match:      88\n" +
		"Avg Auto:       31.0\n" +
		"Alliance:       Alliance 1 (Captain)\n" +
		"Playoff:        Finals — won (6-1-0)\n" +
		"Status:         Team 177 was Rank 1 with a record of 10-2-0 and won the event.\n"
	if out != want {
		t.Errorf("team standing =\n%s\nwant\n%s", out, want)
	}
}

// The sort orders are the season's own tiebreakers, so they are named and
// rounded the way the event says.
func TestTeamStandingNamesTheSortOrders(t *testing.T) {
	srv := newFakeTBA(t, map[string]any{
		"/team/frc177/event/2024cthar/status": teamStatus177At2024ctharJSON,
	})
	out, _, err := runCmd(t, srv, "team", "standing", "177", "--event", "2024cthar", "--format", "table")
	requireNoError(t, err, "")

	// Ranking Score has precision 2, Avg Match precision 0, Avg Auto 1.
	for _, want := range []string{"Ranking Score:  2.50", "Avg Match:      88", "Avg Auto:       31.0"} {
		requireContains(t, out, want)
	}
}

// Before an event starts the API has nothing to say, and each silence means
// something different.
func TestTeamStandingBeforeTheEventStarts(t *testing.T) {
	srv := newFakeTBA(t, map[string]any{
		"/team/frc177/event/2024cthar/status": teamStatusNotStartedJSON,
	})
	out, _, err := runCmd(t, srv, "team", "standing", "177", "--event", "2024cthar", "--format", "table")
	requireNoError(t, err, "")

	want := "Team:      177\n" +
		"Event:     2024cthar\n" +
		"Rank:      not ranked yet\n" +
		"Alliance:  not selected\n" +
		"Playoff:   not started\n"
	if out != want {
		t.Errorf("team standing =\n%s\nwant\n%s", out, want)
	}
}

// Mid-qualification: ranked, but alliance selection has not happened.
func TestTeamStandingDuringQualification(t *testing.T) {
	srv := newFakeTBA(t, map[string]any{
		"/team/frc177/event/2024cthar/status": teamStatusQualsOnlyJSON,
	})
	out, _, err := runCmd(t, srv, "team", "standing", "177", "--event", "2024cthar", "--format", "table")
	requireNoError(t, err, "")

	for _, want := range []string{
		"Rank:           7 of 40",
		"Record:         3-2-1",
		"Played:         6",
		"Ranking Score:  1.83",
		"Alliance:       not selected",
		"Playoff:        not started",
		"Status:         Team 177 is Rank 7 with a record of 3-2-1.",
	} {
		requireContains(t, out, want)
	}
}

// A backup is neither a captain nor a pick; it was called in.
func TestTeamStandingForABackupTeam(t *testing.T) {
	srv := newFakeTBA(t, map[string]any{
		"/team/frc177/event/2024cthar/status": teamStatusBackupJSON,
	})
	out, _, err := runCmd(t, srv, "team", "standing", "177", "--event", "2024cthar", "--format", "table")
	requireNoError(t, err, "")
	requireContains(t, out, "Alliance:  Alliance 3 (backup)")
	requireContains(t, out, "Playoff:   Semifinals — eliminated (3-2-0)")
}

// The API writes its status strings for a web page; a terminal cannot render
// the markup.
func TestTeamStandingStripsHTMLFromTheStatusString(t *testing.T) {
	srv := newFakeTBA(t, map[string]any{
		"/team/frc177/event/2024cthar/status": teamStatus177At2024ctharJSON,
	})
	out, _, err := runCmd(t, srv, "team", "standing", "177", "--event", "2024cthar", "--format", "table")
	requireNoError(t, err, "")
	if strings.Contains(out, "<b>") || strings.Contains(out, "</b>") {
		t.Errorf("HTML leaked into the output:\n%s", out)
	}
}

// The API answers a team that never went to an event with a bare null, not a
// 404. Decoded into the status struct it reads as "not ranked yet, not
// selected, playoffs not started", which is a whole standing for a team that
// was never there.
func TestTeamStandingRejectsANullStatus(t *testing.T) {
	srv := newFakeTBA(t, map[string]any{
		"/team/frc254/event/2024cthar/status": "null",
	})
	out, _, err := runCmd(t, srv, "team", "standing", "254", "2024cthar", "--format", "table")
	requireErrorContains(t, err, "team 254 was not at 2024cthar")
	if got := clierr.ExitCode(err); got != clierr.ExitNotFound {
		t.Errorf("exit code = %d, want %d (not found)", got, clierr.ExitNotFound)
	}
	if out != "" {
		t.Errorf("stdout = %q, want nothing", out)
	}
}

// The JSON form used to print an object of nulls, which a script reads as a
// real standing.
func TestTeamStandingRejectsANullStatusInJSON(t *testing.T) {
	srv := newFakeTBA(t, map[string]any{
		"/team/frc254/event/2024cthar/status": "null",
	})
	out, _, err := runCmd(t, srv, "team", "standing", "254", "2024cthar", "--json")
	if got := clierr.ExitCode(err); got != clierr.ExitNotFound {
		t.Errorf("exit code = %d, want %d (not found)", got, clierr.ExitNotFound)
	}
	if out != "" {
		t.Errorf("stdout = %q, want nothing", out)
	}
}

// A team that is at the event but has not played yet gets an object whose
// sections are all null, which is a different answer from the bare null and
// stays a successful one.
func TestTeamStandingKeepsTheNotStartedObject(t *testing.T) {
	srv := newFakeTBA(t, map[string]any{
		"/team/frc177/event/2024cthar/status": teamStatusNotStartedJSON,
	})
	out, errOut, err := runCmd(t, srv, "team", "standing", "177", "2024cthar", "--format", "table")
	requireNoError(t, err, errOut)
	requireContains(t, out, "Rank:      not ranked yet")
}

// A team that is not attending the event has no status there.
func TestTeamStandingForATeamNotAtTheEvent(t *testing.T) {
	srv := newFakeTBA(t, map[string]any{})
	_, _, err := runCmd(t, srv, "team", "standing", "9999", "--event", "2024cthar")
	if got := clierr.ExitCode(err); got != clierr.ExitNotFound {
		t.Errorf("exit code = %d, want %d (not found)", got, clierr.ExitNotFound)
	}
}

// The event is an argument like everywhere else in the team commands, and
// --event still works for anyone who has typed it that way for a season.
func TestTeamStandingTakesTheEventAsAnArgument(t *testing.T) {
	srv := newFakeTBA(t, map[string]any{
		"/team/frc177/event/2024cthar/status": teamStatus177At2024ctharJSON,
	})
	out, errOut, err := runCmd(t, srv, "team", "standing", "177", "2024cthar", "--format", "table")
	requireNoError(t, err, errOut)
	requireContains(t, out, "Event:          2024cthar")

	want := []string{"/team/frc177/event/2024cthar/status"}
	if got := requestPaths(t, srv); !equalStrings(got, want) {
		t.Errorf("requests = %v, want %v", got, want)
	}
}

// Named twice and differently, there is no way to tell which was meant.
func TestTeamStandingRejectsTheEventGivenTwice(t *testing.T) {
	srv := newFakeTBA(t, map[string]any{})
	_, _, err := runCmd(t, srv, "team", "standing", "177", "2024cthar", "--event", "2024necmp")
	requireErrorContains(t, err, "event given twice")
	if got := clierr.ExitCode(err); got != clierr.ExitUsage {
		t.Errorf("exit code = %d, want %d (usage)", got, clierr.ExitUsage)
	}
	if got := requestPaths(t, srv); len(got) != 0 {
		t.Errorf("a usage error should not reach the API, got %v", got)
	}
}

// The same value twice is not a contradiction, so it is not an error.
func TestTeamStandingAcceptsTheSameEventTwice(t *testing.T) {
	srv := newFakeTBA(t, map[string]any{
		"/team/frc177/event/2024cthar/status": teamStatus177At2024ctharJSON,
	})
	_, errOut, err := runCmd(t, srv, "team", "standing", "177", "2024cthar", "--event", "2024cthar")
	requireNoError(t, err, errOut)
}

// With no event at all, the question is about wherever the team is now — the
// same answer `team next` works out.
func TestTeamStandingAutoDetectsTheCurrentEvent(t *testing.T) {
	srv := newFakeTBA(t, map[string]any{
		"/team/frc177/events/2024":            teamEvents177In2024JSON,
		"/team/frc177/event/2024cthar/status": teamStatus177At2024ctharJSON,
	})
	out, errOut, err := runCmdAt(t, srv, time.Date(2024, 3, 23, 9, 0, 0, 0, time.Local), "team", "standing", "177", "--year", "2024", "--format", "table")
	requireNoError(t, err, errOut)
	requireContains(t, out, "Event:          2024cthar")
}

// Out of season there is no event to stand at, which is an answer rather than
// a failure, exactly as it is for `team next`.
func TestTeamStandingWithNoCurrentEvent(t *testing.T) {
	srv := newFakeTBA(t, map[string]any{
		"/team/frc177/events/2024": teamEvents177In2024JSON,
	})
	out, errOut, err := runCmdAt(t, srv, time.Date(2024, 7, 1, 12, 0, 0, 0, time.Local), "team", "standing", "177", "--year", "2024", "--format", "table")
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

// The API pads sort_orders past the names the season defines. An unnamed
// number is not a statistic, and "Sort Order 6: 0.00" invented one.
func TestTeamStandingOnlyPrintsNamedSortOrders(t *testing.T) {
	srv := newFakeTBA(t, map[string]any{
		"/team/frc177/event/2024cthar/status": teamStatusExtraSortOrderJSON,
	})
	out, errOut, err := runCmd(t, srv, "team", "standing", "177", "2024cthar", "--format", "table")
	requireNoError(t, err, errOut)

	for _, want := range []string{
		"Ranking Score:  2.50",
		"Avg Coop:       0.50",
		"Avg Match:      88",
		"Avg Auto:       31.0",
		"Avg Stage:      12.0",
	} {
		requireContains(t, out, want)
	}
	if strings.Contains(out, "Sort Order") {
		t.Errorf("an unnamed sort order was printed:\n%s", out)
	}
}

// Fewer numbers than names is the other way round, and must not run off the
// end of the list.
func TestTeamStandingSurvivesFewerSortOrdersThanNames(t *testing.T) {
	srv := newFakeTBA(t, map[string]any{
		"/team/frc177/event/2024cthar/status": teamStatusQualsOnlyJSON,
	})
	out, errOut, err := runCmd(t, srv, "team", "standing", "177", "2024cthar", "--format", "table")
	requireNoError(t, err, errOut)
	requireContains(t, out, "Ranking Score:  1.83")
	requireContains(t, out, "Avg Match:      62")
}

func TestTeamStandingRejectsAMalformedEventKey(t *testing.T) {
	srv := newFakeTBA(t, map[string]any{})
	_, _, err := runCmd(t, srv, "team", "standing", "177", "--event", "not-a-key")
	requireErrorContains(t, err, "is not a valid event key")
	if got := requestPaths(t, srv); len(got) != 0 {
		t.Errorf("a malformed key should not reach the API, got %v", got)
	}
}

// The JSON form is the API's own answer, unshaped.
func TestTeamStandingJSON(t *testing.T) {
	srv := newFakeTBA(t, map[string]any{
		"/team/frc177/event/2024cthar/status": teamStatus177At2024ctharJSON,
	})
	out, _, err := runCmd(t, srv, "team", "standing", "177", "--event", "2024cthar", "--json")
	requireNoError(t, err, "")

	obj := decodeJSON(t, out).(map[string]any)
	qual := obj["qual"].(map[string]any)
	ranking := qual["ranking"].(map[string]any)
	if ranking["rank"] != float64(1) {
		t.Errorf("rank = %v", ranking["rank"])
	}
	alliance := obj["alliance"].(map[string]any)
	if alliance["number"] != float64(1) {
		t.Errorf("alliance number = %v", alliance["number"])
	}
}

func TestTeamStandingAcceptsBothTeamSpellings(t *testing.T) {
	for _, arg := range []string{"177", "frc177"} {
		t.Run(arg, func(t *testing.T) {
			srv := newFakeTBA(t, map[string]any{
				"/team/frc177/event/2024cthar/status": teamStatus177At2024ctharJSON,
			})
			_, _, err := runCmd(t, srv, "team", "standing", arg, "--event", "2024cthar")
			requireNoError(t, err, "")
			if got := requestPaths(t, srv); len(got) != 1 || got[0] != "/team/frc177/event/2024cthar/status" {
				t.Errorf("requests = %v", got)
			}
		})
	}
}
