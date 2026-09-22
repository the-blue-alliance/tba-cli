package cmd

import (
	"slices"
	"strings"
	"testing"

	"github.com/the-blue-alliance/tba-cli/internal/clierr"
)

// The CLI contract says a mistake in the invocation exits 2 with a usage hint,
// while a failure of the work exits 1, 4 or 5. Flag values are the easiest
// place for that to slip: every table command reaches the same handful of
// checks, but each of them returns an error somebody has to remember to wrap
// in clierr.Usage, and one that was not wrapped exits 1 from exactly the
// commands nobody happened to write a test for.
//
// So the class is tested rather than the instances: every command that renders
// a table is run with every flag mistake that can be made about a table, and
// all of them have to exit 2.

// flagContractCase is how one leaf command takes part, if it does: args is a
// representative invocation, and skip says why a command renders no table and
// therefore has no --sort or --columns to get wrong.
type flagContractCase struct {
	args []string
	skip string
}

// flagContractCommands covers every leaf command in the tree. A command with
// no entry fails TestEveryLeafCommandIsCoveredByTheFlagContract, so a new one
// has to say which side of the line it is on rather than quietly escaping the
// contract.
//
// The keys are command paths, as cobra spells them.
var flagContractCommands = map[string]flagContractCase{
	// Table commands: everything below goes through outputTable, so --sort,
	// --columns and --format all mean something to it.
	"tba cache list":            {args: []string{"cache", "list"}},
	"tba config list":           {args: []string{"config", "list"}},
	"tba district events":       {args: []string{"district", "events", "2024ne"}},
	"tba district list":         {args: []string{"district", "list", "--year", "2024"}},
	"tba district rankings":     {args: []string{"district", "rankings", "2024ne"}},
	"tba district teams":        {args: []string{"district", "teams", "2024ne"}},
	"tba event alliances":       {args: []string{"event", "alliances", "2024cthar"}},
	"tba event awards":          {args: []string{"event", "awards", "2024cthar"}},
	"tba event district-points": {args: []string{"event", "district-points", "2024cthar"}},
	"tba event insights":        {args: []string{"event", "insights", "2024cthar"}},
	"tba event list":            {args: []string{"event", "list", "--year", "2024"}},
	"tba event matches":         {args: []string{"event", "matches", "2024cthar"}},
	"tba event oprs":            {args: []string{"event", "oprs", "2024cthar"}},
	"tba event predictions":     {args: []string{"event", "predictions", "2024cthar"}},
	"tba event rankings":        {args: []string{"event", "rankings", "2024cthar"}},
	"tba event team-statuses":   {args: []string{"event", "team-statuses", "2024cthar"}},
	"tba event teams":           {args: []string{"event", "teams", "2024cthar"}},
	"tba insight leaderboards":  {args: []string{"insight", "leaderboards", "--year", "2024"}},
	"tba insight notables":      {args: []string{"insight", "notables", "--year", "2024"}},
	"tba team awards":           {args: []string{"team", "awards", "177"}},
	"tba team districts":        {args: []string{"team", "districts", "177"}},
	"tba team events":           {args: []string{"team", "events", "177", "--year", "2024"}},
	"tba team list":             {args: []string{"team", "list", "--year", "2024"}},
	"tba team matches":          {args: []string{"team", "matches", "177", "--year", "2024"}},
	"tba team media":            {args: []string{"team", "media", "177", "--year", "2024"}},
	"tba team next":             {args: []string{"team", "next", "177", "2024cthar", "--all"}},
	"tba team robots":           {args: []string{"team", "robots", "177"}},
	"tba team search":           {args: []string{"team", "search", "robo", "--year", "2024"}},
	"tba team years":            {args: []string{"team", "years", "177"}},

	// Everything else. These still owe exit 2 for a bad --format or --color,
	// which the root checks before any of them runs, but they have no table
	// for --sort and --columns to be wrong about.
	"tba auth login":    {skip: "prompts for a key; prints no table"},
	"tba auth logout":   {skip: "prints no table"},
	"tba auth status":   {skip: "key/value output"},
	"tba cache clear":   {skip: "prints no table"},
	"tba cache info":    {skip: "key/value output"},
	"tba cache prune":   {skip: "prints what it removed, not a table"},
	"tba config get":    {skip: "prints one value"},
	"tba config path":   {skip: "prints one path"},
	"tba config set":    {skip: "writes a setting"},
	"tba config unset":  {skip: "removes a setting"},
	"tba event export":  {skip: "writes files; its own flags are checked in export_test.go"},
	"tba event view":    {skip: "key/value output"},
	"tba event watch":   {skip: "streams changes; --sort and --columns do not apply"},
	"tba match view":    {skip: "key/value output"},
	"tba open":          {skip: "opens a browser; prints no table"},
	"tba status":        {skip: "key/value output"},
	"tba team standing": {skip: "key/value output"},
	"tba team view":     {skip: "key/value output"},
	"tba version":       {skip: "key/value output"},
}

// flagContractInvocations are the mistakes a user makes about a table. Each
// one is a mistake in the command line rather than a failure of the work, so
// each one is exit 2.
//
// The --sort and --columns cases name a format on purpose. A test's stdout is
// not a terminal, so --format defaults to json, where --columns is refused
// before the column name is ever looked at and SelectColumns — the check the
// contract is really about — is never reached.
var flagContractInvocations = []struct {
	name string
	args []string
}{
	{"bad --sort column", []string{"--format", "table", "--sort=bogus"}},
	{"bad --sort column with JSON", []string{"--format", "json", "--sort=bogus"}},
	{"unknown --columns name", []string{"--format", "table", "--columns", "nope"}},
	{"unknown --columns name for a file", []string{"--format", "csv", "--columns", "nope"}},
	{"--columns with JSON", []string{"--columns", "key", "--format", "json"}},
	{"bad --color", []string{"--color", "sometimes"}},
	{"bad --format", []string{"--format", "xml"}},
}

// flagContractPaths is every command the contract covers. leafCommands already
// leaves out cobra's own `help` and `completion`, which write shell scripts
// rather than answer questions; the hidden `docs` generators write files and
// are exercised by docs_test.go.
func flagContractPaths() []string {
	var out []string
	for _, c := range leafCommands(NewRootCmd()) {
		if strings.HasPrefix(c.CommandPath(), "tba docs") {
			continue
		}
		out = append(out, c.CommandPath())
	}
	slices.Sort(out)
	return out
}

// TestEveryLeafCommandIsCoveredByTheFlagContract is what makes the contract
// hold for commands nobody has written yet: a new one is a failure here until
// someone says how to invoke it, or why it has no table.
func TestEveryLeafCommandIsCoveredByTheFlagContract(t *testing.T) {
	paths := flagContractPaths()
	for _, path := range paths {
		if _, ok := flagContractCommands[path]; !ok {
			t.Errorf("%s takes no part in the flag contract; add it to flagContractCommands "+
				"with a representative invocation, or with a skip saying it renders no table", path)
		}
	}
	for path := range flagContractCommands {
		if !contains(paths, path) {
			t.Errorf("flagContractCommands lists %q, which is not a command any more", path)
		}
	}
}

// TestFlagContractCasesInvokeTheirOwnCommand guards the table itself: an entry
// whose args address a different command would test that command twice and
// its own not at all.
func TestFlagContractCasesInvokeTheirOwnCommand(t *testing.T) {
	for _, path := range sortedFlagContractPaths() {
		tc := flagContractCommands[path]
		if tc.skip != "" {
			continue
		}
		want := strings.TrimPrefix(path, "tba ")
		got := strings.Join(nonFlagArgs(tc.args), " ")
		if !strings.HasPrefix(got, want) {
			t.Errorf("%s is invoked as %q", path, strings.Join(tc.args, " "))
		}
	}
}

// nonFlagArgs keeps the words before the first flag, which is the command path
// the args address.
func nonFlagArgs(args []string) []string {
	var out []string
	for _, a := range args {
		if strings.HasPrefix(a, "-") {
			break
		}
		out = append(out, a)
	}
	return out
}

func sortedFlagContractPaths() []string {
	out := make([]string, 0, len(flagContractCommands))
	for path := range flagContractCommands {
		out = append(out, path)
	}
	slices.Sort(out)
	return out
}

// TestTableCommandsRejectBadFlagsAsUsageErrors is the contract itself.
//
// The exit code is the assertion rather than the message: a 404 from a fixture
// somebody forgot exits 5, and a raw error from a check that was never wrapped
// in clierr.Usage exits 1, so anything that is not 2 fails here whether the
// cause is the CLI or the test's own setup.
func TestTableCommandsRejectBadFlagsAsUsageErrors(t *testing.T) {
	for _, path := range sortedFlagContractPaths() {
		tc := flagContractCommands[path]
		if tc.skip != "" {
			continue
		}
		for _, inv := range flagContractInvocations {
			t.Run(strings.ReplaceAll(path, " ", "_")+"/"+strings.ReplaceAll(inv.name, " ", "_"), func(t *testing.T) {
				srv := newFakeTBA(t, flagContractRoutes())
				args := append(append([]string{}, tc.args...), inv.args...)
				err := requireExitCode(t, clierr.ExitUsage, srv, args...)
				if err == nil {
					t.Fatalf("%v succeeded", args)
				}
			})
		}
	}
}

// TestEveryCommandRejectsABadFormatAsAUsageError covers the two checks the
// root makes for every command, table or not: they run in PersistentPreRunE,
// before a single request goes out, so no fixture is needed and a command that
// prints no table is still bound by them.
func TestEveryCommandRejectsABadFormatAsAUsageError(t *testing.T) {
	for _, path := range flagContractPaths() {
		args := strings.Split(strings.TrimPrefix(path, "tba "), " ")
		for _, bad := range [][]string{{"--format", "xml"}, {"--color", "sometimes"}} {
			t.Run(strings.ReplaceAll(path, " ", "_")+"/"+bad[1], func(t *testing.T) {
				_ = requireExitCode(t, clierr.ExitUsage, nil, append(args, bad...)...)
			})
		}
	}
}

// flagContractRoutes is enough of the API for every table command to get as
// far as rendering. The bodies are the package's ordinary fixtures; what
// matters is that no request 404s, since a 404 exits 5 and would hide the
// usage error the test is looking for.
func flagContractRoutes() map[string]any {
	return map[string]any{
		"/status": apiStatusJSON,

		"/districts/2024":            districts2024JSON,
		"/district/2024ne/events":    "[" + event2024ctharJSON + "]",
		"/district/2024ne/rankings":  districtRankings2024neJSON,
		"/district/2024ne/teams":     "[" + teamFRC177JSON + "]",
		"/events/2024":               "[" + event2024ctharJSON + "]",
		"/event/2024cthar":           event2024ctharJSON,
		"/event/2024cthar/alliances": alliances2024ctharJSON,
		"/event/2024cthar/awards":    awards2024ctharJSON,

		"/event/2024cthar/district_points": districtPoints2024ctharJSON,
		"/event/2024cthar/insights":        insights2024ctharJSON,
		"/event/2024cthar/matches":         matches2024ctharJSON,
		"/event/2024cthar/oprs":            oprs2024ctharJSON,
		"/event/2024cthar/predictions":     predictions2024ctharJSON,
		"/event/2024cthar/rankings":        rankings2024ctharJSON,
		"/event/2024cthar/teams":           "[" + teamFRC177JSON + "]",
		"/event/2024cthar/teams/keys":      `["frc177"]`,
		"/event/2024cthar/teams/simple":    teamsSimple2024ctharJSON,
		"/event/2024cthar/teams/statuses":  teamStatuses2024ctharJSON,

		"/insights/leaderboards/2024": leaderboards2024JSON,
		"/insights/notables/2024":     notables2024JSON,

		"/teams/2024/0": "[" + teamFRC177JSON + "]",
		"/teams/2024/1": "[]",

		"/team/frc177/awards":                  teamAwards177JSON,
		"/team/frc177/districts":               teamDistricts177JSON,
		"/team/frc177/event/2024cthar/matches": teamMatches177At2024ctharJSON,
		"/team/frc177/events/2024":             teamEvents177In2024JSON,
		"/team/frc177/matches/2024":            teamMatches177Season2024JSON,
		"/team/frc177/media/2024":              teamMedia177JSON,
		"/team/frc177/robots":                  teamRobots177JSON,
		"/team/frc177/years_participated":      `[2023,2024]`,
	}
}
