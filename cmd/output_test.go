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

func TestColorAlwaysDoesNotTouchDataFormats(t *testing.T) {
	for _, format := range []string{"csv", "tsv", "markdown", "json"} {
		out := districtsCmd(t, "--format", format, "--color", "always")
		if strings.Contains(out, "\x1b") {
			t.Errorf("%s output must stay plain, got %q", format, out)
		}
	}
}
