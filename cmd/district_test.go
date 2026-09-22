package cmd

import (
	"fmt"
	"testing"
)

func TestDistrictListTable(t *testing.T) {
	srv := newFakeTBA(t, map[string]any{"/districts/2024": districts2024JSON})
	out, _, err := runCmd(t, srv, "district", "list", "--year", "2024", "--format", "table")
	requireNoError(t, err, "")

	// Columns are sized from the header and the widest cell.
	got := lines(out)
	if len(got) != 4 {
		t.Fatalf("want header + separator + 2 rows, got %d:\n%s", len(got), out)
	}
	if got[0] != "Key      Name               Abbreviation" {
		t.Errorf("header = %q", got[0])
	}
	if got[1] != "-------  -----------------  ------------" {
		t.Errorf("separator = %q", got[1])
	}
	if got[2] != "2024ne   New England        ne          " {
		t.Errorf("row 1 = %q", got[2])
	}
	if got[3] != "2024fim  FIRST In Michigan  fim         " {
		t.Errorf("row 2 = %q", got[3])
	}
}

// Regression test: `district list` used to fail without an explicit --year.
func TestDistrictListDefaultsToCurrentYear(t *testing.T) {
	path := fmt.Sprintf("/districts/%d", currentYear())
	srv := newFakeTBA(t, map[string]any{path: "[]"})
	_, _, err := runCmd(t, srv, "district", "list")
	requireNoError(t, err, "")
	// The season lookup comes first; this fake serves no /status, so the year
	// falls back to the calendar.
	if got := requestPaths(t, srv); !contains(got, path) {
		t.Errorf("requested %v, want %s among them", got, path)
	}
}

func TestDistrictListJSONRoundTrips(t *testing.T) {
	srv := newFakeTBA(t, map[string]any{"/districts/2024": districts2024JSON})
	out, _, err := runCmd(t, srv, "district", "list", "--year", "2024", "--json")
	requireNoError(t, err, "")

	arr := decodeJSON(t, out).([]any)
	if len(arr) != 2 {
		t.Fatalf("want 2 districts, got %d", len(arr))
	}
	first := arr[0].(map[string]any)
	if first["key"] != "2024ne" || first["abbreviation"] != "ne" {
		t.Errorf("first district = %v", first)
	}
}

func TestDistrictListTSV(t *testing.T) {
	srv := newFakeTBA(t, map[string]any{"/districts/2024": districts2024JSON})
	out, _, err := runCmd(t, srv, "district", "list", "--year", "2024", "--format", "tsv")
	requireNoError(t, err, "")

	want := "Key\tName\tAbbreviation\n2024ne\tNew England\tne\n2024fim\tFIRST In Michigan\tfim\n"
	if out != want {
		t.Errorf("tsv =\n%q\nwant\n%q", out, want)
	}
}

func TestDistrictEvents(t *testing.T) {
	srv := newFakeTBA(t, map[string]any{
		"/district/2024ne/events": "[" + event2024ctharJSON + "," + event2024necmpJSON + "]",
	})
	out, _, err := runCmd(t, srv, "district", "events", "2024ne", "--format", "csv")
	requireNoError(t, err, "")

	want := "Key,Name,Start Date\n" +
		"2024cthar,NE District Hartford Event,2024-03-22\n" +
		"2024necmp,New England FIRST District Championship,2024-04-10\n"
	if out != want {
		t.Errorf("csv =\n%q\nwant\n%q", out, want)
	}
}

func TestDistrictTeams(t *testing.T) {
	srv := newFakeTBA(t, map[string]any{
		"/district/2024ne/teams": "[" + teamFRC177JSON + "," + teamFRC5507JSON + "]",
	})
	out, _, err := runCmd(t, srv, "district", "teams", "2024ne", "--format", "table")
	requireNoError(t, err, "")

	got := lines(out)
	requireContains(t, got[0], "Number")
	requireContains(t, got[2], "177")
	requireContains(t, got[2], "South Windsor, Connecticut, USA")
	requireContains(t, got[3], "5507")
	requireContains(t, got[3], "Robotic Eagles")
}

func TestDistrictSubcommandsAreRegistered(t *testing.T) {
	got := subcommandNames(t, "district")
	for _, want := range []string{"list", "events", "teams", "rankings"} {
		if !contains(got, want) {
			t.Errorf("district %s is not registered (have %v)", want, got)
		}
	}
}
