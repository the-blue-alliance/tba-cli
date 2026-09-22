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
	return &cobra.Command{
		Use:   "view <key>",
		Short: "View match info",
		Example: `  tba match view 2024cthar_qm12
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
				printMatch(cmd.OutOrStdout(), match, playoffType, color, nowFunc())
			})
		},
	}
}

// printMatch writes the human view of a match: what it is, when it is, who
// played and what the game thought of it.
func printMatch(w io.Writer, m api.Match, playoffType *int, color bool, now time.Time) {
	red, blue := m.Alliances[frc.AllianceRed], m.Alliances[frc.AllianceBlue]

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
	printBreakdowns(w, m, color)
	printVideos(w, m)
}

// matchTimeDetail says when a match is, where that time came from, and how far
// off it is, since "Sat 14:32" alone does not say whether that has happened.
func matchTimeDetail(m api.Match, now time.Time) string {
	epoch, source := frc.BestTime(m)
	if epoch == nil {
		return ""
	}
	return fmt.Sprintf("%s (%s, %s)", frc.FormatTime(epoch, time.Local), source, frc.RelativeEpoch(epoch, now))
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

// printBreakdowns writes each alliance's score breakdown. The fields change
// every season, so they are listed as they come rather than interpreted.
func printBreakdowns(w io.Writer, m api.Match, color bool) {
	for _, alliance := range []string{frc.AllianceRed, frc.AllianceBlue} {
		rows := frc.AllianceBreakdown(m, alliance)
		if len(rows) == 0 {
			continue
		}
		label := strings.ToUpper(alliance[:1]) + alliance[1:]
		fmt.Fprintf(w, "\nScore breakdown — %s\n", colorizeAlliance(label, color))
		pairs := make([]string, 0, len(rows)*2)
		for _, kv := range rows {
			pairs = append(pairs, "  "+kv.Key, kv.Value)
		}
		output.PrintKeyValue(w, pairs...)
	}
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
