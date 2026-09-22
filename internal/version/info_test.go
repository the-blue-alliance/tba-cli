package version

import (
	"runtime"
	"runtime/debug"
	"strings"
	"testing"
)

// stubBuildInfo replaces the runtime/debug lookup for one test.
func stubBuildInfo(t *testing.T, bi *debug.BuildInfo, ok bool) {
	t.Helper()
	old := readBuildInfo
	readBuildInfo = func() (*debug.BuildInfo, bool) { return bi, ok }
	t.Cleanup(func() { readBuildInfo = old })
}

func buildInfo(mainVersion string, settings ...debug.BuildSetting) *debug.BuildInfo {
	bi := &debug.BuildInfo{Settings: settings}
	bi.Main.Version = mainVersion
	return bi
}

func setting(key, value string) debug.BuildSetting {
	return debug.BuildSetting{Key: key, Value: value}
}

func TestResolveKeepsTheStampedReleaseMetadata(t *testing.T) {
	set(t, "1.2.3", "abc1234", "2024-05-01T00:00:00Z")
	// A release binary must not have its stamped values second-guessed, even
	// when the build also carries VCS information.
	stubBuildInfo(t, buildInfo("v9.9.9", setting("vcs.revision", "ffffffffffff")), true)

	got := Resolve()
	if got.Version != "1.2.3" || got.Commit != "abc1234" || got.Date != "2024-05-01T00:00:00Z" {
		t.Errorf("Resolve() = %+v, want the ldflags values", got)
	}
}

func TestResolveReportsTheRuntime(t *testing.T) {
	got := Resolve()
	if got.Go != runtime.Version() {
		t.Errorf("Go = %q, want %q", got.Go, runtime.Version())
	}
	if got.OS != runtime.GOOS || got.Arch != runtime.GOARCH {
		t.Errorf("OS/Arch = %q/%q, want %q/%q", got.OS, got.Arch, runtime.GOOS, runtime.GOARCH)
	}
}

func TestResolveFallsBackToVCSInfoForADevBuild(t *testing.T) {
	set(t, "dev", "", "")
	stubBuildInfo(t, buildInfo("(devel)",
		setting("vcs.revision", "0123456789abcdef"),
		setting("vcs.time", "2024-06-02T03:04:05Z"),
		setting("vcs.modified", "false"),
	), true)

	got := Resolve()
	if got.Version != "dev" {
		t.Errorf("Version = %q, want dev", got.Version)
	}
	if got.Commit != "0123456" {
		t.Errorf("Commit = %q, want the short revision 0123456", got.Commit)
	}
	if got.Date != "2024-06-02T03:04:05Z" {
		t.Errorf("Date = %q", got.Date)
	}
}

func TestResolveMarksADirtyTree(t *testing.T) {
	set(t, "dev", "", "")
	stubBuildInfo(t, buildInfo("(devel)",
		setting("vcs.revision", "0123456789abcdef"),
		setting("vcs.modified", "true"),
	), true)

	if got := Resolve().Commit; got != "0123456-dirty" {
		t.Errorf("Commit = %q, want 0123456-dirty", got)
	}
}

func TestResolveUsesTheModuleVersionFromGoInstall(t *testing.T) {
	set(t, "dev", "", "")
	// `go install ...@v1.4.0` records the module version and no VCS settings.
	stubBuildInfo(t, buildInfo("v1.4.0"), true)

	got := Resolve()
	if got.Version != "1.4.0" {
		t.Errorf("Version = %q, want 1.4.0 (the leading v trimmed)", got.Version)
	}
	if got.Commit != "" || got.Date != "" {
		t.Errorf("Commit/Date = %q/%q, want empty when the build knows neither", got.Commit, got.Date)
	}
}

func TestResolveSurvivesMissingBuildInfo(t *testing.T) {
	set(t, "dev", "", "")
	stubBuildInfo(t, nil, false)

	got := Resolve()
	if got.Version != "dev" || got.Commit != "" || got.Date != "" {
		t.Errorf("Resolve() = %+v, want a bare dev build", got)
	}
}

func TestResolveKeepsAShortRevisionWhole(t *testing.T) {
	set(t, "dev", "", "")
	stubBuildInfo(t, buildInfo("(devel)", setting("vcs.revision", "abc12")), true)

	if got := Resolve().Commit; got != "abc12" {
		t.Errorf("Commit = %q, want abc12", got)
	}
}

func TestLineRendersEverythingItKnows(t *testing.T) {
	info := Info{Version: "1.2.3", Commit: "abc1234", Date: "2024-05-01T00:00:00Z", Go: "go1.27.0", OS: "darwin", Arch: "arm64"}
	want := "tba 1.2.3 (abc1234, built 2024-05-01T00:00:00Z, go1.27.0, darwin/arm64)"
	if got := info.Line(); got != want {
		t.Errorf("Line() =\n%q\nwant\n%q", got, want)
	}
}

func TestLineOmitsAnUnknownCommitAndDate(t *testing.T) {
	info := Info{Version: "dev", Go: "go1.27.0", OS: "linux", Arch: "amd64"}
	want := "tba dev (go1.27.0, linux/amd64)"
	if got := info.Line(); got != want {
		t.Errorf("Line() = %q, want %q", got, want)
	}
}

func TestLineOmitsOnlyTheMissingPiece(t *testing.T) {
	info := Info{Version: "dev", Commit: "abc1234", Go: "go1.27.0", OS: "linux", Arch: "amd64"}
	got := info.Line()
	if !strings.Contains(got, "abc1234") {
		t.Errorf("Line() = %q, want it to carry the commit", got)
	}
	if strings.Contains(got, "built") {
		t.Errorf("Line() = %q, want no build date when there is none", got)
	}
}
