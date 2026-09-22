package cmd

import (
	"fmt"
	"strings"
	"testing"

	"github.com/the-blue-alliance/tba-cli/internal/clierr"
)

func TestTeamViewTable(t *testing.T) {
	srv := newFakeTBA(t, map[string]any{"/team/frc177": teamFRC177JSON})
	out, _, err := runCmd(t, srv, "team", "view", "177", "--format", "table")
	requireNoError(t, err, "")

	want := "Team:         177 - Bobcat Robotics\n" +
		"Name:         Gordon & Llura Gund Foundation/RTX & South Windsor High School\n" +
		"Location:     South Windsor, Connecticut, USA\n" +
		"Rookie Year:  1995\n" +
		"Website:      http://www.bobcatrobotics.org\n"
	if out != want {
		t.Errorf("team view table =\n%q\nwant\n%q", out, want)
	}
}

func TestTeamViewJSONRoundTrips(t *testing.T) {
	srv := newFakeTBA(t, map[string]any{"/team/frc177": teamFRC177JSON})
	out, _, err := runCmd(t, srv, "team", "view", "177", "--json")
	requireNoError(t, err, "")

	obj := decodeJSON(t, out).(map[string]any)
	if obj["key"] != "frc177" {
		t.Errorf("key = %v", obj["key"])
	}
	if obj["team_number"] != float64(177) {
		t.Errorf("team_number = %v", obj["team_number"])
	}
	if obj["nickname"] != "Bobcat Robotics" {
		t.Errorf("nickname = %v", obj["nickname"])
	}
	if obj["rookie_year"] != float64(1995) {
		t.Errorf("rookie_year = %v", obj["rookie_year"])
	}
}

func TestTeamViewJq(t *testing.T) {
	srv := newFakeTBA(t, map[string]any{"/team/frc177": teamFRC177JSON})
	out, _, err := runCmd(t, srv, "team", "view", "177", "--jq", ".nickname")
	requireNoError(t, err, "")
	if strings.TrimSpace(out) != `"Bobcat Robotics"` {
		t.Errorf("jq output = %q", out)
	}
}

func TestTeamViewRequiresExactlyOneArg(t *testing.T) {
	srv := newFakeTBA(t, map[string]any{"/team/frc177": teamFRC177JSON})
	if _, _, err := runCmd(t, srv, "team", "view"); err == nil {
		t.Error("want an error with no arguments")
	}
	if _, _, err := runCmd(t, srv, "team", "view", "177", "1073"); err == nil {
		t.Error("want an error with two arguments")
	}
}

func TestTeamListPaginatesUntilEmptyPage(t *testing.T) {
	srv := newFakeTBA(t, map[string]any{
		"/teams/2024/0": "[" + teamFRC177JSON + "," + teamFRC1073JSON + "]",
		"/teams/2024/1": "[" + teamFRC5507JSON + "]",
		"/teams/2024/2": "[]",
	})
	out, _, err := runCmd(t, srv, "team", "list", "--year", "2024", "--format", "table")
	requireNoError(t, err, "")

	got := lines(out)
	if len(got) != 5 {
		t.Fatalf("want header + separator + 3 rows, got %d lines:\n%s", len(got), out)
	}
	if !strings.HasPrefix(got[0], "Number  Name") {
		t.Errorf("header = %q", got[0])
	}
	if !strings.HasPrefix(got[1], "------") {
		t.Errorf("separator = %q", got[1])
	}
	requireContains(t, got[2], "177")
	requireContains(t, got[2], "Bobcat Robotics")
	requireContains(t, got[3], "The Force Team")
	requireContains(t, got[4], "5507")

	wantPaths := []string{"/teams/2024/0", "/teams/2024/1", "/teams/2024/2"}
	gotPaths := requestPaths(t, srv)
	if len(gotPaths) != len(wantPaths) {
		t.Fatalf("requested %v, want %v", gotPaths, wantPaths)
	}
	for i := range wantPaths {
		if gotPaths[i] != wantPaths[i] {
			t.Errorf("request %d = %q, want %q", i, gotPaths[i], wantPaths[i])
		}
	}
}

func TestTeamListJSONIsTheConcatenatedPages(t *testing.T) {
	srv := newFakeTBA(t, map[string]any{
		"/teams/2024/0": "[" + teamFRC177JSON + "]",
		"/teams/2024/1": "[" + teamFRC1073JSON + "]",
		"/teams/2024/2": "[]",
	})
	out, _, err := runCmd(t, srv, "team", "list", "--year", "2024", "--json")
	requireNoError(t, err, "")

	arr, ok := decodeJSON(t, out).([]any)
	if !ok {
		t.Fatalf("want a JSON array, got %s", out)
	}
	if len(arr) != 2 {
		t.Fatalf("want 2 teams, got %d", len(arr))
	}
	if arr[0].(map[string]any)["key"] != "frc177" {
		t.Errorf("first team = %v", arr[0])
	}
	if arr[1].(map[string]any)["key"] != "frc1073" {
		t.Errorf("second team = %v", arr[1])
	}
}

func TestTeamListCSV(t *testing.T) {
	srv := newFakeTBA(t, map[string]any{
		"/teams/2024/0": "[" + teamFRC177JSON + "]",
		"/teams/2024/1": "[]",
	})
	out, _, err := runCmd(t, srv, "team", "list", "--year", "2024", "--format", "csv")
	requireNoError(t, err, "")

	want := "Number,Name,Location\n177,Bobcat Robotics,\"South Windsor, Connecticut, USA\"\n"
	if out != want {
		t.Errorf("csv =\n%q\nwant\n%q", out, want)
	}
}

func TestTeamListTSV(t *testing.T) {
	srv := newFakeTBA(t, map[string]any{
		"/teams/2024/0": "[" + teamFRC177JSON + "]",
		"/teams/2024/1": "[]",
	})
	out, _, err := runCmd(t, srv, "team", "list", "--year", "2024", "--format", "tsv")
	requireNoError(t, err, "")

	want := "Number\tName\tLocation\n177\tBobcat Robotics\tSouth Windsor, Connecticut, USA\n"
	if out != want {
		t.Errorf("tsv =\n%q\nwant\n%q", out, want)
	}
}

func TestTeamListMarkdown(t *testing.T) {
	srv := newFakeTBA(t, map[string]any{
		"/teams/2024/0": "[" + teamFRC177JSON + "]",
		"/teams/2024/1": "[]",
	})
	out, _, err := runCmd(t, srv, "team", "list", "--year", "2024", "--format", "markdown")
	requireNoError(t, err, "")

	want := "| Number | Name | Location |\n" +
		"| --- | --- | --- |\n" +
		"| 177 | Bobcat Robotics | South Windsor, Connecticut, USA |\n"
	if out != want {
		t.Errorf("markdown =\n%q\nwant\n%q", out, want)
	}
}

// Regression test: --year has a default, so omitting it must work.
func TestTeamListDefaultsToCurrentYear(t *testing.T) {
	year := currentYear()
	srv := newFakeTBA(t, map[string]any{
		fmt.Sprintf("/teams/%d/0", year): "[" + teamFRC177JSON + "]",
		fmt.Sprintf("/teams/%d/1", year): "[]",
	})
	out, _, err := runCmd(t, srv, "team", "list")
	requireNoError(t, err, "")
	requireContains(t, out, "frc177")
}

func TestTeamEvents(t *testing.T) {
	srv := newFakeTBA(t, map[string]any{
		"/team/frc177/events/2024": "[" + event2024ctharJSON + "," + event2024necmpJSON + "]",
	})
	out, _, err := runCmd(t, srv, "team", "events", "177", "--year", "2024", "--format", "table")
	requireNoError(t, err, "")

	got := lines(out)
	if len(got) != 4 {
		t.Fatalf("want header + separator + 2 rows, got %d:\n%s", len(got), out)
	}
	requireContains(t, got[0], "Key")
	requireContains(t, got[0], "Start Date")
	requireContains(t, got[2], "2024cthar")
	requireContains(t, got[2], "NE District Hartford Event")
	requireContains(t, got[2], "2024-03-22")
	requireContains(t, got[2], "Hartford, CT, USA")
	requireContains(t, got[3], "2024necmp")
}

// The API lists a team's events by key, which puts 2024necmp -- April's
// district championship -- ahead of the March district events that qualified
// the team for it. A season is read as a season.
func TestTeamEventsListsTheSeasonInOrder(t *testing.T) {
	srv := newFakeTBA(t, map[string]any{
		"/team/frc177/events/2024": teamEvents177In2024JSON,
	})
	out, errOut, err := runCmd(t, srv, "team", "events", "177", "--year", "2024", "--format", "csv")
	requireNoError(t, err, errOut)

	want := []string{"2024ctwat", "2024cthar", "2024necmp"}
	if got := csvColumn(t, out, 0); !equalStrings(got, want) {
		t.Errorf("events = %v, want %v (chronological)", got, want)
	}
}

func TestTeamEventsDefaultsToCurrentYear(t *testing.T) {
	path := fmt.Sprintf("/team/frc177/events/%d", currentYear())
	srv := newFakeTBA(t, map[string]any{path: "[]"})
	_, _, err := runCmd(t, srv, "team", "events", "177")
	requireNoError(t, err, "")
	// The season lookup comes first; this fake serves no /status, so the year
	// falls back to the calendar.
	if got := requestPaths(t, srv); !contains(got, path) {
		t.Errorf("requested %v, want %s among them", got, path)
	}
}

func TestTeamMatches(t *testing.T) {
	srv := newFakeTBA(t, map[string]any{
		"/team/frc177/matches/2024": "[" + match2024ctharQM1JSON + "," + match2024ctharQM2JSON + "]",
	})
	out, _, err := runCmd(t, srv, "team", "matches", "177", "--year", "2024", "--format", "table")
	requireNoError(t, err, "")

	got := lines(out)
	if len(got) != 4 {
		t.Fatalf("want header + separator + 2 rows, got %d:\n%s", len(got), out)
	}
	// A season's listing leads with the event each match belongs to. When is
	// empty for a listing of played matches, and such a column is left out.
	for _, want := range append([]string{"Event"}, matchHeaders...) {
		if want == "When" {
			continue
		}
		requireContains(t, got[0], want)
	}
	if !strings.HasPrefix(got[0], "Event") {
		t.Errorf("header = %q, want the Event column first", got[0])
	}
	requireContains(t, got[2], "2024cthar")
	requireContains(t, got[2], "Qual 1")
	requireContains(t, got[2], "2024cthar_qm1")
	requireContains(t, got[2], "red")
	requireContains(t, got[3], "2024cthar_qm2")
	requireContains(t, got[3], "blue")
}

// Regression test: `team matches` used to fail without an explicit --year.
func TestTeamMatchesDefaultsToCurrentYear(t *testing.T) {
	path := fmt.Sprintf("/team/frc177/matches/%d", currentYear())
	srv := newFakeTBA(t, map[string]any{path: "[]"})
	_, _, err := runCmd(t, srv, "team", "matches", "177")
	requireNoError(t, err, "")
	// The season lookup comes first; this fake serves no /status, so the year
	// falls back to the calendar.
	if got := requestPaths(t, srv); !contains(got, path) {
		t.Errorf("requested %v, want %s among them", got, path)
	}
}

func TestTeamAwardsAllYears(t *testing.T) {
	srv := newFakeTBA(t, map[string]any{"/team/frc177/awards": teamAwards177JSON})
	out, _, err := runCmd(t, srv, "team", "awards", "177", "--format", "table")
	requireNoError(t, err, "")

	if got := requestPaths(t, srv); !contains(got, "/team/frc177/awards") {
		t.Fatalf("requested %v, want the unfiltered awards path", got)
	}
	requireContains(t, out, "Regional Chairman's Award")
	requireContains(t, out, "2007")
	requireContains(t, out, "2024")
}

func TestTeamAwardsForOneYear(t *testing.T) {
	srv := newFakeTBA(t, map[string]any{
		"/team/frc177/awards/2024": `[{"name":"District Event Winner","award_type":1,` +
			`"event_key":"2024cthar","year":2024,` +
			`"recipient_list":[{"team_key":"frc177","awardee":null}]}]`,
	})
	_, _, err := runCmd(t, srv, "team", "awards", "177", "--year", "2024")
	requireNoError(t, err, "")
	if got := requestPaths(t, srv); !contains(got, "/team/frc177/awards/2024") {
		t.Errorf("requested %v, want the 2024 awards path", got)
	}
}

func TestTeamAwardsCSV(t *testing.T) {
	srv := newFakeTBA(t, map[string]any{"/team/frc177/awards": teamAwards177JSON})
	out, _, err := runCmd(t, srv, "team", "awards", "177", "--format", "csv")
	requireNoError(t, err, "")

	want := "Year,Event,Award,Type,Recipient\n" +
		"2024,,District Event Winner,Winner,\n" +
		"2007,,Regional Chairman's Award,Chairman's/Impact,\n"
	if out != want {
		t.Errorf("csv =\n%q\nwant\n%q", out, want)
	}
}

func TestTeamMedia(t *testing.T) {
	srv := newFakeTBA(t, map[string]any{"/team/frc177/media/2024": teamMedia177JSON})
	out, _, err := runCmd(t, srv, "team", "media", "177", "--year", "2024", "--format", "table")
	requireNoError(t, err, "")

	requireContains(t, out, "Type")
	requireContains(t, out, "imgur")
	requireContains(t, out, "https://imgur.com/aBcDeFg")
	requireContains(t, out, "youtube")
}

// Regression test: `team media` used to fail without an explicit --year.
func TestTeamMediaDefaultsToCurrentYear(t *testing.T) {
	path := fmt.Sprintf("/team/frc177/media/%d", currentYear())
	srv := newFakeTBA(t, map[string]any{path: "[]"})
	_, _, err := runCmd(t, srv, "team", "media", "177")
	requireNoError(t, err, "")
	// The season lookup comes first; this fake serves no /status, so the year
	// falls back to the calendar.
	if got := requestPaths(t, srv); !contains(got, path) {
		t.Errorf("requested %v, want %s among them", got, path)
	}
}

func TestTeamRobots(t *testing.T) {
	srv := newFakeTBA(t, map[string]any{"/team/frc177/robots": teamRobots177JSON})
	out, _, err := runCmd(t, srv, "team", "robots", "177", "--format", "table")
	requireNoError(t, err, "")

	got := lines(out)
	if got[0] != "Year  Robot Name " {
		t.Errorf("header = %q", got[0])
	}
	if got[1] != "----  -----------" {
		t.Errorf("separator = %q", got[1])
	}
	if got[2] != "2024  Bobcat 2024" {
		t.Errorf("row = %q", got[2])
	}
}

func TestTeamRobotsJq(t *testing.T) {
	srv := newFakeTBA(t, map[string]any{"/team/frc177/robots": teamRobots177JSON})
	out, _, err := runCmd(t, srv, "team", "robots", "177", "--jq", ".[].robot_name")
	requireNoError(t, err, "")
	if out != "\"Bobcat 2024\"\n\"Sprocket\"\n" {
		t.Errorf("jq output = %q", out)
	}
}

func TestTeamDistricts(t *testing.T) {
	srv := newFakeTBA(t, map[string]any{"/team/frc177/districts": teamDistricts177JSON})
	out, _, err := runCmd(t, srv, "team", "districts", "177", "--format", "markdown")
	requireNoError(t, err, "")

	want := "| Key | Name | Year |\n" +
		"| --- | --- | --- |\n" +
		"| 2023ne | New England | 2023 |\n" +
		"| 2024ne | New England | 2024 |\n"
	if out != want {
		t.Errorf("markdown =\n%q\nwant\n%q", out, want)
	}
}

// A team argument that cannot be a team number is a usage mistake, caught
// before a request is spent on it.
func TestBadTeamArgumentIsAUsageErrorAndCostsNoRequest(t *testing.T) {
	commands := []string{"view", "events", "years", "matches", "awards", "media", "robots", "districts"}
	for _, sub := range commands {
		t.Run(sub, func(t *testing.T) {
			srv := newFakeTBA(t, map[string]any{})
			err := requireExitCode(t, clierr.ExitUsage, srv, "team", sub, "17x7")
			requireErrorContains(t, err, `"17x7" is not a team number`)
			requireErrorContains(t, err, "frc177")
			if got := requestPaths(t, srv); len(got) != 0 {
				t.Errorf("a usage error must not reach the API, got %v", got)
			}
		})
	}
}

func TestTeamArgumentAccepts(t *testing.T) {
	for _, arg := range []string{"177", "frc177", "FRC177", "00007", "99999"} {
		if err := validateTeamArg(arg); err != nil {
			t.Errorf("validateTeamArg(%q) = %v, want it accepted", arg, err)
		}
	}
	// Surrounding space is trimmed, so " 177 " is fine; nothing else is.
	if err := validateTeamArg(" 177 "); err != nil {
		t.Errorf("validateTeamArg(%q) = %v, want it accepted", " 177 ", err)
	}
	for _, arg := range []string{"", "17x7", "frc", "frc17x7", "177177", "-177", "1 7 7"} {
		if err := validateTeamArg(arg); err == nil {
			t.Errorf("validateTeamArg(%q) = nil, want a usage error", arg)
		}
	}
}

func TestEventListRejectsABadTeamFilter(t *testing.T) {
	srv := newFakeTBA(t, map[string]any{})
	err := requireExitCode(t, clierr.ExitUsage, srv, "event", "list", "--year", "2024", "--team", "17x7")
	requireErrorContains(t, err, `"17x7" is not a team number`)
	if got := requestPaths(t, srv); len(got) != 0 {
		t.Errorf("a usage error must not reach the API, got %v", got)
	}
}

func TestTeamSubcommandsAreRegistered(t *testing.T) {
	got := subcommandNames(t, "team")
	for _, want := range []string{"view", "list", "events", "matches", "awards", "media", "robots", "districts"} {
		if !contains(got, want) {
			t.Errorf("team %s is not registered (have %v)", want, got)
		}
	}
}
