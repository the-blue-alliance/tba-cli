package cmd

import (
	"context"
	"fmt"
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

// firstFRCSeason is the earliest season The Blue Alliance holds data for.
const firstFRCSeason = 1992

// resolveYear returns the season a command should ask about: the --year flag
// if it was given, else TBA_YEAR, else the config file, else the season The
// Blue Alliance says is current, else the calendar year.
//
// The last step matters offline. A season is named after the calendar year it
// ends in, so the calendar is right about it except for the few weeks around
// the January kickoff, which is a far better answer than an error.
//
// client is optional, and is how a command lends the season lookup the client
// it already built; every caller that has one does. Without it the lookup
// builds its own, which works but gives the extra request its own rate
// limiter, outside the pacing the rest of the command is doing.
func resolveYear(cmd *cobra.Command, client ...*api.Client) (int, error) {
	s := settings(cmd)
	clk := clockOf(cmd)
	if s.Source("year") != sourceDefault {
		return validateSeason(clk.now(), s.Int("year"), s.origin("year"))
	}
	var lent *api.Client
	if len(client) > 0 {
		lent = client[0]
	}
	return currentSeason(cmd, lent, clk)
}

// validateSeason rejects a year that cannot name an FRC season.
//
// The upper bound is next calendar year rather than this one: a season is
// named after the year it ends in, so from kickoff in January the coming
// season is a real thing to ask about. Past that it is a typo, a four-digit
// number that was meant to be something else, or an event key that lost its
// letters — all of which are better reported than turned into a request for a
// season that does not exist.
func validateSeason(now time.Time, year int, origin string) (int, error) {
	latest := currentYear(now) + 1
	if year < firstFRCSeason || year > latest {
		return 0, clierr.Usage("%s %d is not an FRC season (%d-%d)", origin, year, firstFRCSeason, latest)
	}
	return year, nil
}

// currentSeason asks the API which season is on, remembering the answer for a
// day so that the default --year costs one extra request at most per day.
func currentSeason(cmd *cobra.Command, client *api.Client, clk clock) (int, error) {
	noCache := settings(cmd).Bool("no-cache")

	// The cache is given the same clock, so a pinned "now" decides both which
	// season is current and whether the remembered one has gone stale.
	store, err := season.New(clk.now)
	if err != nil {
		store = nil
	}
	if store != nil && !noCache {
		if year, ok := store.Get(); ok {
			return year, nil
		}
	}
	year, err := fetchCurrentSeason(cmd, client)
	if err != nil {
		return 0, err
	}
	if year > 0 {
		if store != nil {
			_ = store.Put(year)
		}
		return year, nil
	}
	return currentYear(clk.now()), nil
}

// fetchCurrentSeason reads current_season from /status. It returns 0 for a
// failure the calendar can paper over, and an error only for one it cannot.
//
// Almost everything is in the first group — no API key, no network, a 500,
// nonsense in the field — because the season is named after the calendar year
// it ends in, and a command that cannot work out the season on its own is
// about to report the real problem anyway.
//
// The exception is the user's own context ending. Ctrl-C used to be swallowed
// here along with everything else, so the command carried on with a guessed
// year and made another request before failing, instead of stopping when it
// was told to. The test is the caller's context rather than the shape of the
// error, because a --timeout that expires also reports a deadline and is an
// ordinary failure to fall back from.
func fetchCurrentSeason(cmd *cobra.Command, client *api.Client) (int, error) {
	ctx := cmd.Context()
	if ctx == nil {
		ctx = context.Background()
	}
	if client == nil {
		var err error
		if client, err = newClient(cmd); err != nil {
			return 0, nil
		}
	}
	var status api.APIStatus
	if err := client.Get(ctx, "/status", &status); err != nil {
		if ctxErr := ctx.Err(); ctxErr != nil {
			return 0, fmt.Errorf("looking up the current season: %w", ctxErr)
		}
		return 0, nil
	}
	if status.CurrentSeason <= 0 {
		return 0, nil
	}
	return status.CurrentSeason, nil
}

// currentYear is now's calendar year, which is the season's name outside of
// kickoff season.
func currentYear(now time.Time) int {
	return now.Year()
}
