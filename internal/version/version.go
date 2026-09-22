// Package version holds the build metadata stamped into the binary at release
// time. The defaults are what you get from `go build` without ldflags.
package version

import "strings"

// These are overridden at build time with -ldflags -X.
var (
	// Version is the release version, e.g. "1.2.3", or "dev" for a local build.
	Version = "dev"
	// Commit is the git SHA the binary was built from.
	Commit = ""
	// Date is the build timestamp, in RFC 3339.
	Date = ""
)

// String renders the full version for humans: the version on its own for a
// bare build, with the commit and build date appended when they are known.
func String() string {
	var b strings.Builder
	b.WriteString(Version)
	var extra []string
	if Commit != "" {
		extra = append(extra, Commit)
	}
	if Date != "" {
		extra = append(extra, Date)
	}
	if len(extra) > 0 {
		b.WriteString(" (")
		b.WriteString(strings.Join(extra, ", "))
		b.WriteString(")")
	}
	return b.String()
}

// UserAgent is the User-Agent header value the CLI sends.
func UserAgent() string { return "tba-cli/" + Version }
