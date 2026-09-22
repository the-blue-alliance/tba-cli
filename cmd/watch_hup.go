package cmd

import (
	"fmt"
	"io"
	"os"
	"syscall"

	"github.com/the-blue-alliance/tba-cli/internal/output"
)

// stdoutHungUp reports whether the reader on the other end of stdout has gone
// away. It is a variable so that a test can drive the hang-up path without a
// pipe to break.
//
// It matters only to `event watch`. Every other command writes something and
// exits, so a closed pipe surfaces as a failed write; a watch can sit quiet for
// an hour between polls, and `tba event watch ... | head -1` would otherwise
// leave a poller running against a pipeline nobody is reading.
var stdoutHungUp = hungUp

// writerFile digs an *os.File out from behind the writers this CLI wraps
// stdout in. Anything else — a buffer in a test, a strings.Builder — has no
// file descriptor to ask about, and is never treated as hung up.
func writerFile(w io.Writer) *os.File {
	for w != nil {
		switch v := w.(type) {
		case *os.File:
			return v
		case output.Unwrapper:
			w = v.Unwrap()
		default:
			return nil
		}
	}
	return nil
}

// errStdoutClosed is what a hang-up is reported as: EPIPE, which the exit-code
// mapping turns into 141 and main prints nothing for. The reader has already
// left; a message would only go to a terminal that did not ask for one.
func errStdoutClosed() error {
	return fmt.Errorf("stdout closed by the reader: %w", syscall.EPIPE)
}
