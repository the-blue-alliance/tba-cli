package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/the-blue-alliance/tba-cli/internal/api"
	"github.com/the-blue-alliance/tba-cli/internal/output"
)

var matchCmd = &cobra.Command{
	Use:   "match",
	Short: "Work with matches",
}

var matchViewCmd = &cobra.Command{
	Use:   "view <key>",
	Short: "View match info",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := api.NewClient()
		if err != nil {
			return err
		}
		var match api.Match
		if err := client.Get(fmt.Sprintf("/match/%s", args[0]), &match); err != nil {
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
			output.PrintKeyValue(
				"Match", match.Key,
				"Level", match.CompLevel,
				"Red Alliance", fmt.Sprintf("%s (%d)", strings.Join(redTeams, ", "), red.Score),
				"Blue Alliance", fmt.Sprintf("%s (%d)", strings.Join(blueTeams, ", "), blue.Score),
				"Winner", match.WinningAlliance,
			)
		})
	},
}

func init() {
	matchCmd.AddCommand(matchViewCmd)
}
