package output

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"
)

func TestPrintTableAlignsColumns(t *testing.T) {
	var buf bytes.Buffer
	PrintTable(&buf,
		[]string{"Number", "Name", "Location"},
		[][]string{
			{"177", "Bobcat Robotics", "South Windsor, CT"},
			{"1073", "The Force Team", "Hollis, NH"},
		},
	)

	want := "Number  Name             Location         \n" +
		"------  ---------------  -----------------\n" +
		"177     Bobcat Robotics  South Windsor, CT\n" +
		"1073    The Force Team   Hollis, NH       \n"
	if buf.String() != want {
		t.Errorf("table =\n%q\nwant\n%q", buf.String(), want)
	}
}

func TestPrintTableWidensColumnsToTheWidestCell(t *testing.T) {
	var buf bytes.Buffer
	PrintTable(&buf, []string{"A"}, [][]string{{"aaaaaaa"}})

	got := strings.Split(strings.TrimRight(buf.String(), "\n"), "\n")
	if got[0] != "A      " {
		t.Errorf("header = %q", got[0])
	}
	if got[1] != "-------" {
		t.Errorf("separator = %q", got[1])
	}
	if got[2] != "aaaaaaa" {
		t.Errorf("row = %q", got[2])
	}
}

func TestPrintTableWithNoRows(t *testing.T) {
	var buf bytes.Buffer
	PrintTable(&buf, []string{"Key", "Name"}, nil)
	if buf.String() != "Key  Name\n---  ----\n" {
		t.Errorf("table = %q", buf.String())
	}
}

func TestPrintTableToleratesExtraCells(t *testing.T) {
	var buf bytes.Buffer
	PrintTable(&buf, []string{"A"}, [][]string{{"1", "extra"}})
	if buf.String() != "A\n-\n1  extra\n" {
		t.Errorf("table = %q", buf.String())
	}
}

func TestPrintKeyValueAlignsValues(t *testing.T) {
	var buf bytes.Buffer
	PrintKeyValue(&buf,
		"Team", "177 - Bobcat Robotics",
		"Rookie Year", "1995",
		"Website", "http://www.bobcatrobotics.org",
	)

	want := "Team:         177 - Bobcat Robotics\n" +
		"Rookie Year:  1995\n" +
		"Website:      http://www.bobcatrobotics.org\n"
	if buf.String() != want {
		t.Errorf("key/value =\n%q\nwant\n%q", buf.String(), want)
	}

	// Every value must start in the same column.
	valueCol := -1
	for _, line := range strings.Split(strings.TrimRight(buf.String(), "\n"), "\n") {
		trimmed := strings.TrimLeft(line[strings.Index(line, ":")+1:], " ")
		col := len(line) - len(trimmed)
		if valueCol == -1 {
			valueCol = col
		} else if col != valueCol {
			t.Errorf("value in %q starts at column %d, want %d", line, col, valueCol)
		}
	}
}

func TestPrintKeyValueIgnoresATrailingOddKey(t *testing.T) {
	var buf bytes.Buffer
	PrintKeyValue(&buf, "A", "1", "dangling")
	if buf.String() != "A:  1\n" {
		t.Errorf("key/value = %q", buf.String())
	}
}

func TestPrintKeyValueEmpty(t *testing.T) {
	var buf bytes.Buffer
	PrintKeyValue(&buf)
	if buf.String() != "" {
		t.Errorf("want no output, got %q", buf.String())
	}
}

func TestPrintDelimitedQuotesSpecialCharacters(t *testing.T) {
	var buf bytes.Buffer
	err := PrintDelimited(&buf,
		[]string{"Award", "Recipient", "Note"},
		[][]string{
			{"District Event Winner", "177, 1073, 5507", `He said "hi"`},
			{"Multi\nline", "plain", ""},
		},
		',',
	)
	if err != nil {
		t.Fatalf("PrintDelimited: %v", err)
	}

	want := "Award,Recipient,Note\n" +
		"District Event Winner,\"177, 1073, 5507\",\"He said \"\"hi\"\"\"\n" +
		"\"Multi\nline\",plain,\n"
	if buf.String() != want {
		t.Errorf("csv =\n%q\nwant\n%q", buf.String(), want)
	}
}

func TestPrintDelimitedTabDoesNotQuoteCommas(t *testing.T) {
	var buf bytes.Buffer
	if err := PrintDelimited(&buf, []string{"A"}, [][]string{{"x, y"}}, '\t'); err != nil {
		t.Fatalf("PrintDelimited: %v", err)
	}
	if buf.String() != "A\nx, y\n" {
		t.Errorf("tsv = %q", buf.String())
	}
}

func TestPrintDelimitedRejectsAnInvalidDelimiter(t *testing.T) {
	var buf bytes.Buffer
	if err := PrintDelimited(&buf, []string{"A"}, [][]string{{"1"}}, '\n'); err == nil {
		t.Error("want an error for a newline delimiter")
	}
}

func TestPrintMarkdownTableEscapesPipes(t *testing.T) {
	var buf bytes.Buffer
	PrintMarkdownTable(&buf,
		[]string{"Key | Alt", "Name"},
		[][]string{
			{"2024cthar", "NE District | Hartford"},
			{"2024necmp", "New\nEngland"},
		},
	)

	want := "| Key \\| Alt | Name |\n" +
		"| --- | --- |\n" +
		"| 2024cthar | NE District \\| Hartford |\n" +
		"| 2024necmp | New England |\n"
	if buf.String() != want {
		t.Errorf("markdown =\n%q\nwant\n%q", buf.String(), want)
	}
}

func TestPrintMarkdownTableWithNoRows(t *testing.T) {
	var buf bytes.Buffer
	PrintMarkdownTable(&buf, []string{"A", "B"}, nil)
	if buf.String() != "| A | B |\n| --- | --- |\n" {
		t.Errorf("markdown = %q", buf.String())
	}
}

func TestIsTTYIsFalseForABuffer(t *testing.T) {
	if IsTTY(&bytes.Buffer{}) {
		t.Error("a bytes.Buffer is not a TTY")
	}
}

func TestIsTTYIsFalseForARegularFile(t *testing.T) {
	f, err := os.CreateTemp(t.TempDir(), "out")
	if err != nil {
		t.Fatalf("CreateTemp: %v", err)
	}
	defer f.Close()
	if IsTTY(f) {
		t.Error("a regular file is not a TTY")
	}
}

func TestIsTTYIsFalseForAClosedFile(t *testing.T) {
	f, err := os.CreateTemp(t.TempDir(), "out")
	if err != nil {
		t.Fatalf("CreateTemp: %v", err)
	}
	if err := f.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
	if IsTTY(f) {
		t.Error("a closed file is not a TTY")
	}
}

func TestIsTTYIsTrueForADeviceFile(t *testing.T) {
	f, err := os.OpenFile(os.DevNull, os.O_WRONLY, 0)
	if err != nil {
		t.Skipf("cannot open %s: %v", os.DevNull, err)
	}
	defer f.Close()
	fi, err := f.Stat()
	if err != nil || fi.Mode()&os.ModeCharDevice == 0 {
		t.Skipf("%s is not a character device here", os.DevNull)
	}
	if !IsTTY(f) {
		t.Errorf("%s is a character device, so IsTTY should be true", os.DevNull)
	}
}

func TestTeamNumberFromKey(t *testing.T) {
	cases := map[string]string{
		"frc177":  "177",
		"frc5507": "5507",
		"177":     "177",
		"":        "",
	}
	for in, want := range cases {
		if got := TeamNumberFromKey(in); got != want {
			t.Errorf("TeamNumberFromKey(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestFormatLocation(t *testing.T) {
	cases := []struct {
		city, state, country, want string
	}{
		{"South Windsor", "Connecticut", "USA", "South Windsor, Connecticut, USA"},
		{"Hartford", "", "USA", "Hartford, USA"},
		{"", "", "", ""},
		{"", "CT", "", "CT"},
	}
	for _, c := range cases {
		if got := FormatLocation(c.city, c.state, c.country); got != c.want {
			t.Errorf("FormatLocation(%q,%q,%q) = %q, want %q", c.city, c.state, c.country, got, c.want)
		}
	}
}

// wrapped decorates another writer the way the CLI's stdout recorder does.
type wrapped struct{ w io.Writer }

func (w wrapped) Write(p []byte) (int, error) { return w.w.Write(p) }
func (w wrapped) Unwrap() io.Writer           { return w.w }

// saysTerminal is a writer that declares its own terminal-ness.
type saysTerminal struct {
	bytes.Buffer
	tty bool
}

func (s *saysTerminal) IsTerminal() bool { return s.tty }

func TestIsTTYSeesThroughWrappers(t *testing.T) {
	f, err := os.OpenFile(os.DevNull, os.O_WRONLY, 0)
	if err != nil {
		t.Skipf("cannot open %s: %v", os.DevNull, err)
	}
	defer f.Close()
	// Whatever the device answers, a wrapper around it must answer the same.
	if got, want := IsTTY(wrapped{wrapped{f}}), IsTTY(f); got != want {
		t.Errorf("IsTTY(wrapped device) = %v, want %v", got, want)
	}
	if IsTTY(wrapped{&bytes.Buffer{}}) {
		t.Error("a wrapped buffer is still not a TTY")
	}
}

func TestIsTTYTrustsATerminalReporter(t *testing.T) {
	if !IsTTY(&saysTerminal{tty: true}) {
		t.Error("a writer that says it is a terminal should count as one")
	}
	if IsTTY(&saysTerminal{tty: false}) {
		t.Error("a writer that says it is not a terminal should not count as one")
	}
	if !IsTTY(wrapped{&saysTerminal{tty: true}}) {
		t.Error("the reporter should be found behind a wrapper")
	}
}
