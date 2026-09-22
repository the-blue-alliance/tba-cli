package cmd

import (
	"fmt"
	"strings"
	"testing"
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

func TestInsightLeaderboardsRawWhenNotJSON(t *testing.T) {
	srv := newFakeTBA(t, map[string]any{
		"/insights/leaderboards/2024": leaderboards2024JSON,
	})
	out, _, err := runCmd(t, srv, "insight", "leaderboards", "--year", "2024", "--format", "table")
	requireNoError(t, err, "")
	// The endpoint has no stable schema, so the body is passed through as-is.
	if strings.TrimSpace(out) != strings.TrimSpace(leaderboards2024JSON) {
		t.Errorf("raw output =\n%s", out)
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

// Regression test: `insight leaderboards` used to fail without --year.
func TestInsightLeaderboardsDefaultsToCurrentYear(t *testing.T) {
	path := fmt.Sprintf("/insights/leaderboards/%d", currentYear())
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

// Regression test: `insight notables` used to fail without --year.
func TestInsightNotablesDefaultsToCurrentYear(t *testing.T) {
	path := fmt.Sprintf("/insights/notables/%d", currentYear())
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

// Insights have no stable schema, so the tabular formats fall back to JSON
// rather than failing or silently printing the raw body.
func TestInsightLeaderboardsFallsBackToJSONForTabularFormats(t *testing.T) {
	for _, format := range []string{"csv", "tsv", "markdown"} {
		t.Run(format, func(t *testing.T) {
			srv := newFakeTBA(t, map[string]any{
				"/insights/leaderboards/2024": leaderboards2024JSON,
			})
			out, _, err := runCmd(t, srv, "insight", "leaderboards", "--year", "2024", "--format", format)
			requireNoError(t, err, "")
			arr, ok := decodeJSON(t, out).([]any)
			if !ok {
				t.Fatalf("want a JSON array, got:\n%s", out)
			}
			if arr[0].(map[string]any)["name"] != "typed_leaderboard_blue_banners" {
				t.Errorf("unexpected payload:\n%s", out)
			}
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

func TestInsightNotablesRawWhenTable(t *testing.T) {
	srv := newFakeTBA(t, map[string]any{"/insights/notables/2024": notables2024JSON})
	out, _, err := runCmd(t, srv, "insight", "notables", "--year", "2024", "--format", "table")
	requireNoError(t, err, "")
	if strings.TrimSpace(out) != strings.TrimSpace(notables2024JSON) {
		t.Errorf("raw output =\n%s", out)
	}
}
