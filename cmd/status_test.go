package cmd

import (
	"strings"
	"testing"
)

func TestStatusTable(t *testing.T) {
	srv := newFakeTBA(t, map[string]any{"/status": apiStatusJSON})
	out, _, err := runCmd(t, srv, "status", "--format", "table")
	requireNoError(t, err, "")

	want := "Current Season:  2024\nMax Season:      2024\nDatafeed Down:   false\n"
	if out != want {
		t.Errorf("status table =\n%q\nwant\n%q", out, want)
	}
}

func TestStatusJSON(t *testing.T) {
	srv := newFakeTBA(t, map[string]any{"/status": apiStatusJSON})
	out, _, err := runCmd(t, srv, "status", "--json")
	requireNoError(t, err, "")

	obj, ok := decodeJSON(t, out).(map[string]any)
	if !ok {
		t.Fatalf("want a JSON object, got %s", out)
	}
	if obj["current_season"] != float64(2024) {
		t.Errorf("current_season = %v", obj["current_season"])
	}
	if obj["is_datafeed_down"] != false {
		t.Errorf("is_datafeed_down = %v", obj["is_datafeed_down"])
	}
}

func TestStatusDefaultsToJSONWhenNotATTY(t *testing.T) {
	srv := newFakeTBA(t, map[string]any{"/status": apiStatusJSON})
	out, _, err := runCmd(t, srv, "status")
	requireNoError(t, err, "")
	if !strings.HasPrefix(out, "{") {
		t.Errorf("want JSON by default off a TTY, got:\n%s", out)
	}
	decodeJSON(t, out)
}

// status is key-value data, so the tabular formats fall back to JSON. This is
// the shared outputData behaviour that status used to reimplement.
func TestStatusCSVFallsBackToJSON(t *testing.T) {
	srv := newFakeTBA(t, map[string]any{"/status": apiStatusJSON})
	out, _, err := runCmd(t, srv, "status", "--format", "csv")
	requireNoError(t, err, "")
	decodeJSON(t, out)
}

func TestStatusJq(t *testing.T) {
	srv := newFakeTBA(t, map[string]any{"/status": apiStatusJSON})
	out, _, err := runCmd(t, srv, "status", "--jq", ".max_season")
	requireNoError(t, err, "")
	if strings.TrimSpace(out) != "2024" {
		t.Errorf("jq output = %q", out)
	}
}

func TestStatusRequestsStatusPath(t *testing.T) {
	srv := newFakeTBA(t, map[string]any{"/status": apiStatusJSON})
	_, _, err := runCmd(t, srv, "status")
	requireNoError(t, err, "")
	got := requestPaths(t, srv)
	if len(got) != 1 || got[0] != "/status" {
		t.Errorf("requested %v, want [/status]", got)
	}
}

// A terminal gets the human table without asking for it. This walks the real
// path through Run, whose stdout recorder must not hide the terminal.
func TestStatusDefaultsToTableOnATerminal(t *testing.T) {
	srv := newFakeTBA(t, map[string]any{"/status": apiStatusJSON})
	out, errOut, err := runCmdTTY(t, srv, "status")
	requireNoError(t, err, errOut)
	if strings.HasPrefix(strings.TrimSpace(out), "{") {
		t.Fatalf("a terminal should get the table, got JSON:\n%s", out)
	}
	requireContains(t, out, "Current Season:")
}
