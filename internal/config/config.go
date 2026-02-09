package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/viper"
)

const DefaultBaseURL = "https://www.thebluealliance.com/api/v3"

func configDir() string {
	if d := os.Getenv("TBA_CONFIG_DIR"); d != "" {
		return d
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "tba")
}

func AuthFile() string {
	return filepath.Join(configDir(), "auth.yaml")
}

func GetAPIKey(baseURL string) (string, error) {
	if key := os.Getenv("TBA_AUTH_KEY"); key != "" {
		return key, nil
	}

	if baseURL == "" {
		baseURL = DefaultBaseURL
	}

	v := viper.New()
	v.SetConfigFile(AuthFile())
	if err := v.ReadInConfig(); err != nil {
		return "", fmt.Errorf("not authenticated for %s. Run 'tba auth login' first", baseURL)
	}

	// Try new per-URL format first
	keys := v.GetStringMapString("keys")
	if key, ok := keys[baseURL]; ok && key != "" {
		return key, nil
	}

	// Fall back to legacy single-key format (only for the default URL)
	if baseURL == DefaultBaseURL {
		if key := v.GetString("api_key"); key != "" {
			return key, nil
		}
	}

	return "", fmt.Errorf("not authenticated for %s. Run 'tba auth login --base-url %s' first", baseURL, baseURL)
}

func SaveAPIKey(key string, baseURL string) error {
	if baseURL == "" {
		baseURL = DefaultBaseURL
	}

	dir := configDir()
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}

	v := viper.New()
	v.SetConfigFile(AuthFile())
	_ = v.ReadInConfig() // load existing keys, ignore error if file doesn't exist

	// Migrate legacy api_key if present
	if legacyKey := v.GetString("api_key"); legacyKey != "" {
		keys := v.GetStringMapString("keys")
		if keys == nil {
			keys = make(map[string]string)
		}
		if _, exists := keys[DefaultBaseURL]; !exists {
			keys[DefaultBaseURL] = legacyKey
		}
		v.Set("keys", keys)
		v.Set("api_key", nil)
	}

	keys := v.GetStringMapString("keys")
	if keys == nil {
		keys = make(map[string]string)
	}
	keys[baseURL] = key
	v.Set("keys", keys)

	return v.WriteConfigAs(AuthFile())
}

func RemoveAPIKey(baseURL string) error {
	if baseURL == "" {
		baseURL = DefaultBaseURL
	}

	v := viper.New()
	v.SetConfigFile(AuthFile())
	if err := v.ReadInConfig(); err != nil {
		return fmt.Errorf("not authenticated")
	}

	// Migrate legacy api_key if present
	if legacyKey := v.GetString("api_key"); legacyKey != "" {
		keys := v.GetStringMapString("keys")
		if keys == nil {
			keys = make(map[string]string)
		}
		if _, exists := keys[DefaultBaseURL]; !exists {
			keys[DefaultBaseURL] = legacyKey
		}
		v.Set("keys", keys)
		v.Set("api_key", nil)
	}

	keys := v.GetStringMapString("keys")
	if _, ok := keys[baseURL]; !ok {
		return fmt.Errorf("not authenticated for %s", baseURL)
	}
	delete(keys, baseURL)
	v.Set("keys", keys)

	return v.WriteConfigAs(AuthFile())
}
