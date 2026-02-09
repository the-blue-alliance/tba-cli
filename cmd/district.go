package cmd

import (
	"fmt"
	"strconv"

	"github.com/spf13/cobra"
	"github.com/the-blue-alliance/tba-cli/internal/api"
	"github.com/the-blue-alliance/tba-cli/internal/output"
)

var districtCmd = &cobra.Command{
	Use:   "district",
	Short: "Work with districts",
}

var districtListCmd = &cobra.Command{
	Use:   "list",
	Short: "List districts for a year",
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := newClient(cmd)
		if err != nil {
			return err
		}
		year, _ := cmd.Flags().GetInt("year")
		var districts []api.District
		if err := client.Get(fmt.Sprintf("/districts/%d", year), &districts); err != nil {
			return err
		}
		return outputData(cmd, districts, func() {
			rows := make([][]string, len(districts))
			for i, d := range districts {
				rows[i] = []string{d.Key, d.DisplayName, d.Abbreviation}
			}
			output.PrintTable([]string{"Key", "Name", "Abbreviation"}, rows)
		})
	},
}

var districtEventsCmd = &cobra.Command{
	Use:   "events <key>",
	Short: "List district events",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := newClient(cmd)
		if err != nil {
			return err
		}
		var events []api.Event
		if err := client.Get(fmt.Sprintf("/district/%s/events", args[0]), &events); err != nil {
			return err
		}
		return outputData(cmd, events, func() {
			rows := make([][]string, len(events))
			for i, e := range events {
				rows[i] = []string{e.Key, e.Name, e.StartDate}
			}
			output.PrintTable([]string{"Key", "Name", "Start Date"}, rows)
		})
	},
}

var districtTeamsCmd = &cobra.Command{
	Use:   "teams <key>",
	Short: "List district teams",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := newClient(cmd)
		if err != nil {
			return err
		}
		var teams []api.Team
		if err := client.Get(fmt.Sprintf("/district/%s/teams", args[0]), &teams); err != nil {
			return err
		}
		return outputData(cmd, teams, func() {
			rows := make([][]string, len(teams))
			for i, t := range teams {
				rows[i] = []string{fmt.Sprintf("%d", t.TeamNumber), t.Nickname, output.FormatLocation(t.City, t.StateProv, t.Country)}
			}
			output.PrintTable([]string{"Number", "Name", "Location"}, rows)
		})
	},
}

var districtRankingsCmd = &cobra.Command{
	Use:   "rankings <key>",
	Short: "Show district rankings",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := newClient(cmd)
		if err != nil {
			return err
		}
		var rankings []api.DistrictRanking
		if err := client.Get(fmt.Sprintf("/district/%s/rankings", args[0]), &rankings); err != nil {
			return err
		}
		return outputData(cmd, rankings, func() {
			rows := make([][]string, len(rankings))
			for i, r := range rankings {
				rows[i] = []string{strconv.Itoa(r.Rank), output.TeamNumberFromKey(r.TeamKey), strconv.Itoa(r.PointTotal)}
			}
			output.PrintTable([]string{"Rank", "Team", "Points"}, rows)
		})
	},
}

func init() {
	districtListCmd.Flags().Int("year", currentYear(), "Year")
	districtListCmd.MarkFlagRequired("year")

	districtCmd.AddCommand(districtListCmd)
	districtCmd.AddCommand(districtEventsCmd)
	districtCmd.AddCommand(districtTeamsCmd)
	districtCmd.AddCommand(districtRankingsCmd)
}
