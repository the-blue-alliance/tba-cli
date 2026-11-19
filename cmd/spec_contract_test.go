package cmd

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"path"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"sync"
	"testing"

	"github.com/the-blue-alliance/tba-cli/internal/apispec"
)

// Every body newFakeTBA serves is checked here against the GET 200 response
// schema of the spec path it stands in for, and every request a command sends
// must name a path the spec defines. A fixture that breaks the spec on purpose
// says so with offSpec.

// offSpecBody is a route body that is deliberately not what the API sends.
type offSpecBody struct {
	reason string
	body   any
}

// offSpec exempts one route body from the spec check; reason says why.
func offSpec(reason string, body any) offSpecBody {
	return offSpecBody{reason: reason, body: body}
}

// specException tolerates one known disagreement between a fixture and the
// spec where the spec, not the fixture, is believed to be wrong.
type specException struct {
	templates []string // spec path templates the fixture stands in for
	instances []string // JSON pointers into the body; "*" matches one segment
	message   string   // substring of the problem's message
}

var matchTemplates = []string{
	"/match/{match_key}",
	"/event/{event_key}/matches",
	"/team/{team_key}/event/{event_key}/matches",
	"/team/{team_key}/matches/{year}",
}

var specExceptions = []specException{
	// Team_Event_Status_playoff is itself nullable, so a null playoff matches
	// both branches of its oneOf. The API sends null for a team not in the
	// playoffs.
	{
		templates: []string{"/event/{event_key}/teams/statuses", "/team/{team_key}/event/{event_key}/status"},
		instances: []string{"/*/playoff", "/playoff"},
		message:   "'oneOf' failed, subschemas 0, 1 matched",
	},
	// The 2022-and-later breakdown schemas require no fields, so any season's
	// breakdown matches several branches of score_breakdown's oneOf.
	{
		templates: matchTemplates,
		instances: []string{"/score_breakdown", "/*/score_breakdown"},
		message:   "'oneOf' failed, subschemas",
	},
}

var (
	specOnce sync.Once
	spec     *apispec.Spec
	specErr  error
)

func loadSpec(t *testing.T) *apispec.Spec {
	t.Helper()
	specOnce.Do(func() { spec, specErr = apispec.Load() })
	if specErr != nil {
		t.Fatalf("loading the vendored API spec: %v", specErr)
	}
	return spec
}

// checkFixture fails the test if body is not a valid 200 response for the spec
// path that route stands in for.
func checkFixture(t *testing.T, route string, body []byte) {
	t.Helper()
	for _, msg := range fixtureProblems(loadSpec(t), route, body) {
		t.Error(msg)
	}
}

func fixtureProblems(s *apispec.Spec, route string, body []byte) []string {
	tmpl, ok := s.MatchPath(route)
	if !ok {
		return []string{fmt.Sprintf("fixture route %s matches no path in the API spec", route)}
	}
	problems, err := s.Check(apispec.ResponsePointer(tmpl), body)
	if err != nil {
		return []string{fmt.Sprintf("fixture for %s (%s): %v", route, tmpl, err)}
	}
	var out []string
	for _, p := range problems {
		if !excepted(tmpl, p) {
			out = append(out, fmt.Sprintf("fixture for %s does not match the spec for %s: %s", route, tmpl, p))
		}
	}
	return out
}

func excepted(tmpl string, p apispec.Problem) bool {
	for _, e := range specExceptions {
		if !slices.Contains(e.templates, tmpl) || !strings.Contains(p.Message, e.message) {
			continue
		}
		for _, pattern := range e.instances {
			if pointerMatches(pattern, p.Instance) {
				return true
			}
		}
	}
	return false
}

func pointerMatches(pattern, ptr string) bool {
	ok, err := path.Match(pattern, ptr)
	return err == nil && ok
}

// checkRequestPaths fails the test for every request the fake saw that names
// no path in the spec.
func checkRequestPaths(t *testing.T, reqs []recordedRequest) {
	t.Helper()
	s := loadSpec(t)
	seen := map[string]bool{}
	for _, r := range reqs {
		if seen[r.Path] {
			continue
		}
		seen[r.Path] = true
		if _, ok := s.MatchPath(r.Path); !ok {
			t.Errorf("command requested %s, which matches no path in the API spec", r.Path)
		}
	}
}

func TestFixtureCheckRejectsAnInventedField(t *testing.T) {
	got := fixtureProblems(loadSpec(t), "/team/frc177/robots",
		[]byte(`[{"key": "frc177_2024", "robot_name": "Bobcat", "team_key": "frc177", "year": 2024, "color": "blue"}]`))
	if len(got) != 1 || !strings.Contains(got[0], "/0/color") {
		t.Errorf("got %v, want one problem at /0/color", got)
	}
}

func TestFixtureCheckRejectsAMissingRequiredField(t *testing.T) {
	got := fixtureProblems(loadSpec(t), "/team/frc177/robots",
		[]byte(`[{"key": "frc177_2024", "robot_name": "Bobcat", "year": 2024}]`))
	if len(got) != 1 || !strings.Contains(got[0], "team_key") {
		t.Errorf("got %v, want team_key reported missing", got)
	}
}

func TestFixtureCheckRejectsARouteTheSpecLacks(t *testing.T) {
	if got := fixtureProblems(loadSpec(t), "/team/frc177/matches", []byte(`[]`)); len(got) != 1 {
		t.Errorf("got %v, want the route rejected", got)
	}
}

func TestFixtureCheckNarrowsABreakdownToItsSeason(t *testing.T) {
	// totalAutoPoints is a 2026 field, not a 2024 one.
	match := `{"key": "2024cthar_qm1", "comp_level": "qm", "set_number": 1, "match_number": 1,
	  "event_key": "2024cthar", "winning_alliance": "", "time": null, "actual_time": null,
	  "predicted_time": null, "post_result_time": null, "videos": [],
	  "alliances": {"red": {"score": 0, "team_keys": [], "surrogate_team_keys": [], "dq_team_keys": []},
	                "blue": {"score": 0, "team_keys": [], "surrogate_team_keys": [], "dq_team_keys": []}},
	  "score_breakdown": {"red": {"totalAutoPoints": 1}, "blue": {}}}`
	got := fixtureProblems(loadSpec(t), "/match/2024cthar_qm1", []byte(match))
	if len(got) != 1 || !strings.Contains(got[0], "/score_breakdown/red/totalAutoPoints") {
		t.Errorf("got %v, want totalAutoPoints undeclared for 2024", got)
	}
}

// Request paths a test never exercises are still checked: every API path
// format in the command sources must name a spec path. open.go builds website
// paths, not API ones.
func TestEveryPathFormatInTheSourceIsASpecPath(t *testing.T) {
	s := loadSpec(t)
	files, err := filepath.Glob("*.go")
	if err != nil {
		t.Fatal(err)
	}
	verbs := strings.NewReplacer("%s", "x", "%d", "2024")
	checked := 0
	for _, file := range files {
		if strings.HasSuffix(file, "_test.go") || file == "open.go" {
			continue
		}
		f, err := parser.ParseFile(token.NewFileSet(), file, nil, 0)
		if err != nil {
			t.Fatal(err)
		}
		ast.Inspect(f, func(n ast.Node) bool {
			lit, ok := n.(*ast.BasicLit)
			if !ok || lit.Kind != token.STRING {
				return true
			}
			v, err := strconv.Unquote(lit.Value)
			if err != nil || !strings.HasPrefix(v, "/") || (!strings.Contains(v, "%") && v != "/status" && v != "/sponsors") {
				return true
			}
			checked++
			if _, ok := s.MatchPath(verbs.Replace(v)); !ok {
				t.Errorf("%s: %q matches no path in the API spec", file, v)
			}
			return true
		})
	}
	if checked < 30 {
		t.Errorf("checked only %d path formats; has the request code moved?", checked)
	}
}
