package api

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/the-blue-alliance/tba-cli/internal/cache"
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

func (c *Client) Get(path string, result interface{}) error {
	body, err := c.fetch(path)
	if err != nil {
		return err
	}
	return json.Unmarshal(body, result)
}

func (c *Client) GetRaw(path string) (json.RawMessage, error) {
	body, err := c.fetch(path)
	if err != nil {
		return nil, err
	}
	return json.RawMessage(body), nil
}

func (c *Client) fetch(path string) ([]byte, error) {
	url := c.baseURL + path

	var cached *cache.Entry
	if c.useCache && c.cache != nil {
		cached = c.cache.Get(url)
	}

	req, err := http.NewRequest("GET", url, nil)
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
		return nil, fmt.Errorf("API error %d: %s", resp.StatusCode, string(body))
	}
}
