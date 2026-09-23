package cmd

import (
	"cmp"
	"fmt"
	"slices"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
	"github.com/the-blue-alliance/tba-cli/internal/api"
	"github.com/the-blue-alliance/tba-cli/internal/frc"
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
	teamCmd.AddCommand(newTeamSearchCmd())
	teamCmd.AddCommand(newTeamEventsCmd())
	teamCmd.AddCommand(newTeamYearsCmd())
	teamCmd.AddCommand(newTeamMatchesCmd())
	teamCmd.AddCommand(newTeamNextCmd())
	teamCmd.AddCommand(newTeamStandingCmd())
	teamCmd.AddCommand(newTeamAwardsCmd())
	teamCmd.AddCommand(newTeamMediaCmd())
	teamCmd.AddCommand(newTeamRobotsCmd())
	teamCmd.AddCommand(newTeamDistrictsCmd())
	return teamCmd
}

func newTeamViewCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "view <number>",
		Short: "Show a team's details",
		Example: `  tba team view 177
  tba team view frc177 --format json
  tba team view 177 --jq .nickname -r`,
		Args: exactArgs(1, "a team number (e.g. tba team view 177)"),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := validateTeamArg(args[0]); err != nil {
				return err
			}
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
		Short: "List a season's teams",
		Example: `  tba team list --year 2024
  tba teams list --year 2024 --format csv`,
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := newClient(cmd)
			if err != nil {
				return err
			}
			year, err := resolveYear(cmd, client)
			if err != nil {
				return err
			}
			maxPages, _ := cmd.Flags().GetInt("max-pages")
			allTeams, cappedAt, err := fetchTeamPages(cmd, client, year, maxPages)
			if err != nil {
				return err
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
	addYearFlag(c)
	c.Flags().Int("max-pages", defaultMaxPages, "Stop after this many pages of 500 teams; the walk ends at the first empty page anyway")
	return c
}

func newTeamEventsCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "events <number>",
		Short: "List a team's events",
		Example: `  tba team events 177 --year 2024
  tba team events frc177 --year 2024 --format csv`,
		Args: exactArgs(1, "a team number (e.g. tba team events 177)"),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := validateTeamArg(args[0]); err != nil {
				return err
			}
			client, err := newClient(cmd)
			if err != nil {
				return err
			}
			year, err := resolveYear(cmd, client)
			if err != nil {
				return err
			}
			var events []api.Event
			if err := client.Get(cmd.Context(), fmt.Sprintf("/team/%s/events/%d", teamKey(args[0]), year), &events); err != nil {
				return err
			}
			// A season is read as a season: the API lists a team's events by
			// key, which puts April's district championship ahead of March's
			// district events.
			frc.SortEvents(events)
			rows := make([][]string, len(events))
			for i, e := range events {
				rows[i] = []string{e.Key, e.Name, e.StartDate, output.FormatLocation(e.City, e.StateProv, e.Country)}
			}
			return outputTableWithEmptyNote(cmd, events,
				[]string{"Key", "Name", "Start Date", "Location"}, rows,
				fmt.Sprintf("no events for team %s in %d", teamNumberOf(args[0]), year))
		},
	}
	addYearFlag(c)
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
		Args: exactArgs(1, "a team number (e.g. tba team years 177)"),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := validateTeamArg(args[0]); err != nil {
				return err
			}
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
			slices.SortFunc(years, func(a, b int) int { return cmp.Compare(b, a) })
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
		Short: "List a team's matches",
		Long: `List the matches a team played, for one event or for a whole season.

A season spans several events, and every one of them has a Qual 12, so the
listing is grouped by event, in the order the team competed, with an Event
column naming each. One event's listing drops that column and is simply the
match table.`,
		Example: `  tba team matches 177 --year 2024
  tba team matches frc177 --year 2024 --format tsv
  tba team matches 177 --event 2024cthar
  tba team matches 177 --event 2024cthar --upcoming`,
		Args: exactArgs(1, "a team number (e.g. tba team matches 177)"),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := validateTeamArg(args[0]); err != nil {
				return err
			}
			client, err := newClient(cmd)
			if err != nil {
				return err
			}

			team := teamKey(args[0])
			eventKey, _ := cmd.Flags().GetString("event")
			var matches []api.Match
			if eventKey != "" {
				if err := validateEventKey(eventKey); err != nil {
					return err
				}
				path := fmt.Sprintf("/team/%s/event/%s/matches", team, eventKey)
				if err := client.Get(cmd.Context(), path, &matches); err != nil {
					return err
				}
				return renderMatches(cmd, matches, constantPlayoffType(eventPlayoffType(cmd, client, eventKey)), eventKey, nowOf(cmd))
			}

			year, err := resolveYear(cmd, client)
			if err != nil {
				return err
			}
			path := fmt.Sprintf("/team/%s/matches/%d", team, year)
			if err := client.Get(cmd.Context(), path, &matches); err != nil {
				return err
			}
			// The listing spans a whole season, whose events may have run
			// different playoff brackets; the labels then fall back to a guess
			// from each match's own season.
			scope := fmt.Sprintf("team %s in %d", output.TeamNumberFromKey(team), year)
			return renderSeasonMatches(cmd, matches, constantPlayoffType(nil), teamEventOrder(cmd, client, team, year), scope, nowOf(cmd))
		},
	}
	addYearFlag(c)
	c.Flags().String("event", "", "Restrict to one event, by key (e.g. 2024cthar); overrides --year")
	// No --team here: the argument already names the team.
	addMatchFilterFlags(c)
	return c
}

func newTeamAwardsCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "awards <number>",
		Short: "List a team's awards",
		Long: `List the awards a team has won, most recent season first.

Without --year this is the team's whole award history. The event column shows
the event's name, which takes one extra request for the team's event list; if
that request fails the awards are still listed, with the name left blank.

Recipient names the individual who received the award, for awards such as
Dean's List or Woodie Flowers that go to a person rather than to the team.

--type narrows the list to one kind of award. It takes a name, matched
case-insensitively against any part of it (--type impact, --type "dean's"), or
TBA's own award_type code (--type 0). A name that could mean several awards is
an error that lists them.`,
		Example: `  tba team awards 177
  tba team awards frc177 --year 2024 --format markdown
  tba team awards 177 --type impact
  tba team awards 177 --type 0`,
		Args: exactArgs(1, "a team number (e.g. tba team awards 177)"),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := validateTeamArg(args[0]); err != nil {
				return err
			}
			// The award type is resolved before the request, so a name that
			// means nothing costs nothing.
			typeSpec, _ := cmd.Flags().GetString("type")
			filterByType := cmd.Flags().Changed("type")
			wantType := 0
			if filterByType {
				code, err := parseAwardType(typeSpec)
				if err != nil {
					return err
				}
				wantType = code
			}
			client, err := newClient(cmd)
			if err != nil {
				return err
			}
			key := teamKey(args[0])
			// Awards are the one --year that does not mean "this season":
			// its default is every year a team has won anything, so it is
			// read from the flag alone rather than resolved from a season.
			year, _ := cmd.Flags().GetInt("year")
			path := fmt.Sprintf("/team/%s/awards", key)
			if year > 0 {
				path = fmt.Sprintf("/team/%s/awards/%d", key, year)
			}
			var awards []api.Award
			if err := client.Get(cmd.Context(), path, &awards); err != nil {
				return err
			}
			if filterByType {
				awards = filterAwardsByType(awards, wantType)
			}
			names := teamEventNames(cmd, client, key, len(awards) > 0)
			sortTeamAwards(awards)

			rows := make([][]string, len(awards))
			for i, a := range awards {
				rows[i] = []string{
					strconv.Itoa(a.Year),
					names[a.EventKey],
					a.Name,
					awardTypeName(a.AwardType),
					awardeeNames(a),
				}
			}
			note := fmt.Sprintf("no awards for team %s", teamNumberOf(args[0]))
			if filterByType {
				note = fmt.Sprintf("no %s awards for team %s", awardTypeName(wantType), teamNumberOf(args[0]))
			}
			return outputTableWithEmptyNote(cmd, awards,
				[]string{"Year", "Event", "Award", "Type", "Recipient"}, rows, note)
		},
	}
	c.Flags().Int("year", 0, "Season year (default: all years)")
	c.Flags().String("type", "", "Only awards of this kind, by name or TBA award_type code (e.g. impact, 0)")
	return c
}

// teamEventOrder ranks the events a team attended in a season, so that a
// season's matches can be grouped the way the team played them.
//
// It is ordering, not data: a season listing is worth printing even when the
// event list cannot be fetched, so a failure yields a nil order and the
// listing falls back to grouping by event key.
func teamEventOrder(cmd *cobra.Command, client *api.Client, team string, year int) map[string]int {
	var events []api.Event
	if err := client.Get(cmd.Context(), fmt.Sprintf("/team/%s/events/%d", team, year), &events); err != nil {
		return nil
	}
	return frc.EventOrder(events)
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
	slices.SortStableFunc(awards, func(a, b api.Award) int {
		if c := cmp.Compare(b.Year, a.Year); c != 0 {
			return c
		}
		return strings.Compare(a.EventKey, b.EventKey)
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
		Short: "List a team's media",
		Example: `  tba team media 177 --year 2024
  tba team media frc177 --year 2024 --jq '.[].view_url' -r`,
		Args: exactArgs(1, "a team number (e.g. tba team media 177)"),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := validateTeamArg(args[0]); err != nil {
				return err
			}
			client, err := newClient(cmd)
			if err != nil {
				return err
			}
			year, err := resolveYear(cmd, client)
			if err != nil {
				return err
			}
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
	addYearFlag(c)
	return c
}

func newTeamRobotsCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "robots <number>",
		Short: "List a team's robots",
		Example: `  tba team robots 177
  tba team robots frc177 --format csv`,
		Args: exactArgs(1, "a team number (e.g. tba team robots 177)"),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := validateTeamArg(args[0]); err != nil {
				return err
			}
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
		Short: "List a team's districts",
		Example: `  tba team districts 177
  tba team districts frc177 --format json`,
		Args: exactArgs(1, "a team number (e.g. tba team districts 177)"),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := validateTeamArg(args[0]); err != nil {
				return err
			}
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
