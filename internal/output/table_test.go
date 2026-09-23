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
