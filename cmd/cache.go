package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/the-blue-alliance/tba-cli/internal/cache"
)

func newCacheCmd() *cobra.Command {
	cacheCmd := &cobra.Command{
		Use:   "cache",
		Short: "Manage the local response cache",
	}
	cacheCmd.AddCommand(newCacheInfoCmd())
	cacheCmd.AddCommand(newCacheClearCmd())
	return cacheCmd
}

func newCacheInfoCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "info",
		Short: "Show cache directory and size",
		Example: `  tba cache info
  tba cache info --format json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			out := cmd.OutOrStdout()
			c, err := cache.New()
			if err != nil {
				return err
			}
			count, bytes, err := c.Stats()
			if err != nil {
				return err
			}
			fmt.Fprintf(out, "Directory: %s\n", c.Dir())
			fmt.Fprintf(out, "Entries:   %d\n", count)
			fmt.Fprintf(out, "Size:      %s\n", cache.FormatSize(bytes))
			return nil
		},
	}
}

func newCacheClearCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "clear",
		Short: "Remove all cached responses",
		Example: `  tba cache clear
  tba cache clear --format json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := cache.New()
			if err != nil {
				return err
			}
			removed, err := c.Clear()
			if err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Removed %d cache entries from %s\n", removed, c.Dir())
			return nil
		},
	}
}
