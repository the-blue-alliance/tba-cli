package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/the-blue-alliance/tba-cli/internal/api"
	"github.com/the-blue-alliance/tba-cli/internal/output"
)

func newMatchCmd() *cobra.Command {
	matchCmd := &cobra.Command{
		Use:     "match",
		Aliases: []string{"matches"},
		Short:   "Work with matches",
	}
	matchCmd.AddCommand(newMatchViewCmd())
	return matchCmd
}

func newMatchViewCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "view <key>",
		Short: "View match info",
		Example: `  tba match view 2024cthar_qm12
  tba match view 2024cthar_sf3m1 --format json`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := validateMatchKey(args[0]); err != nil {
				return err
			}
			client, err := newClient(cmd)
			if err != nil {
				return err
			}
			var match api.Match
			if err := client.Get(cmd.Context(), fmt.Sprintf("/match/%s", args[0]), &match); err != nil {
				return err
			}
			return outputData(cmd, match, func() {
				red := match.Alliances["red"]
				blue := match.Alliances["blue"]
				redTeams := make([]string, len(red.TeamKeys))
				for i, k := range red.TeamKeys {
					redTeams[i] = output.TeamNumberFromKey(k)
				}
				blueTeams := make([]string, len(blue.TeamKeys))
				for i, k := range blue.TeamKeys {
					blueTeams[i] = output.TeamNumberFromKey(k)
				}
				output.PrintKeyValue(cmd.OutOrStdout(),
					"Match", match.Key,
					"Level", match.CompLevel,
					"Red Alliance", fmt.Sprintf("%s (%d)", strings.Join(redTeams, ", "), red.Score),
					"Blue Alliance", fmt.Sprintf("%s (%d)", strings.Join(blueTeams, ", "), blue.Score),
					"Winner", match.WinningAlliance,
				)
			})
		},
	}
}
