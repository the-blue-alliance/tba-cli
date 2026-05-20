package cmd

import (
	"fmt"
	"strconv"

	"github.com/spf13/cobra"
	"github.com/the-blue-alliance/tba-cli/internal/api"
	"github.com/the-blue-alliance/tba-cli/internal/output"
)

var eventCmd = &cobra.Command{
	Use:   "event",
	Short: "Work with events",
}

var eventViewCmd = &cobra.Command{
	Use:   "view <key>",
	Short: "View event info",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := newClient(cmd)
		if err != nil {
			return err
		}
		var event api.Event
		if err := client.Get(fmt.Sprintf("/event/%s", args[0]), &event); err != nil {
			return err
		}
		return outputData(cmd, event, func() {
			week := "N/A"
			if event.Week != nil {
				week = strconv.Itoa(*event.Week)
			}
			output.PrintKeyValue(
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

var eventListCmd = &cobra.Command{
	Use:   "list",
	Short: "List events for a year",
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := newClient(cmd)
		if err != nil {
			return err
		}
		year, _ := cmd.Flags().GetInt("year")
		var events []api.Event
		if err := client.Get(fmt.Sprintf("/events/%d", year), &events); err != nil {
			return err
		}
		rows := make([][]string, len(events))
		for i, e := range events {
			rows[i] = []string{e.Key, e.Name, e.StartDate, e.EventTypeStr, output.FormatLocation(e.City, e.StateProv, e.Country)}
		}
		return outputTable(cmd, events, []string{"Key", "Name", "Start", "Type", "Location"}, rows)
	},
}

var eventTeamsCmd = &cobra.Command{
	Use:   "teams <key>",
	Short: "List teams at event",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := newClient(cmd)
		if err != nil {
			return err
		}
		var teams []api.Team
		if err := client.Get(fmt.Sprintf("/event/%s/teams", args[0]), &teams); err != nil {
			return err
		}
		rows := make([][]string, len(teams))
		for i, t := range teams {
			rows[i] = []string{fmt.Sprintf("%d", t.TeamNumber), t.Nickname, output.FormatLocation(t.City, t.StateProv, t.Country)}
		}
		return outputTable(cmd, teams, []string{"Number", "Name", "Location"}, rows)
	},
}

var eventMatchesCmd = &cobra.Command{
	Use:   "matches <key>",
	Short: "List matches at event",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := newClient(cmd)
		if err != nil {
			return err
		}
		var matches []api.Match
		if err := client.Get(fmt.Sprintf("/event/%s/matches", args[0]), &matches); err != nil {
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

var eventRankingsCmd = &cobra.Command{
	Use:   "rankings <key>",
	Short: "Show event rankings",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := newClient(cmd)
		if err != nil {
			return err
		}
		var rankings api.EventRankings
		if err := client.Get(fmt.Sprintf("/event/%s/rankings", args[0]), &rankings); err != nil {
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

var eventAlliancesCmd = &cobra.Command{
	Use:   "alliances <key>",
	Short: "Show event alliances",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := newClient(cmd)
		if err != nil {
			return err
		}
		var alliances []api.EventAlliance
		if err := client.Get(fmt.Sprintf("/event/%s/alliances", args[0]), &alliances); err != nil {
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

var eventAwardsCmd = &cobra.Command{
	Use:   "awards <key>",
	Short: "Show event awards",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := newClient(cmd)
		if err != nil {
			return err
		}
		var awards []api.Award
		if err := client.Get(fmt.Sprintf("/event/%s/awards", args[0]), &awards); err != nil {
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

var eventOPRsCmd = &cobra.Command{
	Use:   "oprs <key>",
	Short: "Show event OPRs",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := newClient(cmd)
		if err != nil {
			return err
		}
		var oprs api.EventOPRs
		if err := client.Get(fmt.Sprintf("/event/%s/oprs", args[0]), &oprs); err != nil {
			return err
		}
		var rows [][]string
		for team, opr := range oprs.OPRs {
			dpr := oprs.DPRs[team]
			ccwm := oprs.CCWMs[team]
			rows = append(rows, []string{output.TeamNumberFromKey(team), fmt.Sprintf("%.2f", opr), fmt.Sprintf("%.2f", dpr), fmt.Sprintf("%.2f", ccwm)})
		}
		return outputTable(cmd, oprs, []string{"Team", "OPR", "DPR", "CCWM"}, rows)
	},
}

var eventDistrictPointsCmd = &cobra.Command{
	Use:   "district-points <key>",
	Short: "Show event district points",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := newClient(cmd)
		if err != nil {
			return err
		}
		raw, err := client.GetRaw(fmt.Sprintf("/event/%s/district_points", args[0]))
		if err != nil {
			return err
		}
		return outputData(cmd, raw, func() {
			fmt.Println(string(raw))
		})
	},
}

var eventPredictionsCmd = &cobra.Command{
	Use:   "predictions <key>",
	Short: "Show event predictions",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := newClient(cmd)
		if err != nil {
			return err
		}
		raw, err := client.GetRaw(fmt.Sprintf("/event/%s/predictions", args[0]))
		if err != nil {
			return err
		}
		return outputData(cmd, raw, func() {
			fmt.Println(string(raw))
		})
	},
}

var eventInsightsCmd = &cobra.Command{
	Use:   "insights <key>",
	Short: "Show event insights",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := newClient(cmd)
		if err != nil {
			return err
		}
		raw, err := client.GetRaw(fmt.Sprintf("/event/%s/insights", args[0]))
		if err != nil {
			return err
		}
		return outputData(cmd, raw, func() {
			fmt.Println(string(raw))
		})
	},
}

func init() {
	eventListCmd.Flags().Int("year", currentYear(), "Year")
	eventListCmd.MarkFlagRequired("year")

	eventCmd.AddCommand(eventViewCmd)
	eventCmd.AddCommand(eventListCmd)
	eventCmd.AddCommand(eventTeamsCmd)
	eventCmd.AddCommand(eventMatchesCmd)
	eventCmd.AddCommand(eventRankingsCmd)
	eventCmd.AddCommand(eventAlliancesCmd)
	eventCmd.AddCommand(eventAwardsCmd)
	eventCmd.AddCommand(eventOPRsCmd)
	eventCmd.AddCommand(eventDistrictPointsCmd)
	eventCmd.AddCommand(eventPredictionsCmd)
	eventCmd.AddCommand(eventInsightsCmd)
}
