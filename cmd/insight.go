package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/the-blue-alliance/tba-cli/internal/output"
)

func newInsightCmd() *cobra.Command {
	insightCmd := &cobra.Command{
		Use:   "insight",
		Short: "View insights and leaderboards",
	}
	insightCmd.AddCommand(newInsightLeaderboardsCmd())
	insightCmd.AddCommand(newInsightNotablesCmd())
	return insightCmd
}

func newInsightLeaderboardsCmd() *cobra.Command {
	return newRawInsightCmd("leaderboards", "Show insight leaderboards for a year", "leaderboards")
}

func newInsightNotablesCmd() *cobra.Command {
	return newRawInsightCmd("notables", "Show notable insights for a year", "notables")
}

// newRawInsightCmd builds a command that passes an insights sub-resource
// through untouched, since these endpoints have no stable schema.
func newRawInsightCmd(use, short, resource string) *cobra.Command {
	c := &cobra.Command{
		Use:   use,
		Short: short,
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
			if wantJSON(cmd) {
				return output.PrintJSONWithFilter(cmd.OutOrStdout(), raw, jqExpr(cmd))
			}
			fmt.Fprintln(cmd.OutOrStdout(), string(raw))
			return nil
		},
	}
	c.Flags().Int("year", currentYear(), "Season year (default: current year)")
	return c
}
