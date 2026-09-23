package cmd

import (
	"strings"
	"testing"
)

// years_participated comes back unordered in practice, so the fixture is too.
const teamYears177JSON = `[2022, 1997, 2024, 1995, 2023, 1996]`

func TestTeamYearsTable(t *testing.T) {
	srv := newFakeTBA(t, map[string]any{"/team/frc177/years_participated": teamYears177JSON})
	out, stderr, err := runCmd(t, srv, "team", "years", "177", "--format", "csv")
	requireNoError(t, err, stderr)

	want := "Year\n2024\n2023\n2022\n1997\n1996\n1995\n"
	if out != want {
		t.Errorf("csv =\n%q\nwant\n%q", out, want)
	}
}

func TestTeamYearsAcceptsFrcPrefix(t *testing.T) {
	srv := newFakeTBA(t, map[string]any{"/team/frc177/years_participated": teamYears177JSON})
	_, stderr, err := runCmd(t, srv, "team", "years", "frc177")
	requireNoError(t, err, stderr)
	if got := requestPaths(t, srv); len(got) != 1 || got[0] != "/team/frc177/years_participated" {
		t.Errorf("requested %v, want [/team/frc177/years_participated]", got)
	}
}

func TestTeamYearsJSONIsAPlainArray(t *testing.T) {
	srv := newFakeTBA(t, map[string]any{"/team/frc177/years_participated": teamYears177JSON})
	out, stderr, err := runCmd(t, srv, "team", "years", "177", "--json")
	requireNoError(t, err, stderr)

	arr, ok := decodeJSON(t, out).([]any)
	if !ok {
		t.Fatalf("output is not a JSON array:\n%s", out)
	}
	if len(arr) != 6 || arr[0] != float64(2024) || arr[5] != float64(1995) {
		t.Errorf("json = %s, want the years newest first", strings.TrimSpace(out))
	}
}

func TestTeamYearsSingleColumnHeader(t *testing.T) {
	srv := newFakeTBA(t, map[string]any{"/team/frc177/years_participated": teamYears177JSON})
	out, stderr, err := runCmd(t, srv, "team", "years", "177", "--format", "table")
	requireNoError(t, err, stderr)
	if got := lines(out)[0]; strings.TrimSpace(got) != "Year" {
		t.Errorf("header = %q, want Year", got)
	}
}

func TestTeamYearsEmptyForATeamThatNeverCompeted(t *testing.T) {
	srv := newFakeTBA(t, map[string]any{"/team/frc9999/years_participated": `[]`})
	out, stderr, err := runCmd(t, srv, "team", "years", "9999", "--json")
	requireNoError(t, err, stderr)
	if strings.TrimSpace(out) != "[]" {
		t.Errorf("json = %q, want []", out)
	}
}

func TestTeamYearsIsRegistered(t *testing.T) {
	if !contains(subcommandNames(t, "team"), "years") {
		t.Error("team years is not registered")
	}
}
