// Command refresh re-vendors api_v3.json from the server repository's main
// branch and rewrites UPSTREAM to match. Run it from internal/apispec, which
// is what `go generate ./internal/apispec` does.
package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

const (
	repo     = "the-blue-alliance/the-blue-alliance"
	specPath = "src/backend/web/static/swagger/api_v3.json"
	mainURL  = "https://raw.githubusercontent.com/" + repo + "/main/" + specPath
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "refresh:", err)
		os.Exit(1)
	}
}

func run() error {
	client := &http.Client{Timeout: 30 * time.Second}

	// Pin main to a commit first, then fetch the file at that commit, so the
	// bytes and the recorded commit cannot disagree if main moves in between.
	var commits []struct {
		SHA string `json:"sha"`
	}
	b, err := get(client, "https://api.github.com/repos/"+repo+"/commits?sha=main&per_page=1&path="+specPath)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(b, &commits); err != nil || len(commits) == 0 {
		return fmt.Errorf("reading the latest commit for %s: %v", specPath, err)
	}
	sha := commits[0].SHA

	spec, err := get(client, "https://raw.githubusercontent.com/"+repo+"/"+sha+"/"+specPath)
	if err != nil {
		return err
	}
	var doc struct {
		Info struct {
			Version string `json:"version"`
		} `json:"info"`
	}
	if err := json.Unmarshal(spec, &doc); err != nil {
		return fmt.Errorf("the fetched spec is not JSON: %w", err)
	}

	sum := sha256.Sum256(spec)
	record := fmt.Sprintf(`# api_v3.json is vendored unmodified from The Blue Alliance server repository.
# Do not edit it by hand; refresh it with: go generate ./internal/apispec
source: %s
commit: %s
version: %s
sha256: %s
`, mainURL, sha, doc.Info.Version, hex.EncodeToString(sum[:]))

	if err := os.WriteFile("api_v3.json", spec, 0o644); err != nil {
		return err
	}
	if err := os.WriteFile("UPSTREAM", []byte(record), 0o644); err != nil {
		return err
	}
	fmt.Printf("api_v3.json %s from %s\n", doc.Info.Version, sha)
	return nil
}

func get(client *http.Client, url string) ([]byte, error) {
	resp, err := client.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GET %s: %s", url, resp.Status)
	}
	return io.ReadAll(resp.Body)
}
