package cmd

import (
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/the-blue-alliance/tba-cli/internal/api"
	"github.com/the-blue-alliance/tba-cli/internal/frc"
	"github.com/the-blue-alliance/tba-cli/internal/output"
)

// nowFunc is the clock the commands read, so a test can pin "now" and get the
// same countdown every run.
var nowFunc = time.Now

// tbaWebBase is the public site a match, team or event can be linked to.
const tbaWebBase = "https://www.thebluealliance.com"

func newMatchCmd() *cobra.Command {
	matchCmd := &cobra.Command{
		Use:     "match",
		Aliases: []string{"matches"},
		Short:   "Work with matches",
	}
	matchCmd.AddCommand(newMatchViewCmd())
	return matchCmd
}

func newMatchViewCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "view <key>",
		Short: "Show one match in full",
		Long: `Show one match in full: what it is called, when it is, both alliances by
driver station, the game's own score breakdown and any video.

The breakdown is printed as one table with a column per alliance, since what
a breakdown is for is comparing the two. Its fields change every season and are
documented nowhere, so they are ordered rather than interpreted: the total,
the ranking points, everything else that scores, the penalties, then the rest.
A field neither alliance did anything in is dropped, as are the season's own
constants, the thresholds a bonus is measured against; --full keeps every one.`,
		Example: `  tba match view 2024cthar_qm12
  tba match view 2024cthar_qm12 --full
  tba match view 2024cthar_sf3m1 --format json`,
		Args: exactArgs(1, "a match key (e.g. tba match view 2024cthar_qm12)"),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := validateMatchKey(args[0]); err != nil {
				return err
			}
			client, err := newClient(cmd)
			if err != nil {
				return err
			}
			var match api.Match
			if err := client.Get(cmd.Context(), fmt.Sprintf("/match/%s", args[0]), &match); err != nil {
				return err
			}

			// Only a semifinal's name depends on the event's bracket, so only
			// a semifinal is worth a second request to find out. Looking one
			// match up should not cost two round trips for nothing.
			var playoffType *int
			if frc.LabelDependsOnPlayoffType(match) {
				playoffType = eventPlayoffType(cmd, client, match.EventKey)
			}

			return outputData(cmd, match, func() {
				format, _ := resolveFormat(cmd)
				color, _ := tableColorEnabled(cmd, format)
				mode, _ := colorMode(cmd)
				full, _ := cmd.Flags().GetBool("full")
				printMatch(cmd.OutOrStdout(), match, matchView{
					playoffType: playoffType,
					color:       color,
					mode:        mode,
					full:        full,
					now:         nowFunc(),
				})
			})
		},
	}
	c.Flags().Bool("full", false, "Keep every score breakdown field, including the ones both alliances left at zero")
	return c
}

// matchView is how one match is to be drawn: the bracket its label depends on,
// whether escapes are allowed and how, whether the breakdown is shown whole,
// and the clock its countdown is measured against.
type matchView struct {
	playoffType *int
	color       bool
	mode        output.ColorMode
	full        bool
	now         time.Time
}

// printMatch writes the human view of a match: what it is, when it is, who
// played and what the game thought of it.
func printMatch(w io.Writer, m api.Match, view matchView) {
	red, blue := m.Alliances[frc.AllianceRed], m.Alliances[frc.AllianceBlue]
	playoffType, color, now := view.playoffType, view.color, view.now

	pairs := []string{
		"Match", frc.MatchLabel(m, playoffType),
		"Key", m.Key,
		"Event", m.EventKey,
		"Status", frc.MatchStatus(m),
		"Time", matchTimeDetail(m, now),
		colorizeAlliance("Red", color), stationList(red, frc.AllianceRed),
		"Red Score", scoreCell(m, frc.AllianceRed),
		colorizeAlliance("Blue", color), stationList(blue, frc.AllianceBlue),
		"Blue Score", scoreCell(m, frc.AllianceBlue),
		"Winner", colorizeAlliance(frc.Winner(m), color),
	}
	output.PrintKeyValue(w, pairs...)

	if marks := markLegendFor(red, blue); marks != "" {
		fmt.Fprintf(w, "\n%s\n", marks)
	}
	printBreakdown(w, m, view)
	printVideos(w, m)
}

// matchTimeDetail says when a match is, where that time came from, and how far
// off it is, since "Sat 14:32" alone does not say whether that has happened.
//
// The date is left off only for a match happening today, the same rule a
// listing follows: looking up a match from a past season and being told "Sat
// 14:32" named one of a season's worth of Saturdays, and the relative time
// beside it ("2 years ago") was the only clue which.
func matchTimeDetail(m api.Match, now time.Time) string {
	epoch, source := frc.BestTime(m)
	if epoch == nil {
		return ""
	}
	withDate := frc.NeedsDate([]api.Match{m}, time.Local, now)
	return fmt.Sprintf("%s (%s, %s)", frc.FormatTime(epoch, time.Local, withDate, now), source, frc.RelativeEpoch(epoch, now))
}

// stationList renders an alliance as its driver stations, which is how teams
// and field staff refer to a spot on the field: red 1, blue 3.
func stationList(a api.Alliance, color string) string {
	prefix := strings.ToUpper(color[:1])
	teams := make([]string, len(a.TeamKeys))
	for i, key := range a.TeamKeys {
		teams[i] = fmt.Sprintf("%s%d %s", prefix, i+1, frc.MarkTeam(key, a))
	}
	return strings.Join(teams, ", ")
}

// scoreCell is an alliance's score, or "" for a match that has not been
// played, whose -1 is a placeholder rather than a result.
func scoreCell(m api.Match, color string) string {
	if !frc.Played(m) {
		return ""
	}
	return strconv.Itoa(frc.Score(m, color))
}

func markLegendFor(alliances ...api.Alliance) string {
	for _, a := range alliances {
		for _, team := range frc.MarkedTeams(a) {
			if strings.ContainsAny(team, frc.MarkChars) {
				return frc.Legend
			}
		}
	}
	return ""
}

// printBreakdown writes the score breakdown as one table, an alliance to a
// column. Read down the two columns and the match explains itself; read as two
// separate lists, forty rows apart, it does not.
func printBreakdown(w io.Writer, m api.Match, view matchView) {
	rows := frc.CompareBreakdowns(m, view.full)
	if len(rows) == 0 {
		return
	}
	cells := make([][]string, len(rows))
	for i, row := range rows {
		cells[i] = []string{row.Label, row.Red, row.Blue}
	}
	fmt.Fprintln(w, "\nScore breakdown")
	headers := []string{
		"Stat",
		colorizeAlliance("Red", view.color),
		colorizeAlliance("Blue", view.color),
	}
	table := output.Table{Headers: headers, Rows: cells}
	// How the match was scored and how the game was played are two different
	// readings, and the first is short: a blank line keeps the totals from
	// being read as the head of a forty-row list.
	if n := frc.PointsBandLen(rows); n > 0 && n < len(rows) {
		table.Breaks = map[int]bool{n - 1: true}
	}
	// A failure here is a failure to write to stdout, which the recording
	// writer around it reports for the whole command.
	_ = output.Render(w, table, output.RenderOptions{Format: "table", Color: view.mode})
}

// printVideos lists the match's videos as links that can be clicked or curled.
func printVideos(w io.Writer, m api.Match) {
	if len(m.Videos) == 0 {
		return
	}
	fmt.Fprintln(w, "\nVideos")
	for _, v := range m.Videos {
		fmt.Fprintf(w, "  %s\n", videoURL(m, v))
	}
}

// videoURL turns a video reference into something to open. YouTube keys are
// the only type with a known URL shape; anything else falls back to the match's
// own page on The Blue Alliance, which lists it.
func videoURL(m api.Match, v api.Video) string {
	if v.Type == "youtube" && v.Key != "" {
		return "https://www.youtube.com/watch?v=" + v.Key
	}
	return fmt.Sprintf("%s/match/%s", tbaWebBase, m.Key)
}
