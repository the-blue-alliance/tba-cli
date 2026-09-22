// Package clierr classifies command failures so that main can turn them into
// the exit codes a script expects.
//
//	0   success
//	1   runtime or network failure
//	2   usage: a bad flag, a bad argument, an unknown command
//	4   authentication required
//	5   the thing asked for does not exist
//	130 interrupted (Ctrl-C)
//	141 stdout closed early (a broken pipe)
package clierr

import (
	"context"
	"errors"
	"fmt"
	"strings"
)

// Exit codes. They are part of the CLI's contract with scripts.
const (
	ExitOK         = 0
	ExitFailure    = 1
	ExitUsage      = 2
	ExitAuth       = 4
	ExitNotFound   = 5
	ExitInterrupt  = 130
	ExitBrokenPipe = 141
)

// Kind is the category of a failure.
type Kind int

const (
	// KindFailure is an ordinary runtime or network failure.
	KindFailure Kind = iota
	KindUsage
	KindAuth
	KindNotFound
)

// ExitCode returns the process exit code for a kind.
func (k Kind) ExitCode() int {
	switch k {
	case KindUsage:
		return ExitUsage
	case KindAuth:
		return ExitAuth
	case KindNotFound:
		return ExitNotFound
	default:
		return ExitFailure
	}
}

// Error is an error carrying the category it should be reported as.
type Error struct {
	Kind Kind
	Err  error
}

func (e *Error) Error() string { return e.Err.Error() }
func (e *Error) Unwrap() error { return e.Err }

func newf(kind Kind, format string, a ...any) error {
	return &Error{Kind: kind, Err: fmt.Errorf(format, a...)}
}

// Usage reports a mistake in how the command was invoked (exit 2).
func Usage(format string, a ...any) error { return newf(KindUsage, format, a...) }

// Auth reports that the command needs credentials it does not have (exit 4).
func Auth(format string, a ...any) error { return newf(KindAuth, format, a...) }

// NotFound reports that the requested resource does not exist (exit 5).
func NotFound(format string, a ...any) error { return newf(KindNotFound, format, a...) }

// Wrap tags an existing error with a kind, keeping it unwrappable.
func Wrap(kind Kind, err error) error {
	if err == nil {
		return nil
	}
	return &Error{Kind: kind, Err: err}
}

// IsKind reports whether err was classified as kind.
func IsKind(err error, kind Kind) bool {
	var e *Error
	return errors.As(err, &e) && e.Kind == kind
}

// usagePhrases are the parts of cobra's own parse errors. Cobra reports these
// as plain errors, so the text is the only thing left to recognise them by.
var usagePhrases = []string{
	"unknown command",
	"unknown flag",
	"unknown shorthand flag",
	"invalid argument",
	"flag needs an argument",
	"required flag(s)",
	"arg(s), received",
	"accepts between",
	"requires at least",
	"requires at most",
	"subcommand is required",
}

// LooksLikeUsage reports whether an error message is one of cobra's parse
// complaints.
func LooksLikeUsage(msg string) bool {
	for _, phrase := range usagePhrases {
		if strings.Contains(msg, phrase) {
			return true
		}
	}
	return false
}

// Silent reports whether a failure should be left unsaid.
//
// Two of them should. A reader that hung up is gone and cannot read the
// complaint anyway. Ctrl-C is not news either: the person who pressed it knows
// what happened, and "Error: context canceled" reads like a bug in the tool
// rather than an answer to what they just asked for. Both still carry their
// exit code, so a script can tell what happened.
func Silent(err error) bool {
	if err == nil {
		return false
	}
	code := ExitCode(err)
	return code == ExitBrokenPipe || code == ExitInterrupt
}

// ExitCode maps an error returned by the command tree to a process exit code.
func ExitCode(err error) int {
	if err == nil {
		return ExitOK
	}
	if IsBrokenPipe(err) {
		return ExitBrokenPipe
	}
	if errors.Is(err, context.Canceled) {
		return ExitInterrupt
	}
	var e *Error
	if errors.As(err, &e) {
		return e.Kind.ExitCode()
	}
	if LooksLikeUsage(err.Error()) {
		return ExitUsage
	}
	return ExitFailure
}
