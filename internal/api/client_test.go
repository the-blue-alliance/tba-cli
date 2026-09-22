package api

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

// apiEnv gives each test its own cache and a fixed auth key.
func apiEnv(t *testing.T) string {
	t.Helper()
	cacheDir := t.TempDir()
	t.Setenv("TBA_CACHE_DIR", cacheDir)
	t.Setenv("TBA_CONFIG_DIR", t.TempDir())
	t.Setenv("TBA_AUTH_KEY", "test-key")
	return cacheDir
}

// cacheEntriesDir is where the cache actually writes its files, one level
// below the configured cache directory.
func cacheEntriesDir(dir string) string {
	return filepath.Join(dir, "v1")
}

type recorder struct {
	mu       sync.Mutex
	requests []*http.Request
}

func (r *recorder) add(req *http.Request) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.requests = append(r.requests, req.Clone(req.Context()))
}

func (r *recorder) all() []*http.Request {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]*http.Request, len(r.requests))
	copy(out, r.requests)
	return out
}

type testResponse struct {
	status       string
	body         string
	etag         string
	lastModified string
}

// newServer serves one canned response per request, in order. The last entry
// is repeated once the list runs out.
func newServer(t *testing.T, rec *recorder, responses ...testResponse) *httptest.Server {
	t.Helper()
	var n int
	var mu sync.Mutex
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rec.add(r)
		mu.Lock()
		resp := responses[min(n, len(responses)-1)]
		n++
		mu.Unlock()

		if resp.etag != "" {
			w.Header().Set("ETag", resp.etag)
		}
		if resp.lastModified != "" {
			w.Header().Set("Last-Modified", resp.lastModified)
		}
		code := http.StatusOK
		switch resp.status {
		case "304":
			code = http.StatusNotModified
		case "404":
			code = http.StatusNotFound
		case "500":
			code = http.StatusInternalServerError
		}
		w.WriteHeader(code)
		_, _ = io.WriteString(w, resp.body)
	}))
	t.Cleanup(srv.Close)
	return srv
}

func TestNewClientUsesTheDefaultBaseURLWhenEmpty(t *testing.T) {
	apiEnv(t)
	c, err := NewClient("")
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	if c.baseURL != DefaultBaseURL {
		t.Errorf("baseURL = %q, want %q", c.baseURL, DefaultBaseURL)
	}
}

func TestNewClientFailsWithoutCredentials(t *testing.T) {
	t.Setenv("TBA_CONFIG_DIR", t.TempDir())
	t.Setenv("TBA_CACHE_DIR", t.TempDir())
	t.Setenv("TBA_AUTH_KEY", "")

	if _, err := NewClient(""); err == nil {
		t.Fatal("want an error when no API key is configured")
	}
}

func TestGetSendsAuthAndUserAgentHeaders(t *testing.T) {
	apiEnv(t)
	rec := &recorder{}
	srv := newServer(t, rec, testResponse{body: `{"current_season":2024}`})

	c, err := NewClient(srv.URL)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	var status APIStatus
	if err := c.Get("/status", &status); err != nil {
		t.Fatalf("Get: %v", err)
	}
	if status.CurrentSeason != 2024 {
		t.Errorf("CurrentSeason = %d", status.CurrentSeason)
	}

	reqs := rec.all()
	if len(reqs) != 1 {
		t.Fatalf("want 1 request, got %d", len(reqs))
	}
	if got := reqs[0].Header.Get("X-TBA-Auth-Key"); got != "test-key" {
		t.Errorf("X-TBA-Auth-Key = %q", got)
	}
	if got := reqs[0].Header.Get("User-Agent"); got != userAgent {
		t.Errorf("User-Agent = %q, want %q", got, userAgent)
	}
	if reqs[0].URL.Path != "/status" {
		t.Errorf("path = %q", reqs[0].URL.Path)
	}
}

func TestGetRawReturnsTheBodyUntouched(t *testing.T) {
	apiEnv(t)
	rec := &recorder{}
	body := `{"qual":{"high_score":88}}`
	srv := newServer(t, rec, testResponse{body: body})

	c, err := NewClient(srv.URL)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	raw, err := c.GetRaw("/event/2024cthar/insights")
	if err != nil {
		t.Fatalf("GetRaw: %v", err)
	}
	if string(raw) != body {
		t.Errorf("raw = %s, want %s", raw, body)
	}
}

func TestSuccessfulResponseIsCachedWithItsETag(t *testing.T) {
	cacheDir := apiEnv(t)
	rec := &recorder{}
	srv := newServer(t, rec, testResponse{body: `{"key":"frc177"}`, etag: `"etag-1"`})

	c, err := NewClient(srv.URL)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	if _, err := c.GetRaw("/team/frc177"); err != nil {
		t.Fatalf("GetRaw: %v", err)
	}

	entries, err := os.ReadDir(cacheEntriesDir(cacheDir))
	if err != nil {
		t.Fatalf("ReadDir: %v", err)
	}
	var found bool
	for _, e := range entries {
		if filepath.Ext(e.Name()) == ".json" {
			found = true
		}
	}
	if !found {
		t.Fatalf("nothing was cached in %s", cacheDir)
	}

	cached := c.cache.Get(srv.URL + "/team/frc177")
	if cached == nil {
		t.Fatal("cache entry not found for the request URL")
	}
	if cached.ETag != `"etag-1"` {
		t.Errorf("cached ETag = %q", cached.ETag)
	}
}

func TestConditionalRequestReturnsTheCachedBodyOn304(t *testing.T) {
	apiEnv(t)
	rec := &recorder{}
	srv := newServer(t, rec,
		testResponse{body: `{"key":"frc177","nickname":"Bobcat Robotics"}`, etag: `"etag-1"`},
		testResponse{status: "304", etag: `"etag-1"`},
	)

	c, err := NewClient(srv.URL)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	first, err := c.GetRaw("/team/frc177")
	if err != nil {
		t.Fatalf("first GetRaw: %v", err)
	}

	c2, err := NewClient(srv.URL)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	second, err := c2.GetRaw("/team/frc177")
	if err != nil {
		t.Fatalf("second GetRaw: %v", err)
	}

	var a, b map[string]any
	if err := json.Unmarshal(first, &a); err != nil {
		t.Fatalf("first body: %v", err)
	}
	if err := json.Unmarshal(second, &b); err != nil {
		t.Fatalf("second body: %v", err)
	}
	if a["nickname"] != "Bobcat Robotics" || b["nickname"] != "Bobcat Robotics" {
		t.Errorf("bodies differ: %v vs %v", a, b)
	}

	reqs := rec.all()
	if len(reqs) != 2 {
		t.Fatalf("want 2 requests, got %d", len(reqs))
	}
	if got := reqs[0].Header.Get("If-None-Match"); got != "" {
		t.Errorf("first request sent If-None-Match %q", got)
	}
	if got := reqs[1].Header.Get("If-None-Match"); got != `"etag-1"` {
		t.Errorf("second request If-None-Match = %q", got)
	}
}

func TestLastModifiedIsSentAsIfModifiedSince(t *testing.T) {
	apiEnv(t)
	rec := &recorder{}
	lastMod := "Wed, 01 May 2024 12:00:00 GMT"
	srv := newServer(t, rec,
		testResponse{body: `{"a":1}`, lastModified: lastMod},
		testResponse{body: `{"a":1}`, lastModified: lastMod},
	)

	c, err := NewClient(srv.URL)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	if _, err := c.GetRaw("/x"); err != nil {
		t.Fatalf("GetRaw: %v", err)
	}
	if _, err := c.GetRaw("/x"); err != nil {
		t.Fatalf("GetRaw: %v", err)
	}

	reqs := rec.all()
	if got := reqs[1].Header.Get("If-Modified-Since"); got != lastMod {
		t.Errorf("If-Modified-Since = %q, want %q", got, lastMod)
	}
}

func TestSetUseCacheFalseSkipsConditionalHeadersAndDoesNotWrite(t *testing.T) {
	cacheDir := apiEnv(t)
	rec := &recorder{}
	srv := newServer(t, rec, testResponse{body: `{"a":1}`, etag: `"etag-1"`})

	// Warm the cache with a caching client.
	warm, err := NewClient(srv.URL)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	if _, err := warm.GetRaw("/warm"); err != nil {
		t.Fatalf("GetRaw: %v", err)
	}
	before, err := os.ReadDir(cacheEntriesDir(cacheDir))
	if err != nil {
		t.Fatalf("ReadDir: %v", err)
	}

	c, err := NewClient(srv.URL)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	c.SetUseCache(false)
	if _, err := c.GetRaw("/warm"); err != nil {
		t.Fatalf("GetRaw: %v", err)
	}
	if _, err := c.GetRaw("/cold"); err != nil {
		t.Fatalf("GetRaw: %v", err)
	}

	reqs := rec.all()
	for i, r := range reqs[1:] {
		if got := r.Header.Get("If-None-Match"); got != "" {
			t.Errorf("request %d sent If-None-Match %q with caching off", i+1, got)
		}
		if got := r.Header.Get("If-Modified-Since"); got != "" {
			t.Errorf("request %d sent If-Modified-Since %q with caching off", i+1, got)
		}
	}

	after, err := os.ReadDir(cacheEntriesDir(cacheDir))
	if err != nil {
		t.Fatalf("ReadDir: %v", err)
	}
	if len(after) != len(before) {
		t.Errorf("cache grew from %d to %d files with caching off", len(before), len(after))
	}
}

func TestNotModifiedWithoutACachedEntryIsAnError(t *testing.T) {
	apiEnv(t)
	rec := &recorder{}
	srv := newServer(t, rec, testResponse{status: "304"})

	c, err := NewClient(srv.URL)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	_, err = c.GetRaw("/never-cached")
	if err == nil {
		t.Fatal("want an error for a 304 with no cache entry")
	}
	if !strings.Contains(err.Error(), "304") {
		t.Errorf("error = %v", err)
	}
}

func TestNon2xxIncludesTheStatusCodeAndBody(t *testing.T) {
	apiEnv(t)
	rec := &recorder{}
	srv := newServer(t, rec, testResponse{status: "404", body: `{"Error":"team not found"}`})

	c, err := NewClient(srv.URL)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	_, err = c.GetRaw("/team/frc999999")
	if err == nil {
		t.Fatal("want an error for a 404")
	}
	if !strings.Contains(err.Error(), "404") {
		t.Errorf("error should include the status code: %v", err)
	}
	if !strings.Contains(err.Error(), "team not found") {
		t.Errorf("error should include the response body: %v", err)
	}
}

func TestServerErrorIsReported(t *testing.T) {
	apiEnv(t)
	rec := &recorder{}
	srv := newServer(t, rec, testResponse{status: "500", body: "boom"})

	c, err := NewClient(srv.URL)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	if _, err := c.GetRaw("/status"); err == nil || !strings.Contains(err.Error(), "500") {
		t.Errorf("error = %v, want one mentioning 500", err)
	}
}

func TestTransportFailureIsWrapped(t *testing.T) {
	apiEnv(t)
	rec := &recorder{}
	srv := newServer(t, rec, testResponse{body: "{}"})
	url := srv.URL
	srv.Close()

	c, err := NewClient(url)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	_, err = c.GetRaw("/status")
	if err == nil {
		t.Fatal("want an error when the server is unreachable")
	}
	if !strings.Contains(err.Error(), "request failed") {
		t.Errorf("error = %v", err)
	}
}

func TestGetReportsMalformedJSON(t *testing.T) {
	apiEnv(t)
	rec := &recorder{}
	srv := newServer(t, rec, testResponse{body: "not json"})

	c, err := NewClient(srv.URL)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	var status APIStatus
	if err := c.Get("/status", &status); err == nil {
		t.Fatal("want a decode error")
	}
}

func TestNotModifiedRefreshesValidatedAt(t *testing.T) {
	apiEnv(t)
	rec := &recorder{}
	srv := newServer(t, rec,
		testResponse{body: `{"a":1}`, etag: `"etag-1"`},
		testResponse{status: "304", etag: `"etag-1"`},
	)

	c, err := NewClient(srv.URL)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	if _, err := c.GetRaw("/status"); err != nil {
		t.Fatalf("GetRaw: %v", err)
	}

	url := srv.URL + "/status"
	before := c.cache.Get(url)
	if before == nil {
		t.Fatal("nothing was cached")
	}
	if before.ValidatedAt.IsZero() {
		t.Fatal("the first response did not record ValidatedAt")
	}

	body, err := c.GetRaw("/status")
	if err != nil {
		t.Fatalf("GetRaw: %v", err)
	}
	var got map[string]int
	if err := json.Unmarshal(body, &got); err != nil {
		t.Fatalf("revalidated body is not JSON: %v", err)
	}
	if got["a"] != 1 {
		t.Errorf("revalidated body = %s", body)
	}

	after := c.cache.Get(url)
	if after == nil {
		t.Fatal("the entry was dropped on revalidation")
	}
	if !after.ValidatedAt.After(before.ValidatedAt) {
		t.Errorf("ValidatedAt = %v, want later than %v", after.ValidatedAt, before.ValidatedAt)
	}
	if !after.FetchedAt.Equal(before.FetchedAt) {
		t.Errorf("revalidation changed FetchedAt: %v -> %v", before.FetchedAt, after.FetchedAt)
	}
}

func TestNotModifiedDoesNotTouchTheCacheWhenCachingIsOff(t *testing.T) {
	apiEnv(t)
	rec := &recorder{}
	srv := newServer(t, rec, testResponse{status: "304"})

	c, err := NewClient(srv.URL)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	c.SetUseCache(false)
	if _, err := c.GetRaw("/status"); err == nil {
		t.Fatal("want an error for a 304 with no cached entry")
	}
}
