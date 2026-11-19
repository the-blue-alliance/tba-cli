package api

import (
	"go/ast"
	"go/parser"
	"go/token"
	"reflect"
	"sort"
	"testing"

	"github.com/the-blue-alliance/tba-cli/internal/apispec"
)

// structSchemas maps every struct in types.go to the spec schema it decodes.
// Nested structs are also checked through their parents.
var structSchemas = []struct {
	typ    reflect.Type
	schema string // JSON pointer into the spec
}{
	{reflect.TypeFor[Team](), apispec.SchemaPointer("Team")},
	{reflect.TypeFor[Event](), apispec.SchemaPointer("Event")},
	{reflect.TypeFor[Webcast](), apispec.SchemaPointer("Webcast")},
	{reflect.TypeFor[Match](), apispec.SchemaPointer("Match")},
	{reflect.TypeFor[Alliance](), apispec.SchemaPointer("Match_alliance")},
	{reflect.TypeFor[Video](), apispec.SchemaPointer("Match") + "/properties/videos/items"},
	{reflect.TypeFor[District](), apispec.SchemaPointer("District")},
	{reflect.TypeFor[Ranking](), apispec.SchemaPointer("Event_Ranking") + "/properties/rankings/items"},
	{reflect.TypeFor[WLTRecord](), apispec.SchemaPointer("WLT_Record")},
	{reflect.TypeFor[EventRankings](), apispec.SchemaPointer("Event_Ranking")},
	{reflect.TypeFor[SortOrderInfo](), apispec.SchemaPointer("Event_Ranking") + "/properties/sort_order_info/items"},
	{reflect.TypeFor[EventAlliance](), apispec.SchemaPointer("Elimination_Alliance")},
	{reflect.TypeFor[AllianceBackup](), apispec.SchemaPointer("Elimination_Alliance") + "/properties/backup"},
	{reflect.TypeFor[AllianceStatus](), apispec.SchemaPointer("Elimination_Alliance") + "/properties/status"},
	{reflect.TypeFor[Award](), apispec.SchemaPointer("Award")},
	{reflect.TypeFor[AwardRecipient](), apispec.SchemaPointer("Award_Recipient")},
	{reflect.TypeFor[Media](), apispec.SchemaPointer("Media")},
	{reflect.TypeFor[Robot](), apispec.SchemaPointer("Team_Robot")},
	{reflect.TypeFor[DistrictRanking](), apispec.SchemaPointer("District_Ranking")},
	{reflect.TypeFor[EventOPRs](), apispec.SchemaPointer("Event_OPRs")},
	{reflect.TypeFor[APIStatus](), apispec.SchemaPointer("API_Status")},
	{reflect.TypeFor[DistrictEventPoints](), apispec.SchemaPointer("District_Ranking") + "/properties/event_points/items"},
	{reflect.TypeFor[EventDistrictPoints](), apispec.SchemaPointer("Event_District_Points")},
	{reflect.TypeFor[DistrictPointsDetail](), apispec.SchemaPointer("Event_District_Points") + "/properties/points/additionalProperties"},
	{reflect.TypeFor[DistrictTiebreaker](), apispec.SchemaPointer("Event_District_Points") + "/properties/tiebreakers/additionalProperties"},
	{reflect.TypeFor[TeamEventStatus](), apispec.SchemaPointer("Team_Event_Status")},
	{reflect.TypeFor[TeamEventQualStatus](), apispec.SchemaPointer("Team_Event_Status_rank")},
	{reflect.TypeFor[TeamEventAllianceStatus](), apispec.SchemaPointer("Team_Event_Status_alliance")},
	{reflect.TypeFor[InsightLeaderboard](), apispec.SchemaPointer("LeaderboardInsight")},
	{reflect.TypeFor[InsightLeaderboardData](), apispec.SchemaPointer("LeaderboardInsight") + "/properties/data"},
	{reflect.TypeFor[InsightLeaderboardRanking](), apispec.SchemaPointer("LeaderboardInsight") + "/properties/data/properties/rankings/items"},
	{reflect.TypeFor[InsightNotable](), apispec.SchemaPointer("NotablesInsight")},
	{reflect.TypeFor[InsightNotableData](), apispec.SchemaPointer("NotablesInsight") + "/properties/data"},
	{reflect.TypeFor[InsightNotableEntry](), apispec.SchemaPointer("NotablesInsight") + "/properties/data/properties/entries/items"},
	{reflect.TypeFor[EventInsights](), apispec.SchemaPointer("Event_Insights")},
	{reflect.TypeFor[SponsorsList](), apispec.SchemaPointer("Sponsors_List")},
	{reflect.TypeFor[Sponsor](), apispec.SchemaPointer("Sponsor")},
	// Event_Predictions is an untyped object, so these are checked only for
	// being listed; see TestUntypedSchemasStayUntyped.
	{reflect.TypeFor[EventPredictions](), apispec.SchemaPointer("Event_Predictions")},
	{reflect.TypeFor[MatchPredictionRounds](), apispec.SchemaPointer("Event_Predictions")},
	{reflect.TypeFor[MatchPrediction](), apispec.SchemaPointer("Event_Predictions")},
	{reflect.TypeFor[AlliancePrediction](), apispec.SchemaPointer("Event_Predictions")},
	{reflect.TypeFor[RankingPrediction](), apispec.SchemaPointer("Event_Predictions")},
}

// structExceptions tolerates a struct field the spec does not declare, where
// the spec is believed to be wrong. Keyed by the reported location.
var structExceptions = map[string]string{
	// Ranking is shared with /event/{key}/rankings, which does send extra_stats.
	"TeamEventStatus.qual.ranking.extra_stats": "shared struct",
	"TeamEventQualStatus.ranking.extra_stats":  "shared struct",
	// Spec types it number; the API sends a digit count, as sort_order_info does.
	"EventRankings.extra_stats_info[].precision: Go int cannot hold JSON number": "spec too loose",
}

func TestStructFieldsAreInTheSpec(t *testing.T) {
	s, err := apispec.Load()
	if err != nil {
		t.Fatal(err)
	}
	for _, m := range structSchemas {
		if !s.Has(m.schema) {
			t.Errorf("%s: spec has no schema at %s", m.typ.Name(), m.schema)
			continue
		}
		bad, err := s.UndeclaredStructFields(m.schema, m.typ)
		if err != nil {
			t.Errorf("%s: %v", m.typ.Name(), err)
			continue
		}
		for _, b := range bad {
			if _, ok := structExceptions[b]; !ok {
				t.Errorf("%s (against %s)", b, m.schema)
			}
		}
	}
}

func TestEveryStructInTypesGoHasASchema(t *testing.T) {
	f, err := parser.ParseFile(token.NewFileSet(), "types.go", nil, 0)
	if err != nil {
		t.Fatal(err)
	}
	mapped := map[string]bool{}
	for _, m := range structSchemas {
		mapped[m.typ.Name()] = true
	}
	var missing []string
	for _, d := range f.Decls {
		g, ok := d.(*ast.GenDecl)
		if !ok {
			continue
		}
		for _, sp := range g.Specs {
			ts, ok := sp.(*ast.TypeSpec)
			if !ok {
				continue
			}
			if _, ok := ts.Type.(*ast.StructType); ok && !mapped[ts.Name.Name] {
				missing = append(missing, ts.Name.Name)
			}
		}
	}
	sort.Strings(missing)
	for _, name := range missing {
		t.Errorf("%s is not in structSchemas", name)
	}
}

// An untyped schema checks nothing; fail once the spec types it.
func TestUntypedSchemasStayUntyped(t *testing.T) {
	s, err := apispec.Load()
	if err != nil {
		t.Fatal(err)
	}
	if !s.IsOpen(apispec.SchemaPointer("Event_Predictions")) {
		t.Error("Event_Predictions is now typed in the spec; map the prediction structs to its parts")
	}
}

// Every unexpected field is reported, not just the first.
func TestStructCheckCatchesARenamedField(t *testing.T) {
	s, err := apispec.Load()
	if err != nil {
		t.Fatal(err)
	}
	type renamed struct {
		Key  string `json:"key"`
		Nick string `json:"nick_name"`
		Num  string `json:"team_number"`
	}
	bad, err := s.UndeclaredStructFields(apispec.SchemaPointer("Team"), reflect.TypeFor[renamed]())
	if err != nil {
		t.Fatal(err)
	}
	if len(bad) != 2 {
		t.Errorf("got %v, want nick_name undeclared and team_number mistyped", bad)
	}
}
