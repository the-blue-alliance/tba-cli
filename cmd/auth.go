package cmd

import (
	"bufio"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/the-blue-alliance/tba-cli/internal/config"
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
		RunE: func(cmd *cobra.Command, args []string) error {
			out := cmd.OutOrStdout()
			baseURL := getBaseURL(cmd)
			key, _ := cmd.Flags().GetString("key")
			if key == "" {
				fmt.Fprintf(out, "Enter your TBA API key for %s: ", baseURL)
				reader := bufio.NewReader(cmd.InOrStdin())
				input, err := reader.ReadString('\n')
				if err != nil {
					return err
				}
				key = strings.TrimSpace(input)
			}
			if key == "" {
				return fmt.Errorf("API key cannot be empty")
			}
			if err := config.SaveAPIKey(key, baseURL); err != nil {
				return err
			}
			fmt.Fprintf(out, "Authenticated successfully for %s.\n", baseURL)
			return nil
		},
	}
	c.Flags().String("key", "", "API key (or enter interactively)")
	return c
}

func newAuthStatusCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show authentication status",
		RunE: func(cmd *cobra.Command, args []string) error {
			out := cmd.OutOrStdout()
			baseURL := getBaseURL(cmd)
			key, err := config.GetAPIKey(baseURL)
			if err != nil {
				fmt.Fprintf(out, "Not authenticated for %s.\n", baseURL)
				return nil
			}
			fmt.Fprintf(out, "Authenticated with key: %s\n", maskKey(key))
			fmt.Fprintf(out, "Base URL: %s\n", baseURL)
			fmt.Fprintf(out, "Config file: %s\n", config.AuthFile())
			return nil
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
		RunE: func(cmd *cobra.Command, args []string) error {
			out := cmd.OutOrStdout()
			baseURL := getBaseURL(cmd)
			if err := config.RemoveAPIKey(baseURL); err != nil {
				fmt.Fprintln(out, err)
				return nil
			}
			fmt.Fprintf(out, "Logged out from %s.\n", baseURL)
			return nil
		},
	}
}
