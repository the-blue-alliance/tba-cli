package api

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/the-blue-alliance/tba-cli/internal/cache"
	"github.com/the-blue-alliance/tba-cli/internal/clierr"
	"github.com/the-blue-alliance/tba-cli/internal/config"
)

const DefaultBaseURL = "https://www.thebluealliance.com/api/v3"
const userAgent = "tba-cli"

type Client struct {
	http     *http.Client
	apiKey   string
	baseURL  string
	cache    *cache.Cache
	useCache bool
}

func NewClient(baseURL string) (*Client, error) {
	if baseURL == "" {
		baseURL = DefaultBaseURL
	}
	key, err := config.GetAPIKey(baseURL)
	if err != nil {
		return nil, err
	}
	c := &Client{http: &http.Client{}, apiKey: key, baseURL: baseURL, useCache: true}
	if cc, err := cache.New(); err == nil {
		c.cache = cc
	}
	return c, nil
}

// SetUseCache toggles the on-disk response cache. Cache is enabled by default.
func (c *Client) SetUseCache(b bool) { c.useCache = b }

// Get fetches path with a background context.
func (c *Client) Get(path string, result interface{}) error {
	return c.GetContext(context.Background(), path, result)
}

// GetContext fetches path, decoding the body into result. Cancelling ctx
// aborts the request, which is how Ctrl-C stops a slow command.
func (c *Client) GetContext(ctx context.Context, path string, result interface{}) error {
	body, err := c.fetch(ctx, path)
	if err != nil {
		return err
	}
	return json.Unmarshal(body, result)
}

// GetRaw fetches path with a background context, returning the body untouched.
func (c *Client) GetRaw(path string) (json.RawMessage, error) {
	return c.GetRawContext(context.Background(), path)
}

// GetRawContext returns the body of path untouched.
func (c *Client) GetRawContext(ctx context.Context, path string) (json.RawMessage, error) {
	body, err := c.fetch(ctx, path)
	if err != nil {
		return nil, err
	}
	return json.RawMessage(body), nil
}

func (c *Client) fetch(ctx context.Context, path string) ([]byte, error) {
	url := c.baseURL + path

	var cached *cache.Entry
	if c.useCache && c.cache != nil {
		cached = c.cache.Get(url)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("X-TBA-Auth-Key", c.apiKey)
	req.Header.Set("User-Agent", userAgent)
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
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusNotModified:
		if cached != nil {
			if c.useCache && c.cache != nil {
				_ = c.cache.Touch(url)
			}
			return []byte(cached.Body), nil
		}
		return nil, fmt.Errorf("API returned 304 but no cached response is available")
	case http.StatusOK:
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, err
		}
		if c.useCache && c.cache != nil {
			_ = c.cache.Put(url, resp.Header.Get("ETag"), resp.Header.Get("Last-Modified"), body)
		}
		return body, nil
	default:
		body, _ := io.ReadAll(resp.Body)
		message := truncateErrorBody(string(body))
		switch resp.StatusCode {
		case http.StatusUnauthorized:
			return nil, clierr.Auth("not authenticated for %s (HTTP 401): run 'tba auth login'", c.baseURL)
		case http.StatusNotFound:
			return nil, clierr.NotFound("API error 404: %s", message)
		}
		return nil, fmt.Errorf("API error %d: %s", resp.StatusCode, message)
	}
}
