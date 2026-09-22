package cmd

import (
	"context"
	"fmt"

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

// Run executes an already-built command tree.
//
// Cobra's own usage printing is off (SilenceUsage), so that usage is shown
// exactly when the failure is a mistake in how the command was invoked: a bad
// flag, a bad argument, a malformed key. A failure that happened while doing
// the work — a 404, a timeout — prints only its message, because a wall of
// usage text buries it.
func Run(ctx context.Context, root *cobra.Command) error {
	// Watch stdout, so that a reader hanging up mid-command is reported even
	// though the printing helpers ignore write failures.
	out := &recordingWriter{w: root.OutOrStdout()}
	root.SetOut(out)

	cmd, err := root.ExecuteContextC(ctx)
	if err == nil {
		err = out.Err()
	}
	if err == nil {
		return nil
	}
	if clierr.ExitCode(err) == clierr.ExitUsage {
		if cmd == nil {
			cmd = root
		}
		w := cmd.ErrOrStderr()
		fmt.Fprint(w, cmd.UsageString())
		fmt.Fprintf(w, "Run '%s --help' for usage.\n", cmd.CommandPath())
	}
	return err
}

// Execute runs the CLI with the process's standard streams.
func Execute() error {
	return Run(context.Background(), NewRootCmd())
}
