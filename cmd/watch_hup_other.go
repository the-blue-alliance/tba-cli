//go:build !unix

package cmd

import "io"

// hungUp has no answer on a platform without poll(2). Windows reports a broken
// pipe on the next write instead, which is the older behaviour: a quiet watch
// keeps polling until --for elapses.
func hungUp(io.Writer) bool { return false }
