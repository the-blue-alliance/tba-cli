package cmd

import (
	"bytes"
	"context"
	"strings"
	"testing"
	"time"

	"github.com/spf13/cobra"
)

func TestClockOfFallsBackToTheWallClock(t *testing.T) {
	// A command nobody ran through the root's PersistentPreRunE still has to
	// be able to ask the time.
	cmd := &cobra.Command{Use: "orphan"}
	before := time.Now()
	got := nowOf(cmd)
	if got.Before(before.Add(-time.Minute)) || got.After(time.Now().Add(time.Minute)) {
		t.Errorf("nowOf = %v, want something near %v", got, before)
	}
	if clockOf(cmd).sleep == nil {
		t.Error("the fallback clock cannot sleep")
	}
}

func TestClockOfReadsTheTreesClock(t *testing.T) {
	pinned := time.Date(2024, 3, 22, 14, 31, 7, 0, time.UTC)
	var got time.Time
	root := newRootCmdWithClock(fixedClock(pinned))
	leaf := &cobra.Command{
		Use:  "whenisit",
		RunE: func(cmd *cobra.Command, _ []string) error { got = nowOf(cmd); return nil },
	}
	root.AddCommand(leaf)
	root.SetArgs([]string{"whenisit"})
	if err := Run(context.Background(), root); err != nil {
		t.Fatalf("running: %v", err)
	}
	if !got.Equal(pinned) {
		t.Errorf("nowOf = %v, want %v", got, pinned)
	}
}

// The point of hanging the clock on the tree rather than on the package: each
// invocation carries its own and leaves nothing behind for the next one, so
// two of them can run at once without a t.Cleanup race to put a global back.
func TestEachRunCarriesItsOwnClock(t *testing.T) {
	srv := newFakeTBA(t, map[string]any{"/match/2024cthar_qm12": matchViewQM12JSON})
	cases := []struct {
		at   time.Time
		want string
	}{
		{time.Unix(1711130820+7200, 0), "2h ago"},
		{time.Unix(1711130820+86400, 0), "1d ago"},
	}
	for _, tc := range cases {
		out, _, err := runCmdAt(t, srv, tc.at, "match", "view", "2024cthar_qm12", "--format", "table")
		requireNoError(t, err, "")
		if !strings.Contains(out, tc.want) {
			t.Errorf("run at %v did not say %q:\n%s", tc.at, tc.want, out)
		}
	}

	// And nothing of either run is left where the next one could read it.
	if got := nowOf(&cobra.Command{Use: "orphan"}); got.Before(time.Now().Add(-time.Minute)) {
		t.Errorf("a pinned clock outlived its command tree: %v", got)
	}
}

// Every match in one answer is measured against one instant, so a listing
// cannot say "in 18m" on one row and "in 17m" on another because the second
// row was built a minute later.
func TestAListingMeasuresEveryRowAgainstOneInstant(t *testing.T) {
	ticking := 0
	clk := clock{
		now: func() time.Time {
			ticking++
			return time.Unix(1711122000-1080, 0).Add(time.Duration(ticking) * time.Hour)
		},
		sleep: func(ctx context.Context, _ time.Duration) error { return ctx.Err() },
	}
	srv := eventMatchesServer(t)
	_, _, err := runCmdWith(t, srv, clk, &bytes.Buffer{}, "event", "matches", "2024cthar", "--format", "csv")
	requireNoError(t, err, "")

	// One read for the answer. resolveYear may take another, but the listing
	// itself must not read the clock once per row.
	if ticking > 2 {
		t.Errorf("the clock was read %d times for one listing", ticking)
	}
}

func TestFixedClockSleepsInstantly(t *testing.T) {
	clk := fixedClock(time.Unix(0, 0))
	start := time.Now()
	if err := clk.sleep(context.Background(), time.Hour); err != nil {
		t.Fatalf("sleep: %v", err)
	}
	if elapsed := time.Since(start); elapsed > time.Second {
		t.Errorf("a pinned clock waited %v of real time", elapsed)
	}
}

func TestSystemClockSleepStopsOnCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := systemClock().sleep(ctx, time.Hour); err == nil {
		t.Error("a cancelled sleep should report why it stopped")
	}
}
