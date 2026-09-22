package api

import (
	"context"
	"fmt"
	"io"
	"net/http"

	"github.com/the-blue-alliance/tba-cli/internal/clierr"
	"github.com/the-blue-alliance/tba-cli/internal/config"
)

// ValidateKey checks an API key against a TBA deployment by asking it for
// /status. It exists so that `tba auth login` can refuse a key the API does
// not accept, instead of storing it and failing on the next command.
func ValidateKey(ctx context.Context, baseURL, key string) error {
	if baseURL == "" {
		baseURL = DefaultBaseURL
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, baseURL+"/status", nil)
	if err != nil {
		return err
	}
	req.Header.Set("X-TBA-Auth-Key", key)
	req.Header.Set("User-Agent", userAgent)

	resp, err := (&http.Client{}).Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusOK:
		return nil
	case http.StatusUnauthorized:
		return clierr.Auth("that API key was rejected by %s (HTTP 401). Get one at %s", baseURL, config.APIKeyPage)
	default:
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API error %d: %s", resp.StatusCode, truncateErrorBody(string(body)))
	}
}
