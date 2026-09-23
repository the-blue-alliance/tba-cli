package output

import "strings"

// StripANSI removes ANSI escape sequences from s, leaving the text a terminal
// would actually draw.
//
// It recognises the two forms this CLI and the tools it pipes through emit:
// CSI sequences (ESC [ ... final byte), which is what SGR color is, and OSC
// sequences (ESC ] ... BEL or ESC \), which is how a terminal hyperlink is
// written. Anything else beginning with ESC drops the ESC and the byte after
// it, so a stray escape cannot be counted as printable.
func StripANSI(s string) string {
	if !strings.ContainsRune(s, 0x1b) {
		return s
	}
	var b strings.Builder
	b.Grow(len(s))
	for i := 0; i < len(s); {
		if s[i] != 0x1b {
			b.WriteByte(s[i])
			i++
			continue
		}
		i = skipEscape(s, i)
	}
	return b.String()
}

// skipEscape returns the index just past the escape sequence starting at i,
// which s[i] is known to begin with ESC.
func skipEscape(s string, i int) int {
	i++ // the ESC itself
	if i >= len(s) {
		return i
	}
	switch s[i] {
	case '[':
		// CSI: parameter and intermediate bytes, then a final byte in 0x40-0x7e.
		i++
		for i < len(s) && s[i] >= 0x20 && s[i] <= 0x3f {
			i++
		}
		if i < len(s) {
			i++
		}
		return i
	case ']':
		// OSC: runs until BEL or the ST sequence ESC \.
		i++
		for i < len(s) {
			if s[i] == 0x07 {
				return i + 1
			}
			if s[i] == 0x1b && i+1 < len(s) && s[i+1] == '\\' {
				return i + 2
			}
			i++
		}
		return i
	default:
		return i + 1
	}
}
