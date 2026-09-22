package cmd

import (
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/the-blue-alliance/tba-cli/internal/clierr"
)

func predictionsServer(t *testing.T) *httptest.Server {
	t.Helper()
	return newFakeTBA(t, map[string]any{
		"/event/2024cthar/predictions": predictions2024ctharJSON,
	})
}

func insightsServer(t *testing.T) *httptest.Server {
	t.Helper()
	return newFakeTBA(t, map[string]any{
		"/event/2024cthar/insights": insights2024ctharJSON,
	})
}

func TestEventPredictionsTable(t *testing.T) {
	out, _, err := runCmd(t, predictionsServer(t), "event", "predictions", "2024cthar",
		"--format", "csv")
	requireNoError(t, err, "")
	want := []string{
		"Match,Red Score,Blue Score,Predicted Winner,Confidence",
		"2024cthar_qm1,84.2,63.1,Red,78.53%",
		"2024cthar_qm2,55.5,77.25,Blue,62.31%",
		"2024cthar_qm10,70,71.05,Blue,51.04%",
		"2024cthar_sf1m1,101.5,99.9,Red,50.88%",
		"2024cthar_sf3m1,95,110.2,Blue,66%",
		"2024cthar_f1m1,121,118.4,Red,52%",
	}
	got := lines(out)
	if len(got) != len(want) {
		t.Fatalf("got %d lines, want %d:\n%s", len(got), len(want), out)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("line %d = %q, want %q", i, got[i], want[i])
		}
	}
}

// The order is play order, not the alphabetical order the match keys are in
// and not the random order a Go map hands them out in.
func TestEventPredictionsAreInPlayOrder(t *testing.T) {
	for i := 0; i < 5; i++ {
		out, _, err := runCmd(t, predictionsServer(t), "event", "predictions", "2024cthar",
			"--format", "tsv", "--no-headers", "--columns", "match")
		requireNoError(t, err, "")
		want := "2024cthar_qm1 2024cthar_qm2 2024cthar_qm10 2024cthar_sf1m1 2024cthar_sf3m1 2024cthar_f1m1"
		if got := strings.Join(lines(out), " "); got != want {
			t.Fatalf("run %d ordered them %q", i, got)
		}
	}
}

func TestEventPredictionsRankings(t *testing.T) {
	out, _, err := runCmd(t, predictionsServer(t), "event", "predictions", "2024cthar",
		"--rankings", "--format", "csv")
	requireNoError(t, err, "")
	want := []string{
		"Team,Predicted Rank,Range",
		"177,1,0-2.5",
		"5507,2,1-6",
		"1073,3,2-7",
		// A team the API sent only a rank for has no range to show.
		"230,4,",
	}
	got := lines(out)
	if len(got) != len(want) {
		t.Fatalf("got %d lines, want %d:\n%s", len(got), len(want), out)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("line %d = %q, want %q", i, got[i], want[i])
		}
	}
}

func TestEventPredictionsStats(t *testing.T) {
	out, _, err := runCmd(t, predictionsServer(t), "event", "predictions", "2024cthar",
		"--stats", "--format", "csv")
	requireNoError(t, err, "")
	want := []string{
		"Stat,Value",
		"Match Prediction Stats.Playoff.Brier Scores.Win Loss,0.21",
		"Match Prediction Stats.Qual.Brier Scores.Win Loss,0.14",
		"Stat Mean Vars.Playoff.Score.Mean,94.7",
		"Stat Mean Vars.Playoff.Score.Var,180.5",
		"Stat Mean Vars.Qual.Score.Mean,61.4",
		"Stat Mean Vars.Qual.Score.Var,210.25",
	}
	got := lines(out)
	if len(got) != len(want) {
		t.Fatalf("got %d lines, want %d:\n%s", len(got), len(want), out)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("line %d = %q, want %q", i, got[i], want[i])
		}
	}
}

func TestEventPredictionsRejectsBothTables(t *testing.T) {
	err := requireExitCode(t, clierr.ExitUsage, predictionsServer(t),
		"event", "predictions", "2024cthar", "--rankings", "--stats")
	requireErrorContains(t, err, "pass only one")
}

// An event TBA has not modelled is not an error: the note goes to stderr, the
// table stays empty and the exit code stays 0.
func TestEventPredictionsWithoutAModel(t *testing.T) {
	for _, body := range []string{`{}`, `null`, `{"match_predictions": {}}`,
		`{"match_predictions": {"qual": {}, "playoff": {}}}`} {
		t.Run(body, func(t *testing.T) {
			srv := newFakeTBA(t, map[string]any{"/event/2024cthar/predictions": body})
			out, errOut, err := runCmd(t, srv, "event", "predictions", "2024cthar", "--format", "table")
			requireNoError(t, err, errOut)
			if out != "" {
				t.Errorf("stdout should be empty, got:\n%s", out)
			}
			requireContains(t, errOut, "no predictions available for 2024cthar")
		})
	}
}

// Every view says so when it has nothing to show.
func TestEventPredictionsEmptyViews(t *testing.T) {
	for _, flag := range []string{"--rankings", "--stats"} {
		t.Run(flag, func(t *testing.T) {
			srv := newFakeTBA(t, map[string]any{"/event/2024cthar/predictions": "{}"})
			out, errOut, err := runCmd(t, srv, "event", "predictions", "2024cthar", flag, "--format", "table")
			requireNoError(t, err, errOut)
			if out != "" {
				t.Errorf("stdout should be empty, got:\n%s", out)
			}
			requireContains(t, errOut, "no predictions available")
		})
	}
}

// A program reading stdout as JSON still gets a document to parse, and the
// note stays on stderr where it cannot corrupt it.
func TestEventPredictionsWithoutAModelStillPrintsJSON(t *testing.T) {
	srv := newFakeTBA(t, map[string]any{"/event/2024cthar/predictions": "{}"})
	out, errOut, err := runCmd(t, srv, "event", "predictions", "2024cthar", "--json")
	requireNoError(t, err, errOut)
	if _, ok := decodeJSON(t, out).(map[string]any); !ok {
		t.Errorf("want a JSON object, got:\n%s", out)
	}
	requireContains(t, errOut, "no predictions available")
}

// JSON output is the body the API sent, whichever table was asked for.
func TestEventPredictionsJSONIsTheRawDocument(t *testing.T) {
	for _, args := range [][]string{nil, {"--rankings"}, {"--stats"}} {
		t.Run(strings.Join(append([]string{"default"}, args...), " "), func(t *testing.T) {
			full := append([]string{"event", "predictions", "2024cthar", "--json"}, args...)
			out, _, err := runCmd(t, predictionsServer(t), full...)
			requireNoError(t, err, "")
			obj := decodeJSON(t, out).(map[string]any)
			for _, key := range []string{"match_predictions", "match_prediction_stats",
				"stat_mean_vars", "ranking_predictions"} {
				if _, ok := obj[key]; !ok {
					t.Errorf("%s is missing from the JSON:\n%s", key, out)
				}
			}
		})
	}
}

func TestEventPredictionsJq(t *testing.T) {
	out, _, err := runCmd(t, predictionsServer(t), "event", "predictions", "2024cthar",
		"--jq", ".match_predictions.qual.\"2024cthar_qm1\".red.score")
	requireNoError(t, err, "")
	if strings.TrimSpace(out) != "84.2" {
		t.Errorf("jq output = %q", out)
	}
}

func TestEventPredictionsRejectsABadEventKey(t *testing.T) {
	err := requireExitCode(t, clierr.ExitUsage, nil, "event", "predictions", "not-a-key")
	requireErrorContains(t, err, "not a valid event key")
}

func TestEventInsightsTable(t *testing.T) {
	out, _, err := runCmd(t, insightsServer(t), "event", "insights", "2024cthar", "--format", "csv")
	requireNoError(t, err, "")
	want := []string{
		"Section,Stat,Value",
		"Qualification,Average Score,61.4",
		`Qualification,Average Score By Alliance.Blue,62.3`,
		`Qualification,Average Score By Alliance.Red,60.5`,
		"Qualification,Average Win Margin,18.26",
		`Qualification,High Score,"2024cthar_qm1, 88, Quals 1"`,
		"Qualification,Melody Bonus Achieved,12/60 (20%)",
		"Qualification,Unicorn Matches,1/60 (1.67%)",
		"Playoff,Average Score,94.7",
		"Playoff,Ensemble Bonus Achieved,8/14 (57.14%)",
		`Playoff,High Score,"2024cthar_f1m1, 121, Finals 1"`,
	}
	got := lines(out)
	if len(got) != len(want) {
		t.Fatalf("got %d lines, want %d:\n%s", len(got), len(want), out)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("line %d = %q, want %q", i, got[i], want[i])
		}
	}
}

// Qualification comes before the playoff round, whatever order the JSON was in.
func TestEventInsightsSectionOrder(t *testing.T) {
	out, _, err := runCmd(t, insightsServer(t), "event", "insights", "2024cthar", "--format", "table")
	requireNoError(t, err, "")
	qual := strings.Index(out, "Qualification")
	playoff := strings.Index(out, "Playoff")
	if qual < 0 || playoff < 0 || qual > playoff {
		t.Errorf("sections are out of order:\n%s", out)
	}
}

func TestEventInsightsLevel(t *testing.T) {
	cases := map[string]string{"qual": "Qualification", "playoff": "Playoff"}
	for level, want := range cases {
		t.Run(level, func(t *testing.T) {
			out, _, err := runCmd(t, insightsServer(t), "event", "insights", "2024cthar",
				"--level", level, "--format", "csv", "--no-headers")
			requireNoError(t, err, "")
			for _, line := range lines(out) {
				if !strings.HasPrefix(line, want+",") {
					t.Errorf("--level %s produced %q", level, line)
				}
			}
			if len(lines(out)) == 0 {
				t.Errorf("--level %s produced nothing", level)
			}
		})
	}
}

func TestEventInsightsRejectsAnUnknownLevel(t *testing.T) {
	err := requireExitCode(t, clierr.ExitUsage, insightsServer(t),
		"event", "insights", "2024cthar", "--level", "elims")
	requireErrorContains(t, err, `invalid --level "elims"`)
}

func TestEventInsightsJSONIsTheRawDocument(t *testing.T) {
	out, _, err := runCmd(t, insightsServer(t), "event", "insights", "2024cthar")
	requireNoError(t, err, "")
	obj := decodeJSON(t, out).(map[string]any)
	qual := obj["qual"].(map[string]any)
	if qual["average_score"] != 61.4 {
		t.Errorf("average_score = %v", qual["average_score"])
	}
	// --level is presentation, so it never trims the JSON.
	out, _, err = runCmd(t, insightsServer(t), "event", "insights", "2024cthar",
		"--level", "qual", "--json")
	requireNoError(t, err, "")
	if _, ok := decodeJSON(t, out).(map[string]any)["playoff"]; !ok {
		t.Errorf("--level should not have trimmed the JSON:\n%s", out)
	}
}

func TestEventInsightsOfAnEventWithNone(t *testing.T) {
	srv := newFakeTBA(t, map[string]any{"/event/2024cthar/insights": "{}"})
	out, _, err := runCmd(t, srv, "event", "insights", "2024cthar", "--format", "table")
	requireNoError(t, err, "")
	if got := lines(out); len(got) != 2 || !strings.HasPrefix(got[0], "Section") {
		t.Errorf("output = %q", out)
	}
}

func TestEventInsightsJq(t *testing.T) {
	out, _, err := runCmd(t, insightsServer(t), "event", "insights", "2024cthar",
		"--jq", ".qual.average_score")
	requireNoError(t, err, "")
	if strings.TrimSpace(out) != "61.4" {
		t.Errorf("jq output = %q", out)
	}
}

func TestEventInsightsRejectsABadEventKey(t *testing.T) {
	err := requireExitCode(t, clierr.ExitUsage, nil, "event", "insights", "not-a-key")
	requireErrorContains(t, err, "not a valid event key")
}

// Both commands render in every tabular format, and both carry examples.
func TestEventInsightCommandsHonorEveryFormat(t *testing.T) {
	cases := []struct {
		command string
		newSrv  func(*testing.T) *httptest.Server
		header  string
	}{
		{"predictions", predictionsServer, "Match"},
		{"insights", insightsServer, "Section"},
	}
	for _, tc := range cases {
		t.Run(tc.command, func(t *testing.T) {
			for _, format := range []string{"table", "csv", "tsv", "markdown"} {
				t.Run(format, func(t *testing.T) {
					out, _, err := runCmd(t, tc.newSrv(t), "event", tc.command, "2024cthar", "--format", format)
					requireNoError(t, err, "")
					requireContains(t, out, tc.header)
				})
			}
		})
	}
}

func TestEventInsightCommandsCarryExamples(t *testing.T) {
	for _, c := range NewRootCmd().Commands() {
		if c.Name() != "event" {
			continue
		}
		for _, sub := range c.Commands() {
			if sub.Name() != "predictions" && sub.Name() != "insights" {
				continue
			}
			if strings.TrimSpace(sub.Example) == "" {
				t.Errorf("event %s has no Example block", sub.Name())
			}
		}
	}
}
