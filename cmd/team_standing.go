package cmd

import (
	"fmt"
	"strconv"

	"github.com/spf13/cobra"
	"github.com/the-blue-alliance/tba-cli/internal/api"
	"github.com/the-blue-alliance/tba-cli/internal/frc"
	"github.com/the-blue-alliance/tba-cli/internal/output"
)

func newTeamStandingCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "standing <number>",
		Short: "Show how a team stands at an event",
		Long: "Show a team's rank, record, alliance and playoff result at one event.\n\n" +
			"Each section only appears once that part of the event has happened, so a\n" +
			"team asked about before alliance selection is simply not selected yet.",
		Example: `  tba team standing 177 --event 2024cthar
  tba team standing frc177 --event 2024cthar --json`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			eventKey, _ := cmd.Flags().GetString("event")
			if err := validateEventKey(eventKey); err != nil {
				return err
			}
			client, err := newClient(cmd)
			if err != nil {
				return err
			}
			team := teamKey(args[0])

			var status api.TeamEventStatus
			path := fmt.Sprintf("/team/%s/event/%s/status", team, eventKey)
			if err := client.Get(cmd.Context(), path, &status); err != nil {
				return err
			}
			return outputData(cmd, status, func() {
				output.PrintKeyValue(cmd.OutOrStdout(), standingPairs(status, team, eventKey)...)
			})
		},
	}
	c.Flags().String("event", "", "Event key, e.g. 2024cthar (required)")
	_ = c.MarkFlagRequired("event")
	return c
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
	for i, value := range r.SortOrders {
		name := fmt.Sprintf("Sort Order %d", i+1)
		precision := 2
		if i < len(qual.SortOrderInfo) {
			name = qual.SortOrderInfo[i].Name
			precision = qual.SortOrderInfo[i].Precision
		}
		pairs = append(pairs, name, strconv.FormatFloat(value, 'f', precision, 64))
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
