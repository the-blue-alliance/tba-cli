package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/viper"
)

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

func GetAPIKey() (string, error) {
	if key := os.Getenv("TBA_AUTH_KEY"); key != "" {
		return key, nil
	}

	v := viper.New()
	v.SetConfigFile(AuthFile())
	if err := v.ReadInConfig(); err != nil {
		return "", fmt.Errorf("not authenticated. Run 'tba auth login' first")
	}
	key := v.GetString("api_key")
	if key == "" {
		return "", fmt.Errorf("not authenticated. Run 'tba auth login' first")
	}
	return key, nil
}

func SaveAPIKey(key string) error {
	dir := configDir()
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}
	v := viper.New()
	v.Set("api_key", key)
	return v.WriteConfigAs(AuthFile())
}

func RemoveAPIKey() error {
	return os.Remove(AuthFile())
}
