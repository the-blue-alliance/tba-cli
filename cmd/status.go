package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/the-blue-alliance/tba-cli/internal/api"
	"github.com/the-blue-alliance/tba-cli/internal/output"
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show TBA API status",
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := api.NewClient()
		if err != nil {
			return err
		}

		var status api.APIStatus
		if err := client.Get("/status", &status); err != nil {
			return err
		}

		jsonFlag, _ := cmd.Flags().GetBool("json")
		jqFlag, _ := cmd.Flags().GetString("jq")

		if jsonFlag || jqFlag != "" || !output.IsTTY() {
			return output.PrintJSONWithFilter(status, jqFlag)
		}

		output.PrintKeyValue(
			"Current Season", fmt.Sprintf("%d", status.CurrentSeason),
			"Max Season", fmt.Sprintf("%d", status.MaxSeason),
			"Datafeed Down", fmt.Sprintf("%v", status.IsDatafeedDown),
		)
		return nil
	},
}
