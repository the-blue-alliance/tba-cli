package api

import (
	"encoding/json"
	"strings"
	"unicode/utf8"
)

// maxErrorBodyBytes caps how much of an error response body is quoted back in
// an error message. Some upstream failures answer with a full HTML page, and
// pasting that into the terminal buries the status code.
const maxErrorBodyBytes = 200

// notFoundMessage turns a 404 response body into a sentence.
//
// TBA answers a 404 with {"Error": "event key: 2024chtar does not exist"},
// which already says exactly what is wrong; quoting the JSON around it only
// makes the reader parse braces to find it. Anything else — an HTML error page
// from a proxy, an empty body — keeps the old shape, because then the status
// code is the most useful thing left.
func notFoundMessage(body []byte) string {
	var payload struct {
		Error string `json:"Error"`
	}
	if err := json.Unmarshal(body, &payload); err == nil {
		if detail := strings.TrimSpace(payload.Error); detail != "" {
			return "not found: " + truncateErrorBody(detail)
		}
	}
	return "API error 404: " + truncateErrorBody(string(body))
}

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
