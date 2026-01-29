package cmd

import (
	"github.com/spf13/cobra"
	"github.com/the-blue-alliance/tba-cli/internal/output"
)

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
	return 2025 // Could be dynamic, but keeping simple
}
