package cmd

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/the-blue-alliance/tba-cli/internal/clierr"
	"github.com/the-blue-alliance/tba-cli/internal/config"
)

func TestConfigList(t *testing.T) {
	writeConfig(t, "retries: 5\n")
	t.Setenv("TBA_TIMEOUT", "45s")

	out, _, err := runCmd(t, nil, "config", "list", "--format", "table")
	requireNoError(t, err, "")

	for _, key := range config.KeyNames() {
		requireContains(t, out, key)
	}
	requireContains(t, out, "45s")
	requireContains(t, out, "current season")
}

func TestConfigListReportsSources(t *testing.T) {
	writeConfig(t, "retries: 5\n")
	t.Setenv("TBA_TIMEOUT", "45s")

	out, _, err := runCmd(t, nil, "config", "list", "--json", "--color", "never")
	requireNoError(t, err, "")

	var entries []configEntry
	if err := json.Unmarshal([]byte(out), &entries); err != nil {
		t.Fatalf("config list --json is not a JSON array: %v\n%s", err, out)
	}
	byKey := map[string]configEntry{}
	for _, e := range entries {
		byKey[e.Key] = e
	}
	if len(byKey) != len(config.Keys) {
		t.Fatalf("listed %d keys, want %d", len(byKey), len(config.Keys))
	}
	cases := []struct{ key, value, source string }{
		{"retries", "5", "config"},
		{"timeout", "45s", "env"},
		{"color", "never", "flag"},
		{"no-cache", "false", "default"},
	}
	for _, c := range cases {
		got := byKey[c.key]
		if got.Value != c.value || got.Source != c.source {
			t.Errorf("%s = %+v, want value %q from %q", c.key, got, c.value, c.source)
		}
	}
}

func TestConfigGet(t *testing.T) {
	writeConfig(t, "format: table\nretries: 5\n")

	out, _, err := runCmd(t, nil, "config", "get", "retries")
	requireNoError(t, err, "")
	if strings.TrimSpace(out) != "5" {
		t.Errorf("config get retries = %q, want a bare 5", out)
	}
}

func TestConfigGetPrintsTheEffectiveValueNotTheFileValue(t *testing.T) {
	// format from the file does not apply to a pipe, and `config list` and
	// `config get` report what is actually in effect.
	writeConfig(t, "format: table\n")

	out, _, err := runCmd(t, nil, "config", "get", "format")
	requireNoError(t, err, "")
	if strings.TrimSpace(out) != "auto" {
		t.Errorf("config get format = %q, want auto off a terminal", out)
	}
}

func TestConfigGetJSON(t *testing.T) {
	writeConfig(t, "retries: 5\n")

	out, _, err := runCmd(t, nil, "config", "get", "retries", "--json")
	requireNoError(t, err, "")
	got := decodeJSON(t, out).(map[string]any)
	if got["key"] != "retries" || got["value"] != "5" || got["source"] != "config" {
		t.Errorf("config get --json = %v", got)
	}
}

func TestConfigGetUnknownKeySuggests(t *testing.T) {
	emptyConfigDir(t)
	_, _, err := runCmd(t, nil, "config", "get", "fromat")
	requireErrorContains(t, err, `did you mean "format"`)
	if got := clierr.ExitCode(err); got != clierr.ExitUsage {
		t.Errorf("exit code = %d, want %d", got, clierr.ExitUsage)
	}
}

func TestConfigSetWritesTheFile(t *testing.T) {
	emptyConfigDir(t)

	stdout, stderr, err := runCmd(t, nil, "config", "set", "format", "table")
	requireNoError(t, err, stderr)
	if stdout != "" {
		t.Errorf("config set wrote %q to stdout, which is for data only", stdout)
	}
	requireContains(t, stderr, "format")
	requireContains(t, readConfig(t), "format: table")
}

func TestConfigSetNormalisesTypes(t *testing.T) {
	emptyConfigDir(t)
	cases := [][2]string{
		{"timeout", "1m"},
		{"retries", "0"},
		{"no-color", "true"},
		{"year", "2024"},
	}
	for _, c := range cases {
		if _, _, err := runCmd(t, nil, "config", "set", c[0], c[1]); err != nil {
			t.Fatalf("config set %s %s: %v", c[0], c[1], err)
		}
	}
	body := readConfig(t)
	for _, want := range []string{"timeout: 1m0s", "retries: 0", "no-color: true", "year: 2024"} {
		if !strings.Contains(body, want) {
			t.Errorf("config file missing %q:\n%s", want, body)
		}
	}
}

func TestConfigSetIsPrivate(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("file modes are not meaningful on Windows")
	}
	dir := emptyConfigDir(t)
	if _, _, err := runCmd(t, nil, "config", "set", "retries", "1"); err != nil {
		t.Fatalf("config set: %v", err)
	}
	fi, err := os.Stat(filepath.Join(dir, "config.yaml"))
	if err != nil {
		t.Fatalf("Stat: %v", err)
	}
	if fi.Mode().Perm() != 0600 {
		t.Errorf("config file mode = %v, want 0600", fi.Mode().Perm())
	}
}

func TestConfigSetRejectsUnknownKeys(t *testing.T) {
	emptyConfigDir(t)
	_, _, err := runCmd(t, nil, "config", "set", "fromat", "table")
	requireErrorContains(t, err, `did you mean "format"`)
	if got := clierr.ExitCode(err); got != clierr.ExitUsage {
		t.Errorf("exit code = %d, want %d", got, clierr.ExitUsage)
	}
}

func TestConfigSetRejectsUnknownKeysWithNoNearMiss(t *testing.T) {
	emptyConfigDir(t)
	_, _, err := runCmd(t, nil, "config", "set", "hunter2", "table")
	requireErrorContains(t, err, "known keys:")
}

func TestConfigSetValidatesValues(t *testing.T) {
	cases := [][2]string{
		{"format", "xml"},
		{"color", "sometimes"},
		{"timeout", "soon"},
		{"retries", "many"},
		{"year", "twenty"},
		{"no-color", "maybe"},
		{"base-url", "not a url"},
	}
	for _, c := range cases {
		t.Run(c[0], func(t *testing.T) {
			dir := emptyConfigDir(t)
			_, _, err := runCmd(t, nil, "config", "set", c[0], c[1])
			if err == nil {
				t.Fatalf("config set %s %s was accepted", c[0], c[1])
			}
			if got := clierr.ExitCode(err); got != clierr.ExitUsage {
				t.Errorf("exit code = %d, want %d", got, clierr.ExitUsage)
			}
			if _, err := os.Stat(filepath.Join(dir, "config.yaml")); err == nil {
				t.Error("a rejected value still wrote the config file")
			}
		})
	}
}

func TestConfigUnset(t *testing.T) {
	writeConfig(t, "format: table\nretries: 5\n")

	_, stderr, err := runCmd(t, nil, "config", "unset", "format")
	requireNoError(t, err, stderr)

	body := readConfig(t)
	if strings.Contains(body, "format") {
		t.Errorf("format survived the unset:\n%s", body)
	}
	if !strings.Contains(body, "retries: 5") {
		t.Errorf("unset took the other keys with it:\n%s", body)
	}
}

func TestConfigUnsetRejectsUnknownKeys(t *testing.T) {
	emptyConfigDir(t)
	_, _, err := runCmd(t, nil, "config", "unset", "fromat")
	requireErrorContains(t, err, `did you mean "format"`)
}

func TestConfigPathPrintsTheBarePath(t *testing.T) {
	dir := emptyConfigDir(t)

	out, stderr, err := runCmd(t, nil, "config", "path")
	requireNoError(t, err, stderr)
	if got := strings.TrimSpace(out); got != filepath.Join(dir, "config.yaml") {
		t.Errorf("config path = %q, want the bare path", out)
	}
	if lines := lines(out); len(lines) != 1 {
		t.Errorf("config path printed %d lines, want 1:\n%s", len(lines), out)
	}
}

func TestConfigPathJSON(t *testing.T) {
	dir := emptyConfigDir(t)
	out, _, err := runCmd(t, nil, "config", "path", "--json")
	requireNoError(t, err, "")
	got := decodeJSON(t, out).(map[string]any)
	if got["path"] != filepath.Join(dir, "config.yaml") {
		t.Errorf("config path --json = %v", got)
	}
}

func TestConfigSubcommands(t *testing.T) {
	want := []string{"get", "list", "path", "set", "unset"}
	got := subcommandNames(t, "config")
	for _, name := range want {
		if !contains(got, name) {
			t.Errorf("config is missing the %q subcommand (has %v)", name, got)
		}
	}
}
