package cmd

import (
	"fmt"
	"math"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"unicode"
)

// humanizeName turns a snake_case API identifier into title-cased words, so
// that "most_matches_played" reads as "Most Matches Played". A word that
// already carries capitals is left alone, which keeps acronyms such as "OPR"
// and "RP" intact.
func humanizeName(s string) string {
	words := strings.FieldsFunc(s, func(r rune) bool { return r == '_' || r == ' ' })
	for i, w := range words {
		if strings.ToLower(w) != w {
			continue
		}
		runes := []rune(w)
		runes[0] = unicode.ToUpper(runes[0])
		words[i] = string(runes)
	}
	return strings.Join(words, " ")
}

// humanizeStatKey humanizes a flattened stat key one dotted segment at a time,
// so that "score_by_alliance.red" reads as "Score By Alliance.Red" and the
// nesting the API expressed stays visible.
func humanizeStatKey(key string) string {
	parts := strings.Split(key, ".")
	for i, p := range parts {
		parts[i] = humanizeName(p)
	}
	return strings.Join(parts, ".")
}

// normalizeBoardName folds a leaderboard or notable name to the form --board
// matches on, so that "Blue Banners", "blue_banners" and "BLUE BANNERS" all
// name the same board.
func normalizeBoardName(s string) string {
	return strings.Join(strings.Fields(strings.ToLower(strings.ReplaceAll(s, "_", " "))), " ")
}

// formatNumber renders an insight value at no more than two decimals, dropping
// the zeros a fixed-precision format would leave behind: 88.0 prints as "88"
// and 61.456 as "61.46".
func formatNumber(f float64) string {
	if math.IsNaN(f) || math.IsInf(f, 0) {
		return ""
	}
	s := strconv.FormatFloat(f, 'f', 2, 64)
	if strings.Contains(s, ".") {
		s = strings.TrimRight(s, "0")
		s = strings.TrimSuffix(s, ".")
	}
	if s == "" || s == "-" || s == "-0" {
		return "0"
	}
	return s
}

// formatInsightValue renders one decoded JSON value from an insights payload.
//
// The three-number arrays the insights endpoints are full of are counts:
// [12, 60, 20] means 12 of 60, which is 20 percent, and reads far better as
// "12/60 (20%)" than as three bare numbers. Any other array is simply joined,
// which is what the mixed arrays such as high_score need.
func formatInsightValue(v interface{}) string {
	switch t := v.(type) {
	case nil:
		return ""
	case bool:
		return strconv.FormatBool(t)
	case float64:
		return formatNumber(t)
	case string:
		return t
	case []interface{}:
		if count, total, percent, ok := countTotalPercent(t); ok {
			return fmt.Sprintf("%s/%s (%s%%)", formatNumber(count), formatNumber(total), formatNumber(percent))
		}
		parts := make([]string, len(t))
		for i, e := range t {
			parts[i] = formatInsightValue(e)
		}
		return strings.Join(parts, ", ")
	case map[string]interface{}:
		// Only reached for an object nested inside an array; a top-level one is
		// flattened into its own rows instead.
		pairs := flattenInsightStats("", t)
		parts := make([]string, len(pairs))
		for i, p := range pairs {
			parts[i] = p[0] + "=" + p[1]
		}
		return strings.Join(parts, ", ")
	default:
		return fmt.Sprintf("%v", v)
	}
}

// countTotalPercent recognises the [count, total, percent] triple.
func countTotalPercent(values []interface{}) (count, total, percent float64, ok bool) {
	if len(values) != 3 {
		return 0, 0, 0, false
	}
	nums := make([]float64, 3)
	for i, v := range values {
		f, isNum := v.(float64)
		if !isNum {
			return 0, 0, 0, false
		}
		nums[i] = f
	}
	return nums[0], nums[1], nums[2], true
}

// flattenInsightStats walks a decoded stats object into key/value pairs, giving
// a nested object's children dotted keys ("a.b"). Keys are sorted at every
// level so that Go's random map order cannot change the output between runs.
func flattenInsightStats(prefix string, stats map[string]interface{}) [][2]string {
	keys := make([]string, 0, len(stats))
	for k := range stats {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	out := make([][2]string, 0, len(keys))
	for _, k := range keys {
		name := k
		if prefix != "" {
			name = prefix + "." + k
		}
		if child, ok := stats[k].(map[string]interface{}); ok && len(child) > 0 {
			out = append(out, flattenInsightStats(name, child)...)
			continue
		}
		out = append(out, [2]string{name, formatInsightValue(stats[k])})
	}
	return out
}

// compLevelOrder is the order matches are played in, which is never the order
// their keys sort in alphabetically.
var compLevelOrder = map[string]int{"qm": 0, "ef": 1, "qf": 2, "sf": 3, "f": 4}

// matchKeySuffixPattern splits the part of a match key after the event:
// "qm12" or, for a playoff match, "sf3m1".
var matchKeySuffixPattern = regexp.MustCompile(`^(qm|ef|qf|sf|f)([0-9]+)(?:m([0-9]+))?$`)

// matchKeyOrder ranks a match key for play order: competition level first, then
// set number, then match number. A key that does not parse sorts last, so the
// comparison stays total whatever the API sends.
func matchKeyOrder(key string) (level, set, match int, ok bool) {
	suffix := key
	if i := strings.LastIndex(key, "_"); i >= 0 {
		suffix = key[i+1:]
	}
	m := matchKeySuffixPattern.FindStringSubmatch(suffix)
	if m == nil {
		return len(compLevelOrder), 0, 0, false
	}
	level = compLevelOrder[m[1]]
	first, _ := strconv.Atoi(m[2])
	if m[3] == "" {
		// A qualification key carries only the match number.
		return level, 0, first, true
	}
	second, _ := strconv.Atoi(m[3])
	return level, first, second, true
}

// sortMatchKeys puts match keys in play order: qm before ef, qf, sf and f, then
// by set and match number. Unparseable keys keep a stable place at the end.
func sortMatchKeys(keys []string) {
	sort.SliceStable(keys, func(i, j int) bool {
		li, si, mi, _ := matchKeyOrder(keys[i])
		lj, sj, mj, _ := matchKeyOrder(keys[j])
		switch {
		case li != lj:
			return li < lj
		case si != sj:
			return si < sj
		case mi != mj:
			return mi < mj
		default:
			return keys[i] < keys[j]
		}
	})
}
