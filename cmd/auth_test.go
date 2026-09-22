package cmd

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/the-blue-alliance/tba-cli/internal/clierr"
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

// authServer is a fake TBA that answers /status, which is what `auth login`
// checks a key against.
func authServer(t *testing.T) *httptest.Server {
	t.Helper()
	return newFakeTBA(t, map[string]any{"/status": apiStatusJSON})
}

// rejectingServer answers every request with 401, like TBA does for a key it
// does not know.
func rejectingServer(t *testing.T) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"Error":"Invalid X-TBA-Auth-Key"}`))
	}))
	t.Cleanup(srv.Close)
	return srv
}

func TestAuthLoginWithKeyFlag(t *testing.T) {
	dir := authEnv(t)
	srv := authServer(t)
	stdout, stderr, err := runCmd(t, srv, "auth", "login", "--key", "abcd1234secret")
	requireNoError(t, err, stderr)

	requireContains(t, stderr, "Authenticated successfully for "+srv.URL+".")
	if stdout != "" {
		t.Errorf("login must write nothing to stdout, got:\n%s", stdout)
	}
	if _, err := os.Stat(filepath.Join(dir, "auth.yaml")); err != nil {
		t.Fatalf("auth file was not written: %v", err)
	}
	key, err := config.GetAPIKey(srv.URL)
	if err != nil {
		t.Fatalf("GetAPIKey: %v", err)
	}
	if key != "abcd1234secret" {
		t.Errorf("stored key = %q", key)
	}
}

func TestAuthLoginPromptsOnStderrAndReadsStdin(t *testing.T) {
	authEnv(t)
	srv := authServer(t)
	stdout, stderr, err := runCmdStdin(t, srv, "  typed-key  \n", "auth", "login")
	requireNoError(t, err, stderr)

	requireContains(t, stderr, "Enter your TBA API key for "+srv.URL+": ")
	if stdout != "" {
		t.Errorf("the prompt must not go to stdout, got:\n%s", stdout)
	}
	key, err := config.GetAPIKey(srv.URL)
	if err != nil {
		t.Fatalf("GetAPIKey: %v", err)
	}
	if key != "typed-key" {
		t.Errorf("stored key = %q, want the trimmed stdin value", key)
	}
}

// `tba auth login < key.txt` where the file has no trailing newline.
func TestAuthLoginReadsAKeyWithoutATrailingNewline(t *testing.T) {
	authEnv(t)
	srv := authServer(t)
	_, stderr, err := runCmdStdin(t, srv, "piped-key", "auth", "login")
	requireNoError(t, err, stderr)

	key, err := config.GetAPIKey(srv.URL)
	if err != nil {
		t.Fatalf("GetAPIKey: %v", err)
	}
	if key != "piped-key" {
		t.Errorf("stored key = %q", key)
	}
}

func TestAuthLoginRejectsEmptyKey(t *testing.T) {
	authEnv(t)
	srv := authServer(t)
	_, _, err := runCmdStdin(t, srv, "\n", "auth", "login")
	if err == nil || !strings.Contains(err.Error(), "API key cannot be empty") {
		t.Fatalf("error = %v, want an empty-key error", err)
	}
	if got := clierr.ExitCode(err); got != clierr.ExitUsage {
		t.Errorf("exit code = %d, want %d", got, clierr.ExitUsage)
	}
}

func TestAuthLoginValidatesTheKeyBeforeSaving(t *testing.T) {
	dir := authEnv(t)
	srv := rejectingServer(t)

	_, _, err := runCmd(t, srv, "auth", "login", "--key", "bogus-key")
	if err == nil {
		t.Fatal("want an error when the API rejects the key")
	}
	if got := clierr.ExitCode(err); got != clierr.ExitAuth {
		t.Errorf("exit code = %d, want %d", got, clierr.ExitAuth)
	}
	requireErrorContains(t, err, "rejected")
	if _, statErr := os.Stat(filepath.Join(dir, "auth.yaml")); statErr == nil {
		t.Error("a rejected key must not be written to disk")
	}
}

func TestAuthLoginChecksTheKeyAgainstStatus(t *testing.T) {
	authEnv(t)
	srv := authServer(t)
	_, stderr, err := runCmd(t, srv, "auth", "login", "--key", "abcd1234secret")
	requireNoError(t, err, stderr)

	reqs := requestsTo(t, srv)
	if len(reqs) != 1 || reqs[0].Path != "/status" {
		t.Fatalf("login should check /status, requested %v", requestPaths(t, srv))
	}
	if got := reqs[0].Headers.Get("X-TBA-Auth-Key"); got != "abcd1234secret" {
		t.Errorf("the key under test should be sent, got %q", got)
	}
}

func TestAuthLoginStoresKeysPerBaseURL(t *testing.T) {
	authEnv(t)
	prodSrv := authServer(t)
	localSrv := authServer(t)

	if _, _, err := runCmd(t, prodSrv, "auth", "login", "--key", "prod-key"); err != nil {
		t.Fatalf("prod login: %v", err)
	}
	if _, _, err := runCmd(t, localSrv, "auth", "login", "--key", "local-key"); err != nil {
		t.Fatalf("local login: %v", err)
	}

	prod, err := config.GetAPIKey(prodSrv.URL)
	if err != nil || prod != "prod-key" {
		t.Errorf("prod key = %q (%v)", prod, err)
	}
	local, err := config.GetAPIKey(localSrv.URL)
	if err != nil || local != "local-key" {
		t.Errorf("local key = %q (%v)", local, err)
	}
}

func TestAuthStatusWhenNotAuthenticatedExitsFour(t *testing.T) {
	authEnv(t)
	stdout, _, err := runCmd(t, nil, "auth", "status")
	if err == nil {
		t.Fatal("want an error when no key is configured")
	}
	if got := clierr.ExitCode(err); got != clierr.ExitAuth {
		t.Errorf("exit code = %d, want %d", got, clierr.ExitAuth)
	}
	requireErrorContains(t, err, "not authenticated")
	if stdout != "" {
		t.Errorf("nothing should reach stdout, got:\n%s", stdout)
	}
}

// A first-run dead end: someone who has never had a key is told to log in but
// not where a key comes from.
func TestAuthErrorsAndHelpSayWhereToGetAKey(t *testing.T) {
	const page = "https://www.thebluealliance.com/account"

	authEnv(t)
	_, _, err := runCmd(t, nil, "auth", "status")
	if err == nil {
		t.Fatal("want an error when no key is configured")
	}
	requireErrorContains(t, err, page)

	out, _, helpErr := runCmd(t, nil, "auth", "login", "--help")
	requireNoError(t, helpErr, "")
	requireContains(t, out, page)
}

func TestHTTP401SaysWhereToGetAKey(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"Error":"Invalid X-TBA-Auth-Key"}`))
	}))
	t.Cleanup(srv.Close)

	t.Setenv("TBA_AUTH_KEY", "bogus")
	err := requireExitCode(t, clierr.ExitAuth, srv, "status")
	requireErrorContains(t, err, "https://www.thebluealliance.com/account")
}

func TestAuthStatusTableMasksTheKey(t *testing.T) {
	dir := authEnv(t)
	srv := authServer(t)
	if _, _, err := runCmd(t, srv, "auth", "login", "--key", "abcd1234secret"); err != nil {
		t.Fatalf("login: %v", err)
	}
	out, _, err := runCmd(t, srv, "auth", "status", "--format", "table")
	requireNoError(t, err, "")

	requireContains(t, out, "Authenticated with key:")
	requireContains(t, out, "**********cret")
	requireContains(t, out, "Base URL:")
	requireContains(t, out, srv.URL)
	requireContains(t, out, "Config file:")
	requireContains(t, out, filepath.Join(dir, "auth.yaml"))
	if strings.Contains(out, "abcd1234secret") {
		t.Error("the full key must not be printed")
	}
}

func TestAuthStatusJSON(t *testing.T) {
	dir := authEnv(t)
	srv := authServer(t)
	if _, _, err := runCmd(t, srv, "auth", "login", "--key", "abcd1234secret"); err != nil {
		t.Fatalf("login: %v", err)
	}
	out, _, err := runCmd(t, srv, "auth", "status", "--format", "json")
	requireNoError(t, err, "")

	obj, ok := decodeJSON(t, out).(map[string]any)
	if !ok {
		t.Fatalf("want a JSON object, got:\n%s", out)
	}
	if obj["authenticated"] != true {
		t.Errorf("authenticated = %v", obj["authenticated"])
	}
	if obj["base_url"] != srv.URL {
		t.Errorf("base_url = %v", obj["base_url"])
	}
	if obj["key_masked"] != "**********cret" {
		t.Errorf("key_masked = %v", obj["key_masked"])
	}
	if obj["config_file"] != filepath.Join(dir, "auth.yaml") {
		t.Errorf("config_file = %v", obj["config_file"])
	}
	if strings.Contains(out, "abcd1234secret") {
		t.Error("the full key must not be printed")
	}
}

func TestAuthStatusUsesEnvOverride(t *testing.T) {
	authEnv(t)
	t.Setenv("TBA_AUTH_KEY", "envkey-value")
	out, _, err := runCmd(t, nil, "auth", "status", "--format", "table")
	requireNoError(t, err, "")
	requireContains(t, out, "Authenticated with key:")
	requireContains(t, out, "********alue")
}

func TestAuthStatusWithAShortKeyDoesNotPanic(t *testing.T) {
	authEnv(t)
	t.Setenv("TBA_AUTH_KEY", "abc")
	out, _, err := runCmd(t, nil, "auth", "status", "--format", "table")
	requireNoError(t, err, "")
	requireContains(t, out, "Authenticated with key:")
	requireContains(t, out, "***")
	if strings.Contains(out, "abc\n") {
		t.Error("a short key must not be printed in full")
	}
}

func TestAuthLogout(t *testing.T) {
	authEnv(t)
	srv := authServer(t)
	if _, _, err := runCmd(t, srv, "auth", "login", "--key", "abcd1234secret"); err != nil {
		t.Fatalf("login: %v", err)
	}
	stdout, stderr, err := runCmd(t, srv, "auth", "logout")
	requireNoError(t, err, stderr)
	requireContains(t, stderr, "Logged out from "+srv.URL+".")
	if stdout != "" {
		t.Errorf("logout must write nothing to stdout, got:\n%s", stdout)
	}

	if _, err := config.GetAPIKey(srv.URL); err == nil {
		t.Error("key should be gone after logout")
	}
}

func TestAuthLogoutWhenNotAuthenticatedFails(t *testing.T) {
	authEnv(t)
	_, _, err := runCmd(t, nil, "auth", "logout")
	if err == nil {
		t.Fatal("logging out with nothing stored should fail")
	}
	// The same state `auth status` reports, so the same exit code.
	if got := clierr.ExitCode(err); got != clierr.ExitAuth {
		t.Errorf("exit code = %d, want %d", got, clierr.ExitAuth)
	}
	requireErrorContains(t, err, "not authenticated")
}

func TestAuthLogoutForAnUnknownBaseURLExitsFour(t *testing.T) {
	authEnv(t)
	srv := authServer(t)
	_, _, err := runCmd(t, srv, "auth", "login", "--key", "abcd1234")
	requireNoError(t, err, "")

	_, _, err = runCmd(t, nil, "auth", "logout", "--base-url", "http://elsewhere.example/api/v3")
	if err == nil {
		t.Fatal("logging out of a URL with no key should fail")
	}
	if got := clierr.ExitCode(err); got != clierr.ExitAuth {
		t.Errorf("exit code = %d, want %d", got, clierr.ExitAuth)
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

func TestAuthSubcommandsAreRegistered(t *testing.T) {
	got := subcommandNames(t, "auth")
	for _, want := range []string{"login", "status", "logout"} {
		if !contains(got, want) {
			t.Errorf("auth %s is not registered (have %v)", want, got)
		}
	}
}
