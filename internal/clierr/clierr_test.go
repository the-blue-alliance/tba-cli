package clierr

import (
	"context"
	"errors"
	"fmt"
	"syscall"
	"testing"
)

func TestExitCodeOfNilIsZero(t *testing.T) {
	if got := ExitCode(nil); got != ExitOK {
		t.Errorf("ExitCode(nil) = %d, want %d", got, ExitOK)
	}
}

func TestExitCodeOfTypedErrors(t *testing.T) {
	cases := []struct {
		err  error
		want int
	}{
		{errors.New("connection refused"), ExitFailure},
		{Usage("invalid --format %q", "xml"), ExitUsage},
		{Auth("not authenticated"), ExitAuth},
		{NotFound("no such team"), ExitNotFound},
		{context.Canceled, ExitInterrupt},
		{syscall.EPIPE, ExitBrokenPipe},
	}
	for _, c := range cases {
		if got := ExitCode(c.err); got != c.want {
			t.Errorf("ExitCode(%v) = %d, want %d", c.err, got, c.want)
		}
	}
}

func TestExitCodeSeesThroughWrapping(t *testing.T) {
	cases := []struct {
		err  error
		want int
	}{
		{fmt.Errorf("fetching team: %w", Auth("no key")), ExitAuth},
		{fmt.Errorf("fetching team: %w", NotFound("gone")), ExitNotFound},
		{fmt.Errorf("request failed: %w", context.Canceled), ExitInterrupt},
		{fmt.Errorf("writing output: %w", syscall.EPIPE), ExitBrokenPipe},
	}
	for _, c := range cases {
		if got := ExitCode(c.err); got != c.want {
			t.Errorf("ExitCode(%v) = %d, want %d", c.err, got, c.want)
		}
	}
}

func TestExitCodeRecognisesCobraParseErrors(t *testing.T) {
	messages := []string{
		`unknown command "teem" for "tba"`,
		"unknown flag: --frmat",
		"unknown shorthand flag: 'z' in -z",
		`invalid argument "abc" for "--year" flag: strconv.ParseInt: parsing "abc": invalid syntax`,
		"flag needs an argument: --format",
		`accepts 1 arg(s), received 2`,
		"requires at least 1 arg(s), only received 0",
	}
	for _, msg := range messages {
		if got := ExitCode(errors.New(msg)); got != ExitUsage {
			t.Errorf("ExitCode(%q) = %d, want %d", msg, got, ExitUsage)
		}
	}
}

func TestExitCodeOfAnOrdinaryMessageIsOne(t *testing.T) {
	for _, msg := range []string{"API error 500: boom", "request failed: dial tcp: i/o timeout"} {
		if got := ExitCode(errors.New(msg)); got != ExitFailure {
			t.Errorf("ExitCode(%q) = %d, want %d", msg, got, ExitFailure)
		}
	}
}

func TestWrapKeepsTheOriginalError(t *testing.T) {
	inner := errors.New("boom")
	wrapped := Wrap(KindUsage, inner)
	if !errors.Is(wrapped, inner) {
		t.Error("Wrap should keep the wrapped error reachable")
	}
	if wrapped.Error() != "boom" {
		t.Errorf("message = %q", wrapped.Error())
	}
	if ExitCode(wrapped) != ExitUsage {
		t.Errorf("ExitCode = %d, want %d", ExitCode(wrapped), ExitUsage)
	}
	if Wrap(KindUsage, nil) != nil {
		t.Error("Wrap(nil) should be nil")
	}
}

func TestIsKind(t *testing.T) {
	err := fmt.Errorf("context: %w", Auth("no key"))
	if !IsKind(err, KindAuth) {
		t.Error("IsKind should see through wrapping")
	}
	if IsKind(err, KindNotFound) {
		t.Error("IsKind matched the wrong kind")
	}
	if IsKind(errors.New("plain"), KindUsage) {
		t.Error("a plain error has no kind")
	}
}

func TestIsBrokenPipe(t *testing.T) {
	if !IsBrokenPipe(fmt.Errorf("write: %w", syscall.EPIPE)) {
		t.Error("EPIPE should be recognised as a broken pipe")
	}
	if IsBrokenPipe(errors.New("some other failure")) {
		t.Error("an unrelated error is not a broken pipe")
	}
}
