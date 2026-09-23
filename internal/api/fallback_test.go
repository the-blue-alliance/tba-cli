package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/the-blue-alliance/tba-cli/internal/cache"
	"github.com/the-blue-alliance/tba-cli/internal/clierr"
)

// seedCache writes a cache entry for url with a body and an age, so that the
// fallback path has something to find.
func seedCache(t *testing.T, url, body string, age time.Duration) {
	t.Helper()
	c, err := cache.New()
	if err != nil {
		t.Fatalf("cache.New: %v", err)
	}
	now := time.Now().UTC()
	e := &cache.Entry{
		URL:         url,
		FetchedAt:   now.Add(-age),
		ValidatedAt: now.Add(-age),
		Body:        json.RawMessage(body),
	}
	if err := c.WriteEntry(e); err != nil {
		t.Fatalf("WriteEntry: %v", err)
	}
}

// notedClient builds a client that collects its notes instead of printing
// them, and that never sleeps between retries.
func notedClient(t *testing.T, baseURL string, notes *[]string, opts ...Option) *Client {
	t.Helper()
	opts = append(opts, WithNotifier(func(s string) { *notes = append(*notes, s) }))
	c, err := NewClient(baseURL, opts...)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	c.sleep = func(ctx context.Context, d time.Duration) error { return ctx.Err() }
	return c
}

func onlyNote(t *testing.T, notes []string) string {
	t.Helper()
	if len(notes) != 1 {
		t.Fatalf("want exactly 1 note, got %d: %v", len(notes), notes)
	}
	return notes[0]
}

func TestStaleFallbackServesTheCachedBodyOnA503(t *testing.T) {
	apiEnv(t)
	rec := &recorder{}
	srv := newServer(t, rec, testResponse{status: "503", body: `{"Error":"down"}`})
	seedCache(t, srv.URL+"/status", `{"current_season":2024}`, 12*time.Minute)

	var notes []string
	c := notedClient(t, srv.URL, &notes, WithRetries(1))

	var status APIStatus
	if err := c.Get(t.Context(), "/status", &status); err != nil {
		t.Fatalf("Get: %v", err)
	}
	if status.CurrentSeason != 2024 {
		t.Errorf("CurrentSeason = %d, want the cached 2024", status.CurrentSeason)
	}
	note := onlyNote(t, notes)
	want := "note: /status unavailable (HTTP 503); using cached copy from 12m ago"
	if note != want {
		t.Errorf("note = %q, want %q", note, want)
	}
	if strings.HasSuffix(note, "\n") {
		t.Error("a note should not carry its own newline")
	}
}

func TestStaleFallbackAppliesTo429AndEvery5xx(t *testing.T) {
	for _, status := range []string{"429", "500", "502", "503", "504"} {
		t.Run(status, func(t *testing.T) {
			apiEnv(t)
			rec := &recorder{}
			srv := newServer(t, rec, testResponse{status: status, body: `{"Error":"nope"}`})
			seedCache(t, srv.URL+"/status", `{"current_season":2011}`, time.Hour)

			var notes []string
			c := notedClient(t, srv.URL, &notes, WithRetries(0))

			body, err := c.GetRaw(t.Context(), "/status")
			if err != nil {
				t.Fatalf("Get: %v", err)
			}
			if !strings.Contains(string(body), "2011") {
				t.Errorf("body = %s, want the cached copy", body)
			}
			if got := onlyNote(t, notes); !strings.Contains(got, "HTTP "+status) {
				t.Errorf("note = %q, want it to name HTTP %s", got, status)
			}
		})
	}
}

func TestNoStaleFallbackWithoutACachedCopy(t *testing.T) {
	apiEnv(t)
	rec := &recorder{}
	srv := newServer(t, rec, testResponse{status: "503", body: `{"Error":"down"}`})

	var notes []string
	c := notedClient(t, srv.URL, &notes, WithRetries(0))

	var status APIStatus
	err := c.Get(t.Context(), "/status", &status)
	if err == nil {
		t.Fatal("want an error when nothing is cached")
	}
	if !strings.Contains(err.Error(), "503") {
		t.Errorf("error = %v, want it to name the status", err)
	}
	if len(notes) != 0 {
		t.Errorf("want no notes, got %v", notes)
	}
}

func TestNoStaleFallbackOnA404(t *testing.T) {
	apiEnv(t)
	rec := &recorder{}
	srv := newServer(t, rec, testResponse{status: "404", body: `{"Error":"no such team"}`})
	seedCache(t, srv.URL+"/team/frc177", `{"key":"frc177"}`, time.Hour)

	var notes []string
	c := notedClient(t, srv.URL, &notes)

	if _, err := c.GetRaw(t.Context(), "/team/frc177"); err == nil {
		t.Fatal("a 404 is an answer; it must not be papered over with a cached body")
	}
	if len(notes) != 0 {
		t.Errorf("want no notes, got %v", notes)
	}
}

func TestNoStaleFallbackOnA401(t *testing.T) {
	apiEnv(t)
	rec := &recorder{}
	srv := newServer(t, rec, testResponse{status: "401", body: `{"Error":"bad key"}`})
	seedCache(t, srv.URL+"/status", `{"current_season":2024}`, time.Hour)

	var notes []string
	c := notedClient(t, srv.URL, &notes)

	if _, err := c.GetRaw(t.Context(), "/status"); err == nil {
		t.Fatal("want the auth error, not a cached body")
	}
	if len(notes) != 0 {
		t.Errorf("want no notes, got %v", notes)
	}
}

func TestNoStaleFallbackOnAnOther4xx(t *testing.T) {
	apiEnv(t)
	rec := &recorder{}
	srv := newServer(t, rec, testResponse{status: "400", body: `{"Error":"bad request"}`})
	seedCache(t, srv.URL+"/status", `{"current_season":2024}`, time.Hour)

	var notes []string
	c := notedClient(t, srv.URL, &notes)

	if _, err := c.GetRaw(t.Context(), "/status"); err == nil {
		t.Fatal("want the 400 to surface")
	}
	if len(notes) != 0 {
		t.Errorf("want no notes, got %v", notes)
	}
}

func TestStaleFallbackOnANetworkError(t *testing.T) {
	apiEnv(t)
	rec := &recorder{}
	srv := newServer(t, rec, testResponse{body: `{}`})
	url := srv.URL
	srv.Close() // nothing is listening any more

	seedCache(t, url+"/status", `{"current_season":1999}`, 3*time.Hour)

	var notes []string
	c := notedClient(t, url, &notes, WithRetries(0))

	body, err := c.GetRaw(t.Context(), "/status")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if !strings.Contains(string(body), "1999") {
		t.Errorf("body = %s, want the cached copy", body)
	}
	want := "note: /status unavailable (network error); using cached copy from 3h ago"
	if got := onlyNote(t, notes); got != want {
		t.Errorf("note = %q, want %q", got, want)
	}
}

func TestStaleFallbackOnATimeoutSaysSo(t *testing.T) {
	apiEnv(t)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		select {
		case <-time.After(2 * time.Second):
		case <-r.Context().Done():
		}
	}))
	t.Cleanup(srv.Close)
	seedCache(t, srv.URL+"/status", `{"current_season":2024}`, 25*time.Hour)

	var notes []string
	c := notedClient(t, srv.URL, &notes, WithRetries(0), WithTimeout(20*time.Millisecond))

	if _, err := c.GetRaw(t.Context(), "/status"); err != nil {
		t.Fatalf("Get: %v", err)
	}
	want := "note: /status unavailable (timeout); using cached copy from 1d ago"
	if got := onlyNote(t, notes); got != want {
		t.Errorf("note = %q, want %q", got, want)
	}
}

func TestNoStaleFallbackWhenTheCallerCancels(t *testing.T) {
	apiEnv(t)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		<-r.Context().Done()
	}))
	t.Cleanup(srv.Close)
	seedCache(t, srv.URL+"/status", `{"current_season":2024}`, time.Hour)

	var notes []string
	c := notedClient(t, srv.URL, &notes, WithRetries(0))

	ctx, cancel := context.WithCancel(t.Context())
	go func() {
		time.Sleep(20 * time.Millisecond)
		cancel()
	}()
	if _, err := c.GetRaw(ctx, "/status"); err == nil {
		t.Fatal("Ctrl-C should abort, not quietly serve a cached copy")
	}
	if len(notes) != 0 {
		t.Errorf("want no notes, got %v", notes)
	}
}

func TestNoStaleFallbackWhenTheCacheIsDisabled(t *testing.T) {
	apiEnv(t)
	rec := &recorder{}
	srv := newServer(t, rec, testResponse{status: "503", body: `{"Error":"down"}`})
	seedCache(t, srv.URL+"/status", `{"current_season":2024}`, time.Hour)

	var notes []string
	c := notedClient(t, srv.URL, &notes, WithRetries(0))
	c.SetUseCache(false)

	if _, err := c.GetRaw(t.Context(), "/status"); err == nil {
		t.Fatal("--no-cache asked for a fresh answer; want the failure")
	}
	if len(notes) != 0 {
		t.Errorf("want no notes, got %v", notes)
	}
}

func TestStaleFallbackWithoutANotifierStillServesTheBody(t *testing.T) {
	apiEnv(t)
	rec := &recorder{}
	srv := newServer(t, rec, testResponse{status: "503", body: `{"Error":"down"}`})
	seedCache(t, srv.URL+"/status", `{"current_season":2024}`, time.Hour)

	c, err := NewClient(srv.URL, WithRetries(0))
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	if _, err := c.GetRaw(t.Context(), "/status"); err != nil {
		t.Fatalf("Get: %v", err)
	}
}

func TestOfflineServesTheCacheWithoutAnyRequest(t *testing.T) {
	apiEnv(t)
	rec := &recorder{}
	srv := newServer(t, rec, testResponse{body: `{"current_season":2030}`, etag: `"e"`})
	seedCache(t, srv.URL+"/status", `{"current_season":2024}`, time.Hour)

	c, err := NewClient(srv.URL, WithOffline(true))
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	var status APIStatus
	if err := c.Get(t.Context(), "/status", &status); err != nil {
		t.Fatalf("Get: %v", err)
	}
	if status.CurrentSeason != 2024 {
		t.Errorf("CurrentSeason = %d, want the cached 2024", status.CurrentSeason)
	}
	if n := len(rec.all()); n != 0 {
		t.Errorf("offline made %d requests, want 0", n)
	}
}

// Offline output is byte-for-byte what a live run would have printed, so the
// age of the copy is the only thing that can tell the two apart.
func TestOfflineSaysHowOldTheCachedCopyIs(t *testing.T) {
	apiEnv(t)
	rec := &recorder{}
	srv := newServer(t, rec, testResponse{body: `{}`})
	seedCache(t, srv.URL+"/status", `{"current_season":2024}`, 3*time.Hour)

	var notes []string
	c := notedClient(t, srv.URL, &notes, WithOffline(true))

	if _, err := c.GetRaw(t.Context(), "/status"); err != nil {
		t.Fatalf("GetRaw: %v", err)
	}
	note := onlyNote(t, notes)
	if want := "note: offline: /status from cache (3h ago)"; note != want {
		t.Errorf("note = %q, want %q", note, want)
	}
	if strings.HasSuffix(note, "\n") {
		t.Error("a note should not carry its own newline")
	}
}

// Nothing was served, so there is nothing to date.
func TestOfflineSaysNothingWhenThereIsNoCachedCopy(t *testing.T) {
	apiEnv(t)
	rec := &recorder{}
	srv := newServer(t, rec, testResponse{body: `{}`})

	var notes []string
	c := notedClient(t, srv.URL, &notes, WithOffline(true))

	if _, err := c.GetRaw(t.Context(), "/status"); err == nil {
		t.Fatal("want an error for a path that was never fetched")
	}
	if len(notes) != 0 {
		t.Errorf("notes = %v, want none", notes)
	}
}

func TestOfflineFailsOnAnUncachedPath(t *testing.T) {
	apiEnv(t)
	rec := &recorder{}
	srv := newServer(t, rec, testResponse{body: `{}`})

	c, err := NewClient(srv.URL, WithOffline(true))
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	_, err = c.GetRaw(t.Context(), "/team/frc177")
	if err == nil {
		t.Fatal("want an error for a path that was never fetched")
	}
	want := "not cached: /team/frc177 (run without --offline to fetch)"
	if err.Error() != want {
		t.Errorf("error = %q, want %q", err.Error(), want)
	}
	// The thing asked for is not here, which is what a 404 says too, so a
	// script can treat the two alike.
	if got := clierr.ExitCode(err); got != clierr.ExitNotFound {
		t.Errorf("exit code = %d, want %d", got, clierr.ExitNotFound)
	}
	if n := len(rec.all()); n != 0 {
		t.Errorf("offline made %d requests, want 0", n)
	}
}
