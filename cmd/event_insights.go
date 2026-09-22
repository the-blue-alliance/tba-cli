package cmd

import (
	"encoding/json"
	"fmt"
	"math"
	"sort"
	"strings"

	"github.com/spf13/cobra"
	"github.com/the-blue-alliance/tba-cli/internal/api"
	"github.com/the-blue-alliance/tba-cli/internal/clierr"
	"github.com/the-blue-alliance/tba-cli/internal/output"
)

func newEventPredictionsCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "predictions <key>",
		Short: "Show event predictions",
		Long: `Show what TBA's model expects at an event.

By default the table is Match | Red Score | Blue Score | Predicted Winner |
Confidence, in play order -- qualification matches first, then the playoff
bracket -- rather than the alphabetical order the match keys are in. Confidence
is the model's own probability for the winner it picked.

--rankings switches to the predicted qualification ranking, Team | Predicted
Rank | Range, where Range bounds the rank the model allows for.

--stats shows the model's own numbers instead: the Brier scores and the mean
and variance of each statistic it fits.

Not every event is modelled. One TBA has no predictions for prints a note on
stderr and no rows, and still exits 0.

JSON output stays the document the API sent, whichever table was asked for.`,
		Example: `  tba event predictions 2024cthar
  tba event predictions 2024cthar --rankings
  tba event predictions 2024cthar --stats
  tba event predictions 2024cthar --jq '.match_predictions.qual'`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := validateEventKey(args[0]); err != nil {
				return err
			}
			showRankings, _ := cmd.Flags().GetBool("rankings")
			showStats, _ := cmd.Flags().GetBool("stats")
			if showRankings && showStats {
				return clierr.Usage("--rankings and --stats each choose a different table; pass only one")
			}
			// Resolved up front so a bad --format or --color fails before any
			// request, and so the empty-event path below can honour it.
			format, err := resolveFormat(cmd)
			if err != nil {
				return err
			}
			if _, err := colorMode(cmd); err != nil {
				return err
			}

			client, err := newClient(cmd)
			if err != nil {
				return err
			}
			raw, err := client.GetRaw(cmd.Context(), fmt.Sprintf("/event/%s/predictions", args[0]))
			if err != nil {
				return err
			}
			var predictions api.EventPredictions
			if err := json.Unmarshal(raw, &predictions); err != nil {
				return fmt.Errorf("reading predictions for %s: %w", args[0], err)
			}

			var headers []string
			var rows [][]string
			switch {
			case showRankings:
				headers, rows = rankingPredictionTable(predictions.RankingPredictions)
			case showStats:
				headers, rows = predictionStatsTable(predictions)
			default:
				headers, rows = matchPredictionTable(predictions.MatchPredictions)
			}
			if len(rows) == 0 {
				return noPredictions(cmd, args[0], format, raw)
			}
			return outputTable(cmd, raw, headers, rows)
		},
	}
	c.Flags().Bool("rankings", false, "Show the predicted qualification ranking instead of the matches")
	c.Flags().Bool("stats", false, "Show the model's own statistics instead of the matches")
	return c
}

// noPredictions handles an event the model has nothing to say about. The note
// goes to stderr and the exit code stays 0, since an unmodelled event is a fact
// about the season rather than a failure. Tabular output is left empty: a
// header row with nothing under it would read as though the table were the
// answer. JSON still prints the body, because a program reading stdout as JSON
// needs a document to parse.
func noPredictions(cmd *cobra.Command, key, format string, raw json.RawMessage) error {
	fmt.Fprintf(cmd.ErrOrStderr(), "no predictions available for %s\n", key)
	if format != "json" {
		return nil
	}
	return output.PrintJSONWithFilter(cmd.OutOrStdout(), raw, jqExpr(cmd), rawOutput(cmd))
}

// matchPredictionTable lists every predicted match, qualification rounds first
// and then the playoff bracket, each in play order.
func matchPredictionTable(rounds *api.MatchPredictionRounds) ([]string, [][]string) {
	headers := []string{"Match", "Red Score", "Blue Score", "Predicted Winner", "Confidence"}
	if rounds == nil {
		return headers, nil
	}
	var rows [][]string
	for _, round := range []map[string]api.MatchPrediction{rounds.Qual, rounds.Playoff} {
		keys := make([]string, 0, len(round))
		for k := range round {
			keys = append(keys, k)
		}
		sortMatchKeys(keys)
		for _, k := range keys {
			p := round[k]
			rows = append(rows, []string{
				k,
				formatNumber(p.Red.Score),
				formatNumber(p.Blue.Score),
				allianceLabel(p.WinningAlliance),
				formatConfidence(p.Prob),
			})
		}
	}
	return headers, rows
}

// allianceLabel titles an alliance colour, leaving an unpredicted match blank
// rather than inventing a winner.
func allianceLabel(alliance string) string {
	switch strings.ToLower(alliance) {
	case "":
		return ""
	case "red":
		return "Red"
	case "blue":
		return "Blue"
	default:
		return humanizeName(alliance)
	}
}

// formatConfidence renders a probability as a percentage. A prediction without
// one stays blank, which is not the same as a confidence of zero.
func formatConfidence(prob *float64) string {
	if prob == nil {
		return ""
	}
	return formatNumber(*prob*100) + "%"
}

// rankingPredictionTable lists the predicted finish for each team, best first.
// The API sends a list of numbers per team whose first entry is the rank; the
// rest bound it, and become the Range column.
func rankingPredictionTable(predictions []api.RankingPrediction) ([]string, [][]string) {
	headers := []string{"Team", "Predicted Rank", "Range"}
	ordered := make([]api.RankingPrediction, len(predictions))
	copy(ordered, predictions)
	sort.SliceStable(ordered, func(i, j int) bool {
		a, b := predictedRank(ordered[i]), predictedRank(ordered[j])
		if a != b {
			return a < b
		}
		return teamNumber(ordered[i].TeamKey) < teamNumber(ordered[j].TeamKey)
	})

	rows := make([][]string, 0, len(ordered))
	for _, p := range ordered {
		rank := ""
		if len(p.Values) > 0 {
			rank = formatNumber(p.Values[0])
		}
		rows = append(rows, []string{
			output.TeamNumberFromKey(p.TeamKey),
			rank,
			formatRankRange(p.Values),
		})
	}
	return headers, rows
}

// predictedRank is the value the ranking table sorts on. A team the API sent no
// numbers for sorts last rather than ahead of the field.
func predictedRank(p api.RankingPrediction) float64 {
	if len(p.Values) == 0 {
		return math.MaxFloat64
	}
	return p.Values[0]
}

// formatRankRange bounds a predicted rank with the numbers that follow it, as
// "1-7". A prediction that carries only the rank has no range to show.
func formatRankRange(values []float64) string {
	if len(values) < 2 {
		return ""
	}
	low, high := values[1], values[1]
	for _, v := range values[2:] {
		if v < low {
			low = v
		}
		if v > high {
			high = v
		}
	}
	return formatNumber(low) + "-" + formatNumber(high)
}

// predictionStatsTable flattens the model's own numbers -- the Brier scores and
// the per-statistic means and variances -- into one key/value table.
func predictionStatsTable(p api.EventPredictions) ([]string, [][]string) {
	sections := []struct {
		name  string
		stats map[string]interface{}
	}{
		{"match_prediction_stats", p.MatchPredictionStats},
		{"stat_mean_vars", p.StatMeanVars},
	}
	var rows [][]string
	for _, s := range sections {
		for _, pair := range flattenInsightStats(s.name, s.stats) {
			rows = append(rows, []string{humanizeStatKey(pair[0]), pair[1]})
		}
	}
	return []string{"Stat", "Value"}, rows
}

func newEventInsightsCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "insights <key>",
		Short: "Show event insights",
		Long: `Show the statistics TBA computed for an event.

The table is Section | Stat | Value, with the qualification round first and the
playoff round after it. Which statistics exist is decided by the season's game,
so the rows change from year to year; a nested statistic is flattened into a
dotted name such as "Score By Alliance.Red".

The counting statistics the endpoint is full of arrive as [count, total,
percent] and are rendered "12/60 (20%)". Other numbers are trimmed to two
decimals and lists are joined with commas.

--level shows one round on its own.

JSON output stays the document the API sent.`,
		Example: `  tba event insights 2024cthar
  tba event insights 2024cthar --level playoff
  tba event insights 2024cthar --columns stat,value
  tba event insights 2024cthar --jq .qual.high_score`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := validateEventKey(args[0]); err != nil {
				return err
			}
			level, _ := cmd.Flags().GetString("level")
			level = strings.ToLower(strings.TrimSpace(level))
			if level != "" && level != "qual" && level != "playoff" {
				return clierr.Usage("invalid --level %q (want: qual, playoff)", level)
			}
			client, err := newClient(cmd)
			if err != nil {
				return err
			}
			raw, err := client.GetRaw(cmd.Context(), fmt.Sprintf("/event/%s/insights", args[0]))
			if err != nil {
				return err
			}
			var insights api.EventInsights
			if err := json.Unmarshal(raw, &insights); err != nil {
				return fmt.Errorf("reading insights for %s: %w", args[0], err)
			}

			sections := []struct {
				level string
				label string
				stats map[string]interface{}
			}{
				{"qual", "Qualification", insights.Qual},
				{"playoff", "Playoff", insights.Playoff},
			}
			var rows [][]string
			for _, s := range sections {
				if level != "" && level != s.level {
					continue
				}
				for _, pair := range flattenInsightStats("", s.stats) {
					rows = append(rows, []string{s.label, humanizeStatKey(pair[0]), pair[1]})
				}
			}
			return outputTable(cmd, raw, []string{"Section", "Stat", "Value"}, rows)
		},
	}
	c.Flags().String("level", "", "Only this round: qual or playoff")
	return c
}
