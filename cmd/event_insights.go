package cmd

import (
	"cmp"
	"encoding/json"
	"fmt"
	"math"
	"slices"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
	"github.com/the-blue-alliance/tba-cli/internal/api"
	"github.com/the-blue-alliance/tba-cli/internal/clierr"
	"github.com/the-blue-alliance/tba-cli/internal/frc"
	"github.com/the-blue-alliance/tba-cli/internal/output"
)

func newEventPredictionsCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "predictions <key>",
		Short: "Show TBA's match predictions",
		Long: `Show what TBA's model expects at an event.

By default the table is Match | Key | Red | Blue | Red Score | Blue Score |
Predicted Winner | Confidence, in play order -- qualification matches first,
then the playoff bracket -- rather than the alphabetical order the match keys
are in. Confidence is the model's own probability for the winner it picked.

Match is the label a person uses ("Qual 12", "SF 3"), and Red and Blue are the
teams, both from the event's match list; that list is fetched once alongside
the predictions, and an event without one still gets its labels from the match
keys. A match the model has nothing to say about comes back as a 0-0
prediction, and its winner and confidence are left blank rather than reported
as a coin flip. So is one the model calls even: an exact 50% or two equal
predicted scores is the model declining to pick, not a prediction of red.

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
				// The match list is a convenience -- labels and team lists --
				// so a failure costs those cells, not the table.
				var byKey map[string]api.Match
				if format != "json" {
					byKey = eventMatchesByKey(cmd, client, args[0])
				}
				headers, rows = matchPredictionTable(predictions.MatchPredictions, byKey)
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

// eventMatchesByKey fetches an event's matches, keyed by match key, for the
// labels and team lists the prediction table borrows from them. It is best
// effort: an event whose match list cannot be had still gets its table, with
// labels read out of the match keys and no teams.
func eventMatchesByKey(cmd *cobra.Command, client *api.Client, key string) map[string]api.Match {
	var matches []api.Match
	if err := client.Get(cmd.Context(), fmt.Sprintf("/event/%s/matches", key), &matches); err != nil {
		return nil
	}
	byKey := make(map[string]api.Match, len(matches))
	for _, m := range matches {
		byKey[m.Key] = m
	}
	return byKey
}

// matchPredictionTable lists every predicted match, qualification rounds first
// and then the playoff bracket, each in play order. byKey supplies the labels
// and team lists and may be nil or incomplete.
func matchPredictionTable(rounds *api.MatchPredictionRounds, byKey map[string]api.Match) ([]string, [][]string) {
	headers := []string{"Match", "Key", "Red", "Blue", "Red Score", "Blue Score", "Predicted Winner", "Confidence"}
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
			m, ok := byKey[k]
			if !ok {
				m = frc.MatchFromKey(k)
			}
			winner, confidence := allianceLabel(p.WinningAlliance), formatConfidence(p.Prob)
			if unmodelled(p) || tied(p) {
				// TBA answers for a match it cannot model with 0-0, and then
				// names a winner anyway at a confidence of about a half.
				// "Red / 50%" is not a prediction; it is the absence of one.
				winner, confidence = "", ""
			}
			rows = append(rows, []string{
				frc.MatchLabel(m, nil),
				k,
				strings.Join(frc.MarkedTeams(m.Alliances[frc.AllianceRed]), ", "),
				strings.Join(frc.MarkedTeams(m.Alliances[frc.AllianceBlue]), ", "),
				formatPredictedScore(p.Red.Score),
				formatPredictedScore(p.Blue.Score),
				winner,
				confidence,
			})
		}
	}
	return headers, rows
}

// unmodelled reports whether a prediction says nothing: both alliances are
// expected to score exactly nothing, which no real prediction does.
func unmodelled(p api.MatchPrediction) bool {
	return p.Red.Score == 0 && p.Blue.Score == 0
}

// tied reports whether the model has called it even: an exact half, or two
// alliances it expects to score the same. Naming a winner there is picking a
// side the model did not pick, and "Red / 50.00%" reads as a prediction when
// it is the model saying it cannot separate them.
func tied(p api.MatchPrediction) bool {
	if p.Prob != nil && *p.Prob == 0.5 {
		return true
	}
	return p.Red.Score == p.Blue.Score
}

// allianceLabel names the alliance a prediction picked, leaving an unpredicted
// match blank rather than inventing a winner.
//
// Lowercase, because that is how every other Winner column in the tool writes
// an alliance -- `event matches` and `match view` print the API's own "red" and
// "blue" -- and a column that reads "Red" in one table and "red" in the next
// cannot be compared, sorted or grepped across the two.
func allianceLabel(alliance string) string {
	return strings.ToLower(strings.TrimSpace(alliance))
}

// formatPredictedScore renders a predicted score, always to two decimals.
//
// A model's score is a number it computed, not a number anybody counted, and
// trimming the trailing zeroes made a column of them ragged -- "0", "28.5" and
// "36.42" one under the other, with the decimal point in three places, which is
// three digit-counts to compare two numbers. The Confidence column beside it
// settled the same question the same way.
func formatPredictedScore(f float64) string {
	if math.IsNaN(f) || math.IsInf(f, 0) {
		return ""
	}
	return strconv.FormatFloat(f, 'f', 2, 64)
}

// formatConfidence renders a probability as a percentage. A prediction without
// one stays blank, which is not the same as a confidence of zero.
//
// The decimals are fixed at two rather than trimmed, so that a column of them
// lines up: "50%" beside "51.38%" made the reader count digits to compare two
// numbers that are a percentage point apart.
func formatConfidence(prob *float64) string {
	if prob == nil {
		return ""
	}
	return strconv.FormatFloat(*prob*100, 'f', 2, 64) + "%"
}

// rankingPredictionTable lists the predicted finish for each team, best first.
// The API sends a list of numbers per team whose first entry is the rank; the
// rest bound it, and become the Range column.
func rankingPredictionTable(predictions []api.RankingPrediction) ([]string, [][]string) {
	headers := []string{"Team", "Predicted Rank", "Range"}
	ordered := make([]api.RankingPrediction, len(predictions))
	copy(ordered, predictions)
	slices.SortStableFunc(ordered, func(a, b api.RankingPrediction) int {
		if c := cmp.Compare(predictedRank(a), predictedRank(b)); c != 0 {
			return c
		}
		return frc.CompareTeamKeys(a.TeamKey, b.TeamKey)
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
		Short: "Show the statistics TBA computed for an event",
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
