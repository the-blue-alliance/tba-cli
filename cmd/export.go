package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"

	"github.com/spf13/cobra"
	"github.com/the-blue-alliance/tba-cli/internal/api"
	"github.com/the-blue-alliance/tba-cli/internal/clierr"
	"github.com/the-blue-alliance/tba-cli/internal/frc"
	"github.com/the-blue-alliance/tba-cli/internal/fsutil"
	"github.com/the-blue-alliance/tba-cli/internal/output"
)

// exportFormats lists the accepted --to values, in help-text order. The export
// format is never inferred from a file name or a terminal: a command that
// writes files to disk should do exactly what it was told to do.
const exportFormats = "csv, tsv, json"

// exportFileMode is the mode of a written export file: ordinary data, readable
// by whatever is going to consume it.
const exportFileMode = 0o644

// exportTempPattern names the temporary files an export writes before it
// renames them into place. The leading dot keeps a half-finished export out of
// a shell glob, and the suffix is what os.CreateTemp fills with random digits.
const exportTempPattern = ".tba-export-*"

// exportBackupPattern names the copies --force takes of the files it is about
// to replace, so that a rename failing part way through the set can put the
// directory back the way it found it.
const exportBackupPattern = ".tba-export-backup-*"

// renameFile is os.Rename, replaceable so that a test can see what an export
// does when the filesystem refuses half way through the rename phase.
var renameFile = os.Rename

func newEventExportCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "export <key>",
		Short: "Export an event's data to files",
		Long: `Write everything the API knows about an event to one file per dataset.

--to is required and is the only thing that decides the format: nothing is
inferred from a file name, a terminal or a config file. Each dataset becomes
<dir>/<prefix>-<dataset>.<ext>, where prefix defaults to the event key:

  ` + strings.Join(exportDatasetNames, ", ") + `

For csv and tsv each file carries exactly the columns the matching
` + "`tba event <dataset>`" + ` command prints, with a header row and no color. For
json each file holds the API's own payload, pretty-printed.

The bytes are reproducible: no timestamps are written, rows are in a fixed
order, and two exports of the same data produce identical files. Every file is
written to a temporary file in the target directory and renamed into place only
once all of them have been fetched, so a failure part way through leaves
nothing behind. An existing file is never overwritten without --force.

A dataset the event does not have -- district points at a regional, alliances
before selection -- is reported on stderr and skipped, not treated as a
failure.

The written paths go to stdout, one per line, so they can be piped. Notes and
the summary go to stderr. --format json (or --json) replaces the path list with
a JSON summary of what was written and what was skipped.`,
		Example: `  tba event export 2024cthar --to csv
  tba event export 2024cthar --to json --dir exports
  tba event export 2024cthar --to csv --only matches,rankings
  tba event export 2024cthar --to csv --dry-run
  tba event export 2024cthar --to tsv --force --prefix hartford`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := validateEventKey(args[0]); err != nil {
				return err
			}
			opts, err := exportOptionsFrom(cmd, args[0])
			if err != nil {
				return err
			}
			return runEventExport(cmd, opts)
		},
	}
	c.Flags().String("to", "", "Required. Format to write the files in: "+exportFormats)
	c.Flags().String("dir", ".", "Directory to write the files into")
	c.Flags().String("only", "", "Only export these datasets, comma-separated: "+strings.Join(exportDatasetNames, ", "))
	c.Flags().Bool("force", false, "Overwrite files that already exist")
	c.Flags().Bool("dry-run", false, "Print the files that would be written, without fetching or writing anything")
	c.Flags().String("prefix", "", "Prefix for the file names (default: the event key)")

	_ = c.RegisterFlagCompletionFunc("to", completeFixed("csv", "tsv", "json"))
	_ = c.RegisterFlagCompletionFunc("only", completeFixed(exportDatasetNames...))
	return c
}

// exportDatasetNames is the --only vocabulary and the order the files are
// written in, whatever order --only listed them.
var exportDatasetNames = []string{
	"event", "teams", "matches", "rankings", "alliances",
	"awards", "oprs", "district-points", "team-statuses",
}

// exportDataset is one file's worth of an export: where the data comes from,
// and how it becomes the table the delimited formats write.
type exportDataset struct {
	name string
	path string
	// table renders the payload with the same columns `tba event <name>`
	// prints. It is called for csv and tsv only; json writes the payload.
	table func(e *exporter, raw json.RawMessage) (output.Table, error)
}

// exportOptions is one invocation's settled configuration.
type exportOptions struct {
	key    string
	format string
	dir    string
	prefix string
	force  bool
	dryRun bool
	// jsonSummary replaces the list of written paths with a JSON object
	// describing the whole run.
	jsonSummary bool
	datasets    []exportDataset
}

// exportSkip records a dataset the event simply does not have.
type exportSkip struct {
	Dataset string `json:"dataset"`
	Reason  string `json:"reason"`
}

// exportSummary is what --format json prints, and what the plain output is a
// projection of.
type exportSummary struct {
	Written []string     `json:"written"`
	Skipped []exportSkip `json:"skipped"`
	// DryRun marks a run that wrote nothing, so that a script cannot mistake
	// a preview for an export.
	DryRun bool `json:"dry_run,omitempty"`
}

// exportOptionsFrom validates the flags before anything is fetched or written.
func exportOptionsFrom(cmd *cobra.Command, key string) (exportOptions, error) {
	opts := exportOptions{key: key}

	to, _ := cmd.Flags().GetString("to")
	switch strings.ToLower(strings.TrimSpace(to)) {
	case "":
		return opts, clierr.Usage("--to is required: it says which format to write, one of %s", exportFormats)
	case "csv", "tsv", "json":
		opts.format = strings.ToLower(strings.TrimSpace(to))
	default:
		return opts, clierr.Usage("invalid --to %q (want: %s)", to, exportFormats)
	}

	// --format describes how this command talks to the terminal, not what it
	// writes to disk; only --to chooses that. --format json is the one value
	// that means something here, and it selects the JSON summary.
	//
	// It is read through the settings layer rather than off the flag, so
	// TBA_FORMAT and format: in config.yaml mean here exactly what they mean
	// everywhere else. What stays off the layer is the terminal rule: a bare
	// `auto` leaves the summary as a plain list of paths whether or not
	// stdout is a pipe, because `tba event export ... | xargs` wants file
	// names rather than a JSON object.
	s := settings(cmd)
	if s.Source("format") != sourceDefault {
		raw := s.String("format")
		normalized, ok := normalizeFormat(raw)
		if !ok {
			return opts, clierr.Usage("invalid %s %q (want: %s)", s.origin("format"), raw, validFormats)
		}
		switch normalized {
		case "":
			// auto: see above.
		case "json":
			opts.jsonSummary = true
		default:
			// Name where the format came from, so that "drop --format table"
			// is not advice about a flag the user never typed.
			chosen := "--format " + raw
			if s.Source("format") != sourceFlag {
				chosen = fmt.Sprintf("--format %s from %s", raw, s.origin("format"))
			}
			return opts, clierr.Usage(
				"%s does not choose the export format; use --to %s (--format json prints a JSON summary of the run)",
				chosen, exportFormats)
		}
	}
	if s.Bool("json") || jqExpr(cmd) != "" {
		opts.jsonSummary = true
	}

	opts.dir, _ = cmd.Flags().GetString("dir")
	if strings.TrimSpace(opts.dir) == "" {
		return opts, clierr.Usage("--dir cannot be empty (use . for the current directory)")
	}
	opts.force, _ = cmd.Flags().GetBool("force")
	opts.dryRun, _ = cmd.Flags().GetBool("dry-run")

	opts.prefix, _ = cmd.Flags().GetString("prefix")
	if opts.prefix == "" {
		opts.prefix = key
	}
	// A prefix names a file, not a place to put it: --dir is what chooses the
	// directory, and a separator here would quietly write somewhere else.
	if strings.ContainsAny(opts.prefix, `/\`) {
		return opts, clierr.Usage("--prefix names a file, not a path: %q contains a path separator (use --dir)", opts.prefix)
	}

	only, _ := cmd.Flags().GetString("only")
	names, err := exportDatasetsFrom(only)
	if err != nil {
		return opts, err
	}
	opts.datasets = eventExportDatasets(key, names)
	return opts, nil
}

// exportDatasetsFrom resolves --only to a list of dataset names in the fixed
// export order, so that the same request always produces the same files in the
// same order however the flag was spelled.
func exportDatasetsFrom(only string) ([]string, error) {
	if strings.TrimSpace(only) == "" {
		return exportDatasetNames, nil
	}
	wanted := map[string]bool{}
	for _, part := range strings.Split(only, ",") {
		name := strings.ToLower(strings.TrimSpace(part))
		if name == "" {
			continue
		}
		if !slices.Contains(exportDatasetNames, name) {
			return nil, clierr.Usage("unknown dataset %q in --only (want: %s)", part, strings.Join(exportDatasetNames, ", "))
		}
		wanted[name] = true
	}
	if len(wanted) == 0 {
		return nil, clierr.Usage("--only selects no datasets (want: %s)", strings.Join(exportDatasetNames, ", "))
	}
	var names []string
	for _, name := range exportDatasetNames {
		if wanted[name] {
			names = append(names, name)
		}
	}
	return names, nil
}

// eventExportDatasets pairs each requested dataset name with its endpoint and
// its table builder.
func eventExportDatasets(key string, names []string) []exportDataset {
	at := func(suffix string) string { return "/event/" + key + suffix }
	all := map[string]exportDataset{
		"event":           {"event", at(""), exportEventTable},
		"teams":           {"teams", at("/teams"), exportTeamsTable},
		"matches":         {"matches", at("/matches"), exportMatchesTable},
		"rankings":        {"rankings", at("/rankings"), exportRankingsTable},
		"alliances":       {"alliances", at("/alliances"), exportAlliancesTable},
		"awards":          {"awards", at("/awards"), exportAwardsTable},
		"oprs":            {"oprs", at("/oprs"), exportOPRsTable},
		"district-points": {"district-points", at("/district_points"), exportDistrictPointsTable},
		"team-statuses":   {"team-statuses", at("/teams/statuses"), exportTeamStatusesTable},
	}
	out := make([]exportDataset, 0, len(names))
	for _, name := range names {
		out = append(out, all[name])
	}
	return out
}

// exporter carries the state one export run shares between its datasets.
type exporter struct {
	cmd    *cobra.Command
	client *api.Client
	opts   exportOptions
	// fetched memoises every response, because two datasets can want the same
	// one: the event file and the bracket format that names playoff matches
	// both come from /event/<key>.
	fetched map[string]fetchResult
}

type fetchResult struct {
	raw json.RawMessage
	err error
}

func (e *exporter) get(path string) (json.RawMessage, error) {
	if hit, ok := e.fetched[path]; ok {
		return hit.raw, hit.err
	}
	raw, err := e.client.GetRaw(e.cmd.Context(), path)
	e.fetched[path] = fetchResult{raw: raw, err: err}
	return raw, err
}

// playoffType is the bracket format the match labels depend on. Like the
// `event matches` command, an event that cannot be fetched only costs the
// labels their precision, so the failure is swallowed.
func (e *exporter) playoffType() *int {
	raw, err := e.get("/event/" + e.opts.key)
	if err != nil {
		return nil
	}
	var event api.Event
	if err := json.Unmarshal(raw, &event); err != nil {
		return nil
	}
	return event.PlayoffType
}

// nicknames fills the Name column of the rankings table, which the rankings
// endpoint does not carry. As in `event rankings` it is a convenience: a
// failure leaves the column blank rather than failing the export.
func (e *exporter) nicknames() map[string]string {
	names := map[string]string{}
	raw, err := e.get("/event/" + e.opts.key + "/teams/simple")
	if err != nil {
		return names
	}
	var teams []api.Team
	if err := json.Unmarshal(raw, &teams); err != nil {
		return names
	}
	for _, t := range teams {
		names[t.Key] = t.Nickname
	}
	return names
}

func runEventExport(cmd *cobra.Command, opts exportOptions) error {
	// Every file name is known before a single request is made, so a name
	// clash is reported without spending the API calls that would be thrown
	// away. A dry run is checked too: a preview that promises files the real
	// run would refuse to write is worse than no preview.
	paths := make([]string, len(opts.datasets))
	for i, ds := range opts.datasets {
		paths[i] = filepath.Join(opts.dir, fmt.Sprintf("%s-%s.%s", opts.prefix, ds.name, opts.format))
	}
	if !opts.force {
		if err := refuseExistingFiles(paths); err != nil {
			return err
		}
	}

	if opts.dryRun {
		return reportExport(cmd, opts, exportSummary{Written: paths, Skipped: []exportSkip{}, DryRun: true})
	}

	client, err := newClient(cmd)
	if err != nil {
		return err
	}
	e := &exporter{cmd: cmd, client: client, opts: opts, fetched: map[string]fetchResult{}}

	if err := os.MkdirAll(opts.dir, 0o755); err != nil {
		return err
	}

	// A half-written export is worse than none: the files are all fetched and
	// staged next to their destinations first, and only renamed into place
	// once every one of them is on disk. Renaming within a directory is the
	// cheapest thing the filesystem does, so the window where an export is
	// visibly incomplete is as small as it can be made.
	var pending []staged
	discard := func() {
		for _, s := range pending {
			_ = os.Remove(s.temp)
		}
	}

	summary := exportSummary{Written: []string{}, Skipped: []exportSkip{}}
	for i, ds := range opts.datasets {
		raw, err := e.get(ds.path)
		if err != nil {
			// A 404 is the API saying this event never had that dataset,
			// which is ordinary: a regional awards no district points, and
			// alliances do not exist until selection. Anything else is a real
			// failure and takes the whole export down with it.
			if clierr.IsKind(err, clierr.KindNotFound) {
				summary.Skipped = append(summary.Skipped, exportSkip{Dataset: ds.name, Reason: "the API has no " + ds.name + " for this event (HTTP 404)"})
				continue
			}
			discard()
			return err
		}
		body, err := renderExportFile(e, ds, raw)
		if err != nil {
			discard()
			return fmt.Errorf("rendering %s: %w", ds.name, err)
		}
		temp, err := stageExportFile(opts.dir, body)
		if err != nil {
			discard()
			return err
		}
		pending = append(pending, staged{temp: temp, final: paths[i]})
		summary.Written = append(summary.Written, paths[i])
	}

	if err := commitExport(opts.dir, pending, opts.force); err != nil {
		discard()
		return err
	}
	return reportExport(cmd, opts, summary)
}

// staged is one file on its way into place: the temporary copy that holds its
// bytes, and the name it is going to have.
type staged struct{ temp, final string }

// commitExport renames the staged files into place. Either all of them arrive
// or none of them do, and a run that fails leaves the directory exactly as it
// found it.
//
// The check for existing files is made here as well as at the start of the
// run. The fetches in between take seconds — long enough for another process,
// or the user in another terminal, to write one of these names — and a file
// that appeared in the meantime is precisely the file that must not be
// silently replaced.
//
// Under --force the files that are about to be replaced are moved aside
// rather than overwritten. Renaming N of M files and then failing used to
// leave a directory that was neither the old export nor the new one, with no
// way back to either.
func commitExport(dir string, pending []staged, force bool) error {
	finals := make([]string, len(pending))
	for i, s := range pending {
		finals[i] = s.final
	}
	existing := existingFiles(finals)
	if len(existing) > 0 && !force {
		return overwriteError(existing)
	}

	// backups maps each copy set aside to the name it came from.
	var backups []staged
	undo := func(renamed []string) {
		for _, p := range renamed {
			_ = os.Remove(p)
		}
		for _, b := range backups {
			_ = renameFile(b.temp, b.final)
		}
	}

	for _, p := range existing {
		aside, err := reserveExportName(dir, exportBackupPattern)
		if err != nil {
			undo(nil)
			return err
		}
		if err := renameFile(p, aside); err != nil {
			_ = os.Remove(aside)
			undo(nil)
			return err
		}
		backups = append(backups, staged{temp: aside, final: p})
	}

	var renamed []string
	for _, s := range pending {
		if err := renameFile(s.temp, s.final); err != nil {
			undo(renamed)
			return err
		}
		renamed = append(renamed, s.final)
	}
	for _, b := range backups {
		_ = os.Remove(b.temp)
	}
	return nil
}

// reserveExportName claims a name in dir that nothing else will take, for a
// file that is about to be renamed into it.
func reserveExportName(dir, pattern string) (string, error) {
	f, err := os.CreateTemp(dir, pattern)
	if err != nil {
		return "", err
	}
	name := f.Name()
	if err := f.Close(); err != nil {
		_ = os.Remove(name)
		return "", err
	}
	return name, nil
}

// existingFiles returns the paths that are already there, in the order given.
func existingFiles(paths []string) []string {
	var clashes []string
	for _, p := range paths {
		if _, err := os.Stat(p); err == nil {
			clashes = append(clashes, p)
		}
	}
	return clashes
}

// refuseExistingFiles reports every file that is already there, so that one
// run tells the user about all of them rather than one per attempt.
func refuseExistingFiles(paths []string) error {
	if clashes := existingFiles(paths); len(clashes) > 0 {
		return overwriteError(clashes)
	}
	return nil
}

func overwriteError(clashes []string) error {
	return fmt.Errorf("refusing to overwrite %d existing file(s); pass --force to replace them:\n  %s",
		len(clashes), strings.Join(clashes, "\n  "))
}

// renderExportFile turns one payload into the bytes of its file.
func renderExportFile(e *exporter, ds exportDataset, raw json.RawMessage) ([]byte, error) {
	var buf bytes.Buffer
	if e.opts.format == "json" {
		// json.Indent rather than a re-encode: the file is meant to be the
		// API's own answer, so nothing in it is re-escaped or reordered.
		if err := json.Indent(&buf, raw, "", "  "); err != nil {
			return nil, err
		}
		buf.WriteByte('\n')
		return buf.Bytes(), nil
	}
	table, err := ds.table(e, raw)
	if err != nil {
		return nil, err
	}
	// Never colored: these bytes are read by another program, and an ANSI
	// escape in a cell would be part of the data.
	if err := output.Render(&buf, table, output.RenderOptions{Format: e.opts.format, Color: output.ColorNever}); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// stageExportFile writes one file's bytes next to where it is going to live,
// and returns the temporary name to rename from. An export is ordinary data,
// so it is staged with the mode a written file usually has rather than the
// private mode a temporary file is created with.
func stageExportFile(dir string, body []byte) (string, error) {
	return fsutil.WriteTemp(dir, exportTempPattern, body, exportFileMode)
}

// reportExport writes the paths to stdout and everything else to stderr, so
// that `tba event export ... | xargs` sees nothing but file names.
func reportExport(cmd *cobra.Command, opts exportOptions, summary exportSummary) error {
	errW := cmd.ErrOrStderr()
	for _, s := range summary.Skipped {
		fmt.Fprintf(errW, "note: skipped %s: %s\n", s.Dataset, s.Reason)
	}

	if opts.jsonSummary {
		if err := output.PrintJSONWithFilter(cmd.OutOrStdout(), summary, jqExpr(cmd), rawOutput(cmd)); err != nil {
			return err
		}
	} else {
		for _, p := range summary.Written {
			fmt.Fprintln(cmd.OutOrStdout(), p)
		}
	}

	verb := "wrote"
	if summary.DryRun {
		verb = "dry run: would write"
	}
	fmt.Fprintf(errW, "%s %d file(s) to %s\n", verb, len(summary.Written), opts.dir)
	return nil
}

// --- per-dataset tables -------------------------------------------------
//
// Each one decodes the payload and hands it to the row builder the matching
// `tba event <dataset>` command uses, so the columns cannot drift apart. Where
// a command leaves the API's order alone, the export sorts: a file that is
// diffed against last week's copy has to be in a fixed order, while a table
// read once on a terminal does not.

func exportEventTable(_ *exporter, raw json.RawMessage) (output.Table, error) {
	var event api.Event
	if err := json.Unmarshal(raw, &event); err != nil {
		return output.Table{}, err
	}
	// An event is one object, not a list of them, so it becomes the key/value
	// listing `event view` prints rather than a row of unlabelled columns.
	pairs := eventDetailPairs(event)
	rows := make([][]string, 0, len(pairs)/2)
	for i := 0; i+1 < len(pairs); i += 2 {
		rows = append(rows, []string{pairs[i], pairs[i+1]})
	}
	return output.Table{Headers: []string{"Field", "Value"}, Rows: rows}, nil
}

func exportTeamsTable(_ *exporter, raw json.RawMessage) (output.Table, error) {
	var teams []api.Team
	if err := json.Unmarshal(raw, &teams); err != nil {
		return output.Table{}, err
	}
	sort.SliceStable(teams, func(i, j int) bool { return teams[i].TeamNumber < teams[j].TeamNumber })
	return eventTeamsTable(teams), nil
}

func exportMatchesTable(e *exporter, raw json.RawMessage) (output.Table, error) {
	var matches []api.Match
	if err := json.Unmarshal(raw, &matches); err != nil {
		return output.Table{}, err
	}
	frc.SortMatches(matches)
	rows, _ := matchTableRows(matches, constantPlayoffType(e.playoffType()), false)
	return output.Table{Headers: matchHeaders, Rows: rows}, nil
}

func exportRankingsTable(e *exporter, raw json.RawMessage) (output.Table, error) {
	var rankings api.EventRankings
	if err := json.Unmarshal(raw, &rankings); err != nil {
		return output.Table{}, err
	}
	return eventRankingsTable(&rankings, e.nicknames()), nil
}

func exportAlliancesTable(_ *exporter, raw json.RawMessage) (output.Table, error) {
	var alliances []api.EventAlliance
	if err := json.Unmarshal(raw, &alliances); err != nil {
		return output.Table{}, err
	}
	return eventAlliancesTable(alliances), nil
}

func exportAwardsTable(_ *exporter, raw json.RawMessage) (output.Table, error) {
	var awards []api.Award
	if err := json.Unmarshal(raw, &awards); err != nil {
		return output.Table{}, err
	}
	// Award type is the order FIRST lists awards in, and it is the same every
	// season; the recipient breaks a tie where one award has several winners.
	sort.SliceStable(awards, func(i, j int) bool {
		if awards[i].AwardType != awards[j].AwardType {
			return awards[i].AwardType < awards[j].AwardType
		}
		return awardSortKey(awards[i]) < awardSortKey(awards[j])
	})
	return eventAwardsTable(awards), nil
}

// awardSortKey orders the awards that share a type, by team number where there
// is one and by name otherwise. The number is zero-padded so that 177 sorts
// before 1073 rather than after it.
func awardSortKey(a api.Award) string {
	for _, r := range a.Recipients {
		if r.TeamKey != nil {
			return fmt.Sprintf("%010d", teamNumber(*r.TeamKey))
		}
		if r.Awardee != nil {
			return *r.Awardee
		}
	}
	return a.Name
}

func exportOPRsTable(_ *exporter, raw json.RawMessage) (output.Table, error) {
	var oprs api.EventOPRs
	if err := json.Unmarshal(raw, &oprs); err != nil {
		return output.Table{}, err
	}
	return eventOPRsTable(oprs), nil
}

func exportDistrictPointsTable(_ *exporter, raw json.RawMessage) (output.Table, error) {
	var points api.EventDistrictPoints
	if err := json.Unmarshal(raw, &points); err != nil {
		return output.Table{}, err
	}
	return eventDistrictPointsTable(points, false), nil
}

func exportTeamStatusesTable(_ *exporter, raw json.RawMessage) (output.Table, error) {
	var statuses map[string]*api.TeamEventStatus
	if err := json.Unmarshal(raw, &statuses); err != nil {
		return output.Table{}, err
	}
	return eventTeamStatusesTable(statuses), nil
}

// completeFixed answers a flag's completion with a fixed vocabulary, for the
// flags whose values are a closed set rather than anything from the API.
func completeFixed(values ...string) func(*cobra.Command, []string, string) ([]string, cobra.ShellCompDirective) {
	return func(_ *cobra.Command, _ []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		var out []string
		for _, v := range values {
			if strings.HasPrefix(v, toComplete) {
				out = append(out, v)
			}
		}
		return out, cobra.ShellCompDirectiveNoFileComp
	}
}
