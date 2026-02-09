package api

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/the-blue-alliance/tba-cli/internal/config"
)

const DefaultBaseURL = "https://www.thebluealliance.com/api/v3"
const userAgent = "tba-cli"

type Client struct {
	http    *http.Client
	apiKey  string
	baseURL string
}

func NewClient(baseURL string) (*Client, error) {
	if baseURL == "" {
		baseURL = DefaultBaseURL
	}
	key, err := config.GetAPIKey(baseURL)
	if err != nil {
		return nil, err
	}
	return &Client{http: &http.Client{}, apiKey: key, baseURL: baseURL}, nil
}

func (c *Client) Get(path string, result interface{}) error {
	url := c.baseURL + path
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("X-TBA-Auth-Key", c.apiKey)
	req.Header.Set("User-Agent", userAgent)

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API error %d: %s", resp.StatusCode, string(body))
	}

	return json.NewDecoder(resp.Body).Decode(result)
}

func (c *Client) GetRaw(path string) (json.RawMessage, error) {
	url := c.baseURL + path
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("X-TBA-Auth-Key", c.apiKey)
	req.Header.Set("User-Agent", userAgent)

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API error %d: %s", resp.StatusCode, string(body))
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	return json.RawMessage(data), nil
}
