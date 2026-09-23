package cmd

import (
	"strings"
	"testing"

	"github.com/the-blue-alliance/tba-cli/internal/clierr"
)

// `tba team view -5` used to answer "unknown shorthand flag: '5' in -5",
// which is about pflag's parser rather than about anything anyone typed on
// purpose. There is no -5 flag and there never will be: the argument is a
// team number.
func TestALeadingDashOnANumberIsExplained(t *testing.T) {
	for _, args := range [][]string{
		{"team", "view", "-5"},
		{"team", "view", "-177"},
		{"match", "view", "-2024"},
	} {
		t.Run(strings.Join(args, " "), func(t *testing.T) {
			_, stderr, err := runCmd(t, nil, args...)
			if err == nil {
				t.Fatalf("want an error for %v", args)
			}
			if got := clierr.ExitCode(err); got != clierr.ExitUsage {
				t.Errorf("exit code = %d, want %d", got, clierr.ExitUsage)
			}
			requireErrorContains(t, err, "looks like a negative number")
			requireErrorContains(t, err, "did you mean")
			if strings.Contains(err.Error(), "shorthand") {
				t.Errorf("pflag's wording leaked through: %v", err)
			}
			requireContains(t, stderr, "Usage:")
		})
	}
}

// The suggestion is the same text without the minus sign.
func TestALeadingDashSuggestsTheBareNumber(t *testing.T) {
	_, _, err := runCmd(t, nil, "team", "view", "-177")
	if err == nil {
		t.Fatal("want an error")
	}
	requireErrorContains(t, err, `did you mean "177"`)
}

// A genuinely unknown shorthand is still cobra's business, and still says so.
func TestAnUnknownShorthandStillReadsAsAFlag(t *testing.T) {
	_, _, err := runCmd(t, nil, "team", "view", "-Q", "177")
	if err == nil {
		t.Fatal("want an error for an unknown shorthand")
	}
	if got := clierr.ExitCode(err); got != clierr.ExitUsage {
		t.Errorf("exit code = %d, want %d", got, clierr.ExitUsage)
	}
	requireErrorContains(t, err, "shorthand")
	if strings.Contains(err.Error(), "negative number") {
		t.Errorf("-Q is not a number: %v", err)
	}
}
