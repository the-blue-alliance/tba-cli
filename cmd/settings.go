package cmd

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/the-blue-alliance/tba-cli/internal/clierr"
	"github.com/the-blue-alliance/tba-cli/internal/config"
	"github.com/the-blue-alliance/tba-cli/internal/output"
)

// Where a setting's value came from, as `tba config list` reports it.
const (
	sourceFlag    = "flag"
	sourceEnv     = "env"
	sourceConfig  = "config"
	sourceDefault = "default"
)

// settingsSet is the layered view of the persistent flags: a flag beats an
// environment variable, which beats the config file, which beats the flag's
// own default. Every helper reads through it so that `--timeout 30s`,
// `TBA_TIMEOUT=30s` and `timeout: 30s` in config.yaml are the same thing.
type settingsSet struct {
	v    *viper.Viper
	cmd  *cobra.Command
	file *config.Settings
}

// envKey is the environment variable a setting is read from: the flag name,
// upper-snaked, with the TBA_ prefix (base-url -> TBA_BASE_URL).
func envKey(name string) string {
	return "TBA_" + strings.ToUpper(strings.ReplaceAll(name, "-", "_"))
}

// newSettings builds the layered view for one invocation. It fails only when
// the config file exists but cannot be read, which is worth stopping for: the
// alternative is running with settings the user believes are in effect.
func newSettings(cmd *cobra.Command) (*settingsSet, error) {
	file, err := config.LoadSettings()
	if err != nil {
		return nil, err
	}
	return newSettingsFrom(cmd, file), nil
}

func newSettingsFrom(cmd *cobra.Command, file *config.Settings) *settingsSet {
	v := viper.New()
	v.SetEnvPrefix("TBA")
	v.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))
	v.AutomaticEnv()

	// Binding the flags gives viper both ends of the precedence chain: a flag
	// the user actually typed outranks everything, and the flag's default is
	// the last resort below the config file.
	_ = v.BindPFlags(cmd.Root().PersistentFlags())
	// --year is a per-command flag rather than a persistent one, but it takes
	// part in the same layering.
	v.SetDefault("year", 0)
	if f := cmd.Flags().Lookup("year"); f != nil {
		_ = v.BindPFlag("year", f)
	}
	if len(file.Values) > 0 {
		_ = v.MergeConfigMap(file.Values)
	}
	return &settingsSet{v: v, cmd: cmd, file: file}
}

type settingsCtxKey struct{}

// initSettings resolves the layered settings once per invocation and hangs
// them on the command's context. Unknown keys in the file are reported here,
// once, and then ignored.
func initSettings(cmd *cobra.Command) error {
	s, err := newSettings(cmd)
	if err != nil {
		return err
	}
	for _, name := range s.file.Unknown {
		fmt.Fprintf(cmd.ErrOrStderr(), "warning: unknown key %q in %s\n", name, s.file.Path)
	}
	if err := s.validate(); err != nil {
		return err
	}
	ctx := cmd.Context()
	if ctx == nil {
		ctx = context.Background()
	}
	cmd.SetContext(context.WithValue(ctx, settingsCtxKey{}, s))
	return nil
}

// validate checks the settings whose values have to make sense whatever layer
// they came from.
//
// `tba config set retries -1` was refused while `--retries -1` and
// TBA_RETRIES=-1 were accepted without a word, so the same nonsense was an
// error in one place and silently something else in another. The wording is
// the config file's, with the layer named when it is not the flag the user
// would otherwise go looking for.
func (s *settingsSet) validate() error {
	if s.Int("retries") < 0 {
		return s.settingError("retries", "retries cannot be negative")
	}
	if s.Duration("timeout") <= 0 {
		return s.settingError("timeout", "timeout must be positive")
	}
	return nil
}

// settingError reports a bad value, naming where it came from unless it came
// from the flag the message already mentions.
func (s *settingsSet) settingError(name, message string) error {
	if s.Source(name) == sourceFlag || s.Source(name) == sourceDefault {
		return clierr.Usage("%s", message)
	}
	return clierr.Usage("%s (from %s)", message, s.origin(name))
}

// settings returns the layered settings for this invocation, building them if
// the command was not run through the root's PersistentPreRunE (which is the
// case in a few tests). A config file that cannot be read has already been
// reported by then, so here it simply means "no file".
func settings(cmd *cobra.Command) *settingsSet {
	if ctx := cmd.Context(); ctx != nil {
		if s, ok := ctx.Value(settingsCtxKey{}).(*settingsSet); ok {
			return s
		}
	}
	file, err := config.LoadSettings()
	if err != nil || file == nil {
		file = &config.Settings{Values: map[string]any{}}
	}
	return newSettingsFrom(cmd, file)
}

// Source says which layer supplied a setting's value.
func (s *settingsSet) Source(name string) string {
	if f := s.cmd.Flags().Lookup(name); f != nil && f.Changed {
		return sourceFlag
	}
	// viper ignores an empty environment variable, so an exported but empty
	// TBA_FORMAT is not a source either.
	if os.Getenv(envKey(name)) != "" {
		return sourceEnv
	}
	if _, ok := s.file.Values[name]; ok {
		return sourceConfig
	}
	return sourceDefault
}

func (s *settingsSet) String(name string) string          { return s.v.GetString(name) }
func (s *settingsSet) Bool(name string) bool              { return s.v.GetBool(name) }
func (s *settingsSet) Int(name string) int                { return s.v.GetInt(name) }
func (s *settingsSet) Duration(name string) time.Duration { return s.v.GetDuration(name) }

// ConfigPath is where config.yaml lives, whether or not it exists.
func (s *settingsSet) ConfigPath() string { return s.file.Path }

// Format returns the --format setting for output written to w.
//
// A format named in the config file applies only when w is a terminal: a user
// whose config says `format: table` still gets JSON out of a pipe, so scripts
// keep working on a machine whose owner prefers tables. A format from the
// command line or the environment is deliberate enough to apply either way.
func (s *settingsSet) Format(w io.Writer) string {
	if s.Source("format") == sourceConfig && !output.IsTTY(w) {
		return ""
	}
	return s.String("format")
}

// origin describes where a bad value came from, for error messages.
func (s *settingsSet) origin(name string) string {
	switch s.Source(name) {
	case sourceEnv:
		return envKey(name)
	case sourceConfig:
		return s.file.Path
	default:
		return "--" + name
	}
}
