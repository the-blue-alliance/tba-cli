package cmd

import (
	"strings"
	"testing"
)

func TestRootHelpListsAllTopLevelCommands(t *testing.T) {
	out, _, err := runCmd(t, nil, "--help")
	requireNoError(t, err, "")

	for _, want := range []string{
		"A command-line interface for The Blue Alliance API v3.",
		"auth", "cache", "completion", "district", "event",
		"insight", "match", "status", "team",
		"--base-url string", "--format string", "--jq string", "--json", "--no-cache",
	} {
		requireContains(t, out, want)
	}
}

func TestNewRootCmdReturnsIndependentTrees(t *testing.T) {
	a := NewRootCmd()
	b := NewRootCmd()
	if a == b {
		t.Fatal("NewRootCmd returned the same command twice")
	}
	if err := a.PersistentFlags().Set("format", "csv"); err != nil {
		t.Fatalf("set format: %v", err)
	}
	if got, _ := b.PersistentFlags().GetString("format"); got != "" {
		t.Errorf("flag state leaked between trees: got %q", got)
	}
}

func TestSendsAuthAndUserAgentHeaders(t *testing.T) {
	srv := newFakeTBA(t, map[string]any{"/status": apiStatusJSON})
	_, _, err := runCmd(t, srv, "status")
	requireNoError(t, err, "")

	reqs := requestsTo(t, srv)
	if len(reqs) != 1 {
		t.Fatalf("want 1 request, got %d", len(reqs))
	}
	if got := reqs[0].Headers.Get("X-TBA-Auth-Key"); got != "test-key" {
		t.Errorf("X-TBA-Auth-Key = %q, want test-key", got)
	}
	if got := reqs[0].Headers.Get("User-Agent"); got != "tba-cli" {
		t.Errorf("User-Agent = %q, want tba-cli", got)
	}
	if reqs[0].Method != "GET" {
		t.Errorf("method = %q, want GET", reqs[0].Method)
	}
}

func TestDefaultFormatIsJSONWhenNotATTY(t *testing.T) {
	srv := newFakeTBA(t, map[string]any{"/team/frc177": teamFRC177JSON})
	out, _, err := runCmd(t, srv, "team", "view", "177")
	requireNoError(t, err, "")

	obj, ok := decodeJSON(t, out).(map[string]any)
	if !ok {
		t.Fatalf("want a JSON object, got %s", out)
	}
	if obj["nickname"] != "Bobcat Robotics" {
		t.Errorf("nickname = %v", obj["nickname"])
	}
}

func TestFormatFlagBeatsJSONFlag(t *testing.T) {
	srv := newFakeTBA(t, map[string]any{"/districts/2024": districts2024JSON})
	out, _, err := runCmd(t, srv, "district", "list", "--year", "2024", "--json", "--format", "csv")
	requireNoError(t, err, "")

	if !strings.HasPrefix(out, "Key,Name,Abbreviation\n") {
		t.Errorf("--format should win over --json, got:\n%s", out)
	}
}

func TestFormatFlagBeatsJqFlag(t *testing.T) {
	srv := newFakeTBA(t, map[string]any{"/districts/2024": districts2024JSON})
	out, _, err := runCmd(t, srv, "district", "list", "--year", "2024", "--jq", ".[].key", "--format", "table")
	requireNoError(t, err, "")

	if !strings.HasPrefix(out, "Key ") {
		t.Errorf("--format should win over --jq, got:\n%s", out)
	}
}

func TestJqFlagImpliesJSONFormat(t *testing.T) {
	srv := newFakeTBA(t, map[string]any{"/districts/2024": districts2024JSON})
	out, _, err := runCmd(t, srv, "district", "list", "--year", "2024", "--jq", ".[].key")
	requireNoError(t, err, "")

	want := "\"2024ne\"\n\"2024fim\"\n"
	if out != want {
		t.Errorf("jq output = %q, want %q", out, want)
	}
}

func TestInvalidFormatIsAnError(t *testing.T) {
	srv := newFakeTBA(t, map[string]any{"/status": apiStatusJSON})
	_, _, err := runCmd(t, srv, "status", "--format", "xml")
	if err == nil {
		t.Fatal("want an error for --format xml")
	}
	if !strings.Contains(err.Error(), `invalid --format "xml"`) {
		t.Errorf("error = %v", err)
	}
	if !strings.Contains(err.Error(), "table, json, csv, tsv, markdown") {
		t.Errorf("error should list the valid formats: %v", err)
	}
}

func TestMarkdownAliasMd(t *testing.T) {
	srv := newFakeTBA(t, map[string]any{"/districts/2024": districts2024JSON})
	out, _, err := runCmd(t, srv, "district", "list", "--year", "2024", "--format", "md")
	requireNoError(t, err, "")
	requireContains(t, out, "| Key | Name | Abbreviation |")
	requireContains(t, out, "| --- | --- | --- |")
}

func TestAPIErrorIsSurfaced(t *testing.T) {
	srv := newFakeTBA(t, map[string]any{})
	_, _, err := runCmd(t, srv, "team", "view", "177")
	if err == nil {
		t.Fatal("want an error for an unknown path")
	}
	if !strings.Contains(err.Error(), "404") {
		t.Errorf("error should mention the status code: %v", err)
	}
}

func TestNoCacheFlagSkipsConditionalHeaders(t *testing.T) {
	srv := newFakeTBA(t, map[string]any{"/status": apiStatusJSON})
	setETag(t, srv, "/status", `"v1"`)

	// Warm the cache, then re-run with --no-cache.
	cacheDir := t.TempDir()
	t.Setenv("TBA_CACHE_DIR", cacheDir)
	_, _, err := runCmd(t, srv, "status")
	requireNoError(t, err, "")

	t.Setenv("TBA_CACHE_DIR", cacheDir)
	_, _, err = runCmd(t, srv, "status", "--no-cache")
	requireNoError(t, err, "")

	reqs := requestsTo(t, srv)
	last := reqs[len(reqs)-1]
	if got := last.Headers.Get("If-None-Match"); got != "" {
		t.Errorf("--no-cache should not send If-None-Match, got %q", got)
	}
}

func TestCompletionCommand(t *testing.T) {
	for _, shell := range []string{"bash", "zsh", "fish", "powershell"} {
		t.Run(shell, func(t *testing.T) {
			out, _, err := runCmd(t, nil, "completion", shell)
			requireNoError(t, err, "")
			if len(out) == 0 {
				t.Fatalf("no completion script emitted for %s", shell)
			}
			requireContains(t, out, "tba")
		})
	}
}

func TestCompletionUnknownShellShowsSupportedShells(t *testing.T) {
	out, _, err := runCmd(t, nil, "completion", "tcsh")
	requireNoError(t, err, "")
	// Cobra falls back to the completion command's help, which lists the
	// shells that are actually supported.
	for _, shell := range []string{"bash", "zsh", "fish", "powershell"} {
		requireContains(t, out, shell)
	}
	if strings.Contains(out, "tcsh") {
		t.Errorf("tcsh should not be offered as a shell:\n%s", out)
	}
}
