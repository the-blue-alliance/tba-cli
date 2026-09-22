package cmd

import (
	"strings"
	"testing"
)

func TestTeamListStopsAtMaxPagesAndSaysSo(t *testing.T) {
	srv := newFakeTBA(t, map[string]any{
		"/teams/2024/0": "[" + teamFRC177JSON + "]",
		"/teams/2024/1": "[" + teamFRC1073JSON + "]",
		"/teams/2024/2": "[" + teamFRC5507JSON + "]",
		"/teams/2024/3": "[]",
	})
	out, errOut, err := runCmd(t, srv, "team", "list", "--year", "2024", "--max-pages", "2")
	requireNoError(t, err, errOut)

	wantPaths := []string{"/teams/2024/0", "/teams/2024/1"}
	gotPaths := requestPaths(t, srv)
	if len(gotPaths) != len(wantPaths) {
		t.Fatalf("requested %v, want %v", gotPaths, wantPaths)
	}
	for i := range wantPaths {
		if gotPaths[i] != wantPaths[i] {
			t.Errorf("request %d = %q, want %q", i, gotPaths[i], wantPaths[i])
		}
	}

	arr, ok := decodeJSON(t, out).([]any)
	if !ok {
		t.Fatalf("want a JSON array, got %s", out)
	}
	if len(arr) != 2 {
		t.Errorf("want the 2 fetched pages of teams, got %d", len(arr))
	}
	if errOut != "note: stopped after 2 pages; raise --max-pages to fetch more\n" {
		t.Errorf("stderr = %q", errOut)
	}
}

func TestTeamListSaysNothingWhenTheListEndsNaturally(t *testing.T) {
	srv := newFakeTBA(t, map[string]any{
		"/teams/2024/0": "[" + teamFRC177JSON + "]",
		"/teams/2024/1": "[]",
	})
	_, errOut, err := runCmd(t, srv, "team", "list", "--year", "2024", "--max-pages", "5")
	requireNoError(t, err, errOut)
	if errOut != "" {
		t.Errorf("stderr should stay empty when paging ends on its own, got %q", errOut)
	}
}

func TestTeamListCapNoteStaysOffStdout(t *testing.T) {
	srv := newFakeTBA(t, map[string]any{
		"/teams/2024/0": "[" + teamFRC177JSON + "]",
		"/teams/2024/1": "[" + teamFRC1073JSON + "]",
	})
	out, errOut, err := runCmd(t, srv, "team", "list", "--year", "2024", "--max-pages", "1", "--json")
	requireNoError(t, err, errOut)

	if strings.Contains(out, "note:") {
		t.Errorf("the cap note leaked onto stdout:\n%s", out)
	}
	decodeJSON(t, out) // stdout must stay parseable data
	requireContains(t, errOut, "note: stopped after 1 pages")
}

func TestTeamListMaxPagesDefaultsTo30(t *testing.T) {
	c := newTeamListCmd()
	n, err := c.Flags().GetInt("max-pages")
	if err != nil {
		t.Fatalf("max-pages flag: %v", err)
	}
	if n != 30 {
		t.Errorf("--max-pages default = %d, want 30", n)
	}
	if usage := c.Flags().Lookup("max-pages").Usage; usage != "Stop after this many pages of 500 teams" {
		t.Errorf("--max-pages usage = %q", usage)
	}
}

func TestTeamListMaxPagesZeroMeansNoCap(t *testing.T) {
	srv := newFakeTBA(t, map[string]any{
		"/teams/2024/0": "[" + teamFRC177JSON + "]",
		"/teams/2024/1": "[" + teamFRC1073JSON + "]",
		"/teams/2024/2": "[]",
	})
	_, errOut, err := runCmd(t, srv, "team", "list", "--year", "2024", "--max-pages", "0")
	requireNoError(t, err, errOut)
	if n := len(requestPaths(t, srv)); n != 3 {
		t.Errorf("made %d requests, want 3 (no cap)", n)
	}
	if errOut != "" {
		t.Errorf("stderr = %q, want empty", errOut)
	}
}

func TestTeamListHelpDocumentsMaxPages(t *testing.T) {
	out, _, err := runCmd(t, nil, "team", "list", "--help")
	requireNoError(t, err, "")
	requireContains(t, out, "--max-pages int")
	requireContains(t, out, "Stop after this many pages of 500 teams")
	requireContains(t, out, "(default 30)")
}
