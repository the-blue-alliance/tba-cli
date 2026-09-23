package output

import (
	"fmt"
	"io"
	"os"
)

// ColorMode is the user's stated preference for ANSI escapes, before it is
// resolved against the writer and the environment.
type ColorMode int

const (
	// ColorAuto colors only an interactive terminal that has not opted out.
	ColorAuto ColorMode = iota
	// ColorAlways colors unconditionally, e.g. when piping into a pager.
	ColorAlways
	// ColorNever never colors.
	ColorNever
)

// String renders the mode the way it is spelled on the command line.
func (m ColorMode) String() string {
	switch m {
	case ColorAlways:
		return "always"
	case ColorNever:
		return "never"
	default:
		return "auto"
	}
}

// ParseColorMode turns a --color value into a ColorMode. The empty string means
// auto, so an unset flag needs no special casing.
func ParseColorMode(s string) (ColorMode, error) {
	switch s {
	case "", "auto":
		return ColorAuto, nil
	case "always":
		return ColorAlways, nil
	case "never":
		return ColorNever, nil
	default:
		return ColorAuto, fmt.Errorf("invalid --color %q (want: auto, always, never)", s)
	}
}

// ANSI SGR parameters for the styles this CLI uses. They are hand-rolled rather
// than pulled from a color library so the binary keeps no dependency for four
// escape sequences.
const (
	Bold = "1"
	Dim  = "2"
	Red  = "31"
	Blue = "34"
)

// Colorize wraps s in an ANSI SGR sequence when on is true, and returns it
// untouched otherwise, so callers can stay free of conditionals.
func Colorize(s, code string, on bool) string {
	if !on || code == "" || s == "" {
		return s
	}
	return "\x1b[" + code + "m" + s + "\x1b[0m"
}

// ColorEnabledFor resolves mode for a specific writer.
func ColorEnabledFor(w io.Writer, mode ColorMode) bool {
	return ColorEnabled(mode, IsTTY(w))
}

// ColorEnabled resolves mode against the environment and whether the
// destination is a terminal.
//
// never and always are absolute. In auto mode the precedence is:
//
//   - NO_COLOR set to any non-empty value disables color (no-color.org).
//   - TERM=dumb disables color; such a terminal cannot render the escapes.
//   - CLICOLOR_FORCE set to anything but "0" or "" enables color even when the
//     destination is not a terminal.
//   - Otherwise, color only an interactive terminal.
func ColorEnabled(mode ColorMode, isTTY bool) bool {
	switch mode {
	case ColorNever:
		return false
	case ColorAlways:
		return true
	}
	if os.Getenv("NO_COLOR") != "" {
		return false
	}
	if os.Getenv("TERM") == "dumb" {
		return false
	}
	if force := os.Getenv("CLICOLOR_FORCE"); force != "" && force != "0" {
		return true
	}
	return isTTY
}
