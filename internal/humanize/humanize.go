// Package humanize renders quantities the way someone reads them at a
// glance rather than the way a machine would write them down.
package humanize

import (
	"fmt"
	"time"
)

// Age renders a duration with a single coarse unit: "45s", "12m", "3h", "8d".
//
// One unit rather than several because these appear inside a sentence — "from
// cache (3h ago)", "starts in 12m" — where "2h 14m 06s" is more precision than
// the reader asked for and harder to take in. The magnitude is truncated, so
// "12m" means at least twelve minutes.
//
// A negative duration is measured by its magnitude, so a caller that has
// already chosen a "before"/"after" wording cannot end up printing a minus
// sign. Callers for which the future is meaningless clamp it first.
func Age(d time.Duration) string {
	if d < 0 {
		d = -d
	}
	switch {
	case d < time.Minute:
		return fmt.Sprintf("%ds", int(d/time.Second))
	case d < time.Hour:
		return fmt.Sprintf("%dm", int(d/time.Minute))
	case d < 24*time.Hour:
		return fmt.Sprintf("%dh", int(d/time.Hour))
	default:
		return fmt.Sprintf("%dd", int(d/(24*time.Hour)))
	}
}
