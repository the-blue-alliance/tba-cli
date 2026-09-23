package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/the-blue-alliance/tba-cli/internal/version"
)

// dirContents reads a whole directory into a map, so two generations can be
// compared byte for byte.
func dirContents(t *testing.T, dir string) map[string]string {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("reading %s: %v", dir, err)
	}
	out := make(map[string]string, len(entries))
	for _, e := range entries {
		b, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			t.Fatalf("reading %s: %v", e.Name(), err)
		}
		out[e.Name()] = string(b)
	}
	if len(out) == 0 {
		t.Fatalf("%s is empty", dir)
	}
	return out
}

func requireSameFiles(t *testing.T, a, b map[string]string) {
	t.Helper()
	if len(a) != len(b) {
		t.Fatalf("generated %d files then %d files", len(a), len(b))
	}
	for name, want := range a {
		got, ok := b[name]
		if !ok {
			t.Errorf("%s was generated only once", name)
			continue
		}
		if got != want {
			t.Errorf("%s differs between two generations", name)
		}
	}
}

func TestDocsManIsReproducible(t *testing.T) {
	first, second := t.TempDir(), t.TempDir()
	_, _, err := runCmd(t, nil, "docs", "man", "--dir", first)
	requireNoError(t, err, "")
	_, _, err = runCmd(t, nil, "docs", "man", "--dir", second)
	requireNoError(t, err, "")

	requireSameFiles(t, dirContents(t, first), dirContents(t, second))
}

func TestDocsManWritesAPageForEveryVisibleCommand(t *testing.T) {
	dir := t.TempDir()
	_, _, err := runCmd(t, nil, "docs", "man", "--dir", dir)
	requireNoError(t, err, "")

	pages := dirContents(t, dir)
	for _, want := range []string{"tba.1", "tba-team-view.1", "tba-event-matches.1", "tba-version.1"} {
		if _, ok := pages[want]; !ok {
			t.Errorf("no %s was generated", want)
		}
	}
	// The generators are an implementation detail of the release, not
	// something to document to users.
	if _, ok := pages["tba-docs.1"]; ok {
		t.Error("the hidden docs command got a man page")
	}
	requireContains(t, pages["tba.1"], "The Blue Alliance CLI")
}

func TestDocsManDateComesFromSourceDateEpoch(t *testing.T) {
	// 1717200000 is 2024-06-01T00:00:00Z.
	t.Setenv("SOURCE_DATE_EPOCH", "1717200000")
	dir := t.TempDir()
	_, _, err := runCmd(t, nil, "docs", "man", "--dir", dir)
	requireNoError(t, err, "")

	requireContains(t, dirContents(t, dir)["tba.1"], `"Jun 2024"`)
}

func TestDocsManRejectsAnUnparsableSourceDateEpoch(t *testing.T) {
	t.Setenv("SOURCE_DATE_EPOCH", "yesterday")
	_, _, err := runCmd(t, nil, "docs", "man", "--dir", t.TempDir())
	requireErrorContains(t, err, "SOURCE_DATE_EPOCH")
}

// manDateCmd is a `docs man` command with its flags parsed, which is what
// docsDate reads --date from.
func manDateCmd(t *testing.T, args ...string) *cobra.Command {
	t.Helper()
	cmd := newDocsManCmd()
	if err := cmd.ParseFlags(args); err != nil {
		t.Fatalf("parsing %v: %v", args, err)
	}
	return cmd
}

func TestDocsManDateFlagWinsOverTheEnvironment(t *testing.T) {
	t.Setenv("SOURCE_DATE_EPOCH", "1717200000") // June 2024
	dir := t.TempDir()
	_, _, err := runCmd(t, nil, "docs", "man", "--dir", dir, "--date", "2023-03-04T05:06:07Z")
	requireNoError(t, err, "")

	requireContains(t, dirContents(t, dir)["tba.1"], `"Mar 2023"`)
}

func TestDocsManDateFlagAcceptsAUnixTimestamp(t *testing.T) {
	got, err := docsDate(manDateCmd(t, "--date", "1717200000"))
	if err != nil {
		t.Fatalf("docsDate: %v", err)
	}
	if got.Format("2006-01-02") != "2024-06-01" {
		t.Errorf("docsDate = %v, want 2024-06-01", got)
	}
}

func TestDocsManRejectsAnUnparsableDateFlag(t *testing.T) {
	_, _, err := runCmd(t, nil, "docs", "man", "--dir", t.TempDir(), "--date", "last Tuesday")
	requireErrorContains(t, err, "--date")
}

func TestDocsDateFallsBackToTheBuildDate(t *testing.T) {
	t.Setenv("SOURCE_DATE_EPOCH", "")
	// A stamped release binary: no SOURCE_DATE_EPOCH, but it knows when it was
	// built.
	oldV, oldD := version.Version, version.Date
	version.Version, version.Date = "1.2.3", "2023-03-04T05:06:07Z"
	t.Cleanup(func() { version.Version, version.Date = oldV, oldD })

	got, err := docsDate(manDateCmd(t))
	if err != nil {
		t.Fatalf("docsDate: %v", err)
	}
	if got.Format("2006-01-02") != "2023-03-04" {
		t.Errorf("docsDate = %v, want the build date", got)
	}
}

func TestDocsDateFallsBackToAFixedEpoch(t *testing.T) {
	t.Setenv("SOURCE_DATE_EPOCH", "")
	oldV, oldD := version.Version, version.Date
	version.Version, version.Date = "1.2.3", ""
	t.Cleanup(func() { version.Version, version.Date = oldV, oldD })

	got, err := docsDate(manDateCmd(t))
	if err != nil {
		t.Fatalf("docsDate: %v", err)
	}
	if !got.Equal(fallbackDocDate) {
		t.Errorf("docsDate = %v, want the fixed fallback %v", got, fallbackDocDate)
	}
}

func TestDocsMarkdownIsReproducible(t *testing.T) {
	first, second := t.TempDir(), t.TempDir()
	_, _, err := runCmd(t, nil, "docs", "markdown", "--dir", first)
	requireNoError(t, err, "")
	_, _, err = runCmd(t, nil, "docs", "markdown", "--dir", second)
	requireNoError(t, err, "")

	pages := dirContents(t, first)
	requireSameFiles(t, pages, dirContents(t, second))

	if _, ok := pages["tba_team_view.md"]; !ok {
		t.Errorf("no page for `team view`; got %d pages", len(pages))
	}
	// Cobra's "Auto generated by spf13/cobra on 4-Sep-2024" footer is the
	// clock leaking into the output; it has to stay off.
	for name, body := range pages {
		if strings.Contains(body, "Auto generated by") {
			t.Errorf("%s carries the dated auto-generation footer", name)
		}
	}
}

func TestDocsCompletionsWritesOneScriptPerShell(t *testing.T) {
	first, second := t.TempDir(), t.TempDir()
	_, _, err := runCmd(t, nil, "docs", "completions", "--dir", first)
	requireNoError(t, err, "")
	_, _, err = runCmd(t, nil, "docs", "completions", "--dir", second)
	requireNoError(t, err, "")

	scripts := dirContents(t, first)
	requireSameFiles(t, scripts, dirContents(t, second))
	for _, name := range []string{"tba.bash", "tba.zsh", "tba.fish"} {
		body, ok := scripts[name]
		if !ok {
			t.Errorf("no %s was generated", name)
			continue
		}
		if !strings.Contains(body, "tba") {
			t.Errorf("%s does not mention the binary", name)
		}
	}
}

func TestDocsCreatesTheDirectory(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "nested", "man")
	_, _, err := runCmd(t, nil, "docs", "man", "--dir", dir)
	requireNoError(t, err, "")
	dirContents(t, dir)
}

func TestDocsRejectsAnEmptyDir(t *testing.T) {
	_, _, err := runCmd(t, nil, "docs", "man", "--dir", "")
	requireErrorContains(t, err, "--dir")
}

func TestDocsIsHiddenFromHelp(t *testing.T) {
	out, _, err := runCmd(t, nil, "--help")
	requireNoError(t, err, "")
	for _, line := range lines(out) {
		if strings.HasPrefix(strings.TrimSpace(line), "docs ") {
			t.Errorf("docs is listed in `tba --help`: %q", line)
		}
	}

	// Hidden, but not secret: it still has its own help.
	out, _, err = runCmd(t, nil, "docs", "--help")
	requireNoError(t, err, "")
	for _, want := range []string{"man", "markdown", "completions", "SOURCE_DATE_EPOCH"} {
		requireContains(t, out, want)
	}
}

func TestDocsSubcommands(t *testing.T) {
	got := subcommandNames(t, "docs")
	for _, want := range []string{"man", "markdown", "completions"} {
		if !contains(got, want) {
			t.Errorf("docs subcommands %v are missing %q", got, want)
		}
	}
}
