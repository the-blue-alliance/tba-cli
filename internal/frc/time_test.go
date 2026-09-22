package frc_test

import (
	"testing"
	"time"

	"github.com/the-blue-alliance/tba-cli/internal/api"
	"github.com/the-blue-alliance/tba-cli/internal/frc"
)

// sameYear is a clock in 2024, the year of the timestamps below, so that the
// cases about the weekday and the date are not also cases about the year.
var sameYear = time.Date(2024, 6, 1, 12, 0, 0, 0, time.UTC)

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
	if got := frc.FormatTime(epoch(1711130400), time.UTC, false, sameYear); got != "Fri 18:00" {
		t.Errorf("FormatTime = %q, want %q", got, "Fri 18:00")
	}
}

func TestFormatTimeRespectsTheLocation(t *testing.T) {
	loc, err := time.LoadLocation("America/New_York")
	if err != nil {
		t.Skipf("no tzdata available: %v", err)
	}
	if got := frc.FormatTime(epoch(1711130400), loc, false, sameYear); got != "Fri 14:00" {
		t.Errorf("FormatTime = %q, want %q", got, "Fri 14:00")
	}
}

func TestFormatTimeWithoutATime(t *testing.T) {
	if got := frc.FormatTime(nil, time.UTC, false, sameYear); got != "" {
		t.Errorf("FormatTime(nil) = %q, want %q", got, "")
	}
	if got := frc.FormatTime(epoch(0), time.UTC, false, sameYear); got != "" {
		t.Errorf("FormatTime(0) = %q, want %q", got, "")
	}
}

// A nil location means the machine's own, which is what someone at the event
// reads off their phone.
func TestFormatTimeDefaultsToLocal(t *testing.T) {
	want := time.Unix(1711130400, 0).In(time.Local).Format(frc.TimeLayout)
	if got := frc.FormatTime(epoch(1711130400), nil, false, sameYear); got != want {
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
	if got := frc.FormatTime(epoch(1711130400), time.UTC, true, sameYear); got != "Mar 22 18:00" {
		t.Errorf("FormatTime = %q, want %q", got, "Mar 22 18:00")
	}
	if got := frc.FormatTime(nil, time.UTC, true, sameYear); got != "" {
		t.Errorf("FormatTime(nil) = %q, want %q", got, "")
	}
}

// A dated time in another year carries the year too: "Mar 22 18:00" read in
// 2026 could be any of three March the 22nds, and which season a match belongs
// to is the first thing anybody asks about an old one.
func TestFormatTimeWithTheYearWhenItIsNotThisOne(t *testing.T) {
	later := time.Date(2026, 3, 22, 18, 0, 0, 0, time.UTC)
	if got := frc.FormatTime(epoch(1711130400), time.UTC, true, later); got != "Mar 22 2024 18:00" {
		t.Errorf("FormatTime = %q, want %q", got, "Mar 22 2024 18:00")
	}
	// The year is decided in the reader's own zone, like the date beside it:
	// 2025-01-01 00:30 UTC is still 2024 in New York.
	loc, err := time.LoadLocation("America/New_York")
	if err != nil {
		t.Skipf("no tzdata available: %v", err)
	}
	newYear := time.Date(2025, 1, 1, 0, 30, 0, 0, time.UTC).Unix()
	if got := frc.FormatTime(&newYear, loc, true, time.Date(2024, 12, 31, 12, 0, 0, 0, time.UTC)); got != "Dec 31 19:30" {
		t.Errorf("FormatTime = %q, want %q", got, "Dec 31 19:30")
	}
}

// Without the date there is no year to print: a time written as a weekday is
// one the reader has already been told is today.
func TestFormatTimeWithoutTheDateNeverCarriesTheYear(t *testing.T) {
	later := time.Date(2026, 3, 22, 18, 0, 0, 0, time.UTC)
	if got := frc.FormatTime(epoch(1711130400), time.UTC, false, later); got != "Fri 18:00" {
		t.Errorf("FormatTime = %q, want %q", got, "Fri 18:00")
	}
}

// A listing of matches happening today needs no date on every row: the reader
// knows what day it is.
func TestNeedsDateForTodaysMatches(t *testing.T) {
	matches := []api.Match{
		{ActualTime: epoch(1711130400)}, // 2024-03-22 18:00 UTC
		{ActualTime: epoch(1711141200)}, // 2024-03-22 21:00 UTC
	}
	now := time.Date(2024, 3, 22, 22, 0, 0, 0, time.UTC)
	if frc.NeedsDate(matches, time.UTC, now) {
		t.Error("NeedsDate = true, want false for two times on today, 2024-03-22 UTC")
	}
}

// One day, but not this one: "Sat 13:57" against an April match names one of
// four Saturdays and gives no way to tell which.
func TestNeedsDateForAPastSingleDayListing(t *testing.T) {
	matches := []api.Match{
		{ActualTime: epoch(1711130400)}, // 2024-03-22 UTC
		{ActualTime: epoch(1711141200)}, // 2024-03-22 UTC
	}
	now := time.Date(2024, 4, 30, 9, 0, 0, 0, time.UTC)
	if !frc.NeedsDate(matches, time.UTC, now) {
		t.Error("NeedsDate = false, want true for a listing of a day that is not today")
	}
}

func TestNeedsDateAcrossDays(t *testing.T) {
	matches := []api.Match{
		{ActualTime: epoch(1711130400)}, // 2024-03-22 UTC
		{ActualTime: epoch(1711299600)}, // 2024-03-24 UTC
	}
	now := time.Date(2024, 3, 22, 22, 0, 0, 0, time.UTC)
	if !frc.NeedsDate(matches, time.UTC, now) {
		t.Error("NeedsDate = false, want true when one of the matches is not today")
	}
}

// A schedule that has not been published has no time to date, so a timeless
// match is not a reason to widen the column.
func TestNeedsDateIgnoresMatchesWithoutATime(t *testing.T) {
	now := time.Date(2024, 3, 22, 22, 0, 0, 0, time.UTC)
	matches := []api.Match{
		{ActualTime: epoch(1711130400)}, // today
		{},
		{PredictedTime: epoch(0)},
	}
	if frc.NeedsDate(matches, time.UTC, now) {
		t.Error("NeedsDate = true, want false when the only timed match is today")
	}
	if frc.NeedsDate(nil, time.UTC, now) {
		t.Error("NeedsDate(nil) = true, want false")
	}
}

// Today is where the reader is, not in UTC: two matches either side of
// midnight UTC are one evening in Hartford.
func TestNeedsDateUsesTheGivenLocation(t *testing.T) {
	loc, err := time.LoadLocation("America/New_York")
	if err != nil {
		t.Skipf("no tzdata available: %v", err)
	}
	matches := []api.Match{
		{ActualTime: epoch(1711148400)}, // 2024-03-22 19:00 EDT
		{ActualTime: epoch(1711155600)}, // 2024-03-22 21:00 EDT
	}
	now := time.Unix(1711155600, 0)
	if frc.NeedsDate(matches, loc, now) {
		t.Error("NeedsDate = true, want false for one evening in New York")
	}
	if !frc.NeedsDate(matches, time.UTC, now) {
		t.Error("NeedsDate = false in UTC, where those times straddle midnight")
	}
}
