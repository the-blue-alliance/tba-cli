package frc

import (
	"cmp"
	"fmt"
	"slices"
	"strconv"
	"strings"

	"github.com/the-blue-alliance/tba-cli/internal/api"
)

// KV is one flattened score-breakdown entry.
type KV struct {
	Key   string
	Value string
}

// FlattenBreakdown turns one alliance's score_breakdown into a sorted list of
// readable key/value pairs.
//
// The breakdown's shape changes every season and is documented nowhere, so it
// is rendered generically rather than per-game: nested objects and arrays are
// flattened into dotted paths ("autoCommunity.B.1"), booleans read as yes and
// no, and numbers drop the trailing zeroes JSON gives them. Sorting the keys
// means the same game always prints in the same order.
//
// The sort is aware of the array indices in those paths: a 12-element array
// reads 1, 2, ... 10, 11 rather than the 1, 10, 11, 2 a plain string sort
// gives, which for a game piece grid is the difference between a readable
// listing and a puzzle.
func FlattenBreakdown(breakdown map[string]interface{}) []KV {
	out := make([]KV, 0, len(breakdown))
	flattenInto(&out, "", breakdown)
	slices.SortFunc(out, func(a, b KV) int { return CompareBreakdownKeys(a.Key, b.Key) })
	return out
}

// CompareBreakdownKeys orders two flattened breakdown paths, segment by
// segment, comparing segments that are whole numbers as numbers. It returns
// the usual negative/zero/positive.
func CompareBreakdownKeys(a, b string) int {
	as, bs := strings.Split(a, "."), strings.Split(b, ".")
	for i := 0; i < len(as) && i < len(bs); i++ {
		if c := compareSegment(as[i], bs[i]); c != 0 {
			return c
		}
	}
	// A prefix sorts before what extends it: "auto" before "auto.0".
	return cmp.Compare(len(as), len(bs))
}

// compareSegment orders one path segment. A numeric segment sorts before a
// named one, so the indices of an array stay together even in the unlikely
// event that something else shares their level.
func compareSegment(a, b string) int {
	an, aNum := segmentNumber(a)
	bn, bNum := segmentNumber(b)
	switch {
	case aNum && bNum:
		if c := cmp.Compare(an, bn); c != 0 {
			return c
		}
		// "01" and "1" are different keys that read as the same number;
		// fall back to the text so the order stays total.
		return strings.Compare(a, b)
	case aNum:
		return -1
	case bNum:
		return 1
	default:
		return strings.Compare(a, b)
	}
}

// segmentNumber reads a segment as an array index. Only plain non-negative
// digits count: a key that merely starts with a digit is a name.
func segmentNumber(s string) (int, bool) {
	if s == "" {
		return 0, false
	}
	for _, r := range s {
		if r < '0' || r > '9' {
			return 0, false
		}
	}
	n, err := strconv.Atoi(s)
	if err != nil {
		return 0, false
	}
	return n, true
}

// AllianceBreakdown pulls one alliance's section out of a match's
// score_breakdown, which is keyed by alliance color. It returns nil when the
// match has no breakdown, which is normal for older seasons and for matches
// that have not been played.
func AllianceBreakdown(m api.Match, color string) []KV {
	if m.ScoreBreakdown == nil {
		return nil
	}
	section, ok := m.ScoreBreakdown[color].(map[string]interface{})
	if !ok {
		return nil
	}
	return FlattenBreakdown(section)
}

func flattenInto(out *[]KV, prefix string, v interface{}) {
	switch t := v.(type) {
	case map[string]interface{}:
		for key, child := range t {
			flattenInto(out, join(prefix, key), child)
		}
	case []interface{}:
		if len(t) == 0 {
			*out = append(*out, KV{Key: prefix, Value: ""})
			return
		}
		for i, child := range t {
			flattenInto(out, join(prefix, strconv.Itoa(i)), child)
		}
	default:
		*out = append(*out, KV{Key: prefix, Value: FormatValue(v)})
	}
}

func join(prefix, key string) string {
	if prefix == "" {
		return key
	}
	return prefix + "." + key
}

// FormatValue renders a decoded JSON scalar for a human: a bool as yes or no,
// a number without the trailing zeroes JSON decoding gives it, null as empty.
func FormatValue(v interface{}) string {
	switch t := v.(type) {
	case nil:
		return ""
	case bool:
		if t {
			return "yes"
		}
		return "no"
	case float64:
		return strconv.FormatFloat(t, 'f', -1, 64)
	case string:
		return t
	default:
		return fmt.Sprintf("%v", t)
	}
}

// levelNames are the names a competition calls its rounds, for prose where a
// bare "sf" would not read.
var levelNames = map[string]string{
	LevelQual:         "Qualification",
	LevelEighthFinal:  "Octofinals",
	LevelQuarterFinal: "Quarterfinals",
	LevelSemiFinal:    "Semifinals",
	LevelFinal:        "Finals",
}

// LevelName spells out a comp level. An unrecognised level is returned as
// given, so a future season's round is shown rather than swallowed.
func LevelName(level string) string {
	if name, ok := levelNames[strings.ToLower(strings.TrimSpace(level))]; ok {
		return name
	}
	return level
}

// PickPosition names where a team came in its alliance's selection: the
// captain picks, the others were picked, and a backup was called in after the
// alliance was already formed.
func PickPosition(a api.TeamEventAllianceStatus, teamKey string) string {
	if a.Backup != nil && a.Backup.In == teamKey {
		return "backup"
	}
	switch a.Pick {
	case 0:
		return "Captain"
	case 1:
		return "1st pick"
	case 2:
		return "2nd pick"
	case 3:
		return "3rd pick"
	default:
		return fmt.Sprintf("pick %d", a.Pick)
	}
}

// Record renders a win-loss-tie record, or "" when there is none.
func Record(r *api.WLTRecord) string {
	if r == nil {
		return ""
	}
	return fmt.Sprintf("%d-%d-%d", r.Wins, r.Losses, r.Ties)
}

// StripHTML removes tags from the API's prose status strings, which are
// written for a web page and arrive with <b> and <a> markup in them. Entities
// a terminal cannot show as markup are decoded too.
func StripHTML(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	depth := 0
	for _, r := range s {
		switch {
		case r == '<':
			depth++
		case r == '>' && depth > 0:
			depth--
		case depth == 0:
			b.WriteRune(r)
		}
	}
	return strings.TrimSpace(decodeEntities(b.String()))
}

// entities are the few HTML entities the API's status strings actually use.
var entities = strings.NewReplacer(
	"&amp;", "&",
	"&lt;", "<",
	"&gt;", ">",
	"&quot;", `"`,
	"&#39;", "'",
	"&apos;", "'",
	"&nbsp;", " ",
)

func decodeEntities(s string) string { return entities.Replace(s) }
