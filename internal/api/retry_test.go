package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"golang.org/x/time/rate"
)

// fakeSleeper records the backoff waits without spending any real time.
type fakeSleeper struct {
	delays []time.Duration
	clock  *time.Time
}

func (f *fakeSleeper) sleep(ctx context.Context, d time.Duration) error {
	f.delays = append(f.delays, d)
	if f.clock != nil {
		*f.clock = f.clock.Add(d)
	}
	return ctx.Err()
}

// testClient builds a client whose retry loop takes no real time and whose
// jitter always picks the top of the window, so the backoff sequence is exact.
func testClient(t *testing.T, baseURL string, opts ...Option) (*Client, *fakeSleeper) {
	t.Helper()
	c, err := NewClient(baseURL, opts...)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	fs := &fakeSleeper{}
	c.sleep = fs.sleep
	c.randFloat = func() float64 { return 1 }
	return c, fs
}

func wantDelays(t *testing.T, got, want []time.Duration) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("slept %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("delay %d = %v, want %v (full sequence %v)", i, got[i], want[i], got)
		}
	}
}

func TestNewClientDefaults(t *testing.T) {
	apiEnv(t)
	c, err := NewClient("")
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	if c.timeout != DefaultTimeout {
		t.Errorf("timeout = %v, want %v", c.timeout, DefaultTimeout)
	}
	if c.retries != DefaultRetries {
		t.Errorf("retries = %d, want %d", c.retries, DefaultRetries)
	}
	if c.maxElapsed != DefaultMaxElapsed {
		t.Errorf("maxElapsed = %v, want %v", c.maxElapsed, DefaultMaxElapsed)
	}
	if c.userAgent != "tba-cli/dev" {
		t.Errorf("userAgent = %q, want tba-cli/dev", c.userAgent)
	}
	lim, ok := c.limiter.(*rate.Limiter)
	if !ok {
		t.Fatalf("limiter is %T, want *rate.Limiter", c.limiter)
	}
	if float64(lim.Limit()) != DefaultRateLimit || lim.Burst() != DefaultRateBurst {
		t.Errorf("limiter = %v/%d, want %v/%d", lim.Limit(), lim.Burst(), DefaultRateLimit, DefaultRateBurst)
	}
}

func TestOptionsOverrideTheDefaults(t *testing.T) {
	apiEnv(t)
	hc := &http.Client{}
	c, err := NewClient("",
		WithHTTPClient(hc),
		WithTimeout(3*time.Second),
		WithRetries(7),
		WithMaxElapsed(90*time.Second),
		WithUserAgent("tba-cli-test/9"),
		WithRateLimit(2.5),
	)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	if c.http != hc {
		t.Error("WithHTTPClient did not install the client")
	}
	if c.timeout != 3*time.Second {
		t.Errorf("timeout = %v", c.timeout)
	}
	if c.retries != 7 {
		t.Errorf("retries = %d", c.retries)
	}
	if c.maxElapsed != 90*time.Second {
		t.Errorf("maxElapsed = %v", c.maxElapsed)
	}
	if c.userAgent != "tba-cli-test/9" {
		t.Errorf("userAgent = %q", c.userAgent)
	}
	lim := c.limiter.(*rate.Limiter)
	if float64(lim.Limit()) != 2.5 || lim.Burst() != 3 {
		t.Errorf("limiter = %v/%d, want 2.5/3", lim.Limit(), lim.Burst())
	}
}

func TestNegativeRetriesMeansNone(t *testing.T) {
	apiEnv(t)
	c, err := NewClient("", WithRetries(-4))
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	if c.retries != 0 {
		t.Errorf("retries = %d, want 0", c.retries)
	}
}

func TestUserAgentOptionIsSentOnTheWire(t *testing.T) {
	apiEnv(t)
	rec := &recorder{}
	srv := newServer(t, rec, testResponse{body: "{}"})

	c, _ := testClient(t, srv.URL, WithUserAgent("custom/1.0"))
	if _, err := c.GetRaw(t.Context(), "/status"); err != nil {
		t.Fatalf("GetRaw: %v", err)
	}
	if got := rec.all()[0].Header.Get("User-Agent"); got != "custom/1.0" {
		t.Errorf("User-Agent = %q", got)
	}
}

func TestDefaultUserAgentCarriesTheVersion(t *testing.T) {
	if !strings.HasPrefix(userAgent, "tba-cli/") {
		t.Errorf("userAgent = %q, want a tba-cli/<version> prefix", userAgent)
	}
}

func TestServerErrorsAreRetriedUntilOneSucceeds(t *testing.T) {
	apiEnv(t)
	rec := &recorder{}
	srv := newServer(t, rec,
		testResponse{status: "503", body: "nope"},
		testResponse{status: "502", body: "nope"},
		testResponse{body: `{"ok":true}`},
	)

	c, fs := testClient(t, srv.URL)
	body, err := c.GetRaw(t.Context(), "/status")
	if err != nil {
		t.Fatalf("GetRaw: %v", err)
	}
	if string(body) != `{"ok":true}` {
		t.Errorf("body = %s", body)
	}
	if n := len(rec.all()); n != 3 {
		t.Errorf("made %d requests, want 3", n)
	}
	wantDelays(t, fs.delays, []time.Duration{500 * time.Millisecond, time.Second})
}

func TestRetryableStatuses(t *testing.T) {
	for _, tc := range []struct {
		status string
		want   int // total requests
	}{
		{"429", 4},
		{"500", 4},
		{"502", 4},
		{"503", 4},
		{"504", 4},
		{"400", 1},
		{"401", 1},
		{"403", 1},
		{"404", 1},
		{"418", 1},
		{"501", 1},
	} {
		t.Run(tc.status, func(t *testing.T) {
			apiEnv(t)
			rec := &recorder{}
			srv := newServer(t, rec, testResponse{status: tc.status, body: "x"})

			c, _ := testClient(t, srv.URL)
			if _, err := c.GetRaw(t.Context(), "/status"); err == nil {
				t.Fatal("want an error")
			}
			if n := len(rec.all()); n != tc.want {
				t.Errorf("status %s made %d requests, want %d", tc.status, n, tc.want)
			}
		})
	}
}

func TestBackoffIsExponentialWithJitterAndACeiling(t *testing.T) {
	apiEnv(t)
	rec := &recorder{}
	srv := newServer(t, rec, testResponse{status: "503", body: "x"})

	c, fs := testClient(t, srv.URL, WithRetries(6), WithMaxElapsed(time.Hour))
	if _, err := c.GetRaw(t.Context(), "/status"); err == nil {
		t.Fatal("want an error")
	}
	wantDelays(t, fs.delays, []time.Duration{
		500 * time.Millisecond,
		1 * time.Second,
		2 * time.Second,
		4 * time.Second,
		8 * time.Second,
		8 * time.Second,
	})
}

func TestFullJitterScalesTheWholeWindow(t *testing.T) {
	apiEnv(t)
	rec := &recorder{}
	srv := newServer(t, rec, testResponse{status: "503", body: "x"})

	c, fs := testClient(t, srv.URL, WithRetries(3))
	c.randFloat = func() float64 { return 0.25 }
	if _, err := c.GetRaw(t.Context(), "/status"); err == nil {
		t.Fatal("want an error")
	}
	wantDelays(t, fs.delays, []time.Duration{
		125 * time.Millisecond,
		250 * time.Millisecond,
		500 * time.Millisecond,
	})
}

func TestRetryAfterInSecondsWins(t *testing.T) {
	apiEnv(t)
	rec := &recorder{}
	srv := newServer(t, rec,
		testResponse{status: "429", body: "slow down", retryAfter: "2"},
		testResponse{body: "{}"},
	)

	c, fs := testClient(t, srv.URL)
	if _, err := c.GetRaw(t.Context(), "/status"); err != nil {
		t.Fatalf("GetRaw: %v", err)
	}
	wantDelays(t, fs.delays, []time.Duration{2 * time.Second})
}

func TestRetryAfterAsAnHTTPDateWins(t *testing.T) {
	apiEnv(t)
	rec := &recorder{}
	base := time.Date(2024, 5, 1, 12, 0, 0, 0, time.UTC)
	srv := newServer(t, rec,
		testResponse{status: "503", body: "x", retryAfter: base.Add(5 * time.Second).Format(http.TimeFormat)},
		testResponse{body: "{}"},
	)

	c, fs := testClient(t, srv.URL)
	c.now = func() time.Time { return base }
	if _, err := c.GetRaw(t.Context(), "/status"); err != nil {
		t.Fatalf("GetRaw: %v", err)
	}
	wantDelays(t, fs.delays, []time.Duration{5 * time.Second})
}

func TestRetryAfterInThePastWaitsNoTime(t *testing.T) {
	apiEnv(t)
	rec := &recorder{}
	base := time.Date(2024, 5, 1, 12, 0, 0, 0, time.UTC)
	srv := newServer(t, rec,
		testResponse{status: "503", body: "x", retryAfter: base.Add(-time.Minute).Format(http.TimeFormat)},
		testResponse{body: "{}"},
	)

	c, fs := testClient(t, srv.URL)
	c.now = func() time.Time { return base }
	if _, err := c.GetRaw(t.Context(), "/status"); err != nil {
		t.Fatalf("GetRaw: %v", err)
	}
	wantDelays(t, fs.delays, []time.Duration{0})
}

func TestRetryAfterIsCapped(t *testing.T) {
	apiEnv(t)
	rec := &recorder{}
	srv := newServer(t, rec,
		testResponse{status: "429", body: "x", retryAfter: "600"},
		testResponse{body: "{}"},
	)

	c, fs := testClient(t, srv.URL, WithMaxElapsed(time.Hour))
	if _, err := c.GetRaw(t.Context(), "/status"); err != nil {
		t.Fatalf("GetRaw: %v", err)
	}
	wantDelays(t, fs.delays, []time.Duration{maxRetryAfter})
}

func TestGarbageRetryAfterFallsBackToBackoff(t *testing.T) {
	apiEnv(t)
	rec := &recorder{}
	srv := newServer(t, rec,
		testResponse{status: "503", body: "x", retryAfter: "soon please"},
		testResponse{body: "{}"},
	)

	c, fs := testClient(t, srv.URL)
	if _, err := c.GetRaw(t.Context(), "/status"); err != nil {
		t.Fatalf("GetRaw: %v", err)
	}
	wantDelays(t, fs.delays, []time.Duration{500 * time.Millisecond})
}

func TestFinalErrorReportsTheAttemptCount(t *testing.T) {
	apiEnv(t)
	rec := &recorder{}
	srv := newServer(t, rec, testResponse{status: "503", body: "unavailable"})

	c, _ := testClient(t, srv.URL)
	_, err := c.GetRaw(t.Context(), "/status")
	if err == nil {
		t.Fatal("want an error")
	}
	if got := err.Error(); got != "API error 503 after 4 attempts: unavailable" {
		t.Errorf("error = %q", got)
	}
}

func TestASingleAttemptKeepsThePlainErrorFormat(t *testing.T) {
	apiEnv(t)
	rec := &recorder{}
	srv := newServer(t, rec, testResponse{status: "503", body: "unavailable"})

	c, _ := testClient(t, srv.URL, WithRetries(0))
	_, err := c.GetRaw(t.Context(), "/status")
	if err == nil {
		t.Fatal("want an error")
	}
	if got := err.Error(); got != "API error 503: unavailable" {
		t.Errorf("error = %q", got)
	}
	if n := len(rec.all()); n != 1 {
		t.Errorf("made %d requests with retries disabled, want 1", n)
	}
}

func TestNetworkErrorsAreRetriedAndCounted(t *testing.T) {
	apiEnv(t)
	rec := &recorder{}
	srv := newServer(t, rec, testResponse{body: "{}"})
	url := srv.URL
	srv.Close()

	c, fs := testClient(t, url, WithRetries(2))
	_, err := c.GetRaw(t.Context(), "/status")
	if err == nil {
		t.Fatal("want an error when the server is unreachable")
	}
	if !strings.Contains(err.Error(), "request failed after 3 attempts") {
		t.Errorf("error = %v", err)
	}
	wantDelays(t, fs.delays, []time.Duration{500 * time.Millisecond, time.Second})
}

func TestMaxElapsedStopsTheRetryLoop(t *testing.T) {
	apiEnv(t)
	rec := &recorder{}
	srv := newServer(t, rec, testResponse{status: "503", body: "x"})

	now := time.Date(2024, 5, 1, 12, 0, 0, 0, time.UTC)
	c, fs := testClient(t, srv.URL, WithRetries(10), WithMaxElapsed(2*time.Second))
	fs.clock = &now
	c.now = func() time.Time { return now }

	if _, err := c.GetRaw(t.Context(), "/status"); err == nil {
		t.Fatal("want an error")
	}
	// 500ms + 1s fit inside the 2s budget; the next 2s wait would not.
	wantDelays(t, fs.delays, []time.Duration{500 * time.Millisecond, time.Second})
	if n := len(rec.all()); n != 3 {
		t.Errorf("made %d requests, want 3", n)
	}
}

func TestCancelledContextStopsRetrying(t *testing.T) {
	apiEnv(t)
	rec := &recorder{}
	srv := newServer(t, rec, testResponse{status: "503", body: "x"})

	ctx, cancel := context.WithCancel(t.Context())
	c, fs := testClient(t, srv.URL, WithRetries(5))
	c.sleep = func(ctx context.Context, d time.Duration) error {
		fs.delays = append(fs.delays, d)
		cancel() // the user hit Ctrl-C while we were waiting
		return ctx.Err()
	}

	if _, err := c.GetRaw(ctx, "/status"); err == nil {
		t.Fatal("want an error")
	}
	if n := len(rec.all()); n != 1 {
		t.Errorf("made %d requests after cancellation, want 1", n)
	}
}

func TestAlreadyCancelledContextMakesNoRequest(t *testing.T) {
	apiEnv(t)
	rec := &recorder{}
	srv := newServer(t, rec, testResponse{body: "{}"})

	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	c, _ := testClient(t, srv.URL)
	if _, err := c.GetRaw(ctx, "/status"); err == nil {
		t.Fatal("want an error for a cancelled context")
	}
	if n := len(rec.all()); n != 0 {
		t.Errorf("made %d requests with a cancelled context, want 0", n)
	}
}

func TestPerAttemptTimeoutAborts(t *testing.T) {
	apiEnv(t)
	release := make(chan struct{})
	var hits int
	var mu sync.Mutex
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		hits++
		mu.Unlock()
		select {
		case <-release:
		case <-r.Context().Done():
		}
	}))
	t.Cleanup(func() {
		close(release)
		srv.Close()
	})

	c, _ := testClient(t, srv.URL, WithTimeout(20*time.Millisecond), WithRetries(0))
	_, err := c.GetRaw(t.Context(), "/slow")
	if err == nil {
		t.Fatal("want a timeout error")
	}
	if !strings.Contains(err.Error(), "request failed") {
		t.Errorf("error = %v", err)
	}
	mu.Lock()
	defer mu.Unlock()
	if hits != 1 {
		t.Errorf("server saw %d requests, want 1", hits)
	}
}

// countingLimiter records how often the retry loop asked for permission.
type countingLimiter struct {
	mu    sync.Mutex
	waits int
}

func (l *countingLimiter) Wait(ctx context.Context) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.waits++
	return ctx.Err()
}

func (l *countingLimiter) count() int {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.waits
}

func TestTheLimiterIsConsultedBeforeEveryAttempt(t *testing.T) {
	apiEnv(t)
	rec := &recorder{}
	srv := newServer(t, rec,
		testResponse{status: "503", body: "x"},
		testResponse{status: "503", body: "x"},
		testResponse{body: "{}"},
	)

	lim := &countingLimiter{}
	c, _ := testClient(t, srv.URL, WithLimiter(lim))
	if _, err := c.GetRaw(t.Context(), "/status"); err != nil {
		t.Fatalf("GetRaw: %v", err)
	}
	if got := lim.count(); got != 3 {
		t.Errorf("limiter.Wait called %d times, want 3 (one per attempt)", got)
	}
}

func TestEachRequestWaitsOnTheLimiter(t *testing.T) {
	apiEnv(t)
	rec := &recorder{}
	srv := newServer(t, rec, testResponse{body: "{}"})

	lim := &countingLimiter{}
	c, _ := testClient(t, srv.URL, WithLimiter(lim))
	for i := 0; i < 4; i++ {
		if _, err := c.GetRaw(t.Context(), "/status"); err != nil {
			t.Fatalf("GetRaw: %v", err)
		}
	}
	if got := lim.count(); got != 4 {
		t.Errorf("limiter.Wait called %d times, want 4", got)
	}
}

func TestRateLimitSpacesOutQuickCalls(t *testing.T) {
	apiEnv(t)
	rec := &recorder{}
	srv := newServer(t, rec, testResponse{body: "{}"})

	// 200 req/s with a burst of one: the second and third calls each wait 5ms.
	c, _ := testClient(t, srv.URL, WithLimiter(rate.NewLimiter(rate.Limit(200), 1)))
	start := time.Now()
	for i := 0; i < 3; i++ {
		if _, err := c.GetRaw(t.Context(), "/status"); err != nil {
			t.Fatalf("GetRaw: %v", err)
		}
	}
	if elapsed := time.Since(start); elapsed < 9*time.Millisecond {
		t.Errorf("3 calls took %v, want at least ~10ms of pacing", elapsed)
	}
}

func TestLimiterErrorsAreReported(t *testing.T) {
	apiEnv(t)
	rec := &recorder{}
	srv := newServer(t, rec, testResponse{body: "{}"})

	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	c, _ := testClient(t, srv.URL, WithLimiter(rate.NewLimiter(rate.Limit(1), 1)))
	_, err := c.GetRaw(ctx, "/status")
	if err == nil {
		t.Fatal("want an error when the limiter cannot admit the request")
	}
	if !strings.Contains(err.Error(), "request failed") {
		t.Errorf("error = %v", err)
	}
}

func TestRateLimitZeroDisablesTheLimiter(t *testing.T) {
	apiEnv(t)
	c, err := NewClient("", WithRateLimit(0))
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	if c.limiter != nil {
		t.Errorf("limiter = %v, want nil", c.limiter)
	}
}

func TestParseRetryAfter(t *testing.T) {
	now := time.Date(2024, 5, 1, 12, 0, 0, 0, time.UTC)
	for _, tc := range []struct {
		in   string
		want time.Duration
		ok   bool
	}{
		{"", 0, false},
		{"3", 3 * time.Second, true},
		{" 3 ", 3 * time.Second, true},
		{"0", 0, true},
		{"-1", 0, false},
		{"nonsense", 0, false},
		{now.Add(30 * time.Second).Format(http.TimeFormat), 30 * time.Second, true},
		{now.Add(-30 * time.Second).Format(http.TimeFormat), 0, true},
	} {
		got, ok := parseRetryAfter(tc.in, now)
		if ok != tc.ok || got != tc.want {
			t.Errorf("parseRetryAfter(%q) = %v, %v; want %v, %v", tc.in, got, ok, tc.want, tc.ok)
		}
	}
}
