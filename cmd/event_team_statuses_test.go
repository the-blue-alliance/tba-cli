package cmd

import (
	"strings"
	"testing"
)

func TestEventTeamStatuses(t *testing.T) {
	srv := newFakeTBA(t, map[string]any{
		"/event/2024cthar/teams/statuses": teamStatuses2024ctharJSON,
	})
	out, _, err := runCmd(t, srv, "event", "team-statuses", "2024cthar", "--format", "csv")
	requireNoError(t, err, "")

	want := "Team,Rank,Record,Alliance,Pick,Playoff Level,Round,Playoff Status\n" +
		"177,1,10-2-0,Alliance 1,Captain,F,Finals,won\n" +
		"1073,2,9-3-0,Alliance 1,1,SF,Round 4,eliminated\n" +
		"5507,30,4-8-0,,,,,\n" +
		"2168,,,Alliance 1,Backup,F,Finals,won\n" +
		"9999,,,,,,,\n"
	if out != want {
		t.Errorf("csv =\n%s\nwant\n%s", out, want)
	}
}

// TBA's summary is a 250-character sentence. It is in the JSON either way, and
// only joins the table when it is asked for.
func TestEventTeamStatusesOverallIsOptIn(t *testing.T) {
	srv := newFakeTBA(t, map[string]any{
		"/event/2024cthar/teams/statuses": teamStatuses2024ctharJSON,
	})
	out, _, err := runCmd(t, srv, "event", "team-statuses", "2024cthar", "--format", "csv")
	requireNoError(t, err, "")
	if strings.Contains(out, "Overall") || strings.Contains(out, "was Rank 1 with a record") {
		t.Errorf("the prose column is on by default:\n%s", out)
	}

	out, _, err = runCmd(t, srv, "event", "team-statuses", "2024cthar", "--overall", "--format", "csv")
	requireNoError(t, err, "")
	want := "Team,Rank,Record,Alliance,Pick,Playoff Level,Round,Playoff Status,Overall\n" +
		"177,1,10-2-0,Alliance 1,Captain,F,Finals,won," +
		`"Team 177 was Rank 1 with a record of 10-2-0 in quals, competed in the playoffs as the Captain of Alliance 1, and won the event."` + "\n"
	if !strings.HasPrefix(out, want) {
		t.Errorf("csv =\n%s\nwant it to start\n%s", out, want)
	}
}

// A pre-2023 bracket has no double_elim_round, so the column is empty rather
// than inventing a round.
func TestEventTeamStatusesRoundIsBlankWithoutADoubleElimBracket(t *testing.T) {
	srv := newFakeTBA(t, map[string]any{
		"/event/2019cthar/teams/statuses": teamStatuses2019ctharJSON,
	})
	out, _, err := runCmd(t, srv, "event", "team-statuses", "2019cthar", "--format", "csv")
	requireNoError(t, err, "")

	want := "Team,Rank,Record,Alliance,Pick,Playoff Level,Round,Playoff Status\n" +
		"177,1,10-2-0,Alliance 1,Captain,F,,won\n"
	if out != want {
		t.Errorf("csv =\n%s\nwant\n%s", out, want)
	}
}

// Ranked teams come first in rank order; everyone else follows by team number.
func TestEventTeamStatusesPutUnrankedTeamsLast(t *testing.T) {
	srv := newFakeTBA(t, map[string]any{
		"/event/2024cthar/teams/statuses": teamStatuses2024ctharJSON,
	})
	out, _, err := runCmd(t, srv, "event", "team-statuses", "2024cthar",
		"--format", "csv", "--columns", "team", "--no-headers")
	requireNoError(t, err, "")
	if out != "177\n1073\n5507\n2168\n9999\n" {
		t.Errorf("teams = %q", out)
	}
}

// overall_status_str is marked-up prose spread over more than one line.
func TestEventTeamStatusesStripTheStatusMarkup(t *testing.T) {
	srv := newFakeTBA(t, map[string]any{
		"/event/2024cthar/teams/statuses": teamStatuses2024ctharJSON,
	})
	out, _, err := runCmd(t, srv, "event", "team-statuses", "2024cthar", "--overall", "--format", "tsv")
	requireNoError(t, err, "")

	if strings.Contains(out, "<b>") || strings.Contains(out, "</b>") {
		t.Errorf("markup leaked into the table:\n%s", out)
	}
	requireContains(t, out, "Team 177 was Rank 1 with a record of 10-2-0 in quals, competed")
}

// A team can be in the feed with a null status of its own.
func TestEventTeamStatusesHandleANullStatus(t *testing.T) {
	srv := newFakeTBA(t, map[string]any{
		"/event/2024cthar/teams/statuses": teamStatuses2024ctharJSON,
	})
	out, _, err := runCmd(t, srv, "event", "team-statuses", "2024cthar", "--format", "table")
	requireNoError(t, err, "")

	got := lines(out)
	last := strings.TrimRight(got[len(got)-1], " ")
	if last != "9999" {
		t.Errorf("last row = %q, want just the team number", last)
	}
}

func TestEventTeamStatusesJSONKeepsTheAPIShape(t *testing.T) {
	srv := newFakeTBA(t, map[string]any{
		"/event/2024cthar/teams/statuses": teamStatuses2024ctharJSON,
	})
	out, _, err := runCmd(t, srv, "event", "team-statuses", "2024cthar", "--json")
	requireNoError(t, err, "")

	obj := decodeJSON(t, out).(map[string]any)
	if obj["frc9999"] != nil {
		t.Errorf("frc9999 = %v, want null", obj["frc9999"])
	}
	qual := obj["frc177"].(map[string]any)["qual"].(map[string]any)
	if rank := qual["ranking"].(map[string]any)["rank"]; rank != float64(1) {
		t.Errorf("rank = %v", rank)
	}
}

func TestEventTeamStatusesRequestsTheRightPath(t *testing.T) {
	srv := newFakeTBA(t, map[string]any{
		"/event/2024cthar/teams/statuses": teamStatuses2024ctharJSON,
	})
	_, _, err := runCmd(t, srv, "event", "team-statuses", "2024cthar")
	requireNoError(t, err, "")
	if got := requestPaths(t, srv); len(got) != 1 || got[0] != "/event/2024cthar/teams/statuses" {
		t.Errorf("requested %v", got)
	}
}
