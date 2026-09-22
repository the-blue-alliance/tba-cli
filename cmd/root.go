package cmd

import (
	"github.com/spf13/cobra"
	"github.com/the-blue-alliance/tba-cli/internal/clierr"
)

// NewRootCmd builds a fresh `tba` command tree. Every command is constructed
// per call so that flag state is never shared between invocations, which keeps
// tests independent of each other.
func NewRootCmd() *cobra.Command {
	rootCmd := &cobra.Command{
		Use:           "tba",
		Short:         "The Blue Alliance CLI",
		Long:          "A command-line interface for The Blue Alliance API v3.",
		SilenceErrors: true,
		SilenceUsage:  true,
		// Flags are checked once, before any command does work, so that a
		// contradictory --format is reported without first hitting the API.
		PersistentPreRunE: func(cmd *cobra.Command, _ []string) error {
			_, err := resolveFormat(cmd)
			return err
		},
	}

	// Cobra reports a bad flag as a plain error; tag it so main can exit 2.
	rootCmd.SetFlagErrorFunc(func(_ *cobra.Command, err error) error {
		return clierr.Wrap(clierr.KindUsage, err)
	})

	rootCmd.PersistentFlags().Bool("json", false, "Output as JSON (shorthand for --format=json)")
	rootCmd.PersistentFlags().String("jq", "", "Apply jq expression to JSON output")
	rootCmd.PersistentFlags().BoolP("raw-output", "r", false, "With --jq, print string results without quotes (like jq -r)")
	rootCmd.PersistentFlags().String("base-url", "", "Override API base URL (e.g. http://localhost:8080/api/v3)")
	rootCmd.PersistentFlags().Bool("no-cache", false, "Disable HTTP response cache for this invocation")
	rootCmd.PersistentFlags().String("format", "", "Output format: auto, table, json, csv, tsv, markdown (auto: table on TTY, json otherwise)")

	rootCmd.AddCommand(newAuthCmd())
	rootCmd.AddCommand(newStatusCmd())
	rootCmd.AddCommand(newTeamCmd())
	rootCmd.AddCommand(newEventCmd())
	rootCmd.AddCommand(newMatchCmd())
	rootCmd.AddCommand(newDistrictCmd())
	rootCmd.AddCommand(newInsightCmd())
	rootCmd.AddCommand(newCacheCmd())

	return rootCmd
}

// Execute runs the CLI with the process's standard streams.
func Execute() error {
	return NewRootCmd().Execute()
}
