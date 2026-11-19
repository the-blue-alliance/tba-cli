// Package apispec checks the CLI's test fixtures and response structs against
// a vendored copy of The Blue Alliance's API v3 OpenAPI document.
//
// Only tests import it, so neither the document nor the JSON Schema library
// ends up in the tba binary.
package apispec

//go:generate go run ./refresh

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"sort"
	"strings"
	"sync"

	"github.com/santhosh-tekuri/jsonschema/v6"
	"github.com/santhosh-tekuri/jsonschema/v6/kind"
	"golang.org/x/text/language"
	"golang.org/x/text/message"
)

// specJSON is api_v3.json from the server repository, unmodified. UPSTREAM
// records where it came from; `go generate ./internal/apispec` refreshes both.
//
//go:embed api_v3.json
var specJSON []byte

// upstream is the provenance record the refresh tool writes next to the spec.
//
//go:embed UPSTREAM
var upstream []byte

const docURL = "https://www.thebluealliance.com/swagger/api_v3.json"

// Spec is the parsed OpenAPI document plus a compiler for its schemas.
type Spec struct {
	doc       map[string]any
	templates []string

	mu       sync.Mutex
	compiler *jsonschema.Compiler
	compiled map[string]*jsonschema.Schema
}

var (
	loadOnce sync.Once
	loaded   *Spec
	loadErr  error
)

// Load parses the vendored document once per process.
func Load() (*Spec, error) {
	loadOnce.Do(func() { loaded, loadErr = parse(specJSON) })
	return loaded, loadErr
}

func parse(b []byte) (*Spec, error) {
	doc, err := jsonschema.UnmarshalJSON(bytes.NewReader(b))
	if err != nil {
		return nil, fmt.Errorf("parsing api_v3.json: %w", err)
	}
	m, ok := doc.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("api_v3.json is not an object")
	}
	paths, _ := m["paths"].(map[string]any)
	s := &Spec{doc: m, compiled: map[string]*jsonschema.Schema{}}
	for p := range paths {
		s.templates = append(s.templates, p)
	}
	sort.Strings(s.templates)

	c := jsonschema.NewCompiler()
	// OpenAPI 3.1 schema objects are draft 2020-12 unless they say otherwise.
	c.DefaultDraft(jsonschema.Draft2020)
	if err := c.AddResource(docURL, doc); err != nil {
		return nil, err
	}
	s.compiler = c
	return s, nil
}

// Version is the document's info.version.
func (s *Spec) Version() string {
	info, _ := s.doc["info"].(map[string]any)
	v, _ := info["version"].(string)
	return v
}

// Templates lists every path template the document defines, sorted.
func (s *Spec) Templates() []string { return append([]string(nil), s.templates...) }

// MatchPath finds the path template a concrete request path stands for, e.g.
// "/team/frc177/events/2024" is "/team/{team_key}/events/{year}". When several
// templates fit, the one with the most literal segments wins, which is how
// "/team/frc177/events/keys" is not taken for a year.
func (s *Spec) MatchPath(path string) (string, bool) {
	if i := strings.IndexAny(path, "?#"); i >= 0 {
		path = path[:i]
	}
	got := strings.Split(path, "/")
	best, bestLiterals := "", -1
	for _, tmpl := range s.templates {
		want := strings.Split(tmpl, "/")
		if len(want) != len(got) {
			continue
		}
		literals, ok := 0, true
		for i := range want {
			if isParam(want[i]) {
				if got[i] == "" {
					ok = false
					break
				}
				continue
			}
			if want[i] != got[i] {
				ok = false
				break
			}
			literals++
		}
		if ok && literals > bestLiterals {
			best, bestLiterals = tmpl, literals
		}
	}
	return best, bestLiterals >= 0
}

func isParam(seg string) bool {
	return strings.HasPrefix(seg, "{") && strings.HasSuffix(seg, "}")
}

// ResponsePointer is the JSON pointer of a path's GET 200 response schema.
func ResponsePointer(template string) string {
	return "/paths/" + escape(template) + "/get/responses/200/content/application~1json/schema"
}

// SchemaPointer is the JSON pointer of a named component schema.
func SchemaPointer(name string) string { return "/components/schemas/" + escape(name) }

func escape(tok string) string {
	return strings.ReplaceAll(strings.ReplaceAll(tok, "~", "~0"), "/", "~1")
}

// Has reports whether ptr names a node in the document.
func (s *Spec) Has(ptr string) bool {
	_, ok := s.at(ptr)
	return ok
}

// Validate checks a JSON body against the schema at ptr.
func (s *Spec) Validate(ptr string, body []byte) error {
	sch, err := s.compile(ptr)
	if err != nil {
		return err
	}
	inst, err := jsonschema.UnmarshalJSON(bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("body is not JSON: %w", err)
	}
	return sch.Validate(inst)
}

func (s *Spec) compile(ptr string) (*jsonschema.Schema, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if sch, ok := s.compiled[ptr]; ok {
		return sch, nil
	}
	if _, ok := s.at(ptr); !ok {
		return nil, fmt.Errorf("spec has no node at %s", ptr)
	}
	// Braces in a path template are not legal in a URL fragment.
	frag := strings.NewReplacer("{", url.PathEscape("{"), "}", url.PathEscape("}")).Replace(ptr)
	sch, err := s.compiler.Compile(docURL + "#" + frag)
	if err != nil {
		return nil, err
	}
	s.compiled[ptr] = sch
	return sch, nil
}

// at walks a JSON pointer through the raw document.
func (s *Spec) at(ptr string) (map[string]any, bool) {
	var cur any = s.doc
	if ptr != "" {
		for _, tok := range strings.Split(strings.TrimPrefix(ptr, "/"), "/") {
			tok = strings.ReplaceAll(strings.ReplaceAll(tok, "~1", "/"), "~0", "~")
			m, ok := cur.(map[string]any)
			if !ok {
				return nil, false
			}
			if cur, ok = m[tok]; !ok {
				return nil, false
			}
		}
	}
	m, ok := cur.(map[string]any)
	return m, ok
}

// Upstream is the provenance record of the vendored document.
func Upstream() string { return string(upstream) }

// RawSpec is the vendored document's bytes, for the provenance test.
func RawSpec() []byte { return specJSON }

// decode is json.Unmarshal into any, keeping numbers exact.
func decode(body []byte) (any, error) {
	d := json.NewDecoder(bytes.NewReader(body))
	d.UseNumber()
	var v any
	if err := d.Decode(&v); err != nil {
		return nil, err
	}
	return v, nil
}

// Problem is one way a body disagrees with the spec. Instance is a JSON
// pointer into the body.
type Problem struct {
	Instance string
	Message  string
}

func (p Problem) String() string { return p.Instance + ": " + p.Message }

// Check validates body against the schema at ptr and also lists the members
// the schema does not declare.
func (s *Spec) Check(ptr string, body []byte) ([]Problem, error) {
	var out []Problem
	err := s.Validate(ptr, body)
	var ve *jsonschema.ValidationError
	switch {
	case err == nil:
	case errors.As(err, &ve):
		out = append(out, leaves(ve)...)
	default:
		return nil, err
	}
	extra, err := s.UndeclaredFields(ptr, body)
	if err != nil {
		return nil, err
	}
	for _, at := range extra {
		out = append(out, Problem{Instance: at, Message: "member is not declared in the spec"})
	}
	return out, nil
}

func leaves(e *jsonschema.ValidationError) []Problem {
	if len(e.Causes) == 0 {
		return []Problem{{Instance: pointer(e.InstanceLocation), Message: e.ErrorKind.LocalizedString(printer)}}
	}
	causes := e.Causes
	switch k := e.ErrorKind.(type) {
	case *kind.OneOf, *kind.AnyOf:
		// When no branch matched, the {"type": "null"} branch of a nullable
		// value only adds "want null", which hides the real failure.
		if o, ok := k.(*kind.OneOf); !ok || o.Subschemas == nil {
			var kept []*jsonschema.ValidationError
			for _, c := range causes {
				if !wantsOnlyNull(c) {
					kept = append(kept, c)
				}
			}
			if len(kept) > 0 {
				causes = kept
			}
		}
	}
	var out []Problem
	for _, c := range causes {
		out = append(out, leaves(c)...)
	}
	return out
}

func wantsOnlyNull(e *jsonschema.ValidationError) bool {
	if t, ok := e.ErrorKind.(*kind.Type); ok {
		return len(t.Want) == 1 && t.Want[0] == "null"
	}
	return len(e.Causes) == 1 && wantsOnlyNull(e.Causes[0])
}

func pointer(loc []string) string {
	if len(loc) == 0 {
		return ""
	}
	var b strings.Builder
	for _, tok := range loc {
		b.WriteString("/")
		b.WriteString(escape(tok))
	}
	return b.String()
}

var printer = message.NewPrinter(language.English)
