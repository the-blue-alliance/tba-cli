package apispec

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"reflect"
	"strings"
	"testing"
)

func upstreamField(t *testing.T, name string) string {
	t.Helper()
	for _, line := range strings.Split(strings.ReplaceAll(Upstream(), "\r\n", "\n"), "\n") {
		if v, ok := strings.CutPrefix(line, name+": "); ok {
			return v
		}
	}
	t.Fatalf("UPSTREAM has no %s line", name)
	return ""
}

// The vendored spec is byte-for-byte what UPSTREAM says was fetched.
func TestVendoredSpecIsUnmodified(t *testing.T) {
	// A Windows checkout may have rewritten the line endings.
	sum := sha256.Sum256(bytes.ReplaceAll(RawSpec(), []byte("\r\n"), []byte("\n")))
	if got, want := hex.EncodeToString(sum[:]), upstreamField(t, "sha256"); got != want {
		t.Errorf("api_v3.json sha256 = %s, UPSTREAM says %s; refresh with go generate, do not edit it", got, want)
	}
	s, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if got, want := s.Version(), upstreamField(t, "version"); got != want {
		t.Errorf("spec version %s, UPSTREAM says %s", got, want)
	}
	if c := upstreamField(t, "commit"); len(c) != 40 {
		t.Errorf("UPSTREAM commit %q is not a full SHA", c)
	}
}

func TestMatchPath(t *testing.T) {
	s, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	for path, want := range map[string]string{
		"/team/frc177":                        "/team/{team_key}",
		"/team/frc177/events/2024":            "/team/{team_key}/events/{year}",
		"/team/frc177/events/keys":            "/team/{team_key}/events/keys",
		"/insights/leaderboards/2024":         "/insights/leaderboards/{year}",
		"/insights/2024/teams":                "/insights/{year}/{category}",
		"/teams/2024/0?page=1":                "/teams/{year}/{page_num}",
		"/team/frc177/event/2024cthar/status": "/team/{team_key}/event/{event_key}/status",
	} {
		if got, ok := s.MatchPath(path); !ok || got != want {
			t.Errorf("MatchPath(%q) = %q, %v; want %q", path, got, ok, want)
		}
	}
	for _, path := range []string{"/team/frc177/matches", "/team//events", "/nope"} {
		if got, ok := s.MatchPath(path); ok {
			t.Errorf("MatchPath(%q) = %q, want no match", path, got)
		}
	}
}

func TestCheck(t *testing.T) {
	s, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	ptr := ResponsePointer("/team/{team_key}/robots")
	good := `[{"key": "frc177_2024", "robot_name": "Bobcat", "team_key": "frc177", "year": 2024}]`
	if p, err := s.Check(ptr, []byte(good)); err != nil || len(p) != 0 {
		t.Errorf("valid robots: %v, %v", p, err)
	}
	cases := map[string]string{
		`[{"key": "frc177_2024", "robot_name": "Bobcat", "team_key": "frc177", "year": "2024"}]`:       "/0/year",
		`[{"key": "frc177_2024", "robot_name": "Bobcat", "team_key": "frc177", "year": 2024, "x": 1}]`: "/0/x",
		`[{"key": "frc177_2024", "robot_name": "Bobcat", "year": 2024}]`:                               "/0",
		`[{"key": "frc177_2024", "robot_name": "Bobcat", "team_key": null, "year": 2024}]`:             "/0/team_key",
	}
	for body, at := range cases {
		p, err := s.Check(ptr, []byte(body))
		if err != nil {
			t.Fatal(err)
		}
		if len(p) != 1 || p[0].Instance != at {
			t.Errorf("Check(%s) = %v, want one problem at %s", body, p, at)
		}
	}
}

// A nullable value that is the wrong type reports the type, not "want null".
func TestCheckHidesTheNullBranch(t *testing.T) {
	s, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	body := `{"rankings": [{"team_key": "frc177", "rank": 1, "record": {"wins": "two", "losses": 0, "ties": 0},
	  "qual_average": null, "matches_played": 1, "dq": 0, "sort_orders": [], "extra_stats": []}],
	  "sort_order_info": [], "extra_stats_info": []}`
	p, err := s.Check(ResponsePointer("/event/{event_key}/rankings"), []byte(body))
	if err != nil {
		t.Fatal(err)
	}
	if len(p) != 1 || p[0].Instance != "/rankings/0/record/wins" {
		t.Errorf("got %v, want only /rankings/0/record/wins", p)
	}
}

func TestUndeclaredStructFields(t *testing.T) {
	s, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	type record struct {
		Wins   int     `json:"wins"`
		Draws  int     `json:"draws"`
		Losses float64 `json:"losses"`
		Ties   string  `json:"ties"`
	}
	type robot struct {
		Year   int      `json:"year"`
		Record []record `json:"record"`
		Skip   string   `json:"-"`
		hidden string
	}
	_ = robot{}.hidden
	got, err := s.UndeclaredStructFields(SchemaPointer("WLT_Record"), reflect.TypeFor[record]())
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"record.draws", "record.ties: Go string cannot hold JSON integer"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
	got, err = s.UndeclaredStructFields(SchemaPointer("Team_Robot"), reflect.TypeFor[robot]())
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, []string{"robot.record"}) {
		t.Errorf("got %v, want robot.record", got)
	}
}
