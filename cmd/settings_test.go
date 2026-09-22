package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/the-blue-alliance/tba-cli/internal/clierr"
)

// writeConfig points the next runCmd at a config directory holding body, and
// returns the path of the config file.
func writeConfig(t *testing.T, body string) string {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("TBA_CONFIG_DIR", dir)
	path := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(path, []byte(body), 0600); err != nil {
		t.Fatalf("writing config file: %v", err)
	}
	return path
}

// readConfig returns the config file the last command wrote.
func readConfig(t *testing.T) string {
	t.Helper()
	path := filepath.Join(os.Getenv("TBA_CONFIG_DIR"), "config.yaml")
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading config file: %v", err)
	}
	return string(b)
}

// emptyConfigDir gives the test a config directory of its own with no file in
// it, so that `config set` has somewhere to write.
func emptyConfigDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("TBA_CONFIG_DIR", dir)
	return dir
}

// devNull is a character device, which is as close to a terminal as a test
// can get without one.
func devNull(t *testing.T) *os.File {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("no character-device stand-in for a terminal on Windows")
	}
	f, err := os.Open(os.DevNull)
	if err != nil {
		t.Fatalf("opening %s: %v", os.DevNull, err)
	}
	t.Cleanup(func() { f.Close() })
	return f
}

func TestConfigFileSuppliesADefault(t *testing.T) {
	writeConfig(t, "retries: 0\n")
	srv := newFakeTBA(t, map[string]any{"/status": apiStatusJSON})
	setStatus(t, srv, "/status", 500)

	_, _, err := runCmd(t, srv, "status")
	if err == nil {
		t.Fatal("want the 500 to surface")
	}
	if n := len(requestPaths(t, srv)); n != 1 {
		t.Errorf("made %d requests with retries: 0 in the config file, want 1", n)
	}
}

func TestEnvironmentBeatsTheConfigFile(t *testing.T) {
	writeConfig(t, "retries: 3\n")
	t.Setenv("TBA_RETRIES", "0")
	srv := newFakeTBA(t, map[string]any{"/status": apiStatusJSON})
	setStatus(t, srv, "/status", 500)

	if _, _, err := runCmd(t, srv, "status"); err == nil {
		t.Fatal("want the 500 to surface")
	}
	if n := len(requestPaths(t, srv)); n != 1 {
		t.Errorf("made %d requests with TBA_RETRIES=0, want 1", n)
	}
}

func TestFlagBeatsTheEnvironmentAndTheConfigFile(t *testing.T) {
	writeConfig(t, "retries: 3\n")
	t.Setenv("TBA_RETRIES", "3")
	srv := newFakeTBA(t, map[string]any{"/status": apiStatusJSON})
	setStatus(t, srv, "/status", 500)

	if _, _, err := runCmd(t, srv, "status", "--retries", "0"); err == nil {
		t.Fatal("want the 500 to surface")
	}
	if n := len(requestPaths(t, srv)); n != 1 {
		t.Errorf("made %d requests with --retries 0, want 1", n)
	}
}

func TestBaseURLComesFromTheConfigFile(t *testing.T) {
	srv := newFakeTBA(t, map[string]any{"/status": apiStatusJSON})
	writeConfig(t, "base-url: "+srv.URL+"\n")

	// srv is not passed, so nothing adds --base-url: the file is the only
	// thing that can point the client at the fake.
	_, _, err := runCmd(t, nil, "status")
	requireNoError(t, err, "")
	if got := requestPaths(t, srv); len(got) != 1 || got[0] != "/status" {
		t.Errorf("requested %v, want [/status]", got)
	}
}

func TestBaseURLComesFromTheEnvironment(t *testing.T) {
	srv := newFakeTBA(t, map[string]any{"/status": apiStatusJSON})
	emptyConfigDir(t)
	t.Setenv("TBA_BASE_URL", srv.URL)

	_, _, err := runCmd(t, nil, "status")
	requireNoError(t, err, "")
	if got := requestPaths(t, srv); len(got) != 1 {
		t.Errorf("requested %v, want one request", got)
	}
}

func TestTimeoutComesFromTheConfigFile(t *testing.T) {
	srv := newFakeTBA(t, map[string]any{"/status": apiStatusJSON})
	setDelay(t, srv, "/status", 500*time.Millisecond)
	writeConfig(t, "timeout: 20ms\nretries: 0\n")

	_, _, err := runCmd(t, srv, "status")
	if err == nil {
		t.Fatal("want a timeout error from the configured timeout")
	}
}

func TestNoCacheComesFromTheConfigFile(t *testing.T) {
	t.Setenv("TBA_CACHE_DIR", t.TempDir())
	writeConfig(t, "no-cache: true\n")
	srv := newFakeTBA(t, map[string]any{"/status": apiStatusJSON})
	setETag(t, srv, "/status", `"v1"`)

	for i := 0; i < 2; i++ {
		if _, _, err := runCmd(t, srv, "status"); err != nil {
			t.Fatalf("status: %v", err)
		}
	}
	for _, r := range requestsTo(t, srv) {
		if r.Headers.Get("If-None-Match") != "" {
			t.Error("no-cache: true in the config file still revalidated from the cache")
		}
	}
}

func TestColorComesFromTheConfigFile(t *testing.T) {
	writeConfig(t, "color: always\n")

	out := districtsCmd(t, "--format", "table")
	if !strings.Contains(out, "\x1b[") {
		t.Errorf("color: always in the config file did not colorize:\n%q", out)
	}
}

func TestInvalidConfigValueNamesTheConfigFile(t *testing.T) {
	path := writeConfig(t, "format: xml\n")
	srv := newFakeTBA(t, map[string]any{"/status": apiStatusJSON})

	_, _, err := runCmd(t, srv, "status")
	requireErrorContains(t, err, path)
	if got := clierr.ExitCode(err); got != clierr.ExitUsage {
		t.Errorf("exit code = %d, want %d", got, clierr.ExitUsage)
	}
}

func TestUnknownConfigKeyWarnsOnStderr(t *testing.T) {
	path := writeConfig(t, "format: json\nwidgets: 3\n")
	srv := newFakeTBA(t, map[string]any{"/status": apiStatusJSON})

	stdout, stderr, err := runCmd(t, srv, "status")
	requireNoError(t, err, stderr)
	requireContains(t, stderr, `warning: unknown key "widgets"`)
	requireContains(t, stderr, path)
	// The command still ran, and the warning stayed off stdout.
	decodeJSON(t, stdout)
	if strings.Contains(stdout, "widgets") {
		t.Error("the warning leaked into stdout")
	}
}

func TestMalformedConfigFileIsReported(t *testing.T) {
	path := writeConfig(t, "format: [unterminated\n")
	srv := newFakeTBA(t, map[string]any{"/status": apiStatusJSON})

	_, _, err := runCmd(t, srv, "status")
	requireErrorContains(t, err, path)
	if n := len(requestPaths(t, srv)); n != 0 {
		t.Errorf("made %d requests despite an unreadable config", n)
	}
}

// The format rule: a file value is a preference for what you see, not a
// promise about what a pipe receives.

func TestFormatFromTheConfigFileIsIgnoredOffATerminal(t *testing.T) {
	writeConfig(t, "format: table\n")
	srv := newFakeTBA(t, map[string]any{"/team/frc177": teamFRC177JSON})

	out, _, err := runCmd(t, srv, "team", "view", "177")
	requireNoError(t, err, "")
	decodeJSON(t, out) // piped output stays JSON
}

func TestFormatFromTheConfigFileAppliesOnATerminal(t *testing.T) {
	writeConfig(t, "format: table\n")
	s, err := newSettings(NewRootCmd())
	if err != nil {
		t.Fatalf("newSettings: %v", err)
	}

	if got := s.Format(&bytes.Buffer{}); got != "" {
		t.Errorf("Format(pipe) = %q, want the file value to be ignored", got)
	}
	if got := s.Format(devNull(t)); got != "table" {
		t.Errorf("Format(terminal) = %q, want %q", got, "table")
	}
}

func TestAConflictingFormatSaysWhereItCameFrom(t *testing.T) {
	emptyConfigDir(t)
	t.Setenv("TBA_FORMAT", "table")
	srv := newFakeTBA(t, map[string]any{"/team/frc177": teamFRC177JSON})

	_, _, err := runCmd(t, srv, "team", "view", "177", "--jq", ".nickname")
	requireErrorContains(t, err, "TBA_FORMAT")
	if got := clierr.ExitCode(err); got != clierr.ExitUsage {
		t.Errorf("exit code = %d, want %d", got, clierr.ExitUsage)
	}
}

func TestFormatFromTheEnvironmentAppliesEverywhere(t *testing.T) {
	emptyConfigDir(t)
	t.Setenv("TBA_FORMAT", "table")
	srv := newFakeTBA(t, map[string]any{"/team/frc177": teamFRC177JSON})

	out, _, err := runCmd(t, srv, "team", "view", "177")
	requireNoError(t, err, "")
	requireContains(t, out, "Team:")
	if strings.HasPrefix(strings.TrimSpace(out), "{") {
		t.Errorf("TBA_FORMAT=table was ignored:\n%s", out)
	}
}

func TestFormatFlagBeatsTheConfigFile(t *testing.T) {
	writeConfig(t, "format: json\n")
	srv := newFakeTBA(t, map[string]any{"/team/frc177": teamFRC177JSON})

	out, _, err := runCmd(t, srv, "team", "view", "177", "--format", "table")
	requireNoError(t, err, "")
	requireContains(t, out, "Team:")
}

// `tba config set retries -1` was refused while --retries -1 and TBA_RETRIES=-1
// went through without a word, so the same nonsense was an error in one place
// and quietly something else in another.
func TestRetriesAndTimeoutAreValidatedInEveryLayer(t *testing.T) {
	cases := []struct {
		name string
		args []string
		env  [2]string
		file string
		want string
	}{
		{name: "retries flag", args: []string{"--retries", "-1"}, want: "retries cannot be negative"},
		{name: "retries env", env: [2]string{"TBA_RETRIES", "-1"}, want: "retries cannot be negative"},
		{name: "retries config", file: "retries: -1\n", want: "retries cannot be negative"},
		{name: "timeout flag", args: []string{"--timeout", "0s"}, want: "timeout must be positive"},
		{name: "timeout negative flag", args: []string{"--timeout", "-5s"}, want: "timeout must be positive"},
		{name: "timeout env", env: [2]string{"TBA_TIMEOUT", "0s"}, want: "timeout must be positive"},
		{name: "timeout config", file: "timeout: 0s\n", want: "timeout must be positive"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if c.file != "" {
				writeConfig(t, c.file)
			}
			if c.env[0] != "" {
				t.Setenv(c.env[0], c.env[1])
			}
			srv := newFakeTBA(t, map[string]any{"/team/frc177": teamFRC177JSON})

			_, _, err := runCmd(t, srv, append([]string{"team", "view", "177"}, c.args...)...)
			requireErrorContains(t, err, c.want)
			if got := clierr.ExitCode(err); got != clierr.ExitUsage {
				t.Errorf("exit code = %d, want %d", got, clierr.ExitUsage)
			}
			if got := requestPaths(t, srv); len(got) != 0 {
				t.Errorf("a bad setting should be caught before any request, got %v", got)
			}
		})
	}
}

// A value from somewhere other than the command line says where it came from,
// since "--retries" is not a flag the user passed.
func TestABadSettingNamesItsLayer(t *testing.T) {
	t.Setenv("TBA_RETRIES", "-1")
	srv := newFakeTBA(t, map[string]any{"/team/frc177": teamFRC177JSON})

	_, _, err := runCmd(t, srv, "team", "view", "177")
	requireErrorContains(t, err, "retries cannot be negative (from TBA_RETRIES)")
}

func TestValidRetriesAndTimeoutAreLeftAlone(t *testing.T) {
	srv := newFakeTBA(t, map[string]any{"/team/frc177": teamFRC177JSON})

	// 0 retries is the right setting for a cron job; only a negative count is
	// nonsense.
	_, stderr, err := runCmd(t, srv, "team", "view", "177", "--retries", "0", "--timeout", "1s")
	requireNoError(t, err, stderr)
}
