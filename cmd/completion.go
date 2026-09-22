package cmd

import (
	"encoding/json"
	"net/url"
	"regexp"
	"sort"
	"strings"

	"github.com/spf13/cobra"
	"github.com/the-blue-alliance/tba-cli/internal/api"
	"github.com/the-blue-alliance/tba-cli/internal/cache"
)

// Argument completion is answered entirely from the on-disk response cache.
//
// A shell asks for completions on every Tab, so reaching for the network would
// mean rate limits, a stall on a slow link, and requests the user never asked
// for. Whatever has already been fetched is enough to be useful: after one
// `tba event list` the season's keys complete. An empty cache completes
// nothing, which is the honest answer.

var (
	// An event list lives at /events/{year}, and also at
	// /team/frcN/events[/{year}] and /district/{key}/events.
	cachedEventsPath = regexp.MustCompile(`/events(/[0-9]{4})?$`)
	// Districts for a season.
	cachedDistrictsPath = regexp.MustCompile(`/districts/[0-9]{4}$`)
	// Match lists, per event or per team-season.
	cachedMatchesPath = regexp.MustCompile(`/matches(/[0-9]{4})?$`)
)

// cachedEntries reads the whole cache, or returns nothing if it cannot be
// read. Completion has nowhere to report an error to, and a shell that gets no
// suggestions is better off than one that gets a stack trace in its prompt.
func cachedEntries() []cache.Entry {
	c, err := cache.New()
	if err != nil {
		return nil
	}
	entries, err := c.List()
	if err != nil {
		return nil
	}
	return entries
}

// entryPath is the API path a cache entry was fetched from. Entries are keyed
// by full URL, which carries whichever base URL was in use at the time.
func entryPath(e cache.Entry) string {
	u, err := url.Parse(e.URL)
	if err != nil {
		return e.URL
	}
	return u.Path
}

// suggestionsFrom renders a value -> description map as cobra completions,
// sorted so the shell shows a stable list. Cobra splits each entry on the tab.
func suggestionsFrom(m map[string]string) []string {
	out := make([]string, 0, len(m))
	for value, description := range m {
		if description != "" {
			value += "\t" + description
		}
		out = append(out, value)
	}
	sort.Strings(out)
	return out
}

func hasPrefixFold(s, prefix string) bool {
	return len(s) >= len(prefix) && strings.EqualFold(s[:len(prefix)], prefix)
}

// completeEventKeys suggests event keys found in cached event lists.
func completeEventKeys(_ *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	if len(args) > 0 {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	return suggestionsFrom(cachedEventKeys(toComplete)), cobra.ShellCompDirectiveNoFileComp
}

func cachedEventKeys(toComplete string) map[string]string {
	found := map[string]string{}
	for _, entry := range cachedEntries() {
		if !cachedEventsPath.MatchString(entryPath(entry)) {
			continue
		}
		var events []api.Event
		if err := json.Unmarshal(entry.Body, &events); err != nil {
			continue
		}
		for _, e := range events {
			if e.Key != "" && hasPrefixFold(e.Key, toComplete) {
				found[e.Key] = e.Name
			}
		}
	}
	return found
}

// completeMatchKeys suggests match keys found in cached match lists.
func completeMatchKeys(_ *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	if len(args) > 0 {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	found := map[string]string{}
	for _, entry := range cachedEntries() {
		if !cachedMatchesPath.MatchString(entryPath(entry)) {
			continue
		}
		var matches []api.Match
		if err := json.Unmarshal(entry.Body, &matches); err != nil {
			continue
		}
		for _, m := range matches {
			if m.Key != "" && hasPrefixFold(m.Key, toComplete) {
				found[m.Key] = ""
			}
		}
	}
	return suggestionsFrom(found), cobra.ShellCompDirectiveNoFileComp
}

// completeWebTargets suggests what `tba open` accepts. Team numbers are not
// suggested for the same reason team arguments are not completed at all.
func completeWebTargets(_ *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	if len(args) > 0 {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	found := cachedEventKeys(toComplete)
	matches, _ := completeMatchKeys(nil, nil, toComplete)
	return append(suggestionsFrom(found), matches...), cobra.ShellCompDirectiveNoFileComp
}

// completeDistrictAbbreviations suggests the --district values seen in the
// cache, from district lists and from the districts named on cached events.
func completeDistrictAbbreviations(_ *cobra.Command, _ []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	found := map[string]string{}
	add := func(d api.District) {
		if d.Abbreviation != "" && hasPrefixFold(d.Abbreviation, toComplete) {
			found[d.Abbreviation] = d.DisplayName
		}
	}
	for _, entry := range cachedEntries() {
		path := entryPath(entry)
		switch {
		case cachedDistrictsPath.MatchString(path):
			var districts []api.District
			if err := json.Unmarshal(entry.Body, &districts); err != nil {
				continue
			}
			for _, d := range districts {
				add(d)
			}
		case cachedEventsPath.MatchString(path):
			var events []api.Event
			if err := json.Unmarshal(entry.Body, &events); err != nil {
				continue
			}
			for _, e := range events {
				if e.District != nil {
					add(*e.District)
				}
			}
		}
	}
	return suggestionsFrom(found), cobra.ShellCompDirectiveNoFileComp
}

// completeEventTypes suggests the --type values, which are a fixed list and
// need no cache at all.
func completeEventTypes(_ *cobra.Command, _ []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	// Only the last value of a comma-separated list is being typed.
	prefix := ""
	if i := strings.LastIndex(toComplete, ","); i >= 0 {
		prefix, toComplete = toComplete[:i+1], toComplete[i+1:]
	}
	var out []string
	for _, a := range eventTypeAliases {
		if hasPrefixFold(a.name, toComplete) {
			out = append(out, prefix+a.name)
		}
	}
	return out, cobra.ShellCompDirectiveNoSpace | cobra.ShellCompDirectiveNoFileComp
}

// completeNothing turns off completion for an argument, file names included.
// Team arguments are numbers: there is nothing sensible to offer, and the
// shell's file list is worse than an empty one.
func completeNothing(_ *cobra.Command, _ []string, _ string) ([]string, cobra.ShellCompDirective) {
	return nil, cobra.ShellCompDirectiveNoFileComp
}

// attachCompletions wires argument completion onto a finished command tree.
//
// It walks the tree instead of being set in each constructor so that adding a
// command that takes an event key gets completion for free, and so that all
// the completion behaviour is described in one place.
func attachCompletions(root *cobra.Command) {
	for _, group := range root.Commands() {
		switch group.Name() {
		case "event":
			for _, sub := range group.Commands() {
				if strings.Contains(sub.Use, "<key>") {
					sub.ValidArgsFunction = completeEventKeys
				}
				if sub.Name() == "list" {
					_ = sub.RegisterFlagCompletionFunc("district", completeDistrictAbbreviations)
					_ = sub.RegisterFlagCompletionFunc("type", completeEventTypes)
					_ = sub.RegisterFlagCompletionFunc("team", completeNothing)
				}
			}
		case "match":
			for _, sub := range group.Commands() {
				if strings.Contains(sub.Use, "<key>") {
					sub.ValidArgsFunction = completeMatchKeys
				}
			}
		case "team":
			for _, sub := range group.Commands() {
				if strings.Contains(sub.Use, "<number>") {
					sub.ValidArgsFunction = completeNothing
				}
			}
		case "open":
			group.ValidArgsFunction = completeWebTargets
		}
	}
}
