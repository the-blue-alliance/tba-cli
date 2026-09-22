package cmd

// Fixtures use the real field names returned by The Blue Alliance API v3.

const teamFRC177JSON = `{
  "key": "frc177",
  "team_number": 177,
  "nickname": "Bobcat Robotics",
  "name": "Gordon & Llura Gund Foundation/RTX & South Windsor High School",
  "school_name": "South Windsor High School",
  "city": "South Windsor",
  "state_prov": "Connecticut",
  "country": "USA",
  "address": null,
  "postal_code": "06074",
  "website": "http://www.bobcatrobotics.org",
  "rookie_year": 1995,
  "motto": null
}`

const teamFRC1073JSON = `{
  "key": "frc1073",
  "team_number": 1073,
  "nickname": "The Force Team",
  "name": "Hollis Brookline High School",
  "city": "Hollis",
  "state_prov": "New Hampshire",
  "country": "USA",
  "website": "https://www.theforceteam.com",
  "rookie_year": 2003,
  "motto": null
}`

const teamFRC5507JSON = `{
  "key": "frc5507",
  "team_number": 5507,
  "nickname": "Robotic Eagles",
  "name": "Ellington High School",
  "city": "Ellington",
  "state_prov": "Connecticut",
  "country": "USA",
  "website": "https://www.frc5507.com",
  "rookie_year": 2015,
  "motto": null
}`

const event2024ctharJSON = `{
  "key": "2024cthar",
  "name": "NE District Hartford Event",
  "event_code": "cthar",
  "event_type": 1,
  "district": {
    "abbreviation": "ne",
    "display_name": "New England",
    "key": "2024ne",
    "year": 2024
  },
  "city": "Hartford",
  "state_prov": "CT",
  "country": "USA",
  "start_date": "2024-03-22",
  "end_date": "2024-03-24",
  "year": 2024,
  "short_name": "Hartford",
  "event_type_string": "District",
  "week": 3,
  "address": "55 Forest St, Hartford, CT 06105, USA",
  "location_name": "Hartford Public High School",
  "webcasts": [{"type": "twitch", "channel": "nefirst_red"}],
  "playoff_type": 10
}`

const event2024necmpJSON = `{
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
  "address": "1277 Main St, Springfield, MA 01103, USA",
  "location_name": "MassMutual Center",
  "webcasts": [],
  "playoff_type": 10
}`

const match2024ctharQM1JSON = `{
  "key": "2024cthar_qm1",
  "comp_level": "qm",
  "set_number": 1,
  "match_number": 1,
  "event_key": "2024cthar",
  "time": 1711120800,
  "predicted_time": 1711120800,
  "actual_time": 1711120920,
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
      "surrogate_team_keys": [],
      "dq_team_keys": []
    }
  },
  "score_breakdown": {"red": {"totalPoints": 88}, "blue": {"totalPoints": 61}},
  "videos": [{"type": "youtube", "key": "dQw4w9WgXcQ"}]
}`

const match2024ctharQM2JSON = `{
  "key": "2024cthar_qm2",
  "comp_level": "qm",
  "set_number": 1,
  "match_number": 2,
  "event_key": "2024cthar",
  "winning_alliance": "blue",
  "alliances": {
    "red": {"score": 45, "team_keys": ["frc558", "frc3467", "frc2168"], "surrogate_team_keys": [], "dq_team_keys": []},
    "blue": {"score": 72, "team_keys": ["frc195", "frc1124", "frc6153"], "surrogate_team_keys": [], "dq_team_keys": []}
  },
  "score_breakdown": null,
  "videos": []
}`

const rankings2024ctharJSON = `{
  "rankings": [
    {
      "team_key": "frc1073",
      "rank": 2,
      "record": {"wins": 9, "losses": 3, "ties": 0},
      "qual_average": null,
      "matches_played": 12,
      "dq": 1,
      "sort_orders": [2.3333, 0.0, 80.25, 28.5, 15.0],
      "extra_stats": [28]
    },
    {
      "team_key": "frc177",
      "rank": 1,
      "record": {"wins": 10, "losses": 2, "ties": 0},
      "qual_average": null,
      "matches_played": 12,
      "dq": 0,
      "sort_orders": [2.5, 0.25, 88.0, 31.0, 18.0],
      "extra_stats": [30]
    },
    {
      "team_key": "frc5507",
      "rank": 3,
      "record": {"wins": 7, "losses": 5, "ties": 0},
      "qual_average": null,
      "matches_played": 12,
      "dq": 0,
      "sort_orders": [1.9166, 0.0, 66.0, 20.0, 9.0],
      "extra_stats": [23]
    }
  ],
  "sort_order_info": [
    {"name": "Ranking Score", "precision": 2},
    {"name": "Avg Coop", "precision": 2},
    {"name": "Avg Match", "precision": 2},
    {"name": "Avg Auto", "precision": 2},
    {"name": "Avg Stage", "precision": 2}
  ],
  "extra_stats_info": [{"name": "Total Ranking Points", "precision": 0}]
}`

// 2015 had no win/loss record and ranked on a qualification average instead.
const rankings2015ctwatJSON = `{
  "rankings": [
    {
      "team_key": "frc177",
      "rank": 1,
      "record": null,
      "qual_average": 78.5,
      "matches_played": 8,
      "dq": 0,
      "sort_orders": [78.5, 20.0, 0.0, 40.0, 0.0, 18.5]
    },
    {
      "team_key": "frc1073",
      "rank": 2,
      "record": null,
      "qual_average": 71.25,
      "matches_played": 8,
      "dq": 1,
      "sort_orders": [71.25, 16.0, 0.0, 40.0, 0.0, 15.25]
    }
  ],
  "sort_order_info": [
    {"name": "Qual Avg", "precision": 2},
    {"name": "Auto", "precision": 0},
    {"name": "Container", "precision": 0},
    {"name": "Coopertition", "precision": 0},
    {"name": "Litter", "precision": 0},
    {"name": "Tote", "precision": 2}
  ],
  "extra_stats_info": []
}`

const teamsSimple2024ctharJSON = `[
  {"key": "frc177", "team_number": 177, "nickname": "Bobcat Robotics", "name": "Gordon & Llura Gund Foundation/RTX & South Windsor High School", "city": "South Windsor", "state_prov": "Connecticut", "country": "USA"},
  {"key": "frc1073", "team_number": 1073, "nickname": "The Force Team", "name": "Hollis Brookline High School", "city": "Hollis", "state_prov": "New Hampshire", "country": "USA"},
  {"key": "frc5507", "team_number": 5507, "nickname": "Robotic Eagles", "name": "Ellington High School", "city": "Ellington", "state_prov": "Connecticut", "country": "USA"}
]`

// A double-elimination bracket (2023 onwards): the status carries a
// double_elim_round, and alliance 1 called in a backup team.
const alliances2024ctharJSON = `[
  {
    "name": "Alliance 1",
    "declines": [],
    "picks": ["frc177", "frc1073", "frc5507"],
    "backup": {"in": "frc2168", "out": "frc5507"},
    "status": {
      "playoff_average": null,
      "level": "f",
      "double_elim_round": "Finals",
      "record": {"wins": 6, "losses": 1, "ties": 0},
      "current_level_record": {"wins": 2, "losses": 0, "ties": 0},
      "status": "won"
    }
  },
  {
    "name": null,
    "declines": ["frc558"],
    "picks": ["frc230", "frc195", "frc1071"],
    "backup": null,
    "status": {
      "playoff_average": null,
      "level": "sf",
      "double_elim_round": "Round 5",
      "record": {"wins": 4, "losses": 2, "ties": 0},
      "current_level_record": {"wins": 0, "losses": 1, "ties": 0},
      "status": "eliminated"
    }
  }
]`

// A pre-2023 bracket: quarter/semi/final levels and no double_elim_round. The
// last alliance never played, so its status is null.
const alliances2019ctharJSON = `[
  {
    "name": "Alliance 1",
    "declines": [],
    "picks": ["frc177", "frc1073", "frc5507"],
    "backup": null,
    "status": {
      "playoff_average": 0.0,
      "level": "f",
      "record": {"wins": 5, "losses": 2, "ties": 0},
      "current_level_record": {"wins": 2, "losses": 0, "ties": 0},
      "status": "won"
    }
  },
  {
    "name": "Alliance 4",
    "declines": ["frc2168"],
    "picks": ["frc230", "frc195", "frc558"],
    "backup": {"in": "frc1071", "out": "frc558"},
    "status": {
      "playoff_average": 0.0,
      "level": "qf",
      "record": {"wins": 1, "losses": 2, "ties": 0},
      "current_level_record": {"wins": 1, "losses": 2, "ties": 0},
      "status": "eliminated"
    }
  },
  {
    "name": "Alliance 8",
    "declines": [],
    "picks": ["frc3467", "frc6153"],
    "backup": null,
    "status": null
  }
]`

// Every team at an event, keyed by team. A team with nothing to report maps to
// null, and a team that has not been ranked yet has a null qual section.
const teamStatuses2024ctharJSON = `{
  "frc177": {
    "qual": {
      "num_teams": 40,
      "status": "completed",
      "ranking": {
        "team_key": "frc177",
        "rank": 1,
        "record": {"wins": 10, "losses": 2, "ties": 0},
        "qual_average": null,
        "matches_played": 12,
        "dq": 0,
        "sort_orders": [2.5, 0.25, 88.0, 31.0, 18.0]
      },
      "sort_order_info": [{"name": "Ranking Score", "precision": 2}]
    },
    "alliance": {"name": "Alliance 1", "number": 1, "pick": 0, "backup": null},
    "playoff": {
      "level": "f",
      "double_elim_round": "Finals",
      "current_level_record": {"wins": 2, "losses": 0, "ties": 0},
      "record": {"wins": 6, "losses": 1, "ties": 0},
      "status": "won",
      "playoff_average": null
    },
    "alliance_status_str": "<b>Captain</b> of <b>Alliance 1</b>",
    "playoff_status_str": "<b>Won</b> the event",
    "overall_status_str": "Team 177 was <b>Rank 1</b> with a record of <b>10-2-0</b> in quals,\ncompeted in the playoffs as the <b>Captain</b> of <b>Alliance 1</b>, and <b>won the event</b>.",
    "next_match_key": null,
    "last_match_key": "2024cthar_f1m2"
  },
  "frc1073": {
    "qual": {
      "num_teams": 40,
      "status": "completed",
      "ranking": {
        "team_key": "frc1073",
        "rank": 2,
        "record": {"wins": 9, "losses": 3, "ties": 0},
        "qual_average": null,
        "matches_played": 12,
        "dq": 1,
        "sort_orders": [2.3333, 0.0, 80.25, 28.5, 15.0]
      },
      "sort_order_info": [{"name": "Ranking Score", "precision": 2}]
    },
    "alliance": {"name": "Alliance 1", "number": 1, "pick": 1, "backup": null},
    "playoff": {
      "level": "sf",
      "double_elim_round": "Round 4",
      "current_level_record": {"wins": 0, "losses": 1, "ties": 0},
      "record": {"wins": 3, "losses": 2, "ties": 0},
      "status": "eliminated",
      "playoff_average": null
    },
    "alliance_status_str": "Pick 1 of <b>Alliance 1</b>",
    "playoff_status_str": "<b>Eliminated</b> in <b>Round 4</b>",
    "overall_status_str": "Team 1073 was <b>Rank 2</b> with a record of <b>9-3-0</b> in quals, and was <b>eliminated</b> in the playoffs.",
    "next_match_key": null,
    "last_match_key": "2024cthar_sf4m1"
  },
  "frc5507": {
    "qual": {
      "num_teams": 40,
      "status": "completed",
      "ranking": {
        "team_key": "frc5507",
        "rank": 30,
        "record": {"wins": 4, "losses": 8, "ties": 0},
        "qual_average": null,
        "matches_played": 12,
        "dq": 0,
        "sort_orders": [1.0, 0.0, 44.0, 10.0, 6.0]
      },
      "sort_order_info": [{"name": "Ranking Score", "precision": 2}]
    },
    "alliance": null,
    "playoff": null,
    "alliance_status_str": "Team 5507 was not picked for the playoffs",
    "playoff_status_str": "",
    "overall_status_str": "Team 5507 was <b>Rank 30</b> with a record of <b>4-8-0</b> in quals.",
    "next_match_key": null,
    "last_match_key": "2024cthar_qm82"
  },
  "frc2168": {
    "qual": null,
    "alliance": {"name": "Alliance 1", "number": 1, "pick": -1, "backup": {"in": "frc2168", "out": "frc5507"}},
    "playoff": {
      "level": "f",
      "double_elim_round": "Finals",
      "current_level_record": {"wins": 2, "losses": 0, "ties": 0},
      "record": {"wins": 2, "losses": 0, "ties": 0},
      "status": "won",
      "playoff_average": null
    },
    "alliance_status_str": "Backup on <b>Alliance 1</b>",
    "playoff_status_str": "<b>Won</b> the event",
    "overall_status_str": "Team 2168 competed in the playoffs as a <b>Backup</b> on <b>Alliance 1</b>.",
    "next_match_key": null,
    "last_match_key": "2024cthar_f1m2"
  },
  "frc9999": null
}`

// A pre-2023 event: the playoff status has no double_elim_round at all.
const teamStatuses2019ctharJSON = `{
  "frc177": {
    "qual": {
      "num_teams": 40,
      "status": "completed",
      "ranking": {
        "team_key": "frc177",
        "rank": 1,
        "record": {"wins": 10, "losses": 2, "ties": 0},
        "qual_average": null,
        "matches_played": 12,
        "dq": 0,
        "sort_orders": [2.5]
      },
      "sort_order_info": [{"name": "Ranking Score", "precision": 2}]
    },
    "alliance": {"name": "Alliance 1", "number": 1, "pick": 0, "backup": null},
    "playoff": {
      "level": "f",
      "current_level_record": {"wins": 2, "losses": 0, "ties": 0},
      "record": {"wins": 5, "losses": 2, "ties": 0},
      "status": "won",
      "playoff_average": 0.0
    },
    "alliance_status_str": "<b>Captain</b> of <b>Alliance 1</b>",
    "playoff_status_str": "<b>Won</b> the event",
    "overall_status_str": "Team 177 <b>won the event</b>.",
    "next_match_key": null,
    "last_match_key": "2019cthar_f1m2"
  }
}`

const awards2024ctharJSON = `[
  {
    "name": "District Event Winner",
    "award_type": 1,
    "event_key": "2024cthar",
    "year": 2024,
    "recipient_list": [{"team_key": "frc177", "awardee": null}]
  },
  {
    "name": "Dean's List Finalist Award",
    "award_type": 4,
    "event_key": "2024cthar",
    "year": 2024,
    "recipient_list": [{"team_key": "frc1073", "awardee": "Ada Lovelace"}]
  }
]`

const oprs2024ctharJSON = `{
  "oprs": {"frc177": 55.4321, "frc1073": 41.1234, "frc5507": 30.5},
  "dprs": {"frc177": 20.1111, "frc1073": 25.6666, "frc5507": 28.25},
  "ccwms": {"frc177": 35.3210, "frc1073": 15.4568, "frc5507": 2.25}
}`

const districtPoints2024ctharJSON = `{
  "points": {
    "frc5507": {"qual_points": 12, "alliance_points": 0, "elim_points": 0, "award_points": 0, "total": 12},
    "frc177": {"qual_points": 22, "alliance_points": 16, "elim_points": 30, "award_points": 5, "total": 73},
    "frc1073": {"qual_points": 20, "alliance_points": 14, "elim_points": 30, "award_points": 0, "total": 64},
    "frc230": {"qual_points": 18, "alliance_points": 16, "elim_points": 30, "award_points": 0, "total": 64}
  },
  "tiebreakers": {
    "frc5507": {"highest_qual_scores": [55, 51, 48], "qual_wins": 4},
    "frc177": {"highest_qual_scores": [88, 80, 76], "qual_wins": 10},
    "frc1073": {"highest_qual_scores": [84, 78, 72], "qual_wins": 9},
    "frc230": {"highest_qual_scores": [80, 77, 70], "qual_wins": 8}
  }
}`

// Predictions as the API sends them: match keys in the order a Go map hands
// them out rather than play order, a playoff round alongside the qualification
// one, and the model's own statistics.
const predictions2024ctharJSON = `{
  "match_predictions": {
    "qual": {
      "2024cthar_qm10": {"red": {"score": 70}, "blue": {"score": 71.05}, "winning_alliance": "blue", "prob": 0.5104},
      "2024cthar_qm1": {"red": {"score": 84.2}, "blue": {"score": 63.1}, "winning_alliance": "red", "prob": 0.7853},
      "2024cthar_qm2": {"red": {"score": 55.5}, "blue": {"score": 77.25}, "winning_alliance": "blue", "prob": 0.6231}
    },
    "playoff": {
      "2024cthar_f1m1": {"red": {"score": 121}, "blue": {"score": 118.4}, "winning_alliance": "red", "prob": 0.52},
      "2024cthar_sf1m1": {"red": {"score": 101.5}, "blue": {"score": 99.9}, "winning_alliance": "red", "prob": 0.5088},
      "2024cthar_sf3m1": {"red": {"score": 95}, "blue": {"score": 110.2}, "winning_alliance": "blue", "prob": 0.66}
    }
  },
  "match_prediction_stats": {
    "qual": {"brier_scores": {"win_loss": 0.14}},
    "playoff": {"brier_scores": {"win_loss": 0.2075}}
  },
  "stat_mean_vars": {
    "qual": {"score": {"mean": 61.4, "var": 210.25}},
    "playoff": {"score": {"mean": 94.7, "var": 180.5}}
  },
  "ranking_predictions": [
    ["frc1073", [3, 2, 7, 4, 4]],
    ["frc177", [1, 1, 2.5, 0, 0]],
    ["frc5507", [2, 1, 6, 2, 2]],
    ["frc230", [4]]
  ]
}`

// Event insights as the API sends them: season-specific keys, the
// [count, total, percent] triples the endpoint is full of, a nested object and
// the mixed high_score array.
const insights2024ctharJSON = `{
  "qual": {
    "high_score": ["2024cthar_qm1", 88, "Quals 1"],
    "average_score": 61.4,
    "average_win_margin": 18.256,
    "melody_bonus_achieved": [12, 60, 20],
    "unicorn_matches": [1, 60, 1.6667],
    "average_score_by_alliance": {"red": 60.5, "blue": 62.3}
  },
  "playoff": {
    "high_score": ["2024cthar_f1m1", 121, "Finals 1"],
    "average_score": 94.7,
    "ensemble_bonus_achieved": [8, 14, 57.1429]
  }
}`

const districts2024JSON = `[
  {"abbreviation": "ne", "display_name": "New England", "key": "2024ne", "year": 2024},
  {"abbreviation": "fim", "display_name": "FIRST In Michigan", "key": "2024fim", "year": 2024}
]`

const districtRankings2024neJSON = `[
  {
    "team_key": "frc177",
    "rank": 1,
    "rookie_bonus": 0,
    "point_total": 145,
    "event_points": [
      {"event_key": "2024cthar", "district_cmp": false, "qual_points": 11, "alliance_points": 16, "award_points": 5, "elim_points": 20, "total": 52},
      {"event_key": "2024ctwat", "district_cmp": false, "qual_points": 16, "alliance_points": 14, "award_points": 0, "elim_points": 18, "total": 48},
      {"event_key": "2024necmp", "district_cmp": true, "qual_points": 10, "alliance_points": 15, "award_points": 0, "elim_points": 20, "total": 45}
    ]
  },
  {
    "team_key": "frc1073",
    "rank": 2,
    "rookie_bonus": 0,
    "point_total": 132,
    "event_points": [
      {"event_key": "2024cthar", "district_cmp": false, "qual_points": 10, "alliance_points": 14, "award_points": 0, "elim_points": 16, "total": 40},
      {"event_key": "2024ctwat", "district_cmp": false, "qual_points": 12, "alliance_points": 12, "award_points": 0, "elim_points": 18, "total": 42},
      {"event_key": "2024necmp", "district_cmp": true, "qual_points": 12, "alliance_points": 18, "award_points": 0, "elim_points": 20, "total": 50}
    ]
  },
  {
    "team_key": "frc5507",
    "rank": 3,
    "rookie_bonus": 10,
    "point_total": 78,
    "event_points": [
      {"event_key": "2024cthar", "district_cmp": false, "qual_points": 8, "alliance_points": 10, "award_points": 0, "elim_points": 16, "total": 34},
      {"event_key": "2024ctwat", "district_cmp": false, "qual_points": 14, "alliance_points": 0, "award_points": 10, "elim_points": 10, "total": 34}
    ]
  }
]`

// The same district mid-season: the qualifying events have been played but the
// district championship has not, so every total is a pre-DCMP total and a
// "top N" line means exactly what it says.
const districtRankingsMidSeason2024neJSON = `[
  {
    "team_key": "frc177",
    "rank": 1,
    "rookie_bonus": 0,
    "point_total": 100,
    "event_points": [
      {"event_key": "2024cthar", "district_cmp": false, "qual_points": 11, "alliance_points": 16, "award_points": 5, "elim_points": 20, "total": 52},
      {"event_key": "2024ctwat", "district_cmp": false, "qual_points": 16, "alliance_points": 14, "award_points": 0, "elim_points": 18, "total": 48}
    ]
  },
  {
    "team_key": "frc1073",
    "rank": 2,
    "rookie_bonus": 0,
    "point_total": 82,
    "event_points": [
      {"event_key": "2024cthar", "district_cmp": false, "qual_points": 10, "alliance_points": 14, "award_points": 0, "elim_points": 16, "total": 40},
      {"event_key": "2024ctwat", "district_cmp": false, "qual_points": 12, "alliance_points": 12, "award_points": 0, "elim_points": 18, "total": 42}
    ]
  },
  {
    "team_key": "frc5507",
    "rank": 3,
    "rookie_bonus": 10,
    "point_total": 78,
    "event_points": [
      {"event_key": "2024cthar", "district_cmp": false, "qual_points": 8, "alliance_points": 10, "award_points": 0, "elim_points": 16, "total": 34},
      {"event_key": "2024ctwat", "district_cmp": false, "qual_points": 14, "alliance_points": 0, "award_points": 10, "elim_points": 10, "total": 34}
    ]
  }
]`

// A district whose championship changed the order: 1073 out-scored 177 at the
// DCMP and passed it, so the published rank 1 is not the team that led the
// standings the DCMP cut was made on.
const districtRankingsDCMPFlip2024neJSON = `[
  {
    "team_key": "frc1073",
    "rank": 1,
    "rookie_bonus": 0,
    "point_total": 150,
    "event_points": [
      {"event_key": "2024cthar", "district_cmp": false, "qual_points": 10, "alliance_points": 14, "award_points": 0, "elim_points": 16, "total": 40},
      {"event_key": "2024ctwat", "district_cmp": false, "qual_points": 12, "alliance_points": 12, "award_points": 0, "elim_points": 18, "total": 42},
      {"event_key": "2024necmp", "district_cmp": true, "qual_points": 18, "alliance_points": 20, "award_points": 10, "elim_points": 20, "total": 68}
    ]
  },
  {
    "team_key": "frc177",
    "rank": 2,
    "rookie_bonus": 0,
    "point_total": 145,
    "event_points": [
      {"event_key": "2024cthar", "district_cmp": false, "qual_points": 11, "alliance_points": 16, "award_points": 5, "elim_points": 20, "total": 52},
      {"event_key": "2024ctwat", "district_cmp": false, "qual_points": 16, "alliance_points": 14, "award_points": 0, "elim_points": 18, "total": 48},
      {"event_key": "2024necmp", "district_cmp": true, "qual_points": 10, "alliance_points": 15, "award_points": 0, "elim_points": 20, "total": 45}
    ]
  },
  {
    "team_key": "frc5507",
    "rank": 3,
    "rookie_bonus": 10,
    "point_total": 78,
    "event_points": [
      {"event_key": "2024cthar", "district_cmp": false, "qual_points": 8, "alliance_points": 10, "award_points": 0, "elim_points": 16, "total": 34},
      {"event_key": "2024ctwat", "district_cmp": false, "qual_points": 14, "alliance_points": 0, "award_points": 10, "elim_points": 10, "total": 34}
    ]
  }
]`

const teamAwards177JSON = `[
  {
    "name": "Regional Chairman's Award",
    "award_type": 0,
    "event_key": "2007ct",
    "year": 2007,
    "recipient_list": [{"team_key": "frc177", "awardee": null}]
  },
  {
    "name": "District Event Winner",
    "award_type": 1,
    "event_key": "2024cthar",
    "year": 2024,
    "recipient_list": [{"team_key": "frc177", "awardee": null}]
  }
]`

const teamMedia177JSON = `[
  {
    "type": "imgur",
    "foreign_key": "aBcDeFg",
    "details": {},
    "preferred": true,
    "direct_url": "https://i.imgur.com/aBcDeFg.jpg",
    "view_url": "https://imgur.com/aBcDeFg"
  },
  {
    "type": "youtube",
    "foreign_key": "dQw4w9WgXcQ",
    "details": {},
    "preferred": false,
    "direct_url": "",
    "view_url": "https://youtube.com/watch?v=dQw4w9WgXcQ"
  }
]`

const teamRobots177JSON = `[
  {"key": "frc177_2024", "robot_name": "Bobcat 2024", "team_key": "frc177", "year": 2024},
  {"key": "frc177_2023", "robot_name": "Sprocket", "team_key": "frc177", "year": 2023}
]`

const teamDistricts177JSON = `[
  {"abbreviation": "ne", "display_name": "New England", "key": "2023ne", "year": 2023},
  {"abbreviation": "ne", "display_name": "New England", "key": "2024ne", "year": 2024}
]`

// A season's leaderboards as the API sends them: a board about teams whose
// second place is a tie, a board with more rows than `--limit` shows by
// default, and a board keyed by event rather than by team.
const leaderboards2024JSON = `[
  {
    "name": "typed_leaderboard_blue_banners",
    "year": 2024,
    "data": {
      "key_type": "team",
      "rankings": [
        {"value": 6, "keys": ["frc177"]},
        {"value": 5, "keys": ["frc1073", "frc230"]},
        {"value": 4, "keys": ["frc5507"]}
      ]
    }
  },
  {
    "name": "typed_leaderboard_most_matches_played",
    "year": 2024,
    "data": {
      "key_type": "team",
      "rankings": [
        {"value": 92, "keys": ["frc254"]},
        {"value": 91, "keys": ["frc1114"]},
        {"value": 90, "keys": ["frc118"]},
        {"value": 89, "keys": ["frc2056"]},
        {"value": 88, "keys": ["frc971"]},
        {"value": 87, "keys": ["frc1678"]},
        {"value": 86, "keys": ["frc2767"]},
        {"value": 85, "keys": ["frc1323"]},
        {"value": 84, "keys": ["frc180"]},
        {"value": 83, "keys": ["frc33"]},
        {"value": 82, "keys": ["frc67"]},
        {"value": 81, "keys": ["frc217"]}
      ]
    }
  },
  {
    "name": "typed_leaderboard_highest_median_score_by_event",
    "year": 2024,
    "data": {
      "key_type": "event",
      "rankings": [{"value": 112.5, "keys": ["2024necmp"]}, {"value": 98, "keys": ["2024cthar"]}]
    }
  }
]`

// A leaderboard whose first row is a big tie: 14 teams share the top value,
// which is what the real blue banner board looks like (about 450 teams) and
// what the Key cell has to survive.
const leaderboardsBigTie2024JSON = `[
  {
    "name": "typed_leaderboard_blue_banners",
    "year": 2024,
    "data": {
      "key_type": "team",
      "rankings": [
        {"value": 6, "keys": ["frc177", "frc254", "frc1114", "frc118", "frc2056", "frc971", "frc1678", "frc2767", "frc1323", "frc180", "frc33", "frc67", "frc217", "frc2168"]},
        {"value": 5, "keys": ["frc1073", "frc230"]}
      ]
    }
  }
]`

const notables2024JSON = `[
  {
    "name": "notables_hall_of_fame",
    "year": 2024,
    "data": {"entries": [{"team_key": "frc177", "context": ["2007ct"]}]}
  },
  {
    "name": "notables_world_champions",
    "year": 2024,
    "data": {
      "entries": [
        {"team_key": "frc254", "context": ["2024cmptx", "2018cmptx"]},
        {"team_key": "frc1323", "context": ["2024cmptx"]}
      ]
    }
  }
]`

const apiStatusJSON = `{
  "current_season": 2024,
  "max_season": 2024,
  "is_datafeed_down": false,
  "down_events": [],
  "contbuild_enabled": true,
  "ios": {"latest_app_version": 1, "min_app_version": 1},
  "android": {"latest_app_version": 1, "min_app_version": 1}
}`

// A spread of 2024 events covering every filter `event list` offers: a
// regional, two district events in different districts and countries, a
// district championship, a championship division and an offseason.

const event2024casjJSON = `{
  "key": "2024casj",
  "name": "Silicon Valley Regional",
  "event_code": "casj",
  "event_type": 0,
  "district": null,
  "city": "San Jose",
  "state_prov": "CA",
  "country": "USA",
  "start_date": "2024-03-27",
  "end_date": "2024-03-30",
  "year": 2024,
  "short_name": "Silicon Valley",
  "event_type_string": "Regional",
  "week": 3,
  "address": "1393 S 7th St, San Jose, CA 95112, USA",
  "location_name": "San Jose State University",
  "timezone": "America/Los_Angeles",
  "website": "https://cafirst.org/frc/siliconvalley/",
  "first_event_code": "CASJ",
  "webcasts": [{"type": "twitch", "channel": "firstinspires9"}],
  "playoff_type": 10
}`

const event2024isde3JSON = `{
  "key": "2024isde3",
  "name": "ISR District Event #3",
  "event_code": "isde3",
  "event_type": 1,
  "district": {
    "abbreviation": "isr",
    "display_name": "FIRST Israel",
    "key": "2024isr",
    "year": 2024
  },
  "city": "Tel Aviv-Yafo",
  "state_prov": "",
  "country": "Israel",
  "start_date": "2024-03-12",
  "end_date": "2024-03-14",
  "year": 2024,
  "short_name": "ISR District Event #3",
  "event_type_string": "District",
  "week": 1,
  "address": "Shlomo Group Arena, Tel Aviv-Yafo, Israel",
  "location_name": "Shlomo Group Arena",
  "timezone": "Asia/Jerusalem",
  "webcasts": [],
  "playoff_type": 10
}`

const event2024milJSON = `{
  "key": "2024mil",
  "name": "Milstein Division",
  "event_code": "mil",
  "event_type": 3,
  "district": null,
  "city": "Houston",
  "state_prov": "TX",
  "country": "USA",
  "start_date": "2024-04-17",
  "end_date": "2024-04-20",
  "year": 2024,
  "short_name": "Milstein",
  "event_type_string": "Championship Division",
  "week": null,
  "address": "1001 Avenida De Las Americas, Houston, TX 77010, USA",
  "location_name": "George R. Brown Convention Center",
  "timezone": "America/Chicago",
  "webcasts": [{"type": "youtube", "channel": "abc123XYZ"}],
  "playoff_type": 10
}`

const event2024iriJSON = `{
  "key": "2024iri",
  "name": "Indiana Robotics Invitational",
  "event_code": "iri",
  "event_type": 99,
  "district": null,
  "city": "Indianapolis",
  "state_prov": "IN",
  "country": "USA",
  "start_date": "2024-07-12",
  "end_date": "2024-07-13",
  "year": 2024,
  "short_name": "Indiana Robotics Invitational",
  "event_type_string": "Offseason",
  "week": null,
  "address": "7350 Shadeland Station Way, Indianapolis, IN 46256, USA",
  "location_name": "Lawrence North High School",
  "timezone": "America/Indiana/Indianapolis",
  "webcasts": [],
  "playoff_type": 0
}`

const event2021nhflaJSON = `{
  "key": "2021nhfla",
  "name": "FIRST Innovation Challenge",
  "event_code": "nhfla",
  "event_type": 7,
  "district": null,
  "city": "",
  "state_prov": "",
  "country": "",
  "start_date": "2021-03-01",
  "end_date": "2021-03-01",
  "year": 2021,
  "short_name": "FIRST Innovation Challenge",
  "event_type_string": "Remote",
  "week": 5,
  "address": null,
  "location_name": null,
  "timezone": null,
  "webcasts": [],
  "playoff_type": null
}`

// events2024JSON is deliberately out of calendar order, so that tests see
// `event list` sort it.
const events2024JSON = "[" + event2024milJSON + "," + event2024ctharJSON + "," +
	event2024casjJSON + "," + event2024isde3JSON + "," + event2024necmpJSON + "," +
	event2024iriJSON + "]"

const teamEvents177JSON = "[" + event2024ctharJSON + "," + event2024necmpJSON + "]"
