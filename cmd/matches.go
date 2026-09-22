package cmd

import (
	"fmt"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/the-blue-alliance/tba-cli/internal/api"
	"github.com/the-blue-alliance/tba-cli/internal/clierr"
	"github.com/the-blue-alliance/tba-cli/internal/frc"
	"github.com/the-blue-alliance/tba-cli/internal/output"
)

// matchHeaders are the columns of every match listing. `event matches` and
// `team matches` share them so that a user who learns one listing can read the
// other, and so that --columns and --sort take the same names in both.
//
// The time and its source are two columns rather than one decorated value: a
// script wants "Sat 14:32" without having to strip a marker off it, and anyone
// who does not care can drop the source with --columns.
var matchHeaders = []string{"Match", "Key", "Red", "Blue", "Score (R-B)", "Winner", "Time", "Time Source", "Status"}

// addMatchTableFlags adds the filters that every match listing accepts.
func addMatchTableFlags(c *cobra.Command) {
	c.Flags().String("team", "", "Only matches this team played in (e.g. 177 or frc177)")
	c.Flags().String("level", "", "Only this competition level: "+validMatchLevels)
	c.Flags().Bool("upcoming", false, "Only matches that have not been played, soonest first")
}

// validMatchLevels lists every accepted --level value, in help-text order.
const validMatchLevels = "qm, playoff, ef, qf, sf, f"

// renderMatches filters, orders and prints a match listing.
//
// playoffTypeFor supplies the bracket format a match's labels depend on. It is
// a function rather than a value because `team matches` for a whole season
// spans events that may have run different brackets.
func renderMatches(cmd *cobra.Command, matches []api.Match, playoffTypeFor func(api.Match) *int) error {
	matches, err := filterMatches(cmd, matches)
	if err != nil {
		return err
	}

	// An upcoming listing answers "what is next", so it is ordered by the
	// clock; everything else is ordered the way the event plays.
	if upcoming, _ := cmd.Flags().GetBool("upcoming"); upcoming {
		frc.SortByTime(matches)
	} else {
		frc.SortMatches(matches)
	}

	format, err := resolveFormat(cmd)
	if err != nil {
		return err
	}
	color, err := tableColorEnabled(cmd, format)
	if err != nil {
		return err
	}

	rows := make([][]string, len(matches))
	marked := false
	for i, m := range matches {
		red, blue := m.Alliances[frc.AllianceRed], m.Alliances[frc.AllianceBlue]
		redCell := strings.Join(frc.MarkedTeams(red), ", ")
		blueCell := strings.Join(frc.MarkedTeams(blue), ", ")
		if strings.ContainsAny(redCell, frc.MarkChars) || strings.ContainsAny(blueCell, frc.MarkChars) {
			marked = true
		}

		// An unplayed match scores -1/-1; printing that would look like a
		// result, so the cell stays empty and Status carries the news.
		score := ""
		if frc.Played(m) {
			score = fmt.Sprintf("%d-%d", red.Score, blue.Score)
		}

		epoch, source := frc.BestTime(m)
		rows[i] = []string{
			frc.MatchLabel(m, playoffTypeFor(m)),
			m.Key,
			output.Colorize(redCell, output.Red, color),
			output.Colorize(blueCell, output.Blue, color),
			score,
			colorizeAlliance(frc.Winner(m), color),
			frc.FormatTime(epoch, time.Local),
			source,
			frc.MatchStatus(m),
		}
	}

	if err := outputTable(cmd, matches, matchHeaders, rows); err != nil {
		return err
	}
	// The legend explains the marks in the table above it, so it is only worth
	// printing when a mark is actually there — and it goes to stderr, so it
	// never lands in a file the table was piped into.
	if marked && format == "table" {
		fmt.Fprintln(cmd.ErrOrStderr(), frc.Legend)
	}
	return nil
}

// tableColorEnabled reports whether a command may put ANSI escapes in its
// cells. Color belongs to the aligned table alone: csv, tsv and markdown are
// read by other programs, and JSON never goes through the table at all.
func tableColorEnabled(cmd *cobra.Command, format string) (bool, error) {
	mode, err := colorMode(cmd)
	if err != nil {
		return false, err
	}
	if format != "table" {
		return false, nil
	}
	return output.ColorEnabledFor(cmd.OutOrStdout(), mode), nil
}

// colorizeAlliance paints an alliance color in its own color. "tie" and the
// empty string of an unplayed match are left alone.
func colorizeAlliance(s string, color bool) string {
	switch s {
	case frc.AllianceRed:
		return output.Colorize(s, output.Red, color)
	case frc.AllianceBlue:
		return output.Colorize(s, output.Blue, color)
	default:
		return s
	}
}

// filterMatches applies --team, --level and --upcoming.
func filterMatches(cmd *cobra.Command, matches []api.Match) ([]api.Match, error) {
	team, _ := cmd.Flags().GetString("team")
	level, _ := cmd.Flags().GetString("level")
	upcoming, _ := cmd.Flags().GetBool("upcoming")

	wantTeam := ""
	if strings.TrimSpace(team) != "" {
		wantTeam = teamKey(team)
	}
	wantLevel, err := levelPredicate(level)
	if err != nil {
		return nil, err
	}

	out := make([]api.Match, 0, len(matches))
	for _, m := range matches {
		if wantTeam != "" && !frc.HasTeam(m, wantTeam) {
			continue
		}
		if !wantLevel(m) {
			continue
		}
		if upcoming && frc.Played(m) {
			continue
		}
		out = append(out, m)
	}
	return out, nil
}

// levelPredicate turns a --level value into a test. "playoff" is every
// elimination level at once, which is what someone asking for the playoffs
// means whether the event ran octofinals or a double-elimination bracket.
func levelPredicate(level string) (func(api.Match) bool, error) {
	switch strings.ToLower(strings.TrimSpace(level)) {
	case "":
		return func(api.Match) bool { return true }, nil
	case "qm", "qual", "quals", "qualification":
		return hasLevel(frc.LevelQual), nil
	case "playoff", "playoffs", "elim", "elims", "elimination":
		return func(m api.Match) bool { return !strings.EqualFold(m.CompLevel, frc.LevelQual) }, nil
	case frc.LevelEighthFinal, frc.LevelQuarterFinal, frc.LevelSemiFinal, frc.LevelFinal:
		return hasLevel(strings.ToLower(strings.TrimSpace(level))), nil
	default:
		return nil, clierr.Usage("invalid --level %q (want: %s)", level, validMatchLevels)
	}
}

func hasLevel(level string) func(api.Match) bool {
	return func(m api.Match) bool { return strings.EqualFold(m.CompLevel, level) }
}

// eventPlayoffType fetches an event only to learn how its playoff bracket is
// labelled. A failure is deliberately not fatal: a listing should not fail over
// a decoration, so the labels fall back to a guess from the season instead.
func eventPlayoffType(cmd *cobra.Command, client *api.Client, key string) *int {
	var event api.Event
	if err := client.Get(cmd.Context(), fmt.Sprintf("/event/%s", key), &event); err != nil {
		return nil
	}
	return event.PlayoffType
}

// constantPlayoffType adapts a single event's bracket format to the per-match
// lookup renderMatches wants.
func constantPlayoffType(playoffType *int) func(api.Match) *int {
	return func(api.Match) *int { return playoffType }
}
