package frc_test

import (
	"encoding/json"
	"testing"

	"github.com/the-blue-alliance/tba-cli/internal/api"
)

// The fixtures below are shaped exactly like The Blue Alliance API v3's
// /match responses, including the fields this package ignores, so that a
// change to the struct tags shows up here.

// A 2024 qualification match: played, red wins, a surrogate on blue.
const qual2024JSON = `{
  "key": "2024cthar_qm12",
  "comp_level": "qm",
  "set_number": 1,
  "match_number": 12,
  "event_key": "2024cthar",
  "time": 1711130400,
  "predicted_time": 1711130700,
  "actual_time": 1711130820,
  "post_result_time": 1711130900,
  "winning_alliance": "red",
  "alliances": {
    "red": {
      "score": 88,
      "team_keys": ["frc177", "frc1073", "frc5507"],
      "surrogate_team_keys": [],
      "dq_team_keys": []
    },
    "blue": {
      "score": 61,
      "team_keys": ["frc230", "frc1071", "frc4055"],
      "surrogate_team_keys": ["frc4055"],
      "dq_team_keys": []
    }
  },
  "score_breakdown": {"red": {"totalPoints": 88}, "blue": {"totalPoints": 61}},
  "videos": [{"type": "youtube", "key": "dQw4w9WgXcQ"}]
}`

// An unplayed 2024 qualification match: both scores -1, no winner, and only a
// scheduled and predicted time.
const unplayed2024JSON = `{
  "key": "2024cthar_qm40",
  "comp_level": "qm",
  "set_number": 1,
  "match_number": 40,
  "event_key": "2024cthar",
  "time": 1711220400,
  "predicted_time": 1711221000,
  "actual_time": null,
  "post_result_time": null,
  "winning_alliance": "",
  "alliances": {
    "red": {"score": -1, "team_keys": ["frc558", "frc3467", "frc2168"], "surrogate_team_keys": [], "dq_team_keys": []},
    "blue": {"score": -1, "team_keys": ["frc195", "frc1124", "frc6153"], "surrogate_team_keys": [], "dq_team_keys": []}
  },
  "score_breakdown": null,
  "videos": []
}`

// A 2024 double-elimination semifinal. Every semifinal set holds one match, so
// sf13m1 is the thirteenth semifinal.
const doubleElimSF13JSON = `{
  "key": "2024cthar_sf13m1",
  "comp_level": "sf",
  "set_number": 13,
  "match_number": 1,
  "event_key": "2024cthar",
  "time": 1711299600,
  "predicted_time": 1711299600,
  "actual_time": 1711299780,
  "winning_alliance": "blue",
  "alliances": {
    "red": {"score": 102, "team_keys": ["frc177", "frc1073", "frc5507"], "surrogate_team_keys": [], "dq_team_keys": []},
    "blue": {"score": 118, "team_keys": ["frc230", "frc195", "frc558"], "surrogate_team_keys": [], "dq_team_keys": []}
  },
  "score_breakdown": {"red": {"totalPoints": 102}, "blue": {"totalPoints": 118}},
  "videos": []
}`

// The second match of a 2024 finals series.
const doubleElimFinal2JSON = `{
  "key": "2024cthar_f1m2",
  "comp_level": "f",
  "set_number": 1,
  "match_number": 2,
  "event_key": "2024cthar",
  "time": 1711306800,
  "predicted_time": 1711306800,
  "actual_time": 1711307040,
  "winning_alliance": "red",
  "alliances": {
    "red": {"score": 131, "team_keys": ["frc177", "frc1073", "frc5507"], "surrogate_team_keys": [], "dq_team_keys": []},
    "blue": {"score": 127, "team_keys": ["frc230", "frc195", "frc558"], "surrogate_team_keys": [], "dq_team_keys": []}
  },
  "score_breakdown": {"red": {"totalPoints": 131}, "blue": {"totalPoints": 127}},
  "videos": []
}`

// A 2019 quarterfinal, from the era of best-of-three bracket sets.
const legacyQF2019JSON = `{
  "key": "2019ctwat_qf4m3",
  "comp_level": "qf",
  "set_number": 4,
  "match_number": 3,
  "event_key": "2019ctwat",
  "time": 1552853400,
  "predicted_time": 1552853400,
  "actual_time": 1552853580,
  "winning_alliance": "blue",
  "alliances": {
    "red": {"score": 55, "team_keys": ["frc1071", "frc4055", "frc2168"], "surrogate_team_keys": [], "dq_team_keys": ["frc2168"]},
    "blue": {"score": 62, "team_keys": ["frc177", "frc1073", "frc5507"], "surrogate_team_keys": [], "dq_team_keys": []}
  },
  "score_breakdown": {"red": {"totalPoints": 55}, "blue": {"totalPoints": 62}},
  "videos": []
}`

// A 2019 semifinal, likewise a best-of-three set.
const legacySF2019JSON = `{
  "key": "2019ctwat_sf1m2",
  "comp_level": "sf",
  "set_number": 1,
  "match_number": 2,
  "event_key": "2019ctwat",
  "time": 1552856400,
  "predicted_time": 1552856400,
  "actual_time": 1552856640,
  "winning_alliance": "red",
  "alliances": {
    "red": {"score": 71, "team_keys": ["frc177", "frc1073", "frc5507"], "surrogate_team_keys": [], "dq_team_keys": []},
    "blue": {"score": 44, "team_keys": ["frc230", "frc195", "frc558"], "surrogate_team_keys": [], "dq_team_keys": []}
  },
  "score_breakdown": null,
  "videos": []
}`

// A 2018 octofinal, the only era with an "ef" level at a district event.
const legacyEF2018JSON = `{
  "key": "2018ctwat_ef3m1",
  "comp_level": "ef",
  "set_number": 3,
  "match_number": 1,
  "event_key": "2018ctwat",
  "time": 1520700000,
  "predicted_time": 1520700000,
  "actual_time": 1520700180,
  "winning_alliance": "red",
  "alliances": {
    "red": {"score": 310, "team_keys": ["frc177", "frc1073", "frc5507"], "surrogate_team_keys": [], "dq_team_keys": []},
    "blue": {"score": 288, "team_keys": ["frc230", "frc195", "frc558"], "surrogate_team_keys": [], "dq_team_keys": []}
  },
  "score_breakdown": null,
  "videos": []
}`

// A 2015 qualification match. That season had a coopertition scoring model in
// which both alliances could score the same, and the API reports no winner;
// it also predates predicted_time.
const qual2015TieJSON = `{
  "key": "2015ctwat_qm7",
  "comp_level": "qm",
  "set_number": 1,
  "match_number": 7,
  "event_key": "2015ctwat",
  "time": 1427464800,
  "predicted_time": null,
  "actual_time": null,
  "winning_alliance": "",
  "alliances": {
    "red": {"score": 44, "team_keys": ["frc177", "frc1071", "frc2168"], "surrogate_team_keys": [], "dq_team_keys": []},
    "blue": {"score": 44, "team_keys": ["frc230", "frc195", "frc558"], "surrogate_team_keys": [], "dq_team_keys": []}
  },
  "score_breakdown": null,
  "videos": []
}`

// An Einstein round robin semifinal: one set, fifteen matches.
const roundRobin2019JSON = `{
  "key": "2019cmptx_sf1m3",
  "comp_level": "sf",
  "set_number": 1,
  "match_number": 3,
  "event_key": "2019cmptx",
  "time": 1555704000,
  "predicted_time": 1555704000,
  "actual_time": 1555704120,
  "winning_alliance": "red",
  "alliances": {
    "red": {"score": 80, "team_keys": ["frc177", "frc1073", "frc5507"], "surrogate_team_keys": [], "dq_team_keys": []},
    "blue": {"score": 76, "team_keys": ["frc230", "frc195", "frc558"], "surrogate_team_keys": [], "dq_team_keys": []}
  },
  "score_breakdown": null,
  "videos": []
}`

func mustMatch(t *testing.T, raw string) api.Match {
	t.Helper()
	var m api.Match
	if err := json.Unmarshal([]byte(raw), &m); err != nil {
		t.Fatalf("decoding fixture: %v", err)
	}
	return m
}

func intPtr(n int) *int { return &n }

func epoch(n int64) *int64 { return &n }
