package cmd

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
	"github.com/the-blue-alliance/tba-cli/internal/api"
	"github.com/the-blue-alliance/tba-cli/internal/clierr"
	"github.com/the-blue-alliance/tba-cli/internal/output"
)

// fetchTeamPages walks /teams/{year}/{page} until the API runs out of teams or
// the page budget is spent. The returned cappedAt is 0 when the list ended on
// its own and the page limit when it did not, so the caller can say why it
// stopped without guessing.
//
// Every page is an ordinary cached GET, so a second walk of the same season
// costs a conditional request per page rather than a fresh download.
func fetchTeamPages(cmd *cobra.Command, client *api.Client, year, maxPages int) (teams []api.Team, cappedAt int, err error) {
	for page := 0; ; page++ {
		if maxPages > 0 && page >= maxPages {
			return teams, maxPages, nil
		}
		var pageTeams []api.Team
		path := fmt.Sprintf("/teams/%d/%d", year, page)
		if err := client.Get(cmd.Context(), path, &pageTeams); err != nil {
			return nil, 0, err
		}
		if len(pageTeams) == 0 {
			return teams, 0, nil
		}
		teams = append(teams, pageTeams...)
	}
}

// searchFieldNames lists the searchable fields, in the order they are shown in
// help and in error messages.
var searchFieldNames = []string{"nickname", "name", "location", "number"}

// validSearchFields is the help/error listing of accepted --fields values.
var validSearchFields = strings.Join(searchFieldNames, ", ")

// parseSearchFields turns a comma-separated --fields value into the set of
// fields to search. An empty value means every field.
func parseSearchFields(spec string) (map[string]bool, error) {
	fields := map[string]bool{}
	for _, raw := range strings.Split(spec, ",") {
		name := strings.ToLower(strings.TrimSpace(raw))
		if name == "" {
			continue
		}
		if !knownSearchField(name) {
			return nil, clierr.Usage("invalid --fields %q (want: %s)", raw, validSearchFields)
		}
		fields[name] = true
	}
	if len(fields) == 0 {
		return nil, clierr.Usage("--fields needs at least one of: %s", validSearchFields)
	}
	return fields, nil
}

// knownSearchField reports whether name is one of the searchable fields.
func knownSearchField(name string) bool {
	for _, f := range searchFieldNames {
		if f == name {
			return true
		}
	}
	return false
}

// isDigits reports whether s is a non-empty run of ASCII digits, which is what
// makes a query word a candidate team number.
func isDigits(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

// teamMatchesWord reports whether one lower-cased query word appears in any of
// the searched fields.
//
// Text fields match on substring, because people remember a fragment of a name
// far more often than the whole of it. The team number instead matches only on
// a prefix: "17" is how someone starts typing 177 or 1768, but a substring
// match would also drag in 517 and 1170, which nobody meant.
func teamMatchesWord(t api.Team, word string, fields map[string]bool) bool {
	if fields["nickname"] && strings.Contains(strings.ToLower(t.Nickname), word) {
		return true
	}
	if fields["name"] && strings.Contains(strings.ToLower(t.Name), word) {
		return true
	}
	if fields["location"] {
		location := output.FormatLocation(t.City, t.StateProv, t.Country)
		if strings.Contains(strings.ToLower(location), word) {
			return true
		}
	}
	if fields["number"] && isDigits(word) && strings.HasPrefix(strconv.Itoa(t.TeamNumber), word) {
		return true
	}
	return false
}

// teamMatchesQuery requires every word of the query to land somewhere. The
// words may match different fields — "bobcat windsor" finds the team whose
// nickname is Bobcat Robotics and whose city is South Windsor.
func teamMatchesQuery(t api.Team, words []string, fields map[string]bool) bool {
	for _, w := range words {
		if !teamMatchesWord(t, w, fields) {
			return false
		}
	}
	return true
}

// Search result ranks, best first.
const (
	rankNicknameExact = iota
	rankNicknamePrefix
	rankNicknameSubstring
	rankOtherField
)

// searchRank scores a match so that the team someone typed the name of comes
// first, ahead of the teams that merely mention it. The whole query is
// compared against the nickname, so "bobcat robotics" ranks Bobcat Robotics as
// an exact hit rather than as two separate word matches.
func searchRank(t api.Team, query string, fields map[string]bool) int {
	if !fields["nickname"] {
		return rankOtherField
	}
	nickname := strings.ToLower(t.Nickname)
	switch {
	case nickname == query:
		return rankNicknameExact
	case strings.HasPrefix(nickname, query):
		return rankNicknamePrefix
	case strings.Contains(nickname, query):
		return rankNicknameSubstring
	}
	return rankOtherField
}

// searchTeams filters and ranks a season's teams. Ties break on team number so
// that the same query always prints the same list.
func searchTeams(teams []api.Team, query string, fields map[string]bool) []api.Team {
	words := strings.Fields(query)
	matches := []api.Team{}
	for _, t := range teams {
		if teamMatchesQuery(t, words, fields) {
			matches = append(matches, t)
		}
	}
	ranks := make(map[string]int, len(matches))
	for _, t := range matches {
		ranks[t.Key] = searchRank(t, query, fields)
	}
	sort.SliceStable(matches, func(i, j int) bool {
		a, b := ranks[matches[i].Key], ranks[matches[j].Key]
		if a != b {
			return a < b
		}
		return matches[i].TeamNumber < matches[j].TeamNumber
	})
	return matches
}

func newTeamSearchCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "search <query>...",
		Short: "Search a season's teams by name, location or number",
		Long: `Search a season's teams by nickname, name, location or team number.

Matching is case-insensitive. Text fields match on a substring and the team
number matches on a prefix, so "17" finds 177 and 1768 but not 517. Several
words all have to match, though they may match different fields.

Results are ordered by how well they match: an exact nickname first, then a
nickname the query starts, then a nickname that contains it, then the teams
that matched on some other field, with team number breaking ties.

The search runs over the season's team list, which is about 20 pages of 500
teams. The first search of a season fetches them all; later searches revalidate
the cached pages, so they are cheap. A whole-history search is not offered
because it would repeat that walk for every season.`,
		Example: `  tba team search bobcat
  tba team search 17
  tba team search "south windsor" --fields location
  tba team search robotics --year 2024 --limit 0`,
		Args: cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			// Flags are checked before the client is built, so a typo is
			// reported without touching the network.
			if allYears, _ := cmd.Flags().GetBool("all-years"); allYears {
				return clierr.Usage("--all-years is not supported: every season is its own ~20-page team list, so searching all of them would mean hundreds of requests; search one season at a time with --year")
			}
			year, err := resolveYear(cmd)
			if err != nil {
				return err
			}
			fieldSpec, _ := cmd.Flags().GetString("fields")
			fields, err := parseSearchFields(fieldSpec)
			if err != nil {
				return err
			}
			limit, _ := cmd.Flags().GetInt("limit")
			if limit < 0 {
				return clierr.Usage("--limit %d is negative; use 0 for every match", limit)
			}
			format, err := resolveFormat(cmd)
			if err != nil {
				return err
			}

			client, err := newClient(cmd)
			if err != nil {
				return err
			}
			maxPages, _ := cmd.Flags().GetInt("max-pages")
			teams, cappedAt, err := fetchTeamPages(cmd, client, year, maxPages)
			if err != nil {
				return err
			}

			query := strings.Join(args, " ")
			matches := searchTeams(teams, strings.ToLower(query), fields)
			total := len(matches)
			truncated := limit > 0 && total > limit
			if truncated {
				matches = matches[:limit]
			}

			if err := printTeamSearch(cmd, format, matches); err != nil {
				return err
			}

			errOut := cmd.ErrOrStderr()
			if cappedAt > 0 {
				fmt.Fprintf(errOut, "note: stopped after %d pages; raise --max-pages to fetch more\n", cappedAt)
			}
			switch {
			case total == 0:
				fmt.Fprintf(errOut, "note: no teams match %q\n", query)
			case truncated:
				fmt.Fprintf(errOut, "note: showing %d of %d matches; use --limit 0 for all\n", limit, total)
			}
			return nil
		},
	}
	addYearFlag(c)
	c.Flags().Int("max-pages", 30, "Stop after this many pages of 500 teams")
	c.Flags().Int("limit", 20, "Show at most this many matches (0 for all)")
	c.Flags().String("fields", strings.Join(searchFieldNames, ","), "Fields to search: "+validSearchFields)
	// Offered only so that asking for it gets an explanation rather than
	// cobra's bare "unknown flag".
	c.Flags().Bool("all-years", false, "Not supported; search one season at a time")
	_ = c.Flags().MarkHidden("all-years")
	return c
}

// rookieYear renders a rookie year, leaving the cell empty when the API has no
// year for the team rather than printing a year 0 nobody competed in.
func rookieYear(year int) string {
	if year <= 0 {
		return ""
	}
	return strconv.Itoa(year)
}

// printTeamSearch renders the matches. No match prints nothing at all rather
// than a lone header row: a search that found nothing has no data to show, and
// the explanation belongs on stderr. JSON still prints its empty array, so a
// script can keep parsing stdout unconditionally.
func printTeamSearch(cmd *cobra.Command, format string, matches []api.Team) error {
	if len(matches) == 0 {
		if format == "json" {
			return output.PrintJSONWithFilter(cmd.OutOrStdout(), matches, jqExpr(cmd), rawOutput(cmd))
		}
		return nil
	}
	rows := make([][]string, len(matches))
	for i, t := range matches {
		rows[i] = []string{
			strconv.Itoa(t.TeamNumber),
			t.Nickname,
			output.FormatLocation(t.City, t.StateProv, t.Country),
			rookieYear(t.RookieYear),
		}
	}
	return outputTable(cmd, matches, []string{"Number", "Name", "Location", "Rookie"}, rows)
}
