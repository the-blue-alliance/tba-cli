package frc

import (
	"cmp"
	"slices"
	"time"

	"github.com/the-blue-alliance/tba-cli/internal/api"
)

// DateLayout is how the API writes an event's start_date and end_date: a
// calendar day in the event's own local time, with no clock and no zone.
const DateLayout = "2006-01-02"

// ParseDate reads one of those dates as midnight in loc. It reports false for
// a missing or malformed date rather than guessing.
func ParseDate(s string, loc *time.Location) (time.Time, bool) {
	if s == "" {
		return time.Time{}, false
	}
	if loc == nil {
		loc = time.Local
	}
	t, err := time.ParseInLocation(DateLayout, s, loc)
	if err != nil {
		return time.Time{}, false
	}
	return t, true
}

// IsRunningOn reports whether day falls inside an event's dates, counting both
// the first and the last day: a team at a Saturday event is at that event all
// Saturday, not until some timestamp on it.
func IsRunningOn(e api.Event, day time.Time) bool {
	start, ok := ParseDate(e.StartDate, day.Location())
	if !ok {
		return false
	}
	end, ok := ParseDate(e.EndDate, day.Location())
	if !ok {
		end = start
	}
	day = truncateToDay(day)
	return !day.Before(start) && !day.After(end)
}

// Ended reports whether an event's last day is already behind day. Like
// IsRunningOn it counts whole days: an event that ends today has not ended,
// however late in the evening the question is asked.
//
// An event with no end date has not ended, since guessing would be worse than
// saying nothing.
func Ended(e api.Event, day time.Time) bool {
	end, ok := ParseDate(e.EndDate, day.Location())
	if !ok {
		return false
	}
	return truncateToDay(day).After(end)
}

// CurrentOrNextEvent picks the event a team is at right now, or failing that
// the next one it is going to. It returns false when the season holds neither,
// which is every day outside the competition season.
//
// "Right now" is a whole day: an event that started yesterday and ends
// tomorrow is the current one all of today. Among several candidates the
// earliest start wins, so a team at two events on one day gets the one that
// started first.
func CurrentOrNextEvent(events []api.Event, now time.Time) (api.Event, bool) {
	day := truncateToDay(now)

	var best api.Event
	var bestStart time.Time
	found := false
	consider := func(e api.Event, start time.Time) {
		if !found || start.Before(bestStart) {
			best, bestStart, found = e, start, true
		}
	}

	for _, e := range events {
		if IsRunningOn(e, day) {
			start, _ := ParseDate(e.StartDate, day.Location())
			consider(e, start)
		}
	}
	if found {
		return best, true
	}

	for _, e := range events {
		start, ok := ParseDate(e.StartDate, day.Location())
		if ok && start.After(day) {
			consider(e, start)
		}
	}
	return best, found
}

func truncateToDay(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
}

// NextUnplayed returns the first match that has not been played, ordered by
// when it is expected to start. It reports false when every match is done.
func NextUnplayed(matches []api.Match) (api.Match, bool) {
	upcoming := Unplayed(matches)
	if len(upcoming) == 0 {
		return api.Match{}, false
	}
	return upcoming[0], true
}

// Unplayed returns the matches still to come, soonest first.
func Unplayed(matches []api.Match) []api.Match {
	out := make([]api.Match, 0, len(matches))
	for _, m := range matches {
		if !Played(m) {
			out = append(out, m)
		}
	}
	SortByTime(out)
	return out
}

// LabelDependsOnPlayoffType reports whether a match's label would change with
// the event's bracket format. Only semifinals differ: a double-elimination set
// holds one match and is named by its set alone, while every other level is
// named the same way in every format. A caller can use this to decide whether
// fetching the event is worth an extra request.
func LabelDependsOnPlayoffType(m api.Match) bool {
	return CompLevelOrder(m.CompLevel) == CompLevelOrder(LevelSemiFinal)
}

// SortEvents puts a season's events in the order they are played, in place.
// See CompareEvents for what that order is.
func SortEvents(events []api.Event) {
	slices.SortStableFunc(events, CompareEvents)
}

// EventOrder ranks a team's events chronologically, so a season's worth of
// matches can be grouped the way the team actually played them: Waterbury in
// week 1 before Hartford in week 3, whatever order the API listed them in.
//
// It is SortEvents read as a ranking, so a listing and a grouping of the same
// season cannot disagree about which event came first.
func EventOrder(events []api.Event) map[string]int {
	sorted := make([]api.Event, len(events))
	copy(sorted, events)
	SortEvents(sorted)

	order := make(map[string]int, len(sorted))
	for _, e := range sorted {
		if _, seen := order[e.Key]; !seen {
			order[e.Key] = len(order)
		}
	}
	return order
}

// GroupByEvent regroups an already-ordered listing so that every event's
// matches sit together, events in the order given by order (see EventOrder).
//
// It is a stable sort and does nothing within an event, so whatever order the
// caller chose there — playing order, or by the clock for an upcoming listing —
// survives untouched. An event missing from order sorts after every known one,
// by key, which is what a listing falls back to when the team's event list
// could not be fetched.
func GroupByEvent(matches []api.Match, order map[string]int) {
	slices.SortStableFunc(matches, func(a, b api.Match) int {
		if a.EventKey == b.EventKey {
			return 0
		}
		ra, aok := order[a.EventKey]
		rb, bok := order[b.EventKey]
		if aok != bok {
			if aok {
				return -1
			}
			return 1
		}
		if aok {
			if c := cmp.Compare(ra, rb); c != 0 {
				return c
			}
		}
		return cmp.Compare(a.EventKey, b.EventKey)
	})
}
