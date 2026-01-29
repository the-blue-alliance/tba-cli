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
	Key           string   `json:"key"`
	Name          string   `json:"name"`
	EventCode     string   `json:"event_code"`
	EventType     int      `json:"event_type"`
	City          string   `json:"city"`
	StateProv     string   `json:"state_prov"`
	Country       string   `json:"country"`
	StartDate     string   `json:"start_date"`
	EndDate       string   `json:"end_date"`
	Year          int      `json:"year"`
	ShortName     string   `json:"short_name"`
	EventTypeStr  string   `json:"event_type_string"`
	Week          *int     `json:"week"`
	Address       string   `json:"address"`
	LocationName  string   `json:"location_name"`
	Webcasts      []Webcast `json:"webcasts"`
	PlayoffType   *int     `json:"playoff_type"`
	District      *District `json:"district"`
}

type Webcast struct {
	Type    string `json:"type"`
	Channel string `json:"channel"`
}

type Match struct {
	Key           string                 `json:"key"`
	CompLevel     string                 `json:"comp_level"`
	SetNumber     int                    `json:"set_number"`
	MatchNumber   int                    `json:"match_number"`
	EventKey      string                 `json:"event_key"`
	Time          *int64                 `json:"time"`
	PredictedTime *int64                 `json:"predicted_time"`
	ActualTime    *int64                 `json:"actual_time"`
	Alliances     map[string]Alliance    `json:"alliances"`
	WinningAlliance string              `json:"winning_alliance"`
	ScoreBreakdown map[string]interface{} `json:"score_breakdown"`
	Videos        []Video                `json:"videos"`
}

type Alliance struct {
	Score     int      `json:"score"`
	TeamKeys  []string `json:"team_keys"`
	SurrogateTeamKeys []string `json:"surrogate_team_keys"`
	DQTeamKeys []string `json:"dq_team_keys"`
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
	TeamKey      string      `json:"team_key"`
	Rank         int         `json:"rank"`
	Record       *WLTRecord  `json:"record"`
	QualAverage  *float64    `json:"qual_average"`
	MatchesPlayed int        `json:"matches_played"`
	DQ           int         `json:"dq"`
	SortOrders   []float64   `json:"sort_orders"`
}

type WLTRecord struct {
	Wins   int `json:"wins"`
	Losses int `json:"losses"`
	Ties   int `json:"ties"`
}

type EventRankings struct {
	Rankings   []Ranking       `json:"rankings"`
	SortOrderInfo []SortOrderInfo `json:"sort_order_info"`
}

type SortOrderInfo struct {
	Name      string `json:"name"`
	Precision int    `json:"precision"`
}

type EventAlliance struct {
	Name   *string          `json:"name"`
	Picks  []string         `json:"picks"`
	Status *AllianceStatus  `json:"status"`
	Declines []string       `json:"declines"`
}

type AllianceStatus struct {
	Status string `json:"status"`
	Level  string `json:"level"`
	Record *WLTRecord `json:"record"`
}

type Award struct {
	Name      string         `json:"name"`
	AwardType int            `json:"award_type"`
	EventKey  string         `json:"event_key"`
	Year      int            `json:"year"`
	Recipients []AwardRecipient `json:"recipient_list"`
}

type AwardRecipient struct {
	TeamKey *string `json:"team_key"`
	Awardee *string `json:"awardee"`
}

type Media struct {
	Type      string `json:"type"`
	ForeignKey string `json:"foreign_key"`
	Details   map[string]interface{} `json:"details"`
	DirectURL string `json:"direct_url"`
	ViewURL   string `json:"view_url"`
}

type Robot struct {
	Year      int    `json:"year"`
	RobotName string `json:"robot_name"`
	Key       string `json:"key"`
	TeamKey   string `json:"team_key"`
}

type DistrictRanking struct {
	TeamKey    string `json:"team_key"`
	Rank       int    `json:"rank"`
	PointTotal int    `json:"point_total"`
	RookieBonus int   `json:"rookie_bonus"`
}

type EventOPRs struct {
	OPRs  map[string]float64 `json:"oprs"`
	DPRs  map[string]float64 `json:"dprs"`
	CCWMs map[string]float64 `json:"ccwms"`
}

type APIStatus struct {
	CurrentSeason    int    `json:"current_season"`
	MaxSeason        int    `json:"max_season"`
	IsDatafeedDown   bool   `json:"is_datafeed_down"`
	ContBuildEnabled bool   `json:"contbuild_enabled"`
	AndroidSettings  interface{} `json:"android"`
	IOSSettings      interface{} `json:"ios"`
}
