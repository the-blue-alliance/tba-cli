package output

import (
	"encoding/csv"
	"fmt"
	"io"
	"strings"
)

// tsvSanitizer flattens the characters that would otherwise be structural in a
// TSV stream. "\r\n" collapses to one space so a CRLF cell does not leave a
// double gap; the remaining separators map one-for-one.
var tsvSanitizer = strings.NewReplacer("\r\n", " ", "\t", " ", "\r", " ", "\n", " ")

// renderCSV writes RFC 4180 comma-separated values: fields containing a comma,
// a quote or a newline are quoted and embedded quotes doubled.
func renderCSV(w io.Writer, t Table, noHeaders bool) error {
	cw := csv.NewWriter(w)
	if !noHeaders {
		if err := cw.Write(t.Headers); err != nil {
			return err
		}
	}
	if err := cw.WriteAll(t.Rows); err != nil {
		return err
	}
	cw.Flush()
	return cw.Error()
}

// renderTSV writes tab-separated values that are never quoted. A TSV consumer
// splits on tabs and newlines, so those characters (and carriage returns) are
// replaced by a single space inside every cell instead.
func renderTSV(w io.Writer, t Table, noHeaders bool) error {
	write := func(cells []string) error {
		clean := make([]string, len(cells))
		for i, cell := range cells {
			clean[i] = tsvSanitizer.Replace(cell)
		}
		_, err := fmt.Fprintln(w, strings.Join(clean, "\t"))
		return err
	}
	if !noHeaders {
		if err := write(t.Headers); err != nil {
			return err
		}
	}
	for _, row := range t.Rows {
		if err := write(row); err != nil {
			return err
		}
	}
	return nil
}
