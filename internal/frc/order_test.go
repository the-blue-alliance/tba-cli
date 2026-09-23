package frc

import (
	"slices"
	"strings"
	"testing"

	"github.com/the-blue-alliance/tba-cli/internal/api"
)

// sign reduces a comparator's answer to the part that matters, so a test can
// say "a before b" without pinning how far before.
func sign(n int) int {
	switch {
	case n < 0:
		return -1
	case n > 0:
		return 1
	default:
		return 0
	}
}

// requireTotalOrder checks the three properties a sort relies on: a comparator
// that is not antisymmetric silently gives a different answer depending on the
// order it was handed the data in.
func requireTotalOrder[T any](t *testing.T, name string, values []T, compare func(a, b T) int) {
	t.Helper()
	for i, a := range values {
		if got := compare(a, a); got != 0 {
			t.Errorf("%s: comparing element %d with itself = %d, want 0", name, i, got)
		}
		for j, b := range values {
			if got, want := sign(compare(a, b)), -sign(compare(b, a)); got != want {
				t.Errorf("%s: elements %d and %d compare %d one way and %d the other",
					name, i, j, got, -want)
			}
		}
	}
}

// TestCompareTeamKeysOrdersByTeamNumber is the case that makes a comparator
// worth having at all: sorted as text, 1073 comes before 177.
func TestCompareTeamKeysOrdersByTeamNumber(t *testing.T) {
	keys := []string{"frc5507", "frc177", "frc1073"}
	slices.SortFunc(keys, CompareTeamKeys)
	if got, want := strings.Join(keys, " "), "frc177 frc1073 frc5507"; got != want {
		t.Errorf("sorted = %q, want %q", got, want)
	}
}

func TestCompareTeamKeysAcceptsBareNumbers(t *testing.T) {
	keys := []string{"5507", "177", "1073"}
	slices.SortFunc(keys, CompareTeamKeys)
	if got, want := strings.Join(keys, " "), "177 1073 5507"; got != want {
		t.Errorf("sorted = %q, want %q", got, want)
	}
}

// A key with no number in it still has to land somewhere, every run.
func TestCompareTeamKeysPutsUnreadableKeysLast(t *testing.T) {
	keys := []string{"nonsense", "frc1073", "another", "frc177"}
	slices.SortFunc(keys, CompareTeamKeys)
	if got, want := strings.Join(keys, " "), "frc177 frc1073 another nonsense"; got != want {
		t.Errorf("sorted = %q, want %q", got, want)
	}
	requireTotalOrder(t, "CompareTeamKeys", keys, CompareTeamKeys)
}

func TestTeamKeyNumber(t *testing.T) {
	cases := []struct {
		key  string
		want int
		ok   bool
	}{
		{"frc177", 177, true},
		{"FRC177", 177, true},
		{" frc177 ", 177, true},
		{"177", 177, true},
		{"frc", 0, false},
		{"", 0, false},
		{"frc17x7", 0, false},
	}
	for _, tc := range cases {
		got, ok := TeamKeyNumber(tc.key)
		if got != tc.want || ok != tc.ok {
			t.Errorf("TeamKeyNumber(%q) = (%d, %v), want (%d, %v)", tc.key, got, ok, tc.want, tc.ok)
		}
	}
}

func TestCompareMatchesFollowsPlayOrder(t *testing.T) {
	matches := []api.Match{
		{Key: "f1m1", CompLevel: "f", SetNumber: 1, MatchNumber: 1},
		{Key: "qm10", CompLevel: "qm", SetNumber: 1, MatchNumber: 10},
		{Key: "sf3m1", CompLevel: "sf", SetNumber: 3, MatchNumber: 1},
		{Key: "qm2", CompLevel: "qm", SetNumber: 1, MatchNumber: 2},
		{Key: "sf1m1", CompLevel: "sf", SetNumber: 1, MatchNumber: 1},
		{Key: "qf2m1", CompLevel: "qf", SetNumber: 2, MatchNumber: 1},
		{Key: "ef1m1", CompLevel: "ef", SetNumber: 1, MatchNumber: 1},
	}
	slices.SortStableFunc(matches, CompareMatches)
	want := "qm2 qm10 ef1m1 qf2m1 sf1m1 sf3m1 f1m1"
	if got := matchKeys(matches); got != want {
		t.Errorf("sorted = %q, want %q", got, want)
	}
	requireTotalOrder(t, "CompareMatches", matches, CompareMatches)
}

// A level this build has never heard of belongs after the ones it knows, not
// in the middle of the qualification matches.
func TestCompareMatchesPutsAnUnknownLevelLast(t *testing.T) {
	matches := []api.Match{
		{Key: "zz1", CompLevel: "zz", MatchNumber: 1},
		{Key: "qm1", CompLevel: "qm", MatchNumber: 1},
	}
	slices.SortStableFunc(matches, CompareMatches)
	if got, want := matchKeys(matches), "qm1 zz1"; got != want {
		t.Errorf("sorted = %q, want %q", got, want)
	}
}

func TestCompareMatchTimesPutsUntimedMatchesLast(t *testing.T) {
	late, early := int64(2000), int64(1000)
	matches := []api.Match{
		{Key: "none"},
		{Key: "late", Time: &late},
		{Key: "early", Time: &early},
	}
	slices.SortStableFunc(matches, CompareMatchTimes)
	if got, want := matchKeys(matches), "early late none"; got != want {
		t.Errorf("sorted = %q, want %q", got, want)
	}
	requireTotalOrder(t, "CompareMatchTimes", matches, CompareMatchTimes)
}

func matchKeys(matches []api.Match) string {
	keys := make([]string, len(matches))
	for i, m := range matches {
		keys[i] = m.Key
	}
	return strings.Join(keys, " ")
}

func TestCompareMatchKeysFollowsPlayOrder(t *testing.T) {
	keys := []string{
		"2024cthar_f1m2", "2024cthar_qm10", "2024cthar_sf3m1", "2024cthar_qm2",
		"2024cthar_f1m1", "2024cthar_sf1m1", "2024cthar_qf2m1", "2024cthar_ef1m1",
		"2024cthar_qm1",
	}
	slices.SortStableFunc(keys, CompareMatchKeys)
	want := strings.Join([]string{
		"2024cthar_qm1", "2024cthar_qm2", "2024cthar_qm10",
		"2024cthar_ef1m1", "2024cthar_qf2m1",
		"2024cthar_sf1m1", "2024cthar_sf3m1",
		"2024cthar_f1m1", "2024cthar_f1m2",
	}, " ")
	if got := strings.Join(keys, " "); got != want {
		t.Errorf("sorted = %q, want %q", got, want)
	}
	requireTotalOrder(t, "CompareMatchKeys", keys, CompareMatchKeys)
}

func TestCompareMatchKeysPutsUnreadableKeysLast(t *testing.T) {
	keys := []string{"nonsense", "2024cthar_qm2", "2024cthar_qm1", "also bad"}
	slices.SortStableFunc(keys, CompareMatchKeys)
	if got, want := strings.Join(keys, " "), "2024cthar_qm1 2024cthar_qm2 also bad nonsense"; got != want {
		t.Errorf("sorted = %q, want %q", got, want)
	}
}

// CompareMatchKeys and CompareMatches answer the same question, so a list of
// keys and a list of matches must come out in the same order.
func TestCompareMatchKeysAgreesWithCompareMatches(t *testing.T) {
	keys := []string{
		"2024cthar_f1m2", "2024cthar_qm10", "2024cthar_sf3m1", "2024cthar_qm2",
		"2024cthar_ef1m1", "2024cthar_qf2m1",
	}
	matches := make([]api.Match, len(keys))
	for i, key := range keys {
		matches[i] = MatchFromKey(key)
	}
	slices.SortStableFunc(keys, CompareMatchKeys)
	slices.SortStableFunc(matches, CompareMatches)
	if got := matchKeys(matches); got != strings.Join(keys, " ") {
		t.Errorf("matches sorted %q, keys sorted %q", got, strings.Join(keys, " "))
	}
}

func TestMatchFromKey(t *testing.T) {
	cases := []struct {
		key          string
		event, level string
		set, match   int
	}{
		{"2024cthar_qm12", "2024cthar", "qm", 1, 12},
		{"2024cthar_sf13m1", "2024cthar", "sf", 13, 1},
		{"2024cthar_f1m2", "2024cthar", "f", 1, 2},
		{"2024cthar_zz1", "2024cthar", "", 0, 0},
		{"nonsense", "", "", 0, 0},
	}
	for _, tc := range cases {
		m := MatchFromKey(tc.key)
		if m.Key != tc.key || m.EventKey != tc.event || m.CompLevel != tc.level ||
			m.SetNumber != tc.set || m.MatchNumber != tc.match {
			t.Errorf("MatchFromKey(%q) = %+v, want event %q level %q set %d match %d",
				tc.key, m, tc.event, tc.level, tc.set, tc.match)
		}
	}
}

func TestCompareEventsOrdersByStartDate(t *testing.T) {
	events := []api.Event{
		{Key: "2024necmp", StartDate: "2024-04-03"},
		{Key: "2024ctwat", StartDate: "2024-03-08"},
		{Key: "2024cthar", StartDate: "2024-03-22"},
	}
	slices.SortStableFunc(events, CompareEvents)
	if got, want := eventKeys(events), "2024ctwat 2024cthar 2024necmp"; got != want {
		t.Errorf("sorted = %q, want %q", got, want)
	}
	requireTotalOrder(t, "CompareEvents", events, CompareEvents)
}

// Two events on the same day, and an event the API sent no date for, still
// have to come out the same way every run.
func TestCompareEventsBreaksTiesByKeyAndDatesUndatedLast(t *testing.T) {
	events := []api.Event{
		{Key: "2024undated"},
		{Key: "2024ctwat", StartDate: "2024-03-08"},
		{Key: "2024ctbri", StartDate: "2024-03-08"},
		{Key: "2024broken", StartDate: "not-a-date"},
	}
	slices.SortStableFunc(events, CompareEvents)
	if got, want := eventKeys(events), "2024ctbri 2024ctwat 2024broken 2024undated"; got != want {
		t.Errorf("sorted = %q, want %q", got, want)
	}
	requireTotalOrder(t, "CompareEvents", events, CompareEvents)
}

func eventKeys(events []api.Event) string {
	keys := make([]string, len(events))
	for i, e := range events {
		keys[i] = e.Key
	}
	return strings.Join(keys, " ")
}

func TestCompareRankingsOrdersByRank(t *testing.T) {
	rankings := []api.Ranking{{TeamKey: "frc1073", Rank: 3}, {TeamKey: "frc177", Rank: 1}}
	slices.SortStableFunc(rankings, CompareRankings)
	if rankings[0].Rank != 1 || rankings[1].Rank != 3 {
		t.Errorf("sorted = %+v", rankings)
	}
	requireTotalOrder(t, "CompareRankings", rankings, CompareRankings)
}

func TestCompareDistrictRankingsOrdersByRank(t *testing.T) {
	rankings := []api.DistrictRanking{{TeamKey: "frc1073", Rank: 3}, {TeamKey: "frc177", Rank: 1}}
	slices.SortStableFunc(rankings, CompareDistrictRankings)
	if rankings[0].Rank != 1 || rankings[1].Rank != 3 {
		t.Errorf("sorted = %+v", rankings)
	}
	requireTotalOrder(t, "CompareDistrictRankings", rankings, CompareDistrictRankings)
}

func TestCompareBreakdownKeysIsNumericAware(t *testing.T) {
	keys := []string{"autoCommunity.B.11", "autoCommunity.B.2", "autoCommunity.B.1", "autoPoints"}
	slices.SortStableFunc(keys, CompareBreakdownKeys)
	want := "autoCommunity.B.1 autoCommunity.B.2 autoCommunity.B.11 autoPoints"
	if got := strings.Join(keys, " "); got != want {
		t.Errorf("sorted = %q, want %q", got, want)
	}
	requireTotalOrder(t, "CompareBreakdownKeys", keys, CompareBreakdownKeys)
}
