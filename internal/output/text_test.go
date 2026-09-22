package output

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"
)

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
