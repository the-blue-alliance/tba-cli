package api

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/the-blue-alliance/tba-cli/internal/config"
)

const baseURL = "https://www.thebluealliance.com/api/v3"

type Client struct {
	http   *http.Client
	apiKey string
}

func NewClient() (*Client, error) {
	key, err := config.GetAPIKey()
	if err != nil {
		return nil, err
	}
	return &Client{http: &http.Client{}, apiKey: key}, nil
}

func (c *Client) Get(path string, result interface{}) error {
	url := baseURL + path
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("X-TBA-Auth-Key", c.apiKey)

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
	url := baseURL + path
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("X-TBA-Auth-Key", c.apiKey)

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
