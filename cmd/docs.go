package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/cobra/doc"
	"github.com/the-blue-alliance/tba-cli/internal/clierr"
	"github.com/the-blue-alliance/tba-cli/internal/version"
)

// fallbackDocDate dates generated documentation when nothing else says when
// this binary was built. It is deliberately a constant rather than today: a
// release archive has to contain the same bytes however often it is rebuilt,
// and the clock is the one input that is never the same twice.
var fallbackDocDate = time.Date(2024, time.January, 1, 0, 0, 0, 0, time.UTC)

// newDocsCmd builds the documentation generators.
//
// These exist for the release pipeline and for anyone regenerating the docs by
// hand, not for day-to-day use, so the group is hidden: `tba --help` stays a
// list of things you would actually type.
func newDocsCmd() *cobra.Command {
	docsCmd := &cobra.Command{
		Use:   "docs",
		Short: "Generate documentation and completion scripts",
		Long: "Generate man pages, markdown documentation and shell completion scripts.\n\n" +
			"Output is reproducible: the date in a man page comes from --date, else\n" +
			"SOURCE_DATE_EPOCH, else this binary's build date. Never from the clock.",
		Hidden: true,
		Args:   cobra.NoArgs,
	}
	docsCmd.AddCommand(newDocsManCmd(), newDocsMarkdownCmd(), newDocsCompletionsCmd())
	return docsCmd
}

func newDocsManCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "man",
		Short: "Write man pages for every command",
		Example: `  tba docs man --dir manpages
  tba docs man --dir manpages --date 2024-06-01T00:00:00Z`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			dir, err := docsDir(cmd)
			if err != nil {
				return err
			}
			date, err := docsDate(cmd)
			if err != nil {
				return err
			}
			header := &doc.GenManHeader{
				Title:   "TBA",
				Section: "1",
				Date:    &date,
				// Not the running binary's version: the release generates
				// these pages with `go run`, which carries no version stamp,
				// so a version here would read "dev" in every release archive.
				Source: "tba",
				Manual: "The Blue Alliance CLI",
			}
			return doc.GenManTree(docsTree(), header, dir)
		},
	}
	addDocsDirFlag(cmd)
	cmd.Flags().String("date", "", "Date for the man page header: RFC 3339 or seconds since the Unix epoch (default: SOURCE_DATE_EPOCH, else the build date)")
	return cmd
}

func newDocsMarkdownCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "markdown",
		Aliases: []string{"md"},
		Short:   "Write markdown documentation for every command",
		Example: `  tba docs markdown --dir docs
  tba docs md --dir /tmp/tba-docs`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			dir, err := docsDir(cmd)
			if err != nil {
				return err
			}
			return doc.GenMarkdownTree(docsTree(), dir)
		},
	}
	addDocsDirFlag(cmd)
	return cmd
}

func newDocsCompletionsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "completions",
		Short: "Write bash, zsh and fish completion scripts",
		Long: "Write completion scripts as files, one per shell.\n\n" +
			"`tba completion <shell>` prints one script to stdout; this writes all of\n" +
			"them to a directory, which is what the release archives ship.",
		Example: `  tba docs completions --dir completions
  tba docs completions --dir /tmp/tba-completions`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			dir, err := docsDir(cmd)
			if err != nil {
				return err
			}
			root := docsTree()
			writers := []struct {
				name  string
				write func(string) error
			}{
				{"tba.bash", func(p string) error { return root.GenBashCompletionFileV2(p, true) }},
				{"tba.zsh", root.GenZshCompletionFile},
				{"tba.fish", func(p string) error { return root.GenFishCompletionFile(p, true) }},
			}
			for _, w := range writers {
				if err := w.write(filepath.Join(dir, w.name)); err != nil {
					return fmt.Errorf("writing %s: %w", w.name, err)
				}
			}
			return nil
		},
	}
	addDocsDirFlag(cmd)
	return cmd
}

func addDocsDirFlag(cmd *cobra.Command) {
	cmd.Flags().String("dir", ".", "Directory to write into; created if it does not exist")
}

// docsDir resolves --dir and makes sure it exists.
func docsDir(cmd *cobra.Command) (string, error) {
	dir, _ := cmd.Flags().GetString("dir")
	if dir == "" {
		return "", clierr.Usage("--dir cannot be empty")
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", fmt.Errorf("creating %s: %w", dir, err)
	}
	return dir, nil
}

// docsDate is the timestamp stamped into generated man pages.
//
// --date wins, for a build system that knows which commit it is releasing.
// SOURCE_DATE_EPOCH is the cross-project convention for "pretend this is when
// the build happened", and reproducible-build tooling sets it. Failing both,
// the binary's own build date is just as stable. The clock is never consulted,
// because a page that changes by the day makes every archive differ.
func docsDate(cmd *cobra.Command) (time.Time, error) {
	if flagDate, _ := cmd.Flags().GetString("date"); flagDate != "" {
		t, err := parseDocsDate(flagDate)
		if err != nil {
			return time.Time{}, clierr.Usage("invalid --date %q: want RFC 3339 or seconds since the Unix epoch", flagDate)
		}
		return t, nil
	}
	if epoch := os.Getenv("SOURCE_DATE_EPOCH"); epoch != "" {
		secs, err := strconv.ParseInt(epoch, 10, 64)
		if err != nil {
			return time.Time{}, clierr.Usage("invalid SOURCE_DATE_EPOCH %q: want seconds since the Unix epoch", epoch)
		}
		return time.Unix(secs, 0).UTC(), nil
	}
	if d := version.Resolve().Date; d != "" {
		if t, err := time.Parse(time.RFC3339, d); err == nil {
			return t.UTC(), nil
		}
	}
	return fallbackDocDate, nil
}

// parseDocsDate accepts both spellings a build system has to hand: goreleaser
// offers the commit date as RFC 3339 and as a Unix timestamp.
func parseDocsDate(s string) (time.Time, error) {
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return t.UTC(), nil
	}
	secs, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return time.Time{}, err
	}
	return time.Unix(secs, 0).UTC(), nil
}

// docsTree is the command tree the documentation describes: a fresh one, so
// that generating docs cannot be influenced by the flags this invocation was
// given. The generators stamp "Auto generated by spf13/cobra on <today>" into
// every page unless told not to, which is the other way the clock leaks in, so
// it is turned off on every command in the tree.
func docsTree() *cobra.Command {
	root := NewRootCmd()
	disableAutoGenTag(root)
	return root
}

func disableAutoGenTag(cmd *cobra.Command) {
	cmd.DisableAutoGenTag = true
	for _, c := range cmd.Commands() {
		disableAutoGenTag(c)
	}
}
