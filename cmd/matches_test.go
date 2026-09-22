package cmd

import (
	"encoding/csv"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/the-blue-alliance/tba-cli/internal/clierr"
	"github.com/the-blue-alliance/tba-cli/internal/frc"
	"github.com/the-blue-alliance/tba-cli/internal/output"
)

// localTime renders a Unix timestamp the way a single-day match listing does,
// so the expectations below do not depend on the machine's time zone.
func localTime(epoch int64) string {
	return time.Unix(epoch, 0).In(time.Local).Format(frc.TimeLayout)
}

// localDateTime is localTime for a listing that spans more than one day, which
// carries the date as well.
func localDateTime(epoch int64) string {
	return time.Unix(epoch, 0).In(time.Local).Format(frc.DatedTimeLayout)
}

func eventMatchesServer(t *testing.T) *httptest.Server {
	t.Helper()
	return newFakeTBA(t, map[string]any{
		"/event/2024cthar":         event2024ctharJSON,
		"/event/2024cthar/matches": matches2024ctharJSON,
	})
}

// A match listing is ordered the way the event plays: qualification matches in
// number order, then the elimination rounds. The API returns them in no
// particular order.
func TestEventMatchesOrdersByPlayingOrder(t *testing.T) {
	srv := eventMatchesServer(t)
	out, _, err := runCmd(t, srv, "event", "matches", "2024cthar", "--format", "table")
	requireNoError(t, err, "")

	want := []string{"Qual 2", "Qual 3", "Qual 7", "Qual 12", "SF 13", "Final 2"}
	if got := columnValues(t, out, 0); !equalStrings(got, want) {
		t.Errorf("match order = %v, want %v", got, want)
	}
}

func TestEventMatchesColumns(t *testing.T) {
	srv := eventMatchesServer(t)
	out, _, err := runCmd(t, srv, "event", "matches", "2024cthar", "--format", "csv")
	requireNoError(t, err, "")

	records := parseCSV(t, out)
	wantHeader := []string{
		"Match", "Key", "Red", "Blue", "Score (R-B)", "Winner",
		"Time", "When", "Time Source", "Status",
	}
	if !equalStrings(records[0], wantHeader) {
		t.Errorf("header = %v, want %v", records[0], wantHeader)
	}

	// Qual 12: played, red wins, and a surrogate on blue. The event ran from
	// Friday to Sunday, so its times carry the date; a played match has no
	// countdown left to print.
	want := []string{
		"Qual 12", "2024cthar_qm12",
		"177, 1073, 5507", "230, 1071, 4055*",
		"88-61", "red",
		localDateTime(1711130820), "", "actual", "Played",
	}
	if got := findRow(t, records, "2024cthar_qm12"); !equalStrings(got, want) {
		t.Errorf("qm12 row =\n%v\nwant\n%v", got, want)
	}
}

// An unplayed match scores -1/-1; the score column stays blank and the status
// says so. Its time is the queue's prediction, not a result.
func TestEventMatchesLeavesAnUnplayedScoreBlank(t *testing.T) {
	withNow(t, time.Unix(1711122000-1080, 0))
	srv := eventMatchesServer(t)
	out, _, err := runCmd(t, srv, "event", "matches", "2024cthar", "--format", "csv")
	requireNoError(t, err, "")

	want := []string{
		"Qual 2", "2024cthar_qm2",
		"558, 3467, 2168", "195, 1124, 6153",
		"", "",
		localDateTime(1711122000), "in 18m", "predicted", "Scheduled",
	}
	if got := findRow(t, parseCSV(t, out), "2024cthar_qm2"); !equalStrings(got, want) {
		t.Errorf("unplayed row =\n%v\nwant\n%v", got, want)
	}
	for _, score := range csvColumn(t, out, 4) {
		if strings.HasPrefix(score, "-1") || strings.HasSuffix(score, "-1") {
			t.Errorf("the API's -1 placeholder leaked into a score cell: %q", score)
		}
	}
}

// A played match the API reports no winner for is a tie, not a blank.
func TestEventMatchesShowsATie(t *testing.T) {
	srv := eventMatchesServer(t)
	out, _, err := runCmd(t, srv, "event", "matches", "2024cthar", "--format", "csv")
	requireNoError(t, err, "")

	row := findRow(t, parseCSV(t, out), "2024cthar_qm7")
	if row[4] != "44-44" || row[5] != "tie" {
		t.Errorf("tie row = %v", row)
	}
}

func TestEventMatchesMarksSurrogatesAndDisqualifications(t *testing.T) {
	srv := eventMatchesServer(t)
	out, _, err := runCmd(t, srv, "event", "matches", "2024cthar", "--format", "csv")
	requireNoError(t, err, "")

	records := parseCSV(t, out)
	if got := findRow(t, records, "2024cthar_qm12")[3]; got != "230, 1071, 4055*" {
		t.Errorf("surrogate cell = %q", got)
	}
	if got := findRow(t, records, "2024cthar_qm3")[2]; got != "177, 3467, 2168!" {
		t.Errorf("disqualification cell = %q", got)
	}
}

// The legend explains the marks, and belongs on stderr so it never lands in a
// file the table was piped into.
func TestEventMatchesPrintsTheMarkLegendOnStderr(t *testing.T) {
	srv := eventMatchesServer(t)
	out, errOut, err := runCmd(t, srv, "event", "matches", "2024cthar", "--format", "table")
	requireNoError(t, err, errOut)

	if strings.TrimSpace(errOut) != frc.Legend {
		t.Errorf("stderr = %q, want the legend %q", errOut, frc.Legend)
	}
	if strings.Contains(out, "surrogate") {
		t.Errorf("the legend must not be on stdout:\n%s", out)
	}
}

// With nothing to explain, there is no legend.
func TestEventMatchesOmitsTheLegendWithoutMarks(t *testing.T) {
	srv := newFakeTBA(t, map[string]any{
		"/event/2019ctwat":         event2019ctwatJSON,
		"/event/2019ctwat/matches": matches2019ctwatJSON,
	})
	_, errOut, err := runCmd(t, srv, "event", "matches", "2019ctwat", "--format", "table")
	requireNoError(t, err, errOut)
	if errOut != "" {
		t.Errorf("stderr = %q, want nothing", errOut)
	}
}

// The legend is about the aligned table; a machine-readable format gets no
// commentary at all.
func TestEventMatchesOmitsTheLegendOutsideTableMode(t *testing.T) {
	srv := eventMatchesServer(t)
	for _, format := range []string{"csv", "tsv", "markdown", "json"} {
		t.Run(format, func(t *testing.T) {
			_, errOut, err := runCmd(t, srv, "event", "matches", "2024cthar", "--format", format)
			requireNoError(t, err, errOut)
			if errOut != "" {
				t.Errorf("stderr = %q, want nothing", errOut)
			}
		})
	}
}

// A double-elimination semifinal set is a single match, so it is named by its
// set number alone.
func TestEventMatchesLabelsADoubleElimBracket(t *testing.T) {
	srv := eventMatchesServer(t)
	out, _, err := runCmd(t, srv, "event", "matches", "2024cthar", "--format", "csv")
	requireNoError(t, err, "")

	records := parseCSV(t, out)
	if got := findRow(t, records, "2024cthar_sf13m1")[0]; got != "SF 13" {
		t.Errorf("semifinal label = %q, want %q", got, "SF 13")
	}
	if got := findRow(t, records, "2024cthar_f1m2")[0]; got != "Final 2" {
		t.Errorf("final label = %q, want %q", got, "Final 2")
	}
}

// A 2019 bracket ran best-of-three sets, so both numbers are needed.
func TestEventMatchesLabelsALegacyBracket(t *testing.T) {
	srv := newFakeTBA(t, map[string]any{
		"/event/2019ctwat":         event2019ctwatJSON,
		"/event/2019ctwat/matches": matches2019ctwatJSON,
	})
	out, _, err := runCmd(t, srv, "event", "matches", "2019ctwat", "--format", "csv")
	requireNoError(t, err, "")

	want := []string{"QF 1-1", "QF 4-3", "SF 1-2", "Final 1"}
	if got := csvColumn(t, out, 0); !equalStrings(got, want) {
		t.Errorf("labels = %v, want %v", got, want)
	}
}

// The event is fetched only for its playoff_type. When it cannot be had, the
// listing still prints, labelling playoff matches by the season's format.
func TestEventMatchesLabelsWithoutTheEvent(t *testing.T) {
	srv := newFakeTBA(t, map[string]any{
		"/event/2024cthar/matches": matches2024ctharJSON,
	})
	out, _, err := runCmd(t, srv, "event", "matches", "2024cthar", "--format", "csv")
	requireNoError(t, err, "")

	records := parseCSV(t, out)
	if got := findRow(t, records, "2024cthar_sf13m1")[0]; got != "SF 13" {
		t.Errorf("semifinal label = %q, want the 2023+ double-elim form", got)
	}
}

func TestEventMatchesFetchesTheEventOnlyOnce(t *testing.T) {
	srv := eventMatchesServer(t)
	_, _, err := runCmd(t, srv, "event", "matches", "2024cthar")
	requireNoError(t, err, "")

	want := []string{"/event/2024cthar/matches", "/event/2024cthar"}
	if got := requestPaths(t, srv); !equalStrings(got, want) {
		t.Errorf("requests = %v, want %v", got, want)
	}
}

func TestEventMatchesFilterByTeam(t *testing.T) {
	srv := eventMatchesServer(t)
	for _, arg := range []string{"177", "frc177", "FRC177"} {
		t.Run(arg, func(t *testing.T) {
			out, _, err := runCmd(t, srv, "event", "matches", "2024cthar", "--team", arg, "--format", "csv")
			requireNoError(t, err, "")

			want := []string{"2024cthar_qm3", "2024cthar_qm12", "2024cthar_sf13m1", "2024cthar_f1m2"}
			if got := csvColumn(t, out, 1); !equalStrings(got, want) {
				t.Errorf("matches = %v, want %v", got, want)
			}
		})
	}
}

// A team that played no matches is an empty listing, not an error.
func TestEventMatchesFilterByTeamWithNoMatches(t *testing.T) {
	srv := eventMatchesServer(t)
	out, errOut, err := runCmd(t, srv, "event", "matches", "2024cthar", "--team", "9999", "--format", "csv")
	requireNoError(t, err, errOut)
	if got := csvColumn(t, out, 1); len(got) != 0 {
		t.Errorf("matches = %v, want none", got)
	}
}

func TestEventMatchesFilterByLevel(t *testing.T) {
	cases := map[string][]string{
		"qm":      {"2024cthar_qm2", "2024cthar_qm3", "2024cthar_qm7", "2024cthar_qm12"},
		"playoff": {"2024cthar_sf13m1", "2024cthar_f1m2"},
		"sf":      {"2024cthar_sf13m1"},
		"f":       {"2024cthar_f1m2"},
		"qf":      {},
	}
	for level, want := range cases {
		t.Run(level, func(t *testing.T) {
			srv := eventMatchesServer(t)
			out, _, err := runCmd(t, srv, "event", "matches", "2024cthar", "--level", level, "--format", "csv")
			requireNoError(t, err, "")
			if got := csvColumn(t, out, 1); !equalStrings(got, want) {
				t.Errorf("matches = %v, want %v", got, want)
			}
		})
	}
}

func TestEventMatchesLevelAcceptsSpellings(t *testing.T) {
	srv := eventMatchesServer(t)
	for _, level := range []string{"QM", "qual", "quals", "qualification"} {
		t.Run(level, func(t *testing.T) {
			out, _, err := runCmd(t, srv, "event", "matches", "2024cthar", "--level", level, "--format", "csv")
			requireNoError(t, err, "")
			if got := csvColumn(t, out, 1); len(got) != 4 {
				t.Errorf("matches = %v, want the four qualification matches", got)
			}
		})
	}
}

func TestEventMatchesRejectsAnUnknownLevel(t *testing.T) {
	srv := eventMatchesServer(t)
	_, _, err := runCmd(t, srv, "event", "matches", "2024cthar", "--level", "semis")
	requireErrorContains(t, err, `invalid --level "semis"`)
	requireErrorContains(t, err, "qm, playoff, ef, qf, sf, f")
	if got := clierr.ExitCode(err); got != clierr.ExitUsage {
		t.Errorf("exit code = %d, want %d", got, clierr.ExitUsage)
	}
}

// --upcoming answers "what is next", so it drops played matches and orders
// what is left by the clock rather than by playing order.
func TestEventMatchesUpcoming(t *testing.T) {
	srv := eventMatchesServer(t)
	out, _, err := runCmd(t, srv, "event", "matches", "2024cthar", "--upcoming", "--format", "csv")
	requireNoError(t, err, "")

	if got := csvColumn(t, out, 1); !equalStrings(got, []string{"2024cthar_qm2"}) {
		t.Errorf("matches = %v, want only the unplayed one", got)
	}
}

func TestEventMatchesUpcomingOrdersByTime(t *testing.T) {
	srv := newFakeTBA(t, map[string]any{
		"/event/2024cthar": event2024ctharJSON,
		// A finals match is scheduled before a qualification match, which
		// playing order would never produce.
		"/event/2024cthar/matches": `[
  {"key": "2024cthar_qm9", "comp_level": "qm", "set_number": 1, "match_number": 9, "event_key": "2024cthar",
   "time": 1711200000, "predicted_time": 1711260000, "actual_time": null, "winning_alliance": "",
   "alliances": {"red": {"score": -1, "team_keys": ["frc177"], "surrogate_team_keys": [], "dq_team_keys": []},
                 "blue": {"score": -1, "team_keys": ["frc230"], "surrogate_team_keys": [], "dq_team_keys": []}},
   "score_breakdown": null, "videos": []},
  {"key": "2024cthar_f1m1", "comp_level": "f", "set_number": 1, "match_number": 1, "event_key": "2024cthar",
   "time": 1711190000, "predicted_time": 1711190000, "actual_time": null, "winning_alliance": "",
   "alliances": {"red": {"score": -1, "team_keys": ["frc177"], "surrogate_team_keys": [], "dq_team_keys": []},
                 "blue": {"score": -1, "team_keys": ["frc230"], "surrogate_team_keys": [], "dq_team_keys": []}},
   "score_breakdown": null, "videos": []}
]`,
	})
	out, _, err := runCmd(t, srv, "event", "matches", "2024cthar", "--upcoming", "--format", "csv")
	requireNoError(t, err, "")
	if got := csvColumn(t, out, 1); !equalStrings(got, []string{"2024cthar_f1m1", "2024cthar_qm9"}) {
		t.Errorf("matches = %v, want the earlier finals match first", got)
	}
}

func TestEventMatchesFiltersCombine(t *testing.T) {
	srv := eventMatchesServer(t)
	out, _, err := runCmd(t, srv, "event", "matches", "2024cthar",
		"--team", "177", "--level", "playoff", "--format", "csv")
	requireNoError(t, err, "")
	if got := csvColumn(t, out, 1); !equalStrings(got, []string{"2024cthar_sf13m1", "2024cthar_f1m2"}) {
		t.Errorf("matches = %v", got)
	}
}

// A filter narrows the JSON too, so --jq and --format csv see the same rows.
func TestEventMatchesFiltersApplyToJSON(t *testing.T) {
	srv := eventMatchesServer(t)
	out, _, err := runCmd(t, srv, "event", "matches", "2024cthar", "--level", "playoff", "--json")
	requireNoError(t, err, "")

	arr := decodeJSON(t, out).([]any)
	if len(arr) != 2 {
		t.Fatalf("got %d matches, want 2", len(arr))
	}
	if key := arr[0].(map[string]any)["key"]; key != "2024cthar_sf13m1" {
		t.Errorf("first match = %v", key)
	}
}

// JSON output is the API's own shape, unchanged by the table's columns.
func TestEventMatchesJSONRoundTripsThroughTheTable(t *testing.T) {
	srv := eventMatchesServer(t)
	out, _, err := runCmd(t, srv, "event", "matches", "2024cthar", "--json")
	requireNoError(t, err, "")

	arr := decodeJSON(t, out).([]any)
	first := arr[0].(map[string]any)
	if first["key"] != "2024cthar_qm2" {
		t.Errorf("JSON is not in playing order, first = %v", first["key"])
	}
	alliances := first["alliances"].(map[string]any)
	if _, ok := alliances["red"].(map[string]any)["team_keys"]; !ok {
		t.Error("the API's own alliance shape should survive")
	}
}

// Color is for a terminal. A csv is read by another program, so it never gets
// escapes even when the user asks for color.
func TestEventMatchesCSVIsNeverColored(t *testing.T) {
	srv := eventMatchesServer(t)
	out, _, err := runCmd(t, srv, "event", "matches", "2024cthar", "--format", "csv", "--color", "always")
	requireNoError(t, err, "")
	if strings.Contains(out, "\x1b") {
		t.Errorf("csv output contains ANSI escapes:\n%q", out)
	}
}

func TestEventMatchesColorsTheAllianceCells(t *testing.T) {
	srv := eventMatchesServer(t)
	out, _, err := runCmd(t, srv, "event", "matches", "2024cthar", "--format", "table", "--color", "always")
	requireNoError(t, err, "")

	if !strings.Contains(out, "\x1b[31m177, 1073, 5507\x1b[0m") {
		t.Errorf("the red alliance is not colored red:\n%q", out)
	}
	if !strings.Contains(out, "\x1b[34m230, 195, 558\x1b[0m") {
		t.Errorf("the blue alliance is not colored blue:\n%q", out)
	}
	if !strings.Contains(out, "\x1b[31mred\x1b[0m") {
		t.Errorf("a red winner is not colored red:\n%q", out)
	}
}

// Color must not move the columns: the drawn table is the same either way.
func TestEventMatchesColorDoesNotChangeAlignment(t *testing.T) {
	srv := eventMatchesServer(t)
	plain, _, err := runCmd(t, srv, "event", "matches", "2024cthar", "--format", "table", "--color", "never")
	requireNoError(t, err, "")
	colored, _, err := runCmd(t, srv, "event", "matches", "2024cthar", "--format", "table", "--color", "always")
	requireNoError(t, err, "")

	if got := output.StripANSI(colored); got != plain {
		t.Errorf("colored table draws as\n%s\nwant\n%s", got, plain)
	}
}

func TestEventMatchesColorIsOffWhenNotATerminal(t *testing.T) {
	srv := eventMatchesServer(t)
	out, _, err := runCmd(t, srv, "event", "matches", "2024cthar", "--format", "table")
	requireNoError(t, err, "")
	if strings.Contains(out, "\x1b") {
		t.Errorf("a non-terminal table should not be colored:\n%q", out)
	}
}

// The time source is a column of its own, so a user who does not care can drop
// it without losing the time.
func TestEventMatchesTimeSourceIsADroppableColumn(t *testing.T) {
	srv := eventMatchesServer(t)
	out, _, err := runCmd(t, srv, "event", "matches", "2024cthar",
		"--columns", "match,time", "--format", "csv")
	requireNoError(t, err, "")

	records := parseCSV(t, out)
	if !equalStrings(records[0], []string{"Match", "Time"}) {
		t.Errorf("header = %v", records[0])
	}
	if records[1][1] != localDateTime(1711122000) {
		t.Errorf("time = %q", records[1][1])
	}
}

// A 2021 remote event ran no matches. The listing is empty, and that is not an
// error — but a bare header row is not an answer either, so the reason goes to
// stderr, where it cannot land in a file the table was piped into.
func TestEventMatchesWithNoMatches(t *testing.T) {
	srv := newFakeTBA(t, map[string]any{
		"/event/2021ctwat":         event2021ctwatJSON,
		"/event/2021ctwat/matches": "[]",
	})
	out, errOut, err := runCmd(t, srv, "event", "matches", "2021ctwat", "--format", "table")
	requireNoError(t, err, errOut)

	if got := lines(out); got[0] != strings.Join([]string{}, "") && !strings.HasPrefix(got[0], "Match") {
		t.Errorf("want just a header, got:\n%s", out)
	}
	if len(lines(out)) != 2 {
		t.Errorf("want a header and its separator only, got:\n%s", out)
	}
	if want := "note: no matches posted yet for 2021ctwat\n"; errOut != want {
		t.Errorf("stderr = %q, want %q", errOut, want)
	}
}

// Every format a person reads gets the note; JSON does not, because an empty
// array already says it and its reader is a program.
func TestEventMatchesEmptyNoteByFormat(t *testing.T) {
	for _, format := range []string{"table", "csv", "tsv", "markdown"} {
		t.Run(format, func(t *testing.T) {
			srv := newFakeTBA(t, map[string]any{"/event/2021ctwat/matches": "[]"})
			_, errOut, err := runCmd(t, srv, "event", "matches", "2021ctwat", "--format", format)
			requireNoError(t, err, errOut)
			if want := "note: no matches posted yet for 2021ctwat\n"; errOut != want {
				t.Errorf("stderr = %q, want %q", errOut, want)
			}
		})
	}
}

func TestEventMatchesEmptyJSONIsAnEmptyArray(t *testing.T) {
	srv := newFakeTBA(t, map[string]any{"/event/2021ctwat/matches": "[]"})
	out, errOut, err := runCmd(t, srv, "event", "matches", "2021ctwat", "--json")
	requireNoError(t, err, errOut)
	if strings.TrimSpace(out) != "[]" {
		t.Errorf("stdout = %q, want an empty array", out)
	}
	if errOut != "" {
		t.Errorf("stderr = %q, want nothing alongside JSON", errOut)
	}
}

// A filter that matched nothing is a different answer from an event with no
// schedule, and says so.
func TestEventMatchesNoteWhenTheTeamFilterEmptiesTheListing(t *testing.T) {
	srv := eventMatchesServer(t)
	_, errOut, err := runCmd(t, srv, "event", "matches", "2024cthar", "--team", "9999", "--format", "table")
	requireNoError(t, err, errOut)
	if want := "note: no matches for team 9999 at 2024cthar\n"; errOut != want {
		t.Errorf("stderr = %q, want %q", errOut, want)
	}
}

func TestEventMatchesNoteWhenTheLevelFilterEmptiesTheListing(t *testing.T) {
	srv := eventMatchesServer(t)
	_, errOut, err := runCmd(t, srv, "event", "matches", "2024cthar", "--level", "qf", "--format", "table")
	requireNoError(t, err, errOut)
	if want := "note: no qf matches for 2024cthar\n"; errOut != want {
		t.Errorf("stderr = %q, want %q", errOut, want)
	}
}

// An event whose matches are all played has nothing upcoming, which is what
// asking on the Monday after looks like.
func TestEventMatchesNoteWhenNothingIsUpcoming(t *testing.T) {
	srv := newFakeTBA(t, map[string]any{
		"/event/2019ctwat":         event2019ctwatJSON,
		"/event/2019ctwat/matches": matches2019ctwatJSON,
	})
	_, errOut, err := runCmd(t, srv, "event", "matches", "2019ctwat", "--upcoming", "--format", "table")
	requireNoError(t, err, errOut)
	if want := "note: no upcoming matches for 2019ctwat\n"; errOut != want {
		t.Errorf("stderr = %q, want %q", errOut, want)
	}
}

// A team's season listing names the team and the season it found nothing in.
func TestTeamMatchesNoteWhenASeasonIsEmpty(t *testing.T) {
	srv := newFakeTBA(t, map[string]any{"/team/frc177/matches/2024": "[]"})
	_, errOut, err := runCmd(t, srv, "team", "matches", "177", "--year", "2024", "--format", "table")
	requireNoError(t, err, errOut)
	if want := "note: no matches posted yet for team 177 in 2024\n"; errOut != want {
		t.Errorf("stderr = %q, want %q", errOut, want)
	}
}

// At one event it names the event, the way the question was asked.
func TestTeamMatchesNoteAtAnEvent(t *testing.T) {
	srv := newFakeTBA(t, map[string]any{
		"/team/frc177/event/2021ctwat/matches": "[]",
	})
	_, errOut, err := runCmd(t, srv, "team", "matches", "177", "--event", "2021ctwat", "--format", "table")
	requireNoError(t, err, errOut)
	if want := "note: no matches posted yet for 2021ctwat\n"; errOut != want {
		t.Errorf("stderr = %q, want %q", errOut, want)
	}
}

// A listing with rows in it says nothing at all.
func TestEventMatchesSaysNothingWhenItHasRows(t *testing.T) {
	srv := newFakeTBA(t, map[string]any{
		"/event/2019ctwat":         event2019ctwatJSON,
		"/event/2019ctwat/matches": matches2019ctwatJSON,
	})
	_, errOut, err := runCmd(t, srv, "event", "matches", "2019ctwat", "--format", "table")
	requireNoError(t, err, errOut)
	if errOut != "" {
		t.Errorf("stderr = %q, want nothing", errOut)
	}
}

// A 2015 match had no ties to report and no predicted times; its scheduled
// time is all there is.
func TestEventMatchesFor2015(t *testing.T) {
	srv := newFakeTBA(t, map[string]any{
		"/event/2015ctwat/matches": "[" + match2015ctwatQM7JSON + "]",
	})
	out, _, err := runCmd(t, srv, "event", "matches", "2015ctwat", "--format", "csv")
	requireNoError(t, err, "")

	want := []string{
		"Qual 7", "2015ctwat_qm7",
		"177, 1071, 2168", "230, 195, 558",
		"44-44", "tie",
		localTime(1427464800), "", "scheduled", "Played",
	}
	if got := findRow(t, parseCSV(t, out), "2015ctwat_qm7"); !equalStrings(got, want) {
		t.Errorf("row =\n%v\nwant\n%v", got, want)
	}
}

func TestTeamMatchesColumnsAndOrder(t *testing.T) {
	srv := newFakeTBA(t, map[string]any{
		"/team/frc177/matches/2024": matches2024ctharJSON,
	})
	out, _, err := runCmd(t, srv, "team", "matches", "177", "--year", "2024", "--format", "csv")
	requireNoError(t, err, "")

	records := parseCSV(t, out)
	wantHeader := append([]string{"Event"}, matchHeaders...)
	if !equalStrings(records[0], wantHeader) {
		t.Errorf("header = %v, want %v", records[0], wantHeader)
	}
	want := []string{"2024cthar_qm2", "2024cthar_qm3", "2024cthar_qm7", "2024cthar_qm12", "2024cthar_sf13m1", "2024cthar_f1m2"}
	if got := csvColumn(t, out, 2); !equalStrings(got, want) {
		t.Errorf("order = %v, want %v", got, want)
	}
}

// Without --event the listing spans a season, so there is no single bracket to
// ask about: the second request is the team's event list, which orders the
// groups, not an event fetched for its playoff type.
func TestTeamMatchesForAYearFetchesTheTeamsEvents(t *testing.T) {
	srv := newFakeTBA(t, map[string]any{
		"/team/frc177/matches/2024": matches2024ctharJSON,
		"/team/frc177/events/2024":  teamEvents177In2024JSON,
	})
	_, _, err := runCmd(t, srv, "team", "matches", "177", "--year", "2024")
	requireNoError(t, err, "")

	want := []string{"/team/frc177/matches/2024", "/team/frc177/events/2024"}
	if got := requestPaths(t, srv); !equalStrings(got, want) {
		t.Errorf("requests = %v, want %v", got, want)
	}
}

// A season's listing is grouped by event, in the order the team competed, and
// each row says which event it belongs to. Sorted as one list, Waterbury's
// Qual 46 would land next to Hartford's.
func TestTeamMatchesGroupsASeasonByEvent(t *testing.T) {
	srv := newFakeTBA(t, map[string]any{
		"/team/frc177/matches/2024": teamMatches177Season2024JSON,
		"/team/frc177/events/2024":  teamEvents177In2024JSON,
	})
	out, _, err := runCmd(t, srv, "team", "matches", "177", "--year", "2024", "--format", "csv")
	requireNoError(t, err, "")

	records := parseCSV(t, out)
	if records[0][0] != "Event" {
		t.Errorf("header = %v, want the Event column first", records[0])
	}
	// Waterbury ran in week 1 and Hartford in week 3, though the API listed
	// the events the other way round.
	wantKeys := []string{"2024ctwat_qm5", "2024ctwat_qm46", "2024cthar_qm12", "2024cthar_qm46"}
	if got := csvColumn(t, out, 2); !equalStrings(got, wantKeys) {
		t.Errorf("order = %v, want %v", got, wantKeys)
	}
	wantEvents := []string{"2024ctwat", "2024ctwat", "2024cthar", "2024cthar"}
	if got := csvColumn(t, out, 0); !equalStrings(got, wantEvents) {
		t.Errorf("event column = %v, want %v", got, wantEvents)
	}
}

// The event list only orders the groups. Without it the listing still groups,
// by event key, rather than interleaving the season.
func TestTeamMatchesGroupsWithoutTheEventList(t *testing.T) {
	srv := newFakeTBA(t, map[string]any{
		"/team/frc177/matches/2024": teamMatches177Season2024JSON,
	})
	out, errOut, err := runCmd(t, srv, "team", "matches", "177", "--year", "2024", "--format", "csv")
	requireNoError(t, err, errOut)

	wantKeys := []string{"2024cthar_qm12", "2024cthar_qm46", "2024ctwat_qm5", "2024ctwat_qm46"}
	if got := csvColumn(t, out, 2); !equalStrings(got, wantKeys) {
		t.Errorf("order = %v, want %v", got, wantKeys)
	}
}

// One event needs no Event column: every row would carry the same key.
func TestTeamMatchesAtAnEventHasNoEventColumn(t *testing.T) {
	srv := newFakeTBA(t, map[string]any{
		"/event/2024cthar":                     event2024ctharJSON,
		"/team/frc177/event/2024cthar/matches": teamMatches177At2024ctharJSON,
	})
	out, _, err := runCmd(t, srv, "team", "matches", "177", "--event", "2024cthar", "--format", "csv")
	requireNoError(t, err, "")
	if got := parseCSV(t, out)[0][0]; got != "Match" {
		t.Errorf("first column = %q, want the match label", got)
	}
}

// An event listing never carries the column either: the key is in the command.
func TestEventMatchesHasNoEventColumn(t *testing.T) {
	srv := eventMatchesServer(t)
	out, _, err := runCmd(t, srv, "event", "matches", "2024cthar", "--format", "csv")
	requireNoError(t, err, "")
	if got := parseCSV(t, out)[0][0]; got != "Match" {
		t.Errorf("first column = %q, want the match label", got)
	}
}

func TestTeamMatchesAtAnEvent(t *testing.T) {
	srv := newFakeTBA(t, map[string]any{
		"/event/2024cthar":                     event2024ctharJSON,
		"/team/frc177/event/2024cthar/matches": teamMatches177At2024ctharJSON,
	})
	out, _, err := runCmd(t, srv, "team", "matches", "177", "--event", "2024cthar", "--format", "csv")
	requireNoError(t, err, "")

	want := []string{"/team/frc177/event/2024cthar/matches", "/event/2024cthar"}
	if got := requestPaths(t, srv); !equalStrings(got, want) {
		t.Errorf("requests = %v, want %v", got, want)
	}
	if got := csvColumn(t, out, 0); !equalStrings(got, []string{"Qual 12", "SF 13"}) {
		t.Errorf("labels = %v", got)
	}
}

// --event names one event, so --year has nothing to say.
func TestTeamMatchesEventOverridesYear(t *testing.T) {
	srv := newFakeTBA(t, map[string]any{
		"/event/2024cthar":                     event2024ctharJSON,
		"/team/frc177/event/2024cthar/matches": teamMatches177At2024ctharJSON,
	})
	_, _, err := runCmd(t, srv, "team", "matches", "177", "--event", "2024cthar", "--year", "2019")
	requireNoError(t, err, "")
	for _, p := range requestPaths(t, srv) {
		if strings.Contains(p, "2019") {
			t.Errorf("--year should be ignored alongside --event, but requested %s", p)
		}
	}
}

func TestTeamMatchesRejectsAMalformedEventKey(t *testing.T) {
	srv := newFakeTBA(t, map[string]any{})
	_, _, err := runCmd(t, srv, "team", "matches", "177", "--event", "not-a-key")
	requireErrorContains(t, err, "is not a valid event key")
	if got := requestPaths(t, srv); len(got) != 0 {
		t.Errorf("a malformed key should not reach the API, got %v", got)
	}
}

func TestTeamMatchesUpcoming(t *testing.T) {
	srv := newFakeTBA(t, map[string]any{
		"/event/2024cthar":                     event2024ctharJSON,
		"/team/frc177/event/2024cthar/matches": teamMatches177At2024ctharJSON,
	})
	out, _, err := runCmd(t, srv, "team", "matches", "177", "--event", "2024cthar", "--upcoming", "--format", "csv")
	requireNoError(t, err, "")
	if got := csvColumn(t, out, 1); !equalStrings(got, []string{"2024cthar_sf13m1"}) {
		t.Errorf("matches = %v, want only the unplayed one", got)
	}
}

func TestTeamMatchesFilterByLevel(t *testing.T) {
	srv := newFakeTBA(t, map[string]any{
		"/team/frc177/matches/2024": matches2024ctharJSON,
	})
	out, _, err := runCmd(t, srv, "team", "matches", "177", "--year", "2024", "--level", "playoff", "--format", "csv")
	requireNoError(t, err, "")
	if got := csvColumn(t, out, 2); !equalStrings(got, []string{"2024cthar_sf13m1", "2024cthar_f1m2"}) {
		t.Errorf("matches = %v", got)
	}
}

// `team matches 177` is already about one team, so a --team of its own could
// only disagree with the argument — and `--team 254` used to print an empty
// table rather than say so.
func TestTeamMatchesHasNoTeamFlag(t *testing.T) {
	srv := newFakeTBA(t, map[string]any{
		"/team/frc177/matches/2024": matches2024ctharJSON,
	})
	_, _, err := runCmd(t, srv, "team", "matches", "177", "--year", "2024", "--team", "254")
	requireErrorContains(t, err, "unknown flag: --team")
	if got := clierr.ExitCode(err); got != clierr.ExitUsage {
		t.Errorf("exit code = %d, want %d", got, clierr.ExitUsage)
	}
	if got := requestPaths(t, srv); len(got) != 0 {
		t.Errorf("a rejected flag should not reach the API, got %v", got)
	}
}

// The listing about an event keeps --team: there, narrowing to one team is the
// whole point.
func TestEventMatchesKeepsTheTeamFlag(t *testing.T) {
	srv := eventMatchesServer(t)
	out, _, err := runCmd(t, srv, "event", "matches", "2024cthar", "--team", "5507", "--format", "csv")
	requireNoError(t, err, "")
	want := []string{"2024cthar_qm12", "2024cthar_sf13m1", "2024cthar_f1m2"}
	if got := csvColumn(t, out, 1); !equalStrings(got, want) {
		t.Errorf("matches = %v, want %v", got, want)
	}
}

// The filters that do make sense on a team's listing still do.
func TestTeamMatchesKeepsTheOtherFilters(t *testing.T) {
	srv := newFakeTBA(t, map[string]any{
		"/team/frc177/matches/2024": matches2024ctharJSON,
	})
	out, _, err := runCmd(t, srv, "team", "matches", "177", "--year", "2024",
		"--level", "playoff", "--format", "csv")
	requireNoError(t, err, "")
	if got := csvColumn(t, out, 2); !equalStrings(got, []string{"2024cthar_sf13m1", "2024cthar_f1m2"}) {
		t.Errorf("matches = %v", got)
	}
}

// Test helpers.

func parseCSV(t *testing.T, s string) [][]string {
	t.Helper()
	records, err := csv.NewReader(strings.NewReader(s)).ReadAll()
	if err != nil {
		t.Fatalf("output is not valid CSV: %v\n---\n%s", err, s)
	}
	return records
}

// csvColumn returns one column of a csv body, without its header.
func csvColumn(t *testing.T, s string, col int) []string {
	t.Helper()
	records := parseCSV(t, s)
	out := make([]string, 0, len(records))
	for _, row := range records[1:] {
		if col < len(row) {
			out = append(out, row[col])
		}
	}
	return out
}

// findRow returns the csv row whose Key column is key.
func findRow(t *testing.T, records [][]string, key string) []string {
	t.Helper()
	for _, row := range records[1:] {
		if len(row) > 1 && row[1] == key {
			return row
		}
	}
	t.Fatalf("no row for %q in %v", key, records)
	return nil
}

// columnValues reads a column out of an aligned text table, whose cells are
// padded and separated by two spaces.
func columnValues(t *testing.T, s string, col int) []string {
	t.Helper()
	rows := lines(s)
	if len(rows) < 2 {
		return nil
	}
	out := make([]string, 0, len(rows))
	for _, row := range rows[2:] {
		fields := strings.Split(row, "  ")
		if col < len(fields) {
			out = append(out, strings.TrimSpace(fields[col]))
		}
	}
	return out
}

func equalStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// One competition day needs no date on every row: the weekday and the clock
// are what a team in the pits reads.
func TestEventMatchesOmitsTheDateWithinOneDay(t *testing.T) {
	srv := newFakeTBA(t, map[string]any{
		"/event/2024cthar/matches": "[" + match2024ctharQM1JSON + "," + match2024ctharQM2JSON + "]",
	})
	out, _, err := runCmd(t, srv, "event", "matches", "2024cthar", "--format", "csv")
	requireNoError(t, err, "")

	if got := findRow(t, parseCSV(t, out), "2024cthar_qm1")[6]; got != localTime(1711120920) {
		t.Errorf("time = %q, want %q", got, localTime(1711120920))
	}
}

// A listing that spans days writes the date, because a weekday alone could be
// any weekend of the season.
func TestEventMatchesAddsTheDateAcrossDays(t *testing.T) {
	srv := eventMatchesServer(t)
	out, _, err := runCmd(t, srv, "event", "matches", "2024cthar", "--format", "csv")
	requireNoError(t, err, "")

	row := findRow(t, parseCSV(t, out), "2024cthar_f1m2")
	if got := row[6]; got != localDateTime(1711307040) {
		t.Errorf("time = %q, want %q", got, localDateTime(1711307040))
	}
}

// A season's listing always spans days, so every row carries its date.
func TestTeamMatchesSeasonTimesCarryTheDate(t *testing.T) {
	srv := newFakeTBA(t, map[string]any{
		"/team/frc177/matches/2024": teamMatches177Season2024JSON,
		"/team/frc177/events/2024":  teamEvents177In2024JSON,
	})
	out, _, err := runCmd(t, srv, "team", "matches", "177", "--year", "2024", "--format", "csv")
	requireNoError(t, err, "")

	// Column 7 is Time, one to the right of the season listing's Event column.
	if got := findRow2(t, parseCSV(t, out), "2024ctwat_qm5")[7]; got != localDateTime(1709913780) {
		t.Errorf("time = %q, want %q", got, localDateTime(1709913780))
	}
}

// The countdown answers "when", which only a match still to come has: a played
// one has a score instead, and a relative time on it would change every run.
func TestEventMatchesWhenCountsDownUnplayedMatchesOnly(t *testing.T) {
	withNow(t, time.Unix(1711122000-7200, 0))
	srv := eventMatchesServer(t)
	out, _, err := runCmd(t, srv, "event", "matches", "2024cthar", "--format", "csv")
	requireNoError(t, err, "")

	records := parseCSV(t, out)
	if got := findRow(t, records, "2024cthar_qm2")[7]; got != "in 2h" {
		t.Errorf("When = %q, want %q", got, "in 2h")
	}
	for _, key := range []string{"2024cthar_qm3", "2024cthar_qm7", "2024cthar_qm12", "2024cthar_f1m2"} {
		if got := findRow(t, records, key)[7]; got != "" {
			t.Errorf("When for the played %s = %q, want it empty", key, got)
		}
	}
}

// findRow2 is findRow for a season listing, whose Key column sits behind the
// Event column.
func findRow2(t *testing.T, records [][]string, key string) []string {
	t.Helper()
	for _, row := range records[1:] {
		if len(row) > 2 && row[2] == key {
			return row
		}
	}
	t.Fatalf("no row for %q in %v", key, records)
	return nil
}
