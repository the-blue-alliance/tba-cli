package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newInsightCmd() *cobra.Command {
	insightCmd := &cobra.Command{
		Use:     "insight",
		Aliases: []string{"insights"},
		Short:   "View insights and leaderboards",
	}
	insightCmd.AddCommand(newInsightLeaderboardsCmd())
	insightCmd.AddCommand(newInsightNotablesCmd())
	return insightCmd
}

func newInsightLeaderboardsCmd() *cobra.Command {
	return newRawInsightCmd("leaderboards", "Show insight leaderboards for a year", "leaderboards",
		`  tba insight leaderboards --year 2024
  tba insights leaderboards --year 2024 --jq '.[0].name' -r`)
}

func newInsightNotablesCmd() *cobra.Command {
	return newRawInsightCmd("notables", "Show notable insights for a year", "notables",
		`  tba insight notables --year 2024
  tba insights notables --year 2024 --format json`)
}

// newRawInsightCmd builds a command that passes an insights sub-resource
// through untouched, since these endpoints have no stable schema.
func newRawInsightCmd(use, short, resource, example string) *cobra.Command {
	c := &cobra.Command{
		Use:     use,
		Short:   short,
		Example: example,
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := newClient(cmd)
			if err != nil {
				return err
			}
			year, _ := cmd.Flags().GetInt("year")
			raw, err := client.GetRaw(fmt.Sprintf("/insights/%s/%d", resource, year))
			if err != nil {
				return err
			}
			return outputData(cmd, raw, func() {
				fmt.Fprintln(cmd.OutOrStdout(), string(raw))
			})
		},
	}
	c.Flags().Int("year", currentYear(), "Season year (default: current year)")
	return c
}
