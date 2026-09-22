package cmd

import (
	"testing"

	"github.com/the-blue-alliance/tba-cli/internal/clierr"
)

func TestEventDistrictPointsTable(t *testing.T) {
	srv := newFakeTBA(t, map[string]any{
		"/event/2024cthar/district_points": districtPoints2024ctharJSON,
	})
	out, _, err := runCmd(t, srv, "event", "district-points", "2024cthar", "--format", "csv")
	requireNoError(t, err, "")

	// Highest total first, with the 64-point tie broken by team number (230 < 1073).
	want := "Team,Qual,Alliance,Award,Elim,Total\n" +
		"177,22,16,5,30,73\n" +
		"230,18,16,0,30,64\n" +
		"1073,20,14,0,30,64\n" +
		"5507,12,0,0,0,12\n"
	if out != want {
		t.Errorf("csv =\n%s\nwant\n%s", out, want)
	}
}

func TestEventDistrictPointsTiebreakers(t *testing.T) {
	srv := newFakeTBA(t, map[string]any{
		"/event/2024cthar/district_points": districtPoints2024ctharJSON,
	})
	out, _, err := runCmd(t, srv, "event", "district-points", "2024cthar",
		"--tiebreakers", "--format", "csv")
	requireNoError(t, err, "")

	want := "Team,Qual,Alliance,Award,Elim,Total,Highest Qual Scores,Qual Wins\n" +
		`177,22,16,5,30,73,"88, 80, 76",10` + "\n" +
		`230,18,16,0,30,64,"80, 77, 70",8` + "\n" +
		`1073,20,14,0,30,64,"84, 78, 72",9` + "\n" +
		`5507,12,0,0,0,12,"55, 51, 48",4` + "\n"
	if out != want {
		t.Errorf("csv =\n%s\nwant\n%s", out, want)
	}
}

// The tiebreaker columns stay out of the way unless they are asked for.
func TestEventDistrictPointsTiebreakersAreOptIn(t *testing.T) {
	srv := newFakeTBA(t, map[string]any{
		"/event/2024cthar/district_points": districtPoints2024ctharJSON,
	})
	out, _, err := runCmd(t, srv, "event", "district-points", "2024cthar", "--format", "csv")
	requireNoError(t, err, "")
	if got := lines(out)[0]; got != "Team,Qual,Alliance,Award,Elim,Total" {
		t.Errorf("header = %q", got)
	}
}

// JSON keeps the shape the API returns, so existing --jq expressions still work.
func TestEventDistrictPointsJSONKeepsTheAPIShape(t *testing.T) {
	srv := newFakeTBA(t, map[string]any{
		"/event/2024cthar/district_points": districtPoints2024ctharJSON,
	})
	out, _, err := runCmd(t, srv, "event", "district-points", "2024cthar", "--json")
	requireNoError(t, err, "")

	obj := decodeJSON(t, out).(map[string]any)
	points := obj["points"].(map[string]any)["frc177"].(map[string]any)
	if points["total"] != float64(73) {
		t.Errorf("total = %v", points["total"])
	}
	tiebreakers := obj["tiebreakers"].(map[string]any)["frc177"].(map[string]any)
	if tiebreakers["qual_wins"] != float64(10) {
		t.Errorf("qual_wins = %v", tiebreakers["qual_wins"])
	}
}

func TestEventDistrictPointsJq(t *testing.T) {
	srv := newFakeTBA(t, map[string]any{
		"/event/2024cthar/district_points": districtPoints2024ctharJSON,
	})
	out, _, err := runCmd(t, srv, "event", "district-points", "2024cthar",
		"--jq", ".points.frc177.total")
	requireNoError(t, err, "")
	if out != "73\n" {
		t.Errorf("jq output = %q", out)
	}
}

// --sort reorders the table. The JSON is an object keyed by team rather than an
// array, so there is no row order there to permute; it is left alone.
func TestEventDistrictPointsSortReordersTheTable(t *testing.T) {
	srv := newFakeTBA(t, map[string]any{
		"/event/2024cthar/district_points": districtPoints2024ctharJSON,
	})
	out, _, err := runCmd(t, srv, "event", "district-points", "2024cthar",
		"--format", "csv", "--sort", "team", "--columns", "team", "--no-headers")
	requireNoError(t, err, "")
	if out != "177\n230\n1073\n5507\n" {
		t.Errorf("teams = %q", out)
	}
}

// The JSON here is the API's object keyed by team, not the table's rows, so
// there is no row order to apply to it. Rather than print an unsorted answer to
// a command that asked for a sorted one, --sort says so and exits 2.
func TestEventDistrictPointsSortCannotReorderTheJSONObject(t *testing.T) {
	srv := newFakeTBA(t, map[string]any{
		"/event/2024cthar/district_points": districtPoints2024ctharJSON,
	})
	err := requireExitCode(t, clierr.ExitUsage, srv, "event", "district-points", "2024cthar",
		"--json", "--sort", "total")
	requireErrorContains(t, err, "--sort cannot reorder this JSON payload")
	requireErrorContains(t, err, "--jq")

	// Without --sort the object is printed as the API returned it.
	out, stderr, err := runCmd(t, srv, "event", "district-points", "2024cthar", "--json")
	requireNoError(t, err, stderr)
	obj := decodeJSON(t, out).(map[string]any)
	if _, ok := obj["points"]; !ok {
		t.Errorf("want the API object, got %v", obj)
	}
}
