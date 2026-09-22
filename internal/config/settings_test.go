package config

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// settingsEnv points the package at an empty config directory.
func settingsEnv(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("TBA_CONFIG_DIR", dir)
	return dir
}

func writeSettingsFile(t *testing.T, body string) {
	t.Helper()
	path, err := SettingsFile()
	if err != nil {
		t.Fatalf("SettingsFile: %v", err)
	}
	if err := os.WriteFile(path, []byte(body), 0600); err != nil {
		t.Fatalf("writing config file: %v", err)
	}
}

func TestSettingsFileSitsBesideAuthFile(t *testing.T) {
	dir := settingsEnv(t)
	path, err := SettingsFile()
	if err != nil {
		t.Fatalf("SettingsFile: %v", err)
	}
	if want := filepath.Join(dir, "config.yaml"); path != want {
		t.Errorf("SettingsFile() = %q, want %q", path, want)
	}
}

func TestSettingsFileFollowsXDGConfigHome(t *testing.T) {
	xdg := t.TempDir()
	t.Setenv("TBA_CONFIG_DIR", "")
	t.Setenv("XDG_CONFIG_HOME", xdg)

	path, err := SettingsFile()
	if err != nil {
		t.Fatalf("SettingsFile: %v", err)
	}
	if want := filepath.Join(xdg, "tba", "config.yaml"); path != want {
		t.Errorf("SettingsFile() = %q, want %q", path, want)
	}
}

func TestLoadSettingsWithNoFileIsEmpty(t *testing.T) {
	settingsEnv(t)
	s, err := LoadSettings()
	if err != nil {
		t.Fatalf("LoadSettings: %v", err)
	}
	if len(s.Values) != 0 || len(s.Unknown) != 0 {
		t.Errorf("want an empty result, got %+v", s)
	}
}

func TestLoadSettingsReadsKnownKeys(t *testing.T) {
	settingsEnv(t)
	writeSettingsFile(t, "format: table\nretries: 5\nno-color: true\ntimeout: 30s\n")

	s, err := LoadSettings()
	if err != nil {
		t.Fatalf("LoadSettings: %v", err)
	}
	if s.Values["format"] != "table" {
		t.Errorf("format = %v", s.Values["format"])
	}
	if s.Values["retries"] != 5 {
		t.Errorf("retries = %v (%T)", s.Values["retries"], s.Values["retries"])
	}
	if s.Values["no-color"] != true {
		t.Errorf("no-color = %v", s.Values["no-color"])
	}
	if s.Values["timeout"] != "30s" {
		t.Errorf("timeout = %v", s.Values["timeout"])
	}
}

func TestLoadSettingsCollectsUnknownKeysWithoutFailing(t *testing.T) {
	settingsEnv(t)
	writeSettingsFile(t, "format: table\nwidgets: 3\nfromat: json\n")

	s, err := LoadSettings()
	if err != nil {
		t.Fatalf("an unknown key must not be an error: %v", err)
	}
	if got := strings.Join(s.Unknown, ","); got != "fromat,widgets" {
		t.Errorf("Unknown = %v, want [fromat widgets]", s.Unknown)
	}
	if s.Values["format"] != "table" {
		t.Error("the known keys should still be read")
	}
}

func TestLoadSettingsRejectsAMalformedFile(t *testing.T) {
	settingsEnv(t)
	writeSettingsFile(t, "format: [unterminated\n")

	if _, err := LoadSettings(); err == nil {
		t.Fatal("want an error for a file that cannot be parsed")
	}
}

func TestSetSettingWritesTheValue(t *testing.T) {
	settingsEnv(t)
	if err := SetSetting("format", "table"); err != nil {
		t.Fatalf("SetSetting: %v", err)
	}

	s, err := LoadSettings()
	if err != nil {
		t.Fatalf("LoadSettings: %v", err)
	}
	if s.Values["format"] != "table" {
		t.Errorf("format = %v", s.Values["format"])
	}
}

func TestSetSettingKeepsTheOtherKeys(t *testing.T) {
	settingsEnv(t)
	writeSettingsFile(t, "retries: 5\nwidgets: keep-me\n")

	if err := SetSetting("format", "json"); err != nil {
		t.Fatalf("SetSetting: %v", err)
	}
	path, _ := SettingsFile()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	for _, want := range []string{"retries: 5", "widgets: keep-me", "format: json"} {
		if !strings.Contains(string(b), want) {
			t.Errorf("config file lost %q:\n%s", want, b)
		}
	}
}

func TestSetSettingCreatesTheDirectory(t *testing.T) {
	base := t.TempDir()
	t.Setenv("TBA_CONFIG_DIR", filepath.Join(base, "nested", "tba"))

	if err := SetSetting("retries", 1); err != nil {
		t.Fatalf("SetSetting: %v", err)
	}
	path, _ := SettingsFile()
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("config file was not created: %v", err)
	}
}

func TestSetSettingWritesAPrivateFile(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("file modes are not meaningful on Windows")
	}
	settingsEnv(t)
	if err := SetSetting("timeout", "30s"); err != nil {
		t.Fatalf("SetSetting: %v", err)
	}
	path, _ := SettingsFile()
	fi, err := os.Stat(path)
	if err != nil {
		t.Fatalf("Stat: %v", err)
	}
	if fi.Mode().Perm() != 0600 {
		t.Errorf("mode = %v, want 0600", fi.Mode().Perm())
	}
}

func TestSetSettingLeavesNoTempFiles(t *testing.T) {
	dir := settingsEnv(t)
	if err := SetSetting("retries", 2); err != nil {
		t.Fatalf("SetSetting: %v", err)
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("ReadDir: %v", err)
	}
	if len(entries) != 1 || entries[0].Name() != "config.yaml" {
		names := make([]string, len(entries))
		for i, e := range entries {
			names[i] = e.Name()
		}
		t.Errorf("config dir holds %v, want just config.yaml", names)
	}
}

func TestUnsetSettingRemovesOnlyThatKey(t *testing.T) {
	settingsEnv(t)
	writeSettingsFile(t, "format: table\nretries: 5\n")

	if err := UnsetSetting("format"); err != nil {
		t.Fatalf("UnsetSetting: %v", err)
	}
	s, err := LoadSettings()
	if err != nil {
		t.Fatalf("LoadSettings: %v", err)
	}
	if _, ok := s.Values["format"]; ok {
		t.Error("format survived the unset")
	}
	if s.Values["retries"] != 5 {
		t.Errorf("retries = %v, want 5", s.Values["retries"])
	}
}

func TestUnsetSettingOfAnAbsentKeyIsFine(t *testing.T) {
	settingsEnv(t)
	if err := UnsetSetting("format"); err != nil {
		t.Errorf("unsetting an unset key should be a no-op, got %v", err)
	}
}

func TestLookupKeyKnowsEveryFlagNamedKey(t *testing.T) {
	want := []string{"base-url", "color", "format", "no-cache", "no-color", "retries", "timeout", "year"}
	if got := strings.Join(KeyNames(), ","); got != strings.Join(want, ",") {
		t.Errorf("KeyNames() = %v, want %v", KeyNames(), want)
	}
	for _, name := range want {
		if _, ok := LookupKey(name); !ok {
			t.Errorf("LookupKey(%q) failed", name)
		}
	}
	if _, ok := LookupKey("nope"); ok {
		t.Error("LookupKey invented a key")
	}
}

func TestSuggestKeyFindsTypos(t *testing.T) {
	cases := map[string]string{
		"fromat":   "format",
		"forma":    "format",
		"retires":  "retries",
		"baseurl":  "base-url",
		"nocolor":  "no-color",
		"no_cache": "no-cache",
		"yearz":    "year",
	}
	for typo, want := range cases {
		got, ok := SuggestKey(typo)
		if !ok || got != want {
			t.Errorf("SuggestKey(%q) = %q, %v; want %q", typo, got, ok, want)
		}
	}
}

func TestSuggestKeySaysNothingAboutNonsense(t *testing.T) {
	for _, s := range []string{"hunter2", "completely-different"} {
		if got, ok := SuggestKey(s); ok {
			t.Errorf("SuggestKey(%q) = %q, want no suggestion", s, got)
		}
	}
}

func TestUnknownKeyErrorSuggestsAndLists(t *testing.T) {
	err := UnknownKeyError("fromat")
	if !strings.Contains(err.Error(), `did you mean "format"`) {
		t.Errorf("error should suggest the near miss: %v", err)
	}
	err = UnknownKeyError("hunter2")
	if !strings.Contains(err.Error(), "known keys:") {
		t.Errorf("error should list the keys when there is no near miss: %v", err)
	}
}

func TestParseValueAcceptsGoodValues(t *testing.T) {
	cases := []struct {
		key, raw string
		want     any
	}{
		{"format", "table", "table"},
		{"format", "auto", "auto"},
		{"color", "never", "never"},
		{"no-color", "true", true},
		{"no-color", "0", false},
		{"no-cache", "TRUE", true},
		{"retries", "0", 0},
		{"retries", "5", 5},
		{"timeout", "1m", "1m0s"},
		{"timeout", "500ms", "500ms"},
		{"year", "2024", 2024},
		{"base-url", "http://localhost:8080/api/v3", "http://localhost:8080/api/v3"},
	}
	for _, c := range cases {
		key, ok := LookupKey(c.key)
		if !ok {
			t.Fatalf("no key %q", c.key)
		}
		got, err := key.ParseValue(c.raw)
		if err != nil {
			t.Errorf("%s=%s: %v", c.key, c.raw, err)
			continue
		}
		if got != c.want {
			t.Errorf("%s=%s parsed to %v (%T), want %v", c.key, c.raw, got, got, c.want)
		}
	}
}

func TestParseValueRejectsBadValues(t *testing.T) {
	cases := []struct{ key, raw, mention string }{
		{"format", "xml", "want:"},
		{"color", "sometimes", "want:"},
		{"no-color", "yes-please", "boolean"},
		{"retries", "many", "whole number"},
		{"retries", "-1", "negative"},
		{"timeout", "soon", "duration"},
		{"timeout", "0s", "positive"},
		{"year", "24", "1992"},
		{"year", "twenty", "whole number"},
		{"base-url", "not a url", "http(s) URL"},
		{"base-url", "ftp://example.test", "http(s) URL"},
	}
	for _, c := range cases {
		key, ok := LookupKey(c.key)
		if !ok {
			t.Fatalf("no key %q", c.key)
		}
		_, err := key.ParseValue(c.raw)
		if err == nil {
			t.Errorf("%s=%s was accepted", c.key, c.raw)
			continue
		}
		if !strings.Contains(err.Error(), c.mention) {
			t.Errorf("%s=%s: error %q does not mention %q", c.key, c.raw, err, c.mention)
		}
	}
}
