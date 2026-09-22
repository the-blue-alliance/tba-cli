package cmd

import (
	"sort"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
	"github.com/the-blue-alliance/tba-cli/internal/api"
	"github.com/the-blue-alliance/tba-cli/internal/clierr"
)

// eventTypeAlias maps a friendly --type value onto the TBA event_type codes it
// covers. The order is the order the values are listed in help and in error
// messages, which runs roughly from the smallest event to the largest.
type eventTypeAlias struct {
	name  string
	codes []int
}

var eventTypeAliases = []eventTypeAlias{
	{"regional", []int{0}},
	{"district", []int{1}},
	{"dcmp", []int{2}},
	{"dcmp-division", []int{5}},
	{"cmp-division", []int{3}},
	{"cmp-finals", []int{4}},
	{"foc", []int{6}},
	{"offseason", []int{99}},
	{"preseason", []int{100}},
	{"remote", []int{7}},
	{"unlabeled", []int{-1}},
	{"all", nil},
}

// validEventTypes is the help/error listing of accepted --type values.
var validEventTypes = func() string {
	names := make([]string, len(eventTypeAliases))
	for i, a := range eventTypeAliases {
		names[i] = a.name
	}
	return strings.Join(names, ", ")
}()

// parseEventTypes turns a comma-separated --type value into the set of TBA
// event_type codes to keep. A nil set means "every type": either the flag was
// not given, or it included "all".
func parseEventTypes(spec string) (map[int]bool, error) {
	spec = strings.TrimSpace(spec)
	if spec == "" {
		return nil, nil
	}
	codes := map[int]bool{}
	for _, raw := range strings.Split(spec, ",") {
		name := strings.ToLower(strings.TrimSpace(raw))
		if name == "" {
			continue
		}
		alias, ok := lookupEventType(name)
		if !ok {
			return nil, clierr.Usage("invalid --type %q (want: %s)", raw, validEventTypes)
		}
		if alias.codes == nil {
			// "all" cancels the filter however it is combined.
			return nil, nil
		}
		for _, c := range alias.codes {
			codes[c] = true
		}
	}
	if len(codes) == 0 {
		return nil, nil
	}
	return codes, nil
}

func lookupEventType(name string) (eventTypeAlias, bool) {
	for _, a := range eventTypeAliases {
		if a.name == name {
			return a, true
		}
	}
	return eventTypeAlias{}, false
}

// eventFilter holds the client-side filters for `event list`. Every field is
// optional and they all combine: an event has to pass all of them.
type eventFilter struct {
	// week is the human 1-based competition week; TBA's own week field is
	// 0-based, so it is compared against week-1. 0 means no week filter.
	week     int
	types    map[int]bool
	district string
	state    string
	country  string
}

// firstFRCSeason is the earliest season The Blue Alliance holds events for. A
// year below it is a typo, and answering it with an empty list looks like "that
// season had no events".
const firstFRCSeason = 1992

func eventFilterFromFlags(cmd *cobra.Command) (eventFilter, error) {
	var f eventFilter
	// Week 0 is not a week: the flag is 1-based, the way thebluealliance.com
	// numbers weeks, and 0 is only the "no week filter" default. Asking for it
	// explicitly and being handed the whole season is worse than being told.
	week, _ := cmd.Flags().GetInt("week")
	if cmd.Flags().Changed("week") && week < 1 {
		return f, clierr.Usage("invalid --week %d (weeks are numbered from 1, as thebluealliance.com numbers them)", week)
	}
	f.week = week

	// A season that predates FRC's records would print an empty list, which
	// reads as "no events that year" rather than "no such year".
	if year := settings(cmd).Int("year"); year != 0 && year < firstFRCSeason {
		return f, clierr.Usage("--year %d is before the first FRC season (%d)", year, firstFRCSeason)
	}

	typeSpec, _ := cmd.Flags().GetString("type")
	types, err := parseEventTypes(typeSpec)
	if err != nil {
		return f, err
	}
	f.types = types

	// --team narrows the fetch rather than the fetched list, so the value is
	// checked here and used by the caller; a typo is a usage error instead of
	// a request for a team that cannot exist.
	if team, _ := cmd.Flags().GetString("team"); strings.TrimSpace(team) != "" {
		if err := validateTeamArg(team); err != nil {
			return f, err
		}
	}

	f.district, _ = cmd.Flags().GetString("district")
	f.state, _ = cmd.Flags().GetString("state")
	f.country, _ = cmd.Flags().GetString("country")
	return f, nil
}

func (f eventFilter) match(e api.Event) bool {
	if f.week > 0 {
		if e.Week == nil || *e.Week != f.week-1 {
			return false
		}
	}
	if f.types != nil && !f.types[e.EventType] {
		return false
	}
	if f.district != "" {
		if e.District == nil || !strings.EqualFold(e.District.Abbreviation, f.district) {
			return false
		}
	}
	if f.state != "" && !strings.EqualFold(e.StateProv, f.state) {
		return false
	}
	if f.country != "" && !strings.EqualFold(e.Country, f.country) {
		return false
	}
	return true
}

// apply keeps the events that pass every filter, preserving their order.
func (f eventFilter) apply(events []api.Event) []api.Event {
	out := make([]api.Event, 0, len(events))
	for _, e := range events {
		if f.match(e) {
			out = append(out, e)
		}
	}
	return out
}

// sortEvents orders events the way a season calendar reads: earliest start
// first, with the event key breaking ties so the order is stable and
// reproducible across runs.
func sortEvents(events []api.Event) {
	sort.SliceStable(events, func(i, j int) bool {
		if events[i].StartDate != events[j].StartDate {
			return events[i].StartDate < events[j].StartDate
		}
		return events[i].Key < events[j].Key
	})
}

// humanWeek renders TBA's 0-based week as the 1-based number people use.
// Events with no week (championships, offseasons) get an empty cell.
func humanWeek(week *int) string {
	if week == nil {
		return ""
	}
	return strconv.Itoa(*week + 1)
}

// districtAbbrev is the district's short name, or empty for a non-district event.
func districtAbbrev(d *api.District) string {
	if d == nil {
		return ""
	}
	return d.Abbreviation
}
