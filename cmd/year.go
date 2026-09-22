package cmd

import (
	"time"

	"github.com/spf13/cobra"

	"github.com/the-blue-alliance/tba-cli/internal/api"
	"github.com/the-blue-alliance/tba-cli/internal/clierr"
	"github.com/the-blue-alliance/tba-cli/internal/season"
)

// yearFlagUsage is the help text every season-scoped --year carries. The
// default is not a number, so the flag's zero value is spelled out here
// instead of being printed by cobra as "0".
const yearFlagUsage = "Season year (default: current season)"

// addYearFlag registers the --year flag shared by the season-scoped commands.
func addYearFlag(c *cobra.Command) {
	c.Flags().Int("year", 0, yearFlagUsage)
}

// resolveYear returns the season a command should ask about: the --year flag
// if it was given, else TBA_YEAR, else the config file, else the season The
// Blue Alliance says is current, else the calendar year.
//
// The last step matters offline. A season is named after the calendar year it
// ends in, so the calendar is right about it except for the few weeks around
// the January kickoff, which is a far better answer than an error.
func resolveYear(cmd *cobra.Command) (int, error) {
	s := settings(cmd)
	if s.Source("year") != sourceDefault {
		year := s.Int("year")
		if year <= 0 {
			return 0, clierr.Usage("%s wants a season such as %d", s.origin("year"), currentYear())
		}
		return year, nil
	}
	return currentSeason(cmd), nil
}

// currentSeason asks the API which season is on, remembering the answer for a
// day so that the default --year costs one extra request at most per day.
func currentSeason(cmd *cobra.Command) int {
	noCache := settings(cmd).Bool("no-cache")

	store, err := season.New()
	if err != nil {
		store = nil
	}
	if store != nil && !noCache {
		if year, ok := store.Get(); ok {
			return year
		}
	}
	if year, ok := fetchCurrentSeason(cmd); ok {
		if store != nil {
			_ = store.Put(year)
		}
		return year
	}
	return currentYear()
}

// fetchCurrentSeason reads current_season from /status. Every failure —
// no API key, no network, a 500, nonsense in the field — reports "unknown"
// and lets the caller fall back to the calendar: a command that cannot work
// out the season on its own is about to report the real problem anyway.
func fetchCurrentSeason(cmd *cobra.Command) (int, bool) {
	client, err := newClient(cmd)
	if err != nil {
		return 0, false
	}
	var status api.APIStatus
	if err := client.Get(cmd.Context(), "/status", &status); err != nil {
		return 0, false
	}
	if status.CurrentSeason <= 0 {
		return 0, false
	}
	return status.CurrentSeason, true
}

// currentYear is the calendar year, which is the season's name outside of
// kickoff season.
func currentYear() int {
	return time.Now().Year()
}
