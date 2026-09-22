package output

import (
	"fmt"
	"io"
	"os"
	"strings"
)

// IsTTY reports whether w is a terminal. It returns true only when w is an
// *os.File backed by a character device.
func IsTTY(w io.Writer) bool {
	f, ok := w.(*os.File)
	if !ok {
		return false
	}
	fi, err := f.Stat()
	if err != nil {
		return false
	}
	return fi.Mode()&os.ModeCharDevice != 0
}

// PrintKeyValue writes alternating key/value pairs with the values aligned.
// The label column is sized to the longest "key:" including the colon, so
// every value starts in the same column.
//
// An odd number of arguments is a caller bug, but not a fatal one: the trailing
// key is skipped rather than panicking on the missing value, so a half-built
// detail view still prints what it has.
func PrintKeyValue(w io.Writer, pairs ...string) {
	if len(pairs)%2 != 0 {
		pairs = pairs[:len(pairs)-1]
	}
	width := 0
	for i := 0; i+1 < len(pairs); i += 2 {
		if n := StringWidth(pairs[i]) + 1; n > width {
			width = n
		}
	}
	for i := 0; i+1 < len(pairs); i += 2 {
		fmt.Fprintf(w, "%s  %s\n", padRight(pairs[i]+":", width), pairs[i+1])
	}
}

// TeamNumberFromKey turns "frc177" into "177".
func TeamNumberFromKey(key string) string {
	return strings.TrimPrefix(key, "frc")
}

// FormatLocation joins the non-empty location parts with ", ".
func FormatLocation(city, stateProv, country string) string {
	parts := []string{}
	if city != "" {
		parts = append(parts, city)
	}
	if stateProv != "" {
		parts = append(parts, stateProv)
	}
	if country != "" {
		parts = append(parts, country)
	}
	return strings.Join(parts, ", ")
}
