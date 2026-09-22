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
// The time, how far off it is and where it came from are three columns rather
// than one decorated value: a script wants "Sat 14:32" without having to strip
// a marker off it, and anyone who does not care can drop the rest with
// --columns.
//
// Time carries the date as well when the listing spans more than one day.
// When counts down to a match still to come — "in 18m" — and is empty for one
// already played, whose result is the answer to "when".
var matchHeaders = []string{"Match", "Key", "Red", "Blue", "Score (R-B)", "Winner", "Time", "When", "Time Source", "Status"}

// eventColumn names the extra first column a season-wide `team matches`
// listing carries. It holds the event key rather than the event's name, which
// is far too wide to sit in front of every row.
const eventColumn = "Event"

// headersWithEvent is matchHeaders behind the Event column.
func headersWithEvent() []string {
	return append([]string{eventColumn}, matchHeaders...)
}

// addMatchTableFlags adds the filters that every match listing accepts.
func addMatchTableFlags(c *cobra.Command) {
	c.Flags().String("team", "", "Only matches this team played in (e.g. 177 or frc177)")
	c.Flags().String("level", "", "Only this competition level: "+validMatchLevels)
	c.Flags().Bool("upcoming", false, "Only matches that have not been played, soonest first")
}

// validMatchLevels lists every accepted --level value, in help-text order.
const validMatchLevels = "qm, playoff, ef, qf, sf, f"

// renderMatches filters, orders and prints a match listing for one event.
//
// playoffTypeFor supplies the bracket format a match's labels depend on. It is
// a function rather than a value because `team matches` for a whole season
// spans events that may have run different brackets.
func renderMatches(cmd *cobra.Command, matches []api.Match, playoffTypeFor func(api.Match) *int) error {
	matches, err := filterMatches(cmd, matches)
	if err != nil {
		return err
	}
	orderMatches(cmd, matches)
	return printMatchTable(cmd, matches, playoffTypeFor)
}

// renderSeasonMatches prints a listing that spans a whole season. A team plays
// several events in a year, and the same match numbers come round at each of
// them, so sorting the season as one list interleaves events: Qual 46 at
// Hartford, then Qual 46 at the district championship, then Qual 50 back at
// Hartford. Instead each event's matches are kept together, in the order the
// team played the events, and an Event column says which is which.
//
// order ranks the event keys (see frc.EventOrder). A nil or partial order —
// the team's event list could not be fetched — still groups the listing, by
// event key.
func renderSeasonMatches(cmd *cobra.Command, matches []api.Match, playoffTypeFor func(api.Match) *int, order map[string]int) error {
	matches, err := filterMatches(cmd, matches)
	if err != nil {
		return err
	}
	orderMatches(cmd, matches)
	frc.GroupByEvent(matches, order)
	return printMatchListing(cmd, matches, playoffTypeFor, true)
}

// orderMatches applies the order a listing is read in. An upcoming listing
// answers "what is next", so it is ordered by the clock; everything else is
// ordered the way the event plays.
func orderMatches(cmd *cobra.Command, matches []api.Match) {
	if upcoming, _ := cmd.Flags().GetBool("upcoming"); upcoming {
		frc.SortByTime(matches)
	} else {
		frc.SortMatches(matches)
	}
}

// printMatchTable renders matches in the order given. Callers that have
// already chosen an order — the next-match listing, say — use it directly.
func printMatchTable(cmd *cobra.Command, matches []api.Match, playoffTypeFor func(api.Match) *int) error {
	return printMatchListing(cmd, matches, playoffTypeFor, false)
}

// printMatchListing renders the shared match table, optionally with the Event
// column a season-wide listing needs.
func printMatchListing(cmd *cobra.Command, matches []api.Match, playoffTypeFor func(api.Match) *int, withEvent bool) error {
	format, err := resolveFormat(cmd)
	if err != nil {
		return err
	}
	color, err := tableColorEnabled(cmd, format)
	if err != nil {
		return err
	}

	headers := matchHeaders
	rows, marked := matchTableRows(matches, playoffTypeFor, color)
	if withEvent {
		headers = headersWithEvent()
		for i := range rows {
			rows[i] = append([]string{matches[i].EventKey}, rows[i]...)
		}
	}

	if err := outputTable(cmd, matches, headers, rows); err != nil {
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

// matchTableRows builds the cells of a match listing, in the order given, one
// row per match under matchHeaders. It also reports whether any cell carries a
// surrogate or DQ mark, which is what decides if the legend is worth printing.
//
// Whether the times carry a date is decided here, once, from the matches it is
// given: a listing that covers a single day does not need one on every row.
//
// color is passed in rather than resolved here so that the callers that write
// to a file — `event export` — can ask for the same cells without escapes.
func matchTableRows(matches []api.Match, playoffTypeFor func(api.Match) *int, color bool) (rows [][]string, marked bool) {
	return matchRows(matches, playoffTypeFor, color, frc.SpansDays(matches, time.Local), nowFunc())
}

// matchRows is matchTableRows for a caller that has already settled the date
// question for a wider table than the rows it is drawing now: `event watch`
// prints one poll's changes under a header the first poll sized.
func matchRows(matches []api.Match, playoffTypeFor func(api.Match) *int, color, withDate bool, now time.Time) (rows [][]string, marked bool) {
	rows = make([][]string, len(matches))
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
		// A countdown only means something for a match still to come. On a
		// played one it would say how long ago the result landed, which the
		// score already covers, and it would change every time the listing is
		// printed.
		when := ""
		if !frc.Played(m) {
			when = frc.RelativeEpoch(epoch, now)
		}
		rows[i] = []string{
			frc.MatchLabel(m, playoffTypeFor(m)),
			m.Key,
			output.Colorize(redCell, output.Red, color),
			output.Colorize(blueCell, output.Blue, color),
			score,
			colorizeAlliance(frc.Winner(m), color),
			frc.FormatTime(epoch, time.Local, withDate),
			when,
			source,
			frc.MatchStatus(m),
		}
	}
	return rows, marked
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

// colorizeAlliance paints the name of an alliance in its own color, whether it
// is spelled as the API does ("red") or as a label ("Red"). "tie" and the
// empty string of an unplayed match are left alone.
func colorizeAlliance(s string, color bool) string {
	switch {
	case strings.EqualFold(s, frc.AllianceRed):
		return output.Colorize(s, output.Red, color)
	case strings.EqualFold(s, frc.AllianceBlue):
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

// fetchEvent fetches an event for context rather than for its own sake: the
// bracket format that names its playoff matches, or the name to print above a
// listing. A failure is deliberately not fatal — a listing should not fail over
// a decoration — so it reports whether it got anything.
func fetchEvent(cmd *cobra.Command, client *api.Client, key string) (api.Event, bool) {
	var event api.Event
	if err := client.Get(cmd.Context(), fmt.Sprintf("/event/%s", key), &event); err != nil {
		return api.Event{}, false
	}
	return event, true
}

// eventPlayoffType is fetchEvent for the one field the match labels need. When
// the event cannot be had, the labels fall back to a guess from the season.
func eventPlayoffType(cmd *cobra.Command, client *api.Client, key string) *int {
	event, ok := fetchEvent(cmd, client, key)
	if !ok {
		return nil
	}
	return event.PlayoffType
}

// constantPlayoffType adapts a single event's bracket format to the per-match
// lookup renderMatches wants.
func constantPlayoffType(playoffType *int) func(api.Match) *int {
	return func(api.Match) *int { return playoffType }
}
