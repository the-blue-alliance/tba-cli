package cmd

import (
	"fmt"
	"strings"
	"testing"

	"github.com/the-blue-alliance/tba-cli/internal/clierr"
)

func TestInsightLeaderboards(t *testing.T) {
	srv := newFakeTBA(t, map[string]any{
		"/insights/leaderboards/2024": leaderboards2024JSON,
	})
	out, _, err := runCmd(t, srv, "insight", "leaderboards", "--year", "2024")
	requireNoError(t, err, "")

	arr := decodeJSON(t, out).([]any)
	first := arr[0].(map[string]any)
	if first["name"] != "typed_leaderboard_blue_banners" {
		t.Errorf("name = %v", first["name"])
	}
}

func TestInsightLeaderboardsTable(t *testing.T) {
	srv := newFakeTBA(t, map[string]any{
		"/insights/leaderboards/2024": leaderboards2024JSON,
	})
	out, _, err := runCmd(t, srv, "insight", "leaderboards", "--year", "2024", "--format", "table")
	requireNoError(t, err, "")

	got := lines(out)
	if !strings.HasPrefix(got[0], "Leaderboard") {
		t.Fatalf("header = %q", got[0])
	}
	requireContains(t, out, "Blue Banners")
	requireContains(t, out, "Most Matches Played")
	requireContains(t, out, "Highest Median Score By Event")
	// A tie lists every key that reached the value in one cell, and a team
	// board shows bare numbers.
	requireContains(t, out, "1073, 230")
	// An event board keeps its keys as they are.
	requireContains(t, out, "2024necmp")
	// A whole value prints without the decimals a fixed format would add.
	requireContains(t, out, "112.5")
	if strings.Contains(out, "6.00") {
		t.Errorf("values should not carry trailing zeros:\n%s", out)
	}
}

func TestInsightLeaderboardsCSVColumns(t *testing.T) {
	srv := newFakeTBA(t, map[string]any{
		"/insights/leaderboards/2024": leaderboards2024JSON,
	})
	out, _, err := runCmd(t, srv, "insight", "leaderboards", "--year", "2024",
		"--board", "blue banners", "--format", "csv")
	requireNoError(t, err, "")
	want := []string{
		"Leaderboard,Rank,Key,Value",
		"Blue Banners,1,177,6",
		`Blue Banners,2,"1073, 230",5`,
		"Blue Banners,3,5507,4",
	}
	got := lines(out)
	if len(got) != len(want) {
		t.Fatalf("got %d lines, want %d:\n%s", len(got), len(want), out)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("line %d = %q, want %q", i, got[i], want[i])
		}
	}
}

// --limit keeps a season of boards readable; 0 turns the cap off.
func TestInsightLeaderboardsLimit(t *testing.T) {
	cases := []struct {
		args []string
		want int
	}{
		{nil, 10},
		{[]string{"--limit", "3"}, 3},
		{[]string{"--limit", "0"}, 12},
		{[]string{"--limit", "50"}, 12},
	}
	for _, tc := range cases {
		t.Run(strings.Join(append([]string{"default"}, tc.args...), " "), func(t *testing.T) {
			srv := newFakeTBA(t, map[string]any{
				"/insights/leaderboards/2024": leaderboards2024JSON,
			})
			args := append([]string{"insight", "leaderboards", "--year", "2024",
				"--board", "Most Matches Played", "--format", "csv", "--no-headers"}, tc.args...)
			out, _, err := runCmd(t, srv, args...)
			requireNoError(t, err, "")
			if got := len(lines(out)); got != tc.want {
				t.Errorf("got %d rows, want %d:\n%s", got, tc.want, out)
			}
		})
	}
}

func TestInsightLeaderboardsRejectsANegativeLimit(t *testing.T) {
	srv := newFakeTBA(t, map[string]any{"/insights/leaderboards/2024": leaderboards2024JSON})
	err := requireExitCode(t, clierr.ExitUsage, srv,
		"insight", "leaderboards", "--year", "2024", "--limit", "-1")
	requireErrorContains(t, err, "--limit cannot be negative")
}

// --board takes either spelling of a board's name, in any case.
func TestInsightLeaderboardsBoardSpellings(t *testing.T) {
	for _, board := range []string{
		"Blue Banners", "blue banners", "BLUE BANNERS",
		"blue_banners", "typed_leaderboard_blue_banners",
	} {
		t.Run(board, func(t *testing.T) {
			srv := newFakeTBA(t, map[string]any{
				"/insights/leaderboards/2024": leaderboards2024JSON,
			})
			out, _, err := runCmd(t, srv, "insight", "leaderboards", "--year", "2024",
				"--board", board, "--format", "table")
			requireNoError(t, err, "")
			requireContains(t, out, "Blue Banners")
			if strings.Contains(out, "Most Matches Played") {
				t.Errorf("--board should have filtered the other boards out:\n%s", out)
			}
		})
	}
}

func TestInsightLeaderboardsUnknownBoardListsTheAvailableOnes(t *testing.T) {
	srv := newFakeTBA(t, map[string]any{"/insights/leaderboards/2024": leaderboards2024JSON})
	err := requireExitCode(t, clierr.ExitUsage, srv,
		"insight", "leaderboards", "--year", "2024", "--board", "banners")
	requireErrorContains(t, err, `unknown --board "banners"`)
	requireErrorContains(t, err, "Blue Banners")
	requireErrorContains(t, err, "Most Matches Played")
}

// --board narrows the JSON too, so a filtered run and a piped one agree.
func TestInsightLeaderboardsBoardFiltersJSON(t *testing.T) {
	srv := newFakeTBA(t, map[string]any{"/insights/leaderboards/2024": leaderboards2024JSON})
	out, _, err := runCmd(t, srv, "insight", "leaderboards", "--year", "2024",
		"--board", "Blue Banners", "--json")
	requireNoError(t, err, "")
	arr := decodeJSON(t, out).([]any)
	if len(arr) != 1 {
		t.Fatalf("got %d boards, want 1:\n%s", len(arr), out)
	}
	if arr[0].(map[string]any)["name"] != "typed_leaderboard_blue_banners" {
		t.Errorf("wrong board:\n%s", out)
	}
}

// An unfiltered run passes the body through exactly as the API sent it.
func TestInsightLeaderboardsJSONIsTheRawArray(t *testing.T) {
	srv := newFakeTBA(t, map[string]any{"/insights/leaderboards/2024": leaderboards2024JSON})
	out, _, err := runCmd(t, srv, "insight", "leaderboards", "--year", "2024", "--json")
	requireNoError(t, err, "")
	arr := decodeJSON(t, out).([]any)
	if len(arr) != 3 {
		t.Fatalf("got %d boards, want 3:\n%s", len(arr), out)
	}
	// --limit is presentation: it never truncates the data.
	first := arr[1].(map[string]any)["data"].(map[string]any)["rankings"].([]any)
	if len(first) != 12 {
		t.Errorf("--limit should not have trimmed the JSON: got %d rankings", len(first))
	}
}

func TestInsightLeaderboardsJq(t *testing.T) {
	srv := newFakeTBA(t, map[string]any{
		"/insights/leaderboards/2024": leaderboards2024JSON,
	})
	out, _, err := runCmd(t, srv, "insight", "leaderboards", "--year", "2024",
		"--jq", ".[0].data.rankings[0].keys[0]")
	requireNoError(t, err, "")
	if strings.TrimSpace(out) != `"frc177"` {
		t.Errorf("jq output = %q", out)
	}
}

func TestInsightLeaderboardsOfAnEmptySeason(t *testing.T) {
	srv := newFakeTBA(t, map[string]any{"/insights/leaderboards/1998": "[]"})
	out, _, err := runCmd(t, srv, "insight", "leaderboards", "--year", "1998", "--format", "table")
	requireNoError(t, err, "")
	// Nothing at all: a season TBA has no boards for is an empty listing, not
	// an error -- and not a row of column names with nothing under it either.
	if out != "" {
		t.Errorf("output = %q, want nothing", out)
	}
}

// Regression test: `insight leaderboards` used to fail without --year.
func TestInsightLeaderboardsDefaultsToCurrentYear(t *testing.T) {
	path := fmt.Sprintf("/insights/leaderboards/%d", thisYear())
	srv := newFakeTBA(t, map[string]any{path: "[]"})
	_, _, err := runCmd(t, srv, "insight", "leaderboards")
	requireNoError(t, err, "")
	// The season lookup comes first; this fake serves no /status, so the year
	// falls back to the calendar.
	if got := requestPaths(t, srv); !contains(got, path) {
		t.Errorf("requested %v, want %s among them", got, path)
	}
}

func TestInsightNotables(t *testing.T) {
	srv := newFakeTBA(t, map[string]any{
		"/insights/notables/2024": notables2024JSON,
	})
	out, _, err := runCmd(t, srv, "insight", "notables", "--year", "2024", "--json")
	requireNoError(t, err, "")

	arr := decodeJSON(t, out).([]any)
	first := arr[0].(map[string]any)
	if first["name"] != "notables_hall_of_fame" {
		t.Errorf("name = %v", first["name"])
	}
	if first["year"] != float64(2024) {
		t.Errorf("year = %v", first["year"])
	}
}

func TestInsightNotablesTable(t *testing.T) {
	srv := newFakeTBA(t, map[string]any{"/insights/notables/2024": notables2024JSON})
	out, _, err := runCmd(t, srv, "insight", "notables", "--year", "2024", "--format", "csv")
	requireNoError(t, err, "")
	want := []string{
		"Notable,Team,Context",
		"Hall Of Fame,177,2007ct",
		`World Champions,254,"2024cmptx, 2018cmptx"`,
		"World Champions,1323,2024cmptx",
	}
	got := lines(out)
	if len(got) != len(want) {
		t.Fatalf("got %d lines, want %d:\n%s", len(got), len(want), out)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("line %d = %q, want %q", i, got[i], want[i])
		}
	}
}

func TestInsightNotablesBoardFilter(t *testing.T) {
	srv := newFakeTBA(t, map[string]any{"/insights/notables/2024": notables2024JSON})
	out, _, err := runCmd(t, srv, "insight", "notables", "--year", "2024",
		"--board", "world champions", "--format", "table")
	requireNoError(t, err, "")
	requireContains(t, out, "World Champions")
	if strings.Contains(out, "Hall Of Fame") {
		t.Errorf("--board should have filtered the other board out:\n%s", out)
	}
}

func TestInsightNotablesUnknownBoard(t *testing.T) {
	srv := newFakeTBA(t, map[string]any{"/insights/notables/2024": notables2024JSON})
	err := requireExitCode(t, clierr.ExitUsage, srv,
		"insight", "notables", "--year", "2024", "--board", "einstein")
	requireErrorContains(t, err, `unknown --board "einstein"`)
	requireErrorContains(t, err, "Hall Of Fame")
}

func TestInsightNotablesBoardFiltersJSON(t *testing.T) {
	srv := newFakeTBA(t, map[string]any{"/insights/notables/2024": notables2024JSON})
	out, _, err := runCmd(t, srv, "insight", "notables", "--year", "2024",
		"--board", "Hall Of Fame", "--json")
	requireNoError(t, err, "")
	arr := decodeJSON(t, out).([]any)
	if len(arr) != 1 || arr[0].(map[string]any)["name"] != "notables_hall_of_fame" {
		t.Errorf("want just the hall of fame board:\n%s", out)
	}
}

// Regression test: `insight notables` used to fail without --year.
func TestInsightNotablesDefaultsToCurrentYear(t *testing.T) {
	path := fmt.Sprintf("/insights/notables/%d", thisYear())
	srv := newFakeTBA(t, map[string]any{path: "[]"})
	_, _, err := runCmd(t, srv, "insight", "notables")
	requireNoError(t, err, "")
	// The season lookup comes first; this fake serves no /status, so the year
	// falls back to the calendar.
	if got := requestPaths(t, srv); !contains(got, path) {
		t.Errorf("requested %v, want %s among them", got, path)
	}
}

func TestInsightNotablesJq(t *testing.T) {
	srv := newFakeTBA(t, map[string]any{
		"/insights/notables/2024": notables2024JSON,
	})
	out, _, err := runCmd(t, srv, "insight", "notables", "--year", "2024",
		"--jq", ".[0].data.entries[0].team_key")
	requireNoError(t, err, "")
	if strings.TrimSpace(out) != `"frc177"` {
		t.Errorf("jq output = %q", out)
	}
}

func TestInsightSubcommandsAreRegistered(t *testing.T) {
	got := subcommandNames(t, "insight")
	for _, want := range []string{"leaderboards", "notables"} {
		if !contains(got, want) {
			t.Errorf("insight %s is not registered (have %v)", want, got)
		}
	}
}

// Every tabular format renders the same table, and JSON keeps the API's array.
func TestInsightLeaderboardsHonorsEveryFormat(t *testing.T) {
	for _, format := range []string{"csv", "tsv", "markdown", "table"} {
		t.Run(format, func(t *testing.T) {
			srv := newFakeTBA(t, map[string]any{
				"/insights/leaderboards/2024": leaderboards2024JSON,
			})
			out, _, err := runCmd(t, srv, "insight", "leaderboards", "--year", "2024", "--format", format)
			requireNoError(t, err, "")
			requireContains(t, out, "Leaderboard")
			requireContains(t, out, "Blue Banners")
		})
	}
}

func TestInsightNotablesHonorsFormatJSON(t *testing.T) {
	srv := newFakeTBA(t, map[string]any{"/insights/notables/2024": notables2024JSON})
	out, _, err := runCmd(t, srv, "insight", "notables", "--year", "2024", "--format", "json")
	requireNoError(t, err, "")
	if _, ok := decodeJSON(t, out).([]any); !ok {
		t.Fatalf("want a JSON array, got:\n%s", out)
	}
}

func TestInsightCommandsCarryExamples(t *testing.T) {
	for _, c := range NewRootCmd().Commands() {
		if c.Name() != "insight" {
			continue
		}
		for _, sub := range c.Commands() {
			if strings.TrimSpace(sub.Example) == "" {
				t.Errorf("insight %s has no Example block", sub.Name())
			}
		}
	}
}

// The real blue banner board ties hundreds of teams on one value. The cell has
// to stay readable: the first few keys, then a count of the rest.
func TestInsightLeaderboardsCapABigTie(t *testing.T) {
	srv := newFakeTBA(t, map[string]any{
		"/insights/leaderboards/2024": leaderboardsBigTie2024JSON,
	})
	out, _, err := runCmd(t, srv, "insight", "leaderboards", "--year", "2024",
		"--format", "csv", "--columns", "key", "--no-headers")
	requireNoError(t, err, "")

	got := lines(out)
	want := `"177, 254, 1114, 118, 2056, 971, 1678, 2767, 1323, 180 … +4 more"`
	if got[0] != want {
		t.Errorf("key cell = %s, want %s", got[0], want)
	}
	// A tie that fits is untouched.
	if got[1] != `"1073, 230"` {
		t.Errorf("short tie = %s", got[1])
	}
}

func TestInsightLeaderboardsExpandShowsEveryKey(t *testing.T) {
	srv := newFakeTBA(t, map[string]any{
		"/insights/leaderboards/2024": leaderboardsBigTie2024JSON,
	})
	out, _, err := runCmd(t, srv, "insight", "leaderboards", "--year", "2024",
		"--expand", "--format", "csv", "--columns", "key", "--no-headers")
	requireNoError(t, err, "")

	got := lines(out)[0]
	if strings.Contains(got, "more") {
		t.Errorf("--expand still summarised the tie: %s", got)
	}
	for _, team := range []string{"177", "217", "2168"} {
		if !strings.Contains(got, team) {
			t.Errorf("team %s missing from %s", team, got)
		}
	}
	if n := strings.Count(got, ",") + 1; n != 14 {
		t.Errorf("got %d keys, want 14: %s", n, got)
	}
}

// The cap is presentation. JSON is the document the API sent.
func TestInsightLeaderboardsJSONKeepsEveryKey(t *testing.T) {
	srv := newFakeTBA(t, map[string]any{
		"/insights/leaderboards/2024": leaderboardsBigTie2024JSON,
	})
	out, _, err := runCmd(t, srv, "insight", "leaderboards", "--year", "2024", "--json")
	requireNoError(t, err, "")

	board := decodeJSON(t, out).([]any)[0].(map[string]any)
	rankings := board["data"].(map[string]any)["rankings"].([]any)
	keys := rankings[0].(map[string]any)["keys"].([]any)
	if len(keys) != 14 {
		t.Errorf("json carries %d keys, want 14", len(keys))
	}
}
