package cmd

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
	"github.com/the-blue-alliance/tba-cli/internal/api"
	"github.com/the-blue-alliance/tba-cli/internal/clierr"
	"github.com/the-blue-alliance/tba-cli/internal/output"
)

// The insights endpoints prefix every board's machine name with its kind.
const (
	leaderboardPrefix = "typed_leaderboard_"
	notablePrefix     = "notables_"
)

func newInsightCmd() *cobra.Command {
	insightCmd := &cobra.Command{
		Use:     "insight",
		Aliases: []string{"insights"},
		Short:   "View insights and leaderboards",
	}
	insightCmd.AddCommand(newInsightLeaderboardsCmd())
	insightCmd.AddCommand(newInsightNotablesCmd())
	return insightCmd
}

func newInsightLeaderboardsCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "leaderboards",
		Short: "Show a season's leaderboards",
		Long: `Show every leaderboard TBA publishes for a season.

The table is Leaderboard | Rank | Key | Value, one board after another in the
order the API sends them. Teams tied on a value share a rank and are listed in
one cell, and a board about teams shows bare team numbers.

--board picks a single board by name, in either spelling: "Blue Banners" or
typed_leaderboard_blue_banners, case-insensitively. --limit caps how many rows
each board contributes, which keeps a season's worth of boards readable; pass
--limit 0 for all of them.

A board such as Blue Banners ties hundreds of teams on one value, so a tie
shows the first 10 keys and a count of the rest; --expand prints every key.

JSON output stays the array the API sent, narrowed to the board --board named.`,
		Example: `  tba insight leaderboards --year 2024
  tba insight leaderboards --year 2024 --board "Blue Banners"
  tba insight leaderboards --year 2024 --board "Blue Banners" --expand
  tba insight leaderboards --year 2024 --limit 0 --format csv
  tba insights leaderboards --year 2024 --jq '.[0].name' -r`,
		RunE: func(cmd *cobra.Command, args []string) error {
			limit, _ := cmd.Flags().GetInt("limit")
			if limit < 0 {
				return clierr.Usage("--limit cannot be negative (0 means every row)")
			}
			expand, _ := cmd.Flags().GetBool("expand")
			client, err := newClient(cmd)
			if err != nil {
				return err
			}
			year, err := resolveYear(cmd, client)
			if err != nil {
				return err
			}
			raw, err := client.GetRaw(cmd.Context(), fmt.Sprintf("/insights/leaderboards/%d", year))
			if err != nil {
				return err
			}
			var boards []api.InsightLeaderboard
			if err := json.Unmarshal(raw, &boards); err != nil {
				return fmt.Errorf("reading leaderboards for %d: %w", year, err)
			}

			names := make([]string, len(boards))
			for i, b := range boards {
				names[i] = b.Name
			}
			wanted, err := selectBoards(cmd, names, leaderboardPrefix)
			if err != nil {
				return err
			}

			var rows [][]string
			for _, i := range wanted {
				b := boards[i]
				label := humanizeName(strings.TrimPrefix(b.Name, leaderboardPrefix))
				for rank, r := range b.Data.Rankings {
					if limit > 0 && rank >= limit {
						break
					}
					rows = append(rows, []string{
						label,
						strconv.Itoa(rank + 1),
						joinInsightKeys(r.Keys, b.Data.KeyType, expand),
						formatNumber(r.Value),
					})
				}
			}
			data, err := narrowInsightJSON(raw, wanted, len(boards))
			if err != nil {
				return err
			}
			return outputTable(cmd, data, []string{"Leaderboard", "Rank", "Key", "Value"}, rows)
		},
	}
	addYearFlag(c)
	c.Flags().String("board", "", `Only this leaderboard, by name (e.g. "Blue Banners")`)
	c.Flags().Int("limit", 10, "Rows per leaderboard, or 0 for all of them")
	c.Flags().Bool("expand", false, "List every tied key instead of the first few")
	return c
}

func newInsightNotablesCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "notables",
		Short: "Show a season's notable teams",
		Long: `Show the notable teams TBA lists for a season.

The table is Notable | Team | Context, where Context is whatever the board says
earned the entry -- the events or years behind it -- joined with commas.

--board picks a single board by name, in either spelling: "Hall Of Fame" or
notables_hall_of_fame, case-insensitively.

JSON output stays the array the API sent, narrowed to the board --board named.`,
		Example: `  tba insight notables --year 2024
  tba insight notables --year 2024 --board "Hall Of Fame"
  tba insights notables --year 2024 --format json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := newClient(cmd)
			if err != nil {
				return err
			}
			year, err := resolveYear(cmd, client)
			if err != nil {
				return err
			}
			raw, err := client.GetRaw(cmd.Context(), fmt.Sprintf("/insights/notables/%d", year))
			if err != nil {
				return err
			}
			var boards []api.InsightNotable
			if err := json.Unmarshal(raw, &boards); err != nil {
				return fmt.Errorf("reading notables for %d: %w", year, err)
			}

			names := make([]string, len(boards))
			for i, b := range boards {
				names[i] = b.Name
			}
			wanted, err := selectBoards(cmd, names, notablePrefix)
			if err != nil {
				return err
			}

			var rows [][]string
			for _, i := range wanted {
				b := boards[i]
				label := humanizeName(strings.TrimPrefix(b.Name, notablePrefix))
				for _, e := range b.Data.Entries {
					rows = append(rows, []string{
						label,
						output.TeamNumberFromKey(e.TeamKey),
						strings.Join(e.Context, ", "),
					})
				}
			}
			data, err := narrowInsightJSON(raw, wanted, len(boards))
			if err != nil {
				return err
			}
			return outputTable(cmd, data, []string{"Notable", "Team", "Context"}, rows)
		},
	}
	addYearFlag(c)
	c.Flags().String("board", "", `Only this notable board, by name (e.g. "Hall Of Fame")`)
	return c
}

// selectBoards resolves --board to the indices it names, or to every board when
// the flag was not given. A name that matches nothing is a usage error that
// lists what the season actually has, since the boards change from year to year
// and guessing at one is the common mistake.
func selectBoards(cmd *cobra.Command, names []string, prefix string) ([]int, error) {
	want, _ := cmd.Flags().GetString("board")
	all := make([]int, len(names))
	for i := range names {
		all[i] = i
	}
	if strings.TrimSpace(want) == "" {
		return all, nil
	}

	target := normalizeBoardName(strings.TrimPrefix(strings.TrimSpace(want), prefix))
	labels := make([]string, len(names))
	var matched []int
	for i, n := range names {
		short := strings.TrimPrefix(n, prefix)
		labels[i] = humanizeName(short)
		if normalizeBoardName(short) == target {
			matched = append(matched, i)
		}
	}
	if len(matched) > 0 {
		return matched, nil
	}
	if len(names) == 0 {
		return nil, clierr.Usage("unknown --board %q: this season has no boards", want)
	}
	return nil, clierr.Usage("unknown --board %q (available: %s)", want, strings.Join(labels, ", "))
}

// narrowInsightJSON returns the JSON the command should print. An unfiltered
// run hands back the body exactly as it arrived; a filtered one re-encodes the
// boards that survived, keeping the array shape of the endpoint.
func narrowInsightJSON(raw json.RawMessage, wanted []int, total int) (json.RawMessage, error) {
	if len(wanted) == total {
		return raw, nil
	}
	var items []json.RawMessage
	if err := json.Unmarshal(raw, &items); err != nil {
		return nil, err
	}
	kept := make([]json.RawMessage, 0, len(wanted))
	for _, i := range wanted {
		if i >= 0 && i < len(items) {
			kept = append(kept, items[i])
		}
	}
	out, err := json.Marshal(kept)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// maxInsightKeys is how many keys a tie shows before it is summarised.
//
// The cap exists because the big boards are enormous ties: the 2024 blue
// banner leaderboard puts about 450 team numbers on its first row, which is a
// 3,000-character cell that destroys the table around it. Ten keys is enough
// to see who is there, and the count says how much was left out.
const maxInsightKeys = 10

// joinInsightKeys renders a tie: the keys that reached one value, in one cell.
// A board about teams shows bare team numbers, since "177, 1073" is what a
// person reads a leaderboard for.
//
// expand is --expand: it prints every key, however many there are. JSON output
// is never summarised, since it is the full document the API sent.
func joinInsightKeys(keys []string, keyType string, expand bool) string {
	shown := keys
	hidden := 0
	if !expand && len(keys) > maxInsightKeys {
		shown = keys[:maxInsightKeys]
		hidden = len(keys) - maxInsightKeys
	}
	parts := make([]string, len(shown))
	for i, k := range shown {
		if keyType == "team" {
			parts[i] = output.TeamNumberFromKey(k)
			continue
		}
		parts[i] = k
	}
	joined := strings.Join(parts, ", ")
	if hidden == 0 {
		return joined
	}
	return fmt.Sprintf("%s … +%d more", joined, hidden)
}
