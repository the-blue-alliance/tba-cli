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

			named := ""
			if len(args) == 2 {
				named = args[1]
			}
			choice, err := resolveTeamEvent(cmd, client, team, named, now)
			if err != nil {
				return err
			}
			if !choice.found {
				// Out of season a team is simply not going anywhere. That is
				// an answer, not a failure.
				return printNoResult(cmd, noEventNote(team, choice.year, now, choice.hadEvents))
			}
			event := choice.event

			var matches []api.Match
			path := fmt.Sprintf("/team/%s/event/%s/matches", team, event.Key)
			if err := client.Get(cmd.Context(), path, &matches); err != nil {
				return err
			}
			// No matches at all at a named event is two different answers:
			// the schedule is not out yet, or the team was never going. The
			// event's roster settles it, and is only asked for in the one
			// case where the answer changes what is printed.
			if len(matches) == 0 && named != "" && !teamAtEvent(cmd, client, team, event.Key) {
				return clierr.NotFound("team %s was not at %s",
					output.TeamNumberFromKey(team), event.Key)
			}
			upcoming := frc.Unplayed(matches)
			playoffTypeFor := constantPlayoffType(event.PlayoffType)

			if all, _ := cmd.Flags().GetBool("all"); all {
				listing := matchListing{
					scope:        fmt.Sprintf("team %s at %s", output.TeamNumberFromKey(team), event.Key),
					hadMatches:   len(matches) > 0,
					onlyUpcoming: true,
				}
				return printMatchTable(cmd, upcoming, playoffTypeFor, listing)
			}
			if len(upcoming) == 0 {
				return printNoResult(cmd, noMatchNote(cmd, client, team, event, now))
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

// teamAtEvent reports whether a team is on an event's roster.
//
// It costs one request, so it is only asked when a team has no matches at an
// event at all, which is the one time the answer changes what is printed.
//
// A request that fails answers yes: an event whose roster cannot be fetched is
// no evidence that the team was absent, and the ordinary "no matches" note is
// a better answer than an error about a list nobody asked for.
func teamAtEvent(cmd *cobra.Command, client *api.Client, team, eventKey string) bool {
	var keys []string
	if err := client.Get(cmd.Context(), fmt.Sprintf("/event/%s/teams/keys", eventKey), &keys); err != nil {
		return true
	}
	for _, key := range keys {
		if strings.EqualFold(key, team) {
			return true
		}
	}
	return false
}

// teamEventChoice is which event a team question turned out to be about.
// found is false when the team is neither competing today nor signed up for
// anything later that season, which is most of the calendar; year is the
// season that was searched, and 0 when the event was named outright.
type teamEventChoice struct {
	event api.Event
	found bool
	year  int
	// hadEvents says whether the team was signed up for anything at all that
	// season, which is what tells "the season is over for them" from "this
	// team is not in the season at all". It is read off the event list the
	// search already fetched, so it costs nothing.
	hadEvents bool
}

// resolveTeamEvent works out which event the question is about: the one named
// on the command line, or else the one the team is at today, or else the next
// one it is going to. `team next` and `team standing` share it so that both
// auto-detect the same way.
//
// The named case still fetches the event, because the bracket format and the
// event's name both come from it; a failure there is not fatal, so a key that
// cannot be looked up still gets its matches listed.
func resolveTeamEvent(cmd *cobra.Command, client *api.Client, team, eventKey string, now time.Time) (teamEventChoice, error) {
	if eventKey != "" {
		if err := validateEventKey(eventKey); err != nil {
			return teamEventChoice{}, err
		}
		event, ok := fetchEvent(cmd, client, eventKey)
		if !ok {
			event = api.Event{Key: eventKey}
		}
		return teamEventChoice{event: event, found: true}, nil
	}

	year, err := resolveYear(cmd, client)
	if err != nil {
		return teamEventChoice{}, err
	}
	var events []api.Event
	if err := client.Get(cmd.Context(), fmt.Sprintf("/team/%s/events/%d", team, year), &events); err != nil {
		return teamEventChoice{year: year}, err
	}
	event, ok := frc.CurrentOrNextEvent(events, now)
	return teamEventChoice{event: event, found: ok, year: year, hadEvents: len(events) > 0}, nil
}

// lastCompetitionMonth is the last month of a season worth waiting out. After
// it, a team with nothing left in the current season is between seasons rather
// than done for the year, so the answer points at the next one.
const lastCompetitionMonth = time.August

// noEventNote explains a season with nothing left in it, and points somewhere
// that has an answer.
//
// hadEvents is whether the team was at anything that season, from the event
// list the search already fetched. It decides both halves of the advice. A
// team that competed has a season worth listing, so the note ends with the
// command that lists it — "try --year 2027" on its own was a dead end, since
// the next season's schedule does not exist yet and answers with nothing more
// to go on. A team with no events in the season searched is not in the season
// at all, and neither suggestion would lead anywhere, so neither is made.
func noEventNote(team string, year int, now time.Time, hadEvents bool) string {
	number := output.TeamNumberFromKey(team)
	note := fmt.Sprintf("no current or upcoming event for team %s in %d", number, year)
	if !hadEvents {
		return note
	}
	if now.Month() > lastCompetitionMonth && year <= now.Year() {
		note += fmt.Sprintf("; try --year %d", now.Year()+1)
	}
	return note + fmt.Sprintf("; see 'tba team events %s --year %d'", number, year)
}

// noMatchNote explains an event with no match left to play. An event that is
// over says so and, when the API will tell us, how it ended: "no matches left"
// reads like a schedule gap otherwise.
func noMatchNote(cmd *cobra.Command, client *api.Client, team string, event api.Event, now time.Time) string {
	number := output.TeamNumberFromKey(team)
	if !frc.Ended(event, now) {
		return fmt.Sprintf("no upcoming match for team %s at %s", number, event.Key)
	}
	note := fmt.Sprintf("%s ended %s; no matches left for %s", event.Key, event.EndDate, number)
	if outcome := teamPlayoffOutcome(cmd, client, team, event.Key, number); outcome != "" {
		note += "; " + outcome
	}
	return note
}

// teamPlayoffOutcome asks how the team's event finished. It is one extra
// request for a sentence, so a failure simply leaves the sentence off.
func teamPlayoffOutcome(cmd *cobra.Command, client *api.Client, team, eventKey, number string) string {
	var status api.TeamEventStatus
	path := fmt.Sprintf("/team/%s/event/%s/status", team, eventKey)
	if err := client.Get(cmd.Context(), path, &status); err != nil {
		return ""
	}
	return playoffOutcome(status.Playoff, number)
}

// playoffOutcome turns an event's playoff status into the end of a sentence.
// An event that is over but still says "playing" is stale, and gets nothing.
func playoffOutcome(p *api.AllianceStatus, number string) string {
	if p == nil {
		return ""
	}
	switch strings.ToLower(p.Status) {
	case "won":
		return number + " won the event"
	case "eliminated":
		return "eliminated in " + roundName(p.Level)
	default:
		return ""
	}
}

// roundName is the short name a round is spoken by in a sentence: "SF", but
// "the finals", which nobody calls "F".
func roundName(level string) string {
	if strings.EqualFold(strings.TrimSpace(level), frc.LevelFinal) {
		return "the finals"
	}
	return strings.ToUpper(strings.TrimSpace(level))
}

// printNoResult reports that there is nothing to show. It is not a failure —
// exit 0, nothing on stdout in table mode, and null for a JSON reader, which
// asked a question and needs to be told the answer is nothing.
func printNoResult(cmd *cobra.Command, note string) error {
	fmt.Fprintf(cmd.ErrOrStderr(), "note: %s\n", note)
	return outputData(cmd, nil, func() {})
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
