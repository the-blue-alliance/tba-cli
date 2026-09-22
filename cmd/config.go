package cmd

import (
	"fmt"
	"io"
	"strconv"

	"github.com/spf13/cobra"

	"github.com/the-blue-alliance/tba-cli/internal/api"
	"github.com/the-blue-alliance/tba-cli/internal/config"
	"github.com/the-blue-alliance/tba-cli/internal/output"
)

func newConfigCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "config",
		Short: "Read and write tba's configuration",
		Long: `Read and write tba's configuration file.

The file holds defaults for the persistent flags, under the flag's own name.
A flag beats an environment variable, which beats the file, which beats the
built-in default. A "format" set in the file applies only when output goes to
a terminal, so a table default never changes what a script reads from a pipe.`,
		Example: `  tba config list
  tba config set format table
  tba config get timeout
  tba config unset format
  tba config path`,
	}
	c.AddCommand(newConfigListCmd())
	c.AddCommand(newConfigGetCmd())
	c.AddCommand(newConfigSetCmd())
	c.AddCommand(newConfigUnsetCmd())
	c.AddCommand(newConfigPathCmd())
	return c
}

// configEntry is one row of `tba config list`.
type configEntry struct {
	Key    string `json:"key"`
	Value  string `json:"value"`
	Source string `json:"source"`
}

// settingDisplay renders a setting's effective value as text. Keys whose
// unset state is a sentinel (an empty format, an empty base URL, year 0) show
// what the sentinel actually means, because "" and "0" answer no question.
func settingDisplay(s *settingsSet, name string, w io.Writer) string {
	switch name {
	case "format":
		if v := s.Format(w); v != "" {
			return v
		}
		return "auto"
	case "base-url":
		if v := s.String("base-url"); v != "" {
			return v
		}
		return api.DefaultBaseURL
	case "year":
		if n := s.Int("year"); n != 0 {
			return strconv.Itoa(n)
		}
		return "current season"
	case "timeout":
		return s.Duration(name).String()
	case "retries":
		return strconv.Itoa(s.Int(name))
	case "no-color", "no-cache":
		return strconv.FormatBool(s.Bool(name))
	default:
		return s.String(name)
	}
}

// settingSource is Source with the format rule folded in: a format the file
// asks for but that this invocation cannot honour (because output is not a
// terminal) is not the source of the value in effect.
func settingSource(s *settingsSet, name string, w io.Writer) string {
	if name == "format" && s.Source(name) == sourceConfig && !output.IsTTY(w) {
		return sourceDefault
	}
	return s.Source(name)
}

func newConfigListCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "list",
		Short:   "Show every setting with its effective value and source",
		Aliases: []string{"ls"},
		Args:    cobra.NoArgs,
		Example: `  tba config list
  tba config list --format json
  tba config list --jq '.[] | select(.source != "default")'`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			s := settings(cmd)
			w := cmd.OutOrStdout()
			entries := make([]configEntry, 0, len(config.Keys))
			rows := make([][]string, 0, len(config.Keys))
			for _, key := range config.Keys {
				e := configEntry{
					Key:    key.Name,
					Value:  settingDisplay(s, key.Name, w),
					Source: settingSource(s, key.Name, w),
				}
				entries = append(entries, e)
				rows = append(rows, []string{e.Key, e.Value, e.Source})
			}
			return outputTable(cmd, entries, []string{"Key", "Value", "Source"}, rows)
		},
	}
}

func newConfigGetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "get <key>",
		Short: "Print one setting's effective value",
		Args:  exactArgs(1, "a setting name (e.g. tba config get format)"),
		Example: `  tba config get format
  tba config get timeout
  tba config get year --json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			key, ok := config.LookupKey(args[0])
			if !ok {
				return config.UnknownKeyError(args[0])
			}
			s := settings(cmd)
			w := cmd.OutOrStdout()
			e := configEntry{
				Key:    key.Name,
				Value:  settingDisplay(s, key.Name, w),
				Source: settingSource(s, key.Name, w),
			}
			return printScalar(cmd, e, e.Value)
		},
	}
}

func newConfigSetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "set <key> <value>",
		Short: "Write a setting to the config file",
		Args:  exactArgs(2, "a setting name and a value (e.g. tba config set format table)"),
		Example: `  tba config set format table
  tba config set timeout 30s
  tba config set retries 0
  tba config set year 2024`,
		RunE: func(cmd *cobra.Command, args []string) error {
			key, ok := config.LookupKey(args[0])
			if !ok {
				return config.UnknownKeyError(args[0])
			}
			value, err := key.ParseValue(args[1])
			if err != nil {
				return err
			}
			if err := config.SetSetting(key.Name, value); err != nil {
				return err
			}
			path, err := config.SettingsFile()
			if err != nil {
				return err
			}
			fmt.Fprintf(cmd.ErrOrStderr(), "set %s = %v in %s\n", key.Name, value, path)
			return nil
		},
	}
}

func newConfigUnsetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "unset <key>",
		Short: "Remove a setting from the config file",
		Args:  exactArgs(1, "a setting name (e.g. tba config unset format)"),
		Example: `  tba config unset format
  tba config unset timeout`,
		RunE: func(cmd *cobra.Command, args []string) error {
			key, ok := config.LookupKey(args[0])
			if !ok {
				return config.UnknownKeyError(args[0])
			}
			if err := config.UnsetSetting(key.Name); err != nil {
				return err
			}
			path, err := config.SettingsFile()
			if err != nil {
				return err
			}
			fmt.Fprintf(cmd.ErrOrStderr(), "unset %s in %s\n", key.Name, path)
			return nil
		},
	}
}

func newConfigPathCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "path",
		Short: "Print the path of the config file",
		Args:  cobra.NoArgs,
		// The bare path is what makes `$EDITOR "$(tba config path)"` work.
		Example: `  tba config path
  tba config path --json`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			path, err := config.SettingsFile()
			if err != nil {
				return err
			}
			return printScalar(cmd, map[string]string{"path": path}, path)
		},
	}
}

// printScalar writes a single value that a script is meant to read: the bare
// text, unless JSON was actually asked for. It deliberately does not follow
// --format auto into JSON on a pipe, because `$(tba config path)` and
// `$(tba config get format)` are the whole point of these commands.
func printScalar(cmd *cobra.Command, data any, text string) error {
	s := settings(cmd)
	if s.Bool("json") || s.String("jq") != "" || s.Format(cmd.OutOrStdout()) == "json" {
		return output.PrintJSONWithFilter(cmd.OutOrStdout(), data, jqExpr(cmd), rawOutput(cmd))
	}
	fmt.Fprintln(cmd.OutOrStdout(), text)
	return nil
}
