package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
	"github.com/the-blue-alliance/tba-cli/internal/api"
	"github.com/the-blue-alliance/tba-cli/internal/clierr"
	"github.com/the-blue-alliance/tba-cli/internal/frc"
	"github.com/the-blue-alliance/tba-cli/internal/output"
)

func newTeamStandingCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "standing <number> [event]",
		Short: "Show how a team stands at an event",
		Long: "Show a team's rank, record, alliance and playoff result at one event.\n\n" +
			"With no event, the team's event for today is used, or the next one it is\n" +
			"going to if it is not competing right now, the same way `team next` does.\n\n" +
			"Each section only appears once that part of the event has happened, so a\n" +
			"team asked about before alliance selection is simply not selected yet.",
		Example: `  tba team standing 177
  tba team standing 177 2024cthar
  tba team standing frc177 --event 2024cthar --json`,
		Args: cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			eventKey, err := standingEventKey(cmd, args)
			if err != nil {
				return err
			}
			client, err := newClient(cmd)
			if err != nil {
				return err
			}
			team := teamKey(args[0])

			if eventKey == "" {
				// No event named: the question is about wherever the team is
				// now, which is what `team next` asks too.
				now := nowFunc()
				choice, err := resolveTeamEvent(cmd, client, team, "", now)
				if err != nil {
					return err
				}
				if !choice.found {
					return printNoResult(cmd, noEventNote(team, choice.year, now))
				}
				eventKey = choice.event.Key
			}

			// The body is read raw because its most important answer is a
			// bare null, which decodes into a struct full of nils that is
			// indistinguishable from a team standing at the start line.
			path := fmt.Sprintf("/team/%s/event/%s/status", team, eventKey)
			raw, err := client.GetRaw(cmd.Context(), path)
			if err != nil {
				return err
			}
			if isNullStatus(raw) {
				return clierr.NotFound("team %s was not at %s",
					output.TeamNumberFromKey(team), eventKey)
			}
			var status api.TeamEventStatus
			if err := json.Unmarshal(raw, &status); err != nil {
				return fmt.Errorf("reading the status of team %s at %s: %w",
					output.TeamNumberFromKey(team), eventKey, err)
			}
			return outputData(cmd, status, func() {
				output.PrintKeyValue(cmd.OutOrStdout(), standingPairs(status, team, eventKey)...)
			})
		},
	}
	addYearFlag(c)
	c.Flags().String("event", "", "Event key, e.g. 2024cthar (default: the team's current or next event)")
	return c
}

// standingEventKey reads the event from wherever it was given: the argument,
// which is how the other team commands take it, or --event, which this one
// used to require. It returns "" when neither is there, which means "work it
// out".
func standingEventKey(cmd *cobra.Command, args []string) (string, error) {
	flag := strings.TrimSpace(mustString(cmd, "event"))
	positional := ""
	if len(args) == 2 {
		positional = strings.TrimSpace(args[1])
	}

	key := positional
	switch {
	case positional != "" && flag != "" && positional != flag:
		return "", clierr.Usage("event given twice, as %q and --event %s: name it once", positional, flag)
	case positional == "":
		key = flag
	}
	if key == "" {
		return "", nil
	}
	if err := validateEventKey(key); err != nil {
		return "", err
	}
	return key, nil
}

// isNullStatus reports whether the API answered "this team has no status at
// this event", which it does with a literal null rather than a 404.
//
// A team that simply has not played yet gets an object whose sections are all
// null, and that is a different answer: it is at the event, with nothing to
// report yet. Only the null document means the team was never there.
func isNullStatus(raw []byte) bool {
	trimmed := bytes.TrimSpace(raw)
	return len(trimmed) == 0 || bytes.Equal(trimmed, []byte("null"))
}

// mustString reads a string flag the command is known to have.
func mustString(cmd *cobra.Command, name string) string {
	v, _ := cmd.Flags().GetString(name)
	return v
}

// standingPairs builds the key/value view of a team's event status. Every
// section of the API's answer is optional, and each absence means something
// different — qualification not started, not picked, playoffs not reached — so
// each is spelled out rather than left blank.
func standingPairs(status api.TeamEventStatus, team, eventKey string) []string {
	pairs := []string{
		"Team", output.TeamNumberFromKey(team),
		"Event", eventKey,
	}
	pairs = append(pairs, qualPairs(status.Qual)...)
	pairs = append(pairs, "Alliance", alliancePhrase(status.Alliance, team))
	pairs = append(pairs, "Playoff", playoffPhrase(status.Playoff))
	if overall := frc.StripHTML(status.OverallStatusStr); overall != "" {
		pairs = append(pairs, "Status", overall)
	}
	return pairs
}

func qualPairs(qual *api.TeamEventQualStatus) []string {
	if qual == nil || qual.Ranking == nil {
		return []string{"Rank", "not ranked yet"}
	}
	r := qual.Ranking

	rank := strconv.Itoa(r.Rank)
	if qual.NumTeams > 0 {
		rank = fmt.Sprintf("%d of %d", r.Rank, qual.NumTeams)
	}
	pairs := []string{
		"Rank", rank,
		"Record", frc.Record(r.Record),
		"Played", strconv.Itoa(r.MatchesPlayed),
	}
	// The sort orders are the season's own tiebreakers, and only the event
	// knows what they are called, so they are named from sort_order_info.
	//
	// The API sometimes carries more numbers than names — a padding column the
	// season does not use. An unnamed number is not a statistic, it is noise,
	// so it is dropped rather than printed as "Sort Order 6: 0.00". `event
	// rankings` shows the same columns for the same reason.
	for i, info := range qual.SortOrderInfo {
		if i >= len(r.SortOrders) {
			break
		}
		pairs = append(pairs, info.Name, strconv.FormatFloat(r.SortOrders[i], 'f', info.Precision, 64))
	}
	return pairs
}

func alliancePhrase(a *api.TeamEventAllianceStatus, team string) string {
	if a == nil {
		return "not selected"
	}
	name := a.Name
	if name == "" {
		name = fmt.Sprintf("Alliance %d", a.Number)
	}
	return fmt.Sprintf("%s (%s)", name, frc.PickPosition(*a, team))
}

func playoffPhrase(p *api.AllianceStatus) string {
	if p == nil {
		return "not started"
	}
	phrase := frc.LevelName(p.Level)
	if p.Status != "" {
		phrase += " — " + p.Status
	}
	if record := frc.Record(p.Record); record != "" {
		phrase += " (" + record + ")"
	}
	return phrase
}
