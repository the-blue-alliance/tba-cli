package cmd

import (
	"bytes"
	"context"
	"errors"
	"io"
	"strings"
	"syscall"
	"testing"

	"github.com/the-blue-alliance/tba-cli/internal/clierr"
)

// brokenPipeWriter is stdout after `| head -1` has exited.
type brokenPipeWriter struct{ writes int }

func (b *brokenPipeWriter) Write(p []byte) (int, error) {
	b.writes++
	return 0, syscall.EPIPE
}

func TestRecordingWriterRemembersTheFirstFailure(t *testing.T) {
	underlying := &brokenPipeWriter{}
	w := &recordingWriter{w: underlying}

	if _, err := io.WriteString(w, "hello"); !errors.Is(err, syscall.EPIPE) {
		t.Fatalf("Write error = %v, want EPIPE", err)
	}
	if !errors.Is(w.Err(), syscall.EPIPE) {
		t.Errorf("Err() = %v, want EPIPE", w.Err())
	}

	// Later writes short-circuit rather than poking the dead stream again.
	if _, err := io.WriteString(w, "more"); !errors.Is(err, syscall.EPIPE) {
		t.Errorf("Write error = %v, want the remembered EPIPE", err)
	}
	if underlying.writes != 1 {
		t.Errorf("underlying writer was used %d times, want 1", underlying.writes)
	}
}

func TestRecordingWriterPassesThroughOnSuccess(t *testing.T) {
	var buf bytes.Buffer
	w := &recordingWriter{w: &buf}
	n, err := io.WriteString(w, "data")
	if err != nil || n != 4 {
		t.Fatalf("Write = (%d, %v)", n, err)
	}
	if w.Err() != nil {
		t.Errorf("Err() = %v, want nil", w.Err())
	}
	if buf.String() != "data" {
		t.Errorf("buffer = %q", buf.String())
	}
}

// A closed stdout must exit 141 and say nothing, whatever the format.
func TestBrokenStdoutIsReportedAsExit141(t *testing.T) {
	for _, format := range []string{"table", "json", "csv"} {
		t.Run(format, func(t *testing.T) {
			srv := newFakeTBA(t, map[string]any{"/districts/2024": districts2024JSON})
			t.Setenv("TBA_AUTH_KEY", "test-key")
			t.Setenv("TBA_CACHE_DIR", t.TempDir())
			t.Setenv("TBA_CONFIG_DIR", t.TempDir())

			root := NewRootCmd()
			var stderr bytes.Buffer
			root.SetOut(&brokenPipeWriter{})
			root.SetErr(&stderr)
			root.SetIn(strings.NewReader(""))
			root.SetArgs([]string{"--base-url", srv.URL, "district", "list",
				"--year", "2024", "--format", format})

			err := Run(context.Background(), root)
			if err == nil {
				t.Fatal("a closed stdout should be reported")
			}
			if got := clierr.ExitCode(err); got != clierr.ExitBrokenPipe {
				t.Errorf("exit code = %d (err %v), want %d", got, err, clierr.ExitBrokenPipe)
			}
			if stderr.String() != "" {
				t.Errorf("a broken pipe should print nothing, got:\n%s", stderr.String())
			}
		})
	}
}
