package cmd

// Fixtures for `team next` and `team standing`, shaped like the API's
// /team/{key}/events/{year} and /team/{key}/event/{key}/status responses.

// Team 177's 2024 season: three events, out of order, as the API returns them.
const teamEvents177In2024JSON = `[
  {
    "key": "2024necmp",
    "name": "New England FIRST District Championship",
    "event_code": "necmp",
    "event_type": 2,
    "city": "Springfield",
    "state_prov": "MA",
    "country": "USA",
    "start_date": "2024-04-10",
    "end_date": "2024-04-13",
    "year": 2024,
    "short_name": "New England",
    "event_type_string": "District Championship",
    "week": 6,
    "webcasts": [],
    "playoff_type": 10
  },
  {
    "key": "2024cthar",
    "name": "NE District Hartford Event",
    "event_code": "cthar",
    "event_type": 1,
    "city": "Hartford",
    "state_prov": "CT",
    "country": "USA",
    "start_date": "2024-03-22",
    "end_date": "2024-03-24",
    "year": 2024,
    "short_name": "Hartford",
    "event_type_string": "District",
    "week": 3,
    "webcasts": [],
    "playoff_type": 10
  },
  {
    "key": "2024ctwat",
    "name": "NE District Waterbury Event",
    "event_code": "ctwat",
    "event_type": 1,
    "city": "Waterbury",
    "state_prov": "CT",
    "country": "USA",
    "start_date": "2024-03-08",
    "end_date": "2024-03-10",
    "year": 2024,
    "short_name": "Waterbury",
    "event_type_string": "District",
    "week": 1,
    "webcasts": [],
    "playoff_type": 10
  }
]`

// A finished event: every match this team played is done, so there is no next
// match to report.
const teamMatches177FinishedJSON = `[
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

// A completed event status: ranked first, alliance captain, won the finals.
const teamStatus177At2024ctharJSON = `{
  "qual": {
    "num_teams": 40,
    "ranking": {
      "dq": 0,
      "matches_played": 12,
      "qual_average": null,
      "rank": 1,
      "record": {"losses": 2, "ties": 0, "wins": 10},
      "sort_orders": [2.5, 88.0, 31.0],
      "team_key": "frc177"
    },
    "sort_order_info": [
      {"name": "Ranking Score", "precision": 2},
      {"name": "Avg Match", "precision": 0},
      {"name": "Avg Auto", "precision": 1}
    ],
    "status": "completed"
  },
  "alliance": {
    "backup": null,
    "name": "Alliance 1",
    "number": 1,
    "pick": 0
  },
  "playoff": {
    "current_level_record": {"losses": 0, "ties": 0, "wins": 2},
    "level": "f",
    "playoff_average": null,
    "record": {"losses": 1, "ties": 0, "wins": 6},
    "status": "won"
  },
  "alliance_status_str": "<b>Rank 1</b> with a record of <b>10-2-0</b>",
  "playoff_status_str": "won the <b>Finals</b>",
  "overall_status_str": "Team 177 was <b>Rank 1</b> with a record of <b>10-2-0</b> and <b>won</b> the event.",
  "next_match_key": null,
  "last_match_key": "2024cthar_f1m2"
}`

// A status from before the event starts: nothing has happened yet, so every
// section is null.
const teamStatusNotStartedJSON = `{
  "qual": null,
  "alliance": null,
  "playoff": null,
  "alliance_status_str": "",
  "playoff_status_str": "",
  "overall_status_str": "",
  "next_match_key": null,
  "last_match_key": null
}`

// A status mid-event: ranked, but alliance selection has not happened.
const teamStatusQualsOnlyJSON = `{
  "qual": {
    "num_teams": 40,
    "ranking": {
      "dq": 0,
      "matches_played": 6,
      "qual_average": null,
      "rank": 7,
      "record": {"losses": 2, "ties": 1, "wins": 3},
      "sort_orders": [1.8333, 62.0],
      "team_key": "frc177"
    },
    "sort_order_info": [
      {"name": "Ranking Score", "precision": 2},
      {"name": "Avg Match", "precision": 0}
    ],
    "status": "playing"
  },
  "alliance": null,
  "playoff": null,
  "alliance_status_str": "",
  "playoff_status_str": "",
  "overall_status_str": "Team 177 is <b>Rank 7</b> with a record of <b>3-2-1</b>.",
  "next_match_key": "2024cthar_qm40",
  "last_match_key": "2024cthar_qm33"
}`

// A status for a team that joined an alliance as a backup.
const teamStatusBackupJSON = `{
  "qual": null,
  "alliance": {
    "backup": {"in": "frc177", "out": "frc230"},
    "name": "Alliance 3",
    "number": 3,
    "pick": 2
  },
  "playoff": {
    "current_level_record": {"losses": 2, "ties": 0, "wins": 1},
    "level": "sf",
    "playoff_average": null,
    "record": {"losses": 2, "ties": 0, "wins": 3},
    "status": "eliminated"
  },
  "alliance_status_str": "",
  "playoff_status_str": "",
  "overall_status_str": "",
  "next_match_key": null,
  "last_match_key": null
}`

// The Waterbury district event of 2024, the one team 177 finished before
// Hartford started.
const event2024ctwatJSON = `{
  "key": "2024ctwat",
  "name": "NE District Waterbury Event",
  "event_code": "ctwat",
  "event_type": 1,
  "city": "Waterbury",
  "state_prov": "CT",
  "country": "USA",
  "start_date": "2024-03-08",
  "end_date": "2024-03-10",
  "year": 2024,
  "short_name": "Waterbury",
  "event_type_string": "District",
  "week": 1,
  "webcasts": [],
  "playoff_type": 10
}`
