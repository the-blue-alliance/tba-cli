package humanize

import (
	"testing"
	"time"
)

func TestAge(t *testing.T) {
	cases := []struct {
		in   time.Duration
		want string
	}{
		{0, "0s"},
		{900 * time.Millisecond, "0s"},
		{45 * time.Second, "45s"},
		{59*time.Second + 999*time.Millisecond, "59s"},
		{time.Minute, "1m"},
		{12 * time.Minute, "12m"},
		{90 * time.Minute, "1h"},
		{23 * time.Hour, "23h"},
		{25 * time.Hour, "1d"},
		{40 * 24 * time.Hour, "40d"},
		// Measured by magnitude: the caller has already chosen its wording.
		{-time.Hour, "1h"},
		{-45 * time.Second, "45s"},
	}
	for _, c := range cases {
		if got := Age(c.in); got != c.want {
			t.Errorf("Age(%v) = %q, want %q", c.in, got, c.want)
		}
	}
}
