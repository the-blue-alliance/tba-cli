package api

import (
	"context"
	"strings"
	"testing"
)

func TestTruncateErrorBodyKeepsShortBodies(t *testing.T) {
	body := `{"Error":"team not found"}`
	if got := truncateErrorBody(body); got != body {
		t.Errorf("truncateErrorBody(%q) = %q", body, got)
	}
}

func TestTruncateErrorBodyCutsLongBodies(t *testing.T) {
	body := strings.Repeat("a", 500)
	got := truncateErrorBody(body)
	if want := strings.Repeat("a", maxErrorBodyBytes) + "…"; got != want {
		t.Errorf("truncated body = %q", got)
	}
	if !strings.HasSuffix(got, "…") {
		t.Error("truncated body should end with an ellipsis")
	}
}

func TestTruncateErrorBodyDoesNotSplitRunes(t *testing.T) {
	// 100 three-byte runes is 300 bytes, so the cut lands mid-rune.
	body := strings.Repeat("é", 200)
	got := truncateErrorBody(body)
	trimmed := strings.TrimSuffix(got, "…")
	for _, r := range trimmed {
		if r != 'é' {
			t.Fatalf("truncation split a rune: %q", got)
		}
	}
	if len(trimmed) > maxErrorBodyBytes {
		t.Errorf("truncated body is %d bytes, want at most %d", len(trimmed), maxErrorBodyBytes)
	}
}

func TestLongAPIErrorBodyIsTruncatedInTheMessage(t *testing.T) {
	apiEnv(t)
	rec := &recorder{}
	srv := newServer(t, rec, testResponse{status: "500", body: strings.Repeat("x", 5000)})

	c, err := NewClient(srv.URL, WithRetries(0))
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	_, err = c.GetRaw(context.Background(), "/status")
	if err == nil {
		t.Fatal("want an error for a 500")
	}
	msg := err.Error()
	if !strings.HasPrefix(msg, "API error 500: ") {
		t.Errorf("the status code should come first: %v", msg)
	}
	if !strings.HasSuffix(msg, "…") {
		t.Errorf("a truncated body should end with an ellipsis: %v", msg)
	}
	if len(msg) > len("API error 500: ")+maxErrorBodyBytes+len("…") {
		t.Errorf("error message is %d bytes, want the body capped at %d", len(msg), maxErrorBodyBytes)
	}
}
