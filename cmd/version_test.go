package cmd

import (
	"runtime"
	"strings"
	"testing"

	"github.com/the-blue-alliance/tba-cli/internal/version"
)

func TestVersionTable(t *testing.T) {
	out, _, err := runCmd(t, nil, "version", "--format", "table")
	requireNoError(t, err, "")

	want := version.Resolve().Line() + "\n"
	if out != want {
		t.Errorf("version =\n%q\nwant\n%q", out, want)
	}
	if !strings.HasPrefix(out, "tba "+version.Version) {
		t.Errorf("version line %q does not start with the binary name and version", out)
	}
	for _, part := range []string{runtime.Version(), runtime.GOOS + "/" + runtime.GOARCH} {
		requireContains(t, out, part)
	}
}

func TestVersionJSON(t *testing.T) {
	out, _, err := runCmd(t, nil, "version", "--format", "json")
	requireNoError(t, err, "")

	obj, ok := decodeJSON(t, out).(map[string]any)
	if !ok {
		t.Fatalf("want a JSON object, got %s", out)
	}
	info := version.Resolve()
	want := map[string]any{
		"version": info.Version,
		"commit":  info.Commit,
		"date":    info.Date,
		"go":      info.Go,
		"os":      info.OS,
		"arch":    info.Arch,
	}
	if len(obj) != len(want) {
		t.Errorf("version JSON has keys %v, want exactly %v", keysOf(obj), keysOf(want))
	}
	for k, v := range want {
		if obj[k] != v {
			t.Errorf("%s = %v, want %v", k, obj[k], v)
		}
	}
}

func keysOf(m map[string]any) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}

// The version is a dev build under `go test`, so this is the fallback path:
// runtime/debug fills the commit from the test binary's VCS stamp, or leaves it
// empty. Either way the line stays well-formed rather than printing "dev ()".
func TestVersionOfADevBuildIsStillWellFormed(t *testing.T) {
	if version.Version != "dev" {
		t.Skipf("this binary was stamped as %q", version.Version)
	}
	out, _, err := runCmd(t, nil, "version", "--format", "table")
	requireNoError(t, err, "")

	line := strings.TrimSpace(strings.SplitN(out, "\n", 2)[0])
	if version.Resolve().Commit == "" {
		// No VCS stamp available: the parenthesised part is toolchain only.
		if line != "tba dev ("+runtime.Version()+", "+runtime.GOOS+"/"+runtime.GOARCH+")" {
			t.Errorf("dev version line = %q", line)
		}
	}
	if strings.Contains(line, "()") || strings.Contains(line, ", ,") {
		t.Errorf("dev version line has an empty field: %q", line)
	}
}

func TestVersionJq(t *testing.T) {
	out, _, err := runCmd(t, nil, "version", "--jq", ".os", "-r")
	requireNoError(t, err, "")
	if strings.TrimSpace(out) != runtime.GOOS {
		t.Errorf("jq .os = %q, want %q", out, runtime.GOOS)
	}
}

// version is key-value data, so the tabular formats fall back to JSON.
func TestVersionCSVFallsBackToJSON(t *testing.T) {
	out, _, err := runCmd(t, nil, "version", "--format", "csv")
	requireNoError(t, err, "")
	decodeJSON(t, out)
}

func TestRootVersionFlagPrintsTheSameLine(t *testing.T) {
	flagOut, _, err := runCmd(t, nil, "--version")
	requireNoError(t, err, "")
	cmdOut, _, err := runCmd(t, nil, "version", "--format", "table")
	requireNoError(t, err, "")

	if flagOut != cmdOut {
		t.Errorf("--version printed\n%q\nbut `version` printed\n%q", flagOut, cmdOut)
	}
	// Cobra's default template would say "tba version 1.2.3"; ours must not.
	if strings.HasPrefix(flagOut, "tba version ") {
		t.Errorf("--version used cobra's default template: %q", flagOut)
	}
}

func TestVersionNeedsNoAPIKey(t *testing.T) {
	t.Setenv("TBA_AUTH_KEY", "")
	out, _, err := runCmd(t, nil, "version")
	requireNoError(t, err, "")
	requireContains(t, out, "\"version\"")
}

func TestVersionRejectsArguments(t *testing.T) {
	_, _, err := runCmd(t, nil, "version", "extra")
	requireErrorContains(t, err, "unknown command")
}

func TestVersionIsListedInRootHelp(t *testing.T) {
	out, _, err := runCmd(t, nil, "--help")
	requireNoError(t, err, "")
	requireContains(t, out, "version")
	if !contains(topLevelCommandNames(t), "version") {
		t.Error("version is not a top-level command")
	}
}

// topLevelCommandNames lists the commands registered directly on the root.
func topLevelCommandNames(t *testing.T) []string {
	t.Helper()
	cmds := NewRootCmd().Commands()
	names := make([]string, 0, len(cmds))
	for _, c := range cmds {
		names = append(names, c.Name())
	}
	return names
}
