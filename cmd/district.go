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
		Short: "List districts for a year",
		Example: `  tba district list --year 2024
  tba districts list --year 2024 --format csv`,
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := newClient(cmd)
			if err != nil {
				return err
			}
			year, _ := cmd.Flags().GetInt("year")
			var districts []api.District
			if err := client.Get(cmd.Context(), fmt.Sprintf("/districts/%d", year), &districts); err != nil {
				return err
			}
			rows := make([][]string, len(districts))
			for i, d := range districts {
				rows[i] = []string{d.Key, d.DisplayName, d.Abbreviation}
			}
			return outputTable(cmd, districts, []string{"Key", "Name", "Abbreviation"}, rows)
		},
	}
	c.Flags().Int("year", currentYear(), "Season year (default: current year)")
	return c
}

func newDistrictEventsCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "events <key>",
		Short: "List district events",
		Example: `  tba district events 2024ne
  tba district events 2024ne --format csv`,
		Args: cobra.ExactArgs(1),
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
			return outputTable(cmd, events, []string{"Key", "Name", "Start Date"}, rows)
		},
	}
}

func newDistrictTeamsCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "teams <key>",
		Short: "List district teams",
		Example: `  tba district teams 2024ne
  tba district teams 2024ne --format tsv`,
		Args: cobra.ExactArgs(1),
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
			return outputTable(cmd, teams, []string{"Number", "Name", "Location"}, rows)
		},
	}
}

func newDistrictRankingsCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "rankings <key>",
		Short: "Show district rankings",
		Long: `Show a district's season rankings, one row per team.

A team's points come from its two qualifying district events, shown as Event 1
and Event 2 in the order they were played, plus the district championship
(DCMP) and any rookie bonus. --detail breaks each qualifying event down into
its qual, alliance, award and elim points.

--cutoff N draws a separator after rank N, where the district championship cut
falls, in table and markdown output.`,
		Example: `  tba district rankings 2024ne
  tba district rankings 2024ne --format markdown
  tba district rankings 2024ne --cutoff 80
  tba district rankings 2024ne --detail --columns team,"e1 qual","e2 qual"`,
		Args: cobra.ExactArgs(1),
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

			headers := []string{"Rank", "Team", "Rookie Bonus"}
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

			rows := make([][]string, len(rankings))
			for i, r := range rankings {
				events := qualifyingEvents(r)
				row := []string{
					strconv.Itoa(r.Rank),
					output.TeamNumberFromKey(r.TeamKey),
					strconv.Itoa(r.RookieBonus),
				}
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
				rows[i] = row
			}

			if cutoff > 0 {
				rows = insertCutoff(cmd, rows, rankings, cutoff, format, len(headers))
			}
			return outputTable(cmd, rankings, headers, rows)
		},
	}
	c.Flags().Int("cutoff", 0, "Draw a separator after rank N, where the DCMP cut falls")
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

// insertCutoff splices the DCMP cut marker in after the last team at or above
// rank cutoff.
//
// The marker is a presentational row, so it is only drawn for the formats a
// person reads: csv, tsv and json stay machine-clean. It also depends on the
// rows being in rank order, so --sort takes precedence and says so on stderr
// rather than leaving a line floating in the middle of a re-sorted table.
func insertCutoff(cmd *cobra.Command, rows [][]string, rankings []api.DistrictRanking, cutoff int, format string, columns int) [][]string {
	if format != "table" && format != "markdown" {
		return rows
	}
	if sortSpec, _ := cmd.Flags().GetString("sort"); sortSpec != "" {
		fmt.Fprintf(cmd.ErrOrStderr(), "note: --cutoff needs rank order, so no cut line is drawn with --sort\n")
		return rows
	}
	at := len(rankings)
	for i, r := range rankings {
		if r.Rank > cutoff {
			at = i
			break
		}
	}
	// Nothing to separate: every team is above the cut.
	if at >= len(rows) {
		return rows
	}
	// Padded to the full width so that markdown still sees a well-formed row.
	marker := make([]string, columns)
	marker[0] = fmt.Sprintf("--- DCMP cutoff (top %d) ---", cutoff)
	out := make([][]string, 0, len(rows)+1)
	out = append(out, rows[:at]...)
	out = append(out, marker)
	out = append(out, rows[at:]...)
	return out
}
