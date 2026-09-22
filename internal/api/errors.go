package api

import "unicode/utf8"

// maxErrorBodyBytes caps how much of an error response body is quoted back in
// an error message. Some upstream failures answer with a full HTML page, and
// pasting that into the terminal buries the status code.
const maxErrorBodyBytes = 200

// truncateErrorBody shortens s to maxErrorBodyBytes, marking the cut with an
// ellipsis. The cut never splits a multi-byte rune.
func truncateErrorBody(s string) string {
	if len(s) <= maxErrorBodyBytes {
		return s
	}
	cut := s[:maxErrorBodyBytes]
	for len(cut) > 0 && !utf8.ValidString(cut) {
		cut = cut[:len(cut)-1]
	}
	return cut + "…"
}
