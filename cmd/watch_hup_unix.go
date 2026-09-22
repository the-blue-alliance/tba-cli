//go:build unix

package cmd

import (
	"io"

	"golang.org/x/sys/unix"
)

// hungUp asks the kernel whether w's file descriptor has lost its reader,
// without writing anything to it.
//
// POLLHUP, POLLERR and POLLNVAL are reported in revents whatever is asked for
// in events, so a poll with no events and a zero timeout is a free question:
// it never blocks and never disturbs the stream.
func hungUp(w io.Writer) bool {
	f := writerFile(w)
	if f == nil {
		return false
	}
	fds := []unix.PollFd{{Fd: int32(f.Fd())}}
	n, err := unix.Poll(fds, 0)
	if err != nil || n <= 0 {
		return false
	}
	return fds[0].Revents&(unix.POLLHUP|unix.POLLERR|unix.POLLNVAL) != 0
}
