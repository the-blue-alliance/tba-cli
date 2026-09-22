package output

import (
	"bytes"
	"strings"
	"testing"
)

func render(t *testing.T, tbl Table, opts RenderOptions) string {
	t.Helper()
	var buf bytes.Buffer
	if err := Render(&buf, tbl, opts); err != nil {
		t.Fatalf("Render: %v", err)
	}
	return buf.String()
}

func TestRenderTableAlignsColumns(t *testing.T) {
	got := render(t, Table{
		Headers: []string{"Number", "Name", "Location"},
		Rows: [][]string{
			{"177", "Bobcat Robotics", "South Windsor, CT"},
			{"1073", "The Force Team", "Hollis, NH"},
		},
	}, RenderOptions{Format: "table"})

	want := "Number  Name             Location         \n" +
		"------  ---------------  -----------------\n" +
		"177     Bobcat Robotics  South Windsor, CT\n" +
		"1073    The Force Team   Hollis, NH       \n"
	if got != want {
		t.Errorf("table =\n%q\nwant\n%q", got, want)
	}
}

func TestRenderTableWidensColumnsToTheWidestCell(t *testing.T) {
	got := lines(render(t, Table{Headers: []string{"A"}, Rows: [][]string{{"aaaaaaa"}}}, RenderOptions{Format: "table"}))
	if got[0] != "A      " {
		t.Errorf("header = %q", got[0])
	}
	if got[1] != "-------" {
		t.Errorf("separator = %q", got[1])
	}
	if got[2] != "aaaaaaa" {
		t.Errorf("row = %q", got[2])
	}
}

func TestRenderTableWithNoRows(t *testing.T) {
	got := render(t, Table{Headers: []string{"Key", "Name"}}, RenderOptions{Format: "table"})
	if got != "Key  Name\n---  ----\n" {
		t.Errorf("table = %q", got)
	}
}

func TestRenderTableToleratesExtraCells(t *testing.T) {
	got := render(t, Table{Headers: []string{"A"}, Rows: [][]string{{"1", "extra"}}}, RenderOptions{Format: "table"})
	if got != "A\n-\n1  extra\n" {
		t.Errorf("table = %q", got)
	}
}

func TestRenderTableAlignsWideRunes(t *testing.T) {
	// "チーム" is three double-width runes, so it draws six cells wide even
	// though it is three runes and nine bytes.
	got := lines(render(t, Table{
		Headers: []string{"Team", "Nickname"},
		Rows: [][]string{
			{"177", "チーム"},
			{"1073", "Bobcats"},
		},
	}, RenderOptions{Format: "table"}))

	for _, line := range got {
		if w := StringWidth(line); w != StringWidth(got[0]) {
			t.Errorf("line %q draws %d cells, want %d", line, w, StringWidth(got[0]))
		}
	}
	if !strings.HasPrefix(got[2], "177   チーム") {
		t.Errorf("wide row = %q", got[2])
	}
}

func TestRenderTableAlignsEmoji(t *testing.T) {
	got := lines(render(t, Table{
		Headers: []string{"Name", "Year"},
		Rows: [][]string{
			{"🤖 Bots", "2024"},
			{"Plain", "2023"},
		},
	}, RenderOptions{Format: "table"}))

	for _, line := range got {
		if w := StringWidth(line); w != StringWidth(got[0]) {
			t.Errorf("line %q draws %d cells, want %d", line, w, StringWidth(got[0]))
		}
	}
	// The emoji is two cells wide, so the padded cell is shorter in runes
	// than the seven-cell column it fills.
	if got[2] != "🤖 Bots  2024" {
		t.Errorf("emoji row = %q", got[2])
	}
}

func TestRenderTableNoHeaders(t *testing.T) {
	got := render(t, Table{
		Headers: []string{"Number", "Name"},
		Rows:    [][]string{{"177", "Bobcats"}, {"1073", "Force"}},
	}, RenderOptions{Format: "table", NoHeaders: true})

	// Neither the header nor its separator is printed, and the columns are
	// sized to the data alone.
	if got != "177   Bobcats\n1073  Force  \n" {
		t.Errorf("table = %q", got)
	}
}

func TestRenderMarkdownEscapesPipes(t *testing.T) {
	got := render(t, Table{
		Headers: []string{"Key | Alt", "Name"},
		Rows: [][]string{
			{"2024cthar", "NE District | Hartford"},
			{"2024necmp", "New\nEngland"},
		},
	}, RenderOptions{Format: "markdown"})

	want := "| Key \\| Alt | Name |\n" +
		"| --- | --- |\n" +
		"| 2024cthar | NE District \\| Hartford |\n" +
		"| 2024necmp | New England |\n"
	if got != want {
		t.Errorf("markdown =\n%q\nwant\n%q", got, want)
	}
}

func TestRenderMarkdownWithNoRows(t *testing.T) {
	got := render(t, Table{Headers: []string{"A", "B"}}, RenderOptions{Format: "markdown"})
	if got != "| A | B |\n| --- | --- |\n" {
		t.Errorf("markdown = %q", got)
	}
}

func TestRenderMarkdownNoHeaders(t *testing.T) {
	got := render(t, Table{
		Headers: []string{"A", "B"},
		Rows:    [][]string{{"1", "2"}},
	}, RenderOptions{Format: "markdown", NoHeaders: true})
	if got != "| 1 | 2 |\n" {
		t.Errorf("markdown = %q", got)
	}
}

func TestRenderAcceptsMdAsMarkdown(t *testing.T) {
	got := render(t, Table{Headers: []string{"A"}}, RenderOptions{Format: "md"})
	if got != "| A |\n| --- |\n" {
		t.Errorf("markdown = %q", got)
	}
}

func TestRenderEmptyFormatIsTable(t *testing.T) {
	got := render(t, Table{Headers: []string{"A"}, Rows: [][]string{{"1"}}}, RenderOptions{})
	if got != "A\n-\n1\n" {
		t.Errorf("table = %q", got)
	}
}

func TestRenderRejectsAnUnknownFormat(t *testing.T) {
	var buf bytes.Buffer
	err := Render(&buf, Table{Headers: []string{"A"}}, RenderOptions{Format: "yaml"})
	if err == nil {
		t.Fatal("want an error for an unknown format")
	}
	if !strings.Contains(err.Error(), `"yaml"`) {
		t.Errorf("error = %v, want it to name the format", err)
	}
	if buf.Len() != 0 {
		t.Errorf("want no output, got %q", buf.String())
	}
}

func lines(s string) []string {
	return strings.Split(strings.TrimRight(s, "\n"), "\n")
}

// A divider is a line between rows — a district championship cut, say — and
// the point of it is that it does not live in a cell, so it cannot widen one.
func TestRenderTableDrawsDividers(t *testing.T) {
	got := render(t, Table{
		Headers:  []string{"Rank", "Team"},
		Rows:     [][]string{{"1", "177"}, {"2", "1073"}},
		Dividers: map[int]string{0: "top 1 by pre-DCMP total"},
	}, RenderOptions{Format: "table"})

	want := "Rank  Team\n" +
		"----  ----\n" +
		"1     177 \n" +
		"--- top 1 by pre-DCMP total ---\n" +
		"2     1073\n"
	if got != want {
		t.Errorf("table =\n%q\nwant\n%q", got, want)
	}
}

func TestRenderMarkdownDrawsDividers(t *testing.T) {
	got := render(t, Table{
		Headers:  []string{"Rank", "Team", "Total"},
		Rows:     [][]string{{"1", "177", "145"}, {"2", "1073", "132"}},
		Dividers: map[int]string{0: "DCMP cutoff (top 1)"},
	}, RenderOptions{Format: "markdown"})

	want := "| Rank | Team | Total |\n" +
		"| --- | --- | --- |\n" +
		"| 1 | 177 | 145 |\n" +
		"| --- DCMP cutoff (top 1) --- | | |\n" +
		"| 2 | 1073 | 132 |\n"
	if got != want {
		t.Errorf("markdown =\n%q\nwant\n%q", got, want)
	}
}

// A divider is presentation, so the machine-readable formats never see one.
func TestRenderDividersAreSkippedByDelimitedFormats(t *testing.T) {
	tbl := Table{
		Headers:  []string{"Rank", "Team"},
		Rows:     [][]string{{"1", "177"}, {"2", "1073"}},
		Dividers: map[int]string{0: "DCMP cutoff (top 1)"},
	}
	for _, format := range []string{"csv", "tsv"} {
		got := render(t, tbl, RenderOptions{Format: format})
		if strings.Contains(got, "cutoff") || strings.Contains(got, "---") {
			t.Errorf("%s carries the divider:\n%s", format, got)
		}
		if n := len(lines(got)); n != 3 {
			t.Errorf("%s has %d lines, want header and two rows:\n%s", format, n, got)
		}
	}
}

// A break is the quieter divider: the rows below are a different kind of thing
// from the rows above, which needs a gap rather than a sentence.
func TestRenderTableDrawsBreaks(t *testing.T) {
	got := render(t, Table{
		Headers: []string{"Stat", "Red"},
		Rows:    [][]string{{"Total Points", "96"}, {"Endgame Robot 1", "Parked"}},
		Breaks:  map[int]bool{0: true},
	}, RenderOptions{Format: "table"})

	want := "Stat             Red   \n" +
		"---------------  ------\n" +
		"Total Points     96    \n" +
		"\n" +
		"Endgame Robot 1  Parked\n"
	if got != want {
		t.Errorf("table =\n%q\nwant\n%q", got, want)
	}
}

// A break after the last row would only add a blank line to whatever comes
// next, and a blank line in a data file is a broken record.
func TestRenderBreaksAreSkippedAtTheEndAndByOtherFormats(t *testing.T) {
	tbl := Table{
		Headers: []string{"Stat", "Red"},
		Rows:    [][]string{{"Total Points", "96"}},
		Breaks:  map[int]bool{0: true, 9: true},
	}
	if got := render(t, tbl, RenderOptions{Format: "table"}); got != "Stat          Red\n------------  ---\nTotal Points  96 \n" {
		t.Errorf("table = %q", got)
	}
	for _, format := range []string{"csv", "tsv", "markdown"} {
		got := render(t, tbl, RenderOptions{Format: format})
		if n := len(lines(got)); n > 3 {
			t.Errorf("%s has %d lines, want no blank one:\n%q", format, n, got)
		}
	}
}

// An empty divider is nothing to draw, and one past the last row has no gap to
// sit in.
func TestRenderIgnoresEmptyAndOutOfRangeDividers(t *testing.T) {
	got := render(t, Table{
		Headers:  []string{"Rank"},
		Rows:     [][]string{{"1"}},
		Dividers: map[int]string{0: "", 5: "nowhere"},
	}, RenderOptions{Format: "table"})
	if got != "Rank\n----\n1   \n" {
		t.Errorf("table = %q", got)
	}
}
