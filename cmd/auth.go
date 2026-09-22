package cmd

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/the-blue-alliance/tba-cli/internal/api"
	"github.com/the-blue-alliance/tba-cli/internal/clierr"
	"github.com/the-blue-alliance/tba-cli/internal/config"
	"github.com/the-blue-alliance/tba-cli/internal/output"
	"golang.org/x/term"
)

func newAuthCmd() *cobra.Command {
	authCmd := &cobra.Command{
		Use:   "auth",
		Short: "Manage authentication",
	}
	authCmd.AddCommand(newAuthLoginCmd())
	authCmd.AddCommand(newAuthStatusCmd())
	authCmd.AddCommand(newAuthLogoutCmd())
	return authCmd
}

func newAuthLoginCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "login",
		Short: "Authenticate with TBA API",
		Long: `Store a TBA API key for later commands.

Get one at ` + config.APIKeyPage + `: sign in, then create a
read API key. The key is checked against the API before it is stored, so a
typo fails here rather than on every later command.

Without --key the key is read from the terminal without echoing it, or from
stdin when it is piped, so ` + "`tba auth login < key.txt`" + ` works too.`,
		Example: `  tba auth login
  tba auth login --key abcd1234
  tba auth login --base-url http://localhost:8080/api/v3`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Everything here is conversation, not data, so it goes to stderr:
			// `tba auth login < key.txt` should write nothing to stdout.
			errOut := cmd.ErrOrStderr()
			baseURL := getBaseURL(cmd)
			key, _ := cmd.Flags().GetString("key")
			if key == "" {
				var err error
				key, err = readKey(cmd, baseURL)
				if err != nil {
					return err
				}
			}
			if key == "" {
				return clierr.Usage("API key cannot be empty")
			}
			// Check the key before storing it, so a typo fails here rather
			// than on every later command.
			if err := api.ValidateKey(cmd.Context(), baseURL, key); err != nil {
				return err
			}
			if err := config.SaveAPIKey(key, baseURL); err != nil {
				return err
			}
			fmt.Fprintf(errOut, "Authenticated successfully for %s.\n", baseURL)
			return nil
		},
	}
	c.Flags().String("key", "", "API key (or enter interactively)")
	return c
}

// readKey prompts for an API key. On a terminal the key is read without
// echoing it; when stdin is a pipe or a file a single line is read, so that
// `tba auth login < key.txt` works.
func readKey(cmd *cobra.Command, baseURL string) (string, error) {
	errOut := cmd.ErrOrStderr()
	in := cmd.InOrStdin()

	if f, ok := in.(*os.File); ok && term.IsTerminal(int(f.Fd())) {
		fmt.Fprintf(errOut, "Enter your TBA API key for %s: ", baseURL)
		b, err := term.ReadPassword(int(f.Fd()))
		fmt.Fprintln(errOut)
		if err != nil {
			return "", err
		}
		return strings.TrimSpace(string(b)), nil
	}

	fmt.Fprintf(errOut, "Enter your TBA API key for %s: ", baseURL)
	line, err := bufio.NewReader(in).ReadString('\n')
	if err != nil && !errors.Is(err, io.EOF) {
		return "", err
	}
	return strings.TrimSpace(line), nil
}

// authStatusReport is what `auth status` prints in JSON.
type authStatusReport struct {
	Authenticated bool   `json:"authenticated"`
	BaseURL       string `json:"base_url"`
	KeyMasked     string `json:"key_masked"`
	ConfigFile    string `json:"config_file"`
}

func newAuthStatusCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show authentication status",
		Example: `  tba auth status
  tba auth status --format json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			baseURL := getBaseURL(cmd)
			key, err := config.GetAPIKey(baseURL)
			if err != nil {
				// Reported on stderr by main, with exit code 4, so that a
				// script can tell "no key" from "no data".
				return err
			}
			authFile, err := config.AuthFile()
			if err != nil {
				return err
			}
			report := authStatusReport{
				Authenticated: true,
				BaseURL:       baseURL,
				KeyMasked:     maskKey(key),
				ConfigFile:    authFile,
			}
			return outputData(cmd, report, func() {
				output.PrintKeyValue(cmd.OutOrStdout(),
					"Authenticated with key", report.KeyMasked,
					"Base URL", report.BaseURL,
					"Config file", report.ConfigFile,
				)
			})
		},
	}
}

// maskKey hides everything but the last four characters of an API key, the
// way a payment card is shown. Keys of four characters or fewer are hidden
// completely rather than leaked whole.
func maskKey(key string) string {
	const visible = 4
	runes := []rune(key)
	if len(runes) <= visible {
		return strings.Repeat("*", len(runes))
	}
	return strings.Repeat("*", len(runes)-visible) + string(runes[len(runes)-visible:])
}

func newAuthLogoutCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "logout",
		Short: "Remove stored API key",
		Example: `  tba auth logout
  tba auth logout --base-url http://localhost:8080/api/v3`,
		RunE: func(cmd *cobra.Command, args []string) error {
			baseURL := getBaseURL(cmd)
			// Nothing to remove is a failure: the caller asked for a change
			// that did not happen.
			if err := config.RemoveAPIKey(baseURL); err != nil {
				return err
			}
			fmt.Fprintf(cmd.ErrOrStderr(), "Logged out from %s.\n", baseURL)
			return nil
		},
	}
}
