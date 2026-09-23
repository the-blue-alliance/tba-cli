package config

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

const localURL = "http://localhost:8080/api/v3"

// configEnv points the package at a fresh directory with no env key set.
func configEnv(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("TBA_CONFIG_DIR", dir)
	t.Setenv("TBA_AUTH_KEY", "")
	return dir
}

// mustAuthFile is AuthFile for the many tests that have a config dir set and
// so cannot fail.
func mustAuthFile(t *testing.T) string {
	t.Helper()
	path, err := AuthFile()
	if err != nil {
		t.Fatalf("AuthFile: %v", err)
	}
	return path
}

func writeLegacyAuthFile(t *testing.T, key string) {
	t.Helper()
	if err := os.WriteFile(mustAuthFile(t), []byte("api_key: "+key+"\n"), 0600); err != nil {
		t.Fatalf("writing legacy auth file: %v", err)
	}
}

func TestAuthFileFollowsTBAConfigDir(t *testing.T) {
	dir := configEnv(t)
	if want := filepath.Join(dir, "auth.yaml"); mustAuthFile(t) != want {
		t.Errorf("AuthFile() = %q, want %q", mustAuthFile(t), want)
	}
}

func TestSaveThenGet(t *testing.T) {
	configEnv(t)
	if err := SaveAPIKey("prod-key", DefaultBaseURL); err != nil {
		t.Fatalf("SaveAPIKey: %v", err)
	}
	got, err := GetAPIKey(DefaultBaseURL)
	if err != nil {
		t.Fatalf("GetAPIKey: %v", err)
	}
	if got != "prod-key" {
		t.Errorf("key = %q", got)
	}
}

func TestEmptyBaseURLMeansTheDefault(t *testing.T) {
	configEnv(t)
	if err := SaveAPIKey("prod-key", ""); err != nil {
		t.Fatalf("SaveAPIKey: %v", err)
	}
	got, err := GetAPIKey(DefaultBaseURL)
	if err != nil {
		t.Fatalf("GetAPIKey: %v", err)
	}
	if got != "prod-key" {
		t.Errorf("key = %q", got)
	}
}

func TestKeysAreStoredPerBaseURL(t *testing.T) {
	configEnv(t)
	if err := SaveAPIKey("prod-key", DefaultBaseURL); err != nil {
		t.Fatalf("SaveAPIKey: %v", err)
	}
	if err := SaveAPIKey("local-key", localURL); err != nil {
		t.Fatalf("SaveAPIKey: %v", err)
	}

	prod, err := GetAPIKey(DefaultBaseURL)
	if err != nil || prod != "prod-key" {
		t.Errorf("prod key = %q (%v)", prod, err)
	}
	local, err := GetAPIKey(localURL)
	if err != nil || local != "local-key" {
		t.Errorf("local key = %q (%v)", local, err)
	}
}

func TestSaveOverwritesTheKeyForOneBaseURL(t *testing.T) {
	configEnv(t)
	if err := SaveAPIKey("old", localURL); err != nil {
		t.Fatalf("SaveAPIKey: %v", err)
	}
	if err := SaveAPIKey("new", localURL); err != nil {
		t.Fatalf("SaveAPIKey: %v", err)
	}
	got, err := GetAPIKey(localURL)
	if err != nil || got != "new" {
		t.Errorf("key = %q (%v)", got, err)
	}
}

func TestRemoveAPIKey(t *testing.T) {
	configEnv(t)
	if err := SaveAPIKey("prod-key", DefaultBaseURL); err != nil {
		t.Fatalf("SaveAPIKey: %v", err)
	}
	if err := SaveAPIKey("local-key", localURL); err != nil {
		t.Fatalf("SaveAPIKey: %v", err)
	}

	if err := RemoveAPIKey(localURL); err != nil {
		t.Fatalf("RemoveAPIKey: %v", err)
	}
	if _, err := GetAPIKey(localURL); err == nil {
		t.Error("the removed key is still readable")
	}
	if got, err := GetAPIKey(DefaultBaseURL); err != nil || got != "prod-key" {
		t.Errorf("removing one URL disturbed another: %q (%v)", got, err)
	}
}

func TestRemoveAPIKeyWithNoConfigFile(t *testing.T) {
	configEnv(t)
	err := RemoveAPIKey(DefaultBaseURL)
	if err == nil || err.Error() != "not authenticated" {
		t.Errorf("error = %v, want \"not authenticated\"", err)
	}
}

func TestRemoveAPIKeyForAnUnknownBaseURL(t *testing.T) {
	configEnv(t)
	if err := SaveAPIKey("prod-key", DefaultBaseURL); err != nil {
		t.Fatalf("SaveAPIKey: %v", err)
	}
	err := RemoveAPIKey(localURL)
	if err == nil || err.Error() != "not authenticated for "+localURL {
		t.Errorf("error = %v", err)
	}
}

func TestGetAPIKeyErrorWhenNoConfigFile(t *testing.T) {
	configEnv(t)
	_, err := GetAPIKey(DefaultBaseURL)
	if err == nil {
		t.Fatal("want an error with no config file")
	}
	want := "not authenticated for " + DefaultBaseURL + ". Run 'tba auth login' first"
	if err.Error() != want {
		t.Errorf("error = %q, want %q", err.Error(), want)
	}
}

func TestGetAPIKeyErrorNamesTheBaseURLFlag(t *testing.T) {
	configEnv(t)
	if err := SaveAPIKey("prod-key", DefaultBaseURL); err != nil {
		t.Fatalf("SaveAPIKey: %v", err)
	}
	_, err := GetAPIKey(localURL)
	if err == nil {
		t.Fatal("want an error for an unknown base URL")
	}
	if !strings.Contains(err.Error(), "tba auth login --base-url "+localURL) {
		t.Errorf("error = %q", err.Error())
	}
}

func TestTBAAuthKeyEnvOverridesTheFile(t *testing.T) {
	configEnv(t)
	if err := SaveAPIKey("file-key", DefaultBaseURL); err != nil {
		t.Fatalf("SaveAPIKey: %v", err)
	}
	t.Setenv("TBA_AUTH_KEY", "env-key")

	for _, url := range []string{DefaultBaseURL, localURL, ""} {
		got, err := GetAPIKey(url)
		if err != nil {
			t.Fatalf("GetAPIKey(%q): %v", url, err)
		}
		if got != "env-key" {
			t.Errorf("GetAPIKey(%q) = %q, want env-key", url, got)
		}
	}
}

func TestLegacyAPIKeyIsReadForTheDefaultURL(t *testing.T) {
	configEnv(t)
	writeLegacyAuthFile(t, "legacy-key")

	got, err := GetAPIKey(DefaultBaseURL)
	if err != nil {
		t.Fatalf("GetAPIKey: %v", err)
	}
	if got != "legacy-key" {
		t.Errorf("key = %q, want legacy-key", got)
	}
}

func TestLegacyAPIKeyIsNotUsedForOtherURLs(t *testing.T) {
	configEnv(t)
	writeLegacyAuthFile(t, "legacy-key")

	if _, err := GetAPIKey(localURL); err == nil {
		t.Error("the legacy key must not apply to a custom base URL")
	}
}

func TestSaveMigratesTheLegacyKey(t *testing.T) {
	configEnv(t)
	writeLegacyAuthFile(t, "legacy-key")

	if err := SaveAPIKey("local-key", localURL); err != nil {
		t.Fatalf("SaveAPIKey: %v", err)
	}

	prod, err := GetAPIKey(DefaultBaseURL)
	if err != nil {
		t.Fatalf("GetAPIKey(default): %v", err)
	}
	if prod != "legacy-key" {
		t.Errorf("migrated key = %q, want legacy-key", prod)
	}
	local, err := GetAPIKey(localURL)
	if err != nil || local != "local-key" {
		t.Errorf("local key = %q (%v)", local, err)
	}

	b, err := os.ReadFile(mustAuthFile(t))
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if strings.Contains(string(b), "api_key: legacy-key") {
		t.Errorf("the legacy api_key entry survived migration:\n%s", b)
	}
}

func TestRemoveMigratesTheLegacyKey(t *testing.T) {
	configEnv(t)
	writeLegacyAuthFile(t, "legacy-key")

	// Removing a URL that is not stored still migrates and reports clearly.
	if err := RemoveAPIKey(localURL); err == nil {
		t.Error("want an error removing an unknown base URL")
	}

	if err := RemoveAPIKey(DefaultBaseURL); err != nil {
		t.Fatalf("RemoveAPIKey: %v", err)
	}
	if _, err := GetAPIKey(DefaultBaseURL); err == nil {
		t.Error("the migrated legacy key should be gone")
	}
}

func TestAuthFileIsMode0600AfterSave(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Unix file modes are not meaningful on Windows")
	}
	configEnv(t)
	if err := SaveAPIKey("prod-key", DefaultBaseURL); err != nil {
		t.Fatalf("SaveAPIKey: %v", err)
	}
	fi, err := os.Stat(mustAuthFile(t))
	if err != nil {
		t.Fatalf("Stat: %v", err)
	}
	if fi.Mode().Perm() != 0600 {
		t.Errorf("auth file mode = %04o, want 0600", fi.Mode().Perm())
	}
}

func TestAuthFileIsMode0600AfterRemove(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Unix file modes are not meaningful on Windows")
	}
	configEnv(t)
	if err := SaveAPIKey("prod-key", DefaultBaseURL); err != nil {
		t.Fatalf("SaveAPIKey: %v", err)
	}
	if err := SaveAPIKey("local-key", localURL); err != nil {
		t.Fatalf("SaveAPIKey: %v", err)
	}
	if err := RemoveAPIKey(localURL); err != nil {
		t.Fatalf("RemoveAPIKey: %v", err)
	}
	fi, err := os.Stat(mustAuthFile(t))
	if err != nil {
		t.Fatalf("Stat: %v", err)
	}
	if fi.Mode().Perm() != 0600 {
		t.Errorf("auth file mode = %04o, want 0600", fi.Mode().Perm())
	}
}

func TestGetTightensWidePermissionsOnLegacyFiles(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Unix file modes are not meaningful on Windows")
	}
	configEnv(t)
	writeLegacyAuthFile(t, "legacy-key")
	if err := os.Chmod(mustAuthFile(t), 0644); err != nil {
		t.Fatalf("Chmod: %v", err)
	}

	if _, err := GetAPIKey(DefaultBaseURL); err != nil {
		t.Fatalf("GetAPIKey: %v", err)
	}
	fi, err := os.Stat(mustAuthFile(t))
	if err != nil {
		t.Fatalf("Stat: %v", err)
	}
	if fi.Mode().Perm() != 0600 {
		t.Errorf("auth file mode = %04o, want it tightened to 0600", fi.Mode().Perm())
	}
}

func TestSaveCreatesTheConfigDirectory(t *testing.T) {
	base := t.TempDir()
	dir := filepath.Join(base, "nested", "tba")
	t.Setenv("TBA_CONFIG_DIR", dir)
	t.Setenv("TBA_AUTH_KEY", "")

	if err := SaveAPIKey("prod-key", DefaultBaseURL); err != nil {
		t.Fatalf("SaveAPIKey: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "auth.yaml")); err != nil {
		t.Errorf("auth file was not created: %v", err)
	}
}

// clearHome removes every variable os.UserHomeDir consults, so that it fails
// the way it does for a user with no home directory.
func clearHome(t *testing.T) {
	t.Helper()
	t.Setenv("HOME", "")
	t.Setenv("USERPROFILE", "")
	t.Setenv("HOMEDRIVE", "")
	t.Setenv("HOMEPATH", "")
	t.Setenv("home", "")
}

func TestAuthFileFailsWithoutAHomeDirectory(t *testing.T) {
	t.Setenv("TBA_CONFIG_DIR", "")
	t.Setenv("TBA_AUTH_KEY", "")
	clearHome(t)

	path, err := AuthFile()
	if err == nil {
		t.Fatalf("AuthFile() = %q, want an error rather than a path relative to the working directory", path)
	}
	if !strings.Contains(err.Error(), "TBA_CONFIG_DIR") {
		t.Errorf("the error should name the way out: %v", err)
	}
}

func TestGetAPIKeyFailsWithoutAHomeDirectory(t *testing.T) {
	t.Setenv("TBA_CONFIG_DIR", "")
	t.Setenv("TBA_AUTH_KEY", "")
	clearHome(t)

	if _, err := GetAPIKey(""); err == nil {
		t.Fatal("want an error when the home directory cannot be found")
	}
}

func TestSaveAPIKeyFailsWithoutAHomeDirectory(t *testing.T) {
	t.Setenv("TBA_CONFIG_DIR", "")
	t.Setenv("TBA_AUTH_KEY", "")
	clearHome(t)

	if err := SaveAPIKey("k", ""); err == nil {
		t.Fatal("want an error rather than a config file in the working directory")
	}
	if _, err := os.Stat("auth.yaml"); err == nil {
		t.Error("SaveAPIKey wrote auth.yaml into the working directory")
	}
}

func TestRemoveAPIKeyFailsWithoutAHomeDirectory(t *testing.T) {
	t.Setenv("TBA_CONFIG_DIR", "")
	t.Setenv("TBA_AUTH_KEY", "")
	clearHome(t)

	if err := RemoveAPIKey(""); err == nil {
		t.Fatal("want an error when the home directory cannot be found")
	}
}
