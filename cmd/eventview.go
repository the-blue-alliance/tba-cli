package cmd

import (
	"fmt"

	"github.com/the-blue-alliance/tba-cli/internal/api"
)

// playoffTypeNames spells out TBA's playoff_type codes. The numbers are not
// contiguous and are not in any meaningful order: 5 is the old-style double
// elimination bracket used before 2023, while 10 and 11 are the current one.
var playoffTypeNames = map[int]string{
	0:  "Bracket (8 alliances)",
	1:  "Bracket (16 alliances)",
	2:  "Bracket (4 alliances)",
	3:  "Average score (8 alliances)",
	4:  "Round robin (6 alliances)",
	5:  "Legacy double elimination (8 alliances)",
	6:  "Best of 5 finals",
	7:  "Best of 3 finals",
	8:  "Custom",
	9:  "Bracket (2 alliances)",
	10: "Double elimination (8 alliances)",
	11: "Double elimination (4 alliances)",
}

// playoffTypeName renders a playoff_type for people. An unmapped code still
// prints its number rather than vanishing, because a new playoff format is
// exactly the thing someone would be looking for.
func playoffTypeName(t *int) string {
	if t == nil {
		return ""
	}
	if name, ok := playoffTypeNames[*t]; ok {
		return name
	}
	return fmt.Sprintf("Unknown (%d)", *t)
}

// webcastURL turns a webcast into something that can be opened. TBA stores a
// provider and a channel rather than a link, so the known providers are
// expanded and anything else is shown as-is instead of being guessed at.
func webcastURL(w api.Webcast) string {
	switch w.Type {
	case "twitch":
		return "https://www.twitch.tv/" + w.Channel
	case "youtube":
		return "https://www.youtube.com/watch?v=" + w.Channel
	default:
		return fmt.Sprintf("%s: %s", w.Type, w.Channel)
	}
}

// districtName describes an event's district as "New England (ne)", falling
// back to whichever half is present.
func districtName(d *api.District) string {
	if d == nil {
		return ""
	}
	switch {
	case d.DisplayName != "" && d.Abbreviation != "":
		return fmt.Sprintf("%s (%s)", d.DisplayName, d.Abbreviation)
	case d.DisplayName != "":
		return d.DisplayName
	default:
		return d.Abbreviation
	}
}

// appendPair adds a key/value pair to a PrintKeyValue argument list, skipping
// the row entirely when the value is empty so that a detail view does not fill
// up with blanks.
func appendPair(pairs []string, key, value string) []string {
	if value == "" {
		return pairs
	}
	return append(pairs, key, value)
}
