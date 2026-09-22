package cmd

import (
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/the-blue-alliance/tba-cli/internal/clierr"
)

// groupCommands walks the tree and returns every command that only groups
// others, so a group added after this test was written is covered too.
func groupCommands(cmd *cobra.Command) []*cobra.Command {
	var out []*cobra.Command
	for _, c := range cmd.Commands() {
		if !c.HasSubCommands() {
			continue
		}
		out = append(out, c)
		out = append(out, groupCommands(c)...)
	}
	return out
}

// `tba team blah` used to print the group's help on stdout and exit 0, so a
// typo in a script looked like a successful run. Every group has to answer it
// the way the root answers `tba teem`.
func TestEveryGroupRejectsAnUnknownSubcommand(t *testing.T) {
	groups := groupCommands(NewRootCmd())
	if len(groups) < 5 {
		t.Fatalf("found %d group commands, expected the whole tree", len(groups))
	}
	for _, g := range groups {
		path := strings.Fields(g.CommandPath())[1:]
		t.Run(g.CommandPath(), func(t *testing.T) {
			args := append(append([]string{}, path...), "definitely-not-a-command")
			stdout, stderr, err := runCmd(t, nil, args...)
			if err == nil {
				t.Fatalf("%s accepted an unknown subcommand", g.CommandPath())
			}
			if got := clierr.ExitCode(err); got != clierr.ExitUsage {
				t.Errorf("exit code = %d, want %d", got, clierr.ExitUsage)
			}
			want := "unknown command \"definitely-not-a-command\" for \"" + g.CommandPath() + "\""
			if !strings.Contains(err.Error(), want) {
				t.Errorf("error = %q, want it to contain %q", err.Error(), want)
			}
			if stdout != "" {
				t.Errorf("nothing should reach stdout, got:\n%s", stdout)
			}
			requireContains(t, stderr, "Run '"+g.CommandPath()+" --help' for usage.")
		})
	}
}

// A near miss gets cobra's did-you-mean, the same as on the root.
func TestGroupSuggestsTheCommandYouMeant(t *testing.T) {
	_, _, err := runCmd(t, nil, "cache", "lst")
	if err == nil {
		t.Fatal("want an error for `tba cache lst`")
	}
	if !strings.Contains(err.Error(), "Did you mean this?") || !strings.Contains(err.Error(), "list") {
		t.Errorf("error = %q, want a suggestion of `list`", err.Error())
	}
}

// With no argument at all a group still prints its own help, on stdout, and
// exits 0: that is how someone finds out what is under it.
func TestGroupWithNoArgumentsStillPrintsHelp(t *testing.T) {
	stdout, _, err := runCmd(t, nil, "cache")
	requireNoError(t, err, "")
	requireContains(t, stdout, "Available Commands:")
	requireContains(t, stdout, "prune")
}
