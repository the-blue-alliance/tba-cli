//go:build windows

package clierr

import (
	"errors"
	"strings"
	"syscall"
)

// errNoData is Windows' ERROR_NO_DATA: "the pipe is being closed".
const errNoData = syscall.Errno(232)

// IsBrokenPipe reports whether err is the failure you get writing to a pipe
// whose reader has gone away, as in `tba event matches 2024cthar | head -1`.
func IsBrokenPipe(err error) bool {
	if errors.Is(err, syscall.EPIPE) || errors.Is(err, errNoData) {
		return true
	}
	return strings.Contains(strings.ToLower(err.Error()), "pipe is being closed")
}
