package output

import "testing"

func TestStringWidthCountsTerminalCells(t *testing.T) {
	cases := []struct {
		in   string
		want int
	}{
		{"", 0},
		{"177", 3},
		{"チーム", 6},    // three East Asian wide runes
		{"🤖 Bots", 7}, // a two-cell emoji, a space and four letters
		{"é", 1},      // precomposed
		{"é", 1},     // e plus a combining acute: one grapheme
		{"👨‍👩‍👧", 2},  // a ZWJ family sequence draws as one emoji
		{"Bobcat Robotics", 15},
	}
	for _, c := range cases {
		if got := StringWidth(c.in); got != c.want {
			t.Errorf("StringWidth(%q) = %d, want %d", c.in, got, c.want)
		}
	}
}

func TestPadRight(t *testing.T) {
	cases := []struct {
		in    string
		width int
		want  string
	}{
		{"ab", 4, "ab  "},
		{"チーム", 8, "チーム  "},
		{"🤖", 4, "🤖  "},
		{"toolong", 3, "toolong"},
		{"", 2, "  "},
	}
	for _, c := range cases {
		if got := padRight(c.in, c.width); got != c.want {
			t.Errorf("padRight(%q, %d) = %q, want %q", c.in, c.width, got, c.want)
		}
	}
}
