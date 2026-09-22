package cmd

import (
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/the-blue-alliance/tba-cli/internal/clierr"
)

// The search fixtures are built so that one query exercises every rank: 9999
// is an exact nickname, 50 and 177 are nicknames the query starts, 5000 has it
// further in, and 4000 matches only through its sponsor name. They are spread
// over two pages so that a search has to walk the list to see them all.
const (
	searchTeam9999JSON = `{
  "key": "frc9999", "team_number": 9999, "nickname": "Bobcat",
  "name": "Bobcat Boosters & Ames High School",
  "city": "Ames", "state_prov": "Iowa", "country": "USA", "rookie_year": 2018
}`
	searchTeam177JSON = `{
  "key": "frc177", "team_number": 177, "nickname": "Bobcat Robotics",
  "name": "Gund Foundation/RTX & South Windsor High School",
  "city": "South Windsor", "state_prov": "Connecticut", "country": "USA", "rookie_year": 1995
}`
	searchTeam1768JSON = `{
  "key": "frc1768", "team_number": 1768, "nickname": "Nashoba Robotics",
  "name": "Nashoba Regional High School",
  "city": "Bolton", "state_prov": "Massachusetts", "country": "USA", "rookie_year": 2006
}`
	searchTeam50JSON = `{
  "key": "frc50", "team_number": 50, "nickname": "Bobcat Racers",
  "name": "Bozeman High School",
  "city": "Bozeman", "state_prov": "Montana", "country": "USA", "rookie_year": 1997
}`
	searchTeam5000JSON = `{
  "key": "frc5000", "team_number": 5000, "nickname": "Red Bobcats",
  "name": "Hollis Brookline High School",
  "city": "Hollis", "state_prov": "New Hampshire", "country": "USA", "rookie_year": 2013
}`
	searchTeam4000JSON = `{
  "key": "frc4000", "team_number": 4000, "nickname": "Steel Hawks",
  "name": "Bobcat Industries & The Windsor Regional High School Booster Club",
  "city": "Windsor", "state_prov": "Connecticut", "country": "USA", "rookie_year": 2011
}`
	searchTeam517JSON = `{
  "key": "frc517", "team_number": 517, "nickname": "Gathering Storm",
  "name": "Rockford High School",
  "city": "Rockford", "state_prov": "Illinois", "country": "USA", "rookie_year": 2001
}`
)

// searchRoutes is a two-page season followed by the empty page that ends the
// walk.
func searchRoutes() map[string]any {
	return map[string]any{
		"/teams/2024/0": "[" + strings.Join([]string{searchTeam9999JSON, searchTeam177JSON, searchTeam1768JSON}, ",") + "]",
		"/teams/2024/1": "[" + strings.Join([]string{searchTeam50JSON, searchTeam5000JSON, searchTeam4000JSON, searchTeam517JSON}, ",") + "]",
		"/teams/2024/2": "[]",
	}
}

func newSearchServer(t *testing.T) *httptest.Server {
	t.Helper()
	return newFakeTBA(t, searchRoutes())
}

// otherNotes drops the "fetching the season's team list" line, which every run
// against a fresh cache directory prints and which the notes below are not
// about. The note has tests of its own.
func otherNotes(stderr string) string {
	var kept []string
	for _, line := range strings.Split(stderr, "\n") {
		if strings.HasPrefix(line, "note: fetching the ") {
			continue
		}
		kept = append(kept, line)
	}
	return strings.Join(kept, "\n")
}

// searchNumbers runs a search as CSV and returns the team numbers in order.
func searchNumbers(t *testing.T, srv *httptest.Server, args ...string) []string {
	t.Helper()
	full := append([]string{"team", "search"}, args...)
	full = append(full, "--year", "2024", "--format", "csv")
	out, errOut, err := runCmd(t, srv, full...)
	requireNoError(t, err, errOut)
	rows := lines(out)
	if len(rows) == 1 && rows[0] == "" {
		return nil
	}
	numbers := make([]string, 0, len(rows)-1)
	for _, row := range rows[1:] {
		numbers = append(numbers, strings.Split(row, ",")[0])
	}
	return numbers
}

func requireNumbers(t *testing.T, got []string, want ...string) {
	t.Helper()
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Errorf("team numbers = %v, want %v", got, want)
	}
}

func TestTeamSearchIsRegistered(t *testing.T) {
	if !contains(subcommandNames(t, "team"), "search") {
		t.Errorf("team search is not registered (have %v)", subcommandNames(t, "team"))
	}
}

func TestTeamSearchWalksEveryPage(t *testing.T) {
	srv := newSearchServer(t)
	out, errOut, err := runCmd(t, srv, "team", "search", "robotics", "--year", "2024", "--json")
	requireNoError(t, err, errOut)

	want := []string{"/teams/2024/0", "/teams/2024/1", "/teams/2024/2"}
	got := requestPaths(t, srv)
	if strings.Join(got, " ") != strings.Join(want, " ") {
		t.Errorf("requested %v, want %v", got, want)
	}
	arr, ok := decodeJSON(t, out).([]any)
	if !ok {
		t.Fatalf("want a JSON array, got %s", out)
	}
	// Bobcat Robotics and Nashoba Robotics, one from each page.
	if len(arr) != 2 {
		t.Errorf("matched %d teams, want 2", len(arr))
	}
}

func TestTeamSearchRanksExactThenPrefixThenSubstringThenOtherFields(t *testing.T) {
	srv := newSearchServer(t)
	// 9999 is exactly "Bobcat"; 50 and 177 start with it (50 first on number);
	// 5000 contains it; 4000 has it only in its sponsor name.
	requireNumbers(t, searchNumbers(t, srv, "bobcat"), "9999", "50", "177", "5000", "4000")
}

func TestTeamSearchColumns(t *testing.T) {
	srv := newSearchServer(t)
	out, errOut, err := runCmd(t, srv, "team", "search", "bobcat", "--year", "2024", "--format", "csv", "--limit", "1")
	requireNoError(t, err, errOut)
	rows := lines(out)
	if rows[0] != "Number,Name,Location,Rookie,Matched" {
		t.Errorf("header = %q", rows[0])
	}
	if rows[1] != "9999,Bobcat,\"Ames, Iowa, USA\",2018,nickname: Bobcat" {
		t.Errorf("row = %q", rows[1])
	}
}

func TestTeamSearchRequiresEveryWordToMatch(t *testing.T) {
	srv := newSearchServer(t)
	// "bobcat" from the nickname, "connecticut" from the location for 177;
	// 4000 matches "bobcat" in its name and "connecticut" in its location.
	requireNumbers(t, searchNumbers(t, srv, "bobcat", "connecticut"), "177", "4000")
	// Both words have to land somewhere, and no team has these two.
	requireNumbers(t, searchNumbers(t, srv, "bobcat", "nashoba"))
}

func TestTeamSearchQuotedQueryIsOneString(t *testing.T) {
	srv := newSearchServer(t)
	requireNumbers(t, searchNumbers(t, srv, "south windsor"), "177")
}

func TestTeamSearchMatchesTeamNumbersOnlyByPrefix(t *testing.T) {
	srv := newSearchServer(t)
	// 177 and 1768 start with 17; 517 merely contains it.
	requireNumbers(t, searchNumbers(t, srv, "17"), "177", "1768")
	requireNumbers(t, searchNumbers(t, srv, "5000"), "5000")
}

func TestTeamSearchIsCaseInsensitive(t *testing.T) {
	srv := newSearchServer(t)
	requireNumbers(t, searchNumbers(t, srv, "BOBCAT", "IOWA"), "9999")
}

func TestTeamSearchLimitDefaultsTo20(t *testing.T) {
	c := newTeamSearchCmd()
	n, err := c.Flags().GetInt("limit")
	if err != nil {
		t.Fatalf("limit flag: %v", err)
	}
	if n != 20 {
		t.Errorf("--limit default = %d, want 20", n)
	}
	if pages, _ := c.Flags().GetInt("max-pages"); pages != 30 {
		t.Errorf("--max-pages default = %d, want 30", pages)
	}
}

func TestTeamSearchLimitTruncatesAndSaysSo(t *testing.T) {
	srv := newSearchServer(t)
	out, errOut, err := runCmd(t, srv, "team", "search", "bobcat", "--year", "2024", "--limit", "2", "--json")
	requireNoError(t, err, errOut)

	arr, ok := decodeJSON(t, out).([]any)
	if !ok || len(arr) != 2 {
		t.Fatalf("want 2 teams on stdout, got %s", out)
	}
	if otherNotes(errOut) != "note: showing 2 of 5 matches; use --limit 0 for all\n" {
		t.Errorf("stderr = %q", errOut)
	}
	if strings.Contains(out, "note:") {
		t.Errorf("the truncation note leaked onto stdout:\n%s", out)
	}
}

func TestTeamSearchLimitZeroShowsEverything(t *testing.T) {
	srv := newSearchServer(t)
	out, errOut, err := runCmd(t, srv, "team", "search", "bobcat", "--year", "2024", "--limit", "0", "--json")
	requireNoError(t, err, errOut)
	arr, _ := decodeJSON(t, out).([]any)
	if len(arr) != 5 {
		t.Errorf("want all 5 matches, got %d", len(arr))
	}
	if otherNotes(errOut) != "" {
		t.Errorf("stderr = %q, want nothing but the fetch note", errOut)
	}
}

func TestTeamSearchSaysNothingWhenNothingIsTruncated(t *testing.T) {
	srv := newSearchServer(t)
	_, errOut, err := runCmd(t, srv, "team", "search", "bobcat", "--year", "2024")
	requireNoError(t, err, errOut)
	if otherNotes(errOut) != "" {
		t.Errorf("stderr = %q, want nothing when the default limit is not reached", errOut)
	}
}

func TestTeamSearchWithNoMatchesExitsZero(t *testing.T) {
	srv := newSearchServer(t)
	out, errOut, err := runCmd(t, srv, "team", "search", "nosuchteam", "--year", "2024", "--json")
	requireNoError(t, err, errOut)
	if clierr.ExitCode(err) != clierr.ExitOK {
		t.Errorf("exit code = %d, want 0", clierr.ExitCode(err))
	}
	if strings.TrimSpace(out) != "[]" {
		t.Errorf("stdout = %q, want an empty JSON array", out)
	}
	if otherNotes(errOut) != "note: no teams match \"nosuchteam\"\n" {
		t.Errorf("stderr = %q", errOut)
	}
}

func TestTeamSearchWithNoMatchesPrintsNoTableAtAll(t *testing.T) {
	srv := newSearchServer(t)
	out, errOut, err := runCmd(t, srv, "team", "search", "nosuchteam", "--year", "2024", "--format", "table")
	requireNoError(t, err, errOut)
	if out != "" {
		t.Errorf("stdout = %q, want nothing (not a lone header row)", out)
	}
	requireContains(t, errOut, `note: no teams match "nosuchteam"`)
}

func TestTeamSearchFieldsRestrictTheSearch(t *testing.T) {
	srv := newSearchServer(t)
	// "windsor" is in two locations and two sponsor names, but no nickname.
	requireNumbers(t, searchNumbers(t, srv, "windsor", "--fields", "location"), "177", "4000")
	requireNumbers(t, searchNumbers(t, srv, "windsor", "--fields", "nickname"))
	requireNumbers(t, searchNumbers(t, srv, "bobcat", "--fields", "nickname"), "9999", "50", "177", "5000")
	requireNumbers(t, searchNumbers(t, srv, "bobcat", "--fields", "name"), "4000", "9999")
	requireNumbers(t, searchNumbers(t, srv, "17", "--fields", "number"), "177", "1768")
	requireNumbers(t, searchNumbers(t, srv, "17", "--fields", "nickname,name,location"))
}

func TestTeamSearchRejectsAnUnknownField(t *testing.T) {
	srv := newSearchServer(t)
	err := requireExitCode(t, clierr.ExitUsage, srv, "team", "search", "bobcat", "--year", "2024", "--fields", "nickname,sponsor")
	requireErrorContains(t, err, `invalid --fields "sponsor"`)
	requireErrorContains(t, err, "nickname, name, location, number")
	if len(requestPaths(t, srv)) != 0 {
		t.Errorf("a bad --fields should not reach the network, got %v", requestPaths(t, srv))
	}
}

func TestTeamSearchStopsAtMaxPagesAndSaysSo(t *testing.T) {
	srv := newSearchServer(t)
	out, errOut, err := runCmd(t, srv, "team", "search", "bobcat", "--year", "2024", "--max-pages", "1", "--json")
	requireNoError(t, err, errOut)

	if got := requestPaths(t, srv); len(got) != 1 || got[0] != "/teams/2024/0" {
		t.Errorf("requested %v, want only page 0", got)
	}
	arr, _ := decodeJSON(t, out).([]any)
	if len(arr) != 2 {
		t.Errorf("matched %d teams from page 0, want 2", len(arr))
	}
	requireContains(t, errOut, "note: stopped after 1 pages; raise --max-pages to fetch more")
}

func TestTeamSearchReusesTheCacheOnASecondRun(t *testing.T) {
	t.Setenv("TBA_CACHE_DIR", t.TempDir())
	srv := newSearchServer(t)
	for _, path := range []string{"/teams/2024/0", "/teams/2024/1", "/teams/2024/2"} {
		setETag(t, srv, path, `"etag-`+path+`"`)
	}

	first, errOut, err := runCmd(t, srv, "team", "search", "bobcat", "--year", "2024", "--json")
	requireNoError(t, err, errOut)
	second, errOut, err := runCmd(t, srv, "team", "search", "bobcat", "--year", "2024", "--json")
	requireNoError(t, err, errOut)

	if first != second {
		t.Errorf("revalidated search differs:\n%s\n---\n%s", first, second)
	}
	reqs := requestsTo(t, srv)
	if len(reqs) != 6 {
		t.Fatalf("want 6 requests (3 pages twice), got %d", len(reqs))
	}
	for i, r := range reqs[:3] {
		if got := r.Headers.Get("If-None-Match"); got != "" {
			t.Errorf("first run request %d sent If-None-Match %q", i, got)
		}
	}
	for i, r := range reqs[3:] {
		if got := r.Headers.Get("If-None-Match"); got != `"etag-`+r.Path+`"` {
			t.Errorf("second run request %d If-None-Match = %q", i, got)
		}
	}
}

func TestTeamSearchRejectsWholeHistorySearches(t *testing.T) {
	srv := newSearchServer(t)
	for _, tc := range []struct {
		args []string
		want string
	}{
		{[]string{"team", "search", "bobcat", "--year", "0"}, "season"},
		{[]string{"team", "search", "bobcat", "--all-years"}, "requests"},
	} {
		args := tc.args
		t.Run(strings.Join(args[3:], " "), func(t *testing.T) {
			err := requireExitCode(t, clierr.ExitUsage, srv, args...)
			requireErrorContains(t, err, tc.want)
			if len(requestPaths(t, srv)) != 0 {
				t.Errorf("nothing should be fetched, got %v", requestPaths(t, srv))
			}
		})
	}
}

func TestTeamSearchNeedsAQuery(t *testing.T) {
	srv := newSearchServer(t)
	_ = requireExitCode(t, clierr.ExitUsage, srv, "team", "search")
}

func TestTeamSearchHelpShowsExamplesAndFlags(t *testing.T) {
	out, _, err := runCmd(t, nil, "team", "search", "--help")
	requireNoError(t, err, "")
	requireContains(t, out, "tba team search bobcat")
	requireContains(t, out, "--limit int")
	requireContains(t, out, "(default 20)")
	requireContains(t, out, "Fields to search: nickname, name, location, number")
	requireContains(t, out, "--max-pages int")
}

func TestTeamSearchSupportsColumnsAndSort(t *testing.T) {
	srv := newSearchServer(t)
	out, errOut, err := runCmd(t, srv, "team", "search", "bobcat", "--year", "2024",
		"--format", "csv", "--columns", "number,rookie", "--sort", "rookie")
	requireNoError(t, err, errOut)
	want := []string{"Number,Rookie", "177,1995", "50,1997", "4000,2011", "5000,2013", "9999,2018"}
	if strings.Join(lines(out), "|") != strings.Join(want, "|") {
		t.Errorf("output = %q, want %q", lines(out), want)
	}
}

// A team can match on its full name, which no other column shows, so a search
// for "bobcat" looks like it answered with strangers. Matched says where the
// query landed and what it says there.
func TestTeamSearchExplainsWhyEachTeamMatched(t *testing.T) {
	srv := newSearchServer(t)
	out, errOut, err := runCmd(t, srv, "team", "search", "bobcat", "--year", "2024",
		"--format", "csv", "--columns", "number,matched", "--no-headers")
	requireNoError(t, err, errOut)

	want := []string{
		"9999,nickname: Bobcat",
		"50,nickname: Bobcat Racers",
		"177,nickname: Bobcat Robotics",
		"5000,nickname: Red Bobcats",
		// The one whose nickname says nothing about bobcats, and the long
		// name it did match on, cut to fit.
		`4000,name: Bobcat Industries & The Windsor Regional High School…`,
	}
	got := lines(out)
	if len(got) != len(want) {
		t.Fatalf("got %d rows, want %d:\n%s", len(got), len(want), out)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("row %d = %q, want %q", i, got[i], want[i])
		}
	}
	for _, row := range got {
		if n := len([]rune(strings.SplitN(row, ",", 2)[1])); n > 60 {
			t.Errorf("Matched cell is %d characters: %q", n, row)
		}
	}
}

// A word that lands in the location is credited to the location.
func TestTeamSearchExplainsALocationMatch(t *testing.T) {
	srv := newSearchServer(t)
	out, errOut, err := runCmd(t, srv, "team", "search", "montana", "--year", "2024",
		"--format", "csv", "--columns", "number,matched", "--no-headers")
	requireNoError(t, err, errOut)
	if got := lines(out)[0]; got != `50,"location: Bozeman, Montana, USA"` {
		t.Errorf("row = %q", got)
	}
}

// The field that accounts for the most of the query is the one credited, so a
// nickname hit is never explained by a sponsor.
func TestTeamSearchCreditsTheFieldThatExplainsTheMostWords(t *testing.T) {
	srv := newSearchServer(t)
	out, errOut, err := runCmd(t, srv, "team", "search", "bobcat", "robotics", "--year", "2024",
		"--format", "csv", "--columns", "number,matched", "--no-headers")
	requireNoError(t, err, errOut)
	if got := lines(out)[0]; got != "177,nickname: Bobcat Robotics" {
		t.Errorf("row = %q", got)
	}
}

// The first search of a season is twenty requests and a wait, so it says so
// before the wait. A season already on disk is quick and says nothing.
func TestTeamSearchNotesTheFirstWalkOfASeason(t *testing.T) {
	t.Setenv("TBA_CACHE_DIR", t.TempDir())
	srv := newSearchServer(t)

	_, errOut, err := runCmd(t, srv, "team", "search", "bobcat", "--year", "2024", "--format", "csv")
	requireNoError(t, err, errOut)
	requireContains(t, errOut, "note: fetching the 2024 team list (about 20 pages, cached for next time)")

	_, errOut, err = runCmd(t, srv, "team", "search", "bobcat", "--year", "2024", "--format", "csv")
	requireNoError(t, err, errOut)
	if strings.Contains(errOut, "fetching the 2024 team list") {
		t.Errorf("the note came back for a cached season: %q", errOut)
	}
}

// A season whose pages are only half cached is still a fetch.
func TestTeamSearchNotesAPartiallyCachedSeason(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("TBA_CACHE_DIR", dir)
	srv := newSearchServer(t)

	// One page short of the walk: --max-pages 1 caches page 0 and nothing else.
	_, errOut, err := runCmd(t, srv, "team", "search", "bobcat", "--year", "2024",
		"--max-pages", "1", "--format", "csv")
	requireNoError(t, err, errOut)

	_, errOut, err = runCmd(t, srv, "team", "search", "bobcat", "--year", "2024", "--format", "csv")
	requireNoError(t, err, errOut)
	requireContains(t, errOut, "note: fetching the 2024 team list")
}

// The note is about the wait, so a run that cannot fetch at all is silent.
func TestTeamSearchOfflineDoesNotAnnounceAFetch(t *testing.T) {
	t.Setenv("TBA_CACHE_DIR", t.TempDir())
	srv := newSearchServer(t)
	_, errOut, _ := runCmd(t, srv, "team", "search", "bobcat", "--year", "2024", "--offline")
	if strings.Contains(errOut, "fetching the 2024 team list") {
		t.Errorf("offline announced a fetch it will never make: %q", errOut)
	}
}
