package frc

import (
	"cmp"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/the-blue-alliance/tba-cli/internal/api"
)

// The orderings the whole CLI shares.
//
// Every one of them is a cmp-style comparator meant for slices.SortFunc or
// slices.SortStableFunc: negative when a comes first, zero when neither does,
// positive when b comes first. They are exported because the same question —
// which team, match or event comes first — is asked from the table commands,
// from `event watch` and from `event export`, and an ordering written out
// again at each of those call sites is an ordering that can disagree with
// itself. A closure passed to sort.Slice cannot be shared or tested; a named
// comparator is both.

// CompareTeamKeys orders team keys the way anyone reading a list of teams
// expects: by team number, so 177 comes before 1073 comes before 5507. Sorted
// as text they come out 1073, 177, 5507, which is nobody's idea of a team
// list.
//
// A key whose number will not parse sorts after every key whose will, and ties
// fall back to the text, so the order stays total and reproducible whatever
// the API sends. Map iteration order is random, so every table built from an
// object keyed by team goes through here.
func CompareTeamKeys(a, b string) int {
	an, aok := TeamKeyNumber(a)
	bn, bok := TeamKeyNumber(b)
	switch {
	case aok && bok:
		if c := cmp.Compare(an, bn); c != 0 {
			return c
		}
	case aok:
		return -1
	case bok:
		return 1
	}
	return strings.Compare(a, b)
}

// TeamKeyNumber reads the number out of a team key: 177 from "frc177", and
// from a bare "177". It reports false for anything else, which is how a
// comparison keeps a key it cannot read without pretending it is team zero.
func TeamKeyNumber(key string) (int, bool) {
	s := strings.TrimSpace(key)
	// The prefix is matched whatever its case, because a team argument reaches
	// this both as the API spells it and as a person typed it.
	if len(s) >= 3 && strings.EqualFold(s[:3], "frc") {
		s = s[3:]
	}
	n, err := strconv.Atoi(s)
	if err != nil {
		return 0, false
	}
	return n, true
}

// CompareMatches orders matches the way a competition plays them: by level,
// then by set, then by match number. Alphabetical order by key would put the
// finals before the quarterfinals and qm10 before qm2.
func CompareMatches(a, b api.Match) int {
	if c := cmp.Compare(CompLevelOrder(a.CompLevel), CompLevelOrder(b.CompLevel)); c != 0 {
		return c
	}
	if c := cmp.Compare(a.SetNumber, b.SetNumber); c != 0 {
		return c
	}
	return cmp.Compare(a.MatchNumber, b.MatchNumber)
}

// CompareMatchTimes orders matches by their best known time, earliest first.
// A match with no time at all sorts last: the schedule has not reached it, and
// putting it at the top would answer "what is next" with the one match nobody
// can say anything about.
//
// It compares only the clock, so a caller who wants playing order to break the
// ties sorts by CompareMatches first and then stably by this.
func CompareMatchTimes(a, b api.Match) int {
	at, _ := BestTime(a)
	bt, _ := BestTime(b)
	switch {
	case at == nil && bt == nil:
		return 0
	case at == nil:
		return 1
	case bt == nil:
		return -1
	}
	return cmp.Compare(*at, *bt)
}

// matchKeySuffixPattern splits the part of a match key after the event:
// "qm12" or, for a playoff match, "sf3m1".
var matchKeySuffixPattern = regexp.MustCompile(`^(qm|ef|qf|sf|f)([0-9]+)(?:m([0-9]+))?$`)

// CompareMatchKeys orders match keys the way CompareMatches orders matches:
// level, then set, then match number. It exists for the callers that have only
// keys — an insight names its matches by key and sends no match objects — so
// that a list of keys and a list of matches cannot come out in different
// orders.
//
// A key that does not parse sorts after every key that does, and ties fall
// back to the text, so the order stays total whatever the API sends.
func CompareMatchKeys(a, b string) int {
	la, sa, ma, _ := matchKeyOrder(a)
	lb, sb, mb, _ := matchKeyOrder(b)
	if c := cmp.Compare(la, lb); c != 0 {
		return c
	}
	if c := cmp.Compare(sa, sb); c != 0 {
		return c
	}
	if c := cmp.Compare(ma, mb); c != 0 {
		return c
	}
	return strings.Compare(a, b)
}

// matchKeyOrder ranks a match key for play order: competition level first,
// then set number, then match number. An unparsable key is ranked at the
// unknown level, which is past every real one.
func matchKeyOrder(key string) (level, set, match int, ok bool) {
	suffix := key
	if i := strings.LastIndex(key, "_"); i >= 0 {
		suffix = key[i+1:]
	}
	m := matchKeySuffixPattern.FindStringSubmatch(suffix)
	if m == nil {
		return UnknownCompLevel, 0, 0, false
	}
	level = CompLevelOrder(m[1])
	first, _ := strconv.Atoi(m[2])
	if m[3] == "" {
		// A qualification key carries only the match number.
		return level, 0, first, true
	}
	second, _ := strconv.Atoi(m[3])
	return level, first, second, true
}

// MatchFromKey rebuilds enough of a match from its key to label and order it,
// for the callers a match list never reaches: a prediction document names
// matches the list does not carry, and a key is all there is to go on.
//
// A key it cannot read comes back as a match with nothing but the key, which
// MatchLabel prints as the key itself.
func MatchFromKey(key string) api.Match {
	m := api.Match{Key: key}
	i := strings.LastIndex(key, "_")
	if i < 0 {
		return m
	}
	m.EventKey = key[:i]
	parts := matchKeySuffixPattern.FindStringSubmatch(key[i+1:])
	if parts == nil {
		return m
	}
	m.CompLevel = parts[1]
	first, _ := strconv.Atoi(parts[2])
	if parts[3] == "" {
		// A qualification key carries only the match number.
		m.SetNumber, m.MatchNumber = 1, first
		return m
	}
	m.SetNumber = first
	m.MatchNumber, _ = strconv.Atoi(parts[3])
	return m
}

// CompareEvents puts a season's events in the order they are played, earliest
// start first. It is the order anyone reading a list of events expects, and
// the API returns them in neither that order nor any other.
//
// Events sharing a start date, and events whose start_date is missing or
// malformed, fall back to their key, which keeps the order total and
// reproducible. A dated event is placed before an undated one, which is not a
// real season but is a real API answer. The dates are compared as calendar
// days in UTC, so the machine's time zone cannot reorder two events a day
// apart.
func CompareEvents(a, b api.Event) int {
	at, aok := ParseDate(a.StartDate, time.UTC)
	bt, bok := ParseDate(b.StartDate, time.UTC)
	if aok != bok {
		if aok {
			return -1
		}
		return 1
	}
	if aok {
		if c := at.Compare(bt); c != 0 {
			return c
		}
	}
	return strings.Compare(a.Key, b.Key)
}

// CompareRankings orders an event's qualification rankings by rank, which is
// the one order a ranking table has. The API usually sends them that way and
// is not obliged to.
func CompareRankings(a, b api.Ranking) int {
	return cmp.Compare(a.Rank, b.Rank)
}

// CompareDistrictRankings orders a district's season rankings by rank, for the
// same reason.
func CompareDistrictRankings(a, b api.DistrictRanking) int {
	return cmp.Compare(a.Rank, b.Rank)
}
