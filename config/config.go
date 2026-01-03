package config

import (
	"encoding/json"
	"os"
	"path/filepath"
)

const (
	appName    = "quadmax-wifi-connector"
	configFile = "config.json"
)

// Config holds the persistent application configuration
type Config struct {
	SelectedAdapter string `json:"selected_adapter"`
	SelectedNetwork string `json:"selected_network"`
	PollInterval    int    `json:"poll_interval"` // in seconds
}

// DefaultConfig returns a config with default values
func DefaultConfig() *Config {
	return &Config{
		SelectedAdapter: "",
		SelectedNetwork: "",
		PollInterval:    5,
	}
}

// getConfigPath returns the path to the config file
func getConfigPath() (string, error) {
	appData := os.Getenv("APPDATA")
	if appData == "" {
		// Fallback for non-Windows or testing
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		appData = filepath.Join(homeDir, ".config")
	}

	configDir := filepath.Join(appData, appName)
	return filepath.Join(configDir, configFile), nil
}

// Load reads the configuration from disk
func Load() (*Config, error) {
	configPath, err := getConfigPath()
	if err != nil {
		return DefaultConfig(), err
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			return DefaultConfig(), nil
		}
		return DefaultConfig(), err
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return DefaultConfig(), err
	}

	// Ensure poll interval has a valid value
	if cfg.PollInterval <= 0 {
		cfg.PollInterval = 5
	}

	return &cfg, nil
}

// Save writes the configuration to disk
func (c *Config) Save() error {
	configPath, err := getConfigPath()
	if err != nil {
		return err
	}

	// Ensure directory exists
	configDir := filepath.Dir(configPath)
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(configPath, data, 0644)
}
