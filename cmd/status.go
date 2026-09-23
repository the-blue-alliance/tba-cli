package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/the-blue-alliance/tba-cli/internal/api"
	"github.com/the-blue-alliance/tba-cli/internal/output"
)

func newStatusCmd() *cobra.Command {
	return &cobra.Command{
		Use: "status",
		// "status" is a word people arrive at looking for their team's
		// standing at an event, so the one-liner points them onwards.
		Short: "Show TBA API status (for a team's standing see 'tba team standing')",
		Example: `  tba status
  tba status --format json
  tba status --jq .current_season`,
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := newClient(cmd)
			if err != nil {
				return err
			}

			var status api.APIStatus
			if err := client.Get(cmd.Context(), "/status", &status); err != nil {
				return err
			}

			return outputData(cmd, status, func() {
				output.PrintKeyValue(cmd.OutOrStdout(),
					"Current Season", fmt.Sprintf("%d", status.CurrentSeason),
					"Max Season", fmt.Sprintf("%d", status.MaxSeason),
					"Datafeed Down", fmt.Sprintf("%v", status.IsDatafeedDown),
				)
			})
		},
	}
}
