package cmd

import (
	"fmt"
	"strings"
	"testing"
)

func TestEventViewTable(t *testing.T) {
	srv := newFakeTBA(t, map[string]any{"/event/2024cthar": event2024ctharJSON})
	out, _, err := runCmd(t, srv, "event", "view", "2024cthar", "--format", "table")
	requireNoError(t, err, "")

	want := "Event:     NE District Hartford Event\n" +
		"Key:       2024cthar\n" +
		"Type:      District\n" +
		"Location:  Hartford, CT, USA\n" +
		"Venue:     Hartford Public High School\n" +
		"Dates:     2024-03-22 to 2024-03-24\n" +
		"Week:      3\n"
	if out != want {
		t.Errorf("event view table =\n%q\nwant\n%q", out, want)
	}
}

func TestEventViewWeekNotAvailable(t *testing.T) {
	srv := newFakeTBA(t, map[string]any{
		"/event/2024cmptx": `{"key":"2024cmptx","name":"Einstein Field","event_type_string":"Championship Finals",` +
			`"city":"Houston","state_prov":"TX","country":"USA","location_name":"George R. Brown Convention Center",` +
			`"start_date":"2024-04-17","end_date":"2024-04-20","week":null}`,
	})
	out, _, err := runCmd(t, srv, "event", "view", "2024cmptx", "--format", "table")
	requireNoError(t, err, "")
	requireContains(t, out, "Week:      N/A")
}

func TestEventViewJSONRoundTrips(t *testing.T) {
	srv := newFakeTBA(t, map[string]any{"/event/2024cthar": event2024ctharJSON})
	out, _, err := runCmd(t, srv, "event", "view", "2024cthar", "--json")
	requireNoError(t, err, "")

	obj := decodeJSON(t, out).(map[string]any)
	if obj["key"] != "2024cthar" {
		t.Errorf("key = %v", obj["key"])
	}
	if obj["name"] != "NE District Hartford Event" {
		t.Errorf("name = %v", obj["name"])
	}
	if obj["week"] != float64(3) {
		t.Errorf("week = %v", obj["week"])
	}
}

func TestEventViewJq(t *testing.T) {
	srv := newFakeTBA(t, map[string]any{"/event/2024cthar": event2024ctharJSON})
	out, _, err := runCmd(t, srv, "event", "view", "2024cthar", "--jq", ".district.display_name")
	requireNoError(t, err, "")
	if strings.TrimSpace(out) != `"New England"` {
		t.Errorf("jq output = %q", out)
	}
}

func TestEventList(t *testing.T) {
	srv := newFakeTBA(t, map[string]any{
		"/events/2024": "[" + event2024ctharJSON + "," + event2024necmpJSON + "]",
	})
	out, _, err := runCmd(t, srv, "event", "list", "--year", "2024", "--format", "table")
	requireNoError(t, err, "")

	got := lines(out)
	if len(got) != 4 {
		t.Fatalf("want header + separator + 2 rows, got %d:\n%s", len(got), out)
	}
	for _, want := range []string{"Key", "Name", "Start", "Type", "Location"} {
		requireContains(t, got[0], want)
	}
	requireContains(t, got[2], "2024cthar")
	requireContains(t, got[2], "District")
	requireContains(t, got[3], "District Championship")
}

// Regression test: `event list` used to fail without an explicit --year.
func TestEventListDefaultsToCurrentYear(t *testing.T) {
	path := fmt.Sprintf("/events/%d", currentYear())
	srv := newFakeTBA(t, map[string]any{path: "[]"})
	_, _, err := runCmd(t, srv, "event", "list")
	requireNoError(t, err, "")
	if got := requestPaths(t, srv); len(got) != 1 || got[0] != path {
		t.Errorf("requested %v, want [%s]", got, path)
	}
}

func TestEventListCSVQuotesLocations(t *testing.T) {
	srv := newFakeTBA(t, map[string]any{"/events/2024": "[" + event2024ctharJSON + "]"})
	out, _, err := runCmd(t, srv, "event", "list", "--year", "2024", "--format", "csv")
	requireNoError(t, err, "")

	want := "Key,Name,Start,Type,Location\n" +
		"2024cthar,NE District Hartford Event,2024-03-22,District,\"Hartford, CT, USA\"\n"
	if out != want {
		t.Errorf("csv =\n%q\nwant\n%q", out, want)
	}
}

func TestEventTeams(t *testing.T) {
	srv := newFakeTBA(t, map[string]any{
		"/event/2024cthar/teams": "[" + teamFRC177JSON + "," + teamFRC1073JSON + "]",
	})
	out, _, err := runCmd(t, srv, "event", "teams", "2024cthar", "--format", "table")
	requireNoError(t, err, "")

	got := lines(out)
	if got[0] != "Number  Name             Location                       " {
		t.Errorf("header = %q", got[0])
	}
	requireContains(t, got[2], "177")
	requireContains(t, got[2], "Bobcat Robotics")
	requireContains(t, got[3], "1073")
}

func TestEventMatches(t *testing.T) {
	srv := newFakeTBA(t, map[string]any{
		"/event/2024cthar/matches": "[" + match2024ctharQM1JSON + "," + match2024ctharQM2JSON + "]",
	})
	out, _, err := runCmd(t, srv, "event", "matches", "2024cthar", "--format", "table")
	requireNoError(t, err, "")

	got := lines(out)
	if got[0] != "Key            Level  Red  Blue  Winner" {
		t.Errorf("header = %q", got[0])
	}
	if got[2] != "2024cthar_qm1  qm     88   61    red   " {
		t.Errorf("row 1 = %q", got[2])
	}
	if got[3] != "2024cthar_qm2  qm     45   72    blue  " {
		t.Errorf("row 2 = %q", got[3])
	}
}

func TestEventMatchesJSONRoundTrips(t *testing.T) {
	srv := newFakeTBA(t, map[string]any{
		"/event/2024cthar/matches": "[" + match2024ctharQM1JSON + "]",
	})
	out, _, err := runCmd(t, srv, "event", "matches", "2024cthar", "--json")
	requireNoError(t, err, "")

	arr := decodeJSON(t, out).([]any)
	m := arr[0].(map[string]any)
	alliances := m["alliances"].(map[string]any)
	red := alliances["red"].(map[string]any)
	if red["score"] != float64(88) {
		t.Errorf("red score = %v", red["score"])
	}
	keys := red["team_keys"].([]any)
	if len(keys) != 3 || keys[0] != "frc177" {
		t.Errorf("red team_keys = %v", keys)
	}
}

func TestEventRankings(t *testing.T) {
	srv := newFakeTBA(t, map[string]any{"/event/2024cthar/rankings": rankings2024ctharJSON})
	out, _, err := runCmd(t, srv, "event", "rankings", "2024cthar", "--format", "table")
	requireNoError(t, err, "")

	got := lines(out)
	if got[0] != "Rank  Team  Record  Played" {
		t.Errorf("header = %q", got[0])
	}
	if got[2] != "1     177   10-2-0  12    " {
		t.Errorf("row 1 = %q", got[2])
	}
	if got[3] != "2     1073  9-3-0   12    " {
		t.Errorf("row 2 = %q", got[3])
	}
}

func TestEventRankingsJSONIsTheWrapperObject(t *testing.T) {
	srv := newFakeTBA(t, map[string]any{"/event/2024cthar/rankings": rankings2024ctharJSON})
	out, _, err := runCmd(t, srv, "event", "rankings", "2024cthar", "--json")
	requireNoError(t, err, "")

	obj := decodeJSON(t, out).(map[string]any)
	if _, ok := obj["rankings"]; !ok {
		t.Errorf("want a rankings key, got %v", obj)
	}
	if _, ok := obj["sort_order_info"]; !ok {
		t.Errorf("want a sort_order_info key, got %v", obj)
	}
}

func TestEventAlliances(t *testing.T) {
	srv := newFakeTBA(t, map[string]any{"/event/2024cthar/alliances": alliances2024ctharJSON})
	out, _, err := runCmd(t, srv, "event", "alliances", "2024cthar", "--format", "markdown")
	requireNoError(t, err, "")

	want := "| Alliance | Picks |\n" +
		"| --- | --- |\n" +
		"| Alliance 1 | 177, 1073, 5507 |\n" +
		"| Alliance 2 | 230, 195, 558 |\n"
	if out != want {
		t.Errorf("markdown =\n%q\nwant\n%q", out, want)
	}
}

func TestEventAwards(t *testing.T) {
	srv := newFakeTBA(t, map[string]any{"/event/2024cthar/awards": awards2024ctharJSON})
	out, _, err := runCmd(t, srv, "event", "awards", "2024cthar", "--format", "csv")
	requireNoError(t, err, "")

	want := "Award,Recipient\n" +
		"District Event Winner,177\n" +
		"Dean's List Finalist Award,1073 - Ada Lovelace\n"
	if out != want {
		t.Errorf("csv =\n%q\nwant\n%q", out, want)
	}
}

func TestEventOPRsAreSortedByTeamNumber(t *testing.T) {
	srv := newFakeTBA(t, map[string]any{"/event/2024cthar/oprs": oprs2024ctharJSON})
	out, _, err := runCmd(t, srv, "event", "oprs", "2024cthar", "--format", "csv")
	requireNoError(t, err, "")

	want := "Team,OPR,DPR,CCWM\n" +
		"177,55.43,20.11,35.32\n" +
		"1073,41.12,25.67,15.46\n" +
		"5507,30.50,28.25,2.25\n"
	if out != want {
		t.Errorf("csv =\n%q\nwant\n%q", out, want)
	}
}

func TestEventOPRsJq(t *testing.T) {
	srv := newFakeTBA(t, map[string]any{"/event/2024cthar/oprs": oprs2024ctharJSON})
	out, _, err := runCmd(t, srv, "event", "oprs", "2024cthar", "--jq", ".oprs.frc177")
	requireNoError(t, err, "")
	if strings.TrimSpace(out) != "55.4321" {
		t.Errorf("jq output = %q", out)
	}
}

func TestEventDistrictPointsRawTable(t *testing.T) {
	srv := newFakeTBA(t, map[string]any{
		"/event/2024cthar/district_points": districtPoints2024ctharJSON,
	})
	out, _, err := runCmd(t, srv, "event", "district-points", "2024cthar", "--format", "table")
	requireNoError(t, err, "")
	requireContains(t, out, `"qual_points": 22`)
	decodeJSON(t, out)
}

func TestEventDistrictPointsJSONRoundTrips(t *testing.T) {
	srv := newFakeTBA(t, map[string]any{
		"/event/2024cthar/district_points": districtPoints2024ctharJSON,
	})
	out, _, err := runCmd(t, srv, "event", "district-points", "2024cthar", "--json")
	requireNoError(t, err, "")

	obj := decodeJSON(t, out).(map[string]any)
	points := obj["points"].(map[string]any)["frc177"].(map[string]any)
	if points["total"] != float64(73) {
		t.Errorf("total = %v", points["total"])
	}
}

func TestEventPredictionsJq(t *testing.T) {
	srv := newFakeTBA(t, map[string]any{
		"/event/2024cthar/predictions": predictions2024ctharJSON,
	})
	out, _, err := runCmd(t, srv, "event", "predictions", "2024cthar",
		"--jq", ".match_predictions.qual.\"2024cthar_qm1\".red.score")
	requireNoError(t, err, "")
	if strings.TrimSpace(out) != "84.2" {
		t.Errorf("jq output = %q", out)
	}
}

func TestEventInsights(t *testing.T) {
	srv := newFakeTBA(t, map[string]any{
		"/event/2024cthar/insights": insights2024ctharJSON,
	})
	out, _, err := runCmd(t, srv, "event", "insights", "2024cthar")
	requireNoError(t, err, "")

	obj := decodeJSON(t, out).(map[string]any)
	qual := obj["qual"].(map[string]any)
	if qual["average_score"] != 61.4 {
		t.Errorf("average_score = %v", qual["average_score"])
	}
}

func TestEventSubcommandsAreRegistered(t *testing.T) {
	got := subcommandNames(t, "event")
	for _, want := range []string{
		"view", "list", "teams", "matches", "rankings", "alliances",
		"awards", "oprs", "district-points", "predictions", "insights",
	} {
		if !contains(got, want) {
			t.Errorf("event %s is not registered (have %v)", want, got)
		}
	}
}
