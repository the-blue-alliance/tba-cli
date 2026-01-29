package cmd

import (
	"fmt"
	"strconv"

	"github.com/spf13/cobra"
	"github.com/the-blue-alliance/tba-cli/internal/api"
	"github.com/the-blue-alliance/tba-cli/internal/output"
)

var teamCmd = &cobra.Command{
	Use:   "team",
	Short: "Work with teams",
}

var teamViewCmd = &cobra.Command{
	Use:   "view <number>",
	Short: "View team info",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := api.NewClient()
		if err != nil {
			return err
		}
		var team api.Team
		if err := client.Get(fmt.Sprintf("/team/frc%s", args[0]), &team); err != nil {
			return err
		}
		return outputData(cmd, team, func() {
			output.PrintKeyValue(
				"Team", fmt.Sprintf("%d - %s", team.TeamNumber, team.Nickname),
				"Name", team.Name,
				"Location", output.FormatLocation(team.City, team.StateProv, team.Country),
				"Rookie Year", fmt.Sprintf("%d", team.RookieYear),
				"Website", team.Website,
			)
		})
	},
}

var teamListCmd = &cobra.Command{
	Use:   "list",
	Short: "List teams",
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := api.NewClient()
		if err != nil {
			return err
		}
		year, _ := cmd.Flags().GetInt("year")
		var allTeams []api.Team
		for page := 0; ; page++ {
			var teams []api.Team
			path := fmt.Sprintf("/teams/%d/%d", year, page)
			if err := client.Get(path, &teams); err != nil {
				return err
			}
			if len(teams) == 0 {
				break
			}
			allTeams = append(allTeams, teams...)
		}
		return outputData(cmd, allTeams, func() {
			rows := make([][]string, len(allTeams))
			for i, t := range allTeams {
				rows[i] = []string{
					fmt.Sprintf("%d", t.TeamNumber),
					t.Nickname,
					output.FormatLocation(t.City, t.StateProv, t.Country),
				}
			}
			output.PrintTable([]string{"Number", "Name", "Location"}, rows)
		})
	},
}

var teamEventsCmd = &cobra.Command{
	Use:   "events <number>",
	Short: "List team events",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := api.NewClient()
		if err != nil {
			return err
		}
		year, _ := cmd.Flags().GetInt("year")
		var events []api.Event
		if err := client.Get(fmt.Sprintf("/team/frc%s/events/%d", args[0], year), &events); err != nil {
			return err
		}
		return outputData(cmd, events, func() {
			rows := make([][]string, len(events))
			for i, e := range events {
				rows[i] = []string{e.Key, e.Name, e.StartDate, output.FormatLocation(e.City, e.StateProv, e.Country)}
			}
			output.PrintTable([]string{"Key", "Name", "Start Date", "Location"}, rows)
		})
	},
}

var teamMatchesCmd = &cobra.Command{
	Use:   "matches <number>",
	Short: "List team matches for a year",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := api.NewClient()
		if err != nil {
			return err
		}
		year, _ := cmd.Flags().GetInt("year")
		var matches []api.Match
		if err := client.Get(fmt.Sprintf("/team/frc%s/matches/%d", args[0], year), &matches); err != nil {
			return err
		}
		return outputData(cmd, matches, func() {
			rows := make([][]string, len(matches))
			for i, m := range matches {
				rows[i] = []string{m.Key, m.CompLevel, m.WinningAlliance}
			}
			output.PrintTable([]string{"Key", "Level", "Winner"}, rows)
		})
	},
}

var teamAwardsCmd = &cobra.Command{
	Use:   "awards <number>",
	Short: "List team awards",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := api.NewClient()
		if err != nil {
			return err
		}
		year, _ := cmd.Flags().GetInt("year")
		path := fmt.Sprintf("/team/frc%s/awards", args[0])
		if year > 0 {
			path = fmt.Sprintf("/team/frc%s/awards/%d", args[0], year)
		}
		var awards []api.Award
		if err := client.Get(path, &awards); err != nil {
			return err
		}
		return outputData(cmd, awards, func() {
			rows := make([][]string, len(awards))
			for i, a := range awards {
				rows[i] = []string{a.EventKey, a.Name, strconv.Itoa(a.Year)}
			}
			output.PrintTable([]string{"Event", "Award", "Year"}, rows)
		})
	},
}

var teamMediaCmd = &cobra.Command{
	Use:   "media <number>",
	Short: "List team media",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := api.NewClient()
		if err != nil {
			return err
		}
		year, _ := cmd.Flags().GetInt("year")
		var media []api.Media
		if err := client.Get(fmt.Sprintf("/team/frc%s/media/%d", args[0], year), &media); err != nil {
			return err
		}
		return outputData(cmd, media, func() {
			rows := make([][]string, len(media))
			for i, m := range media {
				rows[i] = []string{m.Type, m.ForeignKey, m.ViewURL}
			}
			output.PrintTable([]string{"Type", "Key", "URL"}, rows)
		})
	},
}

var teamRobotsCmd = &cobra.Command{
	Use:   "robots <number>",
	Short: "List team robots",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := api.NewClient()
		if err != nil {
			return err
		}
		var robots []api.Robot
		if err := client.Get(fmt.Sprintf("/team/frc%s/robots", args[0]), &robots); err != nil {
			return err
		}
		return outputData(cmd, robots, func() {
			rows := make([][]string, len(robots))
			for i, r := range robots {
				rows[i] = []string{strconv.Itoa(r.Year), r.RobotName}
			}
			output.PrintTable([]string{"Year", "Robot Name"}, rows)
		})
	},
}

var teamDistrictsCmd = &cobra.Command{
	Use:   "districts <number>",
	Short: "List team districts",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := api.NewClient()
		if err != nil {
			return err
		}
		var districts []api.District
		if err := client.Get(fmt.Sprintf("/team/frc%s/districts", args[0]), &districts); err != nil {
			return err
		}
		return outputData(cmd, districts, func() {
			rows := make([][]string, len(districts))
			for i, d := range districts {
				rows[i] = []string{d.Key, d.DisplayName, strconv.Itoa(d.Year)}
			}
			output.PrintTable([]string{"Key", "Name", "Year"}, rows)
		})
	},
}

func init() {
	teamListCmd.Flags().Int("year", currentYear(), "Year")
	teamEventsCmd.Flags().Int("year", currentYear(), "Year")
	teamMatchesCmd.Flags().Int("year", currentYear(), "Year")
	teamMatchesCmd.MarkFlagRequired("year")
	teamAwardsCmd.Flags().Int("year", 0, "Year (optional)")
	teamMediaCmd.Flags().Int("year", currentYear(), "Year")
	teamMediaCmd.MarkFlagRequired("year")

	teamCmd.AddCommand(teamViewCmd)
	teamCmd.AddCommand(teamListCmd)
	teamCmd.AddCommand(teamEventsCmd)
	teamCmd.AddCommand(teamMatchesCmd)
	teamCmd.AddCommand(teamAwardsCmd)
	teamCmd.AddCommand(teamMediaCmd)
	teamCmd.AddCommand(teamRobotsCmd)
	teamCmd.AddCommand(teamDistrictsCmd)
}
