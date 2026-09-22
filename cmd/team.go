package cmd

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

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
	teamCmd.AddCommand(newTeamYearsCmd())
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
			if err := client.Get(cmd.Context(), fmt.Sprintf("/team/%s", teamKey(args[0])), &team); err != nil {
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
			maxPages, _ := cmd.Flags().GetInt("max-pages")
			var allTeams []api.Team
			var cappedAt int
			for page := 0; ; page++ {
				if maxPages > 0 && page >= maxPages {
					cappedAt = maxPages
					break
				}
				var teams []api.Team
				path := fmt.Sprintf("/teams/%d/%d", year, page)
				if err := client.Get(cmd.Context(), path, &teams); err != nil {
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
			if err := outputTable(cmd, allTeams, []string{"Number", "Name", "Location"}, rows); err != nil {
				return err
			}
			if cappedAt > 0 {
				fmt.Fprintf(cmd.ErrOrStderr(), "note: stopped after %d pages; raise --max-pages to fetch more\n", cappedAt)
			}
			return nil
		},
	}
	c.Flags().Int("year", currentYear(), "Season year (default: current year)")
	c.Flags().Int("max-pages", 30, "Stop after this many pages of 500 teams")
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
			if err := client.Get(cmd.Context(), fmt.Sprintf("/team/%s/events/%d", teamKey(args[0]), year), &events); err != nil {
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

func newTeamYearsCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "years <number>",
		Short: "List the seasons a team has competed in",
		Long: `List the seasons a team has competed in, most recent first.

The JSON form is the plain array of years the API returns, which makes it easy
to drive a loop over a team's whole history.`,
		Example: `  tba team years 177
  tba team years frc177 --json
  tba team years 177 --jq '.[0]' -r`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := newClient(cmd)
			if err != nil {
				return err
			}
			var years []int
			path := fmt.Sprintf("/team/%s/years_participated", teamKey(args[0]))
			if err := client.Get(cmd.Context(), path, &years); err != nil {
				return err
			}
			// Newest first: the recent seasons are the ones people look up.
			sort.Sort(sort.Reverse(sort.IntSlice(years)))
			rows := make([][]string, len(years))
			for i, y := range years {
				rows[i] = []string{strconv.Itoa(y)}
			}
			return outputTable(cmd, years, []string{"Year"}, rows)
		},
	}
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
			if err := client.Get(cmd.Context(), fmt.Sprintf("/team/%s/matches/%d", teamKey(args[0]), year), &matches); err != nil {
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
		Long: `List the awards a team has won, most recent season first.

Without --year this is the team's whole award history. The event column shows
the event's name, which takes one extra request for the team's event list; if
that request fails the awards are still listed, with the name left blank.

Recipient names the individual who received the award, for awards such as
Dean's List or Woodie Flowers that go to a person rather than to the team.`,
		Example: `  tba team awards 177
  tba team awards frc177 --year 2024 --format markdown
  tba team awards 177 --type 0`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := newClient(cmd)
			if err != nil {
				return err
			}
			key := teamKey(args[0])
			year, _ := cmd.Flags().GetInt("year")
			path := fmt.Sprintf("/team/%s/awards", key)
			if year > 0 {
				path = fmt.Sprintf("/team/%s/awards/%d", key, year)
			}
			var awards []api.Award
			if err := client.Get(cmd.Context(), path, &awards); err != nil {
				return err
			}
			if cmd.Flags().Changed("type") {
				awardType, _ := cmd.Flags().GetInt("type")
				awards = filterAwardsByType(awards, awardType)
			}
			names := teamEventNames(cmd, client, key, len(awards) > 0)
			sortTeamAwards(awards)

			rows := make([][]string, len(awards))
			for i, a := range awards {
				rows[i] = []string{
					strconv.Itoa(a.Year),
					names[a.EventKey],
					a.Name,
					awardeeNames(a),
				}
			}
			return outputTable(cmd, awards, []string{"Year", "Event", "Award", "Recipient"}, rows)
		},
	}
	c.Flags().Int("year", 0, "Season year (default: all years)")
	c.Flags().Int("type", -1, "Only awards with this TBA award_type")
	return c
}

// filterAwardsByType keeps only the awards with the given award_type.
func filterAwardsByType(awards []api.Award, awardType int) []api.Award {
	out := make([]api.Award, 0, len(awards))
	for _, a := range awards {
		if a.AwardType == awardType {
			out = append(out, a)
		}
	}
	return out
}

// sortTeamAwards puts the most recent season first, with the event key
// grouping a season's awards together and keeping the order reproducible.
func sortTeamAwards(awards []api.Award) {
	sort.SliceStable(awards, func(i, j int) bool {
		if awards[i].Year != awards[j].Year {
			return awards[i].Year > awards[j].Year
		}
		return awards[i].EventKey < awards[j].EventKey
	})
}

// awardeeNames lists the people an award went to. Most awards go to the team
// itself and have no awardee, which leaves the column empty.
func awardeeNames(a api.Award) string {
	var names []string
	for _, r := range a.Recipients {
		if r.Awardee != nil && *r.Awardee != "" {
			names = append(names, *r.Awardee)
		}
	}
	return strings.Join(names, ", ")
}

// teamEventNames maps event key to event name from the team's full event
// list, in one request covering every season.
//
// It is decoration, not data: a team's awards are worth printing even when
// the event list cannot be fetched, so a failure here yields an empty map
// rather than an error.
func teamEventNames(cmd *cobra.Command, client *api.Client, key string, wanted bool) map[string]string {
	names := map[string]string{}
	if !wanted {
		return names
	}
	var events []api.Event
	if err := client.Get(cmd.Context(), fmt.Sprintf("/team/%s/events", key), &events); err != nil {
		return names
	}
	for _, e := range events {
		names[e.Key] = e.Name
	}
	return names
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
			if err := client.Get(cmd.Context(), fmt.Sprintf("/team/%s/media/%d", teamKey(args[0]), year), &media); err != nil {
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
			if err := client.Get(cmd.Context(), fmt.Sprintf("/team/%s/robots", teamKey(args[0])), &robots); err != nil {
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
			if err := client.Get(cmd.Context(), fmt.Sprintf("/team/%s/districts", teamKey(args[0])), &districts); err != nil {
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
