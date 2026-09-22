package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/the-blue-alliance/tba-cli/internal/config"
)

// authEnv points config at a fresh directory and clears the env override so
// the on-disk auth file is what the commands actually read.
func authEnv(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("TBA_CONFIG_DIR", dir)
	t.Setenv("TBA_CACHE_DIR", t.TempDir())
	t.Setenv("TBA_AUTH_KEY", "")
	return dir
}

func TestAuthLoginWithKeyFlag(t *testing.T) {
	dir := authEnv(t)
	out, _, err := runCmd(t, nil, "auth", "login", "--key", "abcd1234secret")
	requireNoError(t, err, "")
	requireContains(t, out, "Authenticated successfully for https://www.thebluealliance.com/api/v3.")

	if _, err := os.Stat(filepath.Join(dir, "auth.yaml")); err != nil {
		t.Fatalf("auth file was not written: %v", err)
	}
	key, err := config.GetAPIKey("")
	if err != nil {
		t.Fatalf("GetAPIKey: %v", err)
	}
	if key != "abcd1234secret" {
		t.Errorf("stored key = %q", key)
	}
}

func TestAuthLoginReadsStdin(t *testing.T) {
	authEnv(t)
	out, _, err := runCmdStdin(t, nil, "  typed-key  \n", "auth", "login")
	requireNoError(t, err, "")
	requireContains(t, out, "Enter your TBA API key for https://www.thebluealliance.com/api/v3: ")

	key, err := config.GetAPIKey("")
	if err != nil {
		t.Fatalf("GetAPIKey: %v", err)
	}
	if key != "typed-key" {
		t.Errorf("stored key = %q, want the trimmed stdin value", key)
	}
}

func TestAuthLoginRejectsEmptyKey(t *testing.T) {
	authEnv(t)
	_, _, err := runCmdStdin(t, nil, "\n", "auth", "login")
	if err == nil || !strings.Contains(err.Error(), "API key cannot be empty") {
		t.Fatalf("error = %v, want an empty-key error", err)
	}
}

func TestAuthLoginStoresKeysPerBaseURL(t *testing.T) {
	authEnv(t)
	if _, _, err := runCmd(t, nil, "auth", "login", "--key", "prod-key"); err != nil {
		t.Fatalf("prod login: %v", err)
	}
	if _, _, err := runCmd(t, nil, "auth", "login",
		"--base-url", "http://localhost:8080/api/v3", "--key", "local-key"); err != nil {
		t.Fatalf("local login: %v", err)
	}

	prod, err := config.GetAPIKey("")
	if err != nil || prod != "prod-key" {
		t.Errorf("prod key = %q (%v)", prod, err)
	}
	local, err := config.GetAPIKey("http://localhost:8080/api/v3")
	if err != nil || local != "local-key" {
		t.Errorf("local key = %q (%v)", local, err)
	}
}

func TestAuthStatusWhenNotAuthenticated(t *testing.T) {
	authEnv(t)
	out, _, err := runCmd(t, nil, "auth", "status")
	requireNoError(t, err, "")
	requireContains(t, out, "Not authenticated for https://www.thebluealliance.com/api/v3.")
}

func TestAuthStatusMasksTheKey(t *testing.T) {
	dir := authEnv(t)
	if _, _, err := runCmd(t, nil, "auth", "login", "--key", "abcd1234secret"); err != nil {
		t.Fatalf("login: %v", err)
	}
	out, _, err := runCmd(t, nil, "auth", "status")
	requireNoError(t, err, "")

	requireContains(t, out, "Authenticated with key: **********cret")
	requireContains(t, out, "Base URL: https://www.thebluealliance.com/api/v3")
	requireContains(t, out, "Config file: "+filepath.Join(dir, "auth.yaml"))
	if strings.Contains(out, "abcd1234secret") {
		t.Error("the full key must not be printed")
	}
}

func TestAuthStatusUsesEnvOverride(t *testing.T) {
	authEnv(t)
	t.Setenv("TBA_AUTH_KEY", "envkey-value")
	out, _, err := runCmd(t, nil, "auth", "status")
	requireNoError(t, err, "")
	requireContains(t, out, "Authenticated with key: ********alue")
}

func TestAuthLogout(t *testing.T) {
	authEnv(t)
	if _, _, err := runCmd(t, nil, "auth", "login", "--key", "abcd1234secret"); err != nil {
		t.Fatalf("login: %v", err)
	}
	out, _, err := runCmd(t, nil, "auth", "logout")
	requireNoError(t, err, "")
	requireContains(t, out, "Logged out from https://www.thebluealliance.com/api/v3.")

	if _, err := config.GetAPIKey(""); err == nil {
		t.Error("key should be gone after logout")
	}
}

func TestAuthLogoutWhenNotAuthenticated(t *testing.T) {
	authEnv(t)
	out, _, err := runCmd(t, nil, "auth", "logout")
	// Logout reports the problem on stdout and still exits zero.
	requireNoError(t, err, "")
	requireContains(t, out, "not authenticated")
}

func TestAuthSubcommandsAreRegistered(t *testing.T) {
	got := subcommandNames(t, "auth")
	for _, want := range []string{"login", "status", "logout"} {
		if !contains(got, want) {
			t.Errorf("auth %s is not registered (have %v)", want, got)
		}
	}
}

func TestMaskKey(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{"", ""},
		{"a", "*"},
		{"ab", "**"},
		{"abc", "***"},
		{"abcd", "****"},
		{"abcde", "*bcde"},
		{"abcd1234", "****1234"},
		{"abcd1234secret", "**********cret"},
	}
	for _, c := range cases {
		if got := maskKey(c.in); got != c.want {
			t.Errorf("maskKey(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

// Regression test: masking used to slice the first four bytes off the key and
// panicked on anything shorter.
func TestAuthStatusWithAShortKeyDoesNotPanic(t *testing.T) {
	authEnv(t)
	t.Setenv("TBA_AUTH_KEY", "abc")
	out, _, err := runCmd(t, nil, "auth", "status")
	requireNoError(t, err, "")
	requireContains(t, out, "Authenticated with key: ***")
	if strings.Contains(out, "abc\n") {
		t.Error("a short key must not be printed in full")
	}
}
