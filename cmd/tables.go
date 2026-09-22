package cmd

import (
	"errors"
	"sort"
	"strconv"

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
	return output.Render(w, table, output.RenderOptions{
		Format:    format,
		NoHeaders: settings(cmd).Bool("no-headers"),
		Color:     color,
	})
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
