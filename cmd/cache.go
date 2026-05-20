package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/the-blue-alliance/tba-cli/internal/cache"
)

var cacheCmd = &cobra.Command{
	Use:   "cache",
	Short: "Manage the local response cache",
}

var cacheInfoCmd = &cobra.Command{
	Use:   "info",
	Short: "Show cache directory and size",
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := cache.New()
		if err != nil {
			return err
		}
		count, bytes, err := c.Stats()
		if err != nil {
			return err
		}
		fmt.Printf("Directory: %s\n", c.Dir())
		fmt.Printf("Entries:   %d\n", count)
		fmt.Printf("Size:      %s\n", cache.FormatSize(bytes))
		return nil
	},
}

var cacheClearCmd = &cobra.Command{
	Use:   "clear",
	Short: "Remove all cached responses",
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := cache.New()
		if err != nil {
			return err
		}
		removed, err := c.Clear()
		if err != nil {
			return err
		}
		fmt.Printf("Removed %d cache entries from %s\n", removed, c.Dir())
		return nil
	},
}

func init() {
	cacheCmd.AddCommand(cacheInfoCmd)
	cacheCmd.AddCommand(cacheClearCmd)
	rootCmd.AddCommand(cacheCmd)
}
