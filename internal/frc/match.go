// Package frc holds the FRC competition rules the commands share: how matches
// are ordered, what a match is called on a scoreboard, when it counts as
// played, and how a team is marked when it was a surrogate or disqualified.
//
// It deliberately knows nothing about output formats or the command line, so
// the same rules apply to a table, a detail view and a JSON filter alike.
package frc

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/the-blue-alliance/tba-cli/internal/api"
)

// Competition levels, as the API spells them in comp_level.
const (
	LevelQual         = "qm"
	LevelEighthFinal  = "ef"
	LevelQuarterFinal = "qf"
	LevelSemiFinal    = "sf"
	LevelFinal        = "f"
)

// UnknownCompLevel is the sort key given to a level this build does not know,
// so a new level from a future season lands at the end instead of in the
// middle of the qualification matches.
const UnknownCompLevel = 99

var compLevelOrder = map[string]int{
	LevelQual:         0,
	LevelEighthFinal:  1,
	LevelQuarterFinal: 2,
	LevelSemiFinal:    3,
	LevelFinal:        4,
}

// CompLevelOrder returns the position of a competition level in playing order:
// qualification matches, then the elimination rounds from widest to narrowest.
// Alphabetical order would put the finals before the quarterfinals, so every
// match listing sorts on this instead.
func CompLevelOrder(level string) int {
	if n, ok := compLevelOrder[strings.ToLower(strings.TrimSpace(level))]; ok {
		return n
	}
	return UnknownCompLevel
}

// SortMatches orders matches the way a competition plays them: by level, then
// by set, then by match number. The sort is stable, so matches the API
// considers equal keep the order it returned them in.
func SortMatches(matches []api.Match) {
	sort.SliceStable(matches, func(i, j int) bool {
		return lessByPlayOrder(matches[i], matches[j])
	})
}

func lessByPlayOrder(a, b api.Match) bool {
	if oa, ob := CompLevelOrder(a.CompLevel), CompLevelOrder(b.CompLevel); oa != ob {
		return oa < ob
	}
	if a.SetNumber != b.SetNumber {
		return a.SetNumber < b.SetNumber
	}
	return a.MatchNumber < b.MatchNumber
}

// SortByTime orders matches by their best known time, earliest first. Matches
// with no time at all sort last, and ties fall back to playing order, so an
// event that has not published a schedule still comes out in a sensible order.
func SortByTime(matches []api.Match) {
	SortMatches(matches)
	sort.SliceStable(matches, func(i, j int) bool {
		a, _ := BestTime(matches[i])
		b, _ := BestTime(matches[j])
		if a == nil || b == nil {
			return a != nil && b == nil
		}
		return *a < *b
	})
}

// Playoff bracket formats, as the API reports them in an event's playoff_type.
const (
	PlayoffBracket8Team    = 0
	PlayoffBracket16Team   = 1
	PlayoffBracket4Team    = 2
	PlayoffAvgScore8Team   = 3
	PlayoffRoundRobin6Team = 6
	PlayoffDoubleElim8Team = 10
	PlayoffDoubleElim4Team = 11
)

// FirstDoubleElimYear is the first season FRC ran the double-elimination
// playoff. It is only used to guess how to label a playoff match when the
// event's playoff_type could not be fetched.
const FirstDoubleElimYear = 2023

// IsDoubleElim reports whether a playoff_type is one of the double-elimination
// brackets, in which each semifinal set holds exactly one match and is
// therefore named by its set number alone ("SF 3", not "SF 3-1").
func IsDoubleElim(playoffType *int) bool {
	if playoffType == nil {
		return false
	}
	return *playoffType == PlayoffDoubleElim8Team || *playoffType == PlayoffDoubleElim4Team
}

// MatchLabel renders the name a match is announced by: "Qual 12", "QF 2-1",
// "SF 3", "Final 2".
//
// Playoff sets are numbered differently depending on the bracket. In a
// double-elimination bracket every semifinal set is a single match, so the set
// number alone identifies it ("SF 3"). In a legacy best-of-three bracket, and
// in an Einstein round robin, both numbers are needed ("SF 1-2"). Finals are
// numbered by match in every format, since a final set is the series.
//
// playoffType may be nil when the event could not be fetched; the season the
// match belongs to is then used as a guess.
func MatchLabel(m api.Match, playoffType *int) string {
	level := strings.ToLower(strings.TrimSpace(m.CompLevel))
	switch level {
	case LevelQual:
		return fmt.Sprintf("Qual %d", m.MatchNumber)
	case LevelFinal:
		return fmt.Sprintf("Final %d", m.MatchNumber)
	case LevelSemiFinal:
		if isDoubleElimFor(m, playoffType) {
			return fmt.Sprintf("SF %d", m.SetNumber)
		}
		return fmt.Sprintf("SF %d-%d", m.SetNumber, m.MatchNumber)
	case LevelQuarterFinal:
		return fmt.Sprintf("QF %d-%d", m.SetNumber, m.MatchNumber)
	case LevelEighthFinal:
		return fmt.Sprintf("EF %d-%d", m.SetNumber, m.MatchNumber)
	case "":
		return m.Key
	default:
		return fmt.Sprintf("%s %d-%d", strings.ToUpper(level), m.SetNumber, m.MatchNumber)
	}
}

func isDoubleElimFor(m api.Match, playoffType *int) bool {
	if playoffType != nil {
		return IsDoubleElim(playoffType)
	}
	return MatchYear(m) >= FirstDoubleElimYear
}

// MatchYear reads the season out of a match, which is the first four
// characters of its event key ("2024cthar"). It returns 0 when the key is
// missing or malformed.
func MatchYear(m api.Match) int {
	key := m.EventKey
	if key == "" {
		key = m.Key
	}
	if len(key) < 4 {
		return 0
	}
	year, err := strconv.Atoi(key[:4])
	if err != nil {
		return 0
	}
	return year
}

// Match statuses, as shown in a Status column.
const (
	StatusScheduled = "Scheduled"
	StatusPlayed    = "Played"
)

// Alliance colors.
const (
	AllianceRed  = "red"
	AllianceBlue = "blue"
)

// Winner values that are not an alliance color.
const (
	WinnerTie = "tie"
)

// UnplayedScore is the score the API reports for a match that has not been
// played yet.
const UnplayedScore = -1

// Score returns an alliance's score, or UnplayedScore when the match carries
// no such alliance.
func Score(m api.Match, color string) int {
	a, ok := m.Alliances[color]
	if !ok {
		return UnplayedScore
	}
	return a.Score
}

// Played reports whether a match has a result. The API scores both alliances
// -1 until the match is played, so a single real score is enough.
func Played(m api.Match) bool {
	return Score(m, AllianceRed) != UnplayedScore || Score(m, AllianceBlue) != UnplayedScore
}

// MatchStatus is Played's answer as a column value.
func MatchStatus(m api.Match) string {
	if Played(m) {
		return StatusPlayed
	}
	return StatusScheduled
}

// Winner returns the winning alliance color, "tie" for a played match the API
// reports no winner for, or "" for a match that has not been played. Older
// seasons (2015 and earlier) had no ties, and simply always name a winner.
func Winner(m api.Match) string {
	if !Played(m) {
		return ""
	}
	if m.WinningAlliance == "" {
		return WinnerTie
	}
	return m.WinningAlliance
}

// Marks appended to a team number to explain an unusual appearance.
const (
	// MarkSurrogate flags a team playing an extra match that does not count
	// towards its own ranking.
	MarkSurrogate = "*"
	// MarkDQ flags a team disqualified from the match.
	MarkDQ = "!"
)

// MarkChars is every mark character, for testing whether a rendered cell needs
// the legend.
const MarkChars = MarkSurrogate + MarkDQ

// Legend explains the marks. It is printed to stderr, not stdout, so it never
// lands in a file a user is piping the table into.
const Legend = "* surrogate  ! disqualified"

// MarkTeam renders a team's number within an alliance, with a trailing mark
// when the team was a surrogate or was disqualified. A team that is both gets
// both marks, surrogate first.
func MarkTeam(key string, a api.Alliance) string {
	out := TeamNumber(key)
	if containsKey(a.SurrogateTeamKeys, key) {
		out += MarkSurrogate
	}
	if containsKey(a.DQTeamKeys, key) {
		out += MarkDQ
	}
	return out
}

// MarkedTeams renders every team on an alliance, in station order.
func MarkedTeams(a api.Alliance) []string {
	out := make([]string, len(a.TeamKeys))
	for i, key := range a.TeamKeys {
		out[i] = MarkTeam(key, a)
	}
	return out
}

// TeamNumber turns "frc177" into "177".
func TeamNumber(key string) string {
	return strings.TrimPrefix(key, "frc")
}

// HasTeam reports whether a team played in a match, on either alliance.
func HasTeam(m api.Match, teamKey string) bool {
	for _, a := range m.Alliances {
		if containsKey(a.TeamKeys, teamKey) {
			return true
		}
	}
	return false
}

// AllianceOf returns the color of the alliance a team played on, or "" when it
// did not play in the match.
func AllianceOf(m api.Match, teamKey string) string {
	for _, color := range []string{AllianceRed, AllianceBlue} {
		if a, ok := m.Alliances[color]; ok && containsKey(a.TeamKeys, teamKey) {
			return color
		}
	}
	return ""
}

// Station returns a team's driver station number (1, 2 or 3) within its
// alliance, or 0 when the team did not play in the match. The API lists team
// keys in station order.
func Station(m api.Match, teamKey string) int {
	for _, a := range m.Alliances {
		for i, key := range a.TeamKeys {
			if key == teamKey {
				return i + 1
			}
		}
	}
	return 0
}

func containsKey(keys []string, want string) bool {
	for _, k := range keys {
		if k == want {
			return true
		}
	}
	return false
}
