package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"math/rand/v2"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"golang.org/x/time/rate"

	"github.com/the-blue-alliance/tba-cli/internal/cache"
	"github.com/the-blue-alliance/tba-cli/internal/clierr"
	"github.com/the-blue-alliance/tba-cli/internal/config"
	"github.com/the-blue-alliance/tba-cli/internal/version"
)

const DefaultBaseURL = "https://www.thebluealliance.com/api/v3"

// Defaults for the transport behaviour. Commands can override them through the
// matching Option, which is how --timeout and --retries are wired up.
const (
	// DefaultTimeout bounds a single HTTP attempt.
	DefaultTimeout = 10 * time.Second
	// DefaultRetries is how many times a failed GET is retried.
	DefaultRetries = 3
	// DefaultMaxElapsed caps the wall-clock time spent on one request
	// including every retry and the waiting in between.
	DefaultMaxElapsed = 60 * time.Second
	// DefaultRateLimit and DefaultRateBurst pace outbound requests so that a
	// paging command cannot hammer the API.
	DefaultRateLimit = 10.0
	DefaultRateBurst = 10
)

const (
	initialBackoff = 500 * time.Millisecond
	maxBackoff     = 8 * time.Second
	backoffFactor  = 2
	// maxRetryAfter caps how long a server-supplied Retry-After can park us.
	maxRetryAfter = 30 * time.Second
)

// userAgent identifies the CLI to the API, e.g. "tba-cli/1.2.3".
var userAgent = version.UserAgent()

// Limiter paces outbound requests. *rate.Limiter satisfies it; tests supply
// their own.
type Limiter interface {
	Wait(ctx context.Context) error
}

type Client struct {
	http     *http.Client
	apiKey   string
	baseURL  string
	cache    *cache.Cache
	useCache bool
	offline  bool

	userAgent  string
	timeout    time.Duration
	retries    int
	maxElapsed time.Duration
	limiter    Limiter

	// notify receives one-line human notes ("using cached copy from 12m
	// ago"). The package never writes to a stream itself: the caller decides
	// where a note goes, which in the CLI is stderr.
	notify func(string)

	// Injectable so that tests can drive the retry loop without sleeping.
	sleep     func(ctx context.Context, d time.Duration) error
	randFloat func() float64
	now       func() time.Time
}

// An Option customises a Client at construction time.
type Option func(*Client)

// WithHTTPClient replaces the underlying *http.Client.
func WithHTTPClient(h *http.Client) Option {
	return func(c *Client) {
		if h != nil {
			c.http = h
		}
	}
}

// WithTimeout bounds each individual HTTP attempt. Zero or less disables the
// per-attempt timeout, leaving only the caller's context and the elapsed cap.
func WithTimeout(d time.Duration) Option {
	return func(c *Client) { c.timeout = d }
}

// WithRetries sets how many times a retryable GET failure is retried. 0
// disables retries; negative values are treated as 0.
func WithRetries(n int) Option {
	return func(c *Client) {
		if n < 0 {
			n = 0
		}
		c.retries = n
	}
}

// WithMaxElapsed caps the total wall-clock time for one request including
// retries and backoff waits.
func WithMaxElapsed(d time.Duration) Option {
	return func(c *Client) { c.maxElapsed = d }
}

// WithUserAgent overrides the User-Agent header.
func WithUserAgent(s string) Option {
	return func(c *Client) {
		if s != "" {
			c.userAgent = s
		}
	}
}

// WithRateLimit sets the client-side request rate in requests per second. The
// burst matches the rate (rounded up, at least 1) so that short bursts are
// allowed but a sustained loop settles at rps.
func WithRateLimit(rps float64) Option {
	return func(c *Client) {
		if rps <= 0 {
			c.limiter = nil
			return
		}
		burst := int(math.Ceil(rps))
		if burst < 1 {
			burst = 1
		}
		c.limiter = rate.NewLimiter(rate.Limit(rps), burst)
	}
}

// WithLimiter replaces the rate limiter outright.
func WithLimiter(l Limiter) Option {
	return func(c *Client) { c.limiter = l }
}

// WithNotifier installs a sink for one-line human notes, such as the warning
// that a stale cached copy is being served. Each note is a complete line
// without a trailing newline. Without a notifier the notes are dropped: this
// package never picks a stream to write to.
func WithNotifier(f func(string)) Option {
	return func(c *Client) { c.notify = f }
}

// WithOffline makes the client answer from the on-disk cache alone. No request
// is made, no revalidation happens, and a URL that has never been fetched is
// an error rather than a fetch.
func WithOffline(b bool) Option {
	return func(c *Client) { c.offline = b }
}

func NewClient(baseURL string, opts ...Option) (*Client, error) {
	if baseURL == "" {
		baseURL = DefaultBaseURL
	}
	key, err := config.GetAPIKey(baseURL)
	if err != nil {
		return nil, err
	}
	c := &Client{
		http:       &http.Client{},
		apiKey:     key,
		baseURL:    baseURL,
		useCache:   true,
		userAgent:  userAgent,
		timeout:    DefaultTimeout,
		retries:    DefaultRetries,
		maxElapsed: DefaultMaxElapsed,
		limiter:    rate.NewLimiter(rate.Limit(DefaultRateLimit), DefaultRateBurst),
		sleep:      sleepContext,
		randFloat:  rand.Float64,
		now:        time.Now,
	}
	if cc, err := cache.New(); err == nil {
		c.cache = cc
	}
	for _, opt := range opts {
		opt(c)
	}
	return c, nil
}

// SetUseCache toggles the on-disk response cache. Cache is enabled by default.
func (c *Client) SetUseCache(b bool) { c.useCache = b }

func (c *Client) Get(ctx context.Context, path string, result interface{}) error {
	body, err := c.fetch(ctx, path)
	if err != nil {
		return err
	}
	return json.Unmarshal(body, result)
}

func (c *Client) GetRaw(ctx context.Context, path string) (json.RawMessage, error) {
	body, err := c.fetch(ctx, path)
	if err != nil {
		return nil, err
	}
	return json.RawMessage(body), nil
}

// errStale304 marks a 304 that arrived with nothing in the cache to serve. It
// happens when the entry was pruned between the request and the response, or
// when a proxy answers a request that carried no validators.
var errStale304 = errors.New("304 without a cached response")

func (c *Client) fetch(ctx context.Context, path string) ([]byte, error) {
	url := c.baseURL + path

	var cached *cache.Entry
	if c.useCache && c.cache != nil {
		cached = c.cache.Get(url)
	}

	if c.offline {
		if cached == nil {
			return nil, fmt.Errorf("not cached: %s (run without --offline to fetch)", path)
		}
		return []byte(cached.Body), nil
	}

	body, err := c.send(ctx, url, cached)
	if errors.Is(err, errStale304) {
		// Ask again without the conditional headers so a pruned cache entry
		// or a confused proxy cannot leave us with nothing to show.
		body, err = c.send(ctx, url, nil)
		if errors.Is(err, errStale304) {
			return nil, fmt.Errorf("API returned 304 but no cached response is available")
		}
	}
	if err != nil && cached != nil && ctx.Err() == nil {
		if reason, ok := fallbackReason(err); ok {
			c.notef("note: %s unavailable (%s); using cached copy from %s ago",
				path, reason, cache.FormatAge(c.now().Sub(cached.FetchedAt)))
			return []byte(cached.Body), nil
		}
	}
	return body, err
}

// notef hands a one-line note to whatever the caller installed, if anything.
func (c *Client) notef(format string, args ...interface{}) {
	if c.notify == nil {
		return
	}
	c.notify(fmt.Sprintf(format, args...))
}

// fallbackReason reports whether a failed request is the kind where a stale
// cached copy is better than nothing, and how to describe it in one phrase.
//
// The test is "did we fail to reach an answer" rather than "did the server
// dislike the question": a timeout, a dead connection, a 429 or a 5xx say
// nothing about the resource, so last week's copy is still the best available
// answer. A 401 or a 404 is an answer, and serving a cached body over it would
// hide the very thing the user needs to see.
func fallbackReason(err error) (string, bool) {
	var he *httpError
	if errors.As(err, &he) {
		if isRetryableStatus(he.status) {
			return fmt.Sprintf("HTTP %d", he.status), true
		}
		return "", false
	}
	var ne *netError
	if errors.As(err, &ne) {
		if isTimeout(ne.err) {
			return "timeout", true
		}
		return "network error", true
	}
	return "", false
}

// isTimeout reports whether err is a deadline being hit rather than some other
// transport failure.
func isTimeout(err error) bool {
	if errors.Is(err, context.DeadlineExceeded) || os.IsTimeout(err) {
		return true
	}
	var ne net.Error
	return errors.As(err, &ne) && ne.Timeout()
}

// send runs one request, retrying retryable failures until the retry budget or
// the elapsed-time cap runs out.
func (c *Client) send(ctx context.Context, url string, cached *cache.Entry) ([]byte, error) {
	start := c.now()
	attempts := 0
	var lastErr error

	for {
		attempts++
		if c.limiter != nil {
			if err := c.limiter.Wait(ctx); err != nil {
				return nil, &netError{err: err}
			}
		}

		body, out, err := c.attempt(ctx, url, cached)
		if err == nil {
			return body, nil
		}
		lastErr = err

		if !out.retryable || attempts > c.retries || ctx.Err() != nil {
			break
		}
		delay := c.backoff(attempts, out)
		if c.now().Sub(start)+delay > c.maxElapsed {
			break
		}
		if err := c.sleep(ctx, delay); err != nil {
			break
		}
	}

	return nil, withAttempts(lastErr, attempts)
}

// outcome carries what the retry loop needs to know about a failed attempt.
type outcome struct {
	retryable  bool
	retryAfter time.Duration
	hasWait    bool
}

// attempt performs exactly one HTTP request.
func (c *Client) attempt(ctx context.Context, url string, cached *cache.Entry) ([]byte, outcome, error) {
	reqCtx := ctx
	if c.timeout > 0 {
		var cancel context.CancelFunc
		reqCtx, cancel = context.WithTimeout(ctx, c.timeout)
		defer cancel()
	}

	req, err := http.NewRequestWithContext(reqCtx, http.MethodGet, url, nil)
	if err != nil {
		return nil, outcome{}, err
	}
	req.Header.Set("X-TBA-Auth-Key", c.apiKey)
	req.Header.Set("User-Agent", c.userAgent)
	if cached != nil {
		if cached.ETag != "" {
			req.Header.Set("If-None-Match", cached.ETag)
		}
		if cached.LastModified != "" {
			req.Header.Set("If-Modified-Since", cached.LastModified)
		}
	}

	resp, err := c.http.Do(req)
	if err != nil {
		// A dead connection or a per-attempt timeout is worth another go; a
		// cancelled caller context is not.
		return nil, outcome{retryable: ctx.Err() == nil}, &netError{err: err}
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusNotModified:
		if cached != nil {
			if c.useCache && c.cache != nil {
				_ = c.cache.Touch(url)
			}
			return []byte(cached.Body), outcome{}, nil
		}
		return nil, outcome{}, errStale304
	case http.StatusOK:
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, outcome{retryable: ctx.Err() == nil}, err
		}
		if c.useCache && c.cache != nil {
			_ = c.cache.Put(url, resp.Header.Get("ETag"), resp.Header.Get("Last-Modified"), body)
		}
		return body, outcome{}, nil
	default:
		body, _ := io.ReadAll(resp.Body)
		message := truncateErrorBody(string(body))
		switch resp.StatusCode {
		case http.StatusUnauthorized:
			return nil, outcome{}, clierr.Auth("not authenticated for %s (HTTP 401): run 'tba auth login'", c.baseURL)
		case http.StatusNotFound:
			return nil, outcome{}, clierr.NotFound("API error 404: %s", message)
		}
		return nil, c.outcomeFor(resp), &httpError{status: resp.StatusCode, body: message}
	}
}

// outcomeFor decides whether a non-2xx response should be retried, and how
// long the server asked us to wait.
func (c *Client) outcomeFor(resp *http.Response) outcome {
	out := outcome{retryable: isRetryableStatus(resp.StatusCode)}
	if !out.retryable {
		return out
	}
	if d, ok := parseRetryAfter(resp.Header.Get("Retry-After"), c.now()); ok {
		out.retryAfter, out.hasWait = d, true
	}
	return out
}

// isRetryableStatus reports whether a status is worth another attempt. 4xx
// responses other than 429 are the caller's fault and never retried.
func isRetryableStatus(code int) bool {
	switch code {
	case http.StatusTooManyRequests,
		http.StatusInternalServerError,
		http.StatusBadGateway,
		http.StatusServiceUnavailable,
		http.StatusGatewayTimeout:
		return true
	}
	return false
}

// parseRetryAfter reads a Retry-After header, which is either a number of
// seconds or an HTTP-date.
func parseRetryAfter(v string, now time.Time) (time.Duration, bool) {
	v = strings.TrimSpace(v)
	if v == "" {
		return 0, false
	}
	if secs, err := strconv.Atoi(v); err == nil {
		if secs < 0 {
			return 0, false
		}
		return time.Duration(secs) * time.Second, true
	}
	if t, err := http.ParseTime(v); err == nil {
		d := t.Sub(now)
		if d < 0 {
			d = 0
		}
		return d, true
	}
	return 0, false
}

// backoff returns how long to wait before attempt+1. A Retry-After from the
// server wins (capped); otherwise it is exponential backoff with full jitter.
func (c *Client) backoff(attempt int, out outcome) time.Duration {
	if out.hasWait {
		return min(out.retryAfter, maxRetryAfter)
	}
	d := maxBackoff
	if attempt >= 1 && attempt <= 16 {
		d = initialBackoff * time.Duration(math.Pow(backoffFactor, float64(attempt-1)))
		d = min(d, maxBackoff)
	}
	return time.Duration(c.randFloat() * float64(d))
}

func sleepContext(ctx context.Context, d time.Duration) error {
	if d <= 0 {
		return ctx.Err()
	}
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-t.C:
		return nil
	}
}

// httpError is a non-2xx response. It keeps the original single-attempt
// wording and only mentions attempts once there has been more than one.
type httpError struct {
	status   int
	body     string
	attempts int
}

func (e *httpError) Error() string {
	if e.attempts > 1 {
		return fmt.Sprintf("API error %d after %d attempts: %s", e.status, e.attempts, e.body)
	}
	return fmt.Sprintf("API error %d: %s", e.status, e.body)
}

// netError is a transport-level failure: DNS, connection refused, a timeout.
type netError struct {
	err      error
	attempts int
}

func (e *netError) Error() string {
	if e.attempts > 1 {
		return fmt.Sprintf("request failed after %d attempts: %v", e.attempts, e.err)
	}
	return fmt.Sprintf("request failed: %v", e.err)
}

func (e *netError) Unwrap() error { return e.err }

// withAttempts records the attempt count on errors that can report it.
func withAttempts(err error, attempts int) error {
	if attempts <= 1 {
		return err
	}
	var he *httpError
	if errors.As(err, &he) {
		he.attempts = attempts
		return err
	}
	var ne *netError
	if errors.As(err, &ne) {
		ne.attempts = attempts
		return err
	}
	return err
}
