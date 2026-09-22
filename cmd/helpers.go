package cmd

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"
	"github.com/the-blue-alliance/tba-cli/internal/api"
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
	c, err := api.NewClient(getBaseURL(cmd))
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
		return "", fmt.Errorf("invalid --format %q (want: %s)", raw, validFormats)
	}

	if explicit != "" && explicit != "json" {
		if jqFlag != "" {
			return "", fmt.Errorf("--jq requires JSON output; drop --format %s or use --format json", raw)
		}
		if jsonFlag {
			return "", fmt.Errorf("--json requires JSON output; drop --format %s or use --format json", raw)
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
	switch format {
	case "table":
		humanFn()
		return nil
	default:
		return output.PrintJSONWithFilter(cmd.OutOrStdout(), data, jqExpr(cmd), rawOutput(cmd))
	}
}

// outputTable routes tabular output through the chosen format.
func outputTable(cmd *cobra.Command, data interface{}, headers []string, rows [][]string) error {
	format, err := resolveFormat(cmd)
	if err != nil {
		return err
	}
	w := cmd.OutOrStdout()
	switch format {
	case "json":
		return output.PrintJSONWithFilter(w, data, jqExpr(cmd), rawOutput(cmd))
	case "csv":
		return output.PrintDelimited(w, headers, rows, ',')
	case "tsv":
		return output.PrintDelimited(w, headers, rows, '\t')
	case "markdown":
		output.PrintMarkdownTable(w, headers, rows)
		return nil
	default:
		output.PrintTable(w, headers, rows)
		return nil
	}
}

func currentYear() int {
	return time.Now().Year()
}
