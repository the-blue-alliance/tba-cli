package cmd

import (
	"errors"
	"strings"
	"testing"

	"github.com/the-blue-alliance/tba-cli/internal/clierr"
)

// stubOpener replaces the browser launcher for the duration of a test and
// returns a pointer to whatever URL the command tried to open.
func stubOpener(t *testing.T, err error) *string {
	t.Helper()
	var opened string
	previous := openURL
	openURL = func(url string) error {
		opened = url
		return err
	}
	t.Cleanup(func() { openURL = previous })
	return &opened
}

func openedURL(t *testing.T, args ...string) string {
	t.Helper()
	full := append([]string{"open"}, args...)
	full = append(full, "--print")
	out, stderr, err := runCmd(t, nil, full...)
	requireNoError(t, err, stderr)
	return strings.TrimSpace(out)
}

func TestOpenPrintsURLs(t *testing.T) {
	cases := []struct {
		name string
		args []string
		want string
	}{
		{"team number", []string{"177"}, "https://www.thebluealliance.com/team/177"},
		{"team with frc prefix", []string{"frc177"}, "https://www.thebluealliance.com/team/177"},
		{"team with FRC prefix", []string{"FRC177"}, "https://www.thebluealliance.com/team/177"},
		{"team with year", []string{"177", "--year", "2024"}, "https://www.thebluealliance.com/team/177/2024"},
		{"event key", []string{"2024cthar"}, "https://www.thebluealliance.com/event/2024cthar"},
		{"uppercase event key", []string{"2024CTHAR"}, "https://www.thebluealliance.com/event/2024cthar"},
		{"qual match key", []string{"2024cthar_qm1"}, "https://www.thebluealliance.com/match/2024cthar_qm1"},
		{"playoff match key", []string{"2024cthar_sf3m1"}, "https://www.thebluealliance.com/match/2024cthar_sf3m1"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := openedURL(t, tc.args...); got != tc.want {
				t.Errorf("url = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestOpenYearOnlyAppliesToTeams(t *testing.T) {
	if got := openedURL(t, "2024cthar", "--year", "2024"); got != "https://www.thebluealliance.com/event/2024cthar" {
		t.Errorf("url = %q, want the event page unchanged by --year", got)
	}
}

func TestOpenPrefersTeamOverEventForBareNumbers(t *testing.T) {
	// "20241" is five digits, so it would also satisfy the event key shape.
	if got := openedURL(t, "20241"); got != "https://www.thebluealliance.com/team/20241" {
		t.Errorf("url = %q, want a team page", got)
	}
}

func TestOpenShortPrintFlag(t *testing.T) {
	out, stderr, err := runCmd(t, nil, "open", "177", "-n")
	requireNoError(t, err, stderr)
	if strings.TrimSpace(out) != "https://www.thebluealliance.com/team/177" {
		t.Errorf("stdout = %q", out)
	}
}

func TestOpenPrintDoesNotLaunchABrowser(t *testing.T) {
	opened := stubOpener(t, nil)
	_ = openedURL(t, "177")
	if *opened != "" {
		t.Errorf("--print launched a browser for %q", *opened)
	}
}

func TestOpenLaunchesTheBrowser(t *testing.T) {
	opened := stubOpener(t, nil)
	out, stderr, err := runCmd(t, nil, "open", "2024cthar")
	requireNoError(t, err, stderr)

	if *opened != "https://www.thebluealliance.com/event/2024cthar" {
		t.Errorf("opened %q", *opened)
	}
	if out != "" {
		t.Errorf("stdout = %q, want nothing on stdout", out)
	}
	requireContains(t, stderr, "opening https://www.thebluealliance.com/event/2024cthar")
}

func TestOpenReportsABrowserFailure(t *testing.T) {
	stubOpener(t, errors.New("no browser here"))
	_, _, err := runCmd(t, nil, "open", "177")
	requireErrorContains(t, err, "no browser here")
	if got := clierr.ExitCode(err); got != clierr.ExitFailure {
		t.Errorf("exit code = %d, want %d", got, clierr.ExitFailure)
	}
}

func TestOpenRejectsAnUnrecognizedTarget(t *testing.T) {
	for _, target := range []string{"bobcats", "2024cthar_", "http://example.com", ""} {
		t.Run(target, func(t *testing.T) {
			_, _, err := runCmd(t, nil, "open", target)
			if got := clierr.ExitCode(err); got != clierr.ExitUsage {
				t.Fatalf("exit code for %q = %d (err %v), want %d", target, got, err, clierr.ExitUsage)
			}
			requireErrorContains(t, err, "cannot tell what")
		})
	}
}

// A target that was plainly meant as a team is told that it is not a team
// number, rather than that it is not anything at all.
func TestOpenRejectsABadTeamTarget(t *testing.T) {
	for _, target := range []string{"frc17x7", "1771771"} {
		t.Run(target, func(t *testing.T) {
			_, _, err := runCmd(t, nil, "open", target)
			if got := clierr.ExitCode(err); got != clierr.ExitUsage {
				t.Fatalf("exit code for %q = %d (err %v), want %d", target, got, err, clierr.ExitUsage)
			}
			requireErrorContains(t, err, "is not a team number")
		})
	}
}

func TestOpenNeedsExactlyOneTarget(t *testing.T) {
	_, _, err := runCmd(t, nil, "open")
	if got := clierr.ExitCode(err); got != clierr.ExitUsage {
		t.Errorf("exit code = %d, want %d", got, clierr.ExitUsage)
	}
}

func TestOpenMakesNoAPIRequests(t *testing.T) {
	srv := newFakeTBA(t, map[string]any{"/status": apiStatusJSON})
	stubOpener(t, nil)
	_, stderr, err := runCmd(t, srv, "open", "2024cthar")
	requireNoError(t, err, stderr)
	if got := requestPaths(t, srv); len(got) != 0 {
		t.Errorf("open requested %v, want nothing", got)
	}
}

func TestOpenWorksWithoutAnAPIKey(t *testing.T) {
	t.Setenv("TBA_AUTH_KEY", "")
	t.Setenv("TBA_CONFIG_DIR", t.TempDir())
	out, stderr, err := runCmd(t, nil, "open", "177", "--print")
	requireNoError(t, err, stderr)
	requireContains(t, out, "/team/177")
}

func TestOpenPrintStaysABareURLWhenPiped(t *testing.T) {
	// stdout is a buffer here, so --format auto resolves to json for the
	// commands that print records. A URL is already machine-readable.
	out, stderr, err := runCmd(t, nil, "open", "177", "--print")
	requireNoError(t, err, stderr)
	if out != "https://www.thebluealliance.com/team/177\n" {
		t.Errorf("stdout = %q", out)
	}
}

func TestOpenPrintAsJSON(t *testing.T) {
	out, stderr, err := runCmd(t, nil, "open", "177", "--print", "--json")
	requireNoError(t, err, stderr)
	obj := decodeJSON(t, out).(map[string]any)
	if obj["url"] != "https://www.thebluealliance.com/team/177" {
		t.Errorf("url = %v", obj["url"])
	}
}

func TestOpenPrintWithJq(t *testing.T) {
	out, stderr, err := runCmd(t, nil, "open", "2024cthar", "--print", "--jq", ".url", "-r")
	requireNoError(t, err, stderr)
	if strings.TrimSpace(out) != "https://www.thebluealliance.com/event/2024cthar" {
		t.Errorf("stdout = %q", out)
	}
}

func TestOpenIsRegistered(t *testing.T) {
	var found bool
	for _, c := range NewRootCmd().Commands() {
		if c.Name() == "open" {
			found = true
		}
	}
	if !found {
		t.Error("tba open is not registered")
	}
}
