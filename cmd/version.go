package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/the-blue-alliance/tba-cli/internal/version"
)

// versionTemplate makes `tba --version` print exactly what `tba version`
// prints, instead of cobra's "tba version 1.2.3".
const versionTemplate = "{{.Version}}\n"

func newVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Show the tba version",
		Long: "Show the version, build metadata and toolchain of this binary.\n\n" +
			"`tba --version` prints the same line.",
		Example: `  tba version
  tba version --format json
  tba version --jq .commit -r`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			info := version.Resolve()
			return outputData(cmd, info, func() {
				fmt.Fprintln(cmd.OutOrStdout(), info.Line())
			})
		},
	}
}
