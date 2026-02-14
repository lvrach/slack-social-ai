package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/lvrach/slack-social-ai/internal/schedule"
)

// Config holds the application configuration.
type Config struct {
	Schedule schedule.Schedule `json:"schedule"`
}

// configDir returns the config directory path.
// Exported as a var for testing.
var configDir = defaultConfigDir

func defaultConfigDir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "slack-social-ai")
}

func configPath() string {
	return filepath.Join(configDir(), "config.json")
}

// Exists returns true if a config file has been saved.
func Exists() bool {
	_, err := os.Stat(configPath())
	return err == nil
}

// Load reads the config file. Returns default config if file doesn't exist.
func Load() (Config, error) {
	data, err := os.ReadFile(configPath())
	if err != nil {
		if os.IsNotExist(err) {
			return Config{Schedule: schedule.DefaultSchedule()}, nil
		}
		return Config{}, err
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return Config{}, fmt.Errorf("parse config: %w", err)
	}

	return cfg, nil
}

// Save writes the config to disk.
func Save(cfg Config) error {
	dir := configDir()
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(configPath(), data, 0o600)
}
