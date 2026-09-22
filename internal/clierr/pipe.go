//go:build !windows

package clierr

import (
	"errors"
	"syscall"
)

// IsBrokenPipe reports whether err is the failure you get writing to a pipe
// whose reader has gone away, as in `tba event matches 2024cthar | head -1`.
func IsBrokenPipe(err error) bool {
	return errors.Is(err, syscall.EPIPE)
}
