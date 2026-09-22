package output

import (
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"strings"
)

// Unwrapper is implemented by writers that decorate another writer, such as
// the one that records write failures around stdout. IsTTY looks through
// them so a wrapped terminal is still recognised as a terminal.
type Unwrapper interface {
	Unwrap() io.Writer
}

// TerminalReporter lets a writer state outright whether it is a terminal.
// Tests use it to exercise the terminal code paths without a real one.
type TerminalReporter interface {
	IsTerminal() bool
}

// IsTTY reports whether w is a terminal: an *os.File backed by a character
// device, possibly behind one or more wrappers, or anything that says so via
// TerminalReporter.
func IsTTY(w io.Writer) bool {
	for w != nil {
		switch v := w.(type) {
		case TerminalReporter:
			return v.IsTerminal()
		case *os.File:
			fi, err := v.Stat()
			if err != nil {
				return false
			}
			return fi.Mode()&os.ModeCharDevice != 0
		case Unwrapper:
			w = v.Unwrap()
		default:
			return false
		}
	}
	return false
}

// PrintKeyValue writes alternating key/value pairs with the values aligned.
// The label column is sized to the longest "key:" including the colon, so
// every value starts in the same column.
func PrintKeyValue(w io.Writer, pairs ...string) {
	width := 0
	for i := 0; i < len(pairs)-1; i += 2 {
		if n := len(pairs[i]) + 1; n > width {
			width = n
		}
	}
	for i := 0; i < len(pairs)-1; i += 2 {
		fmt.Fprintf(w, "%-*s  %s\n", width, pairs[i]+":", pairs[i+1])
	}
}

// PrintTable writes an aligned text table with a dashed separator row.
func PrintTable(w io.Writer, headers []string, rows [][]string) {
	// Calculate column widths
	widths := make([]int, len(headers))
	for i, h := range headers {
		widths[i] = len(h)
	}
	for _, row := range rows {
		for i, cell := range row {
			if i < len(widths) && len(cell) > widths[i] {
				widths[i] = len(cell)
			}
		}
	}

	// Print header
	for i, h := range headers {
		if i > 0 {
			fmt.Fprint(w, "  ")
		}
		fmt.Fprintf(w, "%-*s", widths[i], h)
	}
	fmt.Fprintln(w)

	// Print separator
	for i, width := range widths {
		if i > 0 {
			fmt.Fprint(w, "  ")
		}
		fmt.Fprint(w, strings.Repeat("-", width))
	}
	fmt.Fprintln(w)

	// Print rows
	for _, row := range rows {
		for i, cell := range row {
			if i > 0 {
				fmt.Fprint(w, "  ")
			}
			if i < len(widths) {
				fmt.Fprintf(w, "%-*s", widths[i], cell)
			} else {
				fmt.Fprint(w, cell)
			}
		}
		fmt.Fprintln(w)
	}
}

// PrintDelimited writes headers and rows as delimiter-separated values.
func PrintDelimited(w io.Writer, headers []string, rows [][]string, delim rune) error {
	cw := csv.NewWriter(w)
	cw.Comma = delim
	if err := cw.Write(headers); err != nil {
		return err
	}
	if err := cw.WriteAll(rows); err != nil {
		return err
	}
	cw.Flush()
	return cw.Error()
}

// PrintMarkdownTable writes a GitHub-flavored markdown table.
func PrintMarkdownTable(w io.Writer, headers []string, rows [][]string) {
	escape := func(s string) string {
		return strings.ReplaceAll(strings.ReplaceAll(s, "|", "\\|"), "\n", " ")
	}
	fmt.Fprint(w, "|")
	for _, h := range headers {
		fmt.Fprintf(w, " %s |", escape(h))
	}
	fmt.Fprintln(w)
	fmt.Fprint(w, "|")
	for range headers {
		fmt.Fprint(w, " --- |")
	}
	fmt.Fprintln(w)
	for _, row := range rows {
		fmt.Fprint(w, "|")
		for _, cell := range row {
			fmt.Fprintf(w, " %s |", escape(cell))
		}
		fmt.Fprintln(w)
	}
}

// TeamNumberFromKey turns "frc177" into "177".
func TeamNumberFromKey(key string) string {
	return strings.TrimPrefix(key, "frc")
}

// FormatLocation joins the non-empty location parts with ", ".
func FormatLocation(city, stateProv, country string) string {
	parts := []string{}
	if city != "" {
		parts = append(parts, city)
	}
	if stateProv != "" {
		parts = append(parts, stateProv)
	}
	if country != "" {
		parts = append(parts, country)
	}
	return strings.Join(parts, ", ")
}
