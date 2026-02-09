package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/the-blue-alliance/tba-cli/internal/output"
)

var insightCmd = &cobra.Command{
	Use:   "insight",
	Short: "View insights and leaderboards",
}

var insightLeaderboardsCmd = &cobra.Command{
	Use:   "leaderboards",
	Short: "Show insight leaderboards for a year",
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := newClient(cmd)
		if err != nil {
			return err
		}
		year, _ := cmd.Flags().GetInt("year")
		raw, err := client.GetRaw(fmt.Sprintf("/insights/leaderboards/%d", year))
		if err != nil {
			return err
		}
		if wantJSON(cmd) {
			return output.PrintJSONWithFilter(raw, jqExpr(cmd))
		}
		fmt.Println(string(raw))
		return nil
	},
}

var insightNotablesCmd = &cobra.Command{
	Use:   "notables",
	Short: "Show notable insights for a year",
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := newClient(cmd)
		if err != nil {
			return err
		}
		year, _ := cmd.Flags().GetInt("year")
		raw, err := client.GetRaw(fmt.Sprintf("/insights/notables/%d", year))
		if err != nil {
			return err
		}
		if wantJSON(cmd) {
			return output.PrintJSONWithFilter(raw, jqExpr(cmd))
		}
		fmt.Println(string(raw))
		return nil
	},
}

func init() {
	insightLeaderboardsCmd.Flags().Int("year", currentYear(), "Year")
	insightLeaderboardsCmd.MarkFlagRequired("year")
	insightNotablesCmd.Flags().Int("year", currentYear(), "Year")
	insightNotablesCmd.MarkFlagRequired("year")

	insightCmd.AddCommand(insightLeaderboardsCmd)
	insightCmd.AddCommand(insightNotablesCmd)
}
