package frc

import (
	"cmp"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"unicode"

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

// BreakdownRow is one statistic of a match's score breakdown, as each alliance
// scored it.
type BreakdownRow struct {
	// Key is the API's own field name, for a caller that wants to match on it.
	Key string
	// Label is Key written as words.
	Label string
	Red   string
	Blue  string
}

// The breakdown fields worth reading first. The rest of a breakdown is the
// game's bookkeeping; these are the numbers an alliance is judged on.
const (
	breakdownTotalPoints = "totalPoints"
	breakdownRP          = "rp"
)

// penaltyKeys are the fields that explain a score without being part of the
// game, so they sit after the scoring detail rather than among it.
var penaltyKeys = []string{"foulCount", "techFoulCount", "adjustPoints"}

// CompareBreakdowns pairs the two alliances' score breakdowns into one table.
//
// The fields change every season and are documented nowhere, so they cannot be
// interpreted — but they can be ordered usefully and read side by side, which
// is how a breakdown is actually used: what did they score that we did not.
// The order is the total, then the ranking points, then everything else that
// scores points, then the penalties, then the rest alphabetically.
//
// Unless full is set, a row neither alliance did anything in is dropped — zero,
// false, "None" or absent on both sides — along with the season's own
// constants, the thresholds a bonus is measured against. A 2024 breakdown
// carries some forty fields per alliance, most of them nothing, and the ones
// that moved are the story.
func CompareBreakdowns(m api.Match, full bool) []BreakdownRow {
	red := breakdownIndex(AllianceBreakdown(m, AllianceRed))
	blue := breakdownIndex(AllianceBreakdown(m, AllianceBlue))
	if len(red) == 0 && len(blue) == 0 {
		return nil
	}

	keys := make([]string, 0, len(red)+len(blue))
	seen := make(map[string]bool, len(red)+len(blue))
	for _, side := range []map[string]string{red, blue} {
		for key := range side {
			if !seen[key] {
				seen[key] = true
				keys = append(keys, key)
			}
		}
	}
	slices.SortFunc(keys, CompareBreakdownOrder)

	rows := make([]BreakdownRow, 0, len(keys))
	for _, key := range keys {
		row := BreakdownRow{Key: key, Label: HumanizeKey(key), Red: red[key], Blue: blue[key]}
		if !full && (isNothing(row.Red) && isNothing(row.Blue) || isThreshold(key)) {
			continue
		}
		rows = append(rows, row)
	}
	return rows
}

// isThreshold reports whether a field is one of the season's own constants —
// the number of notes a coopertition bonus needs, the points an ensemble
// bonus is worth — rather than something an alliance did. They are the same
// in every match of the season, so both columns always agree and the row says
// nothing about the match; --full still has them for anyone checking the
// game manual against the data.
func isThreshold(key string) bool {
	return strings.Contains(strings.ToLower(key), "threshold")
}

// PointsBandLen is how many of a breakdown's leading rows are the scoring
// summary — the total, the ranking points, everything else that scores, and
// the penalties that explain the total — as opposed to the game's own detail
// underneath. A caller separates the two with it; 0 or len(rows) means there
// is nothing to separate.
func PointsBandLen(rows []BreakdownRow) int {
	for i, row := range rows {
		if breakdownRank(row.Key) == rankOther {
			return i
		}
	}
	return len(rows)
}

func breakdownIndex(kvs []KV) map[string]string {
	out := make(map[string]string, len(kvs))
	for _, kv := range kvs {
		out[kv.Key] = kv.Value
	}
	return out
}

// isNothing reports whether a rendered value is the game's "did not happen":
// zero, false or absent. Such a row is only worth printing under --full.
//
// "None" is in the list because recent seasons write an unfilled slot that way
// — a string, not a null — so "Auto Tower Robot 1  None  None" is a row about
// a robot that did nothing, spelled differently from the zeroes beside it.
func isNothing(value string) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "", "0", "no", "none":
		return true
	default:
		return false
	}
}

// CompareBreakdownOrder orders two breakdown keys the way a reader wants to
// see them: the total first, then ranking points, then anything scored in
// points, then the penalties in their fixed order, then everything else
// alphabetically. It is a cmp-style comparator for slices.SortFunc.
func CompareBreakdownOrder(a, b string) int {
	if c := cmp.Compare(breakdownRank(a), breakdownRank(b)); c != 0 {
		return c
	}
	if breakdownRank(a) == rankPenalty {
		return cmp.Compare(penaltyIndex(a), penaltyIndex(b))
	}
	return cmp.Compare(a, b)
}

// The bands a breakdown key falls into, in printing order.
const (
	rankTotal = iota
	rankRP
	rankPoints
	rankPenalty
	rankOther
)

func breakdownRank(key string) int {
	switch key {
	case breakdownTotalPoints:
		return rankTotal
	case breakdownRP:
		return rankRP
	}
	if penaltyIndex(key) >= 0 {
		return rankPenalty
	}
	if strings.Contains(key, "Points") {
		return rankPoints
	}
	return rankOther
}

func penaltyIndex(key string) int {
	for i, k := range penaltyKeys {
		if k == key {
			return i
		}
	}
	return -1
}

// acronyms are the field names that are initialisms rather than words, and
// would read as "Rp" if they were capitalised like one.
var acronyms = map[string]string{"rp": "RP", "dq": "DQ", "id": "ID"}

// HumanizeKey writes an API field name as words: "autoAmpNoteCount" becomes
// "Auto Amp Note Count". Dotted paths, which is how a nested breakdown field
// arrives, keep their segments in order.
func HumanizeKey(key string) string {
	segments := strings.Split(key, ".")
	words := make([]string, 0, len(segments)*3)
	for _, segment := range segments {
		words = append(words, splitWords(segment)...)
	}
	for i, w := range words {
		words[i] = capitalize(w)
	}
	return strings.Join(words, " ")
}

// splitWords breaks a camelCase (or camelCase123) name into its words. A
// single letter in front of a number stays with it, so a rule number such as
// "g424Penalty" reads as "G424 Penalty" rather than "G 424 Penalty".
func splitWords(s string) []string {
	var words []string
	var current []rune
	flush := func() {
		if len(current) > 0 {
			words = append(words, string(current))
			current = nil
		}
	}
	runes := []rune(s)
	for i, r := range runes {
		switch {
		case i == 0:
		case unicode.IsUpper(r) && !unicode.IsUpper(runes[i-1]),
			unicode.IsUpper(r) && i+1 < len(runes) && unicode.IsLower(runes[i+1]),
			unicode.IsDigit(r) != unicode.IsDigit(runes[i-1]):
			flush()
		}
		current = append(current, r)
	}
	flush()

	// Re-join a lone letter with the number that follows it.
	for i := 0; i+1 < len(words); i++ {
		if len([]rune(words[i])) == 1 && isDigits(words[i+1]) {
			words[i] += words[i+1]
			words = append(words[:i+1], words[i+2:]...)
		}
	}
	return words
}

func isDigits(s string) bool {
	for _, r := range s {
		if !unicode.IsDigit(r) {
			return false
		}
	}
	return s != ""
}

func capitalize(word string) string {
	if word == "" {
		return word
	}
	if full, ok := acronyms[strings.ToLower(word)]; ok {
		return full
	}
	runes := []rune(word)
	runes[0] = unicode.ToUpper(runes[0])
	return string(runes)
}
