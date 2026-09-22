package output

import (
	"bytes"
	"strings"
	"testing"
)

func TestParseColorMode(t *testing.T) {
	cases := map[string]ColorMode{
		"":       ColorAuto,
		"auto":   ColorAuto,
		"always": ColorAlways,
		"never":  ColorNever,
	}
	for in, want := range cases {
		got, err := ParseColorMode(in)
		if err != nil {
			t.Errorf("ParseColorMode(%q): %v", in, err)
			continue
		}
		if got != want {
			t.Errorf("ParseColorMode(%q) = %v, want %v", in, got, want)
		}
	}
}

func TestParseColorModeRejectsUnknownValues(t *testing.T) {
	_, err := ParseColorMode("sometimes")
	if err == nil {
		t.Fatal("want an error for an unknown --color value")
	}
	for _, want := range []string{"sometimes", "auto, always, never"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error %v, want it to mention %q", err, want)
		}
	}
}

func TestColorModeString(t *testing.T) {
	cases := map[ColorMode]string{ColorAuto: "auto", ColorAlways: "always", ColorNever: "never"}
	for mode, want := range cases {
		if got := mode.String(); got != want {
			t.Errorf("%d.String() = %q, want %q", mode, got, want)
		}
	}
}

func TestColorize(t *testing.T) {
	if got := Colorize("hi", Bold, true); got != "\x1b[1mhi\x1b[0m" {
		t.Errorf("Colorize on = %q", got)
	}
	if got := Colorize("hi", Red, false); got != "hi" {
		t.Errorf("Colorize off = %q", got)
	}
	if got := Colorize("", Blue, true); got != "" {
		t.Errorf("Colorize of an empty string = %q, want it left alone", got)
	}
	if got := Colorize("hi", "", true); got != "hi" {
		t.Errorf("Colorize with no code = %q, want it left alone", got)
	}
	if got := Colorize("hi", Dim, true); got != "\x1b[2mhi\x1b[0m" {
		t.Errorf("Colorize dim = %q", got)
	}
}

func TestColorEnabledResolutionMatrix(t *testing.T) {
	cases := []struct {
		name  string
		mode  ColorMode
		isTTY bool
		env   map[string]string
		want  bool
	}{
		{name: "auto on a terminal", mode: ColorAuto, isTTY: true, want: true},
		{name: "auto when piped", mode: ColorAuto, isTTY: false, want: false},
		{name: "never on a terminal", mode: ColorNever, isTTY: true, want: false},
		{name: "always when piped", mode: ColorAlways, isTTY: false, want: true},
		{
			name: "NO_COLOR=1 disables auto on a terminal",
			mode: ColorAuto, isTTY: true, env: map[string]string{"NO_COLOR": "1"}, want: false,
		},
		{
			name: "NO_COLOR with any value disables auto",
			mode: ColorAuto, isTTY: true, env: map[string]string{"NO_COLOR": "0"}, want: false,
		},
		{
			name: "an empty NO_COLOR is not an opt-out",
			mode: ColorAuto, isTTY: true, env: map[string]string{"NO_COLOR": ""}, want: true,
		},
		{
			name: "NO_COLOR does not override --color always",
			mode: ColorAlways, isTTY: false, env: map[string]string{"NO_COLOR": "1"}, want: true,
		},
		{
			name: "TERM=dumb disables auto on a terminal",
			mode: ColorAuto, isTTY: true, env: map[string]string{"TERM": "dumb"}, want: false,
		},
		{
			name: "TERM=xterm leaves auto alone",
			mode: ColorAuto, isTTY: true, env: map[string]string{"TERM": "xterm-256color"}, want: true,
		},
		{
			name: "CLICOLOR_FORCE colors a pipe in auto mode",
			mode: ColorAuto, isTTY: false, env: map[string]string{"CLICOLOR_FORCE": "1"}, want: true,
		},
		{
			name: "CLICOLOR_FORCE=0 forces nothing",
			mode: ColorAuto, isTTY: false, env: map[string]string{"CLICOLOR_FORCE": "0"}, want: false,
		},
		{
			name: "an empty CLICOLOR_FORCE forces nothing",
			mode: ColorAuto, isTTY: false, env: map[string]string{"CLICOLOR_FORCE": ""}, want: false,
		},
		{
			name: "NO_COLOR beats CLICOLOR_FORCE",
			mode: ColorAuto, isTTY: true,
			env:  map[string]string{"NO_COLOR": "1", "CLICOLOR_FORCE": "1"},
			want: false,
		},
		{
			name: "TERM=dumb beats CLICOLOR_FORCE",
			mode: ColorAuto, isTTY: true,
			env:  map[string]string{"TERM": "dumb", "CLICOLOR_FORCE": "1"},
			want: false,
		},
		{
			name: "--color never beats CLICOLOR_FORCE",
			mode: ColorNever, isTTY: true, env: map[string]string{"CLICOLOR_FORCE": "1"}, want: false,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			// Start from a known environment so the developer's own shell
			// cannot change the answer.
			t.Setenv("NO_COLOR", "")
			t.Setenv("CLICOLOR_FORCE", "")
			t.Setenv("TERM", "xterm")
			for k, v := range c.env {
				t.Setenv(k, v)
			}
			if got := ColorEnabled(c.mode, c.isTTY); got != c.want {
				t.Errorf("ColorEnabled(%v, isTTY=%v) = %v, want %v", c.mode, c.isTTY, got, c.want)
			}
		})
	}
}

func TestColorEnabledForABufferIsNeverAuto(t *testing.T) {
	t.Setenv("NO_COLOR", "")
	t.Setenv("CLICOLOR_FORCE", "")
	t.Setenv("TERM", "xterm")
	var buf bytes.Buffer
	if ColorEnabledFor(&buf, ColorAuto) {
		t.Error("a bytes.Buffer is not a terminal, so auto must not color it")
	}
	if !ColorEnabledFor(&buf, ColorAlways) {
		t.Error("--color always must color even a buffer")
	}
}

func TestRenderTableBoldsTheHeaderWhenColored(t *testing.T) {
	t.Setenv("NO_COLOR", "")
	t.Setenv("CLICOLOR_FORCE", "")
	t.Setenv("TERM", "xterm")

	got := render(t, Table{
		Headers: []string{"Number", "Name"},
		Rows:    [][]string{{"177", "Bobcats"}},
	}, RenderOptions{Format: "table", Color: ColorAlways})

	want := "\x1b[1mNumber  Name   \x1b[0m\n" +
		"------  -------\n" +
		"177     Bobcats\n"
	if got != want {
		t.Errorf("table =\n%q\nwant\n%q", got, want)
	}
}

func TestRenderTableHasNoEscapesByDefault(t *testing.T) {
	got := render(t, Table{
		Headers: []string{"Number"},
		Rows:    [][]string{{"177"}},
	}, RenderOptions{Format: "table"})
	if strings.Contains(got, "\x1b") {
		t.Errorf("table into a buffer must be plain text, got %q", got)
	}
}

func TestRenderDataFormatsAreNeverColored(t *testing.T) {
	for _, format := range []string{"csv", "tsv", "markdown"} {
		got := render(t, Table{
			Headers: []string{"A"},
			Rows:    [][]string{{"1"}},
		}, RenderOptions{Format: format, Color: ColorAlways})
		if strings.Contains(got, "\x1b") {
			t.Errorf("%s output must never carry ANSI escapes, got %q", format, got)
		}
	}
}

func TestRenderNoHeadersEmitsNoEscapes(t *testing.T) {
	got := render(t, Table{
		Headers: []string{"A"},
		Rows:    [][]string{{"1"}},
	}, RenderOptions{Format: "table", NoHeaders: true, Color: ColorAlways})
	if got != "1\n" {
		t.Errorf("table = %q, want the bare row with no header escapes", got)
	}
}
