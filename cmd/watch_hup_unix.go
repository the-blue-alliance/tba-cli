//go:build unix

package cmd

import (
	"io"

	"golang.org/x/sys/unix"
)

// hungUp asks the kernel whether w's file descriptor has lost its reader,
// without writing anything to it.
//
// The question is asked as POLLOUT — "may I write?" — with a zero timeout, so
// it never blocks and never disturbs the stream. Asking for no events at all
// would be tidier, and works on Linux, but Darwin leaves revents zero for a
// descriptor nobody asked anything about, which made this a no-op on macOS:
// `tba event watch ... | head -1` kept polling for the full --for.
//
// POLLNVAL is deliberately not a hang-up. It means "not a descriptor I can
// poll for this", and on Darwin /dev/null answers POLLOUT with exactly that,
// so treating it as a hang-up would end `tba event watch > /dev/null` on its
// first poll.
func hungUp(w io.Writer) bool {
	f := writerFile(w)
	if f == nil {
		return false
	}
	fds := []unix.PollFd{{Fd: int32(f.Fd()), Events: unix.POLLOUT}}
	n, err := unix.Poll(fds, 0)
	if err != nil || n <= 0 {
		return false
	}
	return fds[0].Revents&(unix.POLLHUP|unix.POLLERR) != 0
}
