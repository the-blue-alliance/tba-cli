package cmd

import (
	"errors"
	"sort"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
	"github.com/the-blue-alliance/tba-cli/internal/api"
	"github.com/the-blue-alliance/tba-cli/internal/output"
)

// outputTableWith is outputTable for a table that carries more than headers
// and rows — today, the dividers a cut line is drawn with. It routes through
// exactly the same flags, so --sort, --columns and --format behave the same
// whichever entry point a command uses.
func outputTableWith(cmd *cobra.Command, data interface{}, table output.Table) error {
	format, err := resolveFormat(cmd)
	if err != nil {
		return err
	}
	color, err := colorMode(cmd)
	if err != nil {
		return err
	}
	w := cmd.OutOrStdout()

	// Sorting runs before column selection so that a table can be ordered by a
	// column the user chose not to display.
	sortSpec := settings(cmd).String("sort")
	var order []int
	if sortSpec != "" {
		if order, err = table.SortOrder(sortSpec); err != nil {
			return err
		}
		table = table.Reorder(order)
	}

	columns := settings(cmd).String("columns")
	if format == "json" {
		if columns != "" {
			return errors.New("--columns applies to tabular formats; use --jq to shape JSON")
		}
		return output.PrintJSONWithFilter(w, output.PermuteSlice(data, order), jqExpr(cmd), rawOutput(cmd))
	}
	if columns != "" {
		if table, err = table.SelectColumns(columns); err != nil {
			return err
		}
	}
	return renderRows(w, table, output.RenderOptions{
		Format:    format,
		NoHeaders: settings(cmd).Bool("no-headers"),
		Color:     color,
	})
}

// dropEmptyColumnsFor removes the columns nothing in the table filled in —
// but only from the formats a person reads.
//
// A blank stripe down a whole table is a header pretending to be data: 2015
// had no win/loss record, so its rankings have nothing in Record, and a
// finished event has nothing to count down to in When. Dropping the column is
// right for a table on screen and for the markdown that is a table on a page.
//
// csv and tsv keep every column, because what they produce is a file, and a
// file's header is a schema. Dropping a column there made the shape of the
// data depend on which event was asked for: a script that read Record off
// column four got a different column four from a 2015 event, and two exports
// concatenated did not line up. An empty column is the honest answer that the
// season never had that statistic.
func dropEmptyColumnsFor(format string, table output.Table) output.Table {
	if format == "table" || format == "markdown" {
		return table.DropEmptyColumns()
	}
	return table
}

// playoffStatus renders an alliance's playoff standing.
//
// TBA marks the alliance that lost the final "eliminated", the same word it
// gives an alliance knocked out in the first round. Nobody describes second
// place that way, and the level is right there to tell the two apart, so an
// alliance eliminated at level "f" is the finalist.
func playoffStatus(s *api.AllianceStatus) string {
	if s == nil {
		return ""
	}
	if strings.EqualFold(s.Status, "eliminated") && strings.EqualFold(s.Level, "f") {
		return "finalist"
	}
	return s.Status
}

// formatWLT renders a win-loss-tie record. A nil record — 2015, which had no
// win/loss at all — renders as an empty cell rather than a misleading "0-0-0".
func formatWLT(r *api.WLTRecord) string {
	if r == nil {
		return ""
	}
	return strconv.Itoa(r.Wins) + "-" + strconv.Itoa(r.Losses) + "-" + strconv.Itoa(r.Ties)
}

// formatStats renders one row's ranking statistics using the precision the API
// declared for each column, so "Ranking Score" keeps its two decimals while a
// counting stat stays an integer. Missing values leave the cell empty instead
// of printing a zero the API never sent.
func formatStats(values []float64, info []api.SortOrderInfo) []string {
	cells := make([]string, len(info))
	for i := range info {
		if i >= len(values) {
			continue
		}
		precision := info[i].Precision
		if precision < 0 {
			precision = 0
		}
		cells[i] = strconv.FormatFloat(values[i], 'f', precision, 64)
	}
	return cells
}

// teamNumber parses the numeric part of a team key, for ordering. Keys that do
// not parse sort last, keeping the comparison total.
func teamNumber(key string) int {
	n, err := strconv.Atoi(output.TeamNumberFromKey(key))
	if err != nil {
		return 1 << 30
	}
	return n
}

// sortTeamKeys orders team keys by team number, which is the order a person
// reading a list of teams expects. Map iteration order is random, so every
// table built from a keyed object goes through here to stay reproducible.
func sortTeamKeys(keys []string) {
	sort.Slice(keys, func(i, j int) bool {
		a, b := teamNumber(keys[i]), teamNumber(keys[j])
		if a != b {
			return a < b
		}
		return keys[i] < keys[j]
	})
}
