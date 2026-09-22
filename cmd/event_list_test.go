package cmd

import (
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/the-blue-alliance/tba-cli/internal/clierr"
)

// eventListServer serves the mixed 2024 season list plus 177's schedule.
func eventListServer(t *testing.T) *httptest.Server {
	t.Helper()
	return newFakeTBA(t, map[string]any{
		"/events/2024":             events2024JSON,
		"/team/frc177/events/2024": teamEvents177JSON,
	})
}

// listedKeys runs `event list` and returns the key column of each row, which
// is what every filter test asserts on.
func listedKeys(t *testing.T, srv *httptest.Server, args ...string) []string {
	t.Helper()
	full := append([]string{"event", "list", "--year", "2024", "--format", "tsv"}, args...)
	out, stderr, err := runCmd(t, srv, full...)
	requireNoError(t, err, stderr)
	rows := lines(out)
	if len(rows) == 0 || rows[0] == "" {
		return nil
	}
	keys := make([]string, 0, len(rows)-1)
	for _, r := range rows[1:] { // row 0 is the header
		if r == "" {
			continue
		}
		keys = append(keys, strings.SplitN(r, "\t", 2)[0])
	}
	return keys
}

func requireKeys(t *testing.T, got []string, want ...string) {
	t.Helper()
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Errorf("keys = %v, want %v", got, want)
	}
}

func TestEventListColumns(t *testing.T) {
	srv := eventListServer(t)
	out, stderr, err := runCmd(t, srv, "event", "list", "--year", "2024", "--format", "tsv")
	requireNoError(t, err, stderr)

	header := lines(out)[0]
	want := "Key\tName\tType\tWeek\tStart\tEnd\tLocation\tDistrict"
	if header != want {
		t.Errorf("header = %q, want %q", header, want)
	}
}

func TestEventListRowRendersWeekDistrictAndDates(t *testing.T) {
	srv := eventListServer(t)
	out, stderr, err := runCmd(t, srv, "event", "list", "--year", "2024", "--format", "tsv")
	requireNoError(t, err, stderr)

	var row string
	for _, l := range lines(out) {
		if strings.HasPrefix(l, "2024cthar\t") {
			row = l
		}
	}
	// TBA reports week 3 for Hartford; people call that week 4.
	want := "2024cthar\tNE District Hartford Event\tDistrict\t4\t2024-03-22\t2024-03-24\tHartford, CT, USA\tne"
	if row != want {
		t.Errorf("row =\n%q\nwant\n%q", row, want)
	}
}

func TestEventListSortsByStartDateThenKey(t *testing.T) {
	srv := eventListServer(t)
	// The fixture arrives out of order; the command puts it in calendar order.
	requireKeys(t, listedKeys(t, srv),
		"2024isde3", "2024cthar", "2024casj", "2024necmp", "2024mil", "2024iri")
}

func TestEventListWeekIsOneBased(t *testing.T) {
	srv := eventListServer(t)
	// ISR #3 carries TBA week 1, so the human week is 2.
	requireKeys(t, listedKeys(t, srv, "--week", "2"), "2024isde3")
	// Hartford and Silicon Valley both carry TBA week 3.
	requireKeys(t, listedKeys(t, srv, "--week", "4"), "2024cthar", "2024casj")
	// Nothing sits in week 3, and an empty result is not an error.
	requireKeys(t, listedKeys(t, srv, "--week", "3"))
}

func TestEventListWeekSkipsEventsWithoutAWeek(t *testing.T) {
	srv := eventListServer(t)
	for _, key := range listedKeys(t, srv, "--week", "7") {
		if key == "2024mil" || key == "2024iri" {
			t.Errorf("weekless event %s matched --week 7", key)
		}
	}
	requireKeys(t, listedKeys(t, srv, "--week", "7"), "2024necmp")
}

func TestEventListWeekBlankWhenMissing(t *testing.T) {
	srv := eventListServer(t)
	out, stderr, err := runCmd(t, srv, "event", "list", "--year", "2024", "--format", "csv")
	requireNoError(t, err, stderr)
	requireContains(t, out, "2024mil,Milstein Division,Championship Division,,2024-04-17")
}

func TestEventListTypeFilter(t *testing.T) {
	srv := eventListServer(t)
	cases := []struct {
		spec string
		want []string
	}{
		{"regional", []string{"2024casj"}},
		{"district", []string{"2024isde3", "2024cthar"}},
		{"dcmp", []string{"2024necmp"}},
		{"cmp-division", []string{"2024mil"}},
		{"offseason", []string{"2024iri"}},
		{"preseason", nil},
		{"remote", nil},
	}
	for _, tc := range cases {
		t.Run(tc.spec, func(t *testing.T) {
			requireKeys(t, listedKeys(t, srv, "--type", tc.spec), tc.want...)
		})
	}
}

func TestEventListTypeAcceptsMultipleValues(t *testing.T) {
	srv := eventListServer(t)
	requireKeys(t, listedKeys(t, srv, "--type", "dcmp,cmp-division"), "2024necmp", "2024mil")
	// Whitespace and case around each value are forgiven.
	requireKeys(t, listedKeys(t, srv, "--type", "Regional, OFFSEASON"), "2024casj", "2024iri")
}

func TestEventListTypeAllKeepsEverything(t *testing.T) {
	srv := eventListServer(t)
	all := listedKeys(t, srv)
	requireKeys(t, listedKeys(t, srv, "--type", "all"), all...)
	// "all" wins however it is combined, rather than narrowing the result.
	requireKeys(t, listedKeys(t, srv, "--type", "regional,all"), all...)
}

func TestEventListRemoteType(t *testing.T) {
	srv := newFakeTBA(t, map[string]any{
		"/events/2021": "[" + event2021nhflaJSON + "]",
	})
	out, stderr, err := runCmd(t, srv, "event", "list", "--year", "2021", "--type", "remote", "--format", "tsv")
	requireNoError(t, err, stderr)
	requireContains(t, out, "2021nhfla")
	requireContains(t, out, "Remote")
}

func TestEventListRejectsUnknownType(t *testing.T) {
	srv := eventListServer(t)
	err := requireExitCode(t, clierr.ExitUsage, srv, "event", "list", "--year", "2024", "--type", "champs")
	requireErrorContains(t, err, `invalid --type "champs"`)
	for _, name := range []string{"regional", "district", "dcmp", "dcmp-division", "cmp-division", "cmp-finals", "offseason", "preseason", "remote", "all"} {
		requireErrorContains(t, err, name)
	}
}

func TestEventListRejectsUnknownTypeAmongGoodOnes(t *testing.T) {
	srv := eventListServer(t)
	_ = requireExitCode(t, clierr.ExitUsage, srv, "event", "list", "--year", "2024", "--type", "district,nope")
}

func TestEventListRejectsAWeekBelowOne(t *testing.T) {
	// Week 0 is the flag's "no filter" default, so asking for it explicitly
	// used to list the whole season although the help says weeks start at 1.
	for _, week := range []string{"-1", "0"} {
		t.Run("week "+week, func(t *testing.T) {
			srv := eventListServer(t)
			err := requireExitCode(t, clierr.ExitUsage, srv, "event", "list", "--year", "2024", "--week", week)
			requireErrorContains(t, err, "numbered from 1")
			if got := requestPaths(t, srv); len(got) != 0 {
				t.Errorf("a usage error must not reach the API, got %v", got)
			}
		})
	}
}

// Without --week the whole season is listed, which is what week 0 used to do.
func TestEventListWithoutWeekListsEverything(t *testing.T) {
	srv := eventListServer(t)
	if got := listedKeys(t, srv); len(got) < 2 {
		t.Errorf("want the whole season, got %v", got)
	}
}

func TestEventListRejectsAYearBeforeTheFirstSeason(t *testing.T) {
	srv := eventListServer(t)
	err := requireExitCode(t, clierr.ExitUsage, srv, "event", "list", "--year", "1800")
	requireErrorContains(t, err, "--year 1800 is before the first FRC season (1992)")
	if got := requestPaths(t, srv); len(got) != 0 {
		t.Errorf("a usage error must not reach the API, got %v", got)
	}
}

func TestEventListDistrictFilterIgnoresCase(t *testing.T) {
	srv := eventListServer(t)
	requireKeys(t, listedKeys(t, srv, "--district", "ne"), "2024cthar")
	requireKeys(t, listedKeys(t, srv, "--district", "NE"), "2024cthar")
	requireKeys(t, listedKeys(t, srv, "--district", "isr"), "2024isde3")
	requireKeys(t, listedKeys(t, srv, "--district", "fim"))
}

func TestEventListStateFilterIgnoresCase(t *testing.T) {
	srv := eventListServer(t)
	requireKeys(t, listedKeys(t, srv, "--state", "CT"), "2024cthar")
	requireKeys(t, listedKeys(t, srv, "--state", "ct"), "2024cthar")
	requireKeys(t, listedKeys(t, srv, "--state", "TX"), "2024mil")
}

func TestEventListCountryFilterIgnoresCase(t *testing.T) {
	srv := eventListServer(t)
	requireKeys(t, listedKeys(t, srv, "--country", "Israel"), "2024isde3")
	requireKeys(t, listedKeys(t, srv, "--country", "israel"), "2024isde3")
	requireKeys(t, listedKeys(t, srv, "--country", "usa"),
		"2024cthar", "2024casj", "2024necmp", "2024mil", "2024iri")
}

func TestEventListTeamFilterSwitchesEndpoint(t *testing.T) {
	srv := eventListServer(t)
	requireKeys(t, listedKeys(t, srv, "--team", "177"), "2024cthar", "2024necmp")
	if got := requestPaths(t, srv); len(got) != 1 || got[0] != "/team/frc177/events/2024" {
		t.Errorf("requested %v, want [/team/frc177/events/2024]", got)
	}
}

func TestEventListTeamFilterAcceptsFrcPrefix(t *testing.T) {
	srv := eventListServer(t)
	requireKeys(t, listedKeys(t, srv, "--team", "frc177"), "2024cthar", "2024necmp")
}

func TestEventListFiltersCombine(t *testing.T) {
	srv := eventListServer(t)
	requireKeys(t, listedKeys(t, srv, "--type", "district", "--country", "USA"), "2024cthar")
	requireKeys(t, listedKeys(t, srv, "--type", "district", "--week", "4"), "2024cthar")
	requireKeys(t, listedKeys(t, srv, "--district", "ne", "--state", "MA"))
	requireKeys(t, listedKeys(t, srv, "--team", "177", "--type", "dcmp"), "2024necmp")
	requireKeys(t, listedKeys(t, srv,
		"--type", "district,dcmp,regional", "--country", "USA", "--state", "CT", "--district", "ne"),
		"2024cthar")
}

func TestEventListFiltersAreClientSide(t *testing.T) {
	srv := eventListServer(t)
	_ = listedKeys(t, srv, "--week", "4", "--type", "district", "--state", "CT")
	reqs := requestsTo(t, srv)
	if len(reqs) != 1 {
		t.Fatalf("made %d requests, want 1", len(reqs))
	}
	if reqs[0].Path != "/events/2024" || reqs[0].Query != "" {
		t.Errorf("requested %q?%q, want /events/2024 with no query", reqs[0].Path, reqs[0].Query)
	}
}

func TestEventListJSONIsFilteredAndSorted(t *testing.T) {
	srv := eventListServer(t)
	out, stderr, err := runCmd(t, srv, "event", "list", "--year", "2024", "--type", "district", "--json")
	requireNoError(t, err, stderr)

	arr, ok := decodeJSON(t, out).([]any)
	if !ok {
		t.Fatalf("output is not a JSON array:\n%s", out)
	}
	if len(arr) != 2 {
		t.Fatalf("got %d events, want 2:\n%s", len(arr), out)
	}
	first := arr[0].(map[string]any)
	if first["key"] != "2024isde3" {
		t.Errorf("first key = %v, want 2024isde3", first["key"])
	}
	// The JSON stays the API's own shape: week is still TBA's 0-based value.
	if first["week"] != float64(1) {
		t.Errorf("week = %v, want the raw 1", first["week"])
	}
}

func TestEventListEmptyResultIsNotAnError(t *testing.T) {
	srv := eventListServer(t)
	out, stderr, err := runCmd(t, srv, "event", "list", "--year", "2024", "--type", "preseason", "--json")
	requireNoError(t, err, stderr)
	if strings.TrimSpace(out) != "[]" {
		t.Errorf("json = %q, want []", out)
	}
}

func TestEventListFiltersWorkWithColumnSelection(t *testing.T) {
	srv := eventListServer(t)
	out, stderr, err := runCmd(t, srv,
		"event", "list", "--year", "2024", "--district", "ne", "--columns", "key,week,district", "--format", "csv")
	requireNoError(t, err, stderr)
	if out != "Key,Week,District\n2024cthar,4,ne\n" {
		t.Errorf("csv = %q", out)
	}
}
