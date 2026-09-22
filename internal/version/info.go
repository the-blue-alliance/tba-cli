package version

import (
	"runtime"
	"runtime/debug"
	"strings"
)

// devVersion is what Version holds when no release ldflags were applied.
const devVersion = "dev"

// shortCommitLen matches the length goreleaser's .ShortCommit stamps in, so a
// `go install` build and a release build print commits of the same shape.
const shortCommitLen = 7

// Info is the resolved build metadata: the release ldflags where they were
// stamped in, and Go's own build information where they were not.
type Info struct {
	Version string `json:"version"`
	Commit  string `json:"commit"`
	Date    string `json:"date"`
	Go      string `json:"go"`
	OS      string `json:"os"`
	Arch    string `json:"arch"`
}

// readBuildInfo is a seam for tests.
var readBuildInfo = debug.ReadBuildInfo

// Resolve reports what this binary is.
//
// A release binary has its version, commit and date stamped in with -ldflags,
// and those always win. A binary built without them — `go build`, or
// `go install ...@latest` — still knows a good deal about itself through
// runtime/debug: the module version it was installed as, and the VCS revision
// and time when it was built from a checkout. Filling the gaps from there is
// what keeps `tba version` useful for builds that never went through a release.
func Resolve() Info {
	info := Info{
		Version: Version,
		Commit:  Commit,
		Date:    Date,
		Go:      runtime.Version(),
		OS:      runtime.GOOS,
		Arch:    runtime.GOARCH,
	}
	if Version != devVersion {
		return info
	}
	bi, ok := readBuildInfo()
	if !ok {
		return info
	}
	fillFromBuildInfo(&info, bi)
	return info
}

func fillFromBuildInfo(info *Info, bi *debug.BuildInfo) {
	if v := moduleVersion(bi); v != "" {
		info.Version = v
	}

	var revision, vcsTime string
	modified := false
	for _, s := range bi.Settings {
		switch s.Key {
		case "vcs.revision":
			revision = s.Value
		case "vcs.time":
			vcsTime = s.Value
		case "vcs.modified":
			modified = s.Value == "true"
		}
	}
	if info.Commit == "" && revision != "" {
		info.Commit = shortCommit(revision)
		if modified {
			// Same marker `git describe --dirty` uses: the tree had
			// uncommitted changes, so the commit alone does not identify it.
			info.Commit += "-dirty"
		}
	}
	if info.Date == "" {
		info.Date = vcsTime
	}
}

// moduleVersion is the version the main module was built as, for a binary
// installed with `go install module@version`. A build from a checkout reports
// "(devel)", which says nothing the default does not.
func moduleVersion(bi *debug.BuildInfo) string {
	v := bi.Main.Version
	if v == "" || v == "(devel)" {
		return ""
	}
	// Release ldflags stamp "1.2.3"; the module system says "v1.2.3".
	return strings.TrimPrefix(v, "v")
}

func shortCommit(revision string) string {
	if len(revision) > shortCommitLen {
		return revision[:shortCommitLen]
	}
	return revision
}

// Line renders the one-line form printed by `tba version` and `tba --version`:
//
//	tba 1.2.3 (abc1234, built 2024-05-01T00:00:00Z, go1.27.0, darwin/arm64)
//
// The commit and date are left out when they are not known, so a bare
// `go build` still prints something tidy.
func (i Info) Line() string {
	parts := make([]string, 0, 4)
	if i.Commit != "" {
		parts = append(parts, i.Commit)
	}
	if i.Date != "" {
		parts = append(parts, "built "+i.Date)
	}
	parts = append(parts, i.Go, i.OS+"/"+i.Arch)
	return "tba " + i.Version + " (" + strings.Join(parts, ", ") + ")"
}
