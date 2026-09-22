package cmd

import (
	"context"
	"time"

	"github.com/spf13/cobra"
)

// A clock is everything the commands do with time that is not simply
// formatting it: reading the current moment, and waiting.
//
// It is one value hung on the command tree rather than a package-level
// variable, because a variable tests reassign is shared mutable state: two
// tests running at once see each other's clock, and `go test -race` is right
// to complain. Here, a tree built by newRootCmdWithClock carries its own
// clock for as long as it runs, and two trees cannot interfere.
//
// Only `event watch` needs sleep — it is the one command that waits — but it
// belongs beside now for the same reason it always did: a test that pins the
// clock has to control both, or the loop advances instantly against a
// stopped clock and never finishes.
type clock struct {
	now   func() time.Time
	sleep func(ctx context.Context, d time.Duration) error
}

// systemClock is the real one: the wall clock, and a timer.
func systemClock() clock {
	return clock{now: time.Now, sleep: sleepFor}
}

// sleepFor waits d, or until the context is cancelled, so Ctrl-C does not have
// to wait out a poll interval.
func sleepFor(ctx context.Context, d time.Duration) error {
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

type clockCtxKey struct{}

// withClock hangs a clock on a context, the way initSettings hangs the
// settings there. The root's PersistentPreRunE does it once, before anything
// asks what time it is.
func withClock(ctx context.Context, c clock) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	return context.WithValue(ctx, clockCtxKey{}, c)
}

// clockOf returns the clock this invocation runs against, or the real one when
// the command was not run through the root's PersistentPreRunE — which is the
// case in a few tests that call a builder directly.
func clockOf(cmd *cobra.Command) clock {
	if ctx := cmd.Context(); ctx != nil {
		if c, ok := ctx.Value(clockCtxKey{}).(clock); ok {
			return c
		}
	}
	return systemClock()
}

// nowOf is the moment a command's answer is about.
//
// A command reads it once, at the top of its RunE, and hands it down: every
// countdown, relative time and "is this event over" in one answer then refers
// to the same instant, rather than to whenever each of them happened to look.
func nowOf(cmd *cobra.Command) time.Time {
	return clockOf(cmd).now()
}
