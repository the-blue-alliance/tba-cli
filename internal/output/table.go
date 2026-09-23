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
	// Dividers annotates the gaps between rows: the text at key i is drawn
	// after Rows[i], on a line of its own and between dashed rules. It is presentation — a district
	// championship cut line, say — so only the formats a person reads honour
	// it, and csv, tsv and json ignore it entirely.
	//
	// Keys are indices into Rows as the table was built, so a transform that
	// reorders rows drops them rather than leaving a line stranded in the
	// middle of a differently-sorted table.
	Dividers map[int]string
	// Breaks marks the gaps between rows that are worth a blank line: a break
	// at key i leaves one after Rows[i]. It says the rows below are a
	// different kind of thing from the rows above, which is a weaker claim
	// than a divider's labelled rule and needs no words to make.
	//
	// Like Dividers it is presentation, honoured only by the aligned text
	// table: a blank line in csv or markdown is a broken file, not a pause.
	Breaks map[int]bool
}

// dividerAfter returns the text to draw after row i, if any.
func (t Table) dividerAfter(i int) (string, bool) {
	if t.Dividers == nil {
		return "", false
	}
	text, ok := t.Dividers[i]
	return text, ok && text != ""
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

	for i, row := range t.Rows {
		fmt.Fprintln(w, joinPadded(row, widths))
		if text, ok := t.dividerAfter(i); ok {
			// Drawn whole, on its own line: squeezing it into the first cell
			// would widen that column by the length of the sentence.
			fmt.Fprintf(w, "--- %s ---\n", text)
		}
		// Not after the last row, where it would only add a trailing blank
		// line to whatever follows the table.
		if t.Breaks[i] && i+1 < len(t.Rows) {
			fmt.Fprintln(w)
		}
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
	for i, row := range t.Rows {
		fmt.Fprint(w, "|")
		for _, cell := range row {
			fmt.Fprintf(w, " %s |", escape(cell))
		}
		fmt.Fprintln(w)
		if text, ok := t.dividerAfter(i); ok {
			// Markdown has no row that spans the table, so the text goes in
			// the first cell and the rest are left empty, which reads as one
			// wide rule.
			fmt.Fprintf(w, "| --- %s --- |", escape(text))
			for n := 1; n < len(t.Headers); n++ {
				fmt.Fprint(w, " |")
			}
			fmt.Fprintln(w)
		}
	}
}
