package config

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"go.yaml.in/yaml/v3"

	"github.com/the-blue-alliance/tba-cli/internal/clierr"
)

// settingsFileName is the layered-settings file, kept beside auth.yaml.
const settingsFileName = "config.yaml"

// A Kind says what a setting's value has to parse as. It exists so that
// `tba config set retries abc` is refused before any command reads it.
type Kind int

const (
	// KindString is free text.
	KindString Kind = iota
	// KindBool is anything strconv.ParseBool accepts.
	KindBool
	// KindInt is a plain integer.
	KindInt
	// KindDuration is a Go duration such as 10s or 1m.
	KindDuration
	// KindEnum is one of a fixed set of words.
	KindEnum
)

// A Key is one setting config.yaml understands. The names match the persistent
// flags exactly, so that what you type on the command line is what you write
// in the file.
type Key struct {
	Name string
	Kind Kind
	// Enum lists the accepted words when Kind is KindEnum.
	Enum []string
	Help string
}

// Keys are the settings config.yaml may contain, in the order `config list`
// prints them.
var Keys = []Key{
	{Name: "base-url", Kind: KindString, Help: "API base URL"},
	{Name: "color", Kind: KindEnum, Enum: []string{"auto", "always", "never"}, Help: "When to colorize output"},
	{Name: "format", Kind: KindEnum, Enum: []string{"auto", "table", "json", "csv", "tsv", "markdown"}, Help: "Output format (a value from the file applies only on a terminal)"},
	{Name: "no-cache", Kind: KindBool, Help: "Disable the on-disk response cache"},
	{Name: "no-color", Kind: KindBool, Help: "Disable colored output"},
	{Name: "retries", Kind: KindInt, Help: "Retry attempts for 429/5xx/network errors"},
	{Name: "timeout", Kind: KindDuration, Help: "Per-request timeout"},
	{Name: "year", Kind: KindInt, Help: "Default season year"},
}

// firstSeason is the earliest season The Blue Alliance holds data for; a year
// below it is a typo rather than a request.
const firstSeason = 1992

// KeyNames returns every known key, sorted.
func KeyNames() []string {
	names := make([]string, len(Keys))
	for i, k := range Keys {
		names[i] = k.Name
	}
	sort.Strings(names)
	return names
}

// LookupKey finds a key by name.
func LookupKey(name string) (Key, bool) {
	for _, k := range Keys {
		if k.Name == name {
			return k, true
		}
	}
	return Key{}, false
}

// SuggestKey returns the known key a misspelling most likely meant, if one is
// close enough to be worth naming.
func SuggestKey(name string) (string, bool) {
	name = strings.ToLower(name)
	best, bestDist := "", 0
	for _, k := range Keys {
		d := editDistance(name, k.Name)
		// Roughly one mistake per three characters, so "fromat" finds
		// "format" but "nonsense" finds nothing.
		if d > len(k.Name)/3+1 {
			continue
		}
		if best == "" || d < bestDist {
			best, bestDist = k.Name, d
		}
	}
	return best, best != ""
}

// UnknownKeyError describes a key that is not one of ours, naming the closest
// match when there is one.
func UnknownKeyError(name string) error {
	if suggestion, ok := SuggestKey(name); ok {
		return clierr.Usage("unknown config key %q; did you mean %q?", name, suggestion)
	}
	return clierr.Usage("unknown config key %q (known keys: %s)", name, strings.Join(KeyNames(), ", "))
}

// editDistance is the Levenshtein distance between a and b.
func editDistance(a, b string) int {
	prev := make([]int, len(b)+1)
	cur := make([]int, len(b)+1)
	for j := range prev {
		prev[j] = j
	}
	for i := 1; i <= len(a); i++ {
		cur[0] = i
		for j := 1; j <= len(b); j++ {
			cost := 1
			if a[i-1] == b[j-1] {
				cost = 0
			}
			cur[j] = min(prev[j]+1, min(cur[j-1]+1, prev[j-1]+cost))
		}
		prev, cur = cur, prev
	}
	return prev[len(b)]
}

// ParseValue turns the text of `tba config set <key> <value>` into the typed
// value written to the file.
func (k Key) ParseValue(raw string) (any, error) {
	switch k.Kind {
	case KindBool:
		b, err := strconv.ParseBool(raw)
		if err != nil {
			return nil, clierr.Usage("%s wants a boolean (true or false), not %q", k.Name, raw)
		}
		return b, nil
	case KindInt:
		n, err := strconv.Atoi(raw)
		if err != nil {
			return nil, clierr.Usage("%s wants a whole number, not %q", k.Name, raw)
		}
		if k.Name == "retries" && n < 0 {
			return nil, clierr.Usage("retries cannot be negative")
		}
		if k.Name == "year" && n < firstSeason {
			return nil, clierr.Usage("year wants a season from %d onwards, not %q", firstSeason, raw)
		}
		return n, nil
	case KindDuration:
		d, err := time.ParseDuration(raw)
		if err != nil {
			return nil, clierr.Usage("%s wants a duration such as 10s or 1m, not %q", k.Name, raw)
		}
		if d <= 0 {
			return nil, clierr.Usage("%s must be positive", k.Name)
		}
		return d.String(), nil
	case KindEnum:
		for _, v := range k.Enum {
			if raw == v {
				return raw, nil
			}
		}
		return nil, clierr.Usage("invalid %s %q (want: %s)", k.Name, raw, strings.Join(k.Enum, ", "))
	default:
		if k.Name == "base-url" {
			u, err := url.Parse(raw)
			if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
				return nil, clierr.Usage("base-url wants an http(s) URL, not %q", raw)
			}
		}
		return raw, nil
	}
}

// SettingsFile returns the path of the layered settings file.
func SettingsFile() (string, error) {
	dir, err := configDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, settingsFileName), nil
}

// Settings is what config.yaml said: the values of the keys we know, plus the
// names of the ones we do not, which callers report without failing.
type Settings struct {
	Path    string
	Values  map[string]any
	Unknown []string
}

// LoadSettings reads config.yaml. A missing file is not an error: it means
// every setting is at its default. A malformed one is, because carrying on
// would hide settings the user believes are in effect.
func LoadSettings() (*Settings, error) {
	path, err := SettingsFile()
	if err != nil {
		return nil, err
	}
	s := &Settings{Path: path, Values: map[string]any{}}

	b, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return s, nil
		}
		return nil, fmt.Errorf("reading %s: %w", path, err)
	}
	var raw map[string]any
	if err := yaml.Unmarshal(b, &raw); err != nil {
		return nil, fmt.Errorf("reading %s: %w", path, err)
	}
	for name, v := range raw {
		lower := strings.ToLower(name)
		if _, ok := LookupKey(lower); ok {
			s.Values[lower] = v
			continue
		}
		s.Unknown = append(s.Unknown, name)
	}
	sort.Strings(s.Unknown)
	return s, nil
}

// SetSetting writes one key to config.yaml, leaving the rest alone.
func SetSetting(name string, value any) error {
	return updateSettings(func(m map[string]any) { m[name] = value })
}

// UnsetSetting removes one key from config.yaml. Removing a key that is not
// there is not an error: what was asked for is already the case.
func UnsetSetting(name string) error {
	return updateSettings(func(m map[string]any) { delete(m, name) })
}

// updateSettings applies mutate to the file's contents and writes the result
// back atomically, so an interrupted write cannot leave a half-parsed file.
// Unknown keys are preserved: they may belong to a newer tba.
func updateSettings(mutate func(map[string]any)) error {
	path, err := SettingsFile()
	if err != nil {
		return err
	}
	values := map[string]any{}
	if b, err := os.ReadFile(path); err == nil {
		if err := yaml.Unmarshal(b, &values); err != nil {
			return fmt.Errorf("reading %s: %w", path, err)
		}
		if values == nil {
			values = map[string]any{}
		}
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("reading %s: %w", path, err)
	}

	mutate(values)

	out, err := yaml.Marshal(values)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return err
	}
	return writeFileAtomic(path, out)
}

// writeFileAtomic replaces path through a temp file and a rename, so a reader
// sees either the old file or the new one and never a partial write.
func writeFileAtomic(path string, data []byte) error {
	tmp, err := os.CreateTemp(filepath.Dir(path), ".tba-config-*.tmp")
	if err != nil {
		return err
	}
	name := tmp.Name()
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		os.Remove(name)
		return err
	}
	if err := tmp.Close(); err != nil {
		os.Remove(name)
		return err
	}
	// CreateTemp already makes the file 0600, but say so explicitly: the file
	// sits next to auth.yaml and is nobody else's business.
	if err := os.Chmod(name, 0600); err != nil {
		os.Remove(name)
		return err
	}
	if err := os.Rename(name, path); err != nil {
		os.Remove(name)
		return err
	}
	return nil
}
