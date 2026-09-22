package frc

import (
	"time"

	"github.com/the-blue-alliance/tba-cli/internal/api"
	"github.com/the-blue-alliance/tba-cli/internal/humanize"
)

// Where a match time came from. The API carries three, and which one is on
// offer tells the user how much to trust it.
const (
	// TimeActual is when the match was actually played.
	TimeActual = "actual"
	// TimePredicted is the queue's live estimate, which drifts as an event
	// runs ahead of or behind schedule.
	TimePredicted = "predicted"
	// TimeScheduled is the published schedule.
	TimeScheduled = "scheduled"
)

// The two ways a match time is written.
const (
	// TimeLayout shows a weekday and a 24-hour clock, which is all anyone at
	// an event needs: a competition fits in a weekend, so the date would add
	// noise without adding information.
	TimeLayout = "Mon 15:04"
	// DatedTimeLayout adds the calendar date, for a listing that spans more
	// than one day — a whole season, say, where "Sat 11:22" could be any of
	// a dozen Saturdays.
	DatedTimeLayout = "Jan 2 15:04"
)

// BestTime returns the most useful time the API has for a match, and which of
// the three it is: what actually happened, else the live prediction, else the
// published schedule. It returns (nil, "") when the match has no time at all,
// which happens for a schedule that has not been posted yet.
//
// A zero epoch is treated as absent: the API uses it for an unset time, and
// 1970 is never a real answer.
func BestTime(m api.Match) (*int64, string) {
	for _, candidate := range []struct {
		epoch  *int64
		source string
	}{
		{m.ActualTime, TimeActual},
		{m.PredictedTime, TimePredicted},
		{m.Time, TimeScheduled},
	} {
		if candidate.epoch != nil && *candidate.epoch != 0 {
			return candidate.epoch, candidate.source
		}
	}
	return nil, ""
}

// FormatTime renders a Unix timestamp in loc, or "" when there is none. A nil
// location means the machine's local time, which is what someone standing at
// the event wants to read.
//
// withDate adds the calendar date. A caller decides it once for a whole table
// (see NeedsDate) rather than per row, so that every time in a listing is
// written the same way and the column stays one width.
func FormatTime(epoch *int64, loc *time.Location, withDate bool) string {
	if epoch == nil || *epoch == 0 {
		return ""
	}
	if loc == nil {
		loc = time.Local
	}
	layout := TimeLayout
	if withDate {
		layout = DatedTimeLayout
	}
	return time.Unix(*epoch, 0).In(loc).Format(layout)
}

// NeedsDate reports whether the times in a listing have to carry the calendar
// date as well as the weekday, which they do unless every match in it is
// happening today in loc.
//
// "Sat 13:57" is what someone standing at the event wants, and only there: it
// is unambiguous because the reader knows what day it is. Read a month later,
// a listing of an April match that says "Sat 13:57" has named one of four
// Saturdays and given no way to tell which. The old rule asked whether the
// listing spanned more than one day, which let a single-day listing of a
// long-finished event answer with a weekday alone.
//
// Matches with no time at all are ignored: an unpublished schedule is not a
// second day, and a listing with no times at all has no date to print anyway.
func NeedsDate(matches []api.Match, loc *time.Location, now time.Time) bool {
	if loc == nil {
		loc = time.Local
	}
	today := now.In(loc).Format(DateLayout)
	for _, m := range matches {
		epoch, _ := BestTime(m)
		if epoch == nil || *epoch == 0 {
			continue
		}
		if time.Unix(*epoch, 0).In(loc).Format(DateLayout) != today {
			return true
		}
	}
	return false
}

// Relative describes when t happens relative to now: "in 18m", "2h ago",
// "in 2d". The magnitude is truncated rather than rounded, so "in 18m" means
// at least eighteen minutes from now.
func Relative(t, now time.Time) string {
	d := t.Sub(now)
	if d < 0 {
		return Magnitude(-d) + " ago"
	}
	return "in " + Magnitude(d)
}

// RelativeEpoch is Relative for an optional Unix timestamp, returning "" when
// there is none.
func RelativeEpoch(epoch *int64, now time.Time) string {
	if epoch == nil || *epoch == 0 {
		return ""
	}
	return Relative(time.Unix(*epoch, 0), now)
}

// Magnitude renders a non-negative duration with a single coarse unit,
// because a countdown of "1h 3m 12s" is harder to read at a glance than "1h".
// A negative duration is measured as its absolute value, so a caller that has
// already chosen a "before"/"after" wording cannot print a minus sign.
func Magnitude(d time.Duration) string { return humanize.Age(d) }
