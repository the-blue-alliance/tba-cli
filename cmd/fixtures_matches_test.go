package cmd

// Match fixtures for the match listings and `match view`. They use the real
// field names and shapes returned by The Blue Alliance API v3.

// A 2024 district event's matches, deliberately not in playing order so that
// the listings have something to sort. 2024cthar ran a double-elimination
// bracket (playoff_type 10), so its semifinal sets hold one match each.
//
// Between them these cover every case a listing has to render: a played match,
// an unplayed one, a tie, a surrogate, a disqualification, a playoff set and a
// finals series.
const matches2024ctharJSON = `[
  {
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
  },
  {
    "key": "2024cthar_qm12",
    "comp_level": "qm",
    "set_number": 1,
    "match_number": 12,
    "event_key": "2024cthar",
    "time": 1711130400,
    "predicted_time": 1711130700,
    "actual_time": 1711130820,
    "winning_alliance": "red",
    "alliances": {
      "red": {"score": 88, "team_keys": ["frc177", "frc1073", "frc5507"], "surrogate_team_keys": [], "dq_team_keys": []},
      "blue": {"score": 61, "team_keys": ["frc230", "frc1071", "frc4055"], "surrogate_team_keys": ["frc4055"], "dq_team_keys": []}
    },
    "score_breakdown": {"red": {"totalPoints": 88}, "blue": {"totalPoints": 61}},
    "videos": [{"type": "youtube", "key": "dQw4w9WgXcQ"}]
  },
  {
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
    "score_breakdown": null,
    "videos": []
  },
  {
    "key": "2024cthar_qm2",
    "comp_level": "qm",
    "set_number": 1,
    "match_number": 2,
    "event_key": "2024cthar",
    "time": 1711121400,
    "predicted_time": 1711122000,
    "actual_time": null,
    "winning_alliance": "",
    "alliances": {
      "red": {"score": -1, "team_keys": ["frc558", "frc3467", "frc2168"], "surrogate_team_keys": [], "dq_team_keys": []},
      "blue": {"score": -1, "team_keys": ["frc195", "frc1124", "frc6153"], "surrogate_team_keys": [], "dq_team_keys": []}
    },
    "score_breakdown": null,
    "videos": []
  },
  {
    "key": "2024cthar_qm7",
    "comp_level": "qm",
    "set_number": 1,
    "match_number": 7,
    "event_key": "2024cthar",
    "time": 1711126800,
    "predicted_time": 1711126800,
    "actual_time": 1711126920,
    "winning_alliance": "",
    "alliances": {
      "red": {"score": 44, "team_keys": ["frc1071", "frc4055", "frc6153"], "surrogate_team_keys": [], "dq_team_keys": []},
      "blue": {"score": 44, "team_keys": ["frc230", "frc195", "frc558"], "surrogate_team_keys": [], "dq_team_keys": []}
    },
    "score_breakdown": null,
    "videos": []
  },
  {
    "key": "2024cthar_qm3",
    "comp_level": "qm",
    "set_number": 1,
    "match_number": 3,
    "event_key": "2024cthar",
    "time": 1711123200,
    "predicted_time": 1711123200,
    "actual_time": 1711123380,
    "winning_alliance": "blue",
    "alliances": {
      "red": {"score": 31, "team_keys": ["frc177", "frc3467", "frc2168"], "surrogate_team_keys": [], "dq_team_keys": ["frc2168"]},
      "blue": {"score": 77, "team_keys": ["frc1073", "frc1124", "frc6153"], "surrogate_team_keys": [], "dq_team_keys": []}
    },
    "score_breakdown": null,
    "videos": []
  }
]`

// The subset of 2024cthar's matches that team 177 played in.
const teamMatches177At2024ctharJSON = `[
  {
    "key": "2024cthar_qm12",
    "comp_level": "qm",
    "set_number": 1,
    "match_number": 12,
    "event_key": "2024cthar",
    "time": 1711130400,
    "predicted_time": 1711130700,
    "actual_time": 1711130820,
    "winning_alliance": "red",
    "alliances": {
      "red": {"score": 88, "team_keys": ["frc177", "frc1073", "frc5507"], "surrogate_team_keys": [], "dq_team_keys": []},
      "blue": {"score": 61, "team_keys": ["frc230", "frc1071", "frc4055"], "surrogate_team_keys": ["frc4055"], "dq_team_keys": []}
    },
    "score_breakdown": null,
    "videos": []
  },
  {
    "key": "2024cthar_sf13m1",
    "comp_level": "sf",
    "set_number": 13,
    "match_number": 1,
    "event_key": "2024cthar",
    "time": 1711299600,
    "predicted_time": 1711299600,
    "actual_time": null,
    "winning_alliance": "",
    "alliances": {
      "red": {"score": -1, "team_keys": ["frc177", "frc1073", "frc5507"], "surrogate_team_keys": [], "dq_team_keys": []},
      "blue": {"score": -1, "team_keys": ["frc230", "frc195", "frc558"], "surrogate_team_keys": [], "dq_team_keys": []}
    },
    "score_breakdown": null,
    "videos": []
  }
]`

// A 2019 event, from the era of best-of-three bracket sets: its playoff
// matches are named "QF 4-3" rather than "QF 4".
const event2019ctwatJSON = `{
  "key": "2019ctwat",
  "name": "NE District Waterbury Event",
  "event_code": "ctwat",
  "event_type": 1,
  "city": "Waterbury",
  "state_prov": "CT",
  "country": "USA",
  "start_date": "2019-03-15",
  "end_date": "2019-03-17",
  "year": 2019,
  "short_name": "Waterbury",
  "event_type_string": "District",
  "week": 2,
  "address": "1 Waterbury Ln, Waterbury, CT 06702, USA",
  "location_name": "Waterbury Arena",
  "webcasts": [],
  "playoff_type": 0
}`

const matches2019ctwatJSON = `[
  {
    "key": "2019ctwat_qf1m1",
    "comp_level": "qf",
    "set_number": 1,
    "match_number": 1,
    "event_key": "2019ctwat",
    "time": 1552849800,
    "predicted_time": 1552849800,
    "actual_time": 1552849980,
    "winning_alliance": "red",
    "alliances": {
      "red": {"score": 60, "team_keys": ["frc177", "frc1073", "frc5507"], "surrogate_team_keys": [], "dq_team_keys": []},
      "blue": {"score": 48, "team_keys": ["frc230", "frc195", "frc558"], "surrogate_team_keys": [], "dq_team_keys": []}
    },
    "score_breakdown": null,
    "videos": []
  },
  {
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
      "red": {"score": 55, "team_keys": ["frc1071", "frc4055", "frc2168"], "surrogate_team_keys": [], "dq_team_keys": []},
      "blue": {"score": 62, "team_keys": ["frc3467", "frc1124", "frc6153"], "surrogate_team_keys": [], "dq_team_keys": []}
    },
    "score_breakdown": null,
    "videos": []
  },
  {
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
  },
  {
    "key": "2019ctwat_f1m1",
    "comp_level": "f",
    "set_number": 1,
    "match_number": 1,
    "event_key": "2019ctwat",
    "time": 1552860000,
    "predicted_time": 1552860000,
    "actual_time": 1552860180,
    "winning_alliance": "red",
    "alliances": {
      "red": {"score": 82, "team_keys": ["frc177", "frc1073", "frc5507"], "surrogate_team_keys": [], "dq_team_keys": []},
      "blue": {"score": 79, "team_keys": ["frc3467", "frc1124", "frc6153"], "surrogate_team_keys": [], "dq_team_keys": []}
    },
    "score_breakdown": null,
    "videos": []
  }
]`

// A 2021 remote event (event_type 7). The remote season ran no head-to-head
// matches at all, so every listing has to survive an empty result.
const event2021ctwatJSON = `{
  "key": "2021ctwat",
  "name": "New England FIRST Remote Event",
  "event_code": "ctwat",
  "event_type": 7,
  "city": "Hartford",
  "state_prov": "CT",
  "country": "USA",
  "start_date": "2021-03-20",
  "end_date": "2021-03-21",
  "year": 2021,
  "short_name": "New England Remote",
  "event_type_string": "Remote",
  "week": 2,
  "address": null,
  "location_name": null,
  "webcasts": [],
  "playoff_type": null
}`

// One match with everything a detail view can show: a surrogate, a score
// breakdown on both alliances, and a video.
const matchViewQM12JSON = `{
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
    "red": {"score": 88, "team_keys": ["frc177", "frc1073", "frc5507"], "surrogate_team_keys": [], "dq_team_keys": []},
    "blue": {"score": 61, "team_keys": ["frc230", "frc1071", "frc4055"], "surrogate_team_keys": ["frc4055"], "dq_team_keys": []}
  },
  "score_breakdown": {
    "red": {"autoPoints": 20, "totalPoints": 88, "melody": true},
    "blue": {"autoPoints": 10, "totalPoints": 61, "melody": false}
  },
  "videos": [{"type": "youtube", "key": "dQw4w9WgXcQ"}]
}`

// A 2026-shaped breakdown: recent seasons write an unfilled slot as the string
// "None" rather than as a null, and carry the game's own constants -- the
// thresholds a bonus is measured against -- alongside what the alliances did.
const matchView2026JSON = `{
  "key": "2026cthar_qm7",
  "comp_level": "qm",
  "set_number": 1,
  "match_number": 7,
  "event_key": "2026cthar",
  "time": 1774531200,
  "actual_time": 1774531500,
  "winning_alliance": "red",
  "alliances": {
    "red": {"score": 96, "team_keys": ["frc177", "frc1073", "frc5507"], "surrogate_team_keys": [], "dq_team_keys": []},
    "blue": {"score": 74, "team_keys": ["frc230", "frc195", "frc558"], "surrogate_team_keys": [], "dq_team_keys": []}
  },
  "score_breakdown": {
    "red": {
      "totalPoints": 96,
      "rp": 3,
      "autoPoints": 24,
      "foulCount": 0,
      "autoTowerRobot1": "None",
      "autoTowerRobot2": "None",
      "endgameRobot1": "Parked",
      "coopertitionThreshold": 12,
      "ensembleBonusThreshold": 30
    },
    "blue": {
      "totalPoints": 74,
      "rp": 1,
      "autoPoints": 12,
      "foulCount": 0,
      "autoTowerRobot1": "None",
      "autoTowerRobot2": "None",
      "endgameRobot1": "None",
      "coopertitionThreshold": 12,
      "ensembleBonusThreshold": 30
    }
  },
  "videos": []
}`

// A semifinal, whose name depends on the event's bracket.
const matchViewSF13JSON = `{
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
  "score_breakdown": null,
  "videos": []
}`

// A match that has not been played: both scores -1, no winner, no breakdown.
const matchViewUnplayedJSON = `{
  "key": "2024cthar_qm40",
  "comp_level": "qm",
  "set_number": 1,
  "match_number": 40,
  "event_key": "2024cthar",
  "time": 1711220400,
  "predicted_time": 1711221000,
  "actual_time": null,
  "winning_alliance": "",
  "alliances": {
    "red": {"score": -1, "team_keys": ["frc558", "frc3467", "frc2168"], "surrogate_team_keys": [], "dq_team_keys": []},
    "blue": {"score": -1, "team_keys": ["frc195", "frc1124", "frc6153"], "surrogate_team_keys": [], "dq_team_keys": []}
  },
  "score_breakdown": null,
  "videos": []
}`

// A match whose video is hosted somewhere TBA knows but this CLI does not have
// a URL shape for.
const matchViewTBAVideoJSON = `{
  "key": "2024cthar_qm41",
  "comp_level": "qm",
  "set_number": 1,
  "match_number": 41,
  "event_key": "2024cthar",
  "time": 1711224000,
  "predicted_time": 1711224000,
  "actual_time": 1711224180,
  "winning_alliance": "blue",
  "alliances": {
    "red": {"score": 40, "team_keys": ["frc558", "frc3467", "frc2168"], "surrogate_team_keys": [], "dq_team_keys": []},
    "blue": {"score": 66, "team_keys": ["frc195", "frc1124", "frc6153"], "surrogate_team_keys": [], "dq_team_keys": []}
  },
  "score_breakdown": null,
  "videos": [{"type": "tba", "key": "abc123"}]
}`

// A 2015 qualification match. That season scored both alliances the same in a
// coopertition match and reported no winner, and it predates predicted times.
const match2015ctwatQM7JSON = `{
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

// A team's whole 2024 season, as /team/frc177/matches/2024 returns it: two
// district events whose qualification numbers collide, in the arbitrary order
// the API answers with. Sorting this as one list interleaves the events —
// Qual 46 at Waterbury next to Qual 46 at Hartford — which is what grouping by
// event exists to prevent.
const teamMatches177Season2024JSON = `[
  {
    "key": "2024cthar_qm46",
    "comp_level": "qm",
    "set_number": 1,
    "match_number": 46,
    "event_key": "2024cthar",
    "time": 1711211400,
    "predicted_time": 1711211400,
    "actual_time": 1711211580,
    "winning_alliance": "red",
    "alliances": {
      "red": {"score": 94, "team_keys": ["frc177", "frc1073", "frc5507"], "surrogate_team_keys": [], "dq_team_keys": []},
      "blue": {"score": 70, "team_keys": ["frc230", "frc195", "frc558"], "surrogate_team_keys": [], "dq_team_keys": []}
    },
    "score_breakdown": null,
    "videos": []
  },
  {
    "key": "2024ctwat_qm46",
    "comp_level": "qm",
    "set_number": 1,
    "match_number": 46,
    "event_key": "2024ctwat",
    "time": 1709996400,
    "predicted_time": 1709996400,
    "actual_time": 1709996640,
    "winning_alliance": "blue",
    "alliances": {
      "red": {"score": 61, "team_keys": ["frc1071", "frc4055", "frc6153"], "surrogate_team_keys": [], "dq_team_keys": []},
      "blue": {"score": 83, "team_keys": ["frc177", "frc1124", "frc2168"], "surrogate_team_keys": [], "dq_team_keys": []}
    },
    "score_breakdown": null,
    "videos": []
  },
  {
    "key": "2024cthar_qm12",
    "comp_level": "qm",
    "set_number": 1,
    "match_number": 12,
    "event_key": "2024cthar",
    "time": 1711130400,
    "predicted_time": 1711130700,
    "actual_time": 1711130820,
    "winning_alliance": "red",
    "alliances": {
      "red": {"score": 88, "team_keys": ["frc177", "frc1073", "frc5507"], "surrogate_team_keys": [], "dq_team_keys": []},
      "blue": {"score": 61, "team_keys": ["frc230", "frc1071", "frc4055"], "surrogate_team_keys": ["frc4055"], "dq_team_keys": []}
    },
    "score_breakdown": null,
    "videos": []
  },
  {
    "key": "2024ctwat_qm5",
    "comp_level": "qm",
    "set_number": 1,
    "match_number": 5,
    "event_key": "2024ctwat",
    "time": 1709913600,
    "predicted_time": 1709913600,
    "actual_time": 1709913780,
    "winning_alliance": "red",
    "alliances": {
      "red": {"score": 70, "team_keys": ["frc177", "frc1073", "frc5507"], "surrogate_team_keys": [], "dq_team_keys": []},
      "blue": {"score": 54, "team_keys": ["frc230", "frc195", "frc558"], "surrogate_team_keys": [], "dq_team_keys": []}
    },
    "score_breakdown": null,
    "videos": []
  }
]`

// A 2024 Crescendo qualification match with both score breakdowns, shortened
// from a real one but keeping its shape: a total, ranking points, the point
// columns, the fouls, the counts, and the flags neither alliance earned, which
// is most of a real breakdown.
const matchViewBreakdown2024JSON = `{
  "key": "2024cthar_qm18",
  "comp_level": "qm",
  "set_number": 1,
  "match_number": 18,
  "event_key": "2024cthar",
  "time": 1711136400,
  "predicted_time": 1711136400,
  "actual_time": 1711136640,
  "winning_alliance": "red",
  "alliances": {
    "red": {"score": 88, "team_keys": ["frc177", "frc1073", "frc5507"], "surrogate_team_keys": [], "dq_team_keys": []},
    "blue": {"score": 61, "team_keys": ["frc230", "frc195", "frc558"], "surrogate_team_keys": [], "dq_team_keys": []}
  },
  "score_breakdown": {
    "red": {
      "adjustPoints": 0,
      "autoAmpNoteCount": 1,
      "autoAmpNotePoints": 2,
      "autoLeavePoints": 6,
      "autoLineRobot1": "Yes",
      "autoLineRobot2": "Yes",
      "autoLineRobot3": "Yes",
      "autoPoints": 23,
      "autoSpeakerNoteCount": 3,
      "autoSpeakerNotePoints": 15,
      "coopertitionBonusAchieved": false,
      "coopertitionCriteriaMet": false,
      "endGameHarmonyPoints": 2,
      "endGameNoteInTrapPoints": 0,
      "endGameOnStagePoints": 6,
      "endGameParkPoints": 0,
      "endGameRobot1": "StageLeft",
      "endGameTotalStagePoints": 8,
      "ensembleBonusAchieved": true,
      "foulCount": 1,
      "foulPoints": 2,
      "g424Penalty": false,
      "melodyBonusAchieved": true,
      "micCenterStage": false,
      "micStageLeft": true,
      "rp": 5,
      "techFoulCount": 0,
      "teleopAmpNoteCount": 9,
      "teleopAmpNotePoints": 9,
      "teleopPoints": 57,
      "teleopSpeakerNoteAmplifiedCount": 8,
      "teleopSpeakerNoteAmplifiedPoints": 40,
      "teleopSpeakerNoteCount": 4,
      "teleopTotalNotePoints": 57,
      "totalPoints": 88,
      "trapCenterStage": false
    },
    "blue": {
      "adjustPoints": 0,
      "autoAmpNoteCount": 0,
      "autoAmpNotePoints": 0,
      "autoLeavePoints": 4,
      "autoLineRobot1": "Yes",
      "autoLineRobot2": "Yes",
      "autoLineRobot3": "No",
      "autoPoints": 14,
      "autoSpeakerNoteCount": 2,
      "autoSpeakerNotePoints": 10,
      "coopertitionBonusAchieved": false,
      "coopertitionCriteriaMet": false,
      "endGameHarmonyPoints": 0,
      "endGameNoteInTrapPoints": 0,
      "endGameOnStagePoints": 3,
      "endGameParkPoints": 1,
      "endGameRobot1": "Parked",
      "endGameTotalStagePoints": 4,
      "ensembleBonusAchieved": false,
      "foulCount": 0,
      "foulPoints": 0,
      "g424Penalty": false,
      "melodyBonusAchieved": false,
      "micCenterStage": false,
      "micStageLeft": false,
      "rp": 1,
      "techFoulCount": 0,
      "teleopAmpNoteCount": 5,
      "teleopAmpNotePoints": 5,
      "teleopPoints": 43,
      "teleopSpeakerNoteAmplifiedCount": 6,
      "teleopSpeakerNoteAmplifiedPoints": 30,
      "teleopSpeakerNoteCount": 4,
      "teleopTotalNotePoints": 43,
      "totalPoints": 61,
      "trapCenterStage": false
    }
  },
  "videos": []
}`
