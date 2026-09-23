package cmd

import (
	"fmt"
	"os/exec"
	"regexp"
	"runtime"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
	"github.com/the-blue-alliance/tba-cli/internal/clierr"
	"github.com/the-blue-alliance/tba-cli/internal/output"
)

// tbaWebBaseURL is the website, not the API: `open` deals in pages a person
// reads, so it never touches api/v3 and never honours --base-url.
const tbaWebBaseURL = "https://www.thebluealliance.com"

// openURL hands a URL to the desktop's browser. It is a package variable so
// that tests can replace it and never actually launch anything.
var openURL = launchBrowser

func launchBrowser(url string) error {
	var c *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		c = exec.Command("open", url)
	case "windows":
		c = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	default:
		c = exec.Command("xdg-open", url)
	}
	if err := c.Run(); err != nil {
		return fmt.Errorf("could not open a browser (%w); use --print to get the URL instead", err)
	}
	return nil
}

// teamTargetish matches an argument that can only have been meant as a team:
// digits, or anything at all behind an "frc" prefix. Whether it really is a
// team number is validateTeamArg's decision, so that "frc17x7" is told it is
// not a team number rather than that it is not anything at all.
var teamTargetish = regexp.MustCompile(`^(?i:frc.*|[0-9]+)$`)

// webPathFor turns an `open` argument into a path on thebluealliance.com.
//
// A team is tried first: a bare number can never be an event key, while a
// number long enough to look like a year ("20241") would otherwise be
// ambiguous. year is appended only for teams, and only when it was asked for.
func webPathFor(target string, year int) (string, error) {
	trimmed := strings.TrimSpace(target)
	if teamTargetish.MatchString(trimmed) {
		if err := validateTeamArg(trimmed); err != nil {
			return "", err
		}
		number, err := strconv.Atoi(teamNumberOf(trimmed))
		if err == nil {
			if year > 0 {
				return fmt.Sprintf("/team/%d/%d", number, year), nil
			}
			return fmt.Sprintf("/team/%d", number), nil
		}
	}
	// Keys are lowercase on TBA, so a pasted "2024CTHAR" still resolves.
	key := strings.ToLower(trimmed)
	if matchKeyPattern.MatchString(key) {
		return "/match/" + key, nil
	}
	if eventKeyPattern.MatchString(key) {
		return "/event/" + key, nil
	}
	return "", clierr.Usage("cannot tell what %q refers to; expected a team number (177 or frc177), "+
		"an event key (2024cthar) or a match key (2024cthar_qm12)", target)
}

func newOpenCmd() *cobra.Command {
	c := &cobra.Command{
		Use:     "open <team|event|match>",
		Aliases: []string{"browse"},
		Short:   "Open a team, event or match on thebluealliance.com",
		Long: `Open the web page for a team, an event or a match.

The argument is whatever you already have to hand: a team number (177 or
frc177), an event key (2024cthar) or a match key (2024cthar_qm12). --year
picks a team's season page.

Nothing is fetched from the API, so this works without a key and without a
network round trip. --print writes the URL to stdout instead of opening it,
which is what you want in a script or over ssh.`,
		Example: `  tba open 177
  tba open 177 --year 2024
  tba open 2024cthar
  tba open 2024cthar_qm12 --print`,
		Args: exactArgs(1, "a team number, an event key or a match key (e.g. tba open 2024cthar)"),
		RunE: func(cmd *cobra.Command, args []string) error {
			year, _ := cmd.Flags().GetInt("year")
			path, err := webPathFor(args[0], year)
			if err != nil {
				return err
			}
			url := tbaWebBaseURL + path

			if printOnly, _ := cmd.Flags().GetBool("print"); printOnly {
				return printURL(cmd, url)
			}
			// The URL goes to stderr: opening a page produces no data, and a
			// caller that wants the URL asked for --print.
			fmt.Fprintf(cmd.ErrOrStderr(), "opening %s\n", url)
			if err := openURL(url); err != nil {
				return err
			}
			return nil
		},
	}
	c.Flags().Int("year", 0, "Season year for a team page (default: the team's overview)")
	c.Flags().BoolP("print", "n", false, "Print the URL instead of opening it")
	return c
}

// printURL writes a URL to stdout.
//
// Unlike the commands that print records, a URL stays a bare URL when it is
// piped: it is already the machine-readable form, and `tba open X -n | xargs
// curl` should not have to unwrap an object. JSON is produced only when it is
// actually asked for.
func printURL(cmd *cobra.Command, url string) error {
	format, err := resolveFormat(cmd)
	if err != nil {
		return err
	}
	asked := cmd.Flags().Changed("json") || cmd.Flags().Changed("jq") || cmd.Flags().Changed("format")
	if format == "json" && asked {
		return output.PrintJSONWithFilter(cmd.OutOrStdout(), map[string]string{"url": url}, jqExpr(cmd), rawOutput(cmd))
	}
	fmt.Fprintln(cmd.OutOrStdout(), url)
	return nil
}
