package frc_test

import (
	"reflect"
	"strings"
	"testing"

	"github.com/the-blue-alliance/tba-cli/internal/api"
	"github.com/the-blue-alliance/tba-cli/internal/frc"
)

func TestCompLevelOrderFollowsPlayingOrder(t *testing.T) {
	want := []string{"qm", "ef", "qf", "sf", "f"}
	for i, level := range want {
		if got := frc.CompLevelOrder(level); got != i {
			t.Errorf("CompLevelOrder(%q) = %d, want %d", level, got, i)
		}
	}
}

func TestCompLevelOrderIsCaseInsensitive(t *testing.T) {
	if frc.CompLevelOrder("QF") != frc.CompLevelOrder("qf") {
		t.Error("an upper-case level should sort like its lower-case spelling")
	}
}

func TestCompLevelOrderPutsUnknownLevelsLast(t *testing.T) {
	for _, level := range []string{"", "zz", "quarterfinal"} {
		if got := frc.CompLevelOrder(level); got != frc.UnknownCompLevel {
			t.Errorf("CompLevelOrder(%q) = %d, want %d", level, got, frc.UnknownCompLevel)
		}
	}
}

func TestSortMatchesOrdersLevelThenSetThenMatch(t *testing.T) {
	matches := []api.Match{
		{Key: "f1m2", CompLevel: "f", SetNumber: 1, MatchNumber: 2},
		{Key: "qm10", CompLevel: "qm", SetNumber: 1, MatchNumber: 10},
		{Key: "sf2m1", CompLevel: "sf", SetNumber: 2, MatchNumber: 1},
		{Key: "qm2", CompLevel: "qm", SetNumber: 1, MatchNumber: 2},
		{Key: "qf1m1", CompLevel: "qf", SetNumber: 1, MatchNumber: 1},
		{Key: "sf1m1", CompLevel: "sf", SetNumber: 1, MatchNumber: 1},
		{Key: "f1m1", CompLevel: "f", SetNumber: 1, MatchNumber: 1},
		{Key: "ef1m1", CompLevel: "ef", SetNumber: 1, MatchNumber: 1},
	}
	frc.SortMatches(matches)

	want := []string{"qm2", "qm10", "ef1m1", "qf1m1", "sf1m1", "sf2m1", "f1m1", "f1m2"}
	if got := keysOf(matches); !reflect.DeepEqual(got, want) {
		t.Errorf("order = %v, want %v", got, want)
	}
}

// Qualification match 10 must not sort before match 2, which is what a plain
// alphabetical sort of the keys would do.
func TestSortMatchesIsNotAlphabetical(t *testing.T) {
	matches := []api.Match{
		{Key: "2024cthar_qm10", CompLevel: "qm", MatchNumber: 10},
		{Key: "2024cthar_qm2", CompLevel: "qm", MatchNumber: 2},
	}
	frc.SortMatches(matches)
	if matches[0].MatchNumber != 2 {
		t.Errorf("first match is %d, want 2", matches[0].MatchNumber)
	}
}

func TestSortMatchesIsStable(t *testing.T) {
	matches := []api.Match{
		{Key: "b", CompLevel: "qm", MatchNumber: 1},
		{Key: "a", CompLevel: "qm", MatchNumber: 1},
	}
	frc.SortMatches(matches)
	if got := keysOf(matches); !reflect.DeepEqual(got, []string{"b", "a"}) {
		t.Errorf("order = %v, want the input order kept", got)
	}
}

func TestSortMatchesOnEmptySlice(t *testing.T) {
	var matches []api.Match
	frc.SortMatches(matches)
	if len(matches) != 0 {
		t.Errorf("len = %d", len(matches))
	}
}

func TestSortByTimeUsesTheBestKnownTime(t *testing.T) {
	matches := []api.Match{
		{Key: "later", CompLevel: "qm", MatchNumber: 1, Time: epoch(3000)},
		{Key: "earlier", CompLevel: "qm", MatchNumber: 9, PredictedTime: epoch(1000)},
		{Key: "middle", CompLevel: "f", MatchNumber: 1, ActualTime: epoch(2000)},
	}
	frc.SortByTime(matches)
	if got := keysOf(matches); !reflect.DeepEqual(got, []string{"earlier", "middle", "later"}) {
		t.Errorf("order = %v", got)
	}
}

func TestSortByTimePutsUntimedMatchesLastInPlayingOrder(t *testing.T) {
	matches := []api.Match{
		{Key: "notime-sf", CompLevel: "sf", SetNumber: 1, MatchNumber: 1},
		{Key: "timed", CompLevel: "f", MatchNumber: 1, Time: epoch(500)},
		{Key: "notime-qm", CompLevel: "qm", MatchNumber: 3},
	}
	frc.SortByTime(matches)
	if got := keysOf(matches); !reflect.DeepEqual(got, []string{"timed", "notime-qm", "notime-sf"}) {
		t.Errorf("order = %v", got)
	}
}

func TestIsDoubleElim(t *testing.T) {
	cases := []struct {
		playoffType *int
		want        bool
	}{
		{nil, false},
		{intPtr(frc.PlayoffBracket8Team), false},
		{intPtr(frc.PlayoffBracket16Team), false},
		{intPtr(frc.PlayoffRoundRobin6Team), false},
		{intPtr(frc.PlayoffDoubleElim8Team), true},
		{intPtr(frc.PlayoffDoubleElim4Team), true},
	}
	for _, c := range cases {
		if got := frc.IsDoubleElim(c.playoffType); got != c.want {
			t.Errorf("IsDoubleElim(%v) = %v, want %v", c.playoffType, got, c.want)
		}
	}
}

func TestMatchLabel(t *testing.T) {
	doubleElim := intPtr(frc.PlayoffDoubleElim8Team)
	bracket := intPtr(frc.PlayoffBracket8Team)
	roundRobin := intPtr(frc.PlayoffRoundRobin6Team)

	cases := []struct {
		name        string
		raw         string
		playoffType *int
		want        string
	}{
		{"qual", qual2024JSON, doubleElim, "Qual 12"},
		{"qual ignores the bracket format", qual2024JSON, bracket, "Qual 12"},
		{"double elim semifinal is named by its set", doubleElimSF13JSON, doubleElim, "SF 13"},
		{"double elim final is named by its match", doubleElimFinal2JSON, doubleElim, "Final 2"},
		{"legacy quarterfinal", legacyQF2019JSON, bracket, "QF 4-3"},
		{"legacy semifinal", legacySF2019JSON, bracket, "SF 1-2"},
		{"legacy octofinal", legacyEF2018JSON, bracket, "EF 3-1"},
		{"round robin semifinal", roundRobin2019JSON, roundRobin, "SF 1-3"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := frc.MatchLabel(mustMatch(t, c.raw), c.playoffType); got != c.want {
				t.Errorf("MatchLabel = %q, want %q", got, c.want)
			}
		})
	}
}

// An octofinal set is always best-of-three, so it keeps both numbers even at a
// double-elimination event.
func TestMatchLabelKeepsBothNumbersForEFUnderDoubleElim(t *testing.T) {
	m := mustMatch(t, legacyEF2018JSON)
	if got := frc.MatchLabel(m, intPtr(frc.PlayoffDoubleElim8Team)); got != "EF 3-1" {
		t.Errorf("MatchLabel = %q, want %q", got, "EF 3-1")
	}
}

// When the event could not be fetched, the season decides how playoff sets are
// named: double elimination from 2023 on, best-of-three sets before that.
func TestMatchLabelFallsBackToTheSeason(t *testing.T) {
	cases := []struct {
		name string
		raw  string
		want string
	}{
		{"2024 semifinal", doubleElimSF13JSON, "SF 13"},
		{"2019 semifinal", legacySF2019JSON, "SF 1-2"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := frc.MatchLabel(mustMatch(t, c.raw), nil); got != c.want {
				t.Errorf("MatchLabel = %q, want %q", got, c.want)
			}
		})
	}
}

func TestMatchLabelForUnknownLevels(t *testing.T) {
	if got := frc.MatchLabel(api.Match{Key: "2030xx_zz1m4", CompLevel: "zz", SetNumber: 1, MatchNumber: 4}, nil); got != "ZZ 1-4" {
		t.Errorf("MatchLabel = %q", got)
	}
	if got := frc.MatchLabel(api.Match{Key: "2024cthar_qm1"}, nil); got != "2024cthar_qm1" {
		t.Errorf("a match with no comp level should fall back to its key, got %q", got)
	}
}

func TestMatchYear(t *testing.T) {
	if got := frc.MatchYear(mustMatch(t, qual2015TieJSON)); got != 2015 {
		t.Errorf("MatchYear = %d, want 2015", got)
	}
	// With no event key, the match key carries the season.
	if got := frc.MatchYear(api.Match{Key: "2024cthar_qm1"}); got != 2024 {
		t.Errorf("MatchYear = %d, want 2024", got)
	}
	for _, m := range []api.Match{{}, {Key: "abc"}, {Key: "abcdefg_qm1"}} {
		if got := frc.MatchYear(m); got != 0 {
			t.Errorf("MatchYear(%+v) = %d, want 0", m, got)
		}
	}
}

func TestPlayedAndStatus(t *testing.T) {
	played := mustMatch(t, qual2024JSON)
	if !frc.Played(played) || frc.MatchStatus(played) != frc.StatusPlayed {
		t.Errorf("a scored match should be Played, got %q", frc.MatchStatus(played))
	}
	unplayed := mustMatch(t, unplayed2024JSON)
	if frc.Played(unplayed) || frc.MatchStatus(unplayed) != frc.StatusScheduled {
		t.Errorf("a -1/-1 match should be Scheduled, got %q", frc.MatchStatus(unplayed))
	}
}

// A match with no alliances at all is not a crash and is not played.
func TestPlayedWithNoAlliances(t *testing.T) {
	if frc.Played(api.Match{Key: "2024cthar_qm1"}) {
		t.Error("a match with no alliances should not count as played")
	}
	if got := frc.Score(api.Match{}, frc.AllianceRed); got != frc.UnplayedScore {
		t.Errorf("Score = %d, want %d", got, frc.UnplayedScore)
	}
}

// A zero-score match is still played: 0-0 happens.
func TestPlayedWithZeroScores(t *testing.T) {
	m := api.Match{Alliances: map[string]api.Alliance{
		"red":  {Score: 0},
		"blue": {Score: 0},
	}}
	if !frc.Played(m) {
		t.Error("0-0 is a played match")
	}
}

func TestWinner(t *testing.T) {
	cases := []struct {
		name string
		raw  string
		want string
	}{
		{"red wins", qual2024JSON, "red"},
		{"blue wins", doubleElimSF13JSON, "blue"},
		{"unplayed has no winner", unplayed2024JSON, ""},
		{"played with no winning alliance is a tie", qual2015TieJSON, "tie"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := frc.Winner(mustMatch(t, c.raw)); got != c.want {
				t.Errorf("Winner = %q, want %q", got, c.want)
			}
		})
	}
}

func TestMarkTeam(t *testing.T) {
	a := api.Alliance{
		TeamKeys:          []string{"frc177", "frc1073", "frc5507"},
		SurrogateTeamKeys: []string{"frc1073"},
		DQTeamKeys:        []string{"frc5507"},
	}
	cases := map[string]string{
		"frc177":  "177",
		"frc1073": "1073" + frc.MarkSurrogate,
		"frc5507": "5507" + frc.MarkDQ,
	}
	for key, want := range cases {
		if got := frc.MarkTeam(key, a); got != want {
			t.Errorf("MarkTeam(%q) = %q, want %q", key, got, want)
		}
	}
}

// A surrogate can also be disqualified; both marks show, surrogate first.
func TestMarkTeamCarriesBothMarks(t *testing.T) {
	a := api.Alliance{
		TeamKeys:          []string{"frc177"},
		SurrogateTeamKeys: []string{"frc177"},
		DQTeamKeys:        []string{"frc177"},
	}
	if got := frc.MarkTeam("frc177", a); got != "177*!" {
		t.Errorf("MarkTeam = %q, want %q", got, "177*!")
	}
}

func TestMarkedTeams(t *testing.T) {
	m := mustMatch(t, qual2024JSON)
	if got := frc.MarkedTeams(m.Alliances["red"]); !reflect.DeepEqual(got, []string{"177", "1073", "5507"}) {
		t.Errorf("red = %v", got)
	}
	if got := frc.MarkedTeams(m.Alliances["blue"]); !reflect.DeepEqual(got, []string{"230", "1071", "4055*"}) {
		t.Errorf("blue = %v", got)
	}
}

func TestMarkedTeamsFlagsADisqualification(t *testing.T) {
	m := mustMatch(t, legacyQF2019JSON)
	if got := frc.MarkedTeams(m.Alliances["red"]); !reflect.DeepEqual(got, []string{"1071", "4055", "2168!"}) {
		t.Errorf("red = %v", got)
	}
}

func TestLegendCoversEveryMark(t *testing.T) {
	for _, mark := range []string{frc.MarkSurrogate, frc.MarkDQ} {
		if !strings.Contains(frc.Legend, mark) {
			t.Errorf("the legend %q does not explain %q", frc.Legend, mark)
		}
		if !strings.Contains(frc.MarkChars, mark) {
			t.Errorf("MarkChars %q is missing %q", frc.MarkChars, mark)
		}
	}
}

func TestTeamNumber(t *testing.T) {
	if got := frc.TeamNumber("frc177"); got != "177" {
		t.Errorf("TeamNumber = %q", got)
	}
	// A key that is not a team key is left alone rather than mangled.
	if got := frc.TeamNumber("177"); got != "177" {
		t.Errorf("TeamNumber = %q", got)
	}
}

func TestHasTeam(t *testing.T) {
	m := mustMatch(t, qual2024JSON)
	for _, key := range []string{"frc177", "frc4055"} {
		if !frc.HasTeam(m, key) {
			t.Errorf("HasTeam(%q) = false", key)
		}
	}
	if frc.HasTeam(m, "frc9999") {
		t.Error("HasTeam(frc9999) = true")
	}
}

func TestAllianceOfAndStation(t *testing.T) {
	m := mustMatch(t, qual2024JSON)
	cases := []struct {
		key      string
		alliance string
		station  int
	}{
		{"frc177", "red", 1},
		{"frc5507", "red", 3},
		{"frc1071", "blue", 2},
		{"frc9999", "", 0},
	}
	for _, c := range cases {
		if got := frc.AllianceOf(m, c.key); got != c.alliance {
			t.Errorf("AllianceOf(%q) = %q, want %q", c.key, got, c.alliance)
		}
		if got := frc.Station(m, c.key); got != c.station {
			t.Errorf("Station(%q) = %d, want %d", c.key, got, c.station)
		}
	}
}

// A 2021 remote event ran no matches at all; every listing must survive it.
func TestNoMatchesIsNotAnError(t *testing.T) {
	var matches []api.Match
	frc.SortMatches(matches)
	frc.SortByTime(matches)
	if len(matches) != 0 {
		t.Errorf("len = %d, want 0", len(matches))
	}
}

func keysOf(matches []api.Match) []string {
	out := make([]string, len(matches))
	for i, m := range matches {
		out[i] = m.Key
	}
	return out
}
