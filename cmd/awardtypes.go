package cmd

import (
	"cmp"
	"fmt"
	"slices"
	"strconv"
	"strings"

	"github.com/the-blue-alliance/tba-cli/internal/clierr"
)

// TBA identifies awards by a number. The numbers are stable and are what the
// API returns, but nobody knows them by heart, and `--type 0` returning the
// Impact Award while `--type 9` returns Engineering Inspiration is not
// something a listing of integers can be guessed from. The table below is the
// award_type enum from The Blue Alliance's API, in code order.

// awardType is one entry of TBA's award_type enum.
type awardType struct {
	code int
	name string
}

var awardTypes = []awardType{
	{0, "Chairman's/Impact"},
	{1, "Winner"},
	{2, "Finalist"},
	{3, "Woodie Flowers"},
	{4, "Dean's List"},
	{5, "Volunteer"},
	{6, "Founders"},
	{7, "Bart Kamen Memorial"},
	{8, "Make It Loud"},
	{9, "Engineering Inspiration"},
	{10, "Rookie All Star"},
	{11, "Gracious Professionalism"},
	{12, "Coopertition"},
	{13, "Judges"},
	{14, "Highest Rookie Seed"},
	{15, "Rookie Inspiration"},
	{16, "Industrial Design"},
	{17, "Quality"},
	{18, "Safety"},
	{19, "Sportsmanship"},
	{20, "Creativity"},
	{21, "Engineering Excellence"},
	{22, "Entrepreneurship"},
	{23, "Excellence in Design"},
	{24, "Excellence in Design CAD"},
	{25, "Excellence in Design Animation"},
	{26, "Driving Tomorrow's Technology"},
	{27, "Imagery"},
	{28, "Media and Technology"},
	{29, "Innovation in Control"},
	{30, "Spirit"},
	{31, "Website"},
	{32, "Visualization"},
	{33, "Autodesk Inventor"},
	{34, "Future Innovator"},
	{35, "Recognition of Extraordinary Service"},
	{36, "Outstanding Cart"},
	{37, "WSU Aim Higher"},
	{38, "Leadership in Control"},
	{39, "#1 Seed"},
	{40, "Incredible Play"},
	{41, "People's Choice Animation"},
	{42, "Visualization Rising Star"},
	{43, "Best Offensive Round Robin"},
	{44, "Best Play of the Day"},
	{45, "Featherweight in the Finals"},
	{46, "Most Photogenic"},
	{47, "Outstanding Defense"},
	{48, "Power to Simplify"},
	{49, "Against All Odds"},
	{50, "Rising Star"},
	{51, "Chairman's Honorable Mention"},
	{52, "Content Communication Honorable Mention"},
	{53, "Technical Execution Honorable Mention"},
	{54, "Realization"},
	{55, "Realization Honorable Mention"},
	{56, "Design Your Future"},
	{57, "Design Your Future Honorable Mention"},
	{58, "Special Recognition Character Animation"},
	{59, "High Score"},
	{60, "Teacher Pioneer"},
	{61, "Best Craftsmanship"},
	{62, "Best Defensive Match"},
	{63, "Play of the Day"},
	{64, "Programming"},
	{65, "Professionalism"},
	{66, "Solid Design"},
	{67, "FOC Champion"},
	{68, "FOC Finalist"},
	{69, "Wildcard"},
	{70, "Chairman's Finalist"},
	{71, "Other"},
	{72, "Autonomous"},
	{73, "Innovation Challenge Semi-Finalist"},
	{74, "Rookie Game Changer"},
	{75, "Skills Competition Winner"},
	{76, "Skills Competition Finalist"},
	{77, "Innovation Challenge Finalist"},
	{78, "Innovation Challenge Winner"},
	{79, "Sustainability"},
	{80, "Excellence in Engineering"},
	{81, "Digital Animation"},
	{82, "Gracious Professionalism Team"},
	{83, "Judges Team"},
}

// awardTypeByCode indexes the table for the Type column, which looks a code up
// once per row.
var awardTypeByCode = func() map[int]string {
	m := make(map[int]string, len(awardTypes))
	for _, t := range awardTypes {
		m[t.code] = t.name
	}
	return m
}()

// awardTypeName is what goes in the Type column. A code TBA has added since
// this table was written is shown as the number itself rather than as a blank
// cell, so the row still says which awards are alike.
func awardTypeName(code int) string {
	if name, ok := awardTypeByCode[code]; ok {
		return name
	}
	return strconv.Itoa(code)
}

// parseAwardType resolves a --type value to an award_type code.
//
// A value is either a code (`--type 0`) or a name, matched case-insensitively:
// a name that is spelled out in full wins outright, and otherwise the value has
// to appear in exactly one name, so `--type impact` works while `--type
// chairman` says which three awards it could have meant.
func parseAwardType(spec string) (int, error) {
	spec = strings.TrimSpace(spec)
	if spec == "" {
		return 0, clierr.Usage("--type needs an award name or code (e.g. --type impact or --type 0)")
	}
	if code, err := strconv.Atoi(spec); err == nil {
		if _, ok := awardTypeByCode[code]; !ok {
			return 0, clierr.Usage("unknown award type %d (TBA's codes run 0 to %d; a name works too, e.g. --type impact)",
				code, awardTypes[len(awardTypes)-1].code)
		}
		return code, nil
	}

	want := strings.ToLower(spec)
	var matches []awardType
	for _, t := range awardTypes {
		name := strings.ToLower(t.name)
		if name == want {
			return t.code, nil
		}
		if strings.Contains(name, want) {
			matches = append(matches, t)
		}
	}
	switch len(matches) {
	case 1:
		return matches[0].code, nil
	case 0:
		return 0, clierr.Usage("unknown award type %q (did you mean: %s?)", spec, strings.Join(nearestAwardTypes(want), ", "))
	default:
		return 0, clierr.Usage("--type %q matches %d awards (%s); use a longer name or the code",
			spec, len(matches), strings.Join(awardTypeNames(matches), ", "))
	}
}

func awardTypeNames(types []awardType) []string {
	names := make([]string, len(types))
	for i, t := range types {
		names[i] = fmt.Sprintf("%s (%d)", t.name, t.code)
	}
	return names
}

// nearestAwardTypes suggests a few names for a value that matched none, so the
// error is a place to start rather than a wall of eighty-four awards.
func nearestAwardTypes(want string) []string {
	type scored struct {
		name  string
		score int
	}
	var candidates []scored
	for _, t := range awardTypes {
		if n := commonPrefixLen(strings.ToLower(t.name), want); n > 0 {
			candidates = append(candidates, scored{t.name, n})
		}
	}
	slices.SortStableFunc(candidates, func(a, b scored) int { return cmp.Compare(b.score, a.score) })

	const suggestions = 4
	out := make([]string, 0, suggestions)
	for _, c := range candidates {
		if len(out) == suggestions {
			break
		}
		out = append(out, c.name)
	}
	if len(out) > 0 {
		return out
	}
	// Nothing looked alike, so name the awards people ask for most often.
	return []string{"Impact", "Winner", "Finalist", "Engineering Inspiration"}
}

func commonPrefixLen(a, b string) int {
	n := 0
	for n < len(a) && n < len(b) && a[n] == b[n] {
		n++
	}
	return n
}
