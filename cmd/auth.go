package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/the-blue-alliance/tba-cli/internal/config"
)

var authCmd = &cobra.Command{
	Use:   "auth",
	Short: "Manage authentication",
}

var authLoginCmd = &cobra.Command{
	Use:   "login",
	Short: "Authenticate with TBA API",
	RunE: func(cmd *cobra.Command, args []string) error {
		baseURL := getBaseURL(cmd)
		key, _ := cmd.Flags().GetString("key")
		if key == "" {
			fmt.Printf("Enter your TBA API key for %s: ", baseURL)
			reader := bufio.NewReader(os.Stdin)
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
		fmt.Printf("Authenticated successfully for %s.\n", baseURL)
		return nil
	},
}

var authStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show authentication status",
	RunE: func(cmd *cobra.Command, args []string) error {
		baseURL := getBaseURL(cmd)
		key, err := config.GetAPIKey(baseURL)
		if err != nil {
			fmt.Printf("Not authenticated for %s.\n", baseURL)
			return nil
		}
		// Mask key
		masked := key[:4] + strings.Repeat("*", len(key)-4)
		fmt.Printf("Authenticated with key: %s\n", masked)
		fmt.Printf("Base URL: %s\n", baseURL)
		fmt.Printf("Config file: %s\n", config.AuthFile())
		return nil
	},
}

var authLogoutCmd = &cobra.Command{
	Use:   "logout",
	Short: "Remove stored API key",
	RunE: func(cmd *cobra.Command, args []string) error {
		baseURL := getBaseURL(cmd)
		if err := config.RemoveAPIKey(baseURL); err != nil {
			fmt.Println(err)
			return nil
		}
		fmt.Printf("Logged out from %s.\n", baseURL)
		return nil
	},
}

func init() {
	authLoginCmd.Flags().String("key", "", "API key (or enter interactively)")
	authCmd.AddCommand(authLoginCmd)
	authCmd.AddCommand(authStatusCmd)
	authCmd.AddCommand(authLogoutCmd)
}
