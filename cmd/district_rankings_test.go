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
		"/district/2024ne/rankings": districtRankings2024neJSON,
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

func TestDistrictRankingsCutoffInMarkdown(t *testing.T) {
	srv := newFakeTBA(t, map[string]any{
		"/district/2024ne/rankings": districtRankings2024neJSON,
	})
	out, _, err := runCmd(t, srv, "district", "rankings", "2024ne",
		"--cutoff", "1", "--format", "markdown", "--columns", "rank,team")
	requireNoError(t, err, "")

	want := "| Rank | Team |\n" +
		"| --- | --- |\n" +
		"| 1 | 177 |\n" +
		"| --- DCMP cutoff (top 1) --- |  |\n" +
		"| 2 | 1073 |\n" +
		"| 3 | 5507 |\n"
	if out != want {
		t.Errorf("markdown =\n%s\nwant\n%s", out, want)
	}
}

// The cut line is presentation, so the machine-readable formats never see it.
func TestDistrictRankingsCutoffIsTableOnly(t *testing.T) {
	for _, format := range []string{"csv", "tsv", "json"} {
		t.Run(format, func(t *testing.T) {
			srv := newFakeTBA(t, map[string]any{
				"/district/2024ne/rankings": districtRankings2024neJSON,
			})
			out, _, err := runCmd(t, srv, "district", "rankings", "2024ne",
				"--cutoff", "2", "--format", format)
			requireNoError(t, err, "")
			if strings.Contains(out, "DCMP cutoff") {
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
