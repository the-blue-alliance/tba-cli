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

	want := "Team,Rank,Record,Alliance,Pick,Playoff Level,Playoff Status,Overall\n" +
		`177,1,10-2-0,Alliance 1,Captain,F,won,"Team 177 was Rank 1 with a record of 10-2-0 in quals, competed in the playoffs as the Captain of Alliance 1, and won the event."` + "\n" +
		`1073,2,9-3-0,Alliance 1,1,SF,eliminated,"Team 1073 was Rank 2 with a record of 9-3-0 in quals, and was eliminated in the playoffs."` + "\n" +
		"5507,30,4-8-0,,,,,Team 5507 was Rank 30 with a record of 4-8-0 in quals.\n" +
		"2168,,,Alliance 1,Backup,F,won,Team 2168 competed in the playoffs as a Backup on Alliance 1.\n" +
		"9999,,,,,,,\n"
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
	out, _, err := runCmd(t, srv, "event", "team-statuses", "2024cthar", "--format", "tsv")
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
