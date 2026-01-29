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
		key, _ := cmd.Flags().GetString("key")
		if key == "" {
			fmt.Print("Enter your TBA API key: ")
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
		if err := config.SaveAPIKey(key); err != nil {
			return err
		}
		fmt.Println("Authenticated successfully.")
		return nil
	},
}

var authStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show authentication status",
	RunE: func(cmd *cobra.Command, args []string) error {
		key, err := config.GetAPIKey()
		if err != nil {
			fmt.Println("Not authenticated.")
			return nil
		}
		// Mask key
		masked := key[:4] + strings.Repeat("*", len(key)-4)
		fmt.Printf("Authenticated with key: %s\n", masked)
		fmt.Printf("Config file: %s\n", config.AuthFile())
		return nil
	},
}

var authLogoutCmd = &cobra.Command{
	Use:   "logout",
	Short: "Remove stored API key",
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := config.RemoveAPIKey(); err != nil {
			if os.IsNotExist(err) {
				fmt.Println("Not authenticated.")
				return nil
			}
			return err
		}
		fmt.Println("Logged out.")
		return nil
	},
}

func init() {
	authLoginCmd.Flags().String("key", "", "API key (or enter interactively)")
	authCmd.AddCommand(authLoginCmd)
	authCmd.AddCommand(authStatusCmd)
	authCmd.AddCommand(authLogoutCmd)
}
