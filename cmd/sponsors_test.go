package cmd

import (
	"strings"
	"testing"
)

const sponsorsJSON = `{
  "sponsors": [
    {
      "name": "AndyMark",
      "url": "https://www.andymark.com",
      "logo_style": "wordmark",
      "logo_url": "https://www.thebluealliance.com/images/sponsors/andymark.png",
      "logo_url_dark": "https://www.thebluealliance.com/images/sponsors/andymark-dark.png"
    },
    {
      "name": "Example Robotics",
      "url": "https://example.com",
      "logo_style": "mark",
      "logo_url": null,
      "logo_url_dark": null
    }
  ]
}`

func TestSponsorsRequestsSponsorsPath(t *testing.T) {
	srv := newFakeTBA(t, map[string]any{"/sponsors": sponsorsJSON})
	_, _, err := runCmd(t, srv, "sponsors")
	requireNoError(t, err, "")
	if got := requestPaths(t, srv); len(got) != 1 || got[0] != "/sponsors" {
		t.Errorf("requested %v, want [/sponsors]", got)
	}
}

func TestSponsorsJSONIsTheList(t *testing.T) {
	srv := newFakeTBA(t, map[string]any{"/sponsors": sponsorsJSON})
	out, _, err := runCmd(t, srv, "sponsors", "--json")
	requireNoError(t, err, "")
	list, ok := decodeJSON(t, out).([]any)
	if !ok || len(list) != 2 {
		t.Fatalf("want a two-sponsor array, got %s", out)
	}
	second := list[1].(map[string]any)
	if second["name"] != "Example Robotics" || second["logo_url"] != nil {
		t.Errorf("second sponsor = %v", second)
	}
}

func TestSponsorsTableKeepsOrderAndBlanksMissingLogos(t *testing.T) {
	srv := newFakeTBA(t, map[string]any{"/sponsors": sponsorsJSON})
	out, _, err := runCmd(t, srv, "sponsors", "--format", "csv")
	requireNoError(t, err, "")
	want := "Name,URL,Logo Style,Logo,Dark Logo\n" +
		"AndyMark,https://www.andymark.com,wordmark,https://www.thebluealliance.com/images/sponsors/andymark.png,https://www.thebluealliance.com/images/sponsors/andymark-dark.png\n" +
		"Example Robotics,https://example.com,mark,,\n"
	if out != want {
		t.Errorf("csv =\n%s\nwant\n%s", out, want)
	}
}

func TestSponsorsEmptyListPrintsEmptyArray(t *testing.T) {
	srv := newFakeTBA(t, map[string]any{"/sponsors": `{"sponsors": []}`})
	out, _, err := runCmd(t, srv, "sponsors", "--json")
	requireNoError(t, err, "")
	if strings.TrimSpace(out) != "[]" {
		t.Errorf("empty sponsors json = %q, want []", out)
	}
}

func TestSponsorsEmptyListNotesItOnStderr(t *testing.T) {
	srv := newFakeTBA(t, map[string]any{"/sponsors": `{"sponsors": []}`})
	_, errOut, err := runCmd(t, srv, "sponsors", "--format", "table")
	requireNoError(t, err, errOut)
	if !strings.Contains(errOut, "no platinum sponsors") {
		t.Errorf("stderr = %q, want the empty note", errOut)
	}
}

func TestSponsorsRejectsArguments(t *testing.T) {
	_, _, err := runCmd(t, nil, "sponsors", "extra")
	if err == nil {
		t.Fatal("want an error for an unexpected argument")
	}
}
