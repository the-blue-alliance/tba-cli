package cmd

import (
	"strings"
	"testing"
)

func TestMatchViewTable(t *testing.T) {
	srv := newFakeTBA(t, map[string]any{"/match/2024cthar_qm1": match2024ctharQM1JSON})
	out, _, err := runCmd(t, srv, "match", "view", "2024cthar_qm1", "--format", "table")
	requireNoError(t, err, "")

	want := "Match:          2024cthar_qm1\n" +
		"Level:          qm\n" +
		"Red Alliance:   177, 1073, 5507 (88)\n" +
		"Blue Alliance:  230, 1071, 4055 (61)\n" +
		"Winner:         red\n"
	if out != want {
		t.Errorf("match view table =\n%q\nwant\n%q", out, want)
	}
}

func TestMatchViewJSONRoundTrips(t *testing.T) {
	srv := newFakeTBA(t, map[string]any{"/match/2024cthar_qm1": match2024ctharQM1JSON})
	out, _, err := runCmd(t, srv, "match", "view", "2024cthar_qm1", "--json")
	requireNoError(t, err, "")

	obj := decodeJSON(t, out).(map[string]any)
	if obj["key"] != "2024cthar_qm1" {
		t.Errorf("key = %v", obj["key"])
	}
	if obj["comp_level"] != "qm" {
		t.Errorf("comp_level = %v", obj["comp_level"])
	}
	if obj["winning_alliance"] != "red" {
		t.Errorf("winning_alliance = %v", obj["winning_alliance"])
	}
	if obj["event_key"] != "2024cthar" {
		t.Errorf("event_key = %v", obj["event_key"])
	}
}

func TestMatchViewDefaultsToJSON(t *testing.T) {
	srv := newFakeTBA(t, map[string]any{"/match/2024cthar_qm1": match2024ctharQM1JSON})
	out, _, err := runCmd(t, srv, "match", "view", "2024cthar_qm1")
	requireNoError(t, err, "")
	if !strings.HasPrefix(out, "{") {
		t.Errorf("want JSON by default off a TTY, got:\n%s", out)
	}
}

// A match is a single object, so the tabular formats fall back to JSON.
func TestMatchViewTabularFormatsFallBackToJSON(t *testing.T) {
	srv := newFakeTBA(t, map[string]any{"/match/2024cthar_qm1": match2024ctharQM1JSON})
	for _, format := range []string{"csv", "tsv", "markdown"} {
		t.Run(format, func(t *testing.T) {
			out, _, err := runCmd(t, srv, "match", "view", "2024cthar_qm1", "--format", format)
			requireNoError(t, err, "")
			decodeJSON(t, out)
		})
	}
}

func TestMatchViewJq(t *testing.T) {
	srv := newFakeTBA(t, map[string]any{"/match/2024cthar_qm1": match2024ctharQM1JSON})
	out, _, err := runCmd(t, srv, "match", "view", "2024cthar_qm1", "--jq", ".alliances.red.team_keys[]")
	requireNoError(t, err, "")
	if out != "\"frc177\"\n\"frc1073\"\n\"frc5507\"\n" {
		t.Errorf("jq output = %q", out)
	}
}

func TestMatchViewRequiresAKey(t *testing.T) {
	srv := newFakeTBA(t, map[string]any{})
	if _, _, err := runCmd(t, srv, "match", "view"); err == nil {
		t.Error("want an error with no match key")
	}
}
