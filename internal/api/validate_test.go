package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/the-blue-alliance/tba-cli/internal/clierr"
)

func TestValidateKeyAcceptsAKeyTheAPILikes(t *testing.T) {
	var gotPath, gotKey, gotAgent string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotKey = r.Header.Get("X-TBA-Auth-Key")
		gotAgent = r.Header.Get("User-Agent")
		_, _ = w.Write([]byte(`{"current_season":2024}`))
	}))
	t.Cleanup(srv.Close)

	if err := ValidateKey(context.Background(), srv.URL, "good-key"); err != nil {
		t.Fatalf("ValidateKey: %v", err)
	}
	if gotPath != "/status" {
		t.Errorf("path = %q, want /status", gotPath)
	}
	if gotKey != "good-key" {
		t.Errorf("X-TBA-Auth-Key = %q", gotKey)
	}
	if gotAgent != userAgent {
		t.Errorf("User-Agent = %q, want %q", gotAgent, userAgent)
	}
}

func TestValidateKeyRejectsA401(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	t.Cleanup(srv.Close)

	err := ValidateKey(context.Background(), srv.URL, "bad-key")
	if err == nil {
		t.Fatal("want an error for a rejected key")
	}
	if !clierr.IsKind(err, clierr.KindAuth) {
		t.Errorf("error should be an auth error: %v", err)
	}
	if !strings.Contains(err.Error(), srv.URL) {
		t.Errorf("error should name the base URL: %v", err)
	}
}

func TestValidateKeyReportsOtherFailures(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("boom"))
	}))
	t.Cleanup(srv.Close)

	err := ValidateKey(context.Background(), srv.URL, "any")
	if err == nil || !strings.Contains(err.Error(), "500") {
		t.Fatalf("error = %v, want one mentioning 500", err)
	}
	if clierr.IsKind(err, clierr.KindAuth) {
		t.Error("a 500 is not an auth failure")
	}
}

func TestValidateKeyHonoursACancelledContext(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		<-r.Context().Done()
	}))
	t.Cleanup(srv.Close)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := ValidateKey(ctx, srv.URL, "any"); err == nil {
		t.Fatal("want an error for a cancelled context")
	}
}
