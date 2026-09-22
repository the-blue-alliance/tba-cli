package cmd

import (
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

// leafCommands returns every runnable command in the tree, excluding the ones
// cobra generates for us (help, completion).
func leafCommands(root *cobra.Command) []*cobra.Command {
	var out []*cobra.Command
	var walk func(c *cobra.Command)
	walk = func(c *cobra.Command) {
		switch c.Name() {
		case "help", "completion":
			return
		}
		if c.Runnable() && !c.HasSubCommands() {
			out = append(out, c)
		}
		for _, sub := range c.Commands() {
			walk(sub)
		}
	}
	walk(root)
	return out
}

func TestEveryLeafCommandHasExamples(t *testing.T) {
	for _, c := range leafCommands(NewRootCmd()) {
		if c.Example == "" {
			t.Errorf("%q has no Example block", c.CommandPath())
			continue
		}
		got := lines(c.Example)
		if len(got) < 2 {
			t.Errorf("%q has %d example(s), want at least 2", c.CommandPath(), len(got))
		}
		for _, line := range got {
			if !strings.HasPrefix(strings.TrimSpace(line), "tba ") {
				t.Errorf("%q has an example that does not start with tba: %q", c.CommandPath(), line)
			}
		}
	}
}

func TestExamplesShowUpInHelp(t *testing.T) {
	out, _, err := runCmd(t, nil, "event", "matches", "--help")
	requireNoError(t, err, "")
	requireContains(t, out, "Examples:")
	requireContains(t, out, "tba event matches 2024cthar --format csv")
}

func TestGroupCommandsHavePluralAliases(t *testing.T) {
	want := map[string]string{
		"team":     "teams",
		"event":    "events",
		"match":    "matches",
		"district": "districts",
		"insight":  "insights",
	}
	for _, c := range NewRootCmd().Commands() {
		alias, ok := want[c.Name()]
		if !ok {
			continue
		}
		if !contains(c.Aliases, alias) {
			t.Errorf("%s should have the alias %q, has %v", c.Name(), alias, c.Aliases)
		}
		delete(want, c.Name())
	}
	for name := range want {
		t.Errorf("no top-level command named %q", name)
	}
}

func TestPluralAliasesRunTheSameCommand(t *testing.T) {
	cases := []struct {
		args []string
		path string
	}{
		{[]string{"teams", "view", "177"}, "/team/frc177"},
		{[]string{"events", "view", "2024cthar"}, "/event/2024cthar"},
		{[]string{"matches", "view", "2024cthar_qm1"}, "/match/2024cthar_qm1"},
		{[]string{"districts", "rankings", "2024ne"}, "/district/2024ne/rankings"},
		{[]string{"insights", "notables", "--year", "2024"}, "/insights/notables/2024"},
	}
	for _, tc := range cases {
		t.Run(tc.args[0], func(t *testing.T) {
			srv := newFakeTBA(t, map[string]any{
				"/team/frc177":              teamFRC177JSON,
				"/event/2024cthar":          event2024ctharJSON,
				"/match/2024cthar_qm1":      match2024ctharQM1JSON,
				"/district/2024ne/rankings": districtRankings2024neJSON,
				"/insights/notables/2024":   notables2024JSON,
			})
			_, _, err := runCmd(t, srv, tc.args...)
			requireNoError(t, err, "")
			if got := requestPaths(t, srv); len(got) != 1 || got[0] != tc.path {
				t.Errorf("requested %v, want [%s]", got, tc.path)
			}
		})
	}
}

func TestTeamKeyNormalisesBothSpellings(t *testing.T) {
	cases := map[string]string{
		"177":     "frc177",
		"frc177":  "frc177",
		"FRC177":  "frc177",
		" 177 ":   "frc177",
		"1073":    "frc1073",
		"frc1073": "frc1073",
	}
	for in, want := range cases {
		if got := teamKey(in); got != want {
			t.Errorf("teamKey(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestTeamCommandsAcceptBothTeamSpellings(t *testing.T) {
	for _, arg := range []string{"177", "frc177"} {
		t.Run(arg, func(t *testing.T) {
			srv := newFakeTBA(t, map[string]any{
				"/team/frc177":         teamFRC177JSON,
				"/team/frc177/robots":  teamRobots177JSON,
				"/team/frc177/awards":  teamAwards177JSON,
				"/team/frc177/matches": "[]",
			})
			out, _, err := runCmd(t, srv, "team", "view", arg)
			requireNoError(t, err, "")
			obj := decodeJSON(t, out).(map[string]any)
			if obj["key"] != "frc177" {
				t.Errorf("key = %v", obj["key"])
			}
			if _, _, err := runCmd(t, srv, "team", "robots", arg); err != nil {
				t.Errorf("team robots %s: %v", arg, err)
			}
			for _, p := range requestPaths(t, srv) {
				if strings.Contains(p, "frcfrc") {
					t.Errorf("request path was double-prefixed: %s", p)
				}
			}
		})
	}
}

func TestInvalidEventKeyIsRejectedBeforeTheRequest(t *testing.T) {
	for _, key := range []string{"cthar2024", "2024", "24cthar", "2024CTHAR", "2024cthar_qm1"} {
		t.Run(key, func(t *testing.T) {
			srv := newFakeTBA(t, map[string]any{})
			_, _, err := runCmd(t, srv, "event", "view", key)
			if err == nil {
				t.Fatalf("want an error for event key %q", key)
			}
			want := `"` + key + `" is not a valid event key (expected something like 2024cthar)`
			if err.Error() != want {
				t.Errorf("error = %q, want %q", err.Error(), want)
			}
			if got := requestPaths(t, srv); len(got) != 0 {
				t.Errorf("a malformed key should not reach the API, got %v", got)
			}
		})
	}
}

func TestValidEventKeysAreAccepted(t *testing.T) {
	for _, key := range []string{"2024cthar", "2024necmp", "2015ctwat", "2021isde1"} {
		if err := validateEventKey(key); err != nil {
			t.Errorf("validateEventKey(%q) = %v, want nil", key, err)
		}
	}
}

func TestEveryEventSubcommandValidatesItsKey(t *testing.T) {
	commands := [][]string{
		{"event", "view"},
		{"event", "teams"},
		{"event", "matches"},
		{"event", "rankings"},
		{"event", "alliances"},
		{"event", "awards"},
		{"event", "oprs"},
		{"event", "district-points"},
		{"event", "predictions"},
		{"event", "insights"},
	}
	for _, args := range commands {
		t.Run(args[1], func(t *testing.T) {
			srv := newFakeTBA(t, map[string]any{})
			_, _, err := runCmd(t, srv, append(args, "not-a-key")...)
			if err == nil || !strings.Contains(err.Error(), "is not a valid event key") {
				t.Fatalf("error = %v, want an event key error", err)
			}
			if got := requestPaths(t, srv); len(got) != 0 {
				t.Errorf("a malformed key should not reach the API, got %v", got)
			}
		})
	}
}

func TestInvalidMatchKeyIsRejectedBeforeTheRequest(t *testing.T) {
	for _, key := range []string{"2024cthar", "2024cthar_zz1", "qm1", "2024cthar-qm1"} {
		t.Run(key, func(t *testing.T) {
			srv := newFakeTBA(t, map[string]any{})
			_, _, err := runCmd(t, srv, "match", "view", key)
			if err == nil {
				t.Fatalf("want an error for match key %q", key)
			}
			want := `"` + key + `" is not a valid match key (expected something like 2024cthar_qm12)`
			if err.Error() != want {
				t.Errorf("error = %q, want %q", err.Error(), want)
			}
			if got := requestPaths(t, srv); len(got) != 0 {
				t.Errorf("a malformed key should not reach the API, got %v", got)
			}
		})
	}
}

func TestValidMatchKeysAreAccepted(t *testing.T) {
	// Qualification keys carry no set number; playoff keys do.
	for _, key := range []string{
		"2024cthar_qm1", "2024cthar_qm112", "2024cthar_sf3m1",
		"2024cthar_f1m2", "2015ctwat_qf2m3", "2024cthar_ef1m1",
	} {
		if err := validateMatchKey(key); err != nil {
			t.Errorf("validateMatchKey(%q) = %v, want nil", key, err)
		}
	}
}
