package cmd

import (
	"encoding/json"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"testing"

	"github.com/the-blue-alliance/tba-cli/internal/clierr"
)

// exportRoutes is a complete event: every dataset `event export` knows about
// answers, so a test can assert on the whole export at once.
func exportRoutes() map[string]any {
	return map[string]any{
		"/event/2024cthar":                 event2024ctharJSON,
		"/event/2024cthar/teams":           teamsSimple2024ctharJSON,
		"/event/2024cthar/teams/simple":    teamsSimple2024ctharJSON,
		"/event/2024cthar/matches":         matches2024ctharJSON,
		"/event/2024cthar/rankings":        rankings2024ctharJSON,
		"/event/2024cthar/alliances":       alliances2024ctharJSON,
		"/event/2024cthar/awards":          awards2024ctharJSON,
		"/event/2024cthar/oprs":            oprs2024ctharJSON,
		"/event/2024cthar/district_points": districtPoints2024ctharJSON,
		"/event/2024cthar/teams/statuses":  teamStatuses2024ctharJSON,
	}
}

func newExportServer(t *testing.T) *httptest.Server {
	t.Helper()
	return newFakeTBA(t, exportRoutes())
}

// exportedPath is the file one dataset lands in, spelled the way the command
// spells it.
func exportedPath(dir, dataset, ext string) string {
	return filepath.Join(dir, "2024cthar-"+dataset+"."+ext)
}

func readExported(t *testing.T, dir, dataset, ext string) string {
	t.Helper()
	body, err := os.ReadFile(exportedPath(dir, dataset, ext))
	if err != nil {
		t.Fatalf("reading the %s export: %v", dataset, err)
	}
	return string(body)
}

// dirEntries lists a directory, including the dot-files a half-finished export
// would leave behind.
func dirEntries(t *testing.T, dir string) []string {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("reading %s: %v", dir, err)
	}
	names := make([]string, 0, len(entries))
	for _, e := range entries {
		names = append(names, e.Name())
	}
	return names
}

func TestEventExportRequiresTo(t *testing.T) {
	srv := newExportServer(t)
	dir := t.TempDir()
	_, _, err := runCmd(t, srv, "event", "export", "2024cthar", "--dir", dir)

	requireErrorContains(t, err, "--to is required")
	for _, want := range []string{"csv", "tsv", "json"} {
		requireErrorContains(t, err, want)
	}
	if code := clierr.ExitCode(err); code != clierr.ExitUsage {
		t.Errorf("exit code = %d, want %d", code, clierr.ExitUsage)
	}
	if got := requestPaths(t, srv); len(got) != 0 {
		t.Errorf("a missing --to should not reach the API, got %v", got)
	}
	if got := dirEntries(t, dir); len(got) != 0 {
		t.Errorf("directory = %v, want nothing written", got)
	}
}

func TestEventExportRejectsAnUnknownTo(t *testing.T) {
	srv := newExportServer(t)
	_, _, err := runCmd(t, srv, "event", "export", "2024cthar", "--to", "xlsx", "--dir", t.TempDir())

	requireErrorContains(t, err, `invalid --to "xlsx"`)
	requireErrorContains(t, err, "csv, tsv, json")
	if code := clierr.ExitCode(err); code != clierr.ExitUsage {
		t.Errorf("exit code = %d, want %d", code, clierr.ExitUsage)
	}
	if got := requestPaths(t, srv); len(got) != 0 {
		t.Errorf("a bad --to should not reach the API, got %v", got)
	}
}

func TestEventExportFormatDoesNotChooseTheExportFormat(t *testing.T) {
	for _, format := range []string{"csv", "tsv", "markdown", "table"} {
		t.Run(format, func(t *testing.T) {
			srv := newExportServer(t)
			_, _, err := runCmd(t, srv, "event", "export", "2024cthar",
				"--to", "json", "--format", format, "--dir", t.TempDir())

			requireErrorContains(t, err, "--to")
			requireErrorContains(t, err, "does not choose the export format")
			if code := clierr.ExitCode(err); code != clierr.ExitUsage {
				t.Errorf("exit code = %d, want %d", code, clierr.ExitUsage)
			}
			if got := requestPaths(t, srv); len(got) != 0 {
				t.Errorf("a usage error should not reach the API, got %v", got)
			}
		})
	}
}

func TestEventExportWritesOneFilePerDatasetInEveryFormat(t *testing.T) {
	for _, ext := range []string{"csv", "tsv", "json"} {
		t.Run(ext, func(t *testing.T) {
			srv := newExportServer(t)
			dir := t.TempDir()
			_, stderr, err := runCmd(t, srv, "event", "export", "2024cthar", "--to", ext, "--dir", dir)
			requireNoError(t, err, stderr)

			var want []string
			for _, name := range exportDatasetNames {
				want = append(want, "2024cthar-"+name+"."+ext)
			}
			// os.ReadDir sorts; the order the files were written in is
			// asserted on stdout instead.
			sort.Strings(want)
			got := dirEntries(t, dir)
			if strings.Join(got, ",") != strings.Join(want, ",") {
				t.Errorf("files =\n%v\nwant\n%v", got, want)
			}
			requireContains(t, stderr, "wrote 9 file(s)")
		})
	}
}

// The csv and tsv files are the command's own tables: anything else would mean
// two renderings of the same data to keep in step.
func TestEventExportFilesMatchTheEventCommandTables(t *testing.T) {
	cases := []struct{ dataset, command string }{
		{"teams", "teams"},
		{"matches", "matches"},
		{"rankings", "rankings"},
		{"alliances", "alliances"},
		{"awards", "awards"},
		{"oprs", "oprs"},
		{"district-points", "district-points"},
		{"team-statuses", "team-statuses"},
	}
	for _, format := range []string{"csv", "tsv"} {
		for _, tc := range cases {
			t.Run(format+"/"+tc.dataset, func(t *testing.T) {
				srv := newExportServer(t)
				dir := t.TempDir()
				_, stderr, err := runCmd(t, srv, "event", "export", "2024cthar", "--to", format, "--dir", dir)
				requireNoError(t, err, stderr)

				want, stderr, err := runCmd(t, srv, "event", tc.command, "2024cthar", "--format", format)
				requireNoError(t, err, stderr)

				if got := readExported(t, dir, tc.dataset, format); got != want {
					t.Errorf("%s export =\n%q\nwant the `tba event %s` table\n%q", tc.dataset, got, tc.command, want)
				}
			})
		}
	}
}

func TestEventExportCSVHeaderIsTheMatchesTableHeader(t *testing.T) {
	srv := newExportServer(t)
	dir := t.TempDir()
	_, stderr, err := runCmd(t, srv, "event", "export", "2024cthar", "--to", "csv", "--dir", dir)
	requireNoError(t, err, stderr)

	table, stderr, err := runCmd(t, srv, "event", "matches", "2024cthar", "--format", "csv")
	requireNoError(t, err, stderr)

	got := lines(readExported(t, dir, "matches", "csv"))[0]
	if want := lines(table)[0]; got != want {
		t.Errorf("header = %q, want %q", got, want)
	}
	requireContains(t, got, "Match,Key,Red,Blue")
}

func TestEventExportTSVIsTabSeparated(t *testing.T) {
	srv := newExportServer(t)
	dir := t.TempDir()
	_, stderr, err := runCmd(t, srv, "event", "export", "2024cthar", "--to", "tsv", "--dir", dir)
	requireNoError(t, err, stderr)

	header := lines(readExported(t, dir, "oprs", "tsv"))[0]
	if header != "Team\tOPR\tDPR\tCCWM" {
		t.Errorf("header = %q", header)
	}
}

func TestEventExportJSONHoldsTheRawPayload(t *testing.T) {
	srv := newExportServer(t)
	dir := t.TempDir()
	_, stderr, err := runCmd(t, srv, "event", "export", "2024cthar", "--to", "json", "--dir", dir)
	requireNoError(t, err, stderr)

	for dataset, path := range map[string]string{
		"event":           "/event/2024cthar",
		"matches":         "/event/2024cthar/matches",
		"district-points": "/event/2024cthar/district_points",
	} {
		body := readExported(t, dir, dataset, "json")
		var got, want any
		if err := json.Unmarshal([]byte(body), &got); err != nil {
			t.Fatalf("%s export is not JSON: %v", dataset, err)
		}
		if err := json.Unmarshal([]byte(exportRoutes()[path].(string)), &want); err != nil {
			t.Fatalf("fixture for %s is not JSON: %v", path, err)
		}
		if !jsonEqual(got, want) {
			t.Errorf("%s export is not the API payload:\n%s", dataset, body)
		}
		if !strings.HasSuffix(body, "}\n") && !strings.HasSuffix(body, "]\n") {
			t.Errorf("%s export does not end with one newline: %q", dataset, tail(body))
		}
		if strings.Contains(body, "\r") {
			t.Errorf("%s export has CRLF line endings", dataset)
		}
	}
}

// json.Indent, not a re-encode: the file should carry the characters the API
// sent rather than Go's HTML-escaped spelling of them.
func TestEventExportJSONDoesNotReEscapeThePayload(t *testing.T) {
	srv := newExportServer(t)
	dir := t.TempDir()
	_, stderr, err := runCmd(t, srv, "event", "export", "2024cthar", "--to", "json", "--dir", dir, "--only", "teams")
	requireNoError(t, err, stderr)

	body := readExported(t, dir, "teams", "json")
	requireContains(t, body, "Gordon & Llura Gund Foundation")
	if strings.Contains(body, `\u0026`) {
		t.Errorf("export re-escaped an ampersand:\n%s", body)
	}
}

func TestEventExportEventIsAKeyValueTable(t *testing.T) {
	srv := newExportServer(t)
	dir := t.TempDir()
	_, stderr, err := runCmd(t, srv, "event", "export", "2024cthar", "--to", "csv", "--dir", dir, "--only", "event")
	requireNoError(t, err, stderr)

	got := lines(readExported(t, dir, "event", "csv"))
	if got[0] != "Field,Value" {
		t.Errorf("header = %q, want Field,Value", got[0])
	}
	for _, want := range []string{
		"Event,NE District Hartford Event",
		"Key,2024cthar",
		"Week,4",
	} {
		if !contains(got, want) {
			t.Errorf("event export is missing %q:\n%s", want, strings.Join(got, "\n"))
		}
	}
}

func TestEventExportOnlySelectsASubset(t *testing.T) {
	srv := newExportServer(t)
	dir := t.TempDir()
	stdout, stderr, err := runCmd(t, srv, "event", "export", "2024cthar",
		"--to", "csv", "--dir", dir, "--only", "rankings,matches")
	requireNoError(t, err, stderr)

	want := []string{"2024cthar-matches.csv", "2024cthar-rankings.csv"}
	if got := dirEntries(t, dir); strings.Join(got, ",") != strings.Join(want, ",") {
		t.Errorf("files = %v, want %v", got, want)
	}
	// The files come out in the fixed dataset order, not the order --only
	// happened to list them in.
	wantOut := exportedPath(dir, "matches", "csv") + "\n" + exportedPath(dir, "rankings", "csv") + "\n"
	if stdout != wantOut {
		t.Errorf("stdout = %q, want %q", stdout, wantOut)
	}
	for _, unwanted := range []string{"/event/2024cthar/awards", "/event/2024cthar/oprs"} {
		if contains(requestPaths(t, srv), unwanted) {
			t.Errorf("--only should not have fetched %s", unwanted)
		}
	}
}

func TestEventExportRejectsAnUnknownDataset(t *testing.T) {
	srv := newExportServer(t)
	dir := t.TempDir()
	_, _, err := runCmd(t, srv, "event", "export", "2024cthar",
		"--to", "csv", "--dir", dir, "--only", "matches,scouting")

	requireErrorContains(t, err, `unknown dataset "scouting"`)
	for _, name := range exportDatasetNames {
		requireErrorContains(t, err, name)
	}
	if code := clierr.ExitCode(err); code != clierr.ExitUsage {
		t.Errorf("exit code = %d, want %d", code, clierr.ExitUsage)
	}
	if got := dirEntries(t, dir); len(got) != 0 {
		t.Errorf("directory = %v, want nothing written", got)
	}
}

func TestEventExportDryRunFetchesNothingAndWritesNothing(t *testing.T) {
	srv := newExportServer(t)
	dir := t.TempDir()
	stdout, stderr, err := runCmd(t, srv, "event", "export", "2024cthar",
		"--to", "csv", "--dir", dir, "--dry-run")
	requireNoError(t, err, stderr)

	if got := requestPaths(t, srv); len(got) != 0 {
		t.Errorf("a dry run should make no requests, got %v", got)
	}
	if got := dirEntries(t, dir); len(got) != 0 {
		t.Errorf("a dry run should write nothing, got %v", got)
	}
	if got := lines(stdout); len(got) != len(exportDatasetNames) {
		t.Errorf("stdout has %d lines, want %d:\n%s", len(got), len(exportDatasetNames), stdout)
	}
	requireContains(t, stdout, exportedPath(dir, "matches", "csv"))
	requireContains(t, stderr, "dry run: would write 9 file(s)")
}

func TestEventExportDryRunJSONSaysSo(t *testing.T) {
	srv := newExportServer(t)
	stdout, stderr, err := runCmd(t, srv, "event", "export", "2024cthar",
		"--to", "csv", "--dir", t.TempDir(), "--dry-run", "--json")
	requireNoError(t, err, stderr)

	obj := decodeJSON(t, stdout).(map[string]any)
	if obj["dry_run"] != true {
		t.Errorf("dry_run = %v, want true", obj["dry_run"])
	}
}

func TestEventExportRefusesToOverwriteWithoutForce(t *testing.T) {
	srv := newExportServer(t)
	dir := t.TempDir()
	existing := exportedPath(dir, "matches", "csv")
	if err := os.WriteFile(existing, []byte("mine\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	_, _, err := runCmd(t, srv, "event", "export", "2024cthar", "--to", "csv", "--dir", dir)
	requireErrorContains(t, err, "refusing to overwrite")
	requireErrorContains(t, err, existing)
	requireErrorContains(t, err, "--force")
	if code := clierr.ExitCode(err); code != clierr.ExitFailure {
		t.Errorf("exit code = %d, want %d", code, clierr.ExitFailure)
	}
	if got := requestPaths(t, srv); len(got) != 0 {
		t.Errorf("a clash should be found before any request, got %v", got)
	}
	// The clash is reported before anything is written, so the one file that
	// was there is still the only one.
	if got := dirEntries(t, dir); len(got) != 1 {
		t.Errorf("directory = %v, want only the pre-existing file", got)
	}
	if body, _ := os.ReadFile(existing); string(body) != "mine\n" {
		t.Errorf("the existing file was modified: %q", body)
	}
}

func TestEventExportOverwriteErrorListsEveryClash(t *testing.T) {
	srv := newExportServer(t)
	dir := t.TempDir()
	for _, dataset := range []string{"matches", "oprs"} {
		if err := os.WriteFile(exportedPath(dir, dataset, "csv"), []byte("x\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	_, _, err := runCmd(t, srv, "event", "export", "2024cthar", "--to", "csv", "--dir", dir)
	requireErrorContains(t, err, exportedPath(dir, "matches", "csv"))
	requireErrorContains(t, err, exportedPath(dir, "oprs", "csv"))
}

func TestEventExportForceOverwrites(t *testing.T) {
	srv := newExportServer(t)
	dir := t.TempDir()
	existing := exportedPath(dir, "matches", "csv")
	if err := os.WriteFile(existing, []byte("stale\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	_, stderr, err := runCmd(t, srv, "event", "export", "2024cthar", "--to", "csv", "--dir", dir, "--force")
	requireNoError(t, err, stderr)

	body := readExported(t, dir, "matches", "csv")
	if strings.Contains(body, "stale") {
		t.Errorf("--force did not replace the file:\n%s", body)
	}
	requireContains(t, body, "2024cthar_qm1")
}

func TestEventExportSkipsADatasetTheEventDoesNotHave(t *testing.T) {
	routes := exportRoutes()
	delete(routes, "/event/2024cthar/district_points")
	srv := newFakeTBA(t, routes)
	dir := t.TempDir()

	stdout, stderr, err := runCmd(t, srv, "event", "export", "2024cthar", "--to", "csv", "--dir", dir)
	requireNoError(t, err, stderr)

	requireContains(t, stderr, "note: skipped district-points")
	requireContains(t, stderr, "404")
	requireContains(t, stderr, "wrote 8 file(s)")
	if contains(dirEntries(t, dir), "2024cthar-district-points.csv") {
		t.Errorf("a skipped dataset should leave no file: %v", dirEntries(t, dir))
	}
	if strings.Contains(stdout, "district-points") {
		t.Errorf("a skipped dataset should not be on stdout:\n%s", stdout)
	}
	// Everything else is still there: a missing dataset is not a failure.
	if got := len(dirEntries(t, dir)); got != len(exportDatasetNames)-1 {
		t.Errorf("wrote %d files, want %d", got, len(exportDatasetNames)-1)
	}
}

func TestEventExportSkipIsInTheJSONSummary(t *testing.T) {
	routes := exportRoutes()
	delete(routes, "/event/2024cthar/alliances")
	srv := newFakeTBA(t, routes)

	stdout, stderr, err := runCmd(t, srv, "event", "export", "2024cthar",
		"--to", "csv", "--dir", t.TempDir(), "--json")
	requireNoError(t, err, stderr)

	obj := decodeJSON(t, stdout).(map[string]any)
	skipped, ok := obj["skipped"].([]any)
	if !ok || len(skipped) != 1 {
		t.Fatalf("skipped = %v, want one entry", obj["skipped"])
	}
	entry := skipped[0].(map[string]any)
	if entry["dataset"] != "alliances" {
		t.Errorf("dataset = %v, want alliances", entry["dataset"])
	}
	if reason, _ := entry["reason"].(string); !strings.Contains(reason, "404") {
		t.Errorf("reason = %v, want it to mention the 404", entry["reason"])
	}
}

func TestEventExportLeavesNoFilesBehindWhenADatasetFails(t *testing.T) {
	srv := newExportServer(t)
	setStatus(t, srv, "/event/2024cthar/oprs", 500)
	dir := t.TempDir()
	// A file that was already there must survive a failed export untouched.
	keep := filepath.Join(dir, "notes.txt")
	if err := os.WriteFile(keep, []byte("keep me\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	_, _, err := runCmd(t, srv, "event", "export", "2024cthar",
		"--to", "csv", "--dir", dir, "--retries", "0")
	if err == nil {
		t.Fatal("a 500 should fail the export")
	}
	if code := clierr.ExitCode(err); code != clierr.ExitFailure {
		t.Errorf("exit code = %d, want %d", code, clierr.ExitFailure)
	}
	got := dirEntries(t, dir)
	if len(got) != 1 || got[0] != "notes.txt" {
		t.Errorf("directory = %v, want only the pre-existing notes.txt", got)
	}
	if body, _ := os.ReadFile(keep); string(body) != "keep me\n" {
		t.Errorf("the pre-existing file changed: %q", body)
	}
}

func TestEventExportIsReproducible(t *testing.T) {
	srv := newExportServer(t)
	first, second := t.TempDir(), t.TempDir()
	for _, dir := range []string{first, second} {
		_, stderr, err := runCmd(t, srv, "event", "export", "2024cthar", "--to", "csv", "--dir", dir)
		requireNoError(t, err, stderr)
	}
	for _, name := range exportDatasetNames {
		a, err := os.ReadFile(exportedPath(first, name, "csv"))
		if err != nil {
			t.Fatal(err)
		}
		b, err := os.ReadFile(exportedPath(second, name, "csv"))
		if err != nil {
			t.Fatal(err)
		}
		if string(a) != string(b) {
			t.Errorf("%s differs between two exports:\n%q\n%q", name, a, b)
		}
	}
}

func TestEventExportWritesLFLineEndings(t *testing.T) {
	srv := newExportServer(t)
	dir := t.TempDir()
	_, stderr, err := runCmd(t, srv, "event", "export", "2024cthar", "--to", "csv", "--dir", dir)
	requireNoError(t, err, stderr)

	for _, name := range exportDatasetNames {
		body := readExported(t, dir, name, "csv")
		if strings.Contains(body, "\r") {
			t.Errorf("%s has a carriage return in it", name)
		}
		if !strings.HasSuffix(body, "\n") {
			t.Errorf("%s does not end with a newline", name)
		}
	}
}

func TestEventExportStdoutListsExactlyTheWrittenPaths(t *testing.T) {
	srv := newExportServer(t)
	dir := t.TempDir()
	stdout, stderr, err := runCmd(t, srv, "event", "export", "2024cthar", "--to", "tsv", "--dir", dir)
	requireNoError(t, err, stderr)

	var want []string
	for _, name := range exportDatasetNames {
		want = append(want, exportedPath(dir, name, "tsv"))
	}
	if got := lines(stdout); strings.Join(got, "\n") != strings.Join(want, "\n") {
		t.Errorf("stdout =\n%s\nwant\n%s", stdout, strings.Join(want, "\n"))
	}
	// Notes and the summary are the user's business, not the pipeline's.
	if strings.Contains(stdout, "wrote") {
		t.Errorf("the summary belongs on stderr:\n%s", stdout)
	}
	requireContains(t, stderr, "wrote 9 file(s)")
}

func TestEventExportPathsAreRelativeToTheDirAsGiven(t *testing.T) {
	srv := newExportServer(t)
	t.Chdir(t.TempDir())

	stdout, stderr, err := runCmd(t, srv, "event", "export", "2024cthar", "--to", "csv", "--only", "oprs")
	requireNoError(t, err, stderr)
	if stdout != "2024cthar-oprs.csv\n" {
		t.Errorf("stdout = %q, want the bare relative path", stdout)
	}
	if _, err := os.Stat("2024cthar-oprs.csv"); err != nil {
		t.Errorf("the default --dir should be the working directory: %v", err)
	}
}

func TestEventExportJSONSummaryShape(t *testing.T) {
	srv := newExportServer(t)
	dir := t.TempDir()
	stdout, stderr, err := runCmd(t, srv, "event", "export", "2024cthar",
		"--to", "csv", "--dir", dir, "--format", "json")
	requireNoError(t, err, stderr)

	obj := decodeJSON(t, stdout).(map[string]any)
	written, ok := obj["written"].([]any)
	if !ok {
		t.Fatalf("written = %v, want an array", obj["written"])
	}
	if len(written) != len(exportDatasetNames) {
		t.Errorf("written has %d entries, want %d", len(written), len(exportDatasetNames))
	}
	if written[0] != exportedPath(dir, "event", "csv") {
		t.Errorf("written[0] = %v", written[0])
	}
	if skipped, ok := obj["skipped"].([]any); !ok || len(skipped) != 0 {
		t.Errorf("skipped = %v, want an empty array", obj["skipped"])
	}
	if _, present := obj["dry_run"]; present {
		t.Errorf("a real export should not carry dry_run: %v", obj["dry_run"])
	}
}

// TBA_FORMAT is the environment's spelling of --format, so it selects the
// JSON summary exactly as the flag does. Reading the flag alone meant an
// exported TBA_FORMAT=json quietly did nothing here and everything elsewhere.
func TestEventExportFormatComesFromTheEnvironment(t *testing.T) {
	srv := newExportServer(t)
	t.Setenv("TBA_FORMAT", "json")

	stdout, stderr, err := runCmd(t, srv, "event", "export", "2024cthar",
		"--to", "csv", "--dir", t.TempDir(), "--only", "oprs")
	requireNoError(t, err, stderr)

	obj := decodeJSON(t, stdout).(map[string]any)
	if _, ok := obj["written"].([]any); !ok {
		t.Errorf("stdout = %q, want the JSON summary", stdout)
	}
}

func TestEventExportFormatComesFromTheConfigFile(t *testing.T) {
	srv := newExportServer(t)
	writeConfig(t, "format: json\n")

	stdout, stderr, err := runCmd(t, srv, "event", "export", "2024cthar",
		"--to", "csv", "--dir", t.TempDir(), "--only", "oprs")
	requireNoError(t, err, stderr)

	obj := decodeJSON(t, stdout).(map[string]any)
	if _, ok := obj["written"].([]any); !ok {
		t.Errorf("stdout = %q, want the JSON summary", stdout)
	}
}

// A format that cannot select the export format is refused wherever it was
// set, and the message says where that was.
func TestEventExportRejectsANonJSONFormatFromTheEnvironment(t *testing.T) {
	srv := newExportServer(t)
	t.Setenv("TBA_FORMAT", "csv")

	_, _, err := runCmd(t, srv, "event", "export", "2024cthar", "--to", "json", "--dir", t.TempDir())
	requireErrorContains(t, err, "does not choose the export format")
	requireErrorContains(t, err, "TBA_FORMAT")
	if code := clierr.ExitCode(err); code != clierr.ExitUsage {
		t.Errorf("exit code = %d, want %d", code, clierr.ExitUsage)
	}
	if got := requestPaths(t, srv); len(got) != 0 {
		t.Errorf("a usage error should not reach the API, got %v", got)
	}
}

func TestEventExportRejectsANonJSONFormatFromTheConfigFile(t *testing.T) {
	srv := newExportServer(t)
	path := writeConfig(t, "format: table\n")

	_, _, err := runCmd(t, srv, "event", "export", "2024cthar", "--to", "json", "--dir", t.TempDir())
	requireErrorContains(t, err, "does not choose the export format")
	requireErrorContains(t, err, path)
}

// `auto` is the default and means "decide from the terminal" everywhere else.
// Here there is nothing to decide: a piped export is a list of paths to feed
// to something, and turning it into JSON would break the pipeline it was
// written for.
func TestEventExportAutoStaysAListOfPathsWhenPiped(t *testing.T) {
	for _, setup := range []struct {
		name string
		env  string
	}{
		{"flag", ""},
		{"env", "auto"},
	} {
		t.Run(setup.name, func(t *testing.T) {
			srv := newExportServer(t)
			dir := t.TempDir()
			args := []string{"event", "export", "2024cthar", "--to", "csv", "--dir", dir, "--only", "oprs"}
			if setup.env != "" {
				t.Setenv("TBA_FORMAT", setup.env)
			} else {
				args = append(args, "--format", "auto")
			}
			stdout, stderr, err := runCmd(t, srv, args...)
			requireNoError(t, err, stderr)
			if stdout != exportedPath(dir, "oprs", "csv")+"\n" {
				t.Errorf("stdout = %q, want the bare path", stdout)
			}
		})
	}
}

func TestEventExportRejectsAFormatThatIsNotAFormat(t *testing.T) {
	srv := newExportServer(t)
	t.Setenv("TBA_FORMAT", "xlsx")

	_, _, err := runCmd(t, srv, "event", "export", "2024cthar", "--to", "csv", "--dir", t.TempDir())
	requireErrorContains(t, err, `invalid TBA_FORMAT "xlsx"`)
	if code := clierr.ExitCode(err); code != clierr.ExitUsage {
		t.Errorf("exit code = %d, want %d", code, clierr.ExitUsage)
	}
}

func TestEventExportJSONSummaryTakesAJqExpression(t *testing.T) {
	srv := newExportServer(t)
	stdout, stderr, err := runCmd(t, srv, "event", "export", "2024cthar",
		"--to", "csv", "--dir", t.TempDir(), "--jq", ".written | length")
	requireNoError(t, err, stderr)
	if strings.TrimSpace(stdout) != "9" {
		t.Errorf("stdout = %q, want 9", stdout)
	}
}

func TestEventExportPrefixNamesTheFiles(t *testing.T) {
	srv := newExportServer(t)
	dir := t.TempDir()
	_, stderr, err := runCmd(t, srv, "event", "export", "2024cthar",
		"--to", "csv", "--dir", dir, "--only", "oprs", "--prefix", "hartford")
	requireNoError(t, err, stderr)

	if got := dirEntries(t, dir); len(got) != 1 || got[0] != "hartford-oprs.csv" {
		t.Errorf("files = %v, want hartford-oprs.csv", got)
	}
}

func TestEventExportRejectsAPrefixWithAPathSeparator(t *testing.T) {
	srv := newExportServer(t)
	_, _, err := runCmd(t, srv, "event", "export", "2024cthar",
		"--to", "csv", "--dir", t.TempDir(), "--prefix", "sub/dir")
	requireErrorContains(t, err, "--prefix names a file")
	if code := clierr.ExitCode(err); code != clierr.ExitUsage {
		t.Errorf("exit code = %d, want %d", code, clierr.ExitUsage)
	}
}

func TestEventExportCreatesTheDirectory(t *testing.T) {
	srv := newExportServer(t)
	dir := filepath.Join(t.TempDir(), "exports", "2024")
	_, stderr, err := runCmd(t, srv, "event", "export", "2024cthar",
		"--to", "json", "--dir", dir, "--only", "event")
	requireNoError(t, err, stderr)

	if got := dirEntries(t, dir); len(got) != 1 || got[0] != "2024cthar-event.json" {
		t.Errorf("files = %v", got)
	}
}

func TestEventExportRejectsAnEmptyDir(t *testing.T) {
	srv := newExportServer(t)
	_, _, err := runCmd(t, srv, "event", "export", "2024cthar", "--to", "csv", "--dir", "")
	requireErrorContains(t, err, "--dir cannot be empty")
}

func TestEventExportFilesAreReadable(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Windows does not carry Unix file modes")
	}
	srv := newExportServer(t)
	dir := t.TempDir()
	_, stderr, err := runCmd(t, srv, "event", "export", "2024cthar", "--to", "csv", "--dir", dir, "--only", "oprs")
	requireNoError(t, err, stderr)

	info, err := os.Stat(exportedPath(dir, "oprs", "csv"))
	if err != nil {
		t.Fatal(err)
	}
	if got := info.Mode().Perm(); got != exportFileMode {
		t.Errorf("mode = %04o, want %04o", got, exportFileMode)
	}
}

func TestEventExportSortsTeamsByNumber(t *testing.T) {
	routes := exportRoutes()
	routes["/event/2024cthar/teams"] = "[" + teamFRC5507JSON + "," + teamFRC1073JSON + "," + teamFRC177JSON + "]"
	srv := newFakeTBA(t, routes)
	dir := t.TempDir()

	_, stderr, err := runCmd(t, srv, "event", "export", "2024cthar",
		"--to", "csv", "--dir", dir, "--only", "teams")
	requireNoError(t, err, stderr)

	got := lines(readExported(t, dir, "teams", "csv"))
	if len(got) != 4 {
		t.Fatalf("want a header and three teams, got %v", got)
	}
	for i, want := range []string{"177,", "1073,", "5507,"} {
		if !strings.HasPrefix(got[i+1], want) {
			t.Errorf("row %d = %q, want it to start with %q", i+1, got[i+1], want)
		}
	}
}

func TestEventExportSortsAwardsByTypeThenTeam(t *testing.T) {
	routes := exportRoutes()
	routes["/event/2024cthar/awards"] = `[
	  {"name": "Dean's List Finalist Award", "award_type": 4, "recipient_list": [{"team_key": "frc1073", "awardee": "Ada Lovelace"}]},
	  {"name": "Dean's List Finalist Award", "award_type": 4, "recipient_list": [{"team_key": "frc177", "awardee": "Grace Hopper"}]},
	  {"name": "District Event Winner", "award_type": 1, "recipient_list": [{"team_key": "frc230", "awardee": null}]}
	]`
	srv := newFakeTBA(t, routes)
	dir := t.TempDir()

	_, stderr, err := runCmd(t, srv, "event", "export", "2024cthar",
		"--to", "csv", "--dir", dir, "--only", "awards")
	requireNoError(t, err, stderr)

	got := lines(readExported(t, dir, "awards", "csv"))[1:]
	want := []string{
		"District Event Winner,230",
		"Dean's List Finalist Award,177 - Grace Hopper",
		"Dean's List Finalist Award,1073 - Ada Lovelace",
	}
	if strings.Join(got, "\n") != strings.Join(want, "\n") {
		t.Errorf("awards =\n%s\nwant\n%s", strings.Join(got, "\n"), strings.Join(want, "\n"))
	}
}

func TestEventExportMatchesAreInPlayOrder(t *testing.T) {
	srv := newExportServer(t)
	dir := t.TempDir()
	_, stderr, err := runCmd(t, srv, "event", "export", "2024cthar",
		"--to", "csv", "--dir", dir, "--only", "matches")
	requireNoError(t, err, stderr)

	var keys []string
	for _, line := range lines(readExported(t, dir, "matches", "csv"))[1:] {
		keys = append(keys, strings.Split(line, ",")[1])
	}
	quals := 0
	for _, k := range keys {
		if strings.Contains(k, "_qm") {
			quals++
		} else {
			break
		}
	}
	if quals == 0 || quals == len(keys) {
		t.Fatalf("the fixture should have both quals and playoffs, got %v", keys)
	}
	for _, k := range keys[quals:] {
		if strings.Contains(k, "_qm") {
			t.Errorf("a qualification match sorted after a playoff match: %v", keys)
		}
	}
}

// The event is fetched once even though two datasets want it: the event file
// itself, and the bracket format that names the playoff matches.
func TestEventExportFetchesEachEndpointOnce(t *testing.T) {
	srv := newExportServer(t)
	_, stderr, err := runCmd(t, srv, "event", "export", "2024cthar",
		"--to", "csv", "--dir", t.TempDir(), "--only", "event,matches", "--no-cache")
	requireNoError(t, err, stderr)

	counts := map[string]int{}
	for _, p := range requestPaths(t, srv) {
		counts[p]++
	}
	if counts["/event/2024cthar"] != 1 {
		t.Errorf("/event/2024cthar fetched %d times, want 1", counts["/event/2024cthar"])
	}
}

// jsonEqual compares two decoded payloads.
func jsonEqual(a, b any) bool {
	x, errA := json.Marshal(a)
	y, errB := json.Marshal(b)
	return errA == nil && errB == nil && string(x) == string(y)
}

// tail is the last few bytes of a string, for an error message about how a
// file ends.
func tail(s string) string {
	if len(s) > 20 {
		return s[len(s)-20:]
	}
	return s
}
