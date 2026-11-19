package apispec

import (
	"encoding/json"
	"fmt"
	"reflect"
	"sort"
	"strings"
)

// shape is what a schema says about an object or array once $ref, allOf,
// oneOf and anyOf are flattened. Branches are unioned: a field counts as
// declared if any branch declares it, which is the right question for "does
// the spec know this field at all".
type shape struct {
	props map[string][]map[string]any
	addl  []map[string]any
	items []map[string]any
	types map[string]bool
}

// open is a schema that says nothing about its members, e.g. Event_Predictions.
func (sh shape) open() bool {
	return len(sh.props) == 0 && len(sh.addl) == 0 && len(sh.items) == 0
}

func (s *Spec) shapeOf(nodes []map[string]any) shape {
	sh := shape{props: map[string][]map[string]any{}, types: map[string]bool{}}
	seen := map[uintptr]bool{}
	var visit func(n map[string]any, depth int)
	visit = func(n map[string]any, depth int) {
		if n == nil || depth > 32 {
			return
		}
		key := reflect.ValueOf(n).Pointer()
		if seen[key] {
			return
		}
		seen[key] = true
		if ref, ok := n["$ref"].(string); ok && strings.HasPrefix(ref, "#") {
			target, _ := s.at(strings.TrimPrefix(ref, "#"))
			visit(target, depth+1)
		}
		for _, kw := range []string{"allOf", "oneOf", "anyOf"} {
			list, _ := n[kw].([]any)
			for _, b := range list {
				m, _ := b.(map[string]any)
				visit(m, depth+1)
			}
		}
		switch t := n["type"].(type) {
		case string:
			sh.types[t] = true
		case []any:
			for _, v := range t {
				if name, ok := v.(string); ok {
					sh.types[name] = true
				}
			}
		}
		if props, ok := n["properties"].(map[string]any); ok {
			for name, p := range props {
				m, _ := p.(map[string]any)
				sh.props[name] = append(sh.props[name], m)
			}
		}
		if a, ok := n["additionalProperties"].(map[string]any); ok {
			sh.addl = append(sh.addl, a)
		}
		if it, ok := n["items"].(map[string]any); ok {
			sh.items = append(sh.items, it)
		}
	}
	for _, n := range nodes {
		visit(n, 0)
	}
	return sh
}

// UndeclaredFields lists the members of body that the schema at ptr does not
// declare anywhere. JSON Schema lets an object carry extra members unless it
// says otherwise, which the API's schemas never do, so Validate alone would
// accept a fixture with a misspelled or invented field.
func (s *Spec) UndeclaredFields(ptr string, body []byte) ([]string, error) {
	root, ok := s.at(ptr)
	if !ok {
		return nil, fmt.Errorf("spec has no node at %s", ptr)
	}
	v, err := decode(body)
	if err != nil {
		return nil, err
	}
	var out []string
	s.walkValue("", v, []map[string]any{root}, &out)
	sort.Strings(out)
	return out, nil
}

func (s *Spec) walkValue(at string, v any, nodes []map[string]any, out *[]string) {
	sh := s.shapeOf(nodes)
	if sh.open() {
		return
	}
	switch val := v.(type) {
	case map[string]any:
		for name, child := range val {
			subs, ok := sh.props[name]
			if !ok {
				subs = sh.addl
			}
			if name == "score_breakdown" {
				subs = s.season(subs, val["key"])
			}
			if len(subs) == 0 {
				if len(sh.props) > 0 {
					*out = append(*out, at+"/"+escape(name))
				}
				continue
			}
			s.walkValue(at+"/"+escape(name), child, subs, out)
		}
	case []any:
		if len(sh.items) == 0 {
			return
		}
		for i, child := range val {
			s.walkValue(fmt.Sprintf("%s/%d", at, i), child, sh.items, out)
		}
	}
}

// season narrows a match's score_breakdown, a oneOf over every season's
// schema, to the season the match key names. Without it any season's fields
// would pass for any other's.
func (s *Spec) season(nodes []map[string]any, key any) []map[string]any {
	k, _ := key.(string)
	if len(k) < 4 {
		return nodes
	}
	suffix := "_" + k[:4]
	var out []map[string]any
	for _, n := range nodes {
		list, _ := n["oneOf"].([]any)
		for _, b := range list {
			m, _ := b.(map[string]any)
			if ref, ok := m["$ref"].(string); ok && strings.HasSuffix(ref, suffix) {
				target, _ := s.at(strings.TrimPrefix(ref, "#"))
				out = append(out, target)
			}
		}
	}
	if len(out) == 0 {
		return nodes
	}
	return out
}

var jsonUnmarshaler = reflect.TypeOf((*json.Unmarshaler)(nil)).Elem()

// UndeclaredStructFields lists the json tags of t, and of every struct it
// holds, that the schema at ptr does not declare. Nothing below a schema that
// leaves its members untyped is checked, and neither is a type with its own
// UnmarshalJSON.
func (s *Spec) UndeclaredStructFields(ptr string, t reflect.Type) ([]string, error) {
	root, ok := s.at(ptr)
	if !ok {
		return nil, fmt.Errorf("spec has no node at %s", ptr)
	}
	var out []string
	s.walkType(t.Name(), t, []map[string]any{root}, &out, map[reflect.Type]int{})
	sort.Strings(out)
	return out, nil
}

// IsOpen reports whether the schema at ptr leaves its members untyped.
func (s *Spec) IsOpen(ptr string) bool {
	n, ok := s.at(ptr)
	return ok && s.shapeOf([]map[string]any{n}).open()
}

func (s *Spec) walkType(at string, t reflect.Type, nodes []map[string]any, out *[]string, depth map[reflect.Type]int) {
	for t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	if t.Kind() == reflect.Interface || reflect.PointerTo(t).Implements(jsonUnmarshaler) {
		return
	}
	if depth[t] > 2 {
		return
	}
	depth[t]++
	defer func() { depth[t]-- }()

	sh := s.shapeOf(nodes)
	if msg := kindMismatch(t, sh.types); msg != "" {
		*out = append(*out, at+": "+msg)
		return
	}
	if sh.open() {
		return
	}
	switch t.Kind() {
	case reflect.Struct:
		for i := 0; i < t.NumField(); i++ {
			f := t.Field(i)
			if !f.IsExported() {
				continue
			}
			name, _, _ := strings.Cut(f.Tag.Get("json"), ",")
			if name == "-" {
				continue
			}
			if name == "" {
				name = f.Name
			}
			subs, ok := sh.props[name]
			if !ok {
				subs = sh.addl
			}
			if len(subs) == 0 {
				*out = append(*out, at+"."+name)
				continue
			}
			s.walkType(at+"."+name, f.Type, subs, out, depth)
		}
	case reflect.Slice, reflect.Array:
		if len(sh.items) > 0 {
			s.walkType(at+"[]", t.Elem(), sh.items, out, depth)
		}
	case reflect.Map:
		subs := sh.addl
		if len(subs) == 0 {
			// A fixed set of keys read into a map, like a match's red and blue.
			for _, p := range sh.props {
				subs = append(subs, p...)
			}
		}
		if len(subs) > 0 {
			s.walkType(at+"[*]", t.Elem(), subs, out, depth)
		}
	}
}

// kindMismatch says why a Go value of type t cannot hold what the schema's
// types allow, or "" when it can. A schema with no type says nothing.
func kindMismatch(t reflect.Type, types map[string]bool) string {
	if len(types) == 0 {
		return ""
	}
	var ok bool
	switch t.Kind() {
	case reflect.String:
		ok = types["string"]
	case reflect.Bool:
		ok = types["boolean"]
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		// An integer field cannot take a fractional number.
		ok = types["integer"] && !types["number"]
	case reflect.Float32, reflect.Float64:
		ok = types["number"] || types["integer"]
	case reflect.Slice, reflect.Array:
		ok = types["array"]
	case reflect.Map, reflect.Struct:
		ok = types["object"]
	default:
		return ""
	}
	if ok {
		return ""
	}
	want := make([]string, 0, len(types))
	for k := range types {
		want = append(want, k)
	}
	sort.Strings(want)
	return fmt.Sprintf("Go %s cannot hold JSON %s", t.Kind(), strings.Join(want, "|"))
}
