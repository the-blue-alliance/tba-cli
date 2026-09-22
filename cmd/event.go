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

func newEventCmd() *cobra.Command {
	eventCmd := &cobra.Command{
		Use:     "event",
		Aliases: []string{"events"},
		Short:   "Work with events",
	}
	eventCmd.AddCommand(newEventViewCmd())
	eventCmd.AddCommand(newEventListCmd())
	eventCmd.AddCommand(newEventTeamsCmd())
	eventCmd.AddCommand(newEventMatchesCmd())
	eventCmd.AddCommand(newEventRankingsCmd())
	eventCmd.AddCommand(newEventWatchCmd())
	eventCmd.AddCommand(newEventAlliancesCmd())
	eventCmd.AddCommand(newEventTeamStatusesCmd())
	eventCmd.AddCommand(newEventAwardsCmd())
	eventCmd.AddCommand(newEventOPRsCmd())
	eventCmd.AddCommand(newEventDistrictPointsCmd())
	eventCmd.AddCommand(newEventPredictionsCmd())
	eventCmd.AddCommand(newEventInsightsCmd())
	eventCmd.AddCommand(newEventExportCmd())
	return eventCmd
}

func newEventViewCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "view <key>",
		Short: "Show an event's details",
		Example: `  tba event view 2024cthar
  tba event view 2024cthar --format json`,
		Args: exactArgs(1, "an event key (e.g. tba event view 2024cthar)"),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := validateEventKey(args[0]); err != nil {
				return err
			}
			client, err := newClient(cmd)
			if err != nil {
				return err
			}
			var event api.Event
			if err := client.Get(cmd.Context(), fmt.Sprintf("/event/%s", args[0]), &event); err != nil {
				return err
			}
			return outputData(cmd, event, func() {
				output.PrintKeyValue(cmd.OutOrStdout(), eventDetailPairs(event)...)
			})
		},
	}
}

// eventDetailPairs describes an event as alternating key/value strings, in the
// order `event view` prints them. `event export` reuses it so an exported
// event carries exactly the fields the command shows.
func eventDetailPairs(event api.Event) []string {
	week := "N/A"
	if event.Week != nil {
		week = humanWeek(event.Week)
	}
	pairs := []string{
		"Event", event.Name,
		"Key", event.Key,
		"Type", event.EventTypeStr,
	}
	pairs = appendPair(pairs, "District", districtName(event.District))
	pairs = append(pairs,
		"Location", output.FormatLocation(event.City, event.StateProv, event.Country),
		"Venue", event.LocationName,
		"Dates", fmt.Sprintf("%s to %s", event.StartDate, event.EndDate),
		"Week", week,
	)
	pairs = appendPair(pairs, "Playoff", playoffTypeName(event.PlayoffType))
	pairs = appendPair(pairs, "Timezone", event.Timezone)
	pairs = appendPair(pairs, "Website", event.Website)
	if event.FirstEventCode != nil {
		pairs = appendPair(pairs, "First Code", *event.FirstEventCode)
	}
	// One line per webcast: an event can stream several fields at once, and
	// each needs its own link.
	for _, w := range event.Webcasts {
		pairs = appendPair(pairs, "Webcast", webcastURL(w))
	}
	return pairs
}

func newEventListCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "list",
		Short: "List a season's events",
		Long: `List the events of a season.

Every filter is applied to the season's event list after it is fetched, so any
combination of them works and only one request is made. --week takes the
1-based week number shown on The Blue Alliance, not the 0-based one the API
returns.`,
		Example: `  tba event list --year 2024
  tba event list --year 2024 --week 3 --type district
  tba event list --year 2024 --district ne --state CT
  tba event list --year 2024 --team 177
  tba events list --year 2024 --format csv`,
		RunE: func(cmd *cobra.Command, args []string) error {
			filter, err := eventFilterFromFlags(cmd)
			if err != nil {
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
			path := fmt.Sprintf("/events/%d", year)
			if team, _ := cmd.Flags().GetString("team"); strings.TrimSpace(team) != "" {
				path = fmt.Sprintf("/team/%s/events/%d", teamKey(team), year)
			}
			var events []api.Event
			if err := client.Get(cmd.Context(), path, &events); err != nil {
				return err
			}
			events = filter.apply(events)
			sortEvents(events)
			rows := make([][]string, len(events))
			for i, e := range events {
				rows[i] = []string{
					e.Key,
					e.Name,
					e.EventTypeStr,
					humanWeek(e.Week),
					e.StartDate,
					e.EndDate,
					output.FormatLocation(e.City, e.StateProv, e.Country),
					districtAbbrev(e.District),
				}
			}
			headers := []string{"Key", "Name", "Type", "Week", "Start", "End", "Location", "District"}
			note := fmt.Sprintf("no events in %d", year)
			if filterFlagsGiven(cmd) {
				note = "no events match those filters"
			}
			return outputTableWithEmptyNote(cmd, events, headers, rows, note)
		},
	}
	addYearFlag(c)
	c.Flags().Int("week", 0, "Only events in this competition week (1-based, as thebluealliance.com numbers them)")
	c.Flags().String("type", "", "Only events of these types, comma-separated: "+validEventTypes)
	c.Flags().String("district", "", "Only events in this district, by abbreviation (e.g. ne)")
	c.Flags().String("state", "", "Only events in this state or province (e.g. CT)")
	c.Flags().String("country", "", "Only events in this country (e.g. USA)")
	c.Flags().String("team", "", "Only events this team attends (e.g. 177 or frc177)")
	return c
}

func newEventTeamsCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "teams <key>",
		Short: "List the teams at an event",
		Example: `  tba event teams 2024cthar
  tba event teams 2024cthar --format csv`,
		Args: exactArgs(1, "an event key (e.g. tba event teams 2024cthar)"),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := validateEventKey(args[0]); err != nil {
				return err
			}
			client, err := newClient(cmd)
			if err != nil {
				return err
			}
			var teams []api.Team
			if err := client.Get(cmd.Context(), fmt.Sprintf("/event/%s/teams", args[0]), &teams); err != nil {
				return err
			}
			table := eventTeamsTable(teams)
			return outputTableWithEmptyNote(cmd, teams, table.Headers, table.Rows,
				fmt.Sprintf("no teams listed for %s yet", args[0]))
		},
	}
}

// eventTeamsTable renders the teams attending an event, in the order given.
func eventTeamsTable(teams []api.Team) output.Table {
	rows := make([][]string, len(teams))
	for i, t := range teams {
		rows[i] = []string{fmt.Sprintf("%d", t.TeamNumber), t.Nickname, output.FormatLocation(t.City, t.StateProv, t.Country)}
	}
	return output.Table{Headers: []string{"Number", "Name", "Location"}, Rows: rows}
}

func newEventMatchesCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "matches <key>",
		Short: "List an event's matches",
		Example: `  tba event matches 2024cthar
  tba event matches 2024cthar --team 177 --upcoming
  tba event matches 2024cthar --level playoff
  tba event matches 2024cthar --format csv
  tba event matches 2024cthar --jq '.[].key' -r`,
		Args: exactArgs(1, "an event key (e.g. tba event matches 2024cthar)"),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := validateEventKey(args[0]); err != nil {
				return err
			}
			client, err := newClient(cmd)
			if err != nil {
				return err
			}
			var matches []api.Match
			if err := client.Get(cmd.Context(), fmt.Sprintf("/event/%s/matches", args[0]), &matches); err != nil {
				return err
			}
			// The event is fetched second and only for its playoff_type, which
			// decides how playoff matches are named.
			playoffType := eventPlayoffType(cmd, client, args[0])
			return renderMatches(cmd, matches, constantPlayoffType(playoffType), args[0])
		},
	}
	addMatchTableFlags(c)
	return c
}

func newEventRankingsCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "rankings <key>",
		Short: "Show qualification rankings",
		Long: `Show the qualification rankings for an event.

After Rank, Team, Name, Record, Played and DQ the table carries one column per
ranking sort order the season defines -- Ranking Score, Avg Match and so on --
each printed at the precision the API declares, then any extra statistics such
as Total Ranking Points. The columns therefore differ from season to season.`,
		Example: `  tba event rankings 2024cthar
  tba event rankings 2024cthar --format markdown
  tba event rankings 2024cthar --columns rank,team,name,"ranking score"
  tba event rankings 2024cthar --sort=-played`,
		Args: exactArgs(1, "an event key (e.g. tba event rankings 2024cthar)"),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := validateEventKey(args[0]); err != nil {
				return err
			}
			// Resolved up front so a bad --format fails before any request,
			// and so JSON output can skip the nickname lookup it never uses.
			format, err := resolveFormat(cmd)
			if err != nil {
				return err
			}
			client, err := newClient(cmd)
			if err != nil {
				return err
			}
			var rankings api.EventRankings
			if err := client.Get(cmd.Context(), fmt.Sprintf("/event/%s/rankings", args[0]), &rankings); err != nil {
				return err
			}

			// Rankings carry team keys but no names, so the nicknames come
			// from a second call. They are a convenience, not the point of the
			// command, so a failure there leaves the column blank rather than
			// failing a ranking table the user already paid for.
			nicknames := map[string]string{}
			if format != "json" {
				var teams []api.Team
				if err := client.Get(cmd.Context(), fmt.Sprintf("/event/%s/teams/simple", args[0]), &teams); err == nil {
					for _, t := range teams {
						nicknames[t.Key] = t.Nickname
					}
				}
			}

			table := eventRankingsTable(&rankings, nicknames)
			return outputTable(cmd, rankings, table.Headers, table.Rows)
		},
	}
}

// eventRankingsTable renders qualification rankings, lowest rank first. The
// columns after DQ are whatever the season declared, so they change from year
// to year; nicknames fills the Name column and may be nil or incomplete.
//
// A column no team has anything in is dropped. Which columns a ranking table
// has already depends on the season, and 2015 -- which had no win/loss record
// at all -- otherwise prints a Record column that is blank for every team at
// the event.
//
// The rankings are sorted in place: the caller hands JSON output the same
// struct, and the two should agree on the order.
func eventRankingsTable(rankings *api.EventRankings, nicknames map[string]string) output.Table {
	sort.SliceStable(rankings.Rankings, func(i, j int) bool {
		return rankings.Rankings[i].Rank < rankings.Rankings[j].Rank
	})

	headers := []string{"Rank", "Team", "Name", "Record", "Played", "DQ"}
	for _, info := range rankings.SortOrderInfo {
		headers = append(headers, info.Name)
	}
	for _, info := range rankings.ExtraStatsInfo {
		headers = append(headers, info.Name)
	}

	rows := make([][]string, len(rankings.Rankings))
	for i, r := range rankings.Rankings {
		row := []string{
			strconv.Itoa(r.Rank),
			output.TeamNumberFromKey(r.TeamKey),
			nicknames[r.TeamKey],
			formatWLT(r.Record),
			strconv.Itoa(r.MatchesPlayed),
			strconv.Itoa(r.DQ),
		}
		row = append(row, formatStats(r.SortOrders, rankings.SortOrderInfo)...)
		row = append(row, formatStats(r.ExtraStats, rankings.ExtraStatsInfo)...)
		rows[i] = row
	}
	return output.Table{Headers: headers, Rows: rows}.DropEmptyColumns()
}

func newEventAlliancesCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "alliances <key>",
		Short: "Show playoff alliances",
		Long: `Show the playoff alliances at an event, how they were built and how far
they got.

Captain, Pick 1 and Pick 2 are the three teams in selection order. Backup names
the team called in mid-playoffs, as "1234 in for 5678" when the API says who it
replaced. Status and Level come from the alliance's playoff status, and Record
is its playoff win-loss-tie. The alliance that lost the final reads "finalist"
rather than the "eliminated" the API sends for every alliance that went out.`,
		Example: `  tba event alliances 2024cthar
  tba event alliances 2024cthar --format json
  tba event alliances 2024cthar --columns alliance,captain,status`,
		Args: exactArgs(1, "an event key (e.g. tba event alliances 2024cthar)"),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := validateEventKey(args[0]); err != nil {
				return err
			}
			client, err := newClient(cmd)
			if err != nil {
				return err
			}
			var alliances []api.EventAlliance
			if err := client.Get(cmd.Context(), fmt.Sprintf("/event/%s/alliances", args[0]), &alliances); err != nil {
				return err
			}
			table := eventAlliancesTable(alliances)
			return outputTable(cmd, alliances, table.Headers, table.Rows)
		},
	}
}

// eventAlliancesTable renders the playoff alliances in selection order.
func eventAlliancesTable(alliances []api.EventAlliance) output.Table {
	headers := []string{"Alliance", "Captain", "Pick 1", "Pick 2", "Backup", "Status", "Level", "Record", "Declines"}
	rows := make([][]string, len(alliances))
	for i, a := range alliances {
		// Most seasons name their alliances; the ones that do not are still
		// numbered by selection order.
		name := fmt.Sprintf("Alliance %d", i+1)
		if a.Name != nil && *a.Name != "" {
			name = *a.Name
		}
		status, level, record := "", "", ""
		if a.Status != nil {
			status = playoffStatus(a.Status)
			level = strings.ToUpper(a.Status.Level)
			record = formatWLT(a.Status.Record)
		}
		rows[i] = []string{
			name,
			pickAt(a.Picks, 0),
			pickAt(a.Picks, 1),
			pickAt(a.Picks, 2),
			formatBackup(a.Backup),
			status,
			level,
			record,
			joinTeamNumbers(a.Declines),
		}
	}
	return output.Table{Headers: headers, Rows: rows}
}

// pickAt renders the nth alliance pick, or an empty cell when an alliance is
// short a team (a three-team alliance in a two-team season, a bad feed).
func pickAt(picks []string, n int) string {
	if n >= len(picks) {
		return ""
	}
	return output.TeamNumberFromKey(picks[n])
}

// formatBackup describes a backup swap in one cell.
func formatBackup(b *api.AllianceBackup) string {
	if b == nil || b.In == "" {
		return ""
	}
	in := output.TeamNumberFromKey(b.In)
	if b.Out == "" {
		return in
	}
	return fmt.Sprintf("%s in for %s", in, output.TeamNumberFromKey(b.Out))
}

// joinTeamNumbers renders a list of team keys as bare numbers.
func joinTeamNumbers(keys []string) string {
	numbers := make([]string, len(keys))
	for i, k := range keys {
		numbers[i] = output.TeamNumberFromKey(k)
	}
	return strings.Join(numbers, ", ")
}

func newEventAwardsCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "awards <key>",
		Short: "Show an event's awards",
		Example: `  tba event awards 2024cthar
  tba event awards 2024cthar --format csv`,
		Args: exactArgs(1, "an event key (e.g. tba event awards 2024cthar)"),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := validateEventKey(args[0]); err != nil {
				return err
			}
			client, err := newClient(cmd)
			if err != nil {
				return err
			}
			var awards []api.Award
			if err := client.Get(cmd.Context(), fmt.Sprintf("/event/%s/awards", args[0]), &awards); err != nil {
				return err
			}
			table := eventAwardsTable(awards)
			return outputTable(cmd, awards, table.Headers, table.Rows)
		},
	}
}

// eventAwardsTable renders an event's awards in the order given.
func eventAwardsTable(awards []api.Award) output.Table {
	rows := make([][]string, len(awards))
	for i, a := range awards {
		recipient := ""
		if len(a.Recipients) > 0 {
			r := a.Recipients[0]
			if r.TeamKey != nil {
				recipient = output.TeamNumberFromKey(*r.TeamKey)
			}
			if r.Awardee != nil {
				if recipient != "" {
					recipient += " - "
				}
				recipient += *r.Awardee
			}
		}
		rows[i] = []string{a.Name, recipient}
	}
	return output.Table{Headers: []string{"Award", "Recipient"}, Rows: rows}
}

func newEventOPRsCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "oprs <key>",
		Short: "Show OPR, DPR and CCWM for each team",
		Long: `Show the contributions TBA calculates for each team at an event.

OPR is offensive power rating, DPR defensive power rating and CCWM calculated
contribution to winning margin. The table is ordered by OPR, highest first,
since that is the question the command is asked; --sort reorders it by any
column, and ties keep team-number order.`,
		Example: `  tba event oprs 2024cthar
  tba event oprs 2024cthar --sort team
  tba event oprs 2024cthar --format csv`,
		Args: exactArgs(1, "an event key (e.g. tba event oprs 2024cthar)"),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := validateEventKey(args[0]); err != nil {
				return err
			}
			client, err := newClient(cmd)
			if err != nil {
				return err
			}
			var oprs api.EventOPRs
			if err := client.Get(cmd.Context(), fmt.Sprintf("/event/%s/oprs", args[0]), &oprs); err != nil {
				return err
			}
			table := eventOPRsTable(oprs)
			return outputTable(cmd, oprs, table.Headers, table.Rows)
		},
	}
}

// eventOPRsTable renders the calculated contributions, one row per team,
// highest OPR first.
//
// Team-number order was the wrong default: nobody asks for an event's OPRs to
// find out what team 177 scored, they ask to see who the strongest teams were.
// Ties fall back to team number so the order is the same every run, and
// --sort is there for anyone who wants it another way.
func eventOPRsTable(oprs api.EventOPRs) output.Table {
	teamKeys := make([]string, 0, len(oprs.OPRs))
	for team := range oprs.OPRs {
		teamKeys = append(teamKeys, team)
	}
	// Map iteration order is random; sort so the output is stable.
	sortTeamKeys(teamKeys)
	sort.SliceStable(teamKeys, func(i, j int) bool {
		return oprs.OPRs[teamKeys[i]] > oprs.OPRs[teamKeys[j]]
	})
	var rows [][]string
	for _, team := range teamKeys {
		rows = append(rows, []string{
			output.TeamNumberFromKey(team),
			fmt.Sprintf("%.2f", oprs.OPRs[team]),
			fmt.Sprintf("%.2f", oprs.DPRs[team]),
			fmt.Sprintf("%.2f", oprs.CCWMs[team]),
		})
	}
	return output.Table{Headers: []string{"Team", "OPR", "DPR", "CCWM"}, Rows: rows}
}

func newEventDistrictPointsCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "district-points <key>",
		Short: "Show the district points an event awarded",
		Long: `Show the district points an event awarded, highest total first.

--tiebreakers adds the values that break a tie on total points: the team's
highest qualification scores, best first, and its number of qualification wins.

JSON output keeps the shape the API returns, an object keyed by team, so
--sort only reorders the tabular formats.`,
		Example: `  tba event district-points 2024cthar
  tba event district-points 2024cthar --tiebreakers
  tba event district-points 2024cthar --jq '.points.frc177.total'`,
		Args: exactArgs(1, "an event key (e.g. tba event district-points 2024cthar)"),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := validateEventKey(args[0]); err != nil {
				return err
			}
			client, err := newClient(cmd)
			if err != nil {
				return err
			}
			var points api.EventDistrictPoints
			if err := client.Get(cmd.Context(), fmt.Sprintf("/event/%s/district_points", args[0]), &points); err != nil {
				return err
			}

			withTiebreakers, _ := cmd.Flags().GetBool("tiebreakers")
			table := eventDistrictPointsTable(points, withTiebreakers)
			// The parsed struct, not the raw body: PermuteSlice would happily
			// treat a json.RawMessage as a slice of bytes to reorder.
			return outputTable(cmd, points, table.Headers, table.Rows)
		},
	}
	c.Flags().Bool("tiebreakers", false, "Add the highest qual scores and qual wins columns")
	return c
}

// The predictions and insights commands live in event_insights.go.
// eventDistrictPointsTable renders an event's district points, highest total
// first. withTiebreakers adds the two columns that break a tie on total.
func eventDistrictPointsTable(points api.EventDistrictPoints, withTiebreakers bool) output.Table {
	teams := make([]string, 0, len(points.Points))
	for key := range points.Points {
		teams = append(teams, key)
	}
	// Map order is random, so sort by team number first and then by total: the
	// result is total descending with ties broken by team number, and it is
	// the same every run.
	sortTeamKeys(teams)
	sort.SliceStable(teams, func(i, j int) bool {
		return points.Points[teams[i]].Total > points.Points[teams[j]].Total
	})

	headers := []string{"Team", "Qual", "Alliance", "Award", "Elim", "Total"}
	if withTiebreakers {
		headers = append(headers, "Highest Qual Scores", "Qual Wins")
	}

	rows := make([][]string, len(teams))
	for i, key := range teams {
		p := points.Points[key]
		row := []string{
			output.TeamNumberFromKey(key),
			strconv.Itoa(p.QualPoints),
			strconv.Itoa(p.AlliancePoints),
			strconv.Itoa(p.AwardPoints),
			strconv.Itoa(p.ElimPoints),
			strconv.Itoa(p.Total),
		}
		if withTiebreakers {
			tb := points.Tiebreakers[key]
			scores := make([]string, len(tb.HighestQualScores))
			for n, v := range tb.HighestQualScores {
				scores[n] = strconv.Itoa(v)
			}
			row = append(row, strings.Join(scores, ", "), strconv.Itoa(tb.QualWins))
		}
		rows[i] = row
	}
	return output.Table{Headers: headers, Rows: rows}
}
