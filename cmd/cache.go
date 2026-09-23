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

// cacheInfoReport is what `cache info` prints in JSON.
type cacheInfoReport struct {
	Directory string `json:"directory"`
	Entries   int    `json:"entries"`
	Bytes     int64  `json:"bytes"`
	Size      string `json:"size"`
}

func newCacheInfoCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "info",
		Short: "Show cache directory and size",
		Example: `  tba cache info
  tba cache info --format json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := cache.New()
			if err != nil {
				return err
			}
			count, bytes, err := c.Stats()
			if err != nil {
				return err
			}
			report := cacheInfoReport{
				Directory: c.Dir(),
				Entries:   count,
				Bytes:     bytes,
				Size:      cache.FormatSize(bytes),
			}
			return outputData(cmd, report, func() {
				out := cmd.OutOrStdout()
				fmt.Fprintf(out, "Directory: %s\n", report.Directory)
				fmt.Fprintf(out, "Entries:   %d\n", report.Entries)
				fmt.Fprintf(out, "Size:      %s\n", report.Size)
			})
		},
	}
}

// cacheClearReport is what `cache clear` prints in JSON.
type cacheClearReport struct {
	Directory string `json:"directory"`
	Removed   int    `json:"removed"`
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
			report := cacheClearReport{Directory: c.Dir(), Removed: removed}
			return outputData(cmd, report, func() {
				fmt.Fprintf(cmd.OutOrStdout(), "Removed %d cache entries from %s\n",
					report.Removed, report.Directory)
			})
		},
	}
}
