package cmd

import (
	"fmt"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/the-blue-alliance/tba-cli/internal/cache"
	"github.com/the-blue-alliance/tba-cli/internal/clierr"
)

// staleAfter is when a cached entry stops being worth counting as current.
// It is only a reporting threshold: nothing expires on its own.
const staleAfter = 30 * 24 * time.Hour

// defaultPruneAge matches staleAfter, so `cache prune` with no flags removes
// exactly what `cache info` calls stale.
const defaultPruneAge = "30d"

func newCacheCmd() *cobra.Command {
	cacheCmd := &cobra.Command{
		Use:   "cache",
		Short: "Manage the local response cache",
	}
	cacheCmd.AddCommand(newCacheInfoCmd())
	cacheCmd.AddCommand(newCacheListCmd())
	cacheCmd.AddCommand(newCachePruneCmd())
	cacheCmd.AddCommand(newCacheClearCmd())
	return cacheCmd
}

// cacheInfoReport is what `cache info` prints in JSON.
type cacheInfoReport struct {
	Directory       string     `json:"directory"`
	Entries         int        `json:"entries"`
	Bytes           int64      `json:"bytes"`
	Size            string     `json:"size"`
	OldestFetchedAt *time.Time `json:"oldest_fetched_at"`
	NewestFetchedAt *time.Time `json:"newest_fetched_at"`
	StaleCount      int        `json:"stale_count"`
}

func newCacheInfoCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "info",
		Short: "Show the cache directory, size and entry ages",
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
			entries, err := c.List()
			if err != nil {
				return err
			}
			report := cacheInfoReport{
				Directory: c.Dir(),
				Entries:   count,
				Bytes:     bytes,
				Size:      cache.FormatSize(bytes),
			}
			now := time.Now()
			for i, e := range entries {
				if i == 0 || e.FetchedAt.Before(*report.OldestFetchedAt) {
					t := e.FetchedAt
					report.OldestFetchedAt = &t
				}
				if i == 0 || e.FetchedAt.After(*report.NewestFetchedAt) {
					t := e.FetchedAt
					report.NewestFetchedAt = &t
				}
				if now.Sub(e.FetchedAt) > staleAfter {
					report.StaleCount++
				}
			}

			return outputData(cmd, report, func() {
				out := cmd.OutOrStdout()
				fmt.Fprintf(out, "Directory:    %s\n", report.Directory)
				fmt.Fprintf(out, "Entries:      %d\n", report.Entries)
				fmt.Fprintf(out, "Size:         %s\n", report.Size)
				fmt.Fprintf(out, "Oldest:       %s\n", ageOrDash(now, report.OldestFetchedAt))
				fmt.Fprintf(out, "Newest:       %s\n", ageOrDash(now, report.NewestFetchedAt))
				fmt.Fprintf(out, "Stale (>30d): %d\n", report.StaleCount)
			})
		},
	}
}

// ageOrDash renders how long ago t was, or a dash when there is no such entry.
func ageOrDash(now time.Time, t *time.Time) string {
	if t == nil {
		return "-"
	}
	return cache.FormatAge(now.Sub(*t)) + " ago"
}

// cacheListRow is one cached response, as `cache list` reports it.
type cacheListRow struct {
	Path        string     `json:"path"`
	URL         string     `json:"url"`
	FetchedAt   time.Time  `json:"fetched_at"`
	ValidatedAt *time.Time `json:"validated_at,omitempty"`
	Bytes       int64      `json:"bytes"`
	ETag        string     `json:"etag,omitempty"`
}

func newCacheListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List cached responses",
		Example: `  tba cache list
  tba cache list --sort=-size
  tba cache list --columns path,fetched --format json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := cache.New()
			if err != nil {
				return err
			}
			entries, err := c.List()
			if err != nil {
				return err
			}
			now := time.Now()
			rows := make([][]string, 0, len(entries))
			data := make([]cacheListRow, 0, len(entries))
			for _, e := range entries {
				row := cacheListRow{
					Path:      requestPath(e.URL),
					URL:       e.URL,
					FetchedAt: e.FetchedAt,
					Bytes:     e.Size,
					ETag:      e.ETag,
				}
				if !e.ValidatedAt.IsZero() {
					v := e.ValidatedAt
					row.ValidatedAt = &v
				}
				data = append(data, row)
				rows = append(rows, []string{
					row.Path,
					cache.FormatAge(now.Sub(e.FetchedAt)) + " ago",
					ageOrDash(now, row.ValidatedAt),
					cache.FormatSize(e.Size),
					e.ETag,
				})
			}
			// Sorted by path, which is what a human scans for; the URLs the
			// cache sorts by only differ from it by a shared prefix.
			sortRowsByFirstColumn(data, rows)
			return outputTable(cmd, data, []string{"Path", "Fetched", "Validated", "Size", "ETag"}, rows)
		},
	}
}

// sortRowsByFirstColumn keeps the JSON slice and the rendered rows in step
// while ordering both by path.
func sortRowsByFirstColumn(data []cacheListRow, rows [][]string) {
	order := make([]int, len(rows))
	for i := range order {
		order[i] = i
	}
	// A plain insertion sort: caches hold tens of entries, not millions, and
	// this keeps the two slices trivially in step.
	for i := 1; i < len(order); i++ {
		for j := i; j > 0 && data[j].Path < data[j-1].Path; j-- {
			data[j], data[j-1] = data[j-1], data[j]
			rows[j], rows[j-1] = rows[j-1], rows[j]
		}
	}
}

// requestPath reduces a cached URL to the API path it addresses, so the table
// shows "/team/frc177" rather than the whole base URL over and over.
func requestPath(raw string) string {
	u, err := url.Parse(raw)
	if err != nil {
		return raw
	}
	p := strings.TrimPrefix(u.Path, "/api/v3")
	if u.RawQuery != "" {
		p += "?" + u.RawQuery
	}
	if p == "" {
		return "/"
	}
	return p
}

// cachePruneReport is what `cache prune` prints in JSON.
type cachePruneReport struct {
	Removed int      `json:"removed"`
	Bytes   int64    `json:"bytes"`
	DryRun  bool     `json:"dry_run"`
	Paths   []string `json:"paths"`
}

func newCachePruneCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "prune",
		Short: "Remove cache entries older than a given age",
		Long: "Remove cache entries that have not been fetched or revalidated since the given age,\n" +
			"along with any temporary files left behind by an interrupted write.",
		Example: `  tba cache prune
  tba cache prune --older-than 7d
  tba cache prune --older-than 12h --dry-run`,
		RunE: func(cmd *cobra.Command, args []string) error {
			spec, _ := cmd.Flags().GetString("older-than")
			age, err := parseAge(spec)
			if err != nil {
				return err
			}
			dryRun, _ := cmd.Flags().GetBool("dry-run")

			c, err := cache.New()
			if err != nil {
				return err
			}
			res, err := c.Prune(time.Now().Add(-age), dryRun)
			if err != nil {
				return err
			}
			report := cachePruneReport{
				Removed: res.Removed,
				Bytes:   res.Bytes,
				DryRun:  dryRun,
				Paths:   make([]string, 0, len(res.Entries)),
			}
			for _, e := range res.Entries {
				report.Paths = append(report.Paths, requestPath(e.URL))
			}

			return outputData(cmd, report, func() {
				// stdout stays free for data even here, so that
				// `tba cache prune --dry-run` can be eyeballed and piped.
				out := cmd.ErrOrStderr()
				if dryRun {
					for _, p := range report.Paths {
						fmt.Fprintf(out, "would remove %s\n", p)
					}
					fmt.Fprintf(out, "Would remove %d entries (%s)\n", report.Removed, cache.FormatSize(report.Bytes))
					return
				}
				fmt.Fprintf(out, "Removed %d entries (%s)\n", report.Removed, cache.FormatSize(report.Bytes))
			})
		},
	}
	cmd.Flags().String("older-than", defaultPruneAge, "Remove entries untouched for longer than this (e.g. 12h, 30d, 2w)")
	cmd.Flags().Bool("dry-run", false, "List what would be removed without removing anything")
	return cmd
}

// dayWeekPattern matches the durations Go's parser does not know about.
var dayWeekPattern = regexp.MustCompile(`^([0-9]+(?:\.[0-9]+)?)([dw])$`)

// parseAge accepts a Go duration ("90m", "12h") and additionally the day and
// week suffixes people reach for when talking about a cache ("30d", "2w").
func parseAge(s string) (time.Duration, error) {
	trimmed := strings.TrimSpace(s)
	if m := dayWeekPattern.FindStringSubmatch(trimmed); m != nil {
		n, err := strconv.ParseFloat(m[1], 64)
		if err != nil {
			return 0, invalidAge(s)
		}
		hours := 24.0
		if m[2] == "w" {
			hours = 24 * 7
		}
		return time.Duration(n * hours * float64(time.Hour)), nil
	}
	d, err := time.ParseDuration(trimmed)
	if err != nil {
		return 0, invalidAge(s)
	}
	if d < 0 {
		return 0, clierr.Usage("invalid --older-than %q: an age cannot be negative", s)
	}
	return d, nil
}

func invalidAge(s string) error {
	return clierr.Usage("invalid --older-than %q (want a duration like 12h, 30d or 2w)", s)
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
