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
      "team_key": "frc177",
      "rank": 1,
      "record": {"wins": 10, "losses": 2, "ties": 0},
      "qual_average": null,
      "matches_played": 12,
      "dq": 0,
      "sort_orders": [2.5, 88.0, 31.0, 18.0, 0.0],
      "extra_stats": [22]
    },
    {
      "team_key": "frc1073",
      "rank": 2,
      "record": {"wins": 9, "losses": 3, "ties": 0},
      "qual_average": null,
      "matches_played": 12,
      "dq": 0,
      "sort_orders": [2.33, 80.0, 28.0, 15.0, 0.0],
      "extra_stats": [20]
    }
  ],
  "sort_order_info": [
    {"name": "Ranking Score", "precision": 2},
    {"name": "Avg Coop", "precision": 0}
  ],
  "extra_stats_info": [{"name": "Total Ranking Points", "precision": 0}]
}`

const alliances2024ctharJSON = `[
  {
    "name": "Alliance 1",
    "declines": [],
    "picks": ["frc177", "frc1073", "frc5507"],
    "status": {"level": "f", "status": "won", "record": {"wins": 6, "losses": 1, "ties": 0}}
  },
  {
    "name": null,
    "declines": [],
    "picks": ["frc230", "frc195", "frc558"],
    "status": {"level": "f", "status": "eliminated", "record": {"wins": 4, "losses": 2, "ties": 0}}
  }
]`

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
    "frc177": {"qual_points": 22, "alliance_points": 16, "elim_points": 30, "award_points": 5, "total": 73}
  },
  "tiebreakers": {
    "frc177": {"highest_qual_scores": [88, 80, 76], "qual_wins": 10}
  }
}`

const predictions2024ctharJSON = `{
  "match_predictions": {"qual": {"2024cthar_qm1": {"red": {"score": 84.2}, "blue": {"score": 63.1}}}},
  "match_prediction_stats": {"qual": {"brier_scores": {"win_loss": 0.14}}},
  "ranking_predictions": [["frc177", [1, 1, 2.5, 0, 0]]]
}`

const insights2024ctharJSON = `{
  "qual": {"high_score": ["2024cthar_qm1", 88, "Quals 1"], "average_score": 61.4},
  "playoff": {"high_score": ["2024cthar_f1m1", 121, "Finals 1"], "average_score": 94.7}
}`

const districts2024JSON = `[
  {"abbreviation": "ne", "display_name": "New England", "key": "2024ne", "year": 2024},
  {"abbreviation": "fim", "display_name": "FIRST In Michigan", "key": "2024fim", "year": 2024}
]`

const districtRankings2024neJSON = `[
  {"team_key": "frc177", "rank": 1, "rookie_bonus": 0, "point_total": 145, "event_points": []},
  {"team_key": "frc1073", "rank": 2, "rookie_bonus": 0, "point_total": 132, "event_points": []}
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

const leaderboards2024JSON = `[
  {
    "name": "typed_leaderboard_blue_banners",
    "year": 2024,
    "data": {
      "key_type": "team",
      "rankings": [{"value": 6, "keys": ["frc177"]}, {"value": 5, "keys": ["frc1073"]}]
    }
  }
]`

const notables2024JSON = `[
  {
    "name": "notables_hall_of_fame",
    "year": 2024,
    "data": {"entries": [{"team_key": "frc177", "context": ["2007ct"]}]}
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
