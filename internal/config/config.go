package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

const (
	ConfigDirName  = ".please"
	ConfigFileName = "config.json"
)

// AgentType represents the type of LLM agent to use
type AgentType string

const (
	AgentClaude AgentType = "claude-code"
	// Future agents can be added here
	// AgentCodex  AgentType = "codex"
	// AgentGoose  AgentType = "goose"
)

// Config represents the application configuration
type Config struct {
	Agent AgentType `json:"agent"`
}

// GetConfigDir returns the path to the config directory
func GetConfigDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}
	return filepath.Join(home, ConfigDirName), nil
}

// GetConfigPath returns the path to the config file
func GetConfigPath() (string, error) {
	configDir, err := GetConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(configDir, ConfigFileName), nil
}

// Load reads the configuration from disk
func Load() (*Config, error) {
	configPath, err := GetConfigPath()
	if err != nil {
		return nil, err
	}

	// If config doesn't exist, return nil (not an error)
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return nil, nil
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	return &cfg, nil
}

// Save writes the configuration to disk
func Save(cfg *Config) error {
	configDir, err := GetConfigDir()
	if err != nil {
		return err
	}

	// Create config directory if it doesn't exist
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	configPath, err := GetConfigPath()
	if err != nil {
		return err
	}

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(configPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// Exists checks if a configuration file exists
func Exists() (bool, error) {
	configPath, err := GetConfigPath()
	if err != nil {
		return false, err
	}

	_, err = os.Stat(configPath)
	if os.IsNotExist(err) {
		return false, nil
	}
	if err != nil {
		return false, err
	}

	return true, nil
}
