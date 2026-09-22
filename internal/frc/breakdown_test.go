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

// A 2024 Crescendo match with both alliances' breakdowns, shortened from a
// real one but keeping its shape: a total, ranking points, a dozen point
// columns, the fouls, the counts, and a row of flags neither alliance earned.
const compare2024JSON = `{
  "key": "2024cthar_qm12",
  "comp_level": "qm",
  "set_number": 1,
  "match_number": 12,
  "event_key": "2024cthar",
  "winning_alliance": "red",
  "alliances": {
    "red": {"score": 88, "team_keys": ["frc177", "frc1073", "frc5507"], "surrogate_team_keys": [], "dq_team_keys": []},
    "blue": {"score": 61, "team_keys": ["frc230", "frc1071", "frc4055"], "surrogate_team_keys": [], "dq_team_keys": []}
  },
  "score_breakdown": {
    "red": {
      "adjustPoints": 0,
      "autoAmpNoteCount": 1,
      "autoAmpNotePoints": 2,
      "autoLeavePoints": 6,
      "autoLineRobot1": "Yes",
      "autoPoints": 23,
      "autoSpeakerNoteCount": 3,
      "autoSpeakerNotePoints": 15,
      "coopertitionBonusAchieved": false,
      "endGameNoteInTrapPoints": 0,
      "endGameOnStagePoints": 6,
      "endGameRobot1": "StageLeft",
      "foulCount": 1,
      "foulPoints": 2,
      "g424Penalty": false,
      "melodyBonusAchieved": true,
      "rp": 5,
      "techFoulCount": 0,
      "teleopPoints": 57,
      "totalPoints": 88,
      "trapCenterStage": false
    },
    "blue": {
      "adjustPoints": 0,
      "autoAmpNoteCount": 0,
      "autoAmpNotePoints": 0,
      "autoLeavePoints": 4,
      "autoLineRobot1": "Yes",
      "autoPoints": 14,
      "autoSpeakerNoteCount": 2,
      "autoSpeakerNotePoints": 10,
      "coopertitionBonusAchieved": false,
      "endGameNoteInTrapPoints": 0,
      "endGameOnStagePoints": 3,
      "endGameRobot1": "Parked",
      "foulCount": 0,
      "foulPoints": 0,
      "g424Penalty": false,
      "melodyBonusAchieved": false,
      "rp": 1,
      "techFoulCount": 0,
      "teleopPoints": 44,
      "totalPoints": 61,
      "trapCenterStage": false
    }
  },
  "videos": []
}`

// labelsOf is the order a breakdown comes out in.
func labelsOf(rows []frc.BreakdownRow) []string {
	out := make([]string, len(rows))
	for i, row := range rows {
		out[i] = row.Label
	}
	return out
}

func rowFor(rows []frc.BreakdownRow, key string) (frc.BreakdownRow, bool) {
	for _, row := range rows {
		if row.Key == key {
			return row, true
		}
	}
	return frc.BreakdownRow{}, false
}

// The order is the one a breakdown is read in: what they scored, whether it
// earned them anything, where the points came from, what it cost them in
// penalties, and then the detail.
func TestCompareBreakdownsOrdersTheImportantFieldsFirst(t *testing.T) {
	rows := frc.CompareBreakdowns(mustMatch(t, compare2024JSON), false)
	got := labelsOf(rows)

	want := []string{
		"Total Points", "RP",
		"Auto Amp Note Points", "Auto Leave Points", "Auto Points",
		"Auto Speaker Note Points", "End Game On Stage Points", "Foul Points",
		"Teleop Points",
		"Foul Count",
		"Auto Amp Note Count", "Auto Line Robot 1", "Auto Speaker Note Count",
		"End Game Robot 1", "Melody Bonus Achieved",
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("CompareBreakdowns order =\n%v\nwant\n%v", got, want)
	}
}

// Both alliances' values are on one row, which is the whole point: a breakdown
// is read across, not down two lists forty rows apart.
func TestCompareBreakdownsPairsTheAlliances(t *testing.T) {
	rows := frc.CompareBreakdowns(mustMatch(t, compare2024JSON), false)
	row, ok := rowFor(rows, "totalPoints")
	if !ok {
		t.Fatalf("no totalPoints row in %v", labelsOf(rows))
	}
	if row.Red != "88" || row.Blue != "61" {
		t.Errorf("totalPoints = (%q, %q), want (88, 61)", row.Red, row.Blue)
	}
	if row, _ := rowFor(rows, "melodyBonusAchieved"); row.Red != "yes" || row.Blue != "no" {
		t.Errorf("melodyBonusAchieved = (%q, %q), want (yes, no)", row.Red, row.Blue)
	}
}

// Most of a 2024 breakdown is zero for both alliances. Those rows are noise,
// and drowning the dozen that moved in them is what made the old dump
// unreadable.
func TestCompareBreakdownsDropsRowsNeitherAllianceScored(t *testing.T) {
	rows := frc.CompareBreakdowns(mustMatch(t, compare2024JSON), false)
	for _, key := range []string{
		"adjustPoints", "techFoulCount", "endGameNoteInTrapPoints",
		"coopertitionBonusAchieved", "g424Penalty", "trapCenterStage",
	} {
		if _, ok := rowFor(rows, key); ok {
			t.Errorf("%s is zero on both sides and should have been dropped", key)
		}
	}
	// A value that is equal but not nothing stays: both alliances leaving the
	// line is a fact about the match.
	if _, ok := rowFor(rows, "autoLineRobot1"); !ok {
		t.Error("autoLineRobot1 is Yes on both sides and should have been kept")
	}
	// So does one that is zero on one side only.
	if row, ok := rowFor(rows, "foulCount"); !ok || row.Red != "1" || row.Blue != "0" {
		t.Errorf("foulCount = %v, want it kept as (1, 0)", row)
	}
}

// --full is for the moment you want the row that says nothing happened.
func TestCompareBreakdownsFullKeepsEverything(t *testing.T) {
	rows := frc.CompareBreakdowns(mustMatch(t, compare2024JSON), true)
	if len(rows) != 21 {
		t.Errorf("got %d rows, want all 21: %v", len(rows), labelsOf(rows))
	}
	for _, key := range []string{"adjustPoints", "trapCenterStage", "techFoulCount"} {
		if _, ok := rowFor(rows, key); !ok {
			t.Errorf("--full should have kept %s", key)
		}
	}
	// The penalties still follow every scoring column and lead the detail,
	// rather than sorting into the middle of either.
	got := labelsOf(rows)
	penalties := indexOf(got, "Foul Count")
	if got[penalties+1] != "Tech Foul Count" || got[penalties+2] != "Adjust Points" {
		t.Errorf("the penalties are not together in %v", got)
	}
	if last := indexOf(got, "Teleop Points"); last > penalties {
		t.Errorf("a scoring column sorted after the penalties in %v", got)
	}
	if first := indexOf(got, "Auto Amp Note Count"); first != penalties+3 {
		t.Errorf("the detail does not follow the penalties in %v", got)
	}
}

func indexOf(labels []string, want string) int {
	for i, label := range labels {
		if label == want {
			return i
		}
	}
	return -1
}

// A match with no breakdown at all — anything before 2015, or anything not yet
// played — has no table rather than an empty one.
func TestCompareBreakdownsWithoutABreakdown(t *testing.T) {
	if got := frc.CompareBreakdowns(mustMatch(t, unplayed2024JSON), false); got != nil {
		t.Errorf("CompareBreakdowns = %v, want nothing", got)
	}
}

// One alliance's breakdown missing is still worth a table: the other one's is
// the answer, and inventing zeroes for the missing side would be a lie.
func TestCompareBreakdownsWithOneSideOnly(t *testing.T) {
	m := api.Match{
		Key: "2024cthar_qm12",
		ScoreBreakdown: map[string]interface{}{
			"red": map[string]interface{}{"totalPoints": 88.0},
		},
	}
	rows := frc.CompareBreakdowns(m, false)
	row, ok := rowFor(rows, "totalPoints")
	if !ok {
		t.Fatalf("no totalPoints row in %v", labelsOf(rows))
	}
	if row.Red != "88" || row.Blue != "" {
		t.Errorf("totalPoints = (%q, %q), want (88, empty)", row.Red, row.Blue)
	}
}

func TestHumanizeKey(t *testing.T) {
	cases := map[string]string{
		"autoAmpNoteCount":                 "Auto Amp Note Count",
		"totalPoints":                      "Total Points",
		"rp":                               "RP",
		"endGameRobot1":                    "End Game Robot 1",
		"g424Penalty":                      "G424 Penalty",
		"teleopSpeakerNoteAmplifiedPoints": "Teleop Speaker Note Amplified Points",
		"autoCommunity.B.1":                "Auto Community B 1",
		"":                                 "",
	}
	for key, want := range cases {
		if got := frc.HumanizeKey(key); got != want {
			t.Errorf("HumanizeKey(%q) = %q, want %q", key, got, want)
		}
	}
}
