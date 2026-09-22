package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/the-blue-alliance/tba-cli/internal/api"
	"github.com/the-blue-alliance/tba-cli/internal/clierr"
	"github.com/the-blue-alliance/tba-cli/internal/frc"
)

// --- fakes -------------------------------------------------------------

// watchClock pins the clock the watch loop reads and turns its waiting into
// bookkeeping, so a two-hour watch runs in microseconds.
type watchClock struct {
	mu     sync.Mutex
	now    time.Time
	waits  []time.Duration
	onWait func(c *watchClock)
}

// fakeWatchClock installs a pinned clock for the duration of a test.
func fakeWatchClock(t *testing.T, start time.Time) *watchClock {
	t.Helper()
	c := &watchClock{now: start}
	previousNow, previousSleep := nowFunc, watchSleep
	nowFunc = c.Now
	watchSleep = c.Sleep
	t.Cleanup(func() { nowFunc, watchSleep = previousNow, previousSleep })
	return c
}

func (c *watchClock) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.now
}

// Sleep advances the pinned clock rather than the wall clock, then runs the
// hook a test uses to change what the API is about to say.
func (c *watchClock) Sleep(ctx context.Context, d time.Duration) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	c.mu.Lock()
	c.now = c.now.Add(d)
	c.waits = append(c.waits, d)
	hook := c.onWait
	c.mu.Unlock()
	if hook != nil {
		hook(c)
	}
	return ctx.Err()
}

func (c *watchClock) sleptFor() []time.Duration {
	c.mu.Lock()
	defer c.mu.Unlock()
	out := make([]time.Duration, len(c.waits))
	copy(out, c.waits)
	return out
}

// watchSetBody replaces what the fake API serves for a path, which is how a
// test makes something happen between two polls. It lives here rather than in
// the shared helpers so that the watch tests own their own mutation.
func watchSetBody(t *testing.T, srv *httptest.Server, path string, v any) {
	t.Helper()
	body := toJSONBytes(t, v)
	st := stateFor(t, srv)
	st.mu.Lock()
	defer st.mu.Unlock()
	st.bodies[path] = body
}

// --- fixtures ----------------------------------------------------------

const watchEventKey = "2024cthar"
const watchMatchesPath = "/event/2024cthar/matches"
const watchRankingsPath = "/event/2024cthar/rankings"

func watchEpoch(v int64) *int64 { return &v }

// watchQual builds one qualification match at 2024cthar. A score of -1 is the
// API's way of saying the match has not been played.
func watchQual(number int, red, blue []string, redScore, blueScore int, winner string, predicted int64) api.Match {
	m := api.Match{
		Key:             fmt.Sprintf("%s_qm%d", watchEventKey, number),
		CompLevel:       "qm",
		SetNumber:       1,
		MatchNumber:     number,
		EventKey:        watchEventKey,
		WinningAlliance: winner,
		Alliances: map[string]api.Alliance{
			"red":  {Score: redScore, TeamKeys: red},
			"blue": {Score: blueScore, TeamKeys: blue},
		},
	}
	if predicted != 0 {
		m.PredictedTime = watchEpoch(predicted)
	}
	return m
}

var (
	watchRed    = []string{"frc177", "frc1073", "frc5507"}
	watchBlue   = []string{"frc230", "frc1071", "frc4055"}
	watchOther  = []string{"frc558", "frc3467", "frc2168"}
	watchOther2 = []string{"frc195", "frc1124", "frc6153"}
)

// watchMatchesV1 is the event before anything happens: one played match, two
// still to come. Only qm1 and qm3 involve 177.
func watchMatchesV1() []api.Match {
	return []api.Match{
		watchQual(1, watchRed, watchBlue, 88, 61, "red", 1711130400),
		watchQual(2, watchOther, watchOther2, -1, -1, "", 1711131000),
		watchQual(3, watchRed, watchOther2, -1, -1, "", 1711131600),
	}
}

// watchMatchesQM2Played is V1 with qm2 finished.
func watchMatchesQM2Played() []api.Match {
	out := watchMatchesV1()
	out[1] = watchQual(2, watchOther, watchOther2, 101, 99, "red", 1711131000)
	return out
}

func watchRankingsV1() api.EventRankings {
	return api.EventRankings{
		Rankings: []api.Ranking{
			{TeamKey: "frc177", Rank: 1, Record: &api.WLTRecord{Wins: 2}, MatchesPlayed: 2},
			{TeamKey: "frc230", Rank: 2, Record: &api.WLTRecord{Wins: 1, Losses: 1}, MatchesPlayed: 2},
		},
		SortOrderInfo: []api.SortOrderInfo{{Name: "Ranking Score", Precision: 2}},
	}
}

// watchRankingsV2 swaps the top two.
func watchRankingsV2() api.EventRankings {
	return api.EventRankings{
		Rankings: []api.Ranking{
			{TeamKey: "frc230", Rank: 1, Record: &api.WLTRecord{Wins: 2, Losses: 1}, MatchesPlayed: 3},
			{TeamKey: "frc177", Rank: 2, Record: &api.WLTRecord{Wins: 2, Losses: 1}, MatchesPlayed: 3},
		},
		SortOrderInfo: []api.SortOrderInfo{{Name: "Ranking Score", Precision: 2}},
	}
}

func watchServer(t *testing.T) *httptest.Server {
	t.Helper()
	return newFakeTBA(t, map[string]any{
		"/event/2024cthar": event2024ctharJSON,
		watchMatchesPath:   watchMatchesV1(),
		watchRankingsPath:  watchRankingsV1(),
	})
}

// pollCount is how many times the matches endpoint was asked.
func pollCount(t *testing.T, srv *httptest.Server) int {
	t.Helper()
	n := 0
	for _, p := range requestPaths(t, srv) {
		if p == watchMatchesPath {
			n++
		}
	}
	return n
}

// jsonLines decodes every line of a JSON Lines stream, failing if any line is
// not an object of its own.
func jsonLines(t *testing.T, out string) []map[string]any {
	t.Helper()
	if strings.TrimSpace(out) == "" {
		return nil
	}
	var decoded []map[string]any
	for i, line := range lines(out) {
		var v map[string]any
		if err := json.Unmarshal([]byte(line), &v); err != nil {
			t.Fatalf("line %d is not a JSON object: %v\n%s", i+1, err, line)
		}
		decoded = append(decoded, v)
	}
	return decoded
}

// --- first poll --------------------------------------------------------

// The first poll prints the listing a user already knows, column for column,
// so that `event watch` and `event matches` never drift apart.
func TestEventWatchFirstPollPrintsTheSameTableAsEventMatches(t *testing.T) {
	fakeWatchClock(t, time.Date(2024, 3, 22, 14, 31, 7, 0, time.Local))
	srv := watchServer(t)

	watched, errOut, err := runCmd(t, srv, "event", "watch", watchEventKey, "--format", "table", "--max-polls", "1")
	requireNoError(t, err, errOut)

	listed, _, err := runCmd(t, srv, "event", "matches", watchEventKey, "--format", "table")
	requireNoError(t, err, "")

	if watched != listed {
		t.Errorf("the first poll is not the match listing:\n%s\n---\n%s", watched, listed)
	}
	if pollCount(t, srv) != 2 { // one for the watch, one for the listing
		t.Errorf("polls = %d, want 2", pollCount(t, srv))
	}
}

func TestEventWatchFirstPollPrintsTheSnapshotAsOneJSONLine(t *testing.T) {
	fakeWatchClock(t, time.Date(2024, 3, 22, 14, 31, 7, 0, time.Local))
	srv := watchServer(t)

	out, errOut, err := runCmd(t, srv, "event", "watch", watchEventKey, "--max-polls", "1")
	requireNoError(t, err, errOut)

	got := jsonLines(t, out)
	if len(got) != 1 {
		t.Fatalf("want one snapshot line, got %d:\n%s", len(got), out)
	}
	line := got[0]
	if line["type"] != "snapshot" {
		t.Errorf("type = %v, want snapshot", line["type"])
	}
	matches, ok := line["matches"].([]any)
	if !ok || len(matches) != 3 {
		t.Fatalf("snapshot.matches = %v, want three matches", line["matches"])
	}
	if _, ok := line["rankings"]; ok {
		t.Errorf("rankings should be absent without --rankings: %v", line)
	}
	if ts, _ := line["ts"].(string); ts == "" {
		t.Errorf("every line carries a timestamp, got %v", line)
	}
}

// --- later polls -------------------------------------------------------

// runWatch drives a watch whose API changes between polls.
func runWatch(t *testing.T, srv *httptest.Server, change func(), args ...string) (string, string, error) {
	t.Helper()
	clock := fakeWatchClock(t, time.Date(2024, 3, 22, 14, 31, 7, 0, time.Local))
	once := false
	clock.onWait = func(*watchClock) {
		if once || change == nil {
			return
		}
		once = true
		change()
	}
	return runCmd(t, srv, append([]string{"event", "watch", watchEventKey}, args...)...)
}

func TestEventWatchLaterPollPrintsOnlyTheChangedRow(t *testing.T) {
	srv := watchServer(t)
	out, errOut, err := runWatch(t, srv, func() {
		watchSetBody(t, srv, watchMatchesPath, watchMatchesQM2Played())
	}, "--format", "table", "--max-polls", "2")
	requireNoError(t, err, errOut)

	got := lines(out)
	// Header, separator, three rows, then the poll header and one row.
	if len(got) != 7 {
		t.Fatalf("want 7 lines (a table, a poll header and one row), got %d:\n%s", len(got), out)
	}
	if got[5] != "--- 14:32:07 (poll 2) ---" {
		t.Errorf("poll header = %q", got[5])
	}
	if !strings.Contains(got[6], "2024cthar_qm2") || !strings.Contains(got[6], "101-99") {
		t.Errorf("changed row = %q, want qm2 with its result", got[6])
	}
	for _, unchanged := range []string{"2024cthar_qm1", "2024cthar_qm3"} {
		if strings.Contains(got[6], unchanged) {
			t.Errorf("row %q should only carry the match that changed", got[6])
		}
	}
}

// A row printed on a later poll lines up under the table printed first, even
// though it was measured on its own.
func TestEventWatchKeepsColumnWidthsFixedAfterTheFirstPoll(t *testing.T) {
	srv := watchServer(t)
	out, errOut, err := runWatch(t, srv, func() {
		watchSetBody(t, srv, watchMatchesPath, watchMatchesQM2Played())
	}, "--format", "table", "--max-polls", "2")
	requireNoError(t, err, errOut)

	got := lines(out)
	first, later := got[3], got[6] // qm2 in the opening table, qm2 after it played
	if !strings.Contains(first, "2024cthar_qm2") || !strings.Contains(later, "2024cthar_qm2") {
		t.Fatalf("expected qm2 in both %q and %q", first, later)
	}
	for _, column := range []string{"2024cthar_qm2", "558, 3467, 2168", "195, 1124, 6153"} {
		if strings.Index(first, column) != strings.Index(later, column) {
			t.Errorf("column %q moved from %d to %d:\n%s\n%s",
				column, strings.Index(first, column), strings.Index(later, column), first, later)
		}
	}
}

func TestEventWatchEmitsOneJSONLinePerChange(t *testing.T) {
	srv := watchServer(t)
	out, errOut, err := runWatch(t, srv, func() {
		watchSetBody(t, srv, watchMatchesPath, watchMatchesQM2Played())
	}, "--max-polls", "2")
	requireNoError(t, err, errOut)

	got := jsonLines(t, out)
	if len(got) != 2 {
		t.Fatalf("want a snapshot and one change, got %d lines:\n%s", len(got), out)
	}
	change := got[1]
	if change["type"] != "match" || change["change"] != "played" {
		t.Errorf("change line = %v, want a played match", change)
	}
	if change["key"] != "2024cthar_qm2" {
		t.Errorf("key = %v, want 2024cthar_qm2", change["key"])
	}
	if poll, _ := change["poll"].(float64); poll != 2 {
		t.Errorf("poll = %v, want 2", change["poll"])
	}
	wantTS := time.Date(2024, 3, 22, 14, 32, 7, 0, time.Local).Format(time.RFC3339)
	if ts, _ := change["ts"].(string); ts != wantTS {
		t.Errorf("ts = %q, want the RFC3339 time of the poll", ts)
	}
	match, ok := change["match"].(map[string]any)
	if !ok || match["key"] != "2024cthar_qm2" {
		t.Fatalf("the line should carry the whole match, got %v", change["match"])
	}
	if _, ok := match["alliances"]; !ok {
		t.Errorf("the embedded match should be the full API match: %v", match)
	}
}

// A poll the API answers with 304 found nothing new, and says nothing.
func TestEventWatchSaysNothingWhenNothingChanged(t *testing.T) {
	srv := watchServer(t)
	setETag(t, srv, watchMatchesPath, `"v1"`)

	out, errOut, err := runWatch(t, srv, nil, "--max-polls", "3")
	requireNoError(t, err, errOut)

	if got := jsonLines(t, out); len(got) != 1 {
		t.Fatalf("want only the snapshot, got %d lines:\n%s", len(got), out)
	}
	revalidated := 0
	for _, r := range requestsTo(t, srv) {
		if r.Path == watchMatchesPath && r.Headers.Get("If-None-Match") == `"v1"` {
			revalidated++
		}
	}
	if revalidated != 2 {
		t.Errorf("polls 2 and 3 should have been conditional, %d were", revalidated)
	}
}

// actual_time is restamped as scoring is finalised and says nothing new.
func TestEventWatchIgnoresActualTimeOnlyChanges(t *testing.T) {
	srv := watchServer(t)
	out, errOut, err := runWatch(t, srv, func() {
		updated := watchMatchesV1()
		updated[0].ActualTime = watchEpoch(1711130999)
		watchSetBody(t, srv, watchMatchesPath, updated)
	}, "--max-polls", "2")
	requireNoError(t, err, errOut)

	if got := jsonLines(t, out); len(got) != 1 {
		t.Errorf("a new actual_time is not news, got:\n%s", out)
	}
}

func TestEventWatchReportsAddedAndRescheduledMatches(t *testing.T) {
	srv := watchServer(t)
	out, errOut, err := runWatch(t, srv, func() {
		updated := watchMatchesV1()
		updated[2].PredictedTime = watchEpoch(1711132200) // ten minutes later
		updated = append(updated, watchQual(4, watchRed, watchOther, -1, -1, "", 1711132800))
		watchSetBody(t, srv, watchMatchesPath, updated)
	}, "--max-polls", "2")
	requireNoError(t, err, errOut)

	got := jsonLines(t, out)
	if len(got) != 3 {
		t.Fatalf("want a snapshot and two changes, got %d:\n%s", len(got), out)
	}
	want := map[string]string{"2024cthar_qm3": "rescheduled", "2024cthar_qm4": "added"}
	for _, line := range got[1:] {
		key, _ := line["key"].(string)
		if want[key] != line["change"] {
			t.Errorf("%s reported as %v, want %q", key, line["change"], want[key])
		}
		delete(want, key)
	}
	for key := range want {
		t.Errorf("no change reported for %s", key)
	}
}

// A prediction that drifts by less than a minute is the queue breathing.
func TestEventWatchIgnoresSmallScheduleDrift(t *testing.T) {
	srv := watchServer(t)
	out, errOut, err := runWatch(t, srv, func() {
		updated := watchMatchesV1()
		updated[2].PredictedTime = watchEpoch(1711131630) // thirty seconds
		watchSetBody(t, srv, watchMatchesPath, updated)
	}, "--max-polls", "2")
	requireNoError(t, err, errOut)

	if got := jsonLines(t, out); len(got) != 1 {
		t.Errorf("thirty seconds is not a reschedule, got:\n%s", out)
	}
}

// --- --team ------------------------------------------------------------

func TestEventWatchTeamRestrictsTheFeed(t *testing.T) {
	srv := watchServer(t)
	out, errOut, err := runWatch(t, srv, func() {
		watchSetBody(t, srv, watchMatchesPath, watchMatchesQM2Played())
	}, "--team", "177", "--max-polls", "2")
	requireNoError(t, err, errOut)

	got := jsonLines(t, out)
	if len(got) != 1 {
		t.Fatalf("qm2 has no 177 in it, so nothing should have been reported:\n%s", out)
	}
	matches, _ := got[0]["matches"].([]any)
	if len(matches) != 2 {
		t.Fatalf("snapshot should hold 177's two matches, got %d", len(matches))
	}
	for _, m := range matches {
		key := m.(map[string]any)["key"]
		if key != "2024cthar_qm1" && key != "2024cthar_qm3" {
			t.Errorf("unexpected match %v in a --team 177 snapshot", key)
		}
	}
}

func TestEventWatchTeamAcceptsEitherSpelling(t *testing.T) {
	for _, team := range []string{"177", "frc177"} {
		t.Run(team, func(t *testing.T) {
			srv := watchServer(t)
			out, errOut, err := runWatch(t, srv, nil, "--team", team, "--max-polls", "1")
			requireNoError(t, err, errOut)
			matches, _ := jsonLines(t, out)[0]["matches"].([]any)
			if len(matches) != 2 {
				t.Errorf("--team %s matched %d matches, want 2", team, len(matches))
			}
		})
	}
}

// --- --rankings --------------------------------------------------------

func TestEventWatchRankingsReportsMovers(t *testing.T) {
	srv := watchServer(t)
	out, errOut, err := runWatch(t, srv, func() {
		watchSetBody(t, srv, watchRankingsPath, watchRankingsV2())
	}, "--rankings", "--max-polls", "2")
	requireNoError(t, err, errOut)

	got := jsonLines(t, out)
	if _, ok := got[0]["rankings"]; !ok {
		t.Errorf("the snapshot should carry the rankings: %v", got[0])
	}
	if len(got) != 3 {
		t.Fatalf("want a snapshot and two moved teams, got %d:\n%s", len(got), out)
	}
	byTeam := map[string]map[string]any{}
	for _, line := range got[1:] {
		if line["type"] != "ranking" {
			t.Errorf("type = %v, want ranking", line["type"])
		}
		byTeam[line["team_key"].(string)] = line
	}
	climbed := byTeam["frc230"]
	if climbed == nil {
		t.Fatalf("230 climbed and was not reported: %v", byTeam)
	}
	if climbed["rank"] != float64(1) || climbed["previous_rank"] != float64(2) {
		t.Errorf("230 = %v, want rank 1 from 2", climbed)
	}
	if record, ok := climbed["record"].(map[string]any); !ok || record["wins"] != float64(2) {
		t.Errorf("230's record = %v", climbed["record"])
	}
}

// Without --rankings the rankings endpoint is never asked for.
func TestEventWatchLeavesRankingsAloneByDefault(t *testing.T) {
	srv := watchServer(t)
	_, errOut, err := runWatch(t, srv, nil, "--max-polls", "2")
	requireNoError(t, err, errOut)

	if contains(requestPaths(t, srv), watchRankingsPath) {
		t.Errorf("rankings were fetched without --rankings: %v", requestPaths(t, srv))
	}
}

func TestEventWatchRankingsTableFollowsTheMatches(t *testing.T) {
	srv := watchServer(t)
	out, errOut, err := runWatch(t, srv, func() {
		watchSetBody(t, srv, watchRankingsPath, watchRankingsV2())
	}, "--rankings", "--format", "table", "--max-polls", "2")
	requireNoError(t, err, errOut)

	if strings.Count(out, "Rank  Team  Record") != 2 {
		t.Errorf("want the standings once per change, got:\n%s", out)
	}
	matchTable := strings.Index(out, "Score (R-B)")
	standings := strings.Index(out, "Rank  Team  Record")
	if matchTable < 0 || standings < matchTable {
		t.Errorf("the standings should follow the matches:\n%s", out)
	}
	if !strings.Contains(out, "--- 14:32:07 (poll 2) ---") {
		t.Errorf("the second standings need a poll header:\n%s", out)
	}
}

// --- stopping ----------------------------------------------------------

func TestEventWatchStopsWhenForElapses(t *testing.T) {
	srv := watchServer(t)
	out, errOut, err := runWatch(t, srv, nil, "--for", "2h", "--interval", "1h")
	requireNoError(t, err, errOut)

	if errOut != "note: stopped after 2h0m0s (--for)\n" {
		t.Errorf("stderr = %q", errOut)
	}
	if strings.Contains(out, "note:") {
		t.Errorf("the stop reason belongs on stderr:\n%s", out)
	}
	if n := pollCount(t, srv); n != 2 {
		t.Errorf("polls = %d, want 2 within two hours at one an hour", n)
	}
}

// --for 0 means "until Ctrl-C", so it has to be asked for explicitly.
func TestEventWatchForZeroRunsUntilStopped(t *testing.T) {
	srv := watchServer(t)
	_, errOut, err := runWatch(t, srv, nil, "--for", "0", "--max-polls", "4")
	requireNoError(t, err, errOut)

	if !strings.Contains(errOut, "(--max-polls)") {
		t.Errorf("only --max-polls should have stopped this: %q", errOut)
	}
	if n := pollCount(t, srv); n != 4 {
		t.Errorf("polls = %d, want 4", n)
	}
}

func TestEventWatchStopsAfterMaxPolls(t *testing.T) {
	srv := watchServer(t)
	out, errOut, err := runWatch(t, srv, nil, "--max-polls", "3")
	requireNoError(t, err, errOut)

	if errOut != "note: stopped after 3 polls (--max-polls)\n" {
		t.Errorf("stderr = %q", errOut)
	}
	if n := pollCount(t, srv); n != 3 {
		t.Errorf("polls = %d, want 3", n)
	}
	if strings.Contains(out, "note:") {
		t.Errorf("the stop reason belongs on stderr:\n%s", out)
	}
}

// The first poll is immediate: a watch that waited a minute to say anything
// would look broken.
func TestEventWatchPollsImmediately(t *testing.T) {
	srv := watchServer(t)
	clock := fakeWatchClock(t, time.Date(2024, 3, 22, 14, 31, 7, 0, time.Local))
	_, errOut, err := runCmd(t, srv, "event", "watch", watchEventKey, "--max-polls", "2", "--interval", "30s")
	requireNoError(t, err, errOut)

	if got := clock.sleptFor(); len(got) != 1 || got[0] != 30*time.Second {
		t.Errorf("waits = %v, want one wait of 30s between two polls", got)
	}
}

// --- interruption ------------------------------------------------------

func TestEventWatchInterruptedWhileWaitingExits130(t *testing.T) {
	srv := watchServer(t)
	clock := fakeWatchClock(t, time.Date(2024, 3, 22, 14, 31, 7, 0, time.Local))
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	clock.onWait = func(*watchClock) { cancel() }

	_, errOut, err := runCtx(t, ctx, srv.URL, "event", "watch", watchEventKey)
	if got := clierr.ExitCode(err); got != clierr.ExitInterrupt {
		t.Fatalf("exit code = %d (%v), want %d", got, err, clierr.ExitInterrupt)
	}
	if strings.Contains(errOut, "note:") {
		t.Errorf("Ctrl-C is not a stop reason worth explaining: %q", errOut)
	}
}

// A poll cut short by Ctrl-C is the user leaving, not a network wobble: it must
// not be swallowed by the retry-at-next-interval path.
func TestEventWatchInterruptedMidPollExits130(t *testing.T) {
	srv := watchServer(t)
	fakeWatchClock(t, time.Date(2024, 3, 22, 14, 31, 7, 0, time.Local))
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, errOut, err := runCtx(t, ctx, srv.URL, "event", "watch", watchEventKey)
	if got := clierr.ExitCode(err); got != clierr.ExitInterrupt {
		t.Fatalf("exit code = %d (%v), want %d", got, err, clierr.ExitInterrupt)
	}
	if strings.Contains(errOut, "retrying at next interval") {
		t.Errorf("a cancelled poll is not a failed poll: %q", errOut)
	}
}

// --- failures ----------------------------------------------------------

func TestEventWatchKeepsGoingWhenAPollFails(t *testing.T) {
	srv := watchServer(t)
	setStatus(t, srv, watchMatchesPath, http.StatusBadGateway)
	out, errOut, err := runWatch(t, srv, func() {
		setStatus(t, srv, watchMatchesPath, 0)
	}, "--max-polls", "2", "--retries", "0")
	requireNoError(t, err, errOut)

	if !strings.Contains(errOut, "note: poll 1 failed: ") || !strings.Contains(errOut, "; retrying at next interval") {
		t.Errorf("stderr = %q", errOut)
	}
	if got := jsonLines(t, out); len(got) != 1 || got[0]["type"] != "snapshot" {
		t.Errorf("the first poll that worked is the snapshot, got:\n%s", out)
	}
}

func TestEventWatchGivesUpAfterFiveFailedPolls(t *testing.T) {
	srv := watchServer(t)
	setStatus(t, srv, watchMatchesPath, http.StatusBadGateway)

	_, errOut, err := runWatch(t, srv, nil, "--retries", "0")
	if err == nil {
		t.Fatal("want an error once the API stops answering")
	}
	if got := clierr.ExitCode(err); got != clierr.ExitFailure {
		t.Errorf("exit code = %d, want %d", got, clierr.ExitFailure)
	}
	requireErrorContains(t, err, "giving up after 5 polls in a row failed")
	if n := strings.Count(errOut, "retrying at next interval"); n != 4 {
		t.Errorf("notes = %d, want one for each failure that was retried", n)
	}
	if n := pollCount(t, srv); n != 5 {
		t.Errorf("polls = %d, want 5", n)
	}
}

// --- offline -----------------------------------------------------------

func TestEventWatchOfflineServesTheCacheOnceAndStops(t *testing.T) {
	sharedCacheDir(t)
	srv := watchServer(t)
	_, _, err := runCmd(t, srv, "event", "matches", watchEventKey)
	requireNoError(t, err, "")
	before := pollCount(t, srv)

	out, errOut, err := runWatch(t, srv, nil, "--offline")
	requireNoError(t, err, errOut)

	if !strings.Contains(errOut, "--offline") {
		t.Errorf("stderr should explain why the watch stopped: %q", errOut)
	}
	if got := jsonLines(t, out); len(got) != 1 || got[0]["type"] != "snapshot" {
		t.Errorf("offline should still show what it has, got:\n%s", out)
	}
	if got := pollCount(t, srv); got != before {
		t.Errorf("offline made %d request(s)", got-before)
	}
}

// Offline with nothing cached is a failure, not an empty watch.
func TestEventWatchOfflineWithAnEmptyCacheFails(t *testing.T) {
	srv := watchServer(t)
	_, errOut, err := runWatch(t, srv, nil, "--offline")
	if err == nil {
		t.Fatal("want an error when the cache has never seen this event")
	}
	if got := clierr.ExitCode(err); got != clierr.ExitFailure {
		t.Errorf("exit code = %d, want %d", got, clierr.ExitFailure)
	}
	if strings.Contains(errOut, "retrying at next interval") {
		t.Errorf("offline has no next interval to retry at: %q", errOut)
	}
}

// --- usage -------------------------------------------------------------

func TestEventWatchRejectsAnIntervalUnderTheMinimum(t *testing.T) {
	srv := watchServer(t)
	_, _, err := runCmd(t, srv, "event", "watch", watchEventKey, "--interval", "5s")
	if got := clierr.ExitCode(err); got != clierr.ExitUsage {
		t.Fatalf("exit code = %d (%v), want %d", got, err, clierr.ExitUsage)
	}
	requireErrorContains(t, err, "15s")
	if n := pollCount(t, srv); n != 0 {
		t.Errorf("a rejected interval should cost no requests, made %d", n)
	}
}

func TestEventWatchAcceptsTheMinimumInterval(t *testing.T) {
	srv := watchServer(t)
	_, errOut, err := runWatch(t, srv, nil, "--interval", "15s", "--max-polls", "1")
	requireNoError(t, err, errOut)
}

func TestEventWatchRejectsTabularFormats(t *testing.T) {
	for _, format := range []string{"csv", "tsv", "markdown"} {
		t.Run(format, func(t *testing.T) {
			srv := watchServer(t)
			_, _, err := runCmd(t, srv, "event", "watch", watchEventKey, "--format", format)
			if got := clierr.ExitCode(err); got != clierr.ExitUsage {
				t.Fatalf("exit code = %d (%v), want %d", got, err, clierr.ExitUsage)
			}
			requireErrorContains(t, err, "watch streams changes; use table or json")
			if n := pollCount(t, srv); n != 0 {
				t.Errorf("a rejected format should cost no requests, made %d", n)
			}
		})
	}
}

func TestEventWatchRejectsANegativeMaxPolls(t *testing.T) {
	srv := watchServer(t)
	_, _, err := runCmd(t, srv, "event", "watch", watchEventKey, "--max-polls", "-1")
	if got := clierr.ExitCode(err); got != clierr.ExitUsage {
		t.Errorf("exit code = %d (%v), want %d", got, err, clierr.ExitUsage)
	}
}

func TestEventWatchRejectsAMalformedEventKey(t *testing.T) {
	srv := watchServer(t)
	_, _, err := runCmd(t, srv, "event", "watch", "cthar2024")
	if got := clierr.ExitCode(err); got != clierr.ExitUsage {
		t.Errorf("exit code = %d (%v), want %d", got, err, clierr.ExitUsage)
	}
}

// --- --jq --------------------------------------------------------------

func TestEventWatchAppliesJqPerLine(t *testing.T) {
	srv := watchServer(t)
	out, errOut, err := runWatch(t, srv, func() {
		watchSetBody(t, srv, watchMatchesPath, watchMatchesQM2Played())
	}, "--max-polls", "2", "--jq", ".type", "-r")
	requireNoError(t, err, errOut)

	if got := lines(out); !equalStrings(got, []string{"snapshot", "match"}) {
		t.Errorf("lines = %v, want one per change", got)
	}
}

// A filter that selects nothing prints nothing, rather than an empty array.
func TestEventWatchJqCanSelectNothing(t *testing.T) {
	srv := watchServer(t)
	out, errOut, err := runWatch(t, srv, nil, "--max-polls", "2", "--jq", `select(.type=="match") | .key`, "-r")
	requireNoError(t, err, errOut)
	if out != "" {
		t.Errorf("stdout = %q, want nothing", out)
	}
}

// --- wiring ------------------------------------------------------------

func TestEventWatchIsRegisteredOnce(t *testing.T) {
	names := subcommandNames(t, "event")
	n := 0
	for _, name := range names {
		if name == "watch" {
			n++
		}
	}
	if n != 1 {
		t.Errorf("event watch appears %d times in %v", n, names)
	}
}

func TestEventWatchHelpLeadsWithTheDefaults(t *testing.T) {
	out, _, err := runCmd(t, nil, "event", "watch", "--help")
	requireNoError(t, err, "")
	for _, want := range []string{"(default 1m0s)", "(default 2h0m0s)", "minimum 15s"} {
		requireContains(t, out, want)
	}
}

// --- marks and the legend ----------------------------------------------

// watchMarked is V1 with a surrogate on qm2 and a disqualification on qm3, the
// two marks a row can carry.
func watchMarked() []api.Match {
	out := watchMatchesV1()
	out[1].Alliances["red"] = api.Alliance{
		Score: -1, TeamKeys: watchOther, SurrogateTeamKeys: []string{"frc2168"},
	}
	out[2].Alliances["blue"] = api.Alliance{
		Score: -1, TeamKeys: watchOther2, DQTeamKeys: []string{"frc6153"},
	}
	return out
}

// The legend explains the marks once for the whole table, not once per row:
// the first poll used to print it again for every marked match it built.
func TestEventWatchPrintsTheLegendOnceForTheFirstTable(t *testing.T) {
	fakeWatchClock(t, time.Date(2024, 3, 22, 14, 31, 7, 0, time.Local))
	srv := watchServer(t)
	watchSetBody(t, srv, watchMatchesPath, watchMarked())

	out, errOut, err := runCmd(t, srv, "event", "watch", watchEventKey, "--format", "table", "--max-polls", "1")
	requireNoError(t, err, errOut)

	if n := strings.Count(errOut, frc.Legend); n != 1 {
		t.Errorf("the legend appears %d times on stderr, want once:\n%s", n, errOut)
	}
	if strings.Contains(out, "surrogate") {
		t.Errorf("the legend must not be on stdout:\n%s", out)
	}
	requireContains(t, out, "2168*")
	requireContains(t, out, "6153!")
}

// A poll that brings in a mark explains it, and only the first one does.
func TestEventWatchPrintsTheLegendOnceWhenAPollIntroducesAMark(t *testing.T) {
	srv := watchServer(t)
	out, errOut, err := runWatch(t, srv, func() {
		marked := watchMarked()
		marked[1] = watchQual(2, watchOther, watchOther2, 101, 99, "red", 1711131000)
		marked[1].Alliances["red"] = api.Alliance{
			Score: 101, TeamKeys: watchOther, SurrogateTeamKeys: []string{"frc2168"},
		}
		watchSetBody(t, srv, watchMatchesPath, marked)
	}, "--format", "table", "--max-polls", "3")
	requireNoError(t, err, errOut)

	if n := strings.Count(errOut, frc.Legend); n != 1 {
		t.Errorf("the legend appears %d times on stderr, want once:\n%s", n, errOut)
	}
	requireContains(t, out, "2168*")
}

// No marks, nothing to explain.
func TestEventWatchOmitsTheLegendWithoutMarks(t *testing.T) {
	fakeWatchClock(t, time.Date(2024, 3, 22, 14, 31, 7, 0, time.Local))
	srv := watchServer(t)
	_, errOut, err := runCmd(t, srv, "event", "watch", watchEventKey, "--format", "table", "--max-polls", "1")
	requireNoError(t, err, errOut)
	if strings.Contains(errOut, "surrogate") {
		t.Errorf("stderr = %q, want no legend", errOut)
	}
}
