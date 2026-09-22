package cmd

import (
	"strings"
	"testing"
	"time"

	"github.com/the-blue-alliance/tba-cli/internal/frc"
	"github.com/the-blue-alliance/tba-cli/internal/output"
)

// withNow pins the clock, so a countdown or a "2h ago" is the same every run.
func withNow(t *testing.T, at time.Time) {
	t.Helper()
	previous := nowFunc
	nowFunc = func() time.Time { return at }
	t.Cleanup(func() { nowFunc = previous })
}

func TestMatchViewTable(t *testing.T) {
	withNow(t, time.Unix(1711130820+7200, 0))
	srv := newFakeTBA(t, map[string]any{"/match/2024cthar_qm12": matchViewQM12JSON})
	out, _, err := runCmd(t, srv, "match", "view", "2024cthar_qm12", "--format", "table")
	requireNoError(t, err, "")

	want := "Match:       Qual 12\n" +
		"Key:         2024cthar_qm12\n" +
		"Event:       2024cthar\n" +
		"Status:      Played\n" +
		"Time:        " + localTime(1711130820) + " (actual, 2h ago)\n" +
		"Red:         R1 177, R2 1073, R3 5507\n" +
		"Red Score:   88\n" +
		"Blue:        B1 230, B2 1071, B3 4055*\n" +
		"Blue Score:  61\n" +
		"Winner:      red\n" +
		"\n" + frc.Legend + "\n" +
		"\nScore breakdown\n" +
		"Stat          Red  Blue\n" +
		"------------  ---  ----\n" +
		"Total Points  88   61  \n" +
		"Auto Points   20   10  \n" +
		"Melody        yes  no  \n" +
		"\nVideos\n" +
		"  https://www.youtube.com/watch?v=dQw4w9WgXcQ\n"
	if out != want {
		t.Errorf("match view table =\n%s\nwant\n%s", out, want)
	}
}

// A qualification match is named the same way whatever bracket the event ran,
// so looking one up costs a single request.
func TestMatchViewFetchesOnlyTheMatch(t *testing.T) {
	srv := newFakeTBA(t, map[string]any{"/match/2024cthar_qm12": matchViewQM12JSON})
	_, _, err := runCmd(t, srv, "match", "view", "2024cthar_qm12")
	requireNoError(t, err, "")
	if got := requestPaths(t, srv); len(got) != 1 || got[0] != "/match/2024cthar_qm12" {
		t.Errorf("requests = %v, want just the match", got)
	}
}

// A semifinal is the one match whose name depends on the bracket, so it is
// worth asking the event.
func TestMatchViewFetchesTheEventForASemifinal(t *testing.T) {
	srv := newFakeTBA(t, map[string]any{
		"/match/2024cthar_sf13m1": matchViewSF13JSON,
		"/event/2024cthar":        event2024ctharJSON,
	})
	out, _, err := runCmd(t, srv, "match", "view", "2024cthar_sf13m1", "--format", "table")
	requireNoError(t, err, "")

	want := []string{"/match/2024cthar_sf13m1", "/event/2024cthar"}
	if got := requestPaths(t, srv); !equalStrings(got, want) {
		t.Errorf("requests = %v, want %v", got, want)
	}
	requireContains(t, out, "Match:       SF 13")
}

// Without the event, the label still comes out, guessed from the season.
func TestMatchViewLabelsASemifinalWithoutTheEvent(t *testing.T) {
	srv := newFakeTBA(t, map[string]any{"/match/2024cthar_sf13m1": matchViewSF13JSON})
	out, _, err := runCmd(t, srv, "match", "view", "2024cthar_sf13m1", "--format", "table")
	requireNoError(t, err, "")
	requireContains(t, out, "Match:       SF 13")
}

// An unplayed match has no score to show and no result to relate.
func TestMatchViewOfAnUnplayedMatch(t *testing.T) {
	withNow(t, time.Unix(1711221000-1080, 0))
	srv := newFakeTBA(t, map[string]any{"/match/2024cthar_qm40": matchViewUnplayedJSON})
	out, _, err := runCmd(t, srv, "match", "view", "2024cthar_qm40", "--format", "table")
	requireNoError(t, err, "")

	want := "Match:       Qual 40\n" +
		"Key:         2024cthar_qm40\n" +
		"Event:       2024cthar\n" +
		"Status:      Scheduled\n" +
		"Time:        " + localTime(1711221000) + " (predicted, in 18m)\n" +
		"Red:         R1 558, R2 3467, R3 2168\n" +
		"Red Score:   \n" +
		"Blue:        B1 195, B2 1124, B3 6153\n" +
		"Blue Score:  \n" +
		"Winner:      \n"
	if out != want {
		t.Errorf("match view =\n%s\nwant\n%s", out, want)
	}
}

// A match with no score breakdown and no videos simply has no such sections,
// which is every match before 2015.
func TestMatchViewWithoutBreakdownOrVideos(t *testing.T) {
	withNow(t, time.Unix(1427464800, 0))
	srv := newFakeTBA(t, map[string]any{"/match/2015ctwat_qm7": match2015ctwatQM7JSON})
	out, _, err := runCmd(t, srv, "match", "view", "2015ctwat_qm7", "--format", "table")
	requireNoError(t, err, "")

	if strings.Contains(out, "Score breakdown") {
		t.Errorf("a match with no breakdown should have no breakdown section:\n%s", out)
	}
	if strings.Contains(out, "Videos") {
		t.Errorf("a match with no videos should have no video section:\n%s", out)
	}
	requireContains(t, out, "Winner:      tie")
	requireContains(t, out, "Time:        "+localTime(1427464800)+" (scheduled, in 0s)")
}

// A video TBA carries that is not on YouTube links to the match's own page,
// which lists it.
func TestMatchViewLinksNonYouTubeVideosToTBA(t *testing.T) {
	srv := newFakeTBA(t, map[string]any{"/match/2024cthar_qm41": matchViewTBAVideoJSON})
	out, _, err := runCmd(t, srv, "match", "view", "2024cthar_qm41", "--format", "table")
	requireNoError(t, err, "")
	requireContains(t, out, "https://www.thebluealliance.com/match/2024cthar_qm41")
}

func TestMatchViewColorsTheAllianceLabels(t *testing.T) {
	withNow(t, time.Unix(1711130820, 0))
	srv := newFakeTBA(t, map[string]any{"/match/2024cthar_qm12": matchViewQM12JSON})
	out, _, err := runCmd(t, srv, "match", "view", "2024cthar_qm12", "--format", "table", "--color", "always")
	requireNoError(t, err, "")

	for _, want := range []string{"\x1b[31mRed\x1b[0m:", "\x1b[34mBlue\x1b[0m:", "\x1b[31mred\x1b[0m"} {
		if !strings.Contains(out, want) {
			t.Errorf("output is missing %q:\n%q", want, out)
		}
	}
}

// Color must not push the values out of line.
func TestMatchViewColorDoesNotChangeAlignment(t *testing.T) {
	withNow(t, time.Unix(1711130820, 0))
	srv := newFakeTBA(t, map[string]any{"/match/2024cthar_qm12": matchViewQM12JSON})
	plain, _, err := runCmd(t, srv, "match", "view", "2024cthar_qm12", "--format", "table", "--color", "never")
	requireNoError(t, err, "")
	colored, _, err := runCmd(t, srv, "match", "view", "2024cthar_qm12", "--format", "table", "--color", "always")
	requireNoError(t, err, "")
	if got := output.StripANSI(colored); got != plain {
		t.Errorf("colored view draws as\n%s\nwant\n%s", got, plain)
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

// The breakdown is one table, an alliance to a column, in the order a
// breakdown is read: the total, the ranking points, the scoring columns, the
// penalties, then the detail.
func TestMatchViewBreakdownIsSideBySide(t *testing.T) {
	withNow(t, time.Unix(1711136640, 0))
	srv := newFakeTBA(t, map[string]any{"/match/2024cthar_qm18": matchViewBreakdown2024JSON})
	out, errOut, err := runCmd(t, srv, "match", "view", "2024cthar_qm18", "--format", "table")
	requireNoError(t, err, errOut)

	body := out[strings.Index(out, "Score breakdown"):]
	requireContains(t, body, "Stat")
	for label, want := range map[string][]string{
		"Total Points":        {"88", "61"},
		"RP":                  {"5", "1"},
		"Foul Count":          {"1", "0"},
		"Auto Amp Note Count": {"1", "0"},
	} {
		row := breakdownRow(t, body, label)
		if row[1] != want[0] || row[2] != want[1] {
			t.Errorf("%s = %v, want %v", label, row[1:], want)
		}
	}

	total := strings.Index(body, "Total Points")
	rp := strings.Index(body, "RP ")
	teleop := strings.Index(body, "Teleop Points")
	fouls := strings.Index(body, "Foul Count")
	detail := strings.Index(body, "Auto Amp Note Count")
	if !(total < rp && rp < teleop && teleop < fouls && fouls < detail) {
		t.Errorf("breakdown rows are out of order:\n%s", body)
	}
}

// breakdownRow finds a row of the side-by-side breakdown table and returns its
// three cells.
func breakdownRow(t *testing.T, body, label string) []string {
	t.Helper()
	for _, line := range lines(body) {
		if !strings.HasPrefix(line, label+" ") {
			continue
		}
		fields := strings.Fields(strings.TrimPrefix(line, label))
		if len(fields) < 2 {
			t.Fatalf("row %q has no values: %q", label, line)
		}
		return []string{label, fields[0], fields[1]}
	}
	t.Fatalf("no %q row in\n%s", label, body)
	return nil
}

// Most of a 2024 breakdown is zero on both sides. Those rows are dropped, and
// --full brings them back.
func TestMatchViewBreakdownDropsEmptyRowsUnlessFull(t *testing.T) {
	withNow(t, time.Unix(1711136640, 0))
	srv := newFakeTBA(t, map[string]any{"/match/2024cthar_qm18": matchViewBreakdown2024JSON})

	out, errOut, err := runCmd(t, srv, "match", "view", "2024cthar_qm18", "--format", "table")
	requireNoError(t, err, errOut)
	for _, gone := range []string{"Trap Center Stage", "Adjust Points", "Tech Foul Count", "G424 Penalty"} {
		if strings.Contains(out, gone) {
			t.Errorf("%q is zero on both sides and should have been dropped:\n%s", gone, out)
		}
	}

	full, errOut, err := runCmd(t, srv, "match", "view", "2024cthar_qm18", "--format", "table", "--full")
	requireNoError(t, err, errOut)
	for _, want := range []string{"Trap Center Stage", "Adjust Points", "Tech Foul Count", "G424 Penalty"} {
		requireContains(t, full, want)
	}
}

func TestMatchViewBreakdownColorsTheAllianceColumns(t *testing.T) {
	withNow(t, time.Unix(1711136640, 0))
	srv := newFakeTBA(t, map[string]any{"/match/2024cthar_qm18": matchViewBreakdown2024JSON})
	out, errOut, err := runCmd(t, srv, "match", "view", "2024cthar_qm18",
		"--format", "table", "--color", "always")
	requireNoError(t, err, errOut)

	for _, want := range []string{"\x1b[31mRed\x1b[0m", "\x1b[34mBlue\x1b[0m"} {
		if !strings.Contains(out, want) {
			t.Errorf("output is missing %q:\n%q", want, out)
		}
	}
}

// The JSON form is the API's own answer; the table is the only thing that
// reorders or hides anything.
func TestMatchViewBreakdownJSONIsUntouched(t *testing.T) {
	srv := newFakeTBA(t, map[string]any{"/match/2024cthar_qm18": matchViewBreakdown2024JSON})
	out, errOut, err := runCmd(t, srv, "match", "view", "2024cthar_qm18", "--json", "--full")
	requireNoError(t, err, errOut)

	breakdown := decodeJSON(t, out).(map[string]any)["score_breakdown"].(map[string]any)
	red := breakdown["red"].(map[string]any)
	if red["trapCenterStage"] != false {
		t.Errorf("a dropped table row must still be in the JSON: %v", red["trapCenterStage"])
	}
	if red["autoAmpNoteCount"] != float64(1) {
		t.Errorf("autoAmpNoteCount = %v", red["autoAmpNoteCount"])
	}
}
