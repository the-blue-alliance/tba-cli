package cmd

import (
	"fmt"
	"sort"
	"strconv"

	"github.com/spf13/cobra"
	"github.com/the-blue-alliance/tba-cli/internal/api"
	"github.com/the-blue-alliance/tba-cli/internal/clierr"
	"github.com/the-blue-alliance/tba-cli/internal/output"
)

func newDistrictCmd() *cobra.Command {
	districtCmd := &cobra.Command{
		Use:     "district",
		Aliases: []string{"districts"},
		Short:   "Work with districts",
	}
	districtCmd.AddCommand(newDistrictListCmd())
	districtCmd.AddCommand(newDistrictEventsCmd())
	districtCmd.AddCommand(newDistrictTeamsCmd())
	districtCmd.AddCommand(newDistrictRankingsCmd())
	return districtCmd
}

func newDistrictListCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "list",
		Short: "List a season's districts",
		Example: `  tba district list --year 2024
  tba districts list --year 2024 --format csv`,
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := newClient(cmd)
			if err != nil {
				return err
			}
			year, err := resolveYear(cmd)
			if err != nil {
				return err
			}
			var districts []api.District
			if err := client.Get(cmd.Context(), fmt.Sprintf("/districts/%d", year), &districts); err != nil {
				return err
			}
			rows := make([][]string, len(districts))
			for i, d := range districts {
				rows[i] = []string{d.Key, d.DisplayName, d.Abbreviation}
			}
			return outputTableWithEmptyNote(cmd, districts, []string{"Key", "Name", "Abbreviation"}, rows,
				fmt.Sprintf("no districts in %d", year))
		},
	}
	addYearFlag(c)
	return c
}

func newDistrictEventsCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "events <key>",
		Short: "List a district's events",
		Example: `  tba district events 2024ne
  tba district events 2024ne --format csv`,
		Args: exactArgs(1, "a district key (e.g. tba district events 2024ne)"),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := newClient(cmd)
			if err != nil {
				return err
			}
			var events []api.Event
			if err := client.Get(cmd.Context(), fmt.Sprintf("/district/%s/events", args[0]), &events); err != nil {
				return err
			}
			rows := make([][]string, len(events))
			for i, e := range events {
				rows[i] = []string{e.Key, e.Name, e.StartDate}
			}
			return outputTableWithEmptyNote(cmd, events, []string{"Key", "Name", "Start Date"}, rows,
				fmt.Sprintf("no events in district %s", args[0]))
		},
	}
}

func newDistrictTeamsCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "teams <key>",
		Short: "List a district's teams",
		Example: `  tba district teams 2024ne
  tba district teams 2024ne --format tsv`,
		Args: exactArgs(1, "a district key (e.g. tba district teams 2024ne)"),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := newClient(cmd)
			if err != nil {
				return err
			}
			var teams []api.Team
			if err := client.Get(cmd.Context(), fmt.Sprintf("/district/%s/teams", args[0]), &teams); err != nil {
				return err
			}
			rows := make([][]string, len(teams))
			for i, t := range teams {
				rows[i] = []string{fmt.Sprintf("%d", t.TeamNumber), t.Nickname, output.FormatLocation(t.City, t.StateProv, t.Country)}
			}
			return outputTableWithEmptyNote(cmd, teams, []string{"Number", "Name", "Location"}, rows,
				fmt.Sprintf("no teams in district %s", args[0]))
		},
	}
}

func newDistrictRankingsCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "rankings <key>",
		Short: "Show a district's season rankings",
		Long: `Show a district's season rankings, one row per team.

A team's points come from its two qualifying district events, shown as Event 1
and Event 2 in the order they were played, plus the district championship
(DCMP) and any rookie bonus. --detail breaks each qualifying event down into
its qual, alliance, award and elim points.

--cutoff N draws a separator after rank N, where the district championship cut
falls, in table and markdown output.

The ranks the API publishes include district championship points once the DCMP
has been played, so late in a season "top N" is a line through the final
standings rather than through the cut that decided who went. --pre-dcmp ranks
instead on the points a team had before the DCMP, renumbers Rank by that
order, keeps the published rank as Season Rank, adds the pre-DCMP total as its
own column, and puts the cut line there; the cut line says which of the two it
is. The JSON is the API's own answer either way.`,
		Example: `  tba district rankings 2024ne
  tba district rankings 2024ne --format markdown
  tba district rankings 2024ne --cutoff 80
  tba district rankings 2024ne --cutoff 80 --pre-dcmp
  tba district rankings 2024ne --detail --columns team,"e1 qual","e2 qual"`,
		Args: exactArgs(1, "a district key (e.g. tba district rankings 2024ne)"),
		RunE: func(cmd *cobra.Command, args []string) error {
			cutoff, _ := cmd.Flags().GetInt("cutoff")
			if cutoff < 0 {
				return clierr.Usage("--cutoff must be a positive rank, got %d", cutoff)
			}
			format, err := resolveFormat(cmd)
			if err != nil {
				return err
			}
			client, err := newClient(cmd)
			if err != nil {
				return err
			}
			var rankings []api.DistrictRanking
			if err := client.Get(cmd.Context(), fmt.Sprintf("/district/%s/rankings", args[0]), &rankings); err != nil {
				return err
			}
			sort.SliceStable(rankings, func(i, j int) bool { return rankings[i].Rank < rankings[j].Rank })

			preDCMP, _ := cmd.Flags().GetBool("pre-dcmp")
			if preDCMP {
				// Sorted in place: JSON output is handed the same slice, and
				// the two should agree on the order.
				sort.SliceStable(rankings, func(i, j int) bool {
					return preDCMPTotal(rankings[i]) > preDCMPTotal(rankings[j])
				})
			}

			detail, _ := cmd.Flags().GetBool("detail")
			// Two qualifying events is the rule, but a team can be rostered at
			// a third, and dropping those points silently would be worse than
			// an extra column.
			eventColumns := 2
			for _, r := range rankings {
				if n := len(qualifyingEvents(r)); n > eventColumns {
					eventColumns = n
				}
			}

			headers := []string{"Rank"}
			if preDCMP {
				// Rank counts the rows as they are now ordered, so the number
				// the API published needs a column of its own -- and a name
				// that says which of the two it is.
				headers = append(headers, "Season Rank")
			}
			headers = append(headers, "Team", "Rookie Bonus")
			for i := 1; i <= eventColumns; i++ {
				headers = append(headers, fmt.Sprintf("Event %d", i))
				if detail {
					headers = append(headers,
						fmt.Sprintf("E%d Qual", i),
						fmt.Sprintf("E%d Alliance", i),
						fmt.Sprintf("E%d Award", i),
						fmt.Sprintf("E%d Elim", i))
				}
			}
			headers = append(headers, "DCMP", "Total")
			if preDCMP {
				headers = append(headers, "Pre-DCMP")
			}

			rows := make([][]string, len(rankings))
			for i, r := range rankings {
				events := qualifyingEvents(r)
				// A rank next to a cut line has to be the rank the cut was
				// made on: the published one counts district championship
				// points, so under --pre-dcmp it ran 2, 3, 11, 5 down a table
				// ordered by something else.
				rank := r.Rank
				if preDCMP {
					rank = i + 1
				}
				row := []string{strconv.Itoa(rank)}
				if preDCMP {
					row = append(row, strconv.Itoa(r.Rank))
				}
				row = append(row,
					output.TeamNumberFromKey(r.TeamKey),
					strconv.Itoa(r.RookieBonus),
				)
				for n := 0; n < eventColumns; n++ {
					if n >= len(events) {
						row = append(row, "")
						if detail {
							row = append(row, "", "", "", "")
						}
						continue
					}
					e := events[n]
					row = append(row, strconv.Itoa(e.Total))
					if detail {
						row = append(row,
							strconv.Itoa(e.QualPoints),
							strconv.Itoa(e.AlliancePoints),
							strconv.Itoa(e.AwardPoints),
							strconv.Itoa(e.ElimPoints))
					}
				}
				row = append(row, districtCMPPoints(r), strconv.Itoa(r.PointTotal))
				if preDCMP {
					row = append(row, strconv.Itoa(preDCMPTotal(r)))
				}
				rows[i] = row
			}

			table := output.Table{Headers: headers, Rows: rows}
			if cutoff > 0 {
				table.Dividers = cutoffDivider(cmd, rankings, cutoff, preDCMP, format)
			}
			return outputTableWith(cmd, rankings, table)
		},
	}
	c.Flags().Int("cutoff", 0, "Draw a separator after rank N, where the DCMP cut falls")
	c.Flags().Bool("pre-dcmp", false, "Rank on the points each team had before the district championship")
	c.Flags().Bool("detail", false, "Break each qualifying event into qual/alliance/award/elim points")
	return c
}

// qualifyingEvents returns a team's non-championship district events, in the
// order the API listed them, which is the order they were played.
func qualifyingEvents(r api.DistrictRanking) []api.DistrictEventPoints {
	events := make([]api.DistrictEventPoints, 0, len(r.EventPoints))
	for _, e := range r.EventPoints {
		if !e.DistrictCMP {
			events = append(events, e)
		}
	}
	return events
}

// districtCMPPoints totals a team's district championship points, or returns an
// empty cell for a team that has not been to one. A championship can be split
// into divisions, so more than one entry may be flagged.
func districtCMPPoints(r api.DistrictRanking) string {
	total, played := 0, false
	for _, e := range r.EventPoints {
		if e.DistrictCMP {
			total += e.Total
			played = true
		}
	}
	if !played {
		return ""
	}
	return strconv.Itoa(total)
}

// preDCMPTotal is a team's season points without its district championship,
// which is the number the DCMP cut was actually made on.
func preDCMPTotal(r api.DistrictRanking) int {
	total := r.PointTotal
	for _, e := range r.EventPoints {
		if e.DistrictCMP {
			total -= e.Total
		}
	}
	return total
}

// anyDCMPPoints reports whether the district championship has been scored for
// anybody in the list, which is what makes the published totals post-DCMP.
func anyDCMPPoints(rankings []api.DistrictRanking) bool {
	for _, r := range rankings {
		for _, e := range r.EventPoints {
			if e.DistrictCMP && e.Total != 0 {
				return true
			}
		}
	}
	return false
}

// cutoffDivider places the DCMP cut line after the last team above the cut.
//
// The line is presentation, so it is only drawn for the formats a person
// reads: csv, tsv and json stay machine-clean. It also depends on the rows
// being in the order the command put them in, so --sort takes precedence and
// says so on stderr rather than leaving a line floating in the middle of a
// re-sorted table.
//
// What the line says depends on what it is cutting. Once the district
// championship has been scored the published ranks include its points, so a
// "top N" line no longer marks who qualified for it; the label says as much,
// and --pre-dcmp draws the line through the standings that did decide it.
func cutoffDivider(cmd *cobra.Command, rankings []api.DistrictRanking, cutoff int, preDCMP bool, format string) map[int]string {
	if format != "table" && format != "markdown" {
		return nil
	}
	if sortSpec, _ := cmd.Flags().GetString("sort"); sortSpec != "" {
		fmt.Fprintf(cmd.ErrOrStderr(), "note: --cutoff needs rank order, so no cut line is drawn with --sort\n")
		return nil
	}

	at := len(rankings)
	if preDCMP {
		// The rows are already ordered by pre-DCMP total, so the cut is
		// simply after the Nth of them.
		if cutoff < at {
			at = cutoff
		}
	} else {
		for i, r := range rankings {
			if r.Rank > cutoff {
				at = i
				break
			}
		}
	}
	// Nothing to separate: every team is above the cut.
	if at >= len(rankings) || at <= 0 {
		return nil
	}

	label := fmt.Sprintf("DCMP cutoff (top %d)", cutoff)
	switch {
	case preDCMP:
		label = fmt.Sprintf("top %d by pre-DCMP total", cutoff)
	case anyDCMPPoints(rankings):
		label = fmt.Sprintf("top %d by current total (includes DCMP points)", cutoff)
	}
	return map[int]string{at - 1: label}
}
