package cmd

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/spf13/cobra"
	"github.com/the-blue-alliance/tba-cli/internal/api"
	"github.com/the-blue-alliance/tba-cli/internal/clierr"
	"github.com/the-blue-alliance/tba-cli/internal/output"
)

func getBaseURL(cmd *cobra.Command) string {
	baseURL := settings(cmd).String("base-url")
	if baseURL == "" {
		return api.DefaultBaseURL
	}
	return baseURL
}

func newClient(cmd *cobra.Command) (*api.Client, error) {
	s := settings(cmd)
	offline := s.Bool("offline")
	noCache := s.Bool("no-cache")
	if offline && noCache {
		return nil, clierr.Usage("--offline and --no-cache contradict each other: offline mode has only the cache to serve from")
	}

	opts := []api.Option{
		api.WithTimeout(s.Duration("timeout")),
		api.WithRetries(s.Int("retries")),
	}
	opts = append(opts, api.WithOffline(offline), api.WithNotifier(func(note string) {
		fmt.Fprintln(cmd.ErrOrStderr(), note)
	}))
	c, err := api.NewClient(getBaseURL(cmd), opts...)
	if err != nil {
		return nil, err
	}
	if noCache {
		c.SetUseCache(false)
	}
	return c, nil
}

// validFormats lists every accepted --format value, in help-text order.
const validFormats = "auto, table, json, csv, tsv, markdown"

// resolveFormat returns one of: table, json, csv, tsv, markdown.
//
// "auto" (the default when --format is not given) means table on a TTY and
// json otherwise. --json and --jq select JSON, but combining either with an
// explicit non-JSON --format is an error rather than a silent override.
func resolveFormat(cmd *cobra.Command) (string, error) {
	s := settings(cmd)
	jsonFlag := s.Bool("json")
	jqFlag := s.String("jq")

	// The value is checked before the terminal rule is applied, so that a
	// misspelling in the config file is reported rather than quietly ignored
	// on the invocations that would not have used it anyway.
	if _, ok := normalizeFormat(s.String("format")); !ok {
		return "", clierr.Usage("invalid %s %q (want: %s)", s.origin("format"), s.String("format"), validFormats)
	}
	raw := s.Format(cmd.OutOrStdout())
	explicit, _ := normalizeFormat(raw)

	if explicit != "" && explicit != "json" {
		// Say where the format came from when it was not typed on this command
		// line, so that "drop --format table" is not advice about a flag the
		// user never passed.
		chosen := "--format " + raw
		if s.Source("format") != sourceFlag {
			chosen = fmt.Sprintf("--format %s from %s", raw, s.origin("format"))
		}
		if jqFlag != "" {
			return "", clierr.Usage("--jq requires JSON output; drop %s or use --format json", chosen)
		}
		if jsonFlag {
			return "", clierr.Usage("--json requires JSON output; drop %s or use --format json", chosen)
		}
	}
	if explicit != "" {
		return explicit, nil
	}
	if jsonFlag || jqFlag != "" {
		return "json", nil
	}
	if !output.IsTTY(cmd.OutOrStdout()) {
		return "json", nil
	}
	return "table", nil
}

// normalizeFormat maps a --format value to the format it selects, or to "" for
// the values that mean "decide from the TTY". The second result is false for a
// value that is not a format at all.
func normalizeFormat(raw string) (string, bool) {
	switch raw {
	case "", "auto":
		return "", true
	case "table", "json", "csv", "tsv", "markdown":
		return raw, true
	case "md":
		return "markdown", true
	default:
		return "", false
	}
}

// colorMode reads the user's color preference.
//
// no-color is a spelling of color=never and is checked first, so it wins
// whenever it is true, whichever layer either setting came from: a script can
// set it unconditionally without knowing what color says. `tba config list`
// documents the same rule.
//
// A bad color value is still reported when no-color settles the question, so
// that a typo in the config file is not hidden by an unrelated setting.
func colorMode(cmd *cobra.Command) (output.ColorMode, error) {
	s := settings(cmd)
	mode, err := output.ParseColorMode(s.String("color"))
	if err != nil {
		// Like every other bad flag value, this is exit 2.
		return mode, clierr.Wrap(clierr.KindUsage, err)
	}
	if s.Bool("no-color") {
		return output.ColorNever, nil
	}
	return mode, nil
}

func jqExpr(cmd *cobra.Command) string {
	return settings(cmd).String("jq")
}

// checkJQ rejects a --jq expression that is not a jq program.
//
// It runs before the command does any work, so a typo costs no request, and it
// is a usage error: an expression that does not parse is a mistake in the
// command line. An expression that parses but then fails on the data is a
// failure of the run, and keeps exit 1.
func checkJQ(cmd *cobra.Command) error {
	if err := output.ValidateJQ(jqExpr(cmd)); err != nil {
		return clierr.Wrap(clierr.KindUsage, err)
	}
	return nil
}

// rawOutput reports whether --raw-output was given: jq string results are
// printed without their quotes, as jq -r does.
func rawOutput(cmd *cobra.Command) bool {
	return settings(cmd).Bool("raw-output")
}

// outputData routes opaque/key-value output: human renderer for table, JSON otherwise.
// Tabular formats (csv/tsv/markdown) on key-value data fall back to JSON.
func outputData(cmd *cobra.Command, data interface{}, humanFn func()) error {
	format, err := resolveFormat(cmd)
	if err != nil {
		return err
	}
	if _, err := colorMode(cmd); err != nil {
		return err
	}
	switch format {
	case "table":
		humanFn()
		return nil
	default:
		return output.PrintJSONWithFilter(cmd.OutOrStdout(), data, jqExpr(cmd), rawOutput(cmd))
	}
}

// outputTable routes tabular output through the chosen format. Tabular formats
// share one Table so that column selection, sorting and width handling behave
// identically no matter which renderer prints it.
func outputTable(cmd *cobra.Command, data interface{}, headers []string, rows [][]string) error {
	return outputTableWithEmptyNote(cmd, data, headers, rows, "")
}

// outputTableWithEmptyNote is outputTable with something to say when there is
// nothing to show.
//
// A bare header row does not distinguish "this team played no matches at that
// event" from "your filters cancelled each other out" or "you typed the wrong
// event key". note says which, in one line on stderr — never on stdout, so a
// pipeline still sees an empty table — and only in the human-facing formats.
// JSON keeps printing [], which is the answer a script wants.
//
// note reads as the completion of "no ...": "no matches for team 9999 at
// 2024cthar", "no events match those filters".
//
// TODO: the listings owned by other in-flight branches (event matches, event
// rankings, event awards, team matches, team search, insights) should pass a
// note too.
func outputTableWithEmptyNote(cmd *cobra.Command, data interface{}, headers []string, rows [][]string, note string) error {
	format, err := resolveFormat(cmd)
	if err != nil {
		return err
	}
	// The presentation flags are validated even when the chosen format ignores
	// them, so a typo is reported rather than silently doing nothing.
	color, err := colorMode(cmd)
	if err != nil {
		return err
	}
	w := cmd.OutOrStdout()
	table := output.Table{Headers: headers, Rows: rows}

	// Sorting runs before column selection so that a table can be ordered by a
	// column the user chose not to display.
	// A bad --sort or --columns is a mistake in the invocation, not a failure
	// of the work, so it exits 2 like any other flag error.
	sortSpec := settings(cmd).String("sort")
	var order []int
	if sortSpec != "" {
		if order, err = table.SortOrder(sortSpec); err != nil {
			return clierr.Wrap(clierr.KindUsage, err)
		}
		table = table.Reorder(order)
	}

	columns := settings(cmd).String("columns")
	if format == "json" {
		if columns != "" {
			return clierr.Usage("--columns applies to tabular formats; use --jq to shape JSON")
		}
		// --sort is about the order of the result, not its shape, so it also
		// reorders the JSON array the table was built from. When the payload is
		// not that array — an object keyed by team, a document with the rows
		// nested inside — there is no order to apply, and saying so beats
		// printing an unsorted answer to a command that asked for a sorted one.
		if sortSpec != "" && !output.CanPermute(data, order) {
			return clierr.Usage("--sort cannot reorder this JSON payload (it is not a list of rows); " +
				"use --jq to sort it, or drop --format json")
		}
		return output.PrintJSONWithFilter(w, output.PermuteSlice(data, order), jqExpr(cmd), rawOutput(cmd))
	}
	if columns != "" {
		if table, err = table.SelectColumns(columns); err != nil {
			return clierr.Wrap(clierr.KindUsage, err)
		}
	}

	noHeaders := settings(cmd).Bool("no-headers")
	if err := output.Render(w, table, output.RenderOptions{
		Format:    format,
		NoHeaders: noHeaders,
		Color:     color,
	}); err != nil {
		return err
	}
	if len(rows) == 0 && note != "" {
		fmt.Fprintf(cmd.ErrOrStderr(), "note: %s\n", note)
	}
	return nil
}

// exactArgs is cobra.ExactArgs with an error a person can act on.
//
// Cobra's own text is "accepts 1 arg(s), received 0", which says nothing about
// what the missing argument is. what names it and shows one, e.g.
// exactArgs(1, "a team number (e.g. tba team view 177)").
func exactArgs(n int, what string) cobra.PositionalArgs {
	return func(cmd *cobra.Command, args []string) error {
		if len(args) == n {
			return nil
		}
		// The root's name is already on every example, so the complaint reads
		// as "team view needs ...", not "tba team view needs ...".
		name := strings.TrimPrefix(cmd.CommandPath(), cmd.Root().Name()+" ")
		if len(args) < n {
			return clierr.Usage("%s needs %s", name, what)
		}
		return clierr.Usage("%s takes %s, but got %d arguments", name, what, len(args))
	}
}

// teamKey normalises a team argument, so that both "177" and "frc177" (in any
// case) address the same team.
//
// It does not judge what it is given: "17x7" comes back as "frc17x7" and costs
// a request and a raw 404.
//
// TODO: the call sites in cmd/matches.go, cmd/team_next.go,
// cmd/team_standing.go and cmd/team_search.go should call validateTeamArg on
// the argument first, the way the ones in cmd/team.go, cmd/open.go and
// cmd/eventfilter.go do, so that a typo is a usage error instead.
func teamKey(arg string) string {
	return "frc" + teamNumberOf(arg)
}

// teamNumberOf strips an optional "frc" prefix, in any case, and returns what
// is left. It never panics and never rejects anything; validateTeamArg is what
// decides whether the remainder is a team number.
func teamNumberOf(arg string) string {
	trimmed := strings.TrimSpace(arg)
	if len(trimmed) >= 3 && strings.EqualFold(trimmed[:3], "frc") {
		trimmed = trimmed[3:]
	}
	return trimmed
}

// teamArgPattern is a team number as a person writes it: 1 to 5 digits, with
// an optional "frc" prefix in any case. Five digits covers every team number
// FIRST has issued and leaves room for the ones it has not.
var teamArgPattern = regexp.MustCompile(`^(?i:frc)?[0-9]{1,5}$`)

// validateTeamArg rejects something that is not a team number before any HTTP
// call, so that a typo comes back as a usage error instead of a raw 404 body.
func validateTeamArg(arg string) error {
	if !teamArgPattern.MatchString(strings.TrimSpace(arg)) {
		return clierr.Usage("%q is not a team number (expected something like 177 or frc177)", arg)
	}
	return nil
}

var (
	eventKeyPattern = regexp.MustCompile(`^[0-9]{4}[a-z0-9]+$`)
	// Qualification keys have no set number ("2024cthar_qm12"); playoff keys
	// do ("2024cthar_sf3m1").
	matchKeyPattern = regexp.MustCompile(`^[0-9]{4}[a-z0-9]+_(qm|ef|qf|sf|f)[0-9]+(m[0-9]+)?$`)
)

// validateEventKey rejects a malformed event key before any HTTP call, so a
// typo comes back as a usage error instead of a 404.
func validateEventKey(arg string) error {
	if !eventKeyPattern.MatchString(arg) {
		return clierr.Usage("%q is not a valid event key (expected something like 2024cthar)", arg)
	}
	return nil
}

// validateMatchKey rejects a malformed match key before any HTTP call.
func validateMatchKey(arg string) error {
	if !matchKeyPattern.MatchString(arg) {
		return clierr.Usage("%q is not a valid match key (expected something like 2024cthar_qm12)", arg)
	}
	return nil
}
