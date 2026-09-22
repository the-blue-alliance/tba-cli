package cmd

import (
	"strings"
	"testing"

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

// A team that is not attending the event has no status there.
func TestTeamStandingForATeamNotAtTheEvent(t *testing.T) {
	srv := newFakeTBA(t, map[string]any{})
	_, _, err := runCmd(t, srv, "team", "standing", "9999", "--event", "2024cthar")
	if got := clierr.ExitCode(err); got != clierr.ExitNotFound {
		t.Errorf("exit code = %d, want %d (not found)", got, clierr.ExitNotFound)
	}
}

func TestTeamStandingRequiresAnEvent(t *testing.T) {
	srv := newFakeTBA(t, map[string]any{})
	_, _, err := runCmd(t, srv, "team", "standing", "177")
	if err == nil {
		t.Fatal("want an error without --event")
	}
	if got := clierr.ExitCode(err); got != clierr.ExitUsage {
		t.Errorf("exit code = %d, want %d (usage)", got, clierr.ExitUsage)
	}
	if got := requestPaths(t, srv); len(got) != 0 {
		t.Errorf("a missing flag should not reach the API, got %v", got)
	}
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
