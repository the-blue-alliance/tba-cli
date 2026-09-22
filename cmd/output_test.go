package cmd

import (
	"strings"
	"testing"
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

func TestColorAlwaysDoesNotTouchDataFormats(t *testing.T) {
	for _, format := range []string{"csv", "tsv", "markdown", "json"} {
		out := districtsCmd(t, "--format", format, "--color", "always")
		if strings.Contains(out, "\x1b") {
			t.Errorf("%s output must stay plain, got %q", format, out)
		}
	}
}
