package frc

import (
	"fmt"
	"time"

	"github.com/the-blue-alliance/tba-cli/internal/api"
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

// TimeLayout shows a weekday and a 24-hour clock. A competition fits in a
// weekend, so the date adds noise without adding information.
const TimeLayout = "Mon 15:04"

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
func FormatTime(epoch *int64, loc *time.Location) string {
	if epoch == nil || *epoch == 0 {
		return ""
	}
	if loc == nil {
		loc = time.Local
	}
	return time.Unix(*epoch, 0).In(loc).Format(TimeLayout)
}

// Relative describes when t happens relative to now: "in 18m", "2h ago",
// "in 2d". The magnitude is truncated rather than rounded, so "in 18m" means
// at least eighteen minutes from now.
func Relative(t, now time.Time) string {
	d := t.Sub(now)
	if d < 0 {
		return magnitude(-d) + " ago"
	}
	return "in " + magnitude(d)
}

// RelativeEpoch is Relative for an optional Unix timestamp, returning "" when
// there is none.
func RelativeEpoch(epoch *int64, now time.Time) string {
	if epoch == nil || *epoch == 0 {
		return ""
	}
	return Relative(time.Unix(*epoch, 0), now)
}

// magnitude renders a non-negative duration with a single coarse unit, because
// a countdown of "1h 3m 12s" is harder to read at a glance than "1h".
func magnitude(d time.Duration) string {
	switch {
	case d < time.Minute:
		return fmt.Sprintf("%ds", int(d/time.Second))
	case d < time.Hour:
		return fmt.Sprintf("%dm", int(d/time.Minute))
	case d < 24*time.Hour:
		return fmt.Sprintf("%dh", int(d/time.Hour))
	default:
		return fmt.Sprintf("%dd", int(d/(24*time.Hour)))
	}
}
