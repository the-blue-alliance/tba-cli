package output

import (
	"fmt"
	"math"
	"reflect"
	"sort"
	"strconv"
	"strings"
)

// normalizeColumn folds a column reference to its comparable form, so that
// `start_date`, `Start Date` and `start-date` all name the same column.
func normalizeColumn(s string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(s) {
		switch r {
		case ' ', '_', '-':
		default:
			b.WriteRune(r)
		}
	}
	return b.String()
}

// ColumnIndex resolves a user-supplied column reference to a zero-based index.
// A reference matches a header name case-insensitively, ignoring spaces,
// underscores and dashes; failing that, a bare number is taken as a 1-based
// column position.
func (t Table) ColumnIndex(ref string) (int, error) {
	ref = strings.TrimSpace(ref)
	if ref == "" {
		return 0, fmt.Errorf("empty column name")
	}
	want := normalizeColumn(ref)
	for i, h := range t.Headers {
		if normalizeColumn(h) == want {
			return i, nil
		}
	}
	if n, err := strconv.Atoi(ref); err == nil {
		if n >= 1 && n <= len(t.Headers) {
			return n - 1, nil
		}
		return 0, fmt.Errorf("column index %d is out of range (1-%d)", n, len(t.Headers))
	}
	if len(t.Headers) == 0 {
		return 0, fmt.Errorf("unknown column %q: this output has no columns", ref)
	}
	return 0, fmt.Errorf("unknown column %q (valid columns: %s)", ref, strings.Join(t.Headers, ", "))
}

// SelectColumns returns a Table holding only the columns named by spec, a
// comma-separated list, in the order given. A column may be repeated, and a
// short row is padded with empty cells rather than dropped.
func (t Table) SelectColumns(spec string) (Table, error) {
	refs := strings.Split(spec, ",")
	idx := make([]int, 0, len(refs))
	for _, ref := range refs {
		i, err := t.ColumnIndex(ref)
		if err != nil {
			return Table{}, err
		}
		idx = append(idx, i)
	}

	out := Table{Headers: make([]string, len(idx)), Rows: make([][]string, len(t.Rows))}
	for n, i := range idx {
		out.Headers[n] = t.Headers[i]
	}
	for r, row := range t.Rows {
		cells := make([]string, len(idx))
		for n, i := range idx {
			cells[n] = cellAt(row, i)
		}
		out.Rows[r] = cells
	}
	return out, nil
}

// SortOrder returns the permutation of row indices that sorts the table by
// spec, a column reference optionally prefixed with "-" to descend. The sort is
// stable, so rows that compare equal keep the order the API returned them in,
// and it is numeric-aware: two cells that both parse as numbers compare as
// numbers, everything else compares bytewise.
//
// The caller gets indices rather than a sorted Table so that the same
// permutation can be applied to the underlying JSON data.
func (t Table) SortOrder(spec string) ([]int, error) {
	desc := strings.HasPrefix(spec, "-")
	ref := strings.TrimPrefix(spec, "-")
	if strings.TrimSpace(ref) == "" {
		return nil, fmt.Errorf("--sort needs a column name")
	}
	col, err := t.ColumnIndex(ref)
	if err != nil {
		return nil, err
	}

	order := make([]int, len(t.Rows))
	for i := range order {
		order[i] = i
	}
	sort.SliceStable(order, func(a, b int) bool {
		cmp := compareCells(cellAt(t.Rows[order[a]], col), cellAt(t.Rows[order[b]], col))
		if desc {
			return cmp > 0
		}
		return cmp < 0
	})
	return order, nil
}

// Reorder returns a Table whose rows follow order. Indices outside the table
// are skipped, so a stale permutation cannot panic.
func (t Table) Reorder(order []int) Table {
	rows := make([][]string, 0, len(order))
	for _, i := range order {
		if i >= 0 && i < len(t.Rows) {
			rows = append(rows, t.Rows[i])
		}
	}
	return Table{Headers: t.Headers, Rows: rows}
}

// PermuteSlice applies a row permutation to the data behind a table so that
// JSON output matches the order the table would have been printed in. Data that
// is not a slice of the same length is returned untouched: --sort then only
// affects the tabular formats, which is better than reshaping a payload we do
// not understand.
func PermuteSlice(data interface{}, order []int) interface{} {
	v := reflect.ValueOf(data)
	if !v.IsValid() {
		return data
	}
	if v.Kind() != reflect.Slice && v.Kind() != reflect.Array {
		return data
	}
	if v.Len() != len(order) {
		return data
	}
	out := make([]interface{}, 0, len(order))
	for _, i := range order {
		if i < 0 || i >= v.Len() {
			return data
		}
		out = append(out, v.Index(i).Interface())
	}
	return out
}

func cellAt(row []string, i int) string {
	if i < 0 || i >= len(row) {
		return ""
	}
	return row[i]
}

// compareCells orders two cells, preferring a numeric comparison when both
// sides are numbers so that "10" sorts after "9".
func compareCells(a, b string) int {
	if x, err := parseNumber(a); err == nil {
		if y, err := parseNumber(b); err == nil {
			switch {
			case x < y:
				return -1
			case x > y:
				return 1
			default:
				return 0
			}
		}
	}
	return strings.Compare(a, b)
}

func parseNumber(s string) (float64, error) {
	f, err := strconv.ParseFloat(strings.TrimSpace(s), 64)
	if err != nil {
		return 0, err
	}
	if math.IsNaN(f) || math.IsInf(f, 0) {
		return 0, strconv.ErrSyntax
	}
	return f, nil
}
