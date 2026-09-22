package cmd

import (
	"sort"
	"strconv"

	"github.com/the-blue-alliance/tba-cli/internal/api"
	"github.com/the-blue-alliance/tba-cli/internal/output"
)

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
