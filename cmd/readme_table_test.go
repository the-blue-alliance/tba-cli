package cmd

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

// readmePath is the README the command reference lives in, from the cmd
// package's own directory.
func readmePath() string { return filepath.Join("..", "README.md") }

// readmeTableFromFile pulls the command reference out of the README: the
// header line, and every table row under it up to the first line that is not
// one.
func readmeTableFromFile(t *testing.T) string {
	t.Helper()
	b, err := os.ReadFile(readmePath())
	if err != nil {
		t.Fatalf("reading the README: %v", err)
	}
	// A Windows checkout may carry CRLF line endings; the table is compared
	// line by line, so fold them before looking for it.
	body := strings.ReplaceAll(string(b), "\r\n", "\n")
	start := strings.Index(body, readmeTableHeader)
	if start < 0 {
		t.Fatalf("no %q table in the README", strings.SplitN(readmeTableHeader, "\n", 2)[0])
	}
	rest := body[start+len(readmeTableHeader):]
	var out strings.Builder
	out.WriteString(readmeTableHeader)
	for _, line := range strings.Split(rest, "\n") {
		if !strings.HasPrefix(line, "|") {
			break
		}
		out.WriteString(line)
		out.WriteString("\n")
	}
	return out.String()
}

// TestReadmeCommandTableMatchesTheREADME is the contract the generator exists
// for: the table in the README is the table the command tree describes, byte
// for byte. A command added, renamed or re-described without regenerating
// fails here rather than quietly documenting something that is no longer true.
func TestReadmeCommandTableMatchesTheREADME(t *testing.T) {
	want := readmeCommandTable()
	got := readmeTableFromFile(t)
	if got == want {
		return
	}
	t.Errorf("the README's command table is out of date; regenerate it with\n"+
		"    go run ./cmd/tba docs readme-table\n\n%s", tableDiff(got, want))
}

// tableDiff reports the rows that differ between two tables, which is far
// easier to act on than two fifty-line blocks side by side.
func tableDiff(got, want string) string {
	gotRows, wantRows := lines(got), lines(want)
	var b strings.Builder
	for _, row := range gotRows {
		if !contains(wantRows, row) {
			b.WriteString("README only: " + row + "\n")
		}
	}
	for _, row := range wantRows {
		if !contains(gotRows, row) {
			b.WriteString("generated only: " + row + "\n")
		}
	}
	if b.Len() == 0 {
		b.WriteString("the same rows in a different order\n")
	}
	return b.String()
}

func TestDocsReadmeTablePrintsTheTable(t *testing.T) {
	out, _, err := runCmd(t, nil, "docs", "readme-table")
	requireNoError(t, err, "")
	if out != readmeCommandTable() {
		t.Errorf("`docs readme-table` did not print the table it generates:\n%s", out)
	}
	// Reproducible like every other generator: the clock never gets in.
	second, _, err := runCmd(t, nil, "docs", "readme-table")
	requireNoError(t, err, "")
	if second != out {
		t.Error("two generations of the table differ")
	}
}

// readmeCommands is the Command column of every generated row.
func readmeCommands(t *testing.T) []string {
	t.Helper()
	rows := readmeCommandRows(NewRootCmd())
	out := make([]string, len(rows))
	for i, row := range rows {
		out[i] = row.command
	}
	return out
}

func TestReadmeCommandRowsListLeavesOnly(t *testing.T) {
	got := readmeCommands(t)
	for _, want := range []string{"tba team view <number>", "tba event matches <key>", "tba version", "tba status"} {
		if !contains(got, want) {
			t.Errorf("the table is missing %q", want)
		}
	}
	// A group is a heading, not something to run.
	for _, unwanted := range []string{"tba team", "tba event", "tba auth", "tba config"} {
		if contains(got, unwanted) {
			t.Errorf("the table lists the group %q as a command", unwanted)
		}
	}
}

func TestReadmeCommandRowsSkipHiddenCommands(t *testing.T) {
	for _, command := range readmeCommands(t) {
		if strings.HasPrefix(command, "tba docs") {
			t.Errorf("the hidden docs generator is listed: %q", command)
		}
		if strings.HasPrefix(command, "tba help") {
			t.Errorf("cobra's help command is listed: %q", command)
		}
	}
}

// TestReadmeCommandRowsListTheCompletionCommand guards the one command cobra
// only attaches on its way into Execute: a tree built for inspection has no
// `completion` unless the generator asks for it.
func TestReadmeCommandRowsListTheCompletionCommand(t *testing.T) {
	got := readmeCommands(t)
	for _, want := range []string{"tba completion bash", "tba completion zsh", "tba completion fish"} {
		if !contains(got, want) {
			t.Errorf("the table is missing %q; got %v", want, got)
		}
	}
}

func TestReadmeCommandRowsAreSorted(t *testing.T) {
	got := readmeCommands(t)
	if !slices.IsSorted(got) {
		t.Errorf("the table is not in alphabetical order: %v", got)
	}
}

// TestReadmeCommandTableEscapesPipes: a bare pipe would end the markdown cell,
// even inside a code span, and split the row into more columns than the table
// has.
func TestReadmeCommandTableEscapesPipes(t *testing.T) {
	table := readmeCommandTable()
	// Three: the one that opens the row, the one between the columns, and the
	// one that closes it. Anything else is a pipe in a cell that got out.
	for _, row := range lines(table) {
		if n := strings.Count(row, "|") - strings.Count(row, `\|`); n != 3 {
			t.Errorf("row has %d unescaped pipes, want 3: %q", n, row)
		}
	}
	requireContains(t, table, `\|`)
}

func TestReadmeUseLineDropsTheFlagsPlaceholder(t *testing.T) {
	c := &cobra.Command{Use: "view <number>"}
	parent := &cobra.Command{Use: "team"}
	root := &cobra.Command{Use: "tba"}
	root.AddCommand(parent)
	parent.AddCommand(c)

	if got, want := readmeUseLine(c), "tba team view <number>"; got != want {
		t.Errorf("readmeUseLine = %q, want %q", got, want)
	}
	if got, want := readmeUseLine(parent), "tba team"; got != want {
		t.Errorf("readmeUseLine = %q, want %q", got, want)
	}
}

func TestMarkdownCellEscapesPipes(t *testing.T) {
	if got, want := markdownCell("csv|tsv|json"), `csv\|tsv\|json`; got != want {
		t.Errorf("markdownCell = %q, want %q", got, want)
	}
	if got, want := markdownCell("plain"), "plain"; got != want {
		t.Errorf("markdownCell = %q, want %q", got, want)
	}
}

func TestDocsReadmeTableIsUnderTheHiddenGenerators(t *testing.T) {
	out, _, err := runCmd(t, nil, "docs", "--help")
	requireNoError(t, err, "")
	requireContains(t, out, "readme-table")

	if !contains(subcommandNames(t, "docs"), "readme-table") {
		t.Error("docs has no readme-table subcommand")
	}
}
