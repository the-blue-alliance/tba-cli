package cmd

import (
	"strings"
	"testing"
)

func TestEventViewShowsWeekOneBased(t *testing.T) {
	srv := newFakeTBA(t, map[string]any{"/event/2024isde3": event2024isde3JSON})
	out, stderr, err := runCmd(t, srv, "event", "view", "2024isde3", "--format", "table")
	requireNoError(t, err, stderr)
	// The API reports week 1; the event runs in what everyone calls week 2.
	requireContains(t, out, "Week:      2")
}

func TestEventViewShowsDistrict(t *testing.T) {
	srv := newFakeTBA(t, map[string]any{"/event/2024isde3": event2024isde3JSON})
	out, stderr, err := runCmd(t, srv, "event", "view", "2024isde3", "--format", "table")
	requireNoError(t, err, stderr)
	requireContains(t, out, "District:  FIRST Israel (isr)")
}

func TestEventViewOmitsDistrictForNonDistrictEvents(t *testing.T) {
	srv := newFakeTBA(t, map[string]any{"/event/2024casj": event2024casjJSON})
	out, stderr, err := runCmd(t, srv, "event", "view", "2024casj", "--format", "table")
	requireNoError(t, err, stderr)
	if strings.Contains(out, "District:") {
		t.Errorf("a regional should have no District line:\n%s", out)
	}
}

func TestEventViewShowsWebsiteTimezoneAndFirstCode(t *testing.T) {
	srv := newFakeTBA(t, map[string]any{"/event/2024casj": event2024casjJSON})
	out, stderr, err := runCmd(t, srv, "event", "view", "2024casj", "--format", "table")
	requireNoError(t, err, stderr)
	requireContains(t, out, "Website:     https://cafirst.org/frc/siliconvalley/")
	requireContains(t, out, "Timezone:    America/Los_Angeles")
	requireContains(t, out, "First Code:  CASJ")
}

func TestEventViewOmitsAbsentOptionalFields(t *testing.T) {
	srv := newFakeTBA(t, map[string]any{"/event/2024cthar": event2024ctharJSON})
	out, stderr, err := runCmd(t, srv, "event", "view", "2024cthar", "--format", "table")
	requireNoError(t, err, stderr)
	for _, label := range []string{"Website:", "Timezone:", "First Code:"} {
		if strings.Contains(out, label) {
			t.Errorf("%s should be omitted when the API does not supply it:\n%s", label, out)
		}
	}
}

func TestEventViewWebcastURLs(t *testing.T) {
	srv := newFakeTBA(t, map[string]any{
		"/event/2024multi": `{"key":"2024multi","name":"Streamy Event","event_type_string":"Offseason",` +
			`"city":"Hartford","state_prov":"CT","country":"USA","location_name":"Somewhere",` +
			`"start_date":"2024-09-07","end_date":"2024-09-07","week":null,"playoff_type":null,` +
			`"webcasts":[{"type":"twitch","channel":"nefirst_red"},` +
			`{"type":"youtube","channel":"dQw4w9WgXcQ"},` +
			`{"type":"livestream","channel":"12345"}]}`,
	})
	out, stderr, err := runCmd(t, srv, "event", "view", "2024multi", "--format", "table")
	requireNoError(t, err, stderr)

	requireContains(t, out, "Webcast:   https://www.twitch.tv/nefirst_red")
	requireContains(t, out, "Webcast:   https://www.youtube.com/watch?v=dQw4w9WgXcQ")
	requireContains(t, out, "Webcast:   livestream: 12345")
	if n := strings.Count(out, "Webcast:"); n != 3 {
		t.Errorf("got %d webcast lines, want one per webcast", n)
	}
}

func TestEventViewOmitsWebcastsWhenThereAreNone(t *testing.T) {
	srv := newFakeTBA(t, map[string]any{"/event/2024necmp": event2024necmpJSON})
	out, stderr, err := runCmd(t, srv, "event", "view", "2024necmp", "--format", "table")
	requireNoError(t, err, stderr)
	if strings.Contains(out, "Webcast") {
		t.Errorf("no webcasts means no Webcast line:\n%s", out)
	}
}

func TestEventViewPlayoffTypeNames(t *testing.T) {
	cases := []struct {
		playoff string
		want    string
	}{
		{"0", "Bracket (8 alliances)"},
		{"2", "Bracket (4 alliances)"},
		{"3", "Average score (8 alliances)"},
		{"4", "Round robin (6 alliances)"},
		{"5", "Legacy double elimination (8 alliances)"},
		{"10", "Double elimination (8 alliances)"},
		{"11", "Double elimination (4 alliances)"},
		{"42", "Unknown (42)"},
	}
	for _, tc := range cases {
		t.Run(tc.playoff, func(t *testing.T) {
			srv := newFakeTBA(t, map[string]any{
				"/event/2024play": `{"key":"2024play","name":"Playoff Shapes","event_type_string":"Offseason",` +
					`"city":"Hartford","state_prov":"CT","country":"USA","location_name":"Somewhere",` +
					`"start_date":"2024-09-07","end_date":"2024-09-07","week":null,"webcasts":[],` +
					`"playoff_type":` + tc.playoff + `}`,
			})
			out, stderr, err := runCmd(t, srv, "event", "view", "2024play", "--format", "table")
			requireNoError(t, err, stderr)
			requireContains(t, out, "Playoff:   "+tc.want)
		})
	}
}

func TestEventViewOmitsPlayoffTypeWhenNull(t *testing.T) {
	srv := newFakeTBA(t, map[string]any{"/event/2021nhfla": event2021nhflaJSON})
	out, stderr, err := runCmd(t, srv, "event", "view", "2021nhfla", "--format", "table")
	requireNoError(t, err, stderr)
	if strings.Contains(out, "Playoff:") {
		t.Errorf("a null playoff_type should print no Playoff line:\n%s", out)
	}
}

func TestEventViewJSONKeepsRawShape(t *testing.T) {
	srv := newFakeTBA(t, map[string]any{"/event/2024casj": event2024casjJSON})
	out, stderr, err := runCmd(t, srv, "event", "view", "2024casj", "--json")
	requireNoError(t, err, stderr)

	obj := decodeJSON(t, out).(map[string]any)
	// The human view renders week 4; JSON still carries the API's own 3.
	if obj["week"] != float64(3) {
		t.Errorf("week = %v, want the raw 3", obj["week"])
	}
	if obj["first_event_code"] != "CASJ" {
		t.Errorf("first_event_code = %v", obj["first_event_code"])
	}
	if obj["timezone"] != "America/Los_Angeles" {
		t.Errorf("timezone = %v", obj["timezone"])
	}
}
