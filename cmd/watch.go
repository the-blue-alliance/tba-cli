package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/itchyny/gojq"
	"github.com/spf13/cobra"

	"github.com/the-blue-alliance/tba-cli/internal/api"
	"github.com/the-blue-alliance/tba-cli/internal/clierr"
	"github.com/the-blue-alliance/tba-cli/internal/frc"
	"github.com/the-blue-alliance/tba-cli/internal/output"
)

// Watch's defaults are deliberately timid. Every other command makes one round
// of requests and stops; this one keeps asking, on a public API somebody else
// pays for, for as long as it is left running. So the interval is a minute, it
// gives up after two hours unless told otherwise, and the shortest interval it
// will accept is fifteen seconds.
const (
	defaultWatchInterval = 60 * time.Second
	minWatchInterval     = 15 * time.Second
	defaultWatchFor      = 2 * time.Hour

	// watchFailureLimit is how many polls in a row may fail before the command
	// gives up. A single failure is weather; five in a row is a broken link or
	// a wrong event key, and sitting there quietly retrying helps nobody.
	watchFailureLimit = 5

	// watchRescheduleThreshold is how far a predicted time has to move before
	// it counts as news. The queue's estimate jitters by seconds all day long.
	watchRescheduleThreshold = 60 * time.Second
)

// The kinds of change reported for a match, as the `change` field of a JSON
// line.
const (
	watchChangeAdded       = "added"
	watchChangePlayed      = "played"
	watchChangeScore       = "score"
	watchChangeWinner      = "winner"
	watchChangeRescheduled = "rescheduled"
)

// watchSleep waits d, or until the context is cancelled. It is a variable so
// that tests can drive the loop instantly, advancing a pinned nowFunc instead
// of the wall clock.
var watchSleep = func(ctx context.Context, d time.Duration) error {
	if d <= 0 {
		return ctx.Err()
	}
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-t.C:
		return nil
	}
}

func newEventWatchCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "watch <key>",
		Short: "Follow an event's matches as they are played",
		Long: `Poll an event and report what changed since the last look.

Output is append-only: nothing is ever cleared or redrawn, so the stream can be
piped, teed into a file or scrolled back through. The first poll prints the
whole match table; every later poll prints only the rows that changed, under a
line naming the time and the poll number. Column widths are fixed by the first
table so the rows below it stay in line.

Piped, or with --format json, each change is one JSON object on its own line
(JSON Lines), starting with a snapshot of everything as it stood at the first
poll. --jq is applied to each line as it is written. csv, tsv and markdown
describe a finished table and have nothing to say about a stream, so they are
refused. --columns and --sort do not apply either, and are ignored.

Because it polls, the defaults are conservative: a poll a minute for two hours.
--for 0 watches until Ctrl-C. Unchanged polls cost a conditional request the
API answers with a 304, so a long watch is cheaper than it looks.`,
		Example: `  tba event watch 2024cthar
  tba event watch 2024cthar --interval 30s --for 6h
  tba event watch 2024cthar --team 177 --rankings
  tba event watch 2024cthar --format json --jq 'select(.type=="match") | .key' -r`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runEventWatch(cmd, args[0])
		},
	}
	c.Flags().Duration("interval", defaultWatchInterval, "Time between polls (minimum 15s)")
	c.Flags().Duration("for", defaultWatchFor, "Stop after this long; 0 watches until Ctrl-C")
	c.Flags().Int("max-polls", 0, "Stop after this many polls; 0 is unlimited within --for")
	c.Flags().Bool("rankings", false, "Also watch the rankings (default: matches only)")
	c.Flags().String("team", "", "Only report matches this team plays in (e.g. 177 or frc177)")
	return c
}

// watchOptions is one invocation's settled configuration.
type watchOptions struct {
	eventKey string
	interval time.Duration
	forFor   time.Duration
	maxPolls int
	rankings bool
	teamKey  string
	offline  bool
	format   string
}

// watchOptionsFrom validates the flags before a single request goes out, so a
// mistake costs nothing.
func watchOptionsFrom(cmd *cobra.Command, key string) (watchOptions, error) {
	if err := validateEventKey(key); err != nil {
		return watchOptions{}, err
	}
	format, err := resolveFormat(cmd)
	if err != nil {
		return watchOptions{}, err
	}
	if format != "table" && format != "json" {
		return watchOptions{}, clierr.Usage("--format %s: watch streams changes; use table or json", format)
	}

	opts := watchOptions{eventKey: key, format: format, offline: settings(cmd).Bool("offline")}
	opts.interval, _ = cmd.Flags().GetDuration("interval")
	if opts.interval < minWatchInterval {
		return watchOptions{}, clierr.Usage(
			"--interval %s is too short: watch polls a shared API, so the minimum is %s",
			opts.interval, minWatchInterval)
	}
	opts.forFor, _ = cmd.Flags().GetDuration("for")
	if opts.forFor < 0 {
		return watchOptions{}, clierr.Usage("--for cannot be negative (0 watches until Ctrl-C)")
	}
	opts.maxPolls, _ = cmd.Flags().GetInt("max-polls")
	if opts.maxPolls < 0 {
		return watchOptions{}, clierr.Usage("--max-polls cannot be negative (0 is unlimited)")
	}
	opts.rankings, _ = cmd.Flags().GetBool("rankings")
	if team, _ := cmd.Flags().GetString("team"); strings.TrimSpace(team) != "" {
		opts.teamKey = teamKey(team)
	}
	return opts, nil
}

func runEventWatch(cmd *cobra.Command, key string) error {
	opts, err := watchOptionsFrom(cmd, key)
	if err != nil {
		return err
	}
	client, err := newClient(cmd)
	if err != nil {
		return err
	}
	ctx := cmd.Context()
	if ctx == nil {
		ctx = context.Background()
	}
	sink, err := newWatchSink(cmd, opts, client)
	if err != nil {
		return err
	}

	errw := cmd.ErrOrStderr()
	state := &watchState{}
	start := nowFunc()

	// A zero --for means "until Ctrl-C", which is a deadline of never.
	var deadline time.Time
	if opts.forFor > 0 {
		deadline = start.Add(opts.forFor)
	}
	expired := func() bool { return !deadline.IsZero() && !nowFunc().Before(deadline) }
	stopFor := func() error {
		fmt.Fprintf(errw, "note: stopped after %s (--for)\n", opts.forFor)
		return nil
	}

	failures := 0
	for poll := 1; ; poll++ {
		matches, rankings, err := watchPoll(ctx, client, opts)
		switch {
		case err != nil && ctx.Err() != nil:
			// Ctrl-C landed mid-request. That is not a poll that failed, it is
			// the user leaving: report it so the exit code says 130.
			return err
		case err != nil && opts.offline:
			// There is no next interval to retry at: the cache has already
			// said everything it has to say.
			return err
		case err != nil:
			failures++
			if failures >= watchFailureLimit {
				return fmt.Errorf("giving up after %d polls in a row failed; last failure: %w", failures, err)
			}
			fmt.Fprintf(errw, "note: poll %d failed: %v; retrying at next interval\n", poll, err)
		default:
			failures = 0
			if err := state.emit(sink, poll, nowFunc(), matches, rankings); err != nil {
				return err
			}
		}

		switch {
		case opts.offline:
			// Offline there is no second opinion to be had: the cache says
			// what it says, and polling it again would print the same thing
			// forever.
			fmt.Fprintln(errw, "note: --offline answered from the cache once; drop --offline to watch for changes")
			return nil
		case opts.maxPolls > 0 && poll >= opts.maxPolls:
			fmt.Fprintf(errw, "note: stopped after %d polls (--max-polls)\n", opts.maxPolls)
			return nil
		case expired():
			return stopFor()
		}

		wait := opts.interval
		if !deadline.IsZero() {
			if remaining := deadline.Sub(nowFunc()); remaining < wait {
				wait = remaining
			}
		}
		if err := watchSleep(ctx, wait); err != nil {
			return err
		}
		if expired() {
			return stopFor()
		}
	}
}

// watchPoll fetches one round. Both requests have to succeed for the round to
// count: half a poll would report matches against rankings from a minute ago.
func watchPoll(ctx context.Context, client *api.Client, opts watchOptions) ([]api.Match, *api.EventRankings, error) {
	var matches []api.Match
	if err := client.Get(ctx, fmt.Sprintf("/event/%s/matches", opts.eventKey), &matches); err != nil {
		return nil, nil, err
	}
	if opts.teamKey != "" {
		kept := make([]api.Match, 0, len(matches))
		for _, m := range matches {
			if frc.HasTeam(m, opts.teamKey) {
				kept = append(kept, m)
			}
		}
		matches = kept
	}
	frc.SortMatches(matches)

	if !opts.rankings {
		return matches, nil, nil
	}
	var rankings api.EventRankings
	if err := client.Get(ctx, fmt.Sprintf("/event/%s/rankings", opts.eventKey), &rankings); err != nil {
		return nil, nil, err
	}
	sort.SliceStable(rankings.Rankings, func(i, j int) bool {
		return rankings.Rankings[i].Rank < rankings.Rankings[j].Rank
	})
	return matches, &rankings, nil
}

// watchState remembers the last poll, which is the only thing a change can be
// measured against.
type watchState struct {
	started bool
	matches map[string]api.Match
	ranks   map[string]api.Ranking
}

// matchChange is one match that is not what it was.
type matchChange struct {
	change string
	match  api.Match
}

// rankingChange is one team whose rank or record moved.
type rankingChange struct {
	ranking      api.Ranking
	previousRank int
}

// watchNews is everything one poll found worth reporting.
type watchNews struct {
	poll     int
	at       time.Time
	matches  []matchChange
	rankings []rankingChange
	// standings is the whole ranking table as of this poll. The table sink
	// reprints it when anything in it moved, because one team climbing pushes
	// every team it passed down and a handful of loose rows would not add up.
	standings *api.EventRankings
}

// A watchSink renders one poll's news. The two implementations are the append-
// only table and the JSON Lines stream.
type watchSink interface {
	snapshot(poll int, at time.Time, matches []api.Match, rankings *api.EventRankings) error
	changes(news watchNews) error
}

func (s *watchState) emit(sink watchSink, poll int, at time.Time, matches []api.Match, rankings *api.EventRankings) error {
	if !s.started {
		s.started = true
		s.record(matches, rankings)
		return sink.snapshot(poll, at, matches, rankings)
	}
	news := watchNews{
		poll:      poll,
		at:        at,
		matches:   diffMatches(s.matches, matches),
		rankings:  diffRankings(s.ranks, rankings),
		standings: rankings,
	}
	s.record(matches, rankings)
	if len(news.matches) == 0 && len(news.rankings) == 0 {
		return nil
	}
	return sink.changes(news)
}

func (s *watchState) record(matches []api.Match, rankings *api.EventRankings) {
	s.matches = make(map[string]api.Match, len(matches))
	for _, m := range matches {
		s.matches[m.Key] = m
	}
	if rankings == nil {
		return
	}
	s.ranks = make(map[string]api.Ranking, len(rankings.Rankings))
	for _, r := range rankings.Rankings {
		s.ranks[r.TeamKey] = r
	}
}

// diffMatches reports what changed, in playing order (the order it is given).
func diffMatches(previous map[string]api.Match, matches []api.Match) []matchChange {
	var out []matchChange
	for _, m := range matches {
		old, existed := previous[m.Key]
		if change := matchChangeKind(old, m, existed); change != "" {
			out = append(out, matchChange{change: change, match: m})
		}
	}
	return out
}

// matchChangeKind names the most newsworthy difference between two versions of
// a match, or "" when nothing worth reporting moved.
//
// One change is reported per match per poll: a result arriving changes the
// score, the winner and the status all at once, and "played" is what happened.
// A match disappearing from the schedule is not reported, because the API
// dropping a row is far more likely to be a hiccup than a cancellation.
//
// actual_time is deliberately not compared. It is stamped and re-stamped as
// scoring is finalised, and a row that says only "this match still happened
// when it happened" is noise.
func matchChangeKind(old, now api.Match, existed bool) string {
	switch {
	case !existed:
		return watchChangeAdded
	case !frc.Played(old) && frc.Played(now):
		return watchChangePlayed
	case frc.Score(old, frc.AllianceRed) != frc.Score(now, frc.AllianceRed),
		frc.Score(old, frc.AllianceBlue) != frc.Score(now, frc.AllianceBlue):
		return watchChangeScore
	case frc.Winner(old) != frc.Winner(now):
		return watchChangeWinner
	case movedBy(old.PredictedTime, now.PredictedTime, watchRescheduleThreshold):
		return watchChangeRescheduled
	}
	return ""
}

// movedBy reports whether two optional timestamps are at least d apart. A time
// that only appears or only disappears is not a move: the match was not
// rescheduled, the API simply started or stopped guessing.
func movedBy(old, now *int64, d time.Duration) bool {
	if old == nil || now == nil || *old == 0 || *now == 0 {
		return false
	}
	delta := *now - *old
	if delta < 0 {
		delta = -delta
	}
	return time.Duration(delta)*time.Second >= d
}

// diffRankings reports the teams whose rank or record changed, in rank order.
func diffRankings(previous map[string]api.Ranking, rankings *api.EventRankings) []rankingChange {
	if rankings == nil {
		return nil
	}
	var out []rankingChange
	for _, r := range rankings.Rankings {
		old, existed := previous[r.TeamKey]
		if existed && old.Rank == r.Rank && sameRecord(old.Record, r.Record) {
			continue
		}
		// A team seen for the first time has no previous rank, which is
		// reported as 0 rather than invented.
		out = append(out, rankingChange{ranking: r, previousRank: old.Rank})
	}
	return out
}

func sameRecord(a, b *api.WLTRecord) bool {
	if a == nil || b == nil {
		return a == b
	}
	return *a == *b
}

func newWatchSink(cmd *cobra.Command, opts watchOptions, client *api.Client) (watchSink, error) {
	if opts.format == "json" {
		return newWatchJSONSink(cmd)
	}
	color, err := tableColorEnabled(cmd, opts.format)
	if err != nil {
		return nil, err
	}
	mode, err := colorMode(cmd)
	if err != nil {
		return nil, err
	}
	return &watchTableSink{
		out:   cmd.OutOrStdout(),
		errw:  cmd.ErrOrStderr(),
		color: color,
		mode:  mode,
		// The event is fetched once, only for the bracket format its playoff
		// match labels depend on; a failure leaves the labels to guess.
		playoffType: eventPlayoffType(cmd, client, opts.eventKey),
	}, nil
}

// watchJSONSink writes JSON Lines: one self-contained object per line, never a
// growing array, so a reader can act on each change as it arrives.
type watchJSONSink struct {
	out io.Writer
	jq  *gojq.Query
	raw bool
}

func newWatchJSONSink(cmd *cobra.Command) (*watchJSONSink, error) {
	s := &watchJSONSink{out: cmd.OutOrStdout(), raw: rawOutput(cmd)}
	if expr := jqExpr(cmd); expr != "" {
		query, err := gojq.Parse(expr)
		if err != nil {
			return nil, fmt.Errorf("invalid jq expression: %w", err)
		}
		s.jq = query
	}
	return s, nil
}

// watchHeader is the part every line carries: when it was observed and which
// poll saw it.
type watchHeader struct {
	TS   string `json:"ts"`
	Poll int    `json:"poll"`
	Type string `json:"type"`
}

type watchSnapshotLine struct {
	watchHeader
	Matches  []api.Match        `json:"matches"`
	Rankings *api.EventRankings `json:"rankings,omitempty"`
}

type watchMatchLine struct {
	watchHeader
	Key    string    `json:"key"`
	Change string    `json:"change"`
	Match  api.Match `json:"match"`
}

type watchRankingLine struct {
	watchHeader
	TeamKey      string         `json:"team_key"`
	Rank         int            `json:"rank"`
	PreviousRank int            `json:"previous_rank"`
	Record       *api.WLTRecord `json:"record"`
}

func watchHeaderAt(at time.Time, poll int, kind string) watchHeader {
	return watchHeader{TS: at.Format(time.RFC3339), Poll: poll, Type: kind}
}

func (s *watchJSONSink) snapshot(poll int, at time.Time, matches []api.Match, rankings *api.EventRankings) error {
	if matches == nil {
		matches = []api.Match{}
	}
	return s.write(watchSnapshotLine{
		watchHeader: watchHeaderAt(at, poll, "snapshot"),
		Matches:     matches,
		Rankings:    rankings,
	})
}

func (s *watchJSONSink) changes(news watchNews) error {
	for _, c := range news.matches {
		if err := s.write(watchMatchLine{
			watchHeader: watchHeaderAt(news.at, news.poll, "match"),
			Key:         c.match.Key,
			Change:      c.change,
			Match:       c.match,
		}); err != nil {
			return err
		}
	}
	for _, c := range news.rankings {
		if err := s.write(watchRankingLine{
			watchHeader:  watchHeaderAt(news.at, news.poll, "ranking"),
			TeamKey:      c.ranking.TeamKey,
			Rank:         c.ranking.Rank,
			PreviousRank: c.previousRank,
			Record:       c.ranking.Record,
		}); err != nil {
			return err
		}
	}
	return nil
}

// write emits one line and pushes it out, so that a reader waiting on the
// stream sees a change when it happens rather than when a buffer fills.
func (s *watchJSONSink) write(v any) error {
	b, err := json.Marshal(v)
	if err != nil {
		return err
	}
	if s.jq == nil {
		if _, err := fmt.Fprintln(s.out, string(b)); err != nil {
			return err
		}
		return s.flush()
	}

	// --jq applies per line: the filter sees one object, not the stream, so an
	// expression written for a single change works unchanged all day.
	var doc any
	if err := json.Unmarshal(b, &doc); err != nil {
		return err
	}
	iter := s.jq.Run(doc)
	for {
		val, ok := iter.Next()
		if !ok {
			break
		}
		if err, ok := val.(error); ok {
			return err
		}
		if str, ok := val.(string); ok && s.raw {
			if _, err := fmt.Fprintln(s.out, str); err != nil {
				return err
			}
			continue
		}
		out, err := json.Marshal(val)
		if err != nil {
			return err
		}
		if _, err := fmt.Fprintln(s.out, string(out)); err != nil {
			return err
		}
	}
	return s.flush()
}

func (s *watchJSONSink) flush() error {
	if f, ok := s.out.(interface{ Flush() error }); ok {
		return f.Flush()
	}
	return nil
}

// watchTableSink writes the append-only table: the whole thing once, then the
// rows that changed, padded to the widths the first table settled on.
type watchTableSink struct {
	out         io.Writer
	errw        io.Writer
	color       bool
	mode        output.ColorMode
	playoffType *int

	matchWidths []int
	rankHeaders []string
	rankWidths  []int
	legendShown bool
	// withDate is settled by the first poll, so that the rows printed later
	// write their times the same way as the table above them.
	withDate bool
}

func (s *watchTableSink) snapshot(_ int, _ time.Time, matches []api.Match, rankings *api.EventRankings) error {
	s.withDate = frc.SpansDays(matches, time.Local)
	rows := s.matchRows(matches)
	s.matchWidths = watchWidths(matchHeaders, rows)
	if err := output.Render(s.out, output.Table{Headers: matchHeaders, Rows: rows},
		output.RenderOptions{Format: "table", Color: s.mode}); err != nil {
		return err
	}
	if rankings != nil {
		fmt.Fprintln(s.out)
		if err := s.renderRankings(rankings); err != nil {
			return err
		}
	}
	return nil
}

func (s *watchTableSink) changes(news watchNews) error {
	// The header is data in table mode: it is what tells one poll's rows from
	// the next when the stream is read back later.
	fmt.Fprintf(s.out, "--- %s (poll %d) ---\n", news.at.Local().Format("15:04:05"), news.poll)
	for _, c := range news.matches {
		fmt.Fprintln(s.out, watchPadded(s.matchRow(c.match), s.matchWidths))
	}
	if len(news.rankings) > 0 && news.standings != nil {
		fmt.Fprintln(s.out)
		if err := s.renderRankings(news.standings); err != nil {
			return err
		}
	}
	return nil
}

func (s *watchTableSink) matchRows(matches []api.Match) [][]string {
	rows := make([][]string, len(matches))
	for i, m := range matches {
		rows[i] = s.matchRow(m)
	}
	return rows
}

// matchRow renders one match as a row of matchHeaders.
//
// This is a copy of the row `event matches` builds in cmd/matches.go, kept here
// so that a streaming command cannot be broken by a change to a listing
// command, and vice versa. The headers themselves are shared (matchHeaders),
// so the two tables cannot silently grow different columns.
func (s *watchTableSink) matchRow(m api.Match) []string {
	red, blue := m.Alliances[frc.AllianceRed], m.Alliances[frc.AllianceBlue]
	redCell := strings.Join(frc.MarkedTeams(red), ", ")
	blueCell := strings.Join(frc.MarkedTeams(blue), ", ")
	if !s.legendShown && strings.ContainsAny(redCell+blueCell, frc.MarkChars) {
		s.legendShown = true
		fmt.Fprintln(s.errw, frc.Legend)
	}

	score := ""
	if frc.Played(m) {
		score = fmt.Sprintf("%d-%d", red.Score, blue.Score)
	}
	epoch, source := frc.BestTime(m)
	when := ""
	if !frc.Played(m) {
		when = frc.RelativeEpoch(epoch, nowFunc())
	}
	return []string{
		frc.MatchLabel(m, s.playoffType),
		m.Key,
		output.Colorize(redCell, output.Red, s.color),
		output.Colorize(blueCell, output.Blue, s.color),
		score,
		colorizeAlliance(frc.Winner(m), s.color),
		frc.FormatTime(epoch, time.Local, s.withDate),
		when,
		source,
		frc.MatchStatus(m),
	}
}

// renderRankings prints the whole standings table, since a rank changing moves
// every team it passed and a handful of loose rows would not add up.
//
// The columns are the ranking table's without the nickname, which `event
// rankings` pays a second request for. A command that polls should not pay it
// again every minute for a column that never changes.
func (s *watchTableSink) renderRankings(rankings *api.EventRankings) error {
	headers, rows := watchRankingTable(rankings)
	if s.rankWidths == nil {
		s.rankHeaders, s.rankWidths = headers, watchWidths(headers, rows)
		return output.Render(s.out, output.Table{Headers: headers, Rows: rows},
			output.RenderOptions{Format: "table", Color: s.mode})
	}
	// Later tables reuse the first one's headers and widths, so the standings
	// line up down the whole stream.
	fmt.Fprintln(s.out, output.Colorize(watchPadded(s.rankHeaders, s.rankWidths), output.Bold, s.color))
	seps := make([]string, len(s.rankWidths))
	for i, w := range s.rankWidths {
		seps[i] = strings.Repeat("-", w)
	}
	fmt.Fprintln(s.out, strings.Join(seps, "  "))
	for _, row := range rows {
		fmt.Fprintln(s.out, watchPadded(row, s.rankWidths))
	}
	return nil
}

func watchRankingTable(rankings *api.EventRankings) ([]string, [][]string) {
	headers := []string{"Rank", "Team", "Record", "Played", "DQ"}
	for _, info := range rankings.SortOrderInfo {
		headers = append(headers, info.Name)
	}
	for _, info := range rankings.ExtraStatsInfo {
		headers = append(headers, info.Name)
	}
	rows := make([][]string, len(rankings.Rankings))
	for i, r := range rankings.Rankings {
		row := []string{
			strconv.Itoa(r.Rank),
			output.TeamNumberFromKey(r.TeamKey),
			formatWLT(r.Record),
			strconv.Itoa(r.MatchesPlayed),
			strconv.Itoa(r.DQ),
		}
		row = append(row, formatStats(r.SortOrders, rankings.SortOrderInfo)...)
		row = append(row, formatStats(r.ExtraStats, rankings.ExtraStatsInfo)...)
		rows[i] = row
	}
	return headers, rows
}

// watchWidths measures a table the way the text renderer does, so that the
// rows printed later line up under the header printed first.
func watchWidths(headers []string, rows [][]string) []int {
	widths := make([]int, len(headers))
	for i, h := range headers {
		widths[i] = output.StringWidth(h)
	}
	for _, row := range rows {
		for i, cell := range row {
			if i < len(widths) {
				if n := output.StringWidth(cell); n > widths[i] {
					widths[i] = n
				}
			}
		}
	}
	return widths
}

// watchPadded joins cells at fixed widths. A cell wider than its column is
// printed whole and pushes the rest of its row right: losing data to keep a
// column straight is the wrong trade when the value is the news.
func watchPadded(cells []string, widths []int) string {
	var b strings.Builder
	for i, cell := range cells {
		if i > 0 {
			b.WriteString("  ")
		}
		b.WriteString(cell)
		if i < len(widths) {
			if n := widths[i] - output.StringWidth(cell); n > 0 {
				b.WriteString(strings.Repeat(" ", n))
			}
		}
	}
	return b.String()
}
