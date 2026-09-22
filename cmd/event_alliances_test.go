package cmd

import "testing"

func TestEventAlliancesDoubleElim(t *testing.T) {
	srv := newFakeTBA(t, map[string]any{"/event/2024cthar/alliances": alliances2024ctharJSON})
	out, _, err := runCmd(t, srv, "event", "alliances", "2024cthar", "--format", "csv")
	requireNoError(t, err, "")

	want := "Alliance,Captain,Pick 1,Pick 2,Backup,Status,Level,Record,Declines\n" +
		"Alliance 1,177,1073,5507,2168 in for 5507,won,F,6-1-0,\n" +
		"Alliance 2,230,195,1071,,eliminated,SF,4-2-0,558\n"
	if out != want {
		t.Errorf("csv =\n%s\nwant\n%s", out, want)
	}
}

// double_elim_round is not a table column, but it must survive into JSON.
func TestEventAlliancesJSONKeepsTheDoubleElimRound(t *testing.T) {
	srv := newFakeTBA(t, map[string]any{"/event/2024cthar/alliances": alliances2024ctharJSON})
	out, _, err := runCmd(t, srv, "event", "alliances", "2024cthar", "--json")
	requireNoError(t, err, "")

	first := decodeJSON(t, out).([]any)[0].(map[string]any)
	status := first["status"].(map[string]any)
	if status["double_elim_round"] != "Finals" {
		t.Errorf("double_elim_round = %v", status["double_elim_round"])
	}
	backup := first["backup"].(map[string]any)
	if backup["in"] != "frc2168" || backup["out"] != "frc5507" {
		t.Errorf("backup = %v", backup)
	}
}

// A pre-2023 bracket ranks through qf/sf/f, and an alliance that never played
// has no status at all.
func TestEventAlliances2019Bracket(t *testing.T) {
	srv := newFakeTBA(t, map[string]any{"/event/2019cthar/alliances": alliances2019ctharJSON})
	out, _, err := runCmd(t, srv, "event", "alliances", "2019cthar", "--format", "csv")
	requireNoError(t, err, "")

	want := "Alliance,Captain,Pick 1,Pick 2,Backup,Status,Level,Record,Declines\n" +
		"Alliance 1,177,1073,5507,,won,F,5-2-0,\n" +
		"Alliance 4,230,195,558,1071 in for 558,eliminated,QF,1-2-0,2168\n" +
		"Alliance 8,3467,6153,,,,,,\n"
	if out != want {
		t.Errorf("csv =\n%s\nwant\n%s", out, want)
	}
}

// An unnamed alliance is still numbered by its selection order.
func TestEventAlliancesNameFallsBackToItsNumber(t *testing.T) {
	srv := newFakeTBA(t, map[string]any{"/event/2024cthar/alliances": alliances2024ctharJSON})
	out, _, err := runCmd(t, srv, "event", "alliances", "2024cthar",
		"--format", "csv", "--columns", "alliance", "--no-headers")
	requireNoError(t, err, "")
	if out != "Alliance 1\nAlliance 2\n" {
		t.Errorf("names = %q", out)
	}
}

func TestEventAlliancesMarkdown(t *testing.T) {
	srv := newFakeTBA(t, map[string]any{"/event/2024cthar/alliances": alliances2024ctharJSON})
	out, _, err := runCmd(t, srv, "event", "alliances", "2024cthar",
		"--format", "markdown", "--columns", "alliance,status,level,record")
	requireNoError(t, err, "")

	want := "| Alliance | Status | Level | Record |\n" +
		"| --- | --- | --- | --- |\n" +
		"| Alliance 1 | won | F | 6-1-0 |\n" +
		"| Alliance 2 | eliminated | SF | 4-2-0 |\n"
	if out != want {
		t.Errorf("markdown =\n%s\nwant\n%s", out, want)
	}
}
