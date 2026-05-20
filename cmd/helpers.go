package cmd

import (
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
	return api.NewClient(getBaseURL(cmd))
}

func wantJSON(cmd *cobra.Command) bool {
	jsonFlag, _ := cmd.Flags().GetBool("json")
	jqFlag, _ := cmd.Flags().GetString("jq")
	return jsonFlag || jqFlag != "" || !output.IsTTY()
}

func jqExpr(cmd *cobra.Command) string {
	jqFlag, _ := cmd.Flags().GetString("jq")
	return jqFlag
}

func outputData(cmd *cobra.Command, data interface{}, humanFn func()) error {
	if wantJSON(cmd) {
		return output.PrintJSONWithFilter(data, jqExpr(cmd))
	}
	humanFn()
	return nil
}

func currentYear() int {
	return time.Now().Year()
}
