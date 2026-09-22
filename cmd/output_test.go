package cmd

import (
	"strings"
	"testing"

	"github.com/the-blue-alliance/tba-cli/internal/clierr"
)

// districtsCmd is the smallest command that returns a stable multi-row table,
// which makes it the workhorse for the presentation flags.
func districtsCmd(t *testing.T, args ...string) string {
	t.Helper()
	srv := newFakeTBA(t, map[string]any{"/districts/2024": districts2024JSON})
	out, _, err := runCmd(t, srv, append([]string{"district", "list", "--year", "2024"}, args...)...)
	requireNoError(t, err, "")
	return out
}

func districtsCmdErr(t *testing.T, args ...string) error {
	t.Helper()
	srv := newFakeTBA(t, map[string]any{"/districts/2024": districts2024JSON})
	_, _, err := runCmd(t, srv, append([]string{"district", "list", "--year", "2024"}, args...)...)
	if err == nil {
		t.Fatalf("want an error from %v", args)
	}
	return err
}

func TestColorAlwaysBoldsTheTableHeader(t *testing.T) {
	out := districtsCmd(t, "--format", "table", "--color", "always")
	got := lines(out)
	if !strings.HasPrefix(got[0], "\x1b[1m") || !strings.HasSuffix(got[0], "\x1b[0m") {
		t.Errorf("header = %q, want it wrapped in a bold escape", got[0])
	}
	for _, line := range got[1:] {
		if strings.Contains(line, "\x1b") {
			t.Errorf("only the header is colored, but %q carries an escape", line)
		}
	}
}

func TestColorAutoDoesNotColorAPipe(t *testing.T) {
	// The test harness writes into a buffer, which is never a terminal.
	out := districtsCmd(t, "--format", "table")
	if strings.Contains(out, "\x1b") {
		t.Errorf("piped output must be plain, got %q", out)
	}
}

func TestNoColorBeatsColorAlways(t *testing.T) {
	out := districtsCmd(t, "--format", "table", "--color", "always", "--no-color")
	if strings.Contains(out, "\x1b") {
		t.Errorf("--no-color must win, got %q", out)
	}
}

// no-color wins wherever either setting came from, so a script can export it
// unconditionally; `tba config list --help` promises exactly this.
func TestNoColorBeatsColorFromEveryLayer(t *testing.T) {
	t.Setenv("TBA_COLOR", "always")
	out := districtsCmd(t, "--format", "table", "--no-color")
	if strings.Contains(out, "\x1b") {
		t.Errorf("no-color must win over TBA_COLOR, got %q", out)
	}

	help, _, err := runCmd(t, nil, "config", "list", "--help")
	requireNoError(t, err, "")
	requireContains(t, help, "no-color")
	requireContains(t, help, "wins")
}

// A bad --color is still reported when --no-color would have settled the
// question, so a typo in the config file does not go unnoticed.
func TestBadColorValueIsReportedEvenWithNoColor(t *testing.T) {
	srv := newFakeTBA(t, map[string]any{"/districts/2024": districts2024JSON})
	_ = requireExitCode(t, clierr.ExitUsage, srv,
		"district", "list", "--year", "2024", "--color", "sometimes", "--no-color")
}

func TestSortFlagHelpShowsBothForms(t *testing.T) {
	out, _, err := runCmd(t, nil, "district", "list", "--help")
	requireNoError(t, err, "")
	requireContains(t, out, "prefix with - to descend (e.g. --sort=-opr)")
	if strings.Contains(out, "use the --sort=-col form") {
		t.Errorf("the --sort help still claims only one form works:\n%s", out)
	}
}

// Both spellings work, so the help must not imply otherwise.
func TestSortAcceptsASeparateArgument(t *testing.T) {
	spaced := districtsCmd(t, "--format", "csv", "--sort", "-name")
	equals := districtsCmd(t, "--format=csv", "--sort=-name")
	if spaced != equals {
		t.Errorf("--sort -name = %q but --sort=-name = %q", spaced, equals)
	}
}

func TestColorNever(t *testing.T) {
	out := districtsCmd(t, "--format", "table", "--color", "never")
	if strings.Contains(out, "\x1b") {
		t.Errorf("--color never must not color, got %q", out)
	}
}

func TestInvalidColorIsAnError(t *testing.T) {
	err := districtsCmdErr(t, "--color", "sometimes")
	if !strings.Contains(err.Error(), "auto, always, never") {
		t.Errorf("error = %v, want it to list the valid values", err)
	}
}

func TestNoHeadersTable(t *testing.T) {
	out := districtsCmd(t, "--format", "table", "--no-headers")
	got := lines(out)
	if len(got) != 2 {
		t.Fatalf("want 2 rows with no header and no separator, got %d:\n%s", len(got), out)
	}
	// Widths come from the data alone once the header is gone.
	if got[0] != "2024ne   New England        ne " {
		t.Errorf("row 1 = %q", got[0])
	}
	if got[1] != "2024fim  FIRST In Michigan  fim" {
		t.Errorf("row 2 = %q", got[1])
	}
}

func TestNoHeadersCSV(t *testing.T) {
	out := districtsCmd(t, "--format", "csv", "--no-headers")
	if out != "2024ne,New England,ne\n2024fim,FIRST In Michigan,fim\n" {
		t.Errorf("csv = %q", out)
	}
}

func TestNoHeadersTSV(t *testing.T) {
	out := districtsCmd(t, "--format", "tsv", "--no-headers")
	if out != "2024ne\tNew England\tne\n2024fim\tFIRST In Michigan\tfim\n" {
		t.Errorf("tsv = %q", out)
	}
}

func TestNoHeadersMarkdown(t *testing.T) {
	out := districtsCmd(t, "--format", "markdown", "--no-headers")
	want := "| 2024ne | New England | ne |\n| 2024fim | FIRST In Michigan | fim |\n"
	if out != want {
		t.Errorf("markdown = %q", out)
	}
}

func TestNoHeadersLeavesJSONAlone(t *testing.T) {
	out := districtsCmd(t, "--json", "--no-headers")
	arr := decodeJSON(t, out).([]any)
	if len(arr) != 2 {
		t.Fatalf("want 2 districts, got %d", len(arr))
	}
}

func TestColumnsSelectsAndReorders(t *testing.T) {
	out := districtsCmd(t, "--format", "csv", "--columns", "abbreviation,key")
	want := "Abbreviation,Key\nne,2024ne\nfim,2024fim\n"
	if out != want {
		t.Errorf("csv = %q, want %q", out, want)
	}
}

func TestColumnsMatchesHeaderNamesLoosely(t *testing.T) {
	srv := newFakeTBA(t, map[string]any{
		"/district/2024ne/events": "[" + event2024ctharJSON + "]",
	})
	out, _, err := runCmd(t, srv, "district", "events", "2024ne", "--format", "csv", "--columns", "start_date")
	requireNoError(t, err, "")
	if out != "Start Date\n2024-03-22\n" {
		t.Errorf("csv = %q", out)
	}
}

func TestColumnsAcceptsIndices(t *testing.T) {
	out := districtsCmd(t, "--format", "tsv", "--columns", "2,1")
	if out != "Name\tKey\nNew England\t2024ne\nFIRST In Michigan\t2024fim\n" {
		t.Errorf("tsv = %q", out)
	}
}

func TestColumnsWorksWithNoHeaders(t *testing.T) {
	out := districtsCmd(t, "--format", "csv", "--columns", "key", "--no-headers")
	if out != "2024ne\n2024fim\n" {
		t.Errorf("csv = %q", out)
	}
}

func TestUnknownColumnListsTheValidOnes(t *testing.T) {
	err := districtsCmdErr(t, "--format", "csv", "--columns", "nickname")
	for _, want := range []string{"nickname", "Key, Name, Abbreviation"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error = %v, want it to mention %q", err, want)
		}
	}
}

func TestColumnsWithJSONIsAnError(t *testing.T) {
	err := districtsCmdErr(t, "--json", "--columns", "key")
	want := "--columns applies to tabular formats; use --jq to shape JSON"
	if err.Error() != want {
		t.Errorf("error = %v, want %q", err, want)
	}
}

func TestSortOrdersRows(t *testing.T) {
	out := districtsCmd(t, "--format", "csv", "--sort", "name")
	want := "Key,Name,Abbreviation\n2024fim,FIRST In Michigan,fim\n2024ne,New England,ne\n"
	if out != want {
		t.Errorf("csv = %q, want %q", out, want)
	}
}

func TestSortDescends(t *testing.T) {
	out := districtsCmd(t, "--format=csv", "--sort=-name")
	want := "Key,Name,Abbreviation\n2024ne,New England,ne\n2024fim,FIRST In Michigan,fim\n"
	if out != want {
		t.Errorf("csv = %q, want %q", out, want)
	}
}

func TestSortAppliesToTheTableFormat(t *testing.T) {
	out := districtsCmd(t, "--format", "table", "--sort", "abbreviation")
	got := lines(out)
	if !strings.HasPrefix(got[2], "2024fim") {
		t.Errorf("first row = %q, want the fim district first", got[2])
	}
}

func TestSortRunsBeforeColumnSelection(t *testing.T) {
	// The sort column is not displayed, which only works if sorting happens
	// before the columns are narrowed.
	out := districtsCmd(t, "--format", "csv", "--sort", "name", "--columns", "key")
	if out != "Key\n2024fim\n2024ne\n" {
		t.Errorf("csv = %q", out)
	}
}

func TestSortIsNumericAwareAcrossACommand(t *testing.T) {
	srv := newFakeTBA(t, map[string]any{
		"/teams/2024/0": "[" + teamFRC5507JSON + "," + teamFRC177JSON + "]",
		"/teams/2024/1": "[]",
	})
	out, _, err := runCmd(t, srv, "team", "list", "--year", "2024", "--format", "tsv", "--sort", "number")
	requireNoError(t, err, "")
	got := lines(out)
	if got[1] != "177\tBobcat Robotics\tSouth Windsor, Connecticut, USA" {
		t.Errorf("first row = %q, want 177 to sort before 5507 numerically", got[1])
	}
}

func TestSortReordersJSON(t *testing.T) {
	out := districtsCmd(t, "--json", "--sort", "name")
	arr := decodeJSON(t, out).([]any)
	if len(arr) != 2 {
		t.Fatalf("want 2 districts, got %d", len(arr))
	}
	if key := arr[0].(map[string]any)["key"]; key != "2024fim" {
		t.Errorf("first element = %v, want the table's first row", key)
	}
}

func TestSortReordersJSONBeforeJq(t *testing.T) {
	out := districtsCmd(t, "--jq", ".[0].key", "--sort", "-key")
	if strings.TrimSpace(out) != `"2024ne"` {
		t.Errorf("jq output = %q", out)
	}
}

func TestUnknownSortColumnIsAnError(t *testing.T) {
	err := districtsCmdErr(t, "--format", "csv", "--sort", "nickname")
	if !strings.Contains(err.Error(), "Key, Name, Abbreviation") {
		t.Errorf("error = %v, want it to list the valid columns", err)
	}
	// The same complaint reaches a JSON caller, since --sort reorders it too.
	err = districtsCmdErr(t, "--json", "--sort", "nickname")
	if !strings.Contains(err.Error(), "unknown column") {
		t.Errorf("error = %v", err)
	}
}

// A bad --columns or --sort is a mistake in how the command was invoked, so it
// has to exit 2 like any other flag error rather than 1.
func TestBadPresentationFlagsExitTwo(t *testing.T) {
	cases := []struct {
		name string
		args []string
	}{
		{"unknown column", []string{"--format", "csv", "--columns", "Nope"}},
		{"column index out of range", []string{"--format", "csv", "--columns", "9"}},
		{"unknown sort column", []string{"--format", "csv", "--sort", "Nope"}},
		{"unknown sort column with json", []string{"--json", "--sort", "Nope"}},
		{"empty sort column", []string{"--format", "csv", "--sort", "-"}},
		{"columns with json", []string{"--json", "--columns", "key"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			srv := newFakeTBA(t, map[string]any{"/districts/2024": districts2024JSON})
			args := append([]string{"district", "list", "--year", "2024"}, tc.args...)
			_ = requireExitCode(t, clierr.ExitUsage, srv, args...)
		})
	}
}

// --sort must never be a silent no-op: either it orders the JSON, or it says
// it cannot.
func TestSortPermutesAPlainJSONList(t *testing.T) {
	out := districtsCmd(t, "--json", "--sort", "-key")
	arr := decodeJSON(t, out).([]any)
	if len(arr) != 2 {
		t.Fatalf("want 2 districts, got %d:\n%s", len(arr), out)
	}
	if got := arr[0].(map[string]any)["key"]; got != "2024ne" {
		t.Errorf("first key = %v, want 2024ne (descending)", got)
	}
}

func TestSortOnANonListJSONPayloadIsAUsageError(t *testing.T) {
	// event oprs answers with an object keyed by team, which carries no row
	// order for --sort to apply.
	srv := newFakeTBA(t, map[string]any{"/event/2024cthar/oprs": oprs2024ctharJSON})
	err := requireExitCode(t, clierr.ExitUsage, srv, "event", "oprs", "2024cthar", "--json", "--sort", "-opr")
	requireErrorContains(t, err, "--sort cannot reorder this JSON payload")

	// The same command in a tabular format still sorts.
	out, stderr, runErr := runCmd(t, srv, "event", "oprs", "2024cthar", "--format", "csv", "--sort", "-opr")
	requireNoError(t, runErr, stderr)
	if len(lines(out)) < 2 {
		t.Fatalf("want a sorted csv table, got:\n%s", out)
	}
}

// A jq expression that does not parse is a mistake in the command line, so it
// exits 2 and costs no request; one that parses but fails on the data is a
// failure of the run, and stays exit 1.
func TestMalformedJQIsAUsageError(t *testing.T) {
	for _, expr := range []string{".[", "{", "..foo", ". |"} {
		t.Run(expr, func(t *testing.T) {
			srv := newFakeTBA(t, map[string]any{"/districts/2024": districts2024JSON})
			err := requireExitCode(t, clierr.ExitUsage, srv, "district", "list", "--year", "2024", "--jq", expr)
			requireErrorContains(t, err, "invalid jq expression")
			if got := requestPaths(t, srv); len(got) != 0 {
				t.Errorf("a usage error must not reach the API, got %v", got)
			}
		})
	}
}

func TestJQRuntimeFailureIsARuntimeError(t *testing.T) {
	srv := newFakeTBA(t, map[string]any{"/districts/2024": districts2024JSON})
	// A valid program that cannot be applied to this data.
	err := requireExitCode(t, clierr.ExitFailure, srv, "district", "list", "--year", "2024", "--jq", ".[] | .key + 1")
	if err == nil {
		t.Fatal("want an error")
	}
}

// An empty table is ambiguous: nothing there, or a filter that cancelled
// itself out, or a mistyped key. The note says which, on stderr, so stdout
// stays a well-formed empty table.
func TestEmptyResultsCarryANoteOnStderr(t *testing.T) {
	cases := []struct {
		name   string
		routes map[string]any
		args   []string
		want   string
	}{
		{
			"event list with filters",
			map[string]any{"/events/2024": `[]`},
			[]string{"event", "list", "--year", "2024", "--district", "ne"},
			"note: no events match those filters\n",
		},
		{
			"event list without filters",
			map[string]any{"/events/2024": `[]`},
			[]string{"event", "list", "--year", "2024"},
			"note: no events in 2024\n",
		},
		{
			"event teams",
			map[string]any{"/event/2024cthar/teams": `[]`},
			[]string{"event", "teams", "2024cthar"},
			"note: no teams listed for 2024cthar yet\n",
		},
		{
			"team events",
			map[string]any{"/team/frc9999/events/2024": `[]`},
			[]string{"team", "events", "9999", "--year", "2024"},
			"note: no events for team 9999 in 2024\n",
		},
		{
			"team awards",
			map[string]any{"/team/frc9999/awards": `[]`},
			[]string{"team", "awards", "9999"},
			"note: no awards for team 9999\n",
		},
		{
			"team awards by type",
			map[string]any{"/team/frc9999/awards": `[]`},
			[]string{"team", "awards", "9999", "--type", "impact"},
			"note: no Chairman's/Impact awards for team 9999\n",
		},
		{
			"district list",
			map[string]any{"/districts/2024": `[]`},
			[]string{"district", "list", "--year", "2024"},
			"note: no districts in 2024\n",
		},
		{
			"district events",
			map[string]any{"/district/2024ne/events": `[]`},
			[]string{"district", "events", "2024ne"},
			"note: no events in district 2024ne\n",
		},
		{
			"district teams",
			map[string]any{"/district/2024ne/teams": `[]`},
			[]string{"district", "teams", "2024ne"},
			"note: no teams in district 2024ne\n",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			srv := newFakeTBA(t, tc.routes)
			out, stderr, err := runCmd(t, srv, append(tc.args, "--format", "csv")...)
			requireNoError(t, err, stderr)
			if stderr != tc.want {
				t.Errorf("stderr = %q, want %q", stderr, tc.want)
			}
			// stdout is still a table: the header and nothing else.
			if n := len(lines(out)); n != 1 {
				t.Errorf("stdout should be the header alone, got %d lines:\n%s", n, out)
			}
		})
	}
}

// JSON is read by scripts, which want [] and no commentary.
func TestEmptyResultsAreSilentInJSON(t *testing.T) {
	srv := newFakeTBA(t, map[string]any{"/districts/2024": `[]`})
	out, stderr, err := runCmd(t, srv, "district", "list", "--year", "2024", "--json")
	requireNoError(t, err, stderr)
	if strings.TrimSpace(out) != "[]" {
		t.Errorf("stdout = %q, want []", out)
	}
	if stderr != "" {
		t.Errorf("stderr = %q, want nothing", stderr)
	}
}

// A table with rows has nothing to say.
func TestNonEmptyResultsCarryNoNote(t *testing.T) {
	srv := newFakeTBA(t, map[string]any{"/districts/2024": districts2024JSON})
	_, stderr, err := runCmd(t, srv, "district", "list", "--year", "2024", "--format", "csv")
	requireNoError(t, err, stderr)
	if stderr != "" {
		t.Errorf("stderr = %q, want nothing", stderr)
	}
}

func TestColorAlwaysDoesNotTouchDataFormats(t *testing.T) {
	for _, format := range []string{"csv", "tsv", "markdown", "json"} {
		out := districtsCmd(t, "--format", format, "--color", "always")
		if strings.Contains(out, "\x1b") {
			t.Errorf("%s output must stay plain, got %q", format, out)
		}
	}
}
