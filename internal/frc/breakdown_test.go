package frc_test

import (
	"encoding/json"
	"reflect"
	"strconv"
	"testing"

	"github.com/the-blue-alliance/tba-cli/internal/api"
	"github.com/the-blue-alliance/tba-cli/internal/frc"
)

// A slice of a real 2024 score breakdown, including the nested object and the
// string-valued enums the game used.
const breakdown2024JSON = `{
  "autoLineRobot1": "Yes",
  "autoLineRobot2": "No",
  "autoPoints": 20,
  "autoSpeakerNoteCount": 2,
  "coopertitionBonusAchieved": true,
  "endGameHarmonyPoints": 4,
  "endGameRobot1": "Parked",
  "g424Penalty": false,
  "rp": 4,
  "teleopSpeakerNoteAmplifiedPoints": 25.5,
  "totalPoints": 88,
  "foulCount": 0,
  "adjustPoints": -3,
  "nested": {"a": 1, "b": {"c": "deep"}},
  "list": ["x", "y"],
  "empty": [],
  "missing": null
}`

func TestFlattenBreakdown(t *testing.T) {
	var raw map[string]interface{}
	if err := json.Unmarshal([]byte(breakdown2024JSON), &raw); err != nil {
		t.Fatalf("decoding fixture: %v", err)
	}
	got := frc.FlattenBreakdown(raw)

	index := map[string]string{}
	for _, kv := range got {
		index[kv.Key] = kv.Value
	}
	want := map[string]string{
		"autoLineRobot1":                   "Yes",
		"autoPoints":                       "20",
		"coopertitionBonusAchieved":        "yes",
		"g424Penalty":                      "no",
		"teleopSpeakerNoteAmplifiedPoints": "25.5",
		"adjustPoints":                     "-3",
		"nested.a":                         "1",
		"nested.b.c":                       "deep",
		"list.0":                           "x",
		"list.1":                           "y",
		"empty":                            "",
		"missing":                          "",
	}
	for key, value := range want {
		if index[key] != value {
			t.Errorf("%s = %q, want %q", key, index[key], value)
		}
	}
}

// The breakdown's shape changes every season, so the only ordering that can be
// promised is that it is the same every time.
func TestFlattenBreakdownSortsItsKeys(t *testing.T) {
	got := frc.FlattenBreakdown(map[string]interface{}{
		"zeta": 1.0, "alpha": 2.0, "mu": 3.0,
	})
	want := []frc.KV{{Key: "alpha", Value: "2"}, {Key: "mu", Value: "3"}, {Key: "zeta", Value: "1"}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("FlattenBreakdown = %v, want %v", got, want)
	}
}

// A twelve-slot grid is a real shape for a score breakdown, and a plain
// string sort turns it into 1, 10, 11, 2 — which is not a listing anyone can
// read down.
func TestFlattenBreakdownSortsArrayIndicesAsNumbers(t *testing.T) {
	grid := make([]interface{}, 12)
	for i := range grid {
		grid[i] = float64(i)
	}
	got := frc.FlattenBreakdown(map[string]interface{}{"autoCommunity": grid})

	var keys []string
	for _, kv := range got {
		keys = append(keys, kv.Key)
	}
	var want []string
	for i := 0; i < 12; i++ {
		want = append(want, "autoCommunity."+strconv.Itoa(i))
	}
	if !reflect.DeepEqual(keys, want) {
		t.Errorf("keys =\n%v\nwant\n%v", keys, want)
	}
}

func TestCompareBreakdownKeys(t *testing.T) {
	cases := []struct {
		a, b string
		want int
	}{
		{"x.2", "x.10", -1},
		{"x.10", "x.2", 1},
		{"x.2", "x.2", 0},
		{"a", "b", -1},
		// A prefix comes before what extends it.
		{"auto", "auto.0", -1},
		// A nested array is ordered at the level the indices are on.
		{"g.1.b", "g.10.a", -1},
		// Numbers before names at the same level, so indices stay together.
		{"g.3", "g.total", -1},
		// A name that merely starts with a digit is still a name.
		{"g.2x", "g.10", 1},
	}
	for _, c := range cases {
		got := frc.CompareBreakdownKeys(c.a, c.b)
		if (got < 0) != (c.want < 0) || (got > 0) != (c.want > 0) {
			t.Errorf("CompareBreakdownKeys(%q, %q) = %d, want sign %d", c.a, c.b, got, c.want)
		}
	}
}

func TestFlattenBreakdownOfNothing(t *testing.T) {
	if got := frc.FlattenBreakdown(nil); len(got) != 0 {
		t.Errorf("FlattenBreakdown(nil) = %v, want empty", got)
	}
}

func TestAllianceBreakdown(t *testing.T) {
	m := mustMatch(t, qual2024JSON)
	got := frc.AllianceBreakdown(m, frc.AllianceRed)
	if !reflect.DeepEqual(got, []frc.KV{{Key: "totalPoints", Value: "88"}}) {
		t.Errorf("red breakdown = %v", got)
	}
}

// Seasons before 2015 have no breakdown at all, and neither does an unplayed
// match.
func TestAllianceBreakdownWhenThereIsNone(t *testing.T) {
	if got := frc.AllianceBreakdown(mustMatch(t, qual2015TieJSON), frc.AllianceRed); got != nil {
		t.Errorf("breakdown = %v, want nil", got)
	}
	if got := frc.AllianceBreakdown(api.Match{}, frc.AllianceRed); got != nil {
		t.Errorf("breakdown = %v, want nil", got)
	}
	// A breakdown that exists but has no section for this alliance.
	m := api.Match{ScoreBreakdown: map[string]interface{}{"red": "unexpected"}}
	if got := frc.AllianceBreakdown(m, frc.AllianceRed); got != nil {
		t.Errorf("breakdown = %v, want nil", got)
	}
}

func TestFormatValue(t *testing.T) {
	cases := []struct {
		in   interface{}
		want string
	}{
		{nil, ""},
		{true, "yes"},
		{false, "no"},
		{float64(20), "20"},
		{float64(25.5), "25.5"},
		{float64(-3), "-3"},
		{"Parked", "Parked"},
	}
	for _, c := range cases {
		if got := frc.FormatValue(c.in); got != c.want {
			t.Errorf("FormatValue(%v) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestLevelName(t *testing.T) {
	cases := map[string]string{
		"qm": "Qualification",
		"ef": "Octofinals",
		"qf": "Quarterfinals",
		"sf": "Semifinals",
		"f":  "Finals",
		"":   "",
		"zz": "zz",
	}
	for level, want := range cases {
		if got := frc.LevelName(level); got != want {
			t.Errorf("LevelName(%q) = %q, want %q", level, got, want)
		}
	}
}

func TestPickPosition(t *testing.T) {
	cases := []struct {
		name     string
		alliance api.TeamEventAllianceStatus
		teamKey  string
		want     string
	}{
		{"captain", api.TeamEventAllianceStatus{Pick: 0}, "frc177", "Captain"},
		{"first pick", api.TeamEventAllianceStatus{Pick: 1}, "frc177", "1st pick"},
		{"second pick", api.TeamEventAllianceStatus{Pick: 2}, "frc177", "2nd pick"},
		{"third pick", api.TeamEventAllianceStatus{Pick: 3}, "frc177", "3rd pick"},
		{"a longer alliance", api.TeamEventAllianceStatus{Pick: 4}, "frc177", "pick 4"},
		{
			"a backup called in",
			api.TeamEventAllianceStatus{Pick: 2, Backup: &api.AllianceBackup{Out: "frc230", In: "frc177"}},
			"frc177",
			"backup",
		},
		{
			"another team on an alliance that took a backup",
			api.TeamEventAllianceStatus{Pick: 0, Backup: &api.AllianceBackup{Out: "frc230", In: "frc558"}},
			"frc177",
			"Captain",
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := frc.PickPosition(c.alliance, c.teamKey); got != c.want {
				t.Errorf("PickPosition = %q, want %q", got, c.want)
			}
		})
	}
}

func TestRecord(t *testing.T) {
	if got := frc.Record(&api.WLTRecord{Wins: 10, Losses: 2, Ties: 1}); got != "10-2-1" {
		t.Errorf("Record = %q", got)
	}
	if got := frc.Record(nil); got != "" {
		t.Errorf("Record(nil) = %q, want %q", got, "")
	}
}

func TestStripHTML(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{"plain prose", "Team 177 is competing", "Team 177 is competing"},
		{"bold", "Team 177 was <b>Rank 1</b>", "Team 177 was Rank 1"},
		{
			"a link",
			`Team 177 will be competing in the <b>Finals</b> as the <b>Captain</b> of <a href="/event/2024cthar">Alliance 1</a>.`,
			"Team 177 will be competing in the Finals as the Captain of Alliance 1.",
		},
		{"entities", "Bobcats &amp; friends", "Bobcats & friends"},
		{"surrounding whitespace", "  <b>Rank 1</b>  ", "Rank 1"},
		{"an unterminated tag", "Rank 1 <b", "Rank 1"},
		{"a stray close", "Rank > 1", "Rank > 1"},
		{"empty", "", ""},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := frc.StripHTML(c.in); got != c.want {
				t.Errorf("StripHTML(%q) = %q, want %q", c.in, got, c.want)
			}
		})
	}
}
