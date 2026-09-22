package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// The release configuration is only exercised when a tag is pushed, which is
// the worst moment to discover that a line has gone missing. These tests pin
// the handful of facts a release depends on: where the Homebrew tap lives, and
// that the workflow hands goreleaser the token it needs to push there.

func repoFile(t *testing.T, name string) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join("..", name))
	if err != nil {
		t.Fatalf("reading %s: %v", name, err)
	}
	return string(b)
}

func TestGoreleaserPublishesToTheTBATap(t *testing.T) {
	cfg := repoFile(t, ".goreleaser.yml")
	for _, want := range []string{
		"homebrew_casks:",
		"owner: the-blue-alliance",
		"name: homebrew-tap",
		"branch: main",
		`token: "{{ .Env.HOMEBREW_TAP_TOKEN }}"`,
		// A prerelease must not become what `brew install tba` hands out.
		"skip_upload: auto",
	} {
		requireContains(t, cfg, want)
	}
}

func TestGoreleaserCaskShipsManpagesAndCompletions(t *testing.T) {
	cfg := repoFile(t, ".goreleaser.yml")
	for _, want := range []string{
		`manpages:`,
		`"man/*.1"`,
		"bash: completions/tba.bash",
		"zsh: completions/tba.zsh",
		"fish: completions/tba.fish",
	} {
		requireContains(t, cfg, want)
	}
}

// Homebrew wants a license, but inventing one for a repository that has no
// LICENSE file would be a lie. If a license is ever added, this test says where
// to declare it.
func TestGoreleaserClaimsNoLicenseWhileTheRepoHasNone(t *testing.T) {
	_, err := os.Stat(filepath.Join("..", "LICENSE"))
	hasLicense := err == nil

	cfg := repoFile(t, ".goreleaser.yml")
	declares := strings.Contains(cfg, "license:")
	if !hasLicense && declares {
		t.Error(".goreleaser.yml declares a license, but the repository has no LICENSE file")
	}
	if hasLicense && !declares {
		t.Error("the repository has a LICENSE file; declare it in .goreleaser.yml")
	}
}

func TestReleaseWorkflowPassesTheTapToken(t *testing.T) {
	workflow := repoFile(t, filepath.Join(".github", "workflows", "release.yml"))
	requireContains(t, workflow, "HOMEBREW_TAP_TOKEN: ${{ secrets.HOMEBREW_TAP_TOKEN }}")
}
