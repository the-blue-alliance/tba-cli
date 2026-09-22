package cmd

import (
	"io"

	"github.com/the-blue-alliance/tba-cli/internal/output"
)

// recordingWriter remembers the first failure from the writer underneath.
//
// Most of the printing helpers ignore what Fprint returns, so a stdout that
// went away mid-command — `tba event matches 2024cthar | head -1` — would
// otherwise look like a success. Run asks the recorder afterwards and reports
// the failure, which lets a broken pipe exit 141 instead of 0.
type recordingWriter struct {
	w   io.Writer
	err error
}

func (r *recordingWriter) Write(p []byte) (int, error) {
	if r.err != nil {
		// The stream is already gone; stop handing it more bytes.
		return 0, r.err
	}
	n, err := r.w.Write(p)
	if err != nil {
		r.err = err
	}
	return n, err
}

// Err returns the first write failure, if there was one.
func (r *recordingWriter) Err() error { return r.err }

// Unwrap exposes the wrapped writer, so that terminal detection still sees
// the real stdout underneath the recorder.
func (r *recordingWriter) Unwrap() io.Writer { return r.w }

// Compile-time check: the recorder must stay transparent to output.IsTTY.
var _ output.Unwrapper = (*recordingWriter)(nil)
