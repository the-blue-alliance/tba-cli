package cmd

import (
	"github.com/spf13/cobra"
	"github.com/the-blue-alliance/tba-cli/internal/api"
)

func newSponsorsCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "sponsors",
		Short: "List TBA's platinum sponsors",
		Example: `  tba sponsors
  tba sponsors --json
  tba sponsors --jq '.[].name'`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := newClient(cmd)
			if err != nil {
				return err
			}
			var list api.SponsorsList
			if err := client.Get(cmd.Context(), "/sponsors", &list); err != nil {
				return err
			}
			sponsors := list.Sponsors
			if sponsors == nil {
				sponsors = []api.Sponsor{}
			}
			rows := make([][]string, len(sponsors))
			for i, s := range sponsors {
				rows[i] = []string{s.Name, s.URL, s.LogoStyle, deref(s.LogoURL), deref(s.LogoURLDark)}
			}
			return outputTableWithEmptyNote(cmd, sponsors,
				[]string{"Name", "URL", "Logo Style", "Logo", "Dark Logo"}, rows,
				"no platinum sponsors")
		},
	}
}

// deref renders an optional string, with null as an empty cell.
func deref(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
