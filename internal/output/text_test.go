package output

import (
	"bytes"
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

// A key with no value at all must not index past the end of the slice.
func TestPrintKeyValueWithOnlyADanglingKey(t *testing.T) {
	var buf bytes.Buffer
	PrintKeyValue(&buf, "dangling")
	if buf.String() != "" {
		t.Errorf("want no output, got %q", buf.String())
	}
}

// The dangling key must not widen the label column either: it is dropped
// before the widths are measured.
func TestPrintKeyValueDanglingKeyDoesNotWidenTheLabels(t *testing.T) {
	var buf bytes.Buffer
	PrintKeyValue(&buf, "A", "1", "a very long dangling key")
	if buf.String() != "A:  1\n" {
		t.Errorf("key/value = %q", buf.String())
	}
}

func TestPrintKeyValueAlignsWideLabels(t *testing.T) {
	var buf bytes.Buffer
	PrintKeyValue(&buf, "チーム", "177", "Year", "2024")

	// "チーム:" draws 7 cells, so "Year:" is padded out to match.
	want := "チーム:  177\n" +
		"Year:    2024\n"
	if buf.String() != want {
		t.Errorf("key/value =\n%q\nwant\n%q", buf.String(), want)
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
