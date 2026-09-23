package output

import (
	"strings"

	"github.com/rivo/uniseg"
)

// StringWidth reports how many terminal cells s occupies. It is grapheme aware,
// so combining marks, emoji sequences and East Asian wide runes are measured the
// way a terminal draws them rather than by byte or rune count.
func StringWidth(s string) int {
	return uniseg.StringWidth(s)
}

// padRight pads s with spaces so that it occupies width cells. Cells wider than
// width are returned unchanged, which keeps a stray oversized value visible
// instead of truncating data.
func padRight(s string, width int) string {
	n := width - StringWidth(s)
	if n <= 0 {
		return s
	}
	return s + strings.Repeat(" ", n)
}
