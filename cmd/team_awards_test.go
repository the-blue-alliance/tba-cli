package cmd

import (
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/the-blue-alliance/tba-cli/internal/clierr"
)

// A team's awards arrive in no useful order and span its whole history.
const teamAwardsHistory177JSON = `[
  {
    "name": "District Event Winner",
    "award_type": 1,
    "event_key": "2024cthar",
    "year": 2024,
    "recipient_list": [{"team_key": "frc177", "awardee": null}]
  },
  {
    "name": "Regional Chairman's Award",
    "award_type": 0,
    "event_key": "2007ct",
    "year": 2007,
    "recipient_list": [{"team_key": "frc177", "awardee": null}]
  },
  {
    "name": "Woodie Flowers Finalist Award",
    "award_type": 3,
    "event_key": "2024necmp",
    "year": 2024,
    "recipient_list": [{"team_key": "frc177", "awardee": "Bill Beatty"}]
  },
  {
    "name": "Industrial Design Award sponsored by General Motors",
    "award_type": 9,
    "event_key": "2024cthar",
    "year": 2024,
    "recipient_list": [{"team_key": "frc177", "awardee": null}]
  },
  {
    "name": "Dean's List Finalist Award",
    "award_type": 4,
    "event_key": "2024necmp",
    "year": 2024,
    "recipient_list": [
      {"team_key": "frc177", "awardee": "Ada Lovelace"},
      {"team_key": "frc177", "awardee": "Grace Hopper"}
    ]
  }
]`

const event2007ctJSON = `{
  "key": "2007ct",
  "name": "Connecticut Regional",
  "event_code": "ct",
  "event_type": 0,
  "district": null,
  "city": "Hartford",
  "state_prov": "CT",
  "country": "USA",
  "start_date": "2007-03-29",
  "end_date": "2007-03-31",
  "year": 2007,
  "short_name": "Connecticut",
  "event_type_string": "Regional",
  "week": 4,
  "location_name": "Connecticut Convention Center",
  "webcasts": [],
  "playoff_type": null
}`

const teamAllEvents177JSON = "[" + event2007ctJSON + "," + event2024ctharJSON + "," + event2024necmpJSON + "]"

func teamAwardsServer(t *testing.T) *httptest.Server {
	t.Helper()
	return newFakeTBA(t, map[string]any{
		"/team/frc177/awards":      teamAwardsHistory177JSON,
		"/team/frc177/awards/2024": teamAwardsHistory177JSON,
		"/team/frc177/events":      teamAllEvents177JSON,
	})
}

func TestTeamAwardsColumnsOrderAndEventNames(t *testing.T) {
	srv := teamAwardsServer(t)
	out, stderr, err := runCmd(t, srv, "team", "awards", "177", "--format", "tsv")
	requireNoError(t, err, stderr)

	want := strings.Join([]string{
		"Year\tEvent\tAward\tType\tRecipient",
		"2024\tNE District Hartford Event\tDistrict Event Winner\tWinner\t",
		"2024\tNE District Hartford Event\tIndustrial Design Award sponsored by General Motors\tEngineering Inspiration\t",
		"2024\tNew England FIRST District Championship\tWoodie Flowers Finalist Award\tWoodie Flowers\tBill Beatty",
		"2024\tNew England FIRST District Championship\tDean's List Finalist Award\tDean's List\tAda Lovelace, Grace Hopper",
		"2007\tConnecticut Regional\tRegional Chairman's Award\tChairman's/Impact\t",
		"",
	}, "\n")
	if out != want {
		t.Errorf("tsv =\n%q\nwant\n%q", out, want)
	}
}

func TestTeamAwardsLooksUpEventNamesOnce(t *testing.T) {
	srv := teamAwardsServer(t)
	_, stderr, err := runCmd(t, srv, "team", "awards", "177")
	requireNoError(t, err, stderr)

	var lookups int
	for _, p := range requestPaths(t, srv) {
		if p == "/team/frc177/events" {
			lookups++
		}
	}
	if lookups != 1 {
		t.Errorf("made %d event-name lookups, want exactly 1", lookups)
	}
}

func TestTeamAwardsStillListsWhenEventLookupFails(t *testing.T) {
	srv := newFakeTBA(t, map[string]any{"/team/frc177/awards": teamAwardsHistory177JSON})
	// /team/frc177/events is not served, so the name lookup 404s.
	out, stderr, err := runCmd(t, srv, "team", "awards", "177", "--format", "tsv", "--retries", "0")
	requireNoError(t, err, stderr)
	requireContains(t, out, "2007\t\tRegional Chairman's Award\t")
	requireContains(t, out, "District Event Winner")
}

func TestTeamAwardsBlankNameForAnUnknownEvent(t *testing.T) {
	srv := newFakeTBA(t, map[string]any{
		"/team/frc177/awards": `[{"name":"Championship Winner","award_type":1,` +
			`"event_key":"2024cmptx","year":2024,"recipient_list":[{"team_key":"frc177","awardee":null}]}]`,
		"/team/frc177/events": teamAllEvents177JSON,
	})
	out, stderr, err := runCmd(t, srv, "team", "awards", "177", "--format", "tsv")
	requireNoError(t, err, stderr)
	requireContains(t, out, "2024\t\tChampionship Winner\t")
}

func TestTeamAwardsSkipsEventLookupWhenThereAreNoAwards(t *testing.T) {
	srv := newFakeTBA(t, map[string]any{
		"/team/frc9999/awards": `[]`,
		"/team/frc9999/events": `[]`,
	})
	out, stderr, err := runCmd(t, srv, "team", "awards", "9999", "--json")
	requireNoError(t, err, stderr)
	if strings.TrimSpace(out) != "[]" {
		t.Errorf("json = %q, want []", out)
	}
	if got := requestPaths(t, srv); len(got) != 1 || got[0] != "/team/frc9999/awards" {
		t.Errorf("requested %v, want only the awards path", got)
	}
}

func TestTeamAwardsTypeFilter(t *testing.T) {
	srv := teamAwardsServer(t)
	out, stderr, err := runCmd(t, srv, "team", "awards", "177", "--type", "4", "--format", "tsv")
	requireNoError(t, err, stderr)

	want := "Year\tEvent\tAward\tType\tRecipient\n" +
		"2024\tNew England FIRST District Championship\tDean's List Finalist Award\tDean's List\tAda Lovelace, Grace Hopper\n"
	if out != want {
		t.Errorf("tsv =\n%q\nwant\n%q", out, want)
	}
}

func TestTeamAwardsTypeZeroIsAFilterNotADefault(t *testing.T) {
	srv := teamAwardsServer(t)
	// award_type 0 is the Chairman's Award, so 0 has to mean "filter by 0"
	// rather than "no filter given".
	out, stderr, err := runCmd(t, srv, "team", "awards", "177", "--type", "0", "--format", "tsv")
	requireNoError(t, err, stderr)
	if n := len(lines(out)); n != 2 {
		t.Fatalf("want header + 1 row, got %d:\n%s", n, out)
	}
	requireContains(t, out, "Regional Chairman's Award")

	// Without --type nothing is filtered out.
	all, stderr, err := runCmd(t, srv, "team", "awards", "177", "--format", "tsv")
	requireNoError(t, err, stderr)
	if n := len(lines(all)); n != 6 {
		t.Errorf("want header + 5 rows without --type, got %d:\n%s", n, all)
	}
}

// --type takes the award's name, because nobody knows that 9 is Engineering
// Inspiration.
func TestTeamAwardsTypeAcceptsAName(t *testing.T) {
	cases := []struct {
		spec string
		want string
	}{
		{"impact", "Regional Chairman's Award"},
		{"IMPACT", "Regional Chairman's Award"},
		{"engineering inspiration", "Industrial Design Award sponsored by General Motors"},
		{"dean's list", "Dean's List Finalist Award"},
		{"woodie", "Woodie Flowers Finalist Award"},
	}
	for _, tc := range cases {
		t.Run(tc.spec, func(t *testing.T) {
			srv := teamAwardsServer(t)
			out, stderr, err := runCmd(t, srv, "team", "awards", "177", "--type", tc.spec, "--format", "tsv")
			requireNoError(t, err, stderr)
			if n := len(lines(out)); n != 2 {
				t.Fatalf("want header + 1 row, got %d:\n%s", n, out)
			}
			requireContains(t, out, tc.want)
		})
	}
}

// An exact name wins over the awards it is a part of, so --type winner is the
// event Winner award rather than an ambiguity with Skills Competition Winner.
func TestTeamAwardsTypeExactNameWinsOverSubstrings(t *testing.T) {
	srv := teamAwardsServer(t)
	out, stderr, err := runCmd(t, srv, "team", "awards", "177", "--type", "winner", "--format", "tsv")
	requireNoError(t, err, stderr)
	requireContains(t, out, "District Event Winner")
	if n := len(lines(out)); n != 2 {
		t.Fatalf("want header + 1 row, got %d:\n%s", n, out)
	}
}

func TestTeamAwardsTypeRejectsAnUnknownName(t *testing.T) {
	srv := teamAwardsServer(t)
	err := requireExitCode(t, clierr.ExitUsage, srv, "team", "awards", "177", "--type", "chairmanship")
	requireErrorContains(t, err, `unknown award type "chairmanship"`)
	requireErrorContains(t, err, "Chairman's")
	if got := requestPaths(t, srv); len(got) != 0 {
		t.Errorf("a usage error must not reach the API, got %v", got)
	}
}

func TestTeamAwardsTypeRejectsAnAmbiguousName(t *testing.T) {
	srv := teamAwardsServer(t)
	err := requireExitCode(t, clierr.ExitUsage, srv, "team", "awards", "177", "--type", "chairman")
	requireErrorContains(t, err, "matches")
	requireErrorContains(t, err, "Chairman's Finalist (70)")
}

func TestTeamAwardsTypeRejectsAnUnknownCode(t *testing.T) {
	srv := teamAwardsServer(t)
	err := requireExitCode(t, clierr.ExitUsage, srv, "team", "awards", "177", "--type", "900")
	requireErrorContains(t, err, "unknown award type 900")
}

func TestAwardTypeNamesCoverTheEnum(t *testing.T) {
	for i, tc := range awardTypes {
		if tc.code != i {
			t.Fatalf("award type %d is out of order: %+v", i, tc)
		}
		if tc.name == "" {
			t.Errorf("award type %d has no name", tc.code)
		}
	}
	if got := awardTypeName(0); got != "Chairman's/Impact" {
		t.Errorf("awardTypeName(0) = %q", got)
	}
	// A code TBA adds later still says something useful.
	if got := awardTypeName(999); got != "999" {
		t.Errorf("awardTypeName(999) = %q, want the bare code", got)
	}
}

func TestTeamAwardsTypeFilterMatchingNothing(t *testing.T) {
	srv := teamAwardsServer(t)
	out, stderr, err := runCmd(t, srv, "team", "awards", "177", "--type", "77", "--json")
	requireNoError(t, err, stderr)
	if strings.TrimSpace(out) != "[]" {
		t.Errorf("json = %q, want []", out)
	}
}

func TestTeamAwardsWithYearKeepsTheSameColumns(t *testing.T) {
	srv := teamAwardsServer(t)
	out, stderr, err := runCmd(t, srv, "team", "awards", "177", "--year", "2024", "--format", "tsv")
	requireNoError(t, err, stderr)

	if got := lines(out)[0]; got != "Year\tEvent\tAward\tType\tRecipient" {
		t.Errorf("header = %q", got)
	}
	requireContains(t, out, "NE District Hartford Event")
	if got := requestPaths(t, srv); !contains(got, "/team/frc177/awards/2024") {
		t.Errorf("requested %v, want the year-scoped awards path", got)
	}
}

func TestTeamAwardsJSONIsFilteredAndSorted(t *testing.T) {
	srv := teamAwardsServer(t)
	out, stderr, err := runCmd(t, srv, "team", "awards", "177", "--type", "1", "--json")
	requireNoError(t, err, stderr)

	arr := decodeJSON(t, out).([]any)
	if len(arr) != 1 {
		t.Fatalf("got %d awards, want 1:\n%s", len(arr), out)
	}
	if got := arr[0].(map[string]any)["event_key"]; got != "2024cthar" {
		t.Errorf("event_key = %v", got)
	}
}
