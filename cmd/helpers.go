package cmd

import (
	"regexp"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/the-blue-alliance/tba-cli/internal/api"
	"github.com/the-blue-alliance/tba-cli/internal/clierr"
	"github.com/the-blue-alliance/tba-cli/internal/output"
)

func getBaseURL(cmd *cobra.Command) string {
	baseURL, _ := cmd.Flags().GetString("base-url")
	if baseURL == "" {
		return api.DefaultBaseURL
	}
	return baseURL
}

func newClient(cmd *cobra.Command) (*api.Client, error) {
	var opts []api.Option
	if d, err := cmd.Flags().GetDuration("timeout"); err == nil {
		opts = append(opts, api.WithTimeout(d))
	}
	if n, err := cmd.Flags().GetInt("retries"); err == nil {
		opts = append(opts, api.WithRetries(n))
	}
	c, err := api.NewClient(getBaseURL(cmd), opts...)
	if err != nil {
		return nil, err
	}
	if noCache, _ := cmd.Flags().GetBool("no-cache"); noCache {
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
	raw, _ := cmd.Flags().GetString("format")
	jsonFlag, _ := cmd.Flags().GetBool("json")
	jqFlag, _ := cmd.Flags().GetString("jq")

	explicit := ""
	switch raw {
	case "", "auto":
		// Resolved below from the TTY / --json / --jq state.
	case "table", "json", "csv", "tsv", "markdown":
		explicit = raw
	case "md":
		explicit = "markdown"
	default:
		return "", clierr.Usage("invalid --format %q (want: %s)", raw, validFormats)
	}

	if explicit != "" && explicit != "json" {
		if jqFlag != "" {
			return "", clierr.Usage("--jq requires JSON output; drop --format %s or use --format json", raw)
		}
		if jsonFlag {
			return "", clierr.Usage("--json requires JSON output; drop --format %s or use --format json", raw)
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

// colorMode reads the user's color preference. --no-color is a spelling of
// --color=never and wins, so scripts can set it unconditionally.
func colorMode(cmd *cobra.Command) (output.ColorMode, error) {
	if noColor, _ := cmd.Flags().GetBool("no-color"); noColor {
		return output.ColorNever, nil
	}
	mode, _ := cmd.Flags().GetString("color")
	return output.ParseColorMode(mode)
}

func jqExpr(cmd *cobra.Command) string {
	jqFlag, _ := cmd.Flags().GetString("jq")
	return jqFlag
}

// rawOutput reports whether --raw-output was given: jq string results are
// printed without their quotes, as jq -r does.
func rawOutput(cmd *cobra.Command) bool {
	raw, _ := cmd.Flags().GetBool("raw-output")
	return raw
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

	if format == "json" {
		return output.PrintJSONWithFilter(w, data, jqExpr(cmd), rawOutput(cmd))
	}
	return output.Render(w, table, output.RenderOptions{Format: format, Color: color})
}

// teamKey normalises a team argument, so that both "177" and "frc177" (in any
// case) address the same team.
func teamKey(arg string) string {
	trimmed := strings.TrimSpace(arg)
	if len(trimmed) >= 3 && strings.EqualFold(trimmed[:3], "frc") {
		trimmed = trimmed[3:]
	}
	return "frc" + trimmed
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

func currentYear() int {
	return time.Now().Year()
}
