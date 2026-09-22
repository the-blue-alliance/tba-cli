package output

import (
	"strings"
	"testing"
)

func TestRenderCSVQuotesSpecialCharacters(t *testing.T) {
	got := render(t, Table{
		Headers: []string{"Award", "Recipient", "Note"},
		Rows: [][]string{
			{"District Event Winner", "177, 1073, 5507", `He said "hi"`},
			{"Multi\nline", "plain", ""},
		},
	}, RenderOptions{Format: "csv"})

	want := "Award,Recipient,Note\n" +
		"District Event Winner,\"177, 1073, 5507\",\"He said \"\"hi\"\"\"\n" +
		"\"Multi\nline\",plain,\n"
	if got != want {
		t.Errorf("csv =\n%q\nwant\n%q", got, want)
	}
}

func TestRenderCSVNoHeaders(t *testing.T) {
	got := render(t, Table{
		Headers: []string{"A", "B"},
		Rows:    [][]string{{"1", "2"}, {"3", "4"}},
	}, RenderOptions{Format: "csv", NoHeaders: true})
	if got != "1,2\n3,4\n" {
		t.Errorf("csv = %q", got)
	}
}

func TestRenderTSVNeverQuotes(t *testing.T) {
	got := render(t, Table{
		Headers: []string{"A", "B", "C"},
		Rows: [][]string{
			{"x, y", `He said "hi"`, "tab\there"},
			{"Multi\nline", "cr\rreturn", "crlf\r\nhere"},
		},
	}, RenderOptions{Format: "tsv"})

	want := "A\tB\tC\n" +
		"x, y\tHe said \"hi\"\ttab here\n" +
		"Multi line\tcr return\tcrlf here\n"
	if got != want {
		t.Errorf("tsv =\n%q\nwant\n%q", got, want)
	}
	if strings.Count(got, "\n") != 3 {
		t.Errorf("tsv has %d lines, want one per row", strings.Count(got, "\n"))
	}
}

func TestRenderTSVKeepsOneFieldPerColumn(t *testing.T) {
	got := render(t, Table{
		Headers: []string{"Award", "Recipient"},
		Rows:    [][]string{{"Winner", "177, 1073"}},
	}, RenderOptions{Format: "tsv"})
	if got != "Award\tRecipient\nWinner\t177, 1073\n" {
		t.Errorf("tsv = %q", got)
	}
	for _, line := range lines(got) {
		if n := len(strings.Split(line, "\t")); n != 2 {
			t.Errorf("line %q has %d fields, want 2", line, n)
		}
	}
}

func TestRenderTSVNoHeaders(t *testing.T) {
	got := render(t, Table{
		Headers: []string{"A", "B"},
		Rows:    [][]string{{"1", "2"}},
	}, RenderOptions{Format: "tsv", NoHeaders: true})
	if got != "1\t2\n" {
		t.Errorf("tsv = %q", got)
	}
}
