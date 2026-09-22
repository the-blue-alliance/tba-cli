package frc_test

import (
	"testing"
	"time"

	"github.com/the-blue-alliance/tba-cli/internal/api"
	"github.com/the-blue-alliance/tba-cli/internal/frc"
)

func TestBestTimePrefersWhatActuallyHappened(t *testing.T) {
	m := mustMatch(t, qual2024JSON)
	got, source := frc.BestTime(m)
	if got == nil || *got != 1711130820 {
		t.Fatalf("epoch = %v, want the actual time", got)
	}
	if source != frc.TimeActual {
		t.Errorf("source = %q, want %q", source, frc.TimeActual)
	}
}

func TestBestTimeFallsBackToThePrediction(t *testing.T) {
	m := mustMatch(t, unplayed2024JSON)
	got, source := frc.BestTime(m)
	if got == nil || *got != 1711221000 {
		t.Fatalf("epoch = %v, want the predicted time", got)
	}
	if source != frc.TimePredicted {
		t.Errorf("source = %q, want %q", source, frc.TimePredicted)
	}
}

// A 2015 match predates predicted times, so only the schedule is on offer.
func TestBestTimeFallsBackToTheSchedule(t *testing.T) {
	m := mustMatch(t, qual2015TieJSON)
	got, source := frc.BestTime(m)
	if got == nil || *got != 1427464800 {
		t.Fatalf("epoch = %v, want the scheduled time", got)
	}
	if source != frc.TimeScheduled {
		t.Errorf("source = %q, want %q", source, frc.TimeScheduled)
	}
}

func TestBestTimeWithNoTimeAtAll(t *testing.T) {
	got, source := frc.BestTime(api.Match{Key: "2024cthar_qm1"})
	if got != nil || source != "" {
		t.Errorf("BestTime = (%v, %q), want (nil, \"\")", got, source)
	}
}

// The API uses 0 for an unset time; 1970 is never the answer.
func TestBestTimeTreatsZeroAsAbsent(t *testing.T) {
	m := api.Match{ActualTime: epoch(0), PredictedTime: epoch(0), Time: epoch(1711130400)}
	got, source := frc.BestTime(m)
	if got == nil || *got != 1711130400 || source != frc.TimeScheduled {
		t.Errorf("BestTime = (%v, %q), want the scheduled time", got, source)
	}
}

func TestFormatTime(t *testing.T) {
	// 2024-03-22 18:00:00 UTC was a Friday.
	if got := frc.FormatTime(epoch(1711130400), time.UTC, false); got != "Fri 18:00" {
		t.Errorf("FormatTime = %q, want %q", got, "Fri 18:00")
	}
}

func TestFormatTimeRespectsTheLocation(t *testing.T) {
	loc, err := time.LoadLocation("America/New_York")
	if err != nil {
		t.Skipf("no tzdata available: %v", err)
	}
	if got := frc.FormatTime(epoch(1711130400), loc, false); got != "Fri 14:00" {
		t.Errorf("FormatTime = %q, want %q", got, "Fri 14:00")
	}
}

func TestFormatTimeWithoutATime(t *testing.T) {
	if got := frc.FormatTime(nil, time.UTC, false); got != "" {
		t.Errorf("FormatTime(nil) = %q, want %q", got, "")
	}
	if got := frc.FormatTime(epoch(0), time.UTC, false); got != "" {
		t.Errorf("FormatTime(0) = %q, want %q", got, "")
	}
}

// A nil location means the machine's own, which is what someone at the event
// reads off their phone.
func TestFormatTimeDefaultsToLocal(t *testing.T) {
	want := time.Unix(1711130400, 0).In(time.Local).Format(frc.TimeLayout)
	if got := frc.FormatTime(epoch(1711130400), nil, false); got != want {
		t.Errorf("FormatTime = %q, want %q", got, want)
	}
}

func TestRelative(t *testing.T) {
	now := time.Date(2024, 3, 22, 14, 0, 0, 0, time.UTC)
	cases := []struct {
		name string
		at   time.Time
		want string
	}{
		{"in seconds", now.Add(45 * time.Second), "in 45s"},
		{"in minutes", now.Add(18 * time.Minute), "in 18m"},
		{"minutes truncate", now.Add(18*time.Minute + 59*time.Second), "in 18m"},
		{"in hours", now.Add(3 * time.Hour), "in 3h"},
		{"in days", now.Add(50 * time.Hour), "in 2d"},
		{"hours ago", now.Add(-2 * time.Hour), "2h ago"},
		{"minutes ago", now.Add(-90 * time.Second), "1m ago"},
		{"days ago", now.Add(-72 * time.Hour), "3d ago"},
		{"right now", now, "in 0s"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := frc.Relative(c.at, now); got != c.want {
				t.Errorf("Relative = %q, want %q", got, c.want)
			}
		})
	}
}

// One second short of an hour is still counted in minutes, so the units never
// skip.
func TestRelativeUnitBoundaries(t *testing.T) {
	now := time.Date(2024, 3, 22, 14, 0, 0, 0, time.UTC)
	cases := []struct {
		d    time.Duration
		want string
	}{
		{time.Minute - time.Second, "in 59s"},
		{time.Minute, "in 1m"},
		{time.Hour - time.Second, "in 59m"},
		{time.Hour, "in 1h"},
		{24*time.Hour - time.Second, "in 23h"},
		{24 * time.Hour, "in 1d"},
	}
	for _, c := range cases {
		if got := frc.Relative(now.Add(c.d), now); got != c.want {
			t.Errorf("Relative(+%s) = %q, want %q", c.d, got, c.want)
		}
	}
}

func TestRelativeEpoch(t *testing.T) {
	now := time.Unix(1711130400, 0)
	if got := frc.RelativeEpoch(epoch(1711130400+600), now); got != "in 10m" {
		t.Errorf("RelativeEpoch = %q, want %q", got, "in 10m")
	}
	if got := frc.RelativeEpoch(nil, now); got != "" {
		t.Errorf("RelativeEpoch(nil) = %q, want %q", got, "")
	}
	if got := frc.RelativeEpoch(epoch(0), now); got != "" {
		t.Errorf("RelativeEpoch(0) = %q, want %q", got, "")
	}
}

// A listing that spans more than one day writes the date instead of the
// weekday, because "Sat 11:22" could be any Saturday of the season.
func TestFormatTimeWithTheDate(t *testing.T) {
	if got := frc.FormatTime(epoch(1711130400), time.UTC, true); got != "Mar 22 18:00" {
		t.Errorf("FormatTime = %q, want %q", got, "Mar 22 18:00")
	}
	if got := frc.FormatTime(nil, time.UTC, true); got != "" {
		t.Errorf("FormatTime(nil) = %q, want %q", got, "")
	}
}

// One competition day needs no date on every row.
func TestSpansDaysWithinOneDay(t *testing.T) {
	matches := []api.Match{
		{ActualTime: epoch(1711130400)}, // 2024-03-22 18:00 UTC
		{ActualTime: epoch(1711141200)}, // 2024-03-22 21:00 UTC
	}
	if frc.SpansDays(matches, time.UTC) {
		t.Error("SpansDays = true, want false for two times on 2024-03-22 UTC")
	}
}

func TestSpansDaysAcrossDays(t *testing.T) {
	matches := []api.Match{
		{ActualTime: epoch(1711130400)}, // 2024-03-22 UTC
		{ActualTime: epoch(1711299600)}, // 2024-03-24 UTC
	}
	if !frc.SpansDays(matches, time.UTC) {
		t.Error("SpansDays = false, want true for times two days apart")
	}
}

// A schedule that has not been published says nothing about how many days the
// listing covers, so timeless matches are ignored rather than counted as a
// second day.
func TestSpansDaysIgnoresMatchesWithoutATime(t *testing.T) {
	matches := []api.Match{
		{ActualTime: epoch(1711130400)},
		{},
		{PredictedTime: epoch(0)},
	}
	if frc.SpansDays(matches, time.UTC) {
		t.Error("SpansDays = true, want false when only one match has a time")
	}
	if frc.SpansDays(nil, time.UTC) {
		t.Error("SpansDays(nil) = true, want false")
	}
}

// The days are counted where the reader is, not in UTC: two matches either
// side of midnight UTC are one evening in Hartford.
func TestSpansDaysUsesTheGivenLocation(t *testing.T) {
	loc, err := time.LoadLocation("America/New_York")
	if err != nil {
		t.Skipf("no tzdata available: %v", err)
	}
	matches := []api.Match{
		{ActualTime: epoch(1711148400)}, // 2024-03-22 19:00 EDT
		{ActualTime: epoch(1711155600)}, // 2024-03-22 21:00 EDT
	}
	if frc.SpansDays(matches, loc) {
		t.Error("SpansDays = true, want false for one evening in New York")
	}
	if !frc.SpansDays(matches, time.UTC) {
		t.Error("SpansDays = false in UTC, where those times straddle midnight")
	}
}
