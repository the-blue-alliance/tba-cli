package cmd

import (
	"fmt"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/the-blue-alliance/tba-cli/internal/api"
	"github.com/the-blue-alliance/tba-cli/internal/frc"
	"github.com/the-blue-alliance/tba-cli/internal/output"
)

func newTeamNextCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "next <number> [event]",
		Short: "Show a team's next match",
		Long: "Show the next match a team has not played yet.\n\n" +
			"With no event, the team's event for today is used, or the next one it is\n" +
			"going to if it is not competing right now.",
		Example: `  tba team next 177
  tba team next 177 2024cthar
  tba team next frc177 --all
  tba team next 177 --year 2024 --json`,
		Args: cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := newClient(cmd)
			if err != nil {
				return err
			}
			team := teamKey(args[0])
			now := nowFunc()

			event, err := resolveTeamEvent(cmd, client, team, args, now)
			if err != nil {
				return err
			}

			var matches []api.Match
			path := fmt.Sprintf("/team/%s/event/%s/matches", team, event.Key)
			if err := client.Get(cmd.Context(), path, &matches); err != nil {
				return err
			}
			upcoming := frc.Unplayed(matches)
			playoffTypeFor := constantPlayoffType(event.PlayoffType)

			if all, _ := cmd.Flags().GetBool("all"); all {
				return printMatchTable(cmd, upcoming, playoffTypeFor)
			}
			if len(upcoming) == 0 {
				return fmt.Errorf("no upcoming match for team %s at %s",
					output.TeamNumberFromKey(team), event.Key)
			}

			next := upcoming[0]
			return outputData(cmd, next, func() {
				format, _ := resolveFormat(cmd)
				color, _ := tableColorEnabled(cmd, format)
				printNextMatch(cmd, next, event, team, playoffTypeFor(next), color, now)
			})
		},
	}
	addYearFlag(c)
	c.Flags().Bool("all", false, "List every upcoming match as a table, not just the next one")
	return c
}

// resolveTeamEvent works out which event the question is about: the one named
// on the command line, or else the one the team is at today, or else the next
// one it is going to.
//
// The named case still fetches the event, because the bracket format and the
// event's name both come from it; a failure there is not fatal, so a key that
// cannot be looked up still gets its matches listed.
func resolveTeamEvent(cmd *cobra.Command, client *api.Client, team string, args []string, now time.Time) (api.Event, error) {
	if len(args) == 2 {
		if err := validateEventKey(args[1]); err != nil {
			return api.Event{}, err
		}
		event, ok := fetchEvent(cmd, client, args[1])
		if !ok {
			event = api.Event{Key: args[1]}
		}
		return event, nil
	}

	year, err := resolveYear(cmd)
	if err != nil {
		return api.Event{}, err
	}
	var events []api.Event
	if err := client.Get(cmd.Context(), fmt.Sprintf("/team/%s/events/%d", team, year), &events); err != nil {
		return api.Event{}, err
	}
	event, ok := frc.CurrentOrNextEvent(events, now)
	if !ok {
		return api.Event{}, fmt.Errorf("no current or upcoming event for team %s in %d",
			output.TeamNumberFromKey(team), year)
	}
	return event, nil
}

// printNextMatch answers the questions a team in the pits actually has: which
// match, on which alliance, from which station, with and against whom, and how
// long they have.
func printNextMatch(cmd *cobra.Command, m api.Match, event api.Event, team string, playoffType *int, color bool, now time.Time) {
	alliance := frc.AllianceOf(m, team)
	epoch, source := frc.BestTime(m)

	timeCell := ""
	if epoch != nil {
		timeCell = fmt.Sprintf("%s (%s)", frc.FormatTime(epoch, time.Local, false), source)
	}

	pairs := []string{
		"Event", eventLabel(event),
		"Match", fmt.Sprintf("%s (%s)", frc.MatchLabel(m, playoffType), m.Key),
		"Alliance", colorizeAlliance(alliance, color),
		"Station", stationLabel(m, team),
		"Partners", strings.Join(partnersOf(m, team, alliance), ", "),
		"Opponents", strings.Join(opponentsOf(m, alliance), ", "),
		"Time", timeCell,
	}
	if epoch != nil {
		// A match that should already have started is overdue, not "in -3m".
		d := time.Unix(*epoch, 0).Sub(now)
		if d < 0 {
			pairs = append(pairs, "Overdue by", frc.Magnitude(d))
		} else {
			pairs = append(pairs, "Starts in", frc.Magnitude(d))
		}
	}
	output.PrintKeyValue(cmd.OutOrStdout(), pairs...)
}

// eventLabel names an event for a heading, falling back to its key alone when
// the event itself could not be fetched.
func eventLabel(event api.Event) string {
	if event.Name == "" {
		return event.Key
	}
	return fmt.Sprintf("%s (%s)", event.Name, event.Key)
}

// stationLabel is the driver station a team plays from, as the field announces
// it: R1, B3.
func stationLabel(m api.Match, team string) string {
	alliance := frc.AllianceOf(m, team)
	station := frc.Station(m, team)
	if alliance == "" || station == 0 {
		return ""
	}
	return fmt.Sprintf("%s%d", strings.ToUpper(alliance[:1]), station)
}

// partnersOf lists the team's alliance without the team itself.
func partnersOf(m api.Match, team, alliance string) []string {
	a := m.Alliances[alliance]
	out := make([]string, 0, len(a.TeamKeys))
	for _, key := range a.TeamKeys {
		if key != team {
			out = append(out, frc.MarkTeam(key, a))
		}
	}
	return out
}

// opponentsOf lists the other alliance.
func opponentsOf(m api.Match, alliance string) []string {
	other := frc.AllianceBlue
	if alliance == frc.AllianceBlue {
		other = frc.AllianceRed
	}
	return frc.MarkedTeams(m.Alliances[other])
}
