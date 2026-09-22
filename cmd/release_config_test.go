package cmd

import (
	"bytes"
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
