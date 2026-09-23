package version

import (
	"strings"
	"testing"
)

func set(t *testing.T, v, commit, date string) {
	t.Helper()
	oldV, oldC, oldD := Version, Commit, Date
	Version, Commit, Date = v, commit, date
	t.Cleanup(func() { Version, Commit, Date = oldV, oldC, oldD })
}

func TestDefaultsAreADevBuild(t *testing.T) {
	if Version != "dev" {
		t.Errorf("Version = %q, want dev", Version)
	}
	if Commit != "" || Date != "" {
		t.Errorf("Commit/Date = %q/%q, want empty", Commit, Date)
	}
}

func TestStringIsJustTheVersionWithoutBuildMetadata(t *testing.T) {
	set(t, "1.2.3", "", "")
	if got := String(); got != "1.2.3" {
		t.Errorf("String() = %q, want 1.2.3", got)
	}
}

func TestStringIncludesCommitAndDate(t *testing.T) {
	set(t, "1.2.3", "abc1234", "2024-05-01T00:00:00Z")
	got := String()
	if got != "1.2.3 (abc1234, 2024-05-01T00:00:00Z)" {
		t.Errorf("String() = %q", got)
	}
}

func TestStringWithOnlyACommit(t *testing.T) {
	set(t, "1.2.3", "abc1234", "")
	if got := String(); got != "1.2.3 (abc1234)" {
		t.Errorf("String() = %q", got)
	}
}

func TestUserAgentCarriesTheVersion(t *testing.T) {
	set(t, "1.2.3", "abc1234", "2024-05-01T00:00:00Z")
	if got := UserAgent(); got != "tba-cli/1.2.3" {
		t.Errorf("UserAgent() = %q, want tba-cli/1.2.3", got)
	}
	if strings.Contains(UserAgent(), "abc1234") {
		t.Error("the User-Agent should not carry build metadata")
	}
}
