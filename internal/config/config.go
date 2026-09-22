package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/viper"
	"github.com/the-blue-alliance/tba-cli/internal/clierr"
)

const DefaultBaseURL = "https://www.thebluealliance.com/api/v3"

func configDir() (string, error) {
	if d := os.Getenv("TBA_CONFIG_DIR"); d != "" {
		return d, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("cannot determine your home directory (%w); set TBA_CONFIG_DIR to choose where tba keeps its config", err)
	}
	return filepath.Join(home, ".config", "tba"), nil
}

// AuthFile returns the path of the file that stores API keys.
func AuthFile() (string, error) {
	dir, err := configDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "auth.yaml"), nil
}

// secureAuthFile sets the auth file mode to 0600 if it exists and is broader.
// Best-effort: errors from Stat/Chmod are returned to the caller but the file
// may not exist yet (first login), which is not an error.
func secureAuthFile(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	if info.Mode().Perm() != 0600 {
		return os.Chmod(path, 0600)
	}
	return nil
}

func GetAPIKey(baseURL string) (string, error) {
	if key := os.Getenv("TBA_AUTH_KEY"); key != "" {
		return key, nil
	}

	if baseURL == "" {
		baseURL = DefaultBaseURL
	}

	authFile, err := AuthFile()
	if err != nil {
		return "", err
	}

	v := viper.New()
	v.SetConfigFile(authFile)
	if err := v.ReadInConfig(); err != nil {
		return "", clierr.Auth("not authenticated for %s. Run 'tba auth login' first", baseURL)
	}
	_ = secureAuthFile(authFile) // remediate legacy files written with wider perms; ignore errors

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

	return "", clierr.Auth("not authenticated for %s. Run 'tba auth login --base-url %s' first", baseURL, baseURL)
}

// migrateLegacyKey folds a pre-per-URL "api_key" entry into the "keys" map.
// The legacy entry is blanked rather than deleted because viper has no way to
// unset a key; GetAPIKey treats an empty api_key as absent.
func migrateLegacyKey(v *viper.Viper) {
	legacyKey := v.GetString("api_key")
	if legacyKey == "" {
		return
	}
	keys := v.GetStringMapString("keys")
	if keys == nil {
		keys = make(map[string]string)
	}
	if _, exists := keys[DefaultBaseURL]; !exists {
		keys[DefaultBaseURL] = legacyKey
	}
	v.Set("keys", keys)
	v.Set("api_key", "")
}

func SaveAPIKey(key string, baseURL string) error {
	if baseURL == "" {
		baseURL = DefaultBaseURL
	}

	dir, err := configDir()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}
	authFile := filepath.Join(dir, "auth.yaml")

	v := viper.New()
	v.SetConfigFile(authFile)
	_ = v.ReadInConfig() // load existing keys, ignore error if file doesn't exist

	migrateLegacyKey(v)

	keys := v.GetStringMapString("keys")
	if keys == nil {
		keys = make(map[string]string)
	}
	keys[baseURL] = key
	v.Set("keys", keys)

	if err := v.WriteConfigAs(authFile); err != nil {
		return err
	}
	return os.Chmod(authFile, 0600)
}

func RemoveAPIKey(baseURL string) error {
	if baseURL == "" {
		baseURL = DefaultBaseURL
	}

	authFile, err := AuthFile()
	if err != nil {
		return err
	}

	v := viper.New()
	v.SetConfigFile(authFile)
	if err := v.ReadInConfig(); err != nil {
		return fmt.Errorf("not authenticated")
	}

	migrateLegacyKey(v)

	keys := v.GetStringMapString("keys")
	if _, ok := keys[baseURL]; !ok {
		return fmt.Errorf("not authenticated for %s", baseURL)
	}
	delete(keys, baseURL)
	v.Set("keys", keys)

	if err := v.WriteConfigAs(authFile); err != nil {
		return err
	}
	return os.Chmod(authFile, 0600)
}
