package output

import (
	"fmt"
	"io"
	"strings"
)

// Table is a render-format-independent tabular result: a header row and zero or
// more data rows. It is a plain value type, so callers can copy it freely; the
// transforms in this package return a new Table instead of mutating the
// receiver.
type Table struct {
	Headers []string
	Rows    [][]string
}

// RenderOptions controls how a Table is written.
type RenderOptions struct {
	// Format is one of table, csv, tsv, markdown.
	Format string
	// NoHeaders omits the header row (and, for markdown, its separator).
	NoHeaders bool
	// Color says whether the renderer may emit ANSI escapes. It is resolved
	// against the writer and the environment at render time.
	Color ColorMode
}

// Render writes t to w in the requested format.
func Render(w io.Writer, t Table, opts RenderOptions) error {
	switch opts.Format {
	case "table", "":
		renderText(w, t, opts.NoHeaders, ColorEnabledFor(w, opts.Color))
		return nil
	case "csv":
		return renderCSV(w, t, opts.NoHeaders)
	case "tsv":
		return renderTSV(w, t, opts.NoHeaders)
	case "markdown", "md":
		renderMarkdown(w, t, opts.NoHeaders)
		return nil
	default:
		return fmt.Errorf("unknown output format %q (want: table, csv, tsv, markdown)", opts.Format)
	}
}

// columnWidths measures every visible cell so that columns line up in a
// terminal. Widths are grapheme-cluster aware, so a CJK or emoji cell reserves
// the space it actually draws in.
func columnWidths(t Table, noHeaders bool) []int {
	n := len(t.Headers)
	if noHeaders {
		n = 0
		for _, row := range t.Rows {
			if len(row) > n {
				n = len(row)
			}
		}
	}
	widths := make([]int, n)
	if !noHeaders {
		for i, h := range t.Headers {
			widths[i] = StringWidth(h)
		}
	}
	for _, row := range t.Rows {
		for i, cell := range row {
			if i < len(widths) {
				if n := StringWidth(cell); n > widths[i] {
					widths[i] = n
				}
			}
		}
	}
	return widths
}

// renderText writes an aligned text table with a dashed separator under the
// headers. Cells past the last known column are printed unpadded rather than
// dropped, so a ragged row never loses data.
func renderText(w io.Writer, t Table, noHeaders, color bool) {
	widths := columnWidths(t, noHeaders)

	if !noHeaders {
		fmt.Fprintln(w, Colorize(joinPadded(t.Headers, widths), Bold, color))

		seps := make([]string, len(widths))
		for i, width := range widths {
			seps[i] = strings.Repeat("-", width)
		}
		fmt.Fprintln(w, strings.Join(seps, "  "))
	}

	for _, row := range t.Rows {
		fmt.Fprintln(w, joinPadded(row, widths))
	}
}

func joinPadded(cells []string, widths []int) string {
	var b strings.Builder
	for i, cell := range cells {
		if i > 0 {
			b.WriteString("  ")
		}
		if i < len(widths) {
			b.WriteString(padRight(cell, widths[i]))
		} else {
			b.WriteString(cell)
		}
	}
	return b.String()
}

// renderMarkdown writes a GitHub-flavored markdown table. Pipes are escaped and
// newlines flattened so a cell can never break out of its row.
func renderMarkdown(w io.Writer, t Table, noHeaders bool) {
	escape := func(s string) string {
		return strings.ReplaceAll(strings.ReplaceAll(s, "|", "\\|"), "\n", " ")
	}
	if !noHeaders {
		fmt.Fprint(w, "|")
		for _, h := range t.Headers {
			fmt.Fprintf(w, " %s |", escape(h))
		}
		fmt.Fprintln(w)
		fmt.Fprint(w, "|")
		for range t.Headers {
			fmt.Fprint(w, " --- |")
		}
		fmt.Fprintln(w)
	}
	for _, row := range t.Rows {
		fmt.Fprint(w, "|")
		for _, cell := range row {
			fmt.Fprintf(w, " %s |", escape(cell))
		}
		fmt.Fprintln(w)
	}
}
