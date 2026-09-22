package frc_test

import (
	"testing"
	"time"

	"github.com/the-blue-alliance/tba-cli/internal/api"
	"github.com/the-blue-alliance/tba-cli/internal/frc"
)

func day(y int, m time.Month, d int) time.Time {
	return time.Date(y, m, d, 12, 0, 0, 0, time.UTC)
}

func TestParseDate(t *testing.T) {
	got, ok := frc.ParseDate("2024-03-22", time.UTC)
	if !ok {
		t.Fatal("ParseDate returned not-ok for a well-formed date")
	}
	if want := time.Date(2024, 3, 22, 0, 0, 0, 0, time.UTC); !got.Equal(want) {
		t.Errorf("ParseDate = %v, want %v", got, want)
	}
}

func TestParseDateRejectsRubbish(t *testing.T) {
	for _, s := range []string{"", "not a date", "22-03-2024", "2024-13-45"} {
		if _, ok := frc.ParseDate(s, time.UTC); ok {
			t.Errorf("ParseDate(%q) reported ok", s)
		}
	}
}

// An event runs for its whole last day, not until midnight at its start.
func TestIsRunningOn(t *testing.T) {
	e := api.Event{Key: "2024cthar", StartDate: "2024-03-22", EndDate: "2024-03-24"}
	cases := []struct {
		day  time.Time
		want bool
	}{
		{day(2024, 3, 21), false},
		{day(2024, 3, 22), true},
		{day(2024, 3, 23), true},
		{day(2024, 3, 24), true},
		{day(2024, 3, 25), false},
	}
	for _, c := range cases {
		if got := frc.IsRunningOn(e, c.day); got != c.want {
			t.Errorf("IsRunningOn(%s) = %v, want %v", c.day.Format(frc.DateLayout), got, c.want)
		}
	}
}

// A one-day event with no end date still runs on its start date.
func TestIsRunningOnWithoutAnEndDate(t *testing.T) {
	e := api.Event{Key: "2024ctoff", StartDate: "2024-10-05"}
	if !frc.IsRunningOn(e, day(2024, 10, 5)) {
		t.Error("a one-day event should run on its own day")
	}
	if frc.IsRunningOn(e, day(2024, 10, 6)) {
		t.Error("a one-day event should not run the day after")
	}
}

func TestIsRunningOnWithoutAStartDate(t *testing.T) {
	if frc.IsRunningOn(api.Event{Key: "2024x"}, day(2024, 3, 22)) {
		t.Error("an event with no dates cannot be running")
	}
}

func season2024() []api.Event {
	return []api.Event{
		{Key: "2024necmp", StartDate: "2024-04-10", EndDate: "2024-04-13"},
		{Key: "2024cthar", StartDate: "2024-03-22", EndDate: "2024-03-24"},
		{Key: "2024ctwat", StartDate: "2024-03-08", EndDate: "2024-03-10"},
	}
}

func TestCurrentOrNextEventPrefersTheOneRunningToday(t *testing.T) {
	got, ok := frc.CurrentOrNextEvent(season2024(), day(2024, 3, 23))
	if !ok || got.Key != "2024cthar" {
		t.Errorf("event = %q (ok=%v), want 2024cthar", got.Key, ok)
	}
}

func TestCurrentOrNextEventFallsBackToTheNextOne(t *testing.T) {
	got, ok := frc.CurrentOrNextEvent(season2024(), day(2024, 3, 15))
	if !ok || got.Key != "2024cthar" {
		t.Errorf("event = %q (ok=%v), want 2024cthar", got.Key, ok)
	}
}

// The day after an event ends, the next one is what matters.
func TestCurrentOrNextEventTheDayAfter(t *testing.T) {
	got, ok := frc.CurrentOrNextEvent(season2024(), day(2024, 3, 25))
	if !ok || got.Key != "2024necmp" {
		t.Errorf("event = %q (ok=%v), want 2024necmp", got.Key, ok)
	}
}

func TestCurrentOrNextEventOnTheFirstDay(t *testing.T) {
	got, ok := frc.CurrentOrNextEvent(season2024(), day(2024, 3, 8))
	if !ok || got.Key != "2024ctwat" {
		t.Errorf("event = %q (ok=%v), want 2024ctwat", got.Key, ok)
	}
}

func TestCurrentOrNextEventAfterTheSeason(t *testing.T) {
	if _, ok := frc.CurrentOrNextEvent(season2024(), day(2024, 7, 1)); ok {
		t.Error("there is no event after the season ends")
	}
}

func TestCurrentOrNextEventWithNoEvents(t *testing.T) {
	if _, ok := frc.CurrentOrNextEvent(nil, day(2024, 3, 23)); ok {
		t.Error("a team with no events has no next event")
	}
}

// Two events on one day: the one that started first is the one being played.
func TestCurrentOrNextEventPrefersTheEarlierStart(t *testing.T) {
	events := []api.Event{
		{Key: "2024b", StartDate: "2024-03-23", EndDate: "2024-03-24"},
		{Key: "2024a", StartDate: "2024-03-22", EndDate: "2024-03-24"},
	}
	got, ok := frc.CurrentOrNextEvent(events, day(2024, 3, 23))
	if !ok || got.Key != "2024a" {
		t.Errorf("event = %q, want 2024a", got.Key)
	}
}

func TestCurrentOrNextEventSkipsUndatedEvents(t *testing.T) {
	events := []api.Event{
		{Key: "2024undated"},
		{Key: "2024cthar", StartDate: "2024-03-22", EndDate: "2024-03-24"},
	}
	got, ok := frc.CurrentOrNextEvent(events, day(2024, 3, 1))
	if !ok || got.Key != "2024cthar" {
		t.Errorf("event = %q, want 2024cthar", got.Key)
	}
}

func TestNextUnplayed(t *testing.T) {
	matches := []api.Match{
		mustMatch(t, qual2024JSON),
		mustMatch(t, unplayed2024JSON),
	}
	got, ok := frc.NextUnplayed(matches)
	if !ok || got.Key != "2024cthar_qm40" {
		t.Errorf("next = %q (ok=%v), want the unplayed match", got.Key, ok)
	}
}

func TestNextUnplayedOrdersByTime(t *testing.T) {
	matches := []api.Match{
		{Key: "later", CompLevel: "qm", MatchNumber: 1, PredictedTime: epoch(2000), Alliances: unplayedAlliances()},
		{Key: "sooner", CompLevel: "qm", MatchNumber: 9, PredictedTime: epoch(1000), Alliances: unplayedAlliances()},
	}
	got, ok := frc.NextUnplayed(matches)
	if !ok || got.Key != "sooner" {
		t.Errorf("next = %q, want the sooner one", got.Key)
	}
}

func TestNextUnplayedWhenEverythingIsPlayed(t *testing.T) {
	if _, ok := frc.NextUnplayed([]api.Match{mustMatch(t, qual2024JSON)}); ok {
		t.Error("a finished event has no next match")
	}
	if _, ok := frc.NextUnplayed(nil); ok {
		t.Error("no matches means no next match")
	}
}

func TestUnplayedDropsPlayedMatches(t *testing.T) {
	matches := []api.Match{
		mustMatch(t, qual2024JSON),
		mustMatch(t, unplayed2024JSON),
		mustMatch(t, doubleElimFinal2JSON),
	}
	got := frc.Unplayed(matches)
	if len(got) != 1 || got[0].Key != "2024cthar_qm40" {
		t.Errorf("Unplayed = %v", keysOf(got))
	}
}

// Only a semifinal's name changes with the bracket, so only a semifinal is
// worth an extra request to find out.
func TestLabelDependsOnPlayoffType(t *testing.T) {
	cases := map[string]bool{"qm": false, "ef": false, "qf": false, "sf": true, "f": false}
	for level, want := range cases {
		m := api.Match{CompLevel: level, SetNumber: 1, MatchNumber: 1}
		if got := frc.LabelDependsOnPlayoffType(m); got != want {
			t.Errorf("LabelDependsOnPlayoffType(%q) = %v, want %v", level, got, want)
		}
	}
}

func unplayedAlliances() map[string]api.Alliance {
	return map[string]api.Alliance{
		"red":  {Score: -1, TeamKeys: []string{"frc177"}},
		"blue": {Score: -1, TeamKeys: []string{"frc230"}},
	}
}
