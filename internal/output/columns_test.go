package output

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"
)

func eventsTable() Table {
	return Table{
		Headers: []string{"Key", "Name", "Start Date"},
		Rows: [][]string{
			{"2024cthar", "Hartford", "2024-03-22"},
			{"2024necmp", "New England", "2024-04-10"},
			{"2024ctwat", "Waterbury", "2024-03-08"},
		},
	}
}

func TestColumnIndexMatchesNamesLoosely(t *testing.T) {
	tbl := eventsTable()
	for _, ref := range []string{"Start Date", "start date", "start_date", "start-date", "STARTDATE", "  Start Date  "} {
		got, err := tbl.ColumnIndex(ref)
		if err != nil {
			t.Errorf("ColumnIndex(%q): %v", ref, err)
			continue
		}
		if got != 2 {
			t.Errorf("ColumnIndex(%q) = %d, want 2", ref, got)
		}
	}
}

func TestColumnIndexAcceptsAOneBasedIndex(t *testing.T) {
	tbl := eventsTable()
	for ref, want := range map[string]int{"1": 0, "2": 1, "3": 2} {
		got, err := tbl.ColumnIndex(ref)
		if err != nil {
			t.Errorf("ColumnIndex(%q): %v", ref, err)
			continue
		}
		if got != want {
			t.Errorf("ColumnIndex(%q) = %d, want %d", ref, got, want)
		}
	}
}

func TestColumnIndexPrefersANameOverAnIndex(t *testing.T) {
	tbl := Table{Headers: []string{"Name", "2"}}
	got, err := tbl.ColumnIndex("2")
	if err != nil {
		t.Fatalf("ColumnIndex: %v", err)
	}
	if got != 1 {
		t.Errorf("ColumnIndex(\"2\") = %d, want the column literally named \"2\"", got)
	}
}

func TestColumnIndexRejectsUnknownNames(t *testing.T) {
	_, err := eventsTable().ColumnIndex("Nickname")
	if err == nil {
		t.Fatal("want an error for an unknown column")
	}
	for _, want := range []string{"Nickname", "Key, Name, Start Date"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error %v, want it to mention %q", err, want)
		}
	}
}

func TestColumnIndexRejectsAnOutOfRangeIndex(t *testing.T) {
	_, err := eventsTable().ColumnIndex("9")
	if err == nil || !strings.Contains(err.Error(), "out of range") {
		t.Errorf("error = %v, want an out-of-range complaint", err)
	}
}

func TestColumnIndexRejectsAnEmptyReference(t *testing.T) {
	if _, err := eventsTable().ColumnIndex("  "); err == nil {
		t.Error("want an error for an empty column name")
	}
}

func TestSelectColumnsReorders(t *testing.T) {
	got, err := eventsTable().SelectColumns("start_date,key")
	if err != nil {
		t.Fatalf("SelectColumns: %v", err)
	}
	want := Table{
		Headers: []string{"Start Date", "Key"},
		Rows: [][]string{
			{"2024-03-22", "2024cthar"},
			{"2024-04-10", "2024necmp"},
			{"2024-03-08", "2024ctwat"},
		},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("SelectColumns =\n%#v\nwant\n%#v", got, want)
	}
}

func TestSelectColumnsPadsShortRows(t *testing.T) {
	tbl := Table{Headers: []string{"A", "B", "C"}, Rows: [][]string{{"1"}}}
	got, err := tbl.SelectColumns("C,A")
	if err != nil {
		t.Fatalf("SelectColumns: %v", err)
	}
	if !reflect.DeepEqual(got.Rows, [][]string{{"", "1"}}) {
		t.Errorf("rows = %#v", got.Rows)
	}
}

func TestSelectColumnsPropagatesAnUnknownColumn(t *testing.T) {
	if _, err := eventsTable().SelectColumns("Key,Nope"); err == nil {
		t.Error("want an error naming the unknown column")
	}
}

func TestSortOrderIsStable(t *testing.T) {
	tbl := Table{
		Headers: []string{"Team", "Rank"},
		Rows: [][]string{
			{"a", "1"},
			{"b", "1"},
			{"c", "1"},
			{"d", "0"},
		},
	}
	order, err := tbl.SortOrder("Rank")
	if err != nil {
		t.Fatalf("SortOrder: %v", err)
	}
	if !reflect.DeepEqual(order, []int{3, 0, 1, 2}) {
		t.Errorf("order = %v, want equal rows to keep their input order", order)
	}
}

func TestSortOrderIsNumericAware(t *testing.T) {
	tbl := Table{
		Headers: []string{"Team"},
		Rows:    [][]string{{"1073"}, {"9"}, {"177"}, {"2.5"}},
	}
	order, err := tbl.SortOrder("Team")
	if err != nil {
		t.Fatalf("SortOrder: %v", err)
	}
	got := tbl.Reorder(order)
	want := [][]string{{"2.5"}, {"9"}, {"177"}, {"1073"}}
	if !reflect.DeepEqual(got.Rows, want) {
		t.Errorf("rows = %#v, want numeric order %#v", got.Rows, want)
	}
}

func TestSortOrderFallsBackToStringCompare(t *testing.T) {
	tbl := Table{
		Headers: []string{"Name"},
		Rows:    [][]string{{"Waterbury"}, {"10 Ton"}, {"Hartford"}, {""}},
	}
	got := tbl.Reorder(mustOrder(t, tbl, "Name"))
	want := [][]string{{""}, {"10 Ton"}, {"Hartford"}, {"Waterbury"}}
	if !reflect.DeepEqual(got.Rows, want) {
		t.Errorf("rows = %#v, want %#v", got.Rows, want)
	}
}

func TestSortOrderMixesNumbersAndText(t *testing.T) {
	// One side is not a number, so the pair compares as text rather than
	// silently treating the word as zero.
	tbl := Table{Headers: []string{"Score"}, Rows: [][]string{{"10"}, {"n/a"}, {"9"}}}
	got := tbl.Reorder(mustOrder(t, tbl, "Score"))
	want := [][]string{{"9"}, {"10"}, {"n/a"}}
	if !reflect.DeepEqual(got.Rows, want) {
		t.Errorf("rows = %#v, want %#v", got.Rows, want)
	}
}

func TestSortOrderDescends(t *testing.T) {
	tbl := Table{Headers: []string{"Team"}, Rows: [][]string{{"177"}, {"9"}, {"1073"}}}
	got := tbl.Reorder(mustOrder(t, tbl, "-Team"))
	want := [][]string{{"1073"}, {"177"}, {"9"}}
	if !reflect.DeepEqual(got.Rows, want) {
		t.Errorf("rows = %#v, want %#v", got.Rows, want)
	}
}

func TestSortOrderDescendingIsAlsoStable(t *testing.T) {
	tbl := Table{
		Headers: []string{"Team", "Rank"},
		Rows:    [][]string{{"a", "1"}, {"b", "1"}, {"c", "2"}},
	}
	if got := mustOrder(t, tbl, "-Rank"); !reflect.DeepEqual(got, []int{2, 0, 1}) {
		t.Errorf("order = %v, want equal rows to keep their input order", got)
	}
}

func TestSortOrderByIndex(t *testing.T) {
	tbl := eventsTable()
	got := tbl.Reorder(mustOrder(t, tbl, "3"))
	if got.Rows[0][0] != "2024ctwat" {
		t.Errorf("first row = %v, want the earliest start date", got.Rows[0])
	}
}

func TestSortOrderToleratesShortRows(t *testing.T) {
	tbl := Table{Headers: []string{"A", "B"}, Rows: [][]string{{"1"}, {"1", "0"}}}
	if got := mustOrder(t, tbl, "B"); !reflect.DeepEqual(got, []int{0, 1}) {
		t.Errorf("order = %v, want the missing cell to sort as empty", got)
	}
}

func TestSortOrderRejectsAnUnknownColumn(t *testing.T) {
	if _, err := eventsTable().SortOrder("Nope"); err == nil {
		t.Error("want an error for an unknown sort column")
	}
	if _, err := eventsTable().SortOrder("-"); err == nil {
		t.Error("want an error for a bare -")
	}
}

func TestReorderIgnoresOutOfRangeIndices(t *testing.T) {
	tbl := Table{Headers: []string{"A"}, Rows: [][]string{{"1"}, {"2"}}}
	got := tbl.Reorder([]int{1, 7, -1, 0})
	if !reflect.DeepEqual(got.Rows, [][]string{{"2"}, {"1"}}) {
		t.Errorf("rows = %#v", got.Rows)
	}
	if !reflect.DeepEqual(got.Headers, tbl.Headers) {
		t.Errorf("headers = %#v, want them carried over", got.Headers)
	}
}

func TestPermuteSlice(t *testing.T) {
	data := []string{"a", "b", "c"}
	got := PermuteSlice(data, []int{2, 0, 1})
	if !reflect.DeepEqual(got, []interface{}{"c", "a", "b"}) {
		t.Errorf("PermuteSlice = %#v", got)
	}
}

func TestPermuteSliceLeavesOtherShapesAlone(t *testing.T) {
	obj := map[string]int{"a": 1}
	if got := PermuteSlice(obj, []int{0}); !reflect.DeepEqual(got, obj) {
		t.Errorf("a map must come back untouched, got %#v", got)
	}
	mismatched := []string{"a", "b"}
	if got := PermuteSlice(mismatched, []int{0}); !reflect.DeepEqual(got, mismatched) {
		t.Errorf("a slice of a different length must come back untouched, got %#v", got)
	}
	if got := PermuteSlice(nil, []int{0}); got != nil {
		t.Errorf("nil must come back untouched, got %#v", got)
	}
	// No permutation at all must not turn a nil slice into an empty one,
	// which would print [] where the API said null.
	var empty []string
	if got := PermuteSlice(empty, nil); !reflect.DeepEqual(got, empty) {
		t.Errorf("an empty permutation must be a no-op, got %#v", got)
	}
}

func TestCanPermuteAnswersWhetherAnOrderApplies(t *testing.T) {
	cases := []struct {
		name  string
		data  interface{}
		order []int
		want  bool
	}{
		{"matching slice", []string{"a", "b"}, []int{1, 0}, true},
		{"empty slice, empty order", []string{}, []int{}, true},
		{"map", map[string]int{"a": 1}, []int{0}, false},
		{"struct", struct{ A int }{1}, []int{0}, false},
		{"length mismatch", []string{"a", "b"}, []int{0}, false},
		{"raw json", json.RawMessage(`[1,2]`), []int{1, 0}, false},
		{"nil", nil, []int{0}, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := CanPermute(tc.data, tc.order); got != tc.want {
				t.Errorf("CanPermute(%#v, %v) = %v, want %v", tc.data, tc.order, got, tc.want)
			}
		})
	}
}

func mustOrder(t *testing.T, tbl Table, spec string) []int {
	t.Helper()
	order, err := tbl.SortOrder(spec)
	if err != nil {
		t.Fatalf("SortOrder(%q): %v", spec, err)
	}
	return order
}

func TestPermuteSliceLeavesRawJSONAlone(t *testing.T) {
	raw := json.RawMessage(`[1,2]`)
	got := PermuteSlice(raw, []int{1, 0, 2, 3, 4})
	if b, ok := got.(json.RawMessage); !ok || string(b) != `[1,2]` {
		t.Fatalf("PermuteSlice reordered raw JSON bytes: %#v", got)
	}
}

// Column selection leaves the rows where they are, so a divider still marks
// the same gap afterwards.
func TestSelectColumnsKeepsDividers(t *testing.T) {
	tbl := Table{
		Headers:  []string{"Rank", "Team", "Total"},
		Rows:     [][]string{{"1", "177", "145"}, {"2", "1073", "132"}},
		Dividers: map[int]string{0: "cut"},
	}
	got, err := tbl.SelectColumns("team")
	if err != nil {
		t.Fatalf("SelectColumns: %v", err)
	}
	if got.Dividers[0] != "cut" {
		t.Errorf("dividers = %v, want the cut to survive", got.Dividers)
	}
}

// A re-sorted table has no honest place for a line that marked a gap in the
// original order.
func TestReorderDropsDividers(t *testing.T) {
	tbl := Table{
		Headers:  []string{"Rank", "Team"},
		Rows:     [][]string{{"1", "177"}, {"2", "1073"}},
		Dividers: map[int]string{0: "cut"},
	}
	if got := tbl.Reorder([]int{1, 0}); got.Dividers != nil {
		t.Errorf("dividers = %v, want none after a reorder", got.Dividers)
	}
}
