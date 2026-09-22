package cmd

import (
	"net/http/httptest"
	"strings"
	"testing"
)

// complete drives cobra's hidden __complete command, which is what a shell
// calls on Tab. The last line of its output is the directive, so only the
// suggestion lines are returned.
func complete(t *testing.T, srv *httptest.Server, args ...string) []string {
	t.Helper()
	out, stderr, err := runCmd(t, srv, append([]string{"__complete"}, args...)...)
	requireNoError(t, err, stderr)
	var suggestions []string
	for _, l := range lines(out) {
		if l == "" || strings.HasPrefix(l, ":") {
			continue
		}
		suggestions = append(suggestions, l)
	}
	return suggestions
}

// warmCache points the cache at a fresh directory and runs a command against
// the fake API to fill it, the way a user's first real command would.
func warmCache(t *testing.T, args ...string) *httptest.Server {
	t.Helper()
	t.Setenv("TBA_CACHE_DIR", t.TempDir())
	srv := newFakeTBA(t, map[string]any{
		"/events/2024":             events2024JSON,
		"/districts/2024":          districts2024JSON,
		"/event/2024cthar/matches": "[" + match2024ctharQM1JSON + "," + match2024ctharQM2JSON + "]",
	})
	_, stderr, err := runCmd(t, srv, args...)
	requireNoError(t, err, stderr)
	return srv
}

func TestEventKeyCompletionFromAWarmCache(t *testing.T) {
	warmCache(t, "event", "list", "--year", "2024")

	got := complete(t, nil, "event", "view", "2024c")
	want := []string{
		"2024casj\tSilicon Valley Regional",
		"2024cthar\tNE District Hartford Event",
	}
	if strings.Join(got, "|") != strings.Join(want, "|") {
		t.Errorf("completions = %v, want %v", got, want)
	}
}

func TestEventKeyCompletionCoversEveryEventSubcommand(t *testing.T) {
	warmCache(t, "event", "list", "--year", "2024")
	for _, sub := range []string{"view", "teams", "matches", "rankings", "alliances", "awards", "oprs", "export"} {
		t.Run(sub, func(t *testing.T) {
			got := complete(t, nil, "event", sub, "2024ct")
			if len(got) != 1 || !strings.HasPrefix(got[0], "2024cthar\t") {
				t.Errorf("completions = %v, want 2024cthar", got)
			}
		})
	}
}

func TestEventKeyCompletionIsEmptyWithoutACache(t *testing.T) {
	t.Setenv("TBA_CACHE_DIR", t.TempDir())
	if got := complete(t, nil, "event", "view", "2024c"); len(got) != 0 {
		t.Errorf("completions = %v, want none from an empty cache", got)
	}
}

func TestEventKeyCompletionMakesNoRequests(t *testing.T) {
	srv := warmCache(t, "event", "list", "--year", "2024")
	before := len(requestPaths(t, srv))
	_ = complete(t, srv, "event", "view", "2024")
	if after := len(requestPaths(t, srv)); after != before {
		t.Errorf("completion made %d requests, want none", after-before)
	}
}

func TestEventKeyCompletionOnlyOffersTheFirstArgument(t *testing.T) {
	warmCache(t, "event", "list", "--year", "2024")
	if got := complete(t, nil, "event", "view", "2024cthar", "2024c"); len(got) != 0 {
		t.Errorf("completions = %v, want none for a second argument", got)
	}
}

func TestEventKeyCompletionFromATeamsEventList(t *testing.T) {
	t.Setenv("TBA_CACHE_DIR", t.TempDir())
	srv := newFakeTBA(t, map[string]any{"/team/frc177/events/2024": teamEvents177JSON})
	_, stderr, err := runCmd(t, srv, "team", "events", "177", "--year", "2024")
	requireNoError(t, err, stderr)

	got := complete(t, nil, "event", "view", "2024ne")
	if len(got) != 1 || !strings.HasPrefix(got[0], "2024necmp\t") {
		t.Errorf("completions = %v, want 2024necmp", got)
	}
}

func TestMatchKeyCompletionFromAWarmCache(t *testing.T) {
	warmCache(t, "event", "matches", "2024cthar")

	got := complete(t, nil, "match", "view", "2024cthar_qm")
	want := []string{"2024cthar_qm1", "2024cthar_qm2"}
	if strings.Join(got, "|") != strings.Join(want, "|") {
		t.Errorf("completions = %v, want %v", got, want)
	}
}

func TestMatchKeyCompletionIsEmptyWithoutACache(t *testing.T) {
	t.Setenv("TBA_CACHE_DIR", t.TempDir())
	if got := complete(t, nil, "match", "view", "2024"); len(got) != 0 {
		t.Errorf("completions = %v, want none", got)
	}
}

func TestDistrictFlagCompletionFromCachedDistricts(t *testing.T) {
	t.Setenv("TBA_CACHE_DIR", t.TempDir())
	srv := newFakeTBA(t, map[string]any{"/districts/2024": districts2024JSON})
	_, stderr, err := runCmd(t, srv, "district", "list", "--year", "2024")
	requireNoError(t, err, stderr)

	got := complete(t, nil, "event", "list", "--district", "")
	want := []string{"fim\tFIRST In Michigan", "ne\tNew England"}
	if strings.Join(got, "|") != strings.Join(want, "|") {
		t.Errorf("completions = %v, want %v", got, want)
	}
}

func TestDistrictFlagCompletionFromCachedEvents(t *testing.T) {
	warmCache(t, "event", "list", "--year", "2024")
	got := complete(t, nil, "event", "list", "--district", "")
	// The event list names New England and FIRST Israel, and nothing else.
	want := []string{"isr\tFIRST Israel", "ne\tNew England"}
	if strings.Join(got, "|") != strings.Join(want, "|") {
		t.Errorf("completions = %v, want %v", got, want)
	}
}

func TestDistrictFlagCompletionIsEmptyWithoutACache(t *testing.T) {
	t.Setenv("TBA_CACHE_DIR", t.TempDir())
	if got := complete(t, nil, "event", "list", "--district", ""); len(got) != 0 {
		t.Errorf("completions = %v, want none", got)
	}
}

func TestTypeFlagCompletionNeedsNoCache(t *testing.T) {
	t.Setenv("TBA_CACHE_DIR", t.TempDir())
	got := complete(t, nil, "event", "list", "--type", "d")
	want := []string{"district", "dcmp", "dcmp-division"}
	if strings.Join(got, "|") != strings.Join(want, "|") {
		t.Errorf("completions = %v, want %v", got, want)
	}
}

func TestTypeFlagCompletionContinuesAList(t *testing.T) {
	t.Setenv("TBA_CACHE_DIR", t.TempDir())
	got := complete(t, nil, "event", "list", "--type", "district,reg")
	if len(got) != 1 || got[0] != "district,regional" {
		t.Errorf("completions = %v, want district,regional", got)
	}
}

func TestTeamArgumentsCompleteToNothing(t *testing.T) {
	warmCache(t, "event", "list", "--year", "2024")
	for _, sub := range []string{"view", "events", "awards", "years", "matches"} {
		t.Run(sub, func(t *testing.T) {
			if got := complete(t, nil, "team", sub, "17"); len(got) != 0 {
				t.Errorf("completions = %v, want none for a team number", got)
			}
		})
	}
}

func TestOpenCompletesEventAndMatchKeys(t *testing.T) {
	t.Setenv("TBA_CACHE_DIR", t.TempDir())
	srv := newFakeTBA(t, map[string]any{
		"/events/2024":             events2024JSON,
		"/event/2024cthar/matches": "[" + match2024ctharQM1JSON + "]",
	})
	_, stderr, err := runCmd(t, srv, "event", "list", "--year", "2024")
	requireNoError(t, err, stderr)
	_, stderr, err = runCmd(t, srv, "event", "matches", "2024cthar")
	requireNoError(t, err, stderr)

	got := complete(t, nil, "open", "2024cthar")
	joined := strings.Join(got, "|")
	if !strings.Contains(joined, "2024cthar\tNE District Hartford Event") {
		t.Errorf("completions = %v, want the event key", got)
	}
	if !strings.Contains(joined, "2024cthar_qm1") {
		t.Errorf("completions = %v, want the match key", got)
	}
}

func TestCompletionIgnoresUnrelatedCacheEntries(t *testing.T) {
	t.Setenv("TBA_CACHE_DIR", t.TempDir())
	srv := newFakeTBA(t, map[string]any{
		"/team/frc177": teamFRC177JSON,
		"/status":      apiStatusJSON,
	})
	_, stderr, err := runCmd(t, srv, "team", "view", "177")
	requireNoError(t, err, stderr)
	_, stderr, err = runCmd(t, srv, "status")
	requireNoError(t, err, stderr)

	if got := complete(t, nil, "event", "view", ""); len(got) != 0 {
		t.Errorf("completions = %v, want none", got)
	}
}
