//go:build unix

package cmd

import (
	"os"
	"testing"
)

// The probe used to poll for no events at all, which Darwin answers with an
// empty revents, so a hang-up was never seen on macOS. A pipe whose reader has
// gone is the case `tba event watch ... | head -1` produces.
func TestHungUpSeesAClosedPipe(t *testing.T) {
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("Pipe: %v", err)
	}
	defer w.Close()
	if hungUp(w) {
		t.Error("a pipe with a reader on it is not hung up")
	}
	if err := r.Close(); err != nil {
		t.Fatalf("closing the read end: %v", err)
	}
	if !hungUp(w) {
		t.Error("a pipe whose reader has gone should read as hung up")
	}
}

// `tba event watch > /dev/null` is a legitimate way to run a watch for its
// exit code alone. On Darwin /dev/null answers a POLLOUT poll with POLLNVAL,
// so counting that as a hang-up would stop the watch on its first poll.
func TestHungUpIsFalseForDevNull(t *testing.T) {
	f, err := os.OpenFile(os.DevNull, os.O_WRONLY, 0)
	if err != nil {
		t.Skipf("cannot open %s: %v", os.DevNull, err)
	}
	defer f.Close()
	if hungUp(f) {
		t.Errorf("%s is not a reader that hung up", os.DevNull)
	}
}

// A regular file has no reader to lose, and a writer with no descriptor at all
// — a buffer in a test — can never be asked.
func TestHungUpIsFalseWithoutAHangUp(t *testing.T) {
	f, err := os.CreateTemp(t.TempDir(), "out")
	if err != nil {
		t.Fatalf("CreateTemp: %v", err)
	}
	defer f.Close()
	if hungUp(f) {
		t.Error("a regular file is not hung up")
	}
	if hungUp(nopWriter{}) {
		t.Error("a writer with no file descriptor is not hung up")
	}
}
