package cmd

import (
	"strings"
	"testing"
)

func TestEventRankingsColumnsFollowTheSeasonsSortOrders(t *testing.T) {
	srv := newFakeTBA(t, map[string]any{
		"/event/2024cthar/rankings":     rankings2024ctharJSON,
		"/event/2024cthar/teams/simple": teamsSimple2024ctharJSON,
	})
	out, _, err := runCmd(t, srv, "event", "rankings", "2024cthar", "--format", "csv")
	requireNoError(t, err, "")

	want := "Rank,Team,Name,Record,Played,DQ,Ranking Score,Avg Coop,Avg Match,Avg Auto,Avg Stage,Total Ranking Points\n" +
		"1,177,Bobcat Robotics,10-2-0,12,0,2.50,0.25,88.00,31.00,18.00,30\n" +
		"2,1073,The Force Team,9-3-0,12,1,2.33,0.00,80.25,28.50,15.00,28\n" +
		"3,5507,Robotic Eagles,7-5-0,12,0,1.92,0.00,66.00,20.00,9.00,23\n"
	if out != want {
		t.Errorf("csv =\n%s\nwant\n%s", out, want)
	}
}

// The fixture lists rank 2 first; the table is ordered by rank regardless.
func TestEventRankingsAreSortedByRank(t *testing.T) {
	srv := newFakeTBA(t, map[string]any{
		"/event/2024cthar/rankings":     rankings2024ctharJSON,
		"/event/2024cthar/teams/simple": teamsSimple2024ctharJSON,
	})
	out, _, err := runCmd(t, srv, "event", "rankings", "2024cthar",
		"--format", "csv", "--columns", "rank", "--no-headers")
	requireNoError(t, err, "")
	if out != "1\n2\n3\n" {
		t.Errorf("ranks = %q", out)
	}
}

func TestEventRankingsTable(t *testing.T) {
	srv := newFakeTBA(t, map[string]any{
		"/event/2024cthar/rankings":     rankings2024ctharJSON,
		"/event/2024cthar/teams/simple": teamsSimple2024ctharJSON,
	})
	out, _, err := runCmd(t, srv, "event", "rankings", "2024cthar", "--format", "table")
	requireNoError(t, err, "")

	got := lines(out)
	if !strings.HasPrefix(got[0], "Rank  Team  Name             Record  Played  DQ  Ranking Score") {
		t.Errorf("header = %q", got[0])
	}
	if !strings.HasPrefix(got[2], "1     177   Bobcat Robotics  10-2-0  12      0   2.50") {
		t.Errorf("row 1 = %q", got[2])
	}
}

// 2015 had no win/loss record and its own set of sort orders, so the Record
// column is blank and the statistic columns are named after that season.
func TestEventRankings2015HasNoRecord(t *testing.T) {
	srv := newFakeTBA(t, map[string]any{
		"/event/2015ctwat/rankings": rankings2015ctwatJSON,
	})
	out, _, err := runCmd(t, srv, "event", "rankings", "2015ctwat", "--format", "csv")
	requireNoError(t, err, "")

	want := "Rank,Team,Name,Record,Played,DQ,Qual Avg,Auto,Container,Coopertition,Litter,Tote\n" +
		"1,177,,,8,0,78.50,20,0,40,0,18.50\n" +
		"2,1073,,,8,1,71.25,16,0,40,0,15.25\n"
	if out != want {
		t.Errorf("csv =\n%s\nwant\n%s", out, want)
	}
}

// The nickname lookup is a second request, and a nicety: losing it costs the
// Name column, not the ranking table.
func TestEventRankingsSurviveAMissingTeamList(t *testing.T) {
	srv := newFakeTBA(t, map[string]any{
		"/event/2024cthar/rankings": rankings2024ctharJSON,
	})
	out, stderr, err := runCmd(t, srv, "event", "rankings", "2024cthar", "--format", "csv")
	requireNoError(t, err, stderr)

	got := lines(out)
	if got[1] != "1,177,,10-2-0,12,0,2.50,0.25,88.00,31.00,18.00,30" {
		t.Errorf("row 1 = %q", got[1])
	}
	if !contains(requestPaths(t, srv), "/event/2024cthar/teams/simple") {
		t.Errorf("the nickname lookup was never attempted: %v", requestPaths(t, srv))
	}
}

// JSON does not carry the Name column, so it must not pay for it either.
func TestEventRankingsJSONSkipsTheNicknameLookup(t *testing.T) {
	srv := newFakeTBA(t, map[string]any{
		"/event/2024cthar/rankings":     rankings2024ctharJSON,
		"/event/2024cthar/teams/simple": teamsSimple2024ctharJSON,
	})
	out, _, err := runCmd(t, srv, "event", "rankings", "2024cthar", "--json")
	requireNoError(t, err, "")

	if got := requestPaths(t, srv); len(got) != 1 || got[0] != "/event/2024cthar/rankings" {
		t.Errorf("requested %v, want only the rankings", got)
	}
	obj := decodeJSON(t, out).(map[string]any)
	for _, key := range []string{"rankings", "sort_order_info", "extra_stats_info"} {
		if _, ok := obj[key]; !ok {
			t.Errorf("want a %s key, got %v", key, obj)
		}
	}
}

// A season's statistic columns are addressable like any other column.
func TestEventRankingsSortOrderColumnsAreSelectable(t *testing.T) {
	srv := newFakeTBA(t, map[string]any{
		"/event/2024cthar/rankings":     rankings2024ctharJSON,
		"/event/2024cthar/teams/simple": teamsSimple2024ctharJSON,
	})
	out, _, err := runCmd(t, srv, "event", "rankings", "2024cthar",
		"--format", "csv", "--columns", "team,ranking score,total ranking points")
	requireNoError(t, err, "")

	want := "Team,Ranking Score,Total Ranking Points\n" +
		"177,2.50,30\n" +
		"1073,2.33,28\n" +
		"5507,1.92,23\n"
	if out != want {
		t.Errorf("csv =\n%s\nwant\n%s", out, want)
	}
}

func TestEventRankingsSortByDQ(t *testing.T) {
	srv := newFakeTBA(t, map[string]any{
		"/event/2024cthar/rankings":     rankings2024ctharJSON,
		"/event/2024cthar/teams/simple": teamsSimple2024ctharJSON,
	})
	out, _, err := runCmd(t, srv, "event", "rankings", "2024cthar",
		"--format", "csv", "--sort", "-dq", "--columns", "team", "--no-headers")
	requireNoError(t, err, "")
	if out != "1073\n177\n5507\n" {
		t.Errorf("teams = %q", out)
	}
}
