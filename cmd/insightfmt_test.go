package cmd

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestHumanizeName(t *testing.T) {
	cases := map[string]string{
		"blue_banners":        "Blue Banners",
		"most_matches_played": "Most Matches Played",
		"hall_of_fame":        "Hall Of Fame",
		"average_score":       "Average Score",
		"high_score":          "High Score",
		"":                    "",
		"score":               "Score",
		// A word that already carries capitals is left alone, so the acronyms
		// FRC statistics are full of survive.
		"average_OPR": "Average OPR",
		"unicat_ñame": "Unicat Ñame",
	}
	for in, want := range cases {
		if got := humanizeName(in); got != want {
			t.Errorf("humanizeName(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestHumanizeStatKeyKeepsNesting(t *testing.T) {
	if got := humanizeStatKey("score_by_alliance.red"); got != "Score By Alliance.Red" {
		t.Errorf("got %q", got)
	}
}

func TestNormalizeBoardName(t *testing.T) {
	want := "blue banners"
	for _, in := range []string{"Blue Banners", "blue_banners", "  BLUE   BANNERS ", "blue banners"} {
		if got := normalizeBoardName(in); got != want {
			t.Errorf("normalizeBoardName(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestFormatNumberTrimsToTwoDecimals(t *testing.T) {
	cases := map[float64]string{
		88:        "88",
		61.4:      "61.4",
		61.456:    "61.46",
		0:         "0",
		-0.001:    "0",
		112.5:     "112.5",
		0.7853:    "0.79",
		-12.126:   "-12.13",
		1234567.8: "1234567.8",
	}
	for in, want := range cases {
		if got := formatNumber(in); got != want {
			t.Errorf("formatNumber(%v) = %q, want %q", in, got, want)
		}
	}
}

func TestFormatInsightValue(t *testing.T) {
	cases := []struct {
		body string
		want string
	}{
		{`[12, 60, 20]`, "12/60 (20%)"},
		{`[12, 60, 20.456]`, "12/60 (20.46%)"},
		{`["2024cthar_qm1", 88, "Quals 1"]`, "2024cthar_qm1, 88, Quals 1"},
		{`["a", "b"]`, "a, b"},
		{`[1, 2]`, "1, 2"},
		{`61.4`, "61.4"},
		{`"hello"`, "hello"},
		{`true`, "true"},
		{`null`, ""},
		{`[]`, ""},
		{`{"mean": 61.4, "var": 210.25}`, "mean=61.4, var=210.25"},
	}
	for _, tc := range cases {
		var v interface{}
		if err := json.Unmarshal([]byte(tc.body), &v); err != nil {
			t.Fatalf("fixture %s: %v", tc.body, err)
		}
		if got := formatInsightValue(v); got != tc.want {
			t.Errorf("formatInsightValue(%s) = %q, want %q", tc.body, got, tc.want)
		}
	}
}

func TestFlattenInsightStatsIsSortedAndDotted(t *testing.T) {
	var stats map[string]interface{}
	body := `{
	  "score": {"mean": 61.4, "var": 210.25},
	  "average_score": 55,
	  "bonus": {"auto": {"achieved": [12, 60, 20]}},
	  "empty": {}
	}`
	if err := json.Unmarshal([]byte(body), &stats); err != nil {
		t.Fatalf("fixture: %v", err)
	}
	got := flattenInsightStats("", stats)
	want := [][2]string{
		{"average_score", "55"},
		{"bonus.auto.achieved", "12/60 (20%)"},
		{"empty", ""},
		{"score.mean", "61.4"},
		{"score.var", "210.25"},
	}
	if len(got) != len(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("pair %d = %v, want %v", i, got[i], want[i])
		}
	}
}

func TestSortMatchKeysFollowsPlayOrder(t *testing.T) {
	keys := []string{
		"2024cthar_f1m2", "2024cthar_qm10", "2024cthar_sf3m1", "2024cthar_qm2",
		"2024cthar_f1m1", "2024cthar_sf1m1", "2024cthar_qf2m1", "2024cthar_ef1m1",
		"2024cthar_qm1",
	}
	sortMatchKeys(keys)
	want := []string{
		"2024cthar_qm1", "2024cthar_qm2", "2024cthar_qm10",
		"2024cthar_ef1m1", "2024cthar_qf2m1",
		"2024cthar_sf1m1", "2024cthar_sf3m1",
		"2024cthar_f1m1", "2024cthar_f1m2",
	}
	if strings.Join(keys, " ") != strings.Join(want, " ") {
		t.Errorf("got %v\nwant %v", keys, want)
	}
}

// An unparseable key must not swallow the ones around it, and the order has to
// stay the same on every run.
func TestSortMatchKeysPutsUnknownKeysLast(t *testing.T) {
	keys := []string{"nonsense", "2024cthar_qm2", "2024cthar_qm1", "also bad"}
	sortMatchKeys(keys)
	want := []string{"2024cthar_qm1", "2024cthar_qm2", "also bad", "nonsense"}
	if strings.Join(keys, " ") != strings.Join(want, " ") {
		t.Errorf("got %v, want %v", keys, want)
	}
}
