package cmd

import (
	"net/http/httptest"
	"strings"
	"testing"
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
		"Year\tEvent\tAward\tRecipient",
		"2024\tNE District Hartford Event\tDistrict Event Winner\t",
		"2024\tNE District Hartford Event\tIndustrial Design Award sponsored by General Motors\t",
		"2024\tNew England FIRST District Championship\tWoodie Flowers Finalist Award\tBill Beatty",
		"2024\tNew England FIRST District Championship\tDean's List Finalist Award\tAda Lovelace, Grace Hopper",
		"2007\tConnecticut Regional\tRegional Chairman's Award\t",
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

	want := "Year\tEvent\tAward\tRecipient\n" +
		"2024\tNew England FIRST District Championship\tDean's List Finalist Award\tAda Lovelace, Grace Hopper\n"
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

	if got := lines(out)[0]; got != "Year\tEvent\tAward\tRecipient" {
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
