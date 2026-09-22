package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"
	"github.com/the-blue-alliance/tba-cli/internal/api"
	"github.com/the-blue-alliance/tba-cli/internal/clierr"
	"github.com/the-blue-alliance/tba-cli/internal/version"
)

// NewRootCmd builds a fresh `tba` command tree. Every command is constructed
// per call so that flag state is never shared between invocations, which keeps
// tests independent of each other.
func NewRootCmd() *cobra.Command {
	rootCmd := &cobra.Command{
		Use:   "tba",
		Short: "The Blue Alliance CLI",
		Long:  "A command-line interface for The Blue Alliance API v3.",
		// `tba --help` is where someone lands first, and a list of nouns does
		// not show what the tool is for. These are the questions people
		// actually turn up with, in the order they tend to ask them.
		Example: `  tba team next 177
  tba event matches 2024cthar --team 177 --upcoming
  tba event rankings 2024cthar
  tba team standing 177 2024cthar
  tba event export 2024cthar --to csv
  tba event list --week 4 --district ne`,
		SilenceErrors: true,
		SilenceUsage:  true,
		// Setting Version gives the root a --version flag.
		Version: version.Resolve().Line(),
		// Flags are checked once, before any command does work, so that a
		// contradictory --format is reported without first hitting the API.
		PersistentPreRunE: func(cmd *cobra.Command, _ []string) error {
			// Resolve flags, environment and config file into one view before
			// anything reads a setting, then check the ones whose value can be
			// wrong, so that a bad --format is reported without first hitting
			// the API.
			if err := initSettings(cmd); err != nil {
				return err
			}
			if _, err := resolveFormat(cmd); err != nil {
				return err
			}
			if err := checkJQ(cmd); err != nil {
				return err
			}
			_, err := colorMode(cmd)
			return err
		},
	}

	rootCmd.SetVersionTemplate(versionTemplate)

	// Cobra reports a bad flag as a plain error; tag it so main can exit 2.
	rootCmd.SetFlagErrorFunc(func(_ *cobra.Command, err error) error {
		return clierr.Wrap(clierr.KindUsage, err)
	})

	rootCmd.PersistentFlags().Bool("json", false, "Output as JSON (shorthand for --format=json)")
	rootCmd.PersistentFlags().String("jq", "", "Apply jq expression to JSON output")
	rootCmd.PersistentFlags().BoolP("raw-output", "r", false, "With --jq, print string results without quotes (like jq -r)")
	rootCmd.PersistentFlags().String("base-url", "", "Override API base URL (e.g. http://localhost:8080/api/v3)")
	rootCmd.PersistentFlags().Bool("no-cache", false, "Disable HTTP response cache for this invocation")
	rootCmd.PersistentFlags().Bool("offline", false, "Never contact the API; answer from the local cache only")
	rootCmd.PersistentFlags().String("format", "", "Output format: auto, table, json, csv, tsv, markdown (auto: table on TTY, json otherwise)")
	rootCmd.PersistentFlags().Duration("timeout", api.DefaultTimeout, "Per-request timeout (e.g. 10s, 1m)")
	rootCmd.PersistentFlags().Int("retries", api.DefaultRetries, "Retry attempts for 429/5xx/network errors; 0 disables")
	rootCmd.PersistentFlags().Bool("no-headers", false, "Omit the header row from table, csv, tsv and markdown output")
	rootCmd.PersistentFlags().String("columns", "", "Select and order columns by header name or 1-based index (e.g. --columns key,name)")
	// The example is a column every listing has and every format can order
	// by. It used to be --sort=-opr, which fails the moment the output is
	// piped: the OPR payload is an object keyed by team, and an object has no
	// row order to rearrange.
	rootCmd.PersistentFlags().String("sort", "", "Sort rows by a column; prefix with - to descend (e.g. --sort=name)")
	rootCmd.PersistentFlags().String("color", "auto", "When to colorize output: auto, always, never")
	rootCmd.PersistentFlags().Bool("no-color", false, "Disable colored output (alias for --color=never; wins over --color)")

	rootCmd.AddCommand(newAuthCmd())
	rootCmd.AddCommand(newStatusCmd())
	rootCmd.AddCommand(newTeamCmd())
	rootCmd.AddCommand(newEventCmd())
	rootCmd.AddCommand(newMatchCmd())
	rootCmd.AddCommand(newDistrictCmd())
	rootCmd.AddCommand(newInsightCmd())
	rootCmd.AddCommand(newCacheCmd())
	rootCmd.AddCommand(newVersionCmd())
	rootCmd.AddCommand(newDocsCmd())
	rootCmd.AddCommand(newOpenCmd())
	rootCmd.AddCommand(newConfigCmd())

	// A command that only groups others must still refuse an unknown one;
	// applied to the finished tree so a new group cannot forget it.
	applyGroupArgs(rootCmd)

	// Argument completion is wired onto the finished tree; see completion.go.
	attachCompletions(rootCmd)

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
		printUsageHint(cmd)
	}
	return err
}

// printUsageHint shows how the command is called and where the rest is.
//
// Cobra's full usage block is about twenty-five lines, most of them the global
// flags, and it pushes the one line that says what went wrong off the top of a
// small terminal. The call shape is the part that helps in the moment; --help
// is one keystroke away for everything else.
func printUsageHint(cmd *cobra.Command) {
	w := cmd.ErrOrStderr()
	fmt.Fprintf(w, "Usage:\n  %s\n", cmd.UseLine())
	if cmd.HasAvailableSubCommands() {
		fmt.Fprintf(w, "  %s [command]\n", cmd.CommandPath())
	}
	fmt.Fprintf(w, "Run '%s --help' for usage.\n", cmd.CommandPath())
}

// Execute runs the CLI with the process's standard streams.
//
// The command tree runs under a context that is cancelled on SIGINT or
// SIGTERM, so Ctrl-C aborts an in-flight request instead of waiting for it.
func Execute() error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	return Run(ctx, NewRootCmd())
}
