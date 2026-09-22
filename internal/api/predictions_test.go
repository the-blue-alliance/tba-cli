package api

import (
	"encoding/json"
	"testing"
)

func TestRankingPredictionUnmarshalsThePairShape(t *testing.T) {
	var got []RankingPrediction
	body := `[["frc177", [1, 1, 2.5, 0, 0]], ["frc1073", [2, 0.5, 3, 1, 1]]]`
	if err := json.Unmarshal([]byte(body), &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("got %d predictions, want 2", len(got))
	}
	if got[0].TeamKey != "frc177" {
		t.Errorf("team key = %q", got[0].TeamKey)
	}
	want := []float64{1, 1, 2.5, 0, 0}
	if len(got[0].Values) != len(want) {
		t.Fatalf("values = %v, want %v", got[0].Values, want)
	}
	for i := range want {
		if got[0].Values[i] != want[i] {
			t.Errorf("values[%d] = %v, want %v", i, got[0].Values[i], want[i])
		}
	}
	if got[1].TeamKey != "frc1073" {
		t.Errorf("second team key = %q", got[1].TeamKey)
	}
}

// A season that puts something other than a number in the list still yields
// the ranks it does carry, rather than failing the whole response.
func TestRankingPredictionDropsNonNumbers(t *testing.T) {
	var got RankingPrediction
	if err := json.Unmarshal([]byte(`["frc254", [3, "n/a", null, 7]]`), &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(got.Values) != 2 || got.Values[0] != 3 || got.Values[1] != 7 {
		t.Errorf("values = %v, want [3 7]", got.Values)
	}
}

func TestRankingPredictionWithoutValues(t *testing.T) {
	var got RankingPrediction
	if err := json.Unmarshal([]byte(`["frc254"]`), &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if got.TeamKey != "frc254" || got.Values != nil {
		t.Errorf("got %+v, want frc254 with no values", got)
	}
}

func TestRankingPredictionRejectsMalformedPairs(t *testing.T) {
	for _, body := range []string{`[]`, `{"team": "frc177"}`, `[7, [1]]`} {
		var got RankingPrediction
		if err := json.Unmarshal([]byte(body), &got); err == nil {
			t.Errorf("unmarshalling %s should fail, got %+v", body, got)
		}
	}
}

func TestRankingPredictionRoundTripsToThePairShape(t *testing.T) {
	var got RankingPrediction
	if err := json.Unmarshal([]byte(`["frc177", [1, 2.5]]`), &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	out, err := json.Marshal(got)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if string(out) != `["frc177",[1,2.5]]` {
		t.Errorf("marshalled to %s", out)
	}
}

func TestEventPredictionsDecodesTheFullDocument(t *testing.T) {
	body := `{
	  "match_predictions": {
	    "qual": {"2024cthar_qm1": {"red": {"score": 84.2}, "blue": {"score": 63.1}, "winning_alliance": "red", "prob": 0.7853}},
	    "playoff": {"2024cthar_sf3m1": {"red": {"score": 101.5}, "blue": {"score": 99.9}, "winning_alliance": "blue", "prob": 0.51}}
	  },
	  "match_prediction_stats": {"qual": {"brier_scores": {"win_loss": 0.14}}},
	  "stat_mean_vars": {"qual": {"score": {"mean": 61.4, "var": 210.2}}},
	  "ranking_predictions": [["frc177", [1, 1, 2.5, 0, 0]]]
	}`
	var got EventPredictions
	if err := json.Unmarshal([]byte(body), &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if got.MatchPredictions == nil {
		t.Fatal("match predictions are missing")
	}
	qm1 := got.MatchPredictions.Qual["2024cthar_qm1"]
	if qm1.Red.Score != 84.2 || qm1.Blue.Score != 63.1 {
		t.Errorf("qm1 scores = %v / %v", qm1.Red.Score, qm1.Blue.Score)
	}
	if qm1.WinningAlliance != "red" {
		t.Errorf("qm1 winner = %q", qm1.WinningAlliance)
	}
	if qm1.Prob == nil || *qm1.Prob != 0.7853 {
		t.Errorf("qm1 prob = %v", qm1.Prob)
	}
	if _, ok := got.MatchPredictions.Playoff["2024cthar_sf3m1"]; !ok {
		t.Error("playoff predictions are missing")
	}
	if len(got.RankingPredictions) != 1 || got.RankingPredictions[0].TeamKey != "frc177" {
		t.Errorf("ranking predictions = %+v", got.RankingPredictions)
	}
	if got.MatchPredictionStats == nil || got.StatMeanVars == nil {
		t.Error("the stats sections should decode into untyped maps")
	}
}

// A prediction with no probability must stay distinguishable from one whose
// probability really is zero.
func TestMatchPredictionWithoutProb(t *testing.T) {
	var got MatchPrediction
	if err := json.Unmarshal([]byte(`{"red": {"score": 1}, "blue": {"score": 2}}`), &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if got.Prob != nil {
		t.Errorf("prob = %v, want nil", *got.Prob)
	}
}

func TestEventPredictionsDecodesAnEmptyDocument(t *testing.T) {
	for _, body := range []string{`{}`, `null`} {
		var got EventPredictions
		if err := json.Unmarshal([]byte(body), &got); err != nil {
			t.Fatalf("unmarshalling %s: %v", body, err)
		}
		if got.MatchPredictions != nil || len(got.RankingPredictions) != 0 {
			t.Errorf("%s decoded to %+v", body, got)
		}
	}
}

func TestInsightLeaderboardDecodesTies(t *testing.T) {
	body := `[{
	  "name": "typed_leaderboard_blue_banners",
	  "year": 2024,
	  "data": {"key_type": "team", "rankings": [{"value": 6, "keys": ["frc177", "frc1073"]}]}
	}]`
	var got []InsightLeaderboard
	if err := json.Unmarshal([]byte(body), &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(got) != 1 || got[0].Data.KeyType != "team" {
		t.Fatalf("got %+v", got)
	}
	if len(got[0].Data.Rankings[0].Keys) != 2 || got[0].Data.Rankings[0].Value != 6 {
		t.Errorf("rankings = %+v", got[0].Data.Rankings)
	}
}

func TestInsightNotableDecodesContext(t *testing.T) {
	body := `[{"name": "notables_hall_of_fame", "year": 2024, "data": {"entries": [{"team_key": "frc177", "context": ["2007ct"]}]}}]`
	var got []InsightNotable
	if err := json.Unmarshal([]byte(body), &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(got) != 1 || got[0].Data.Entries[0].TeamKey != "frc177" {
		t.Fatalf("got %+v", got)
	}
	if len(got[0].Data.Entries[0].Context) != 1 || got[0].Data.Entries[0].Context[0] != "2007ct" {
		t.Errorf("context = %v", got[0].Data.Entries[0].Context)
	}
}

func TestEventInsightsKeepsSeasonSpecificStats(t *testing.T) {
	body := `{"qual": {"high_score": ["2024cthar_qm1", 88, "Quals 1"], "average_score": 61.4}, "playoff": {}}`
	var got EventInsights
	if err := json.Unmarshal([]byte(body), &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if got.Qual["average_score"] != 61.4 {
		t.Errorf("average_score = %v", got.Qual["average_score"])
	}
	if _, ok := got.Qual["high_score"].([]interface{}); !ok {
		t.Errorf("high_score = %T", got.Qual["high_score"])
	}
}
