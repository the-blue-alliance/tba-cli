package api

type Team struct {
	Key        string  `json:"key"`
	TeamNumber int     `json:"team_number"`
	Name       string  `json:"name"`
	Nickname   string  `json:"nickname"`
	City       string  `json:"city"`
	StateProv  string  `json:"state_prov"`
	Country    string  `json:"country"`
	Website    string  `json:"website"`
	RookieYear int     `json:"rookie_year"`
	Motto      *string `json:"motto"`
	SchoolName *string `json:"school_name"`
}

type Event struct {
	Key          string    `json:"key"`
	Name         string    `json:"name"`
	EventCode    string    `json:"event_code"`
	EventType    int       `json:"event_type"`
	City         string    `json:"city"`
	StateProv    string    `json:"state_prov"`
	Country      string    `json:"country"`
	StartDate    string    `json:"start_date"`
	EndDate      string    `json:"end_date"`
	Year         int       `json:"year"`
	ShortName    string    `json:"short_name"`
	EventTypeStr string    `json:"event_type_string"`
	Week         *int      `json:"week"`
	Address      string    `json:"address"`
	LocationName string    `json:"location_name"`
	Webcasts     []Webcast `json:"webcasts"`
	PlayoffType  *int      `json:"playoff_type"`
	District     *District `json:"district"`
	Website      string    `json:"website"`
	Timezone     string    `json:"timezone"`
	// FirstEventCode is the event's code in FIRST's own systems, which is not
	// always the same as EventCode and is absent for unofficial events.
	FirstEventCode *string `json:"first_event_code"`
}

type Webcast struct {
	Type    string `json:"type"`
	Channel string `json:"channel"`
}

type Match struct {
	Key             string                 `json:"key"`
	CompLevel       string                 `json:"comp_level"`
	SetNumber       int                    `json:"set_number"`
	MatchNumber     int                    `json:"match_number"`
	EventKey        string                 `json:"event_key"`
	Time            *int64                 `json:"time"`
	PredictedTime   *int64                 `json:"predicted_time"`
	ActualTime      *int64                 `json:"actual_time"`
	Alliances       map[string]Alliance    `json:"alliances"`
	WinningAlliance string                 `json:"winning_alliance"`
	ScoreBreakdown  map[string]interface{} `json:"score_breakdown"`
	Videos          []Video                `json:"videos"`
}

type Alliance struct {
	Score             int      `json:"score"`
	TeamKeys          []string `json:"team_keys"`
	SurrogateTeamKeys []string `json:"surrogate_team_keys"`
	DQTeamKeys        []string `json:"dq_team_keys"`
}

type Video struct {
	Type string `json:"type"`
	Key  string `json:"key"`
}

type District struct {
	Key          string `json:"key"`
	Abbreviation string `json:"abbreviation"`
	DisplayName  string `json:"display_name"`
	Year         int    `json:"year"`
}

type Ranking struct {
	TeamKey       string     `json:"team_key"`
	Rank          int        `json:"rank"`
	Record        *WLTRecord `json:"record"`
	QualAverage   *float64   `json:"qual_average"`
	MatchesPlayed int        `json:"matches_played"`
	DQ            int        `json:"dq"`
	SortOrders    []float64  `json:"sort_orders"`
	ExtraStats    []float64  `json:"extra_stats"`
}

type WLTRecord struct {
	Wins   int `json:"wins"`
	Losses int `json:"losses"`
	Ties   int `json:"ties"`
}

type EventRankings struct {
	Rankings       []Ranking       `json:"rankings"`
	SortOrderInfo  []SortOrderInfo `json:"sort_order_info"`
	ExtraStatsInfo []SortOrderInfo `json:"extra_stats_info"`
}

type SortOrderInfo struct {
	Name      string `json:"name"`
	Precision int    `json:"precision"`
}

type EventAlliance struct {
	Name     *string         `json:"name"`
	Picks    []string        `json:"picks"`
	Status   *AllianceStatus `json:"status"`
	Declines []string        `json:"declines"`
	Backup   *AllianceBackup `json:"backup"`
}

// AllianceBackup records a backup team swap: In replaced Out.
type AllianceBackup struct {
	Out string `json:"out"`
	In  string `json:"in"`
}

type AllianceStatus struct {
	Status             string     `json:"status"`
	Level              string     `json:"level"`
	Record             *WLTRecord `json:"record"`
	CurrentLevelRecord *WLTRecord `json:"current_level_record"`
	PlayoffAverage     *float64   `json:"playoff_average"`
	// DoubleElimRound is only sent for double-elimination brackets
	// (2023 onwards), e.g. "Round 4".
	DoubleElimRound *string `json:"double_elim_round"`
}

type Award struct {
	Name       string           `json:"name"`
	AwardType  int              `json:"award_type"`
	EventKey   string           `json:"event_key"`
	Year       int              `json:"year"`
	Recipients []AwardRecipient `json:"recipient_list"`
}

type AwardRecipient struct {
	TeamKey *string `json:"team_key"`
	Awardee *string `json:"awardee"`
}

type Media struct {
	Type       string                 `json:"type"`
	ForeignKey string                 `json:"foreign_key"`
	Details    map[string]interface{} `json:"details"`
	DirectURL  string                 `json:"direct_url"`
	ViewURL    string                 `json:"view_url"`
}

type Robot struct {
	Year      int    `json:"year"`
	RobotName string `json:"robot_name"`
	Key       string `json:"key"`
	TeamKey   string `json:"team_key"`
}

type DistrictRanking struct {
	TeamKey     string                `json:"team_key"`
	Rank        int                   `json:"rank"`
	PointTotal  int                   `json:"point_total"`
	RookieBonus int                   `json:"rookie_bonus"`
	EventPoints []DistrictEventPoints `json:"event_points"`
}

type EventOPRs struct {
	OPRs  map[string]float64 `json:"oprs"`
	DPRs  map[string]float64 `json:"dprs"`
	CCWMs map[string]float64 `json:"ccwms"`
}

type APIStatus struct {
	CurrentSeason    int         `json:"current_season"`
	MaxSeason        int         `json:"max_season"`
	IsDatafeedDown   bool        `json:"is_datafeed_down"`
	ContBuildEnabled bool        `json:"contbuild_enabled"`
	AndroidSettings  interface{} `json:"android"`
	IOSSettings      interface{} `json:"ios"`
}

// DistrictEventPoints is one event's contribution to a team's district ranking.
// DistrictCMP marks the district championship, which is scored separately from
// the qualifying events.
type DistrictEventPoints struct {
	EventKey       string `json:"event_key"`
	DistrictCMP    bool   `json:"district_cmp"`
	QualPoints     int    `json:"qual_points"`
	AlliancePoints int    `json:"alliance_points"`
	AwardPoints    int    `json:"award_points"`
	ElimPoints     int    `json:"elim_points"`
	Total          int    `json:"total"`
}

// EventDistrictPoints is the district points one event awarded, keyed by team.
type EventDistrictPoints struct {
	Points      map[string]DistrictPointsDetail `json:"points"`
	Tiebreakers map[string]DistrictTiebreaker   `json:"tiebreakers"`
}

// DistrictPointsDetail is one team's district points at a single event.
type DistrictPointsDetail struct {
	QualPoints     int `json:"qual_points"`
	AlliancePoints int `json:"alliance_points"`
	AwardPoints    int `json:"award_points"`
	ElimPoints     int `json:"elim_points"`
	Total          int `json:"total"`
}

// DistrictTiebreaker holds the values that break a district points tie.
type DistrictTiebreaker struct {
	HighestQualScores []int `json:"highest_qual_scores"`
	QualWins          int   `json:"qual_wins"`
}

// TeamEventStatus is a team's standing at one event. Every section is optional:
// a team that has not played yet has no Qual, and one that was not picked has
// no Alliance.
type TeamEventStatus struct {
	Qual              *TeamEventQualStatus     `json:"qual"`
	Alliance          *TeamEventAllianceStatus `json:"alliance"`
	Playoff           *AllianceStatus          `json:"playoff"`
	AllianceStatusStr string                   `json:"alliance_status_str"`
	PlayoffStatusStr  string                   `json:"playoff_status_str"`
	OverallStatusStr  string                   `json:"overall_status_str"`
	NextMatchKey      *string                  `json:"next_match_key"`
	LastMatchKey      *string                  `json:"last_match_key"`
}

// TeamEventQualStatus is the qualification-round half of a team event status.
type TeamEventQualStatus struct {
	NumTeams      int             `json:"num_teams"`
	Status        string          `json:"status"`
	Ranking       *Ranking        `json:"ranking"`
	SortOrderInfo []SortOrderInfo `json:"sort_order_info"`
}

// TeamEventAllianceStatus says which alliance picked a team, and in what slot.
// Pick is 0 for the captain, 1 and 2 for the first and second picks, and -1 for
// a backup team called in mid-playoffs.
type TeamEventAllianceStatus struct {
	Name   string          `json:"name"`
	Number int             `json:"number"`
	Pick   int             `json:"pick"`
	Backup *AllianceBackup `json:"backup"`
}
