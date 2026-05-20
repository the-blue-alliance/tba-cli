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

// resolveFormat returns one of: table, json, csv, tsv, markdown.
// Precedence: explicit --format > --json / --jq > TTY default.
func resolveFormat(cmd *cobra.Command) (string, error) {
	format, _ := cmd.Flags().GetString("format")
	jsonFlag, _ := cmd.Flags().GetBool("json")
	jqFlag, _ := cmd.Flags().GetString("jq")

	if format != "" {
		switch format {
		case "table", "json", "csv", "tsv", "markdown", "md":
			if format == "md" {
				format = "markdown"
			}
			return format, nil
		default:
			return "", fmt.Errorf("invalid --format %q (want: table, json, csv, tsv, markdown)", format)
		}
	}
	if jsonFlag || jqFlag != "" {
		return "json", nil
	}
	if !output.IsTTY() {
		return "json", nil
	}
	return "table", nil
}

func wantJSON(cmd *cobra.Command) bool {
	f, err := resolveFormat(cmd)
	return err == nil && f == "json"
}

func jqExpr(cmd *cobra.Command) string {
	jqFlag, _ := cmd.Flags().GetString("jq")
	return jqFlag
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
		return output.PrintJSONWithFilter(data, jqExpr(cmd))
	}
}

// outputTable routes tabular output through the chosen format.
func outputTable(cmd *cobra.Command, data interface{}, headers []string, rows [][]string) error {
	format, err := resolveFormat(cmd)
	if err != nil {
		return err
	}
	switch format {
	case "json":
		return output.PrintJSONWithFilter(data, jqExpr(cmd))
	case "csv":
		return output.PrintDelimited(headers, rows, ',')
	case "tsv":
		return output.PrintDelimited(headers, rows, '\t')
	case "markdown":
		output.PrintMarkdownTable(headers, rows)
		return nil
	default:
		output.PrintTable(headers, rows)
		return nil
	}
}

func currentYear() int {
	return time.Now().Year()
}
