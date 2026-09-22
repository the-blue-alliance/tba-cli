package cmd

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"text/template"

	"go.yaml.in/yaml/v3"
)

// repoFile reads a file from the repository root, which is one level above the
// package these tests live in.
func repoFile(t *testing.T, name string) []byte {
	t.Helper()
	b, err := os.ReadFile(filepath.Join("..", name))
	if err != nil {
		t.Fatalf("reading %s: %v", name, err)
	}
	return b
}

func decodeYAML(t *testing.T, name string) map[string]any {
	t.Helper()
	var out map[string]any
	if err := yaml.Unmarshal(repoFile(t, name), &out); err != nil {
		t.Fatalf("parsing %s: %v", name, err)
	}
	return out
}

// The Slack announcement is the last thing a release does, so an announcer
// that fails marks an already-published release as a failed run. A repository
// or a fork without the SLACK_WEBHOOK secret has nothing to announce to, which
// is not an error.
func TestGoreleaserSkipsTheSlackAnnouncementWithoutAWebhook(t *testing.T) {
	config := decodeYAML(t, ".goreleaser.yml")
	announce, ok := config["announce"].(map[string]any)
	if !ok {
		t.Fatalf("announce = %v, want a mapping", config["announce"])
	}
	skip, ok := announce["skip"].(string)
	if !ok {
		t.Fatal("announce has no skip guard: a tag pushed without SLACK_WEBHOOK would fail the release after publishing it")
	}

	tmpl, err := template.New("skip").Parse(skip)
	if err != nil {
		t.Fatalf("announce.skip is not a valid template: %v", err)
	}
	for _, c := range []struct {
		name    string
		env     map[string]string
		skipped string
	}{
		{"with a webhook", map[string]string{"SLACK_WEBHOOK": "https://hooks.example/x"}, "false"},
		{"without one", map[string]string{}, "true"},
		{"with an empty one", map[string]string{"SLACK_WEBHOOK": ""}, "true"},
	} {
		t.Run(c.name, func(t *testing.T) {
			var buf bytes.Buffer
			if err := tmpl.Execute(&buf, struct{ Env map[string]string }{c.env}); err != nil {
				t.Fatalf("rendering announce.skip: %v", err)
			}
			if got := strings.TrimSpace(buf.String()); got != c.skipped {
				t.Errorf("skip = %q, want %q", got, c.skipped)
			}
		})
	}
}

// The workflow passes the webhook in, so the guard has something to read.
func TestReleaseWorkflowPassesTheWebhookThrough(t *testing.T) {
	body := string(repoFile(t, ".github/workflows/release.yml"))
	if !strings.Contains(body, "SLACK_WEBHOOK:") {
		t.Error("the release workflow does not set SLACK_WEBHOOK, so the announce guard can never be true")
	}
}

// Two tags pushed close together must not run two releases over the same
// artifacts, and a release in flight must never be cancelled: by the time it
// is cancellable it has already published.
func TestReleaseWorkflowSerialisesReleases(t *testing.T) {
	workflow := decodeYAML(t, ".github/workflows/release.yml")
	concurrency, ok := workflow["concurrency"].(map[string]any)
	if !ok {
		t.Fatalf("concurrency = %v, want a mapping", workflow["concurrency"])
	}
	if group, _ := concurrency["group"].(string); group == "" {
		t.Error("concurrency has no group")
	}
	if cancel, _ := concurrency["cancel-in-progress"].(bool); cancel {
		t.Error("cancel-in-progress is true; a release that has published artifacts must be allowed to finish")
	}
}

// spdxID names the license a LICENSE file carries, so that a packaging
// manifest can be checked against the file itself rather than against a
// constant somebody has to remember to change.
func spdxID(text string) string {
	switch {
	case strings.Contains(text, "MIT License"):
		return "MIT"
	case strings.Contains(text, "Apache License"):
		return "Apache-2.0"
	case strings.Contains(text, "BSD 3-Clause"):
		return "BSD-3-Clause"
	}
	return ""
}

// caskLicenseProblems reports every homebrew_casks entry whose declared
// license is missing or disagrees with the LICENSE file.
//
// A config with no cask at all has no problems: the repository is licensed by
// its LICENSE file whether or not anything packages the archives, and the
// Homebrew cask arrives on a branch of its own. What is never acceptable is a
// cask that installs these binaries without saying under what terms, or one
// that names terms the shipped file does not carry.
func caskLicenseProblems(config map[string]any, spdx string) []string {
	casks, ok := config["homebrew_casks"].([]any)
	if !ok {
		return nil
	}
	var problems []string
	for i, entry := range casks {
		cask, ok := entry.(map[string]any)
		if !ok {
			problems = append(problems, fmt.Sprintf("homebrew_casks[%d] is not a mapping", i))
			continue
		}
		name, _ := cask["name"].(string)
		if name == "" {
			name = fmt.Sprintf("homebrew_casks[%d]", i)
		}
		switch declared, _ := cask["license"].(string); {
		case declared == "":
			problems = append(problems, fmt.Sprintf("%s declares no license; the LICENSE file says %s", name, spdx))
		case declared != spdx:
			problems = append(problems, fmt.Sprintf("%s declares license %q, but the LICENSE file is %s", name, declared, spdx))
		}
	}
	return problems
}

// The release archives carry compiled binaries, so the repository needs a
// license file, and anything that packages those binaries has to agree with
// it.
func TestTheLicenseFileAndTheGoreleaserConfigAgree(t *testing.T) {
	spdx := spdxID(string(repoFile(t, "LICENSE")))
	if spdx == "" {
		t.Fatal("LICENSE is not a license this test recognises; teach spdxID about it")
	}
	for _, problem := range caskLicenseProblems(decodeYAML(t, ".goreleaser.yml"), spdx) {
		t.Error(problem)
	}
}

func TestCaskLicenseProblems(t *testing.T) {
	for _, c := range []struct {
		name   string
		config string
		want   int
	}{
		{"no cask entry at all", "builds:\n  - main: ./cmd/tba\n", 0},
		{"a cask that agrees", "homebrew_casks:\n  - name: tba\n    license: MIT\n", 0},
		{"a cask with no license", "homebrew_casks:\n  - name: tba\n", 1},
		{"a cask with the wrong license", "homebrew_casks:\n  - name: tba\n    license: Apache-2.0\n", 1},
	} {
		t.Run(c.name, func(t *testing.T) {
			var config map[string]any
			if err := yaml.Unmarshal([]byte(c.config), &config); err != nil {
				t.Fatalf("parsing the test config: %v", err)
			}
			if got := caskLicenseProblems(config, "MIT"); len(got) != c.want {
				t.Errorf("problems = %v, want %d of them", got, c.want)
			}
		})
	}
}
