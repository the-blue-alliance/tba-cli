package output

import (
	"strings"
	"testing"
)

func TestStripANSI(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{"plain text is untouched", "177, 1073, 5507", "177, 1073, 5507"},
		{"empty", "", ""},
		{"a color sequence", "\x1b[31mred\x1b[0m", "red"},
		{"bold and dim", "\x1b[1mhead\x1b[0m \x1b[2mnote\x1b[0m", "head note"},
		{"a multi-parameter SGR", "\x1b[1;31;47mx\x1b[0m", "x"},
		{"a terminal hyperlink", "\x1b]8;;https://thebluealliance.com\x07TBA\x1b]8;;\x07", "TBA"},
		{"a hyperlink terminated by ST", "\x1b]8;;http://x\x1b\\TBA\x1b]8;;\x1b\\", "TBA"},
		{"a truncated escape", "abc\x1b", "abc"},
		{"a truncated CSI", "abc\x1b[3", "abc"},
		{"a lone escape", "a\x1bZb", "ab"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := StripANSI(c.in); got != c.want {
				t.Errorf("StripANSI(%q) = %q, want %q", c.in, got, c.want)
			}
		})
	}
}

// Color must not change how wide a cell is, or every column after a colored one
// would be pushed out of line.
func TestStringWidthIgnoresColor(t *testing.T) {
	plain := "177, 1073, 5507"
	colored := Colorize(plain, Red, true)
	if colored == plain {
		t.Fatal("the fixture is not actually colored")
	}
	if got, want := StringWidth(colored), StringWidth(plain); got != want {
		t.Errorf("StringWidth(colored) = %d, want %d", got, want)
	}
}

func TestPadRightPadsAColoredCellToItsVisibleWidth(t *testing.T) {
	got := padRight(Colorize("red", Red, true), 6)
	if want := "\x1b[31mred\x1b[0m   "; got != want {
		t.Errorf("padRight = %q, want %q", got, want)
	}
}

// A table whose cells are colored draws exactly like the same table without
// color, once the escapes are taken back out.
func TestRenderTextAlignsColoredCells(t *testing.T) {
	plain := Table{
		Headers: []string{"Red", "Blue", "Status"},
		Rows:    [][]string{{"177", "230", "Played"}, {"1073", "195", "Scheduled"}},
	}
	colored := Table{
		Headers: plain.Headers,
		Rows: [][]string{
			{Colorize("177", Red, true), Colorize("230", Blue, true), "Played"},
			{Colorize("1073", Red, true), Colorize("195", Blue, true), "Scheduled"},
		},
	}

	render := func(tbl Table) string {
		var b strings.Builder
		if err := Render(&b, tbl, RenderOptions{Format: "table", Color: ColorNever}); err != nil {
			t.Fatalf("Render: %v", err)
		}
		return b.String()
	}
	if got, want := StripANSI(render(colored)), render(plain); got != want {
		t.Errorf("colored table draws as\n%q\nwant\n%q", got, want)
	}
}
