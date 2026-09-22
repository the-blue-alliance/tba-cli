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

func newEventTeamStatusesCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "team-statuses <key>",
		Short: "Show where every team at an event stands",
		Long: `Show one row per team at an event: qualification rank and record, the
alliance that picked them and in which slot, how far they got in the playoffs,
and TBA's own one-line summary.

Teams are listed by rank, with teams that have no rank yet last. Overall is
TBA's overall_status_str with its markup removed.`,
		Example: `  tba event team-statuses 2024cthar
  tba event team-statuses 2024cthar --format csv
  tba event team-statuses 2024cthar --columns team,rank,record,overall`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := validateEventKey(args[0]); err != nil {
				return err
			}
			client, err := newClient(cmd)
			if err != nil {
				return err
			}
			// The endpoint answers with an object keyed by team, and a team
			// with nothing to report maps to a bare null.
			var statuses map[string]*api.TeamEventStatus
			if err := client.Get(cmd.Context(), fmt.Sprintf("/event/%s/teams/statuses", args[0]), &statuses); err != nil {
				return err
			}

			table := eventTeamStatusesTable(statuses)
			// The parsed map, so --jq still sees the shape the API returns.
			// It is not a slice, so --sort reorders the table only.
			return outputTable(cmd, statuses, table.Headers, table.Rows)
		},
	}
}

// eventTeamStatusesTable renders one row per team at an event, by rank, with
// the teams that have no rank yet last.
func eventTeamStatusesTable(statuses map[string]*api.TeamEventStatus) output.Table {
	teams := make([]string, 0, len(statuses))
	for key := range statuses {
		teams = append(teams, key)
	}
	sortTeamKeys(teams)
	sort.SliceStable(teams, func(i, j int) bool {
		a, aOK := qualRank(statuses[teams[i]])
		b, bOK := qualRank(statuses[teams[j]])
		if aOK != bOK {
			// An unranked team sorts after every ranked one rather than at
			// rank zero.
			return aOK
		}
		return aOK && a < b
	})

	headers := []string{"Team", "Rank", "Record", "Alliance", "Pick", "Playoff Level", "Playoff Status", "Overall"}
	rows := make([][]string, len(teams))
	for i, key := range teams {
		rows[i] = teamStatusRow(key, statuses[key])
	}
	return output.Table{Headers: headers, Rows: rows}
}

// qualRank returns a team's qualification rank, and whether it has one at all.
func qualRank(s *api.TeamEventStatus) (int, bool) {
	if s == nil || s.Qual == nil || s.Qual.Ranking == nil || s.Qual.Ranking.Rank == 0 {
		return 0, false
	}
	return s.Qual.Ranking.Rank, true
}

// teamStatusRow renders one team's row. Every part of a status is optional, so
// a team that has not played, was not picked, or is simply absent from the
// feed still gets a row with its number in it.
func teamStatusRow(key string, s *api.TeamEventStatus) []string {
	row := []string{output.TeamNumberFromKey(key), "", "", "", "", "", "", ""}
	if s == nil {
		return row
	}
	if s.Qual != nil && s.Qual.Ranking != nil {
		if rank := s.Qual.Ranking.Rank; rank > 0 {
			row[1] = strconv.Itoa(rank)
		}
		row[2] = formatWLT(s.Qual.Ranking.Record)
	}
	if s.Alliance != nil {
		row[3] = s.Alliance.Name
		if row[3] == "" {
			row[3] = fmt.Sprintf("Alliance %d", s.Alliance.Number)
		}
		row[4] = formatAlliancePick(s.Alliance.Pick)
	}
	if s.Playoff != nil {
		row[5] = strings.ToUpper(s.Playoff.Level)
		row[6] = s.Playoff.Status
	}
	row[7] = output.StripHTML(s.OverallStatusStr)
	return row
}

// formatAlliancePick names an alliance slot: TBA numbers the captain 0 and the
// picks from 1, and uses a negative index for a backup team.
func formatAlliancePick(pick int) string {
	switch {
	case pick < 0:
		return "Backup"
	case pick == 0:
		return "Captain"
	default:
		return strconv.Itoa(pick)
	}
}
