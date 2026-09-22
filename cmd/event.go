package cmd

import (
	"fmt"
	"sort"
	"strconv"

	"github.com/spf13/cobra"
	"github.com/the-blue-alliance/tba-cli/internal/api"
	"github.com/the-blue-alliance/tba-cli/internal/output"
)

func newEventCmd() *cobra.Command {
	eventCmd := &cobra.Command{
		Use:     "event",
		Aliases: []string{"events"},
		Short:   "Work with events",
	}
	eventCmd.AddCommand(newEventViewCmd())
	eventCmd.AddCommand(newEventListCmd())
	eventCmd.AddCommand(newEventTeamsCmd())
	eventCmd.AddCommand(newEventMatchesCmd())
	eventCmd.AddCommand(newEventRankingsCmd())
	eventCmd.AddCommand(newEventAlliancesCmd())
	eventCmd.AddCommand(newEventAwardsCmd())
	eventCmd.AddCommand(newEventOPRsCmd())
	eventCmd.AddCommand(newEventDistrictPointsCmd())
	eventCmd.AddCommand(newEventPredictionsCmd())
	eventCmd.AddCommand(newEventInsightsCmd())
	return eventCmd
}

func newEventViewCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "view <key>",
		Short: "View event info",
		Example: `  tba event view 2024cthar
  tba event view 2024necmp --format json`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := validateEventKey(args[0]); err != nil {
				return err
			}
			client, err := newClient(cmd)
			if err != nil {
				return err
			}
			var event api.Event
			if err := client.Get(cmd.Context(), fmt.Sprintf("/event/%s", args[0]), &event); err != nil {
				return err
			}
			return outputData(cmd, event, func() {
				week := "N/A"
				if event.Week != nil {
					week = strconv.Itoa(*event.Week)
				}
				output.PrintKeyValue(cmd.OutOrStdout(),
					"Event", event.Name,
					"Key", event.Key,
					"Type", event.EventTypeStr,
					"Location", output.FormatLocation(event.City, event.StateProv, event.Country),
					"Venue", event.LocationName,
					"Dates", fmt.Sprintf("%s to %s", event.StartDate, event.EndDate),
					"Week", week,
				)
			})
		},
	}
}

func newEventListCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "list",
		Short: "List events for a year",
		Example: `  tba event list --year 2024
  tba events list --year 2024 --format csv`,
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := newClient(cmd)
			if err != nil {
				return err
			}
			year, _ := cmd.Flags().GetInt("year")
			var events []api.Event
			if err := client.Get(cmd.Context(), fmt.Sprintf("/events/%d", year), &events); err != nil {
				return err
			}
			rows := make([][]string, len(events))
			for i, e := range events {
				rows[i] = []string{e.Key, e.Name, e.StartDate, e.EventTypeStr, output.FormatLocation(e.City, e.StateProv, e.Country)}
			}
			return outputTable(cmd, events, []string{"Key", "Name", "Start", "Type", "Location"}, rows)
		},
	}
	c.Flags().Int("year", currentYear(), "Season year (default: current year)")
	return c
}

func newEventTeamsCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "teams <key>",
		Short: "List teams at event",
		Example: `  tba event teams 2024cthar
  tba event teams 2024cthar --format csv`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := validateEventKey(args[0]); err != nil {
				return err
			}
			client, err := newClient(cmd)
			if err != nil {
				return err
			}
			var teams []api.Team
			if err := client.Get(cmd.Context(), fmt.Sprintf("/event/%s/teams", args[0]), &teams); err != nil {
				return err
			}
			rows := make([][]string, len(teams))
			for i, t := range teams {
				rows[i] = []string{fmt.Sprintf("%d", t.TeamNumber), t.Nickname, output.FormatLocation(t.City, t.StateProv, t.Country)}
			}
			return outputTable(cmd, teams, []string{"Number", "Name", "Location"}, rows)
		},
	}
}

func newEventMatchesCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "matches <key>",
		Short: "List matches at event",
		Example: `  tba event matches 2024cthar --format csv
  tba event matches 2024cthar --jq '.[].key' -r`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := validateEventKey(args[0]); err != nil {
				return err
			}
			client, err := newClient(cmd)
			if err != nil {
				return err
			}
			var matches []api.Match
			if err := client.Get(cmd.Context(), fmt.Sprintf("/event/%s/matches", args[0]), &matches); err != nil {
				return err
			}
			rows := make([][]string, len(matches))
			for i, m := range matches {
				blueScore, redScore := "", ""
				if a, ok := m.Alliances["blue"]; ok {
					blueScore = strconv.Itoa(a.Score)
				}
				if a, ok := m.Alliances["red"]; ok {
					redScore = strconv.Itoa(a.Score)
				}
				rows[i] = []string{m.Key, m.CompLevel, redScore, blueScore, m.WinningAlliance}
			}
			return outputTable(cmd, matches, []string{"Key", "Level", "Red", "Blue", "Winner"}, rows)
		},
	}
}

func newEventRankingsCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "rankings <key>",
		Short: "Show event rankings",
		Example: `  tba event rankings 2024cthar
  tba event rankings 2024cthar --format markdown`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := validateEventKey(args[0]); err != nil {
				return err
			}
			client, err := newClient(cmd)
			if err != nil {
				return err
			}
			var rankings api.EventRankings
			if err := client.Get(cmd.Context(), fmt.Sprintf("/event/%s/rankings", args[0]), &rankings); err != nil {
				return err
			}
			rows := make([][]string, len(rankings.Rankings))
			for i, r := range rankings.Rankings {
				record := ""
				if r.Record != nil {
					record = fmt.Sprintf("%d-%d-%d", r.Record.Wins, r.Record.Losses, r.Record.Ties)
				}
				rows[i] = []string{strconv.Itoa(r.Rank), output.TeamNumberFromKey(r.TeamKey), record, strconv.Itoa(r.MatchesPlayed)}
			}
			return outputTable(cmd, rankings, []string{"Rank", "Team", "Record", "Played"}, rows)
		},
	}
}

func newEventAlliancesCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "alliances <key>",
		Short: "Show event alliances",
		Example: `  tba event alliances 2024cthar
  tba event alliances 2024cthar --format json`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := validateEventKey(args[0]); err != nil {
				return err
			}
			client, err := newClient(cmd)
			if err != nil {
				return err
			}
			var alliances []api.EventAlliance
			if err := client.Get(cmd.Context(), fmt.Sprintf("/event/%s/alliances", args[0]), &alliances); err != nil {
				return err
			}
			rows := make([][]string, len(alliances))
			for i, a := range alliances {
				name := fmt.Sprintf("Alliance %d", i+1)
				if a.Name != nil {
					name = *a.Name
				}
				picks := ""
				for j, p := range a.Picks {
					if j > 0 {
						picks += ", "
					}
					picks += output.TeamNumberFromKey(p)
				}
				rows[i] = []string{name, picks}
			}
			return outputTable(cmd, alliances, []string{"Alliance", "Picks"}, rows)
		},
	}
}

func newEventAwardsCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "awards <key>",
		Short: "Show event awards",
		Example: `  tba event awards 2024cthar
  tba event awards 2024cthar --format csv`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := validateEventKey(args[0]); err != nil {
				return err
			}
			client, err := newClient(cmd)
			if err != nil {
				return err
			}
			var awards []api.Award
			if err := client.Get(cmd.Context(), fmt.Sprintf("/event/%s/awards", args[0]), &awards); err != nil {
				return err
			}
			rows := make([][]string, len(awards))
			for i, a := range awards {
				recipient := ""
				if len(a.Recipients) > 0 {
					r := a.Recipients[0]
					if r.TeamKey != nil {
						recipient = output.TeamNumberFromKey(*r.TeamKey)
					}
					if r.Awardee != nil {
						if recipient != "" {
							recipient += " - "
						}
						recipient += *r.Awardee
					}
				}
				rows[i] = []string{a.Name, recipient}
			}
			return outputTable(cmd, awards, []string{"Award", "Recipient"}, rows)
		},
	}
}

func newEventOPRsCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "oprs <key>",
		Short: "Show event OPRs",
		Example: `  tba event oprs 2024cthar
  tba event oprs 2024cthar --format csv`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := validateEventKey(args[0]); err != nil {
				return err
			}
			client, err := newClient(cmd)
			if err != nil {
				return err
			}
			var oprs api.EventOPRs
			if err := client.Get(cmd.Context(), fmt.Sprintf("/event/%s/oprs", args[0]), &oprs); err != nil {
				return err
			}
			teamKeys := make([]string, 0, len(oprs.OPRs))
			for team := range oprs.OPRs {
				teamKeys = append(teamKeys, team)
			}
			// Map iteration order is random; sort so the output is stable.
			sort.Slice(teamKeys, func(i, j int) bool {
				a, errA := strconv.Atoi(output.TeamNumberFromKey(teamKeys[i]))
				b, errB := strconv.Atoi(output.TeamNumberFromKey(teamKeys[j]))
				if errA == nil && errB == nil {
					return a < b
				}
				return teamKeys[i] < teamKeys[j]
			})
			var rows [][]string
			for _, team := range teamKeys {
				rows = append(rows, []string{
					output.TeamNumberFromKey(team),
					fmt.Sprintf("%.2f", oprs.OPRs[team]),
					fmt.Sprintf("%.2f", oprs.DPRs[team]),
					fmt.Sprintf("%.2f", oprs.CCWMs[team]),
				})
			}
			return outputTable(cmd, oprs, []string{"Team", "OPR", "DPR", "CCWM"}, rows)
		},
	}
}

func newEventDistrictPointsCmd() *cobra.Command {
	return newRawEventCmd("district-points", "Show event district points", "district_points",
		`  tba event district-points 2024cthar
  tba event district-points 2024cthar --jq '.points.frc177.total'`)
}

func newEventPredictionsCmd() *cobra.Command {
	return newRawEventCmd("predictions", "Show event predictions", "predictions",
		`  tba event predictions 2024cthar
  tba event predictions 2024cthar --format json`)
}

func newEventInsightsCmd() *cobra.Command {
	return newRawEventCmd("insights", "Show event insights", "insights",
		`  tba event insights 2024cthar
  tba event insights 2024cthar --jq .qual.high_score`)
}

// newRawEventCmd builds a command that passes an event sub-resource through
// untouched, since these endpoints have no stable schema.
func newRawEventCmd(use, short, resource, example string) *cobra.Command {
	return &cobra.Command{
		Use:     use + " <key>",
		Short:   short,
		Example: example,
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := validateEventKey(args[0]); err != nil {
				return err
			}
			client, err := newClient(cmd)
			if err != nil {
				return err
			}
			raw, err := client.GetRaw(cmd.Context(), fmt.Sprintf("/event/%s/%s", args[0], resource))
			if err != nil {
				return err
			}
			return outputData(cmd, raw, func() {
				fmt.Fprintln(cmd.OutOrStdout(), string(raw))
			})
		},
	}
}
