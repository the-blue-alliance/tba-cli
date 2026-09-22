package cmd

import (
	"strings"
	"testing"
)

func TestDistrictRankingsColumns(t *testing.T) {
	srv := newFakeTBA(t, map[string]any{
		"/district/2024ne/rankings": districtRankings2024neJSON,
	})
	out, _, err := runCmd(t, srv, "district", "rankings", "2024ne", "--format", "csv")
	requireNoError(t, err, "")

	want := "Rank,Team,Rookie Bonus,Event 1,Event 2,DCMP,Total\n" +
		"1,177,0,52,48,45,145\n" +
		"2,1073,0,40,42,50,132\n" +
		"3,5507,10,34,34,,78\n"
	if out != want {
		t.Errorf("csv =\n%s\nwant\n%s", out, want)
	}
}

func TestDistrictRankingsDetail(t *testing.T) {
	srv := newFakeTBA(t, map[string]any{
		"/district/2024ne/rankings": districtRankings2024neJSON,
	})
	out, _, err := runCmd(t, srv, "district", "rankings", "2024ne", "--detail", "--format", "csv")
	requireNoError(t, err, "")

	want := "Rank,Team,Rookie Bonus," +
		"Event 1,E1 Qual,E1 Alliance,E1 Award,E1 Elim," +
		"Event 2,E2 Qual,E2 Alliance,E2 Award,E2 Elim,DCMP,Total\n" +
		"1,177,0,52,11,16,5,20,48,16,14,0,18,45,145\n" +
		"2,1073,0,40,10,14,0,16,42,12,12,0,18,50,132\n" +
		"3,5507,10,34,8,10,0,16,34,14,0,10,10,,78\n"
	if out != want {
		t.Errorf("csv =\n%s\nwant\n%s", out, want)
	}
}

func TestDistrictRankingsCutoffInTable(t *testing.T) {
	srv := newFakeTBA(t, map[string]any{
		"/district/2024ne/rankings": districtRankingsMidSeason2024neJSON,
	})
	out, _, err := runCmd(t, srv, "district", "rankings", "2024ne", "--cutoff", "2", "--format", "table")
	requireNoError(t, err, "")

	got := lines(out)
	// Header, separator, rank 1, rank 2, cut line, rank 3.
	if len(got) != 6 {
		t.Fatalf("want 6 lines, got %d:\n%s", len(got), out)
	}
	if strings.TrimRight(got[4], " ") != "--- DCMP cutoff (top 2) ---" {
		t.Errorf("cut line = %q", got[4])
	}
	if !strings.HasPrefix(got[5], "3 ") {
		t.Errorf("row after the cut = %q", got[5])
	}
}

// The cut line used to be written into the first cell, which widened the Rank
// column to the length of the sentence. It is a line of its own now, so the
// columns are as wide as the data and nothing else.
func TestDistrictRankingsCutoffLeavesTheRankColumnAlone(t *testing.T) {
	srv := newFakeTBA(t, map[string]any{
		"/district/2024ne/rankings": districtRankingsMidSeason2024neJSON,
	})
	out, _, err := runCmd(t, srv, "district", "rankings", "2024ne",
		"--cutoff", "2", "--format", "table", "--columns", "rank,team")
	requireNoError(t, err, "")

	got := lines(out)
	if got[0] != "Rank  Team" {
		t.Errorf("header = %q, want the columns sized to their data", got[0])
	}
	if strings.TrimRight(got[2], " ") != "1     177" {
		t.Errorf("first row = %q", got[2])
	}
	if strings.TrimRight(got[4], " ") != "--- DCMP cutoff (top 2) ---" {
		t.Errorf("cut line = %q", got[4])
	}
}

func TestDistrictRankingsCutoffInMarkdown(t *testing.T) {
	srv := newFakeTBA(t, map[string]any{
		"/district/2024ne/rankings": districtRankingsMidSeason2024neJSON,
	})
	out, _, err := runCmd(t, srv, "district", "rankings", "2024ne",
		"--cutoff", "1", "--format", "markdown", "--columns", "rank,team")
	requireNoError(t, err, "")

	want := "| Rank | Team |\n" +
		"| --- | --- |\n" +
		"| 1 | 177 |\n" +
		"| --- DCMP cutoff (top 1) --- | |\n" +
		"| 2 | 1073 |\n" +
		"| 3 | 5507 |\n"
	if out != want {
		t.Errorf("markdown =\n%s\nwant\n%s", out, want)
	}
}

// Once the DCMP has been scored the published totals include its points, so a
// line through them is not the cut that decided who went, and says so.
func TestDistrictRankingsCutoffSaysWhenTotalsIncludeDCMP(t *testing.T) {
	srv := newFakeTBA(t, map[string]any{
		"/district/2024ne/rankings": districtRankings2024neJSON,
	})
	out, _, err := runCmd(t, srv, "district", "rankings", "2024ne", "--cutoff", "2", "--format", "table")
	requireNoError(t, err, "")
	requireContains(t, out, "--- top 2 by current total (includes DCMP points) ---")
}

// --pre-dcmp ranks on the points a team had before the district championship,
// which is the standing the cut was actually made on.
func TestDistrictRankingsPreDCMP(t *testing.T) {
	srv := newFakeTBA(t, map[string]any{
		"/district/2024ne/rankings": districtRankingsDCMPFlip2024neJSON,
	})
	out, _, err := runCmd(t, srv, "district", "rankings", "2024ne",
		"--pre-dcmp", "--cutoff", "1", "--format", "table",
		"--columns", "rank,team,dcmp,total,pre-dcmp")
	requireNoError(t, err, "")

	got := lines(out)
	want := []string{
		"Rank  Team  DCMP  Total  Pre-DCMP",
		"----  ----  ----  -----  --------",
		"2     177   45    145    100",
		"--- top 1 by pre-DCMP total ---",
		"1     1073  68    150    82",
		"3     5507        78     78",
	}
	if len(got) != len(want) {
		t.Fatalf("got %d lines, want %d:\n%s", len(got), len(want), out)
	}
	for i := range want {
		if strings.TrimRight(got[i], " ") != want[i] {
			t.Errorf("line %d = %q, want %q", i, strings.TrimRight(got[i], " "), want[i])
		}
	}
}

// The reordering is the answer, not a view of it, so JSON follows it too.
func TestDistrictRankingsPreDCMPReordersJSON(t *testing.T) {
	srv := newFakeTBA(t, map[string]any{
		"/district/2024ne/rankings": districtRankingsDCMPFlip2024neJSON,
	})
	out, _, err := runCmd(t, srv, "district", "rankings", "2024ne", "--pre-dcmp", "--json")
	requireNoError(t, err, "")

	arr := decodeJSON(t, out).([]any)
	want := []string{"frc177", "frc1073", "frc5507"}
	for i, key := range want {
		if got := arr[i].(map[string]any)["team_key"]; got != key {
			t.Errorf("team %d = %v, want %s", i, got, key)
		}
	}
}

// Without --pre-dcmp there is no extra column to explain.
func TestDistrictRankingsPreDCMPColumnIsOptIn(t *testing.T) {
	srv := newFakeTBA(t, map[string]any{
		"/district/2024ne/rankings": districtRankings2024neJSON,
	})
	out, _, err := runCmd(t, srv, "district", "rankings", "2024ne", "--format", "csv")
	requireNoError(t, err, "")
	if strings.Contains(out, "Pre-DCMP") {
		t.Errorf("Pre-DCMP column appeared unasked:\n%s", out)
	}
}

// The cut line is presentation, so the machine-readable formats never see it.
func TestDistrictRankingsCutoffIsTableOnly(t *testing.T) {
	for _, format := range []string{"csv", "tsv", "json"} {
		t.Run(format, func(t *testing.T) {
			srv := newFakeTBA(t, map[string]any{
				"/district/2024ne/rankings": districtRankingsMidSeason2024neJSON,
			})
			out, _, err := runCmd(t, srv, "district", "rankings", "2024ne",
				"--cutoff", "2", "--format", format)
			requireNoError(t, err, "")
			if strings.Contains(out, "cutoff") || strings.Contains(out, "---") {
				t.Errorf("%s output carries the cut line:\n%s", format, out)
			}
		})
	}
}

// A cutoff past the last team has nothing to separate.
func TestDistrictRankingsCutoffBeyondTheListDrawsNothing(t *testing.T) {
	srv := newFakeTBA(t, map[string]any{
		"/district/2024ne/rankings": districtRankings2024neJSON,
	})
	out, _, err := runCmd(t, srv, "district", "rankings", "2024ne", "--cutoff", "50", "--format", "table")
	requireNoError(t, err, "")
	if strings.Contains(out, "DCMP cutoff") {
		t.Errorf("output carries a cut line:\n%s", out)
	}
}

// A re-sorted table is no longer in rank order, so the cut line would be a lie.
func TestDistrictRankingsCutoffStandsDownForSort(t *testing.T) {
	srv := newFakeTBA(t, map[string]any{
		"/district/2024ne/rankings": districtRankings2024neJSON,
	})
	out, stderr, err := runCmd(t, srv, "district", "rankings", "2024ne",
		"--cutoff", "2", "--sort", "team", "--format", "table")
	requireNoError(t, err, stderr)
	if strings.Contains(out, "DCMP cutoff") {
		t.Errorf("output carries a cut line:\n%s", out)
	}
	requireContains(t, stderr, "--cutoff needs rank order")
}

func TestDistrictRankingsRejectsANegativeCutoff(t *testing.T) {
	srv := newFakeTBA(t, map[string]any{})
	_, _, err := runCmd(t, srv, "district", "rankings", "2024ne", "--cutoff", "-3")
	requireErrorContains(t, err, "--cutoff must be a positive rank")
	if got := requestPaths(t, srv); len(got) != 0 {
		t.Errorf("a bad flag should not reach the API, got %v", got)
	}
}

func TestDistrictRankingsSortReordersJSONToo(t *testing.T) {
	srv := newFakeTBA(t, map[string]any{
		"/district/2024ne/rankings": districtRankings2024neJSON,
	})
	out, _, err := runCmd(t, srv, "district", "rankings", "2024ne", "--json", "--sort", "team")
	requireNoError(t, err, "")

	arr := decodeJSON(t, out).([]any)
	keys := make([]string, len(arr))
	for i, v := range arr {
		keys[i] = v.(map[string]any)["team_key"].(string)
	}
	want := []string{"frc177", "frc1073", "frc5507"}
	for i := range want {
		if keys[i] != want[i] {
			t.Fatalf("team order = %v, want %v", keys, want)
		}
	}
}

func TestDistrictRankingsJq(t *testing.T) {
	srv := newFakeTBA(t, map[string]any{
		"/district/2024ne/rankings": districtRankings2024neJSON,
	})
	out, _, err := runCmd(t, srv, "district", "rankings", "2024ne", "--jq", ".[0].point_total")
	requireNoError(t, err, "")
	if strings.TrimSpace(out) != "145" {
		t.Errorf("jq output = %q", out)
	}
}
