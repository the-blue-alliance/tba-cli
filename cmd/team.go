package cmd

import (
	"fmt"
	"strconv"

	"github.com/spf13/cobra"
	"github.com/the-blue-alliance/tba-cli/internal/api"
	"github.com/the-blue-alliance/tba-cli/internal/output"
)

func newTeamCmd() *cobra.Command {
	teamCmd := &cobra.Command{
		Use:     "team",
		Aliases: []string{"teams"},
		Short:   "Work with teams",
	}
	teamCmd.AddCommand(newTeamViewCmd())
	teamCmd.AddCommand(newTeamListCmd())
	teamCmd.AddCommand(newTeamEventsCmd())
	teamCmd.AddCommand(newTeamMatchesCmd())
	teamCmd.AddCommand(newTeamAwardsCmd())
	teamCmd.AddCommand(newTeamMediaCmd())
	teamCmd.AddCommand(newTeamRobotsCmd())
	teamCmd.AddCommand(newTeamDistrictsCmd())
	return teamCmd
}

func newTeamViewCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "view <number>",
		Short: "View team info",
		Example: `  tba team view 177
  tba team view frc177 --format json
  tba team view 1073 --jq .nickname -r`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := newClient(cmd)
			if err != nil {
				return err
			}
			var team api.Team
			if err := client.Get(fmt.Sprintf("/team/%s", teamKey(args[0])), &team); err != nil {
				return err
			}
			return outputData(cmd, team, func() {
				output.PrintKeyValue(cmd.OutOrStdout(),
					"Team", fmt.Sprintf("%d - %s", team.TeamNumber, team.Nickname),
					"Name", team.Name,
					"Location", output.FormatLocation(team.City, team.StateProv, team.Country),
					"Rookie Year", fmt.Sprintf("%d", team.RookieYear),
					"Website", team.Website,
				)
			})
		},
	}
}

func newTeamListCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "list",
		Short: "List teams",
		Example: `  tba team list --year 2024
  tba teams list --year 2024 --format csv`,
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := newClient(cmd)
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
			rows := make([][]string, len(allTeams))
			for i, t := range allTeams {
				rows[i] = []string{
					fmt.Sprintf("%d", t.TeamNumber),
					t.Nickname,
					output.FormatLocation(t.City, t.StateProv, t.Country),
				}
			}
			return outputTable(cmd, allTeams, []string{"Number", "Name", "Location"}, rows)
		},
	}
	c.Flags().Int("year", currentYear(), "Season year (default: current year)")
	return c
}

func newTeamEventsCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "events <number>",
		Short: "List team events",
		Example: `  tba team events 177 --year 2024
  tba team events frc177 --year 2024 --format csv`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := newClient(cmd)
			if err != nil {
				return err
			}
			year, _ := cmd.Flags().GetInt("year")
			var events []api.Event
			if err := client.Get(fmt.Sprintf("/team/%s/events/%d", teamKey(args[0]), year), &events); err != nil {
				return err
			}
			rows := make([][]string, len(events))
			for i, e := range events {
				rows[i] = []string{e.Key, e.Name, e.StartDate, output.FormatLocation(e.City, e.StateProv, e.Country)}
			}
			return outputTable(cmd, events, []string{"Key", "Name", "Start Date", "Location"}, rows)
		},
	}
	c.Flags().Int("year", currentYear(), "Season year (default: current year)")
	return c
}

func newTeamMatchesCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "matches <number>",
		Short: "List team matches for a year",
		Example: `  tba team matches 177 --year 2024
  tba team matches frc177 --year 2024 --format tsv`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := newClient(cmd)
			if err != nil {
				return err
			}
			year, _ := cmd.Flags().GetInt("year")
			var matches []api.Match
			if err := client.Get(fmt.Sprintf("/team/%s/matches/%d", teamKey(args[0]), year), &matches); err != nil {
				return err
			}
			rows := make([][]string, len(matches))
			for i, m := range matches {
				rows[i] = []string{m.Key, m.CompLevel, m.WinningAlliance}
			}
			return outputTable(cmd, matches, []string{"Key", "Level", "Winner"}, rows)
		},
	}
	c.Flags().Int("year", currentYear(), "Season year (default: current year)")
	return c
}

func newTeamAwardsCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "awards <number>",
		Short: "List team awards",
		Example: `  tba team awards 177
  tba team awards frc177 --year 2024 --format markdown`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := newClient(cmd)
			if err != nil {
				return err
			}
			year, _ := cmd.Flags().GetInt("year")
			path := fmt.Sprintf("/team/%s/awards", teamKey(args[0]))
			if year > 0 {
				path = fmt.Sprintf("/team/%s/awards/%d", teamKey(args[0]), year)
			}
			var awards []api.Award
			if err := client.Get(path, &awards); err != nil {
				return err
			}
			rows := make([][]string, len(awards))
			for i, a := range awards {
				rows[i] = []string{a.EventKey, a.Name, strconv.Itoa(a.Year)}
			}
			return outputTable(cmd, awards, []string{"Event", "Award", "Year"}, rows)
		},
	}
	c.Flags().Int("year", 0, "Season year (default: all years)")
	return c
}

func newTeamMediaCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "media <number>",
		Short: "List team media",
		Example: `  tba team media 177 --year 2024
  tba team media frc177 --year 2024 --jq '.[].view_url' -r`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := newClient(cmd)
			if err != nil {
				return err
			}
			year, _ := cmd.Flags().GetInt("year")
			var media []api.Media
			if err := client.Get(fmt.Sprintf("/team/%s/media/%d", teamKey(args[0]), year), &media); err != nil {
				return err
			}
			rows := make([][]string, len(media))
			for i, m := range media {
				rows[i] = []string{m.Type, m.ForeignKey, m.ViewURL}
			}
			return outputTable(cmd, media, []string{"Type", "Key", "URL"}, rows)
		},
	}
	c.Flags().Int("year", currentYear(), "Season year (default: current year)")
	return c
}

func newTeamRobotsCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "robots <number>",
		Short: "List team robots",
		Example: `  tba team robots 177
  tba team robots frc177 --format csv`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := newClient(cmd)
			if err != nil {
				return err
			}
			var robots []api.Robot
			if err := client.Get(fmt.Sprintf("/team/%s/robots", teamKey(args[0])), &robots); err != nil {
				return err
			}
			rows := make([][]string, len(robots))
			for i, r := range robots {
				rows[i] = []string{strconv.Itoa(r.Year), r.RobotName}
			}
			return outputTable(cmd, robots, []string{"Year", "Robot Name"}, rows)
		},
	}
}

func newTeamDistrictsCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "districts <number>",
		Short: "List team districts",
		Example: `  tba team districts 177
  tba team districts frc177 --format json`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := newClient(cmd)
			if err != nil {
				return err
			}
			var districts []api.District
			if err := client.Get(fmt.Sprintf("/team/%s/districts", teamKey(args[0])), &districts); err != nil {
				return err
			}
			rows := make([][]string, len(districts))
			for i, d := range districts {
				rows[i] = []string{d.Key, d.DisplayName, strconv.Itoa(d.Year)}
			}
			return outputTable(cmd, districts, []string{"Key", "Name", "Year"}, rows)
		},
	}
}
