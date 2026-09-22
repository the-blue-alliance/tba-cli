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

// 2015 had no win/loss record and its own set of sort orders. A Record column
// that is blank for every team at the event is noise on screen, so the table
// does not print it at all, and the statistic columns are named after that
// season.
func TestEventRankings2015HasNoRecordColumnInTheTable(t *testing.T) {
	srv := newFakeTBA(t, map[string]any{
		"/event/2015ctwat/rankings":     rankings2015ctwatJSON,
		"/event/2015ctwat/teams/simple": teamsSimple2024ctharJSON,
	})
	out, _, err := runCmd(t, srv, "event", "rankings", "2015ctwat", "--format", "table")
	requireNoError(t, err, "")

	if got := lines(out)[0]; !strings.HasPrefix(got, "Rank  Team  Name             Played  DQ  Qual Avg") {
		t.Errorf("header = %q, want no Record column", got)
	}
}

// csv keeps it. A file's header is a schema, and a script that reads Record off
// a column cannot have the columns move because the event it asked about was
// played in a season that had no win/loss record.
func TestEventRankings2015KeepsTheRecordColumnInCSV(t *testing.T) {
	srv := newFakeTBA(t, map[string]any{
		"/event/2015ctwat/rankings":     rankings2015ctwatJSON,
		"/event/2015ctwat/teams/simple": teamsSimple2024ctharJSON,
	})
	out, _, err := runCmd(t, srv, "event", "rankings", "2015ctwat", "--format", "csv")
	requireNoError(t, err, "")

	want := "Rank,Team,Name,Record,Played,DQ,Qual Avg,Auto,Container,Coopertition,Litter,Tote\n" +
		"1,177,Bobcat Robotics,,8,0,78.50,20,0,40,0,18.50\n" +
		"2,1073,The Force Team,,8,1,71.25,16,0,40,0,15.25\n"
	if out != want {
		t.Errorf("csv =\n%s\nwant\n%s", out, want)
	}
}

// A dropped column is gone, not hidden: asking the table for it is the
// ordinary unknown column error, listing what the table does have.
func TestEventRankings2015RejectsTheRecordColumn(t *testing.T) {
	srv := newFakeTBA(t, map[string]any{
		"/event/2015ctwat/rankings":     rankings2015ctwatJSON,
		"/event/2015ctwat/teams/simple": teamsSimple2024ctharJSON,
	})
	_, _, err := runCmd(t, srv, "event", "rankings", "2015ctwat", "--format", "table", "--columns", "record")
	requireErrorContains(t, err, `unknown column "record"`)
	requireErrorContains(t, err, "valid columns: Rank, Team, Name, Played, DQ, Qual Avg")
}

// The same column is selectable in csv, where it was never dropped.
func TestEventRankings2015CSVStillHasTheRecordColumn(t *testing.T) {
	srv := newFakeTBA(t, map[string]any{
		"/event/2015ctwat/rankings":     rankings2015ctwatJSON,
		"/event/2015ctwat/teams/simple": teamsSimple2024ctharJSON,
	})
	out, stderr, err := runCmd(t, srv, "event", "rankings", "2015ctwat",
		"--format", "csv", "--columns", "team,record", "--no-headers")
	requireNoError(t, err, stderr)
	if out != "177,\n1073,\n" {
		t.Errorf("csv = %q", out)
	}
}

// A column with something in it for one team and not another stays: a blank
// there is data, not an empty column.
func TestEventRankingsKeepAPartlyFilledColumn(t *testing.T) {
	srv := newFakeTBA(t, map[string]any{
		"/event/2024cthar/rankings":     rankings2024ctharJSON,
		"/event/2024cthar/teams/simple": teamsSimple2024partialJSON,
	})
	out, _, err := runCmd(t, srv, "event", "rankings", "2024cthar",
		"--format", "csv", "--columns", "team,name", "--no-headers")
	requireNoError(t, err, "")
	if out != "177,Bobcat Robotics\n1073,\n5507,\n" {
		t.Errorf("csv = %q", out)
	}
}

// The nickname lookup is a second request, and a nicety: losing it costs the
// Name column, not the ranking table. With no name for anybody, the column
// itself goes rather than a stripe of blanks down the table.
func TestEventRankingsSurviveAMissingTeamList(t *testing.T) {
	srv := newFakeTBA(t, map[string]any{
		"/event/2024cthar/rankings": rankings2024ctharJSON,
	})
	out, stderr, err := runCmd(t, srv, "event", "rankings", "2024cthar", "--format", "table")
	requireNoError(t, err, stderr)

	got := lines(out)
	if !strings.HasPrefix(got[0], "Rank  Team  Record  Played  DQ  Ranking Score") {
		t.Errorf("header = %q, want no Name column", got[0])
	}
	if !strings.HasPrefix(got[2], "1     177   10-2-0  12      0   2.50") {
		t.Errorf("row 1 = %q", got[2])
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

// Records are three numbers. Sorted as text, 10-2-0 would come first because
// "1" beats "7" and "9".
func TestEventRankingsSortByRecord(t *testing.T) {
	srv := newFakeTBA(t, map[string]any{
		"/event/2024cthar/rankings":     rankings2024ctharJSON,
		"/event/2024cthar/teams/simple": teamsSimple2024ctharJSON,
	})
	out, _, err := runCmd(t, srv, "event", "rankings", "2024cthar",
		"--format", "csv", "--sort", "record", "--columns", "team,record", "--no-headers")
	requireNoError(t, err, "")

	want := "5507,7-5-0\n1073,9-3-0\n177,10-2-0\n"
	if out != want {
		t.Errorf("csv =\n%s\nwant\n%s", out, want)
	}
}
