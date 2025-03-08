package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// Config represents the application configuration
type Config struct {
	// Storage settings
	DataDirectory  string `json:"data_directory" yaml:"data_directory"`
	BackupEnabled  bool   `json:"backup_enabled" yaml:"backup_enabled"`
	BackupInterval int    `json:"backup_interval" yaml:"backup_interval"` // Days between backups

	// Session settings
	RecoveryTime         time.Duration `json:"recovery_time" yaml:"recovery_time"`                   // In minutes
	DefaultSessionLength time.Duration `json:"default_session_length" yaml:"default_session_length"` // In minutes

	// UI settings
	EnableMouse       bool   `json:"enable_mouse" yaml:"enable_mouse"`
	ColorTheme        string `json:"color_theme" yaml:"color_theme"` // "light", "dark", "system"
	ShowNotifications bool   `json:"show_notifications" yaml:"show_notifications"`

	// Custom interruption categories
	CustomInterruptionTags []string `json:"custom_interruption_tags" yaml:"custom_interruption_tags"`

	// Security
	EnableEncryption bool   `json:"enable_encryption" yaml:"enable_encryption"`
	EncryptionKey    string `json:"encryption_key,omitempty" yaml:"encryption_key,omitempty"` // Only used if manually set
	PasswordProtect  bool   `json:"password_protect" yaml:"password_protect"`
	PasswordHash     string `json:"password_hash,omitempty" yaml:"password_hash,omitempty"`
}

// DefaultConfig returns the default configuration
func DefaultConfig() *Config {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		homeDir = "."
	}

	return &Config{
		DataDirectory:  filepath.Join(homeDir, ".interruption-tracker"),
		BackupEnabled:  true,
		BackupInterval: 7, // Weekly backups

		RecoveryTime:         10 * time.Minute,
		DefaultSessionLength: 25 * time.Minute, // Pomodoro-style default

		EnableMouse:       true,
		ColorTheme:        "system",
		ShowNotifications: true,

		CustomInterruptionTags: []string{},

		EnableEncryption: false,
		PasswordProtect:  false,
	}
}

// ConfigFileType represents the type of configuration file
type ConfigFileType int

const (
	// ConfigFileTypeJSON indicates a JSON configuration file
	ConfigFileTypeJSON ConfigFileType = iota
	// ConfigFileTypeYAML indicates a YAML configuration file
	ConfigFileTypeYAML
)

// ConfigPath returns the path to the config file
func ConfigPath() (string, error) {
	// Get user's home directory
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("could not determine home directory: %w", err)
	}

	// Define possible config directories in priority order
	configDirs := []string{
		// ~/.interruption-tracker
		filepath.Join(homeDir, ".interruption-tracker"),
	}

	// Add ~/.config/interruption-tracker on Unix-like systems
	sysConfigDir, err := os.UserConfigDir()
	if err == nil {
		configDirs = append(configDirs, filepath.Join(sysConfigDir, "interruption-tracker"))
	}

	// Check each directory for a config file
	for _, dir := range configDirs {
		// Ensure directory exists
		if err := os.MkdirAll(dir, 0755); err != nil {
			continue
		}

		// Check for YAML config first
		yamlPath := filepath.Join(dir, "config.yaml")
		if _, err := os.Stat(yamlPath); err == nil {
			return yamlPath, nil
		}

		// Try with yml extension
		yamlPath = filepath.Join(dir, "config.yml")
		if _, err := os.Stat(yamlPath); err == nil {
			return yamlPath, nil
		}

		// Check for JSON config
		jsonPath := filepath.Join(dir, "config.json")
		if _, err := os.Stat(jsonPath); err == nil {
			return jsonPath, nil
		}
	}

	// If no config file found, use the default location in home directory
	return filepath.Join(homeDir, ".interruption-tracker", "config.json"), nil
}

// LoadConfigFromPath loads the configuration from a specific path
func LoadConfigFromPath(configPath string) (*Config, error) {
	// Check if file exists
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("config file does not exist: %s", configPath)
	}

	// Read the file
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("could not read config file: %w", err)
	}

	// Parse the config based on file extension
	var config Config
	if strings.HasSuffix(configPath, ".yaml") || strings.HasSuffix(configPath, ".yml") {
		if err := yaml.Unmarshal(data, &config); err != nil {
			return nil, fmt.Errorf("could not parse YAML config file: %w", err)
		}
	} else {
		if err := json.Unmarshal(data, &config); err != nil {
			return nil, fmt.Errorf("could not parse JSON config file: %w", err)
		}
	}

	// Ensure config is valid and has all required fields
	if config.DataDirectory == "" {
		homeDir, _ := os.UserHomeDir()
		config.DataDirectory = filepath.Join(homeDir, ".interruption-tracker")
	}

	// Convert recovery time from stored minutes to duration
	if config.RecoveryTime == 0 {
		config.RecoveryTime = 10 * time.Minute
	}

	return &config, nil
}

// LoadConfig loads the configuration from disk
func LoadConfig() (*Config, error) {
	configPath, err := ConfigPath()
	if err != nil {
		return DefaultConfig(), err
	}

	// Check if file exists
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		// Create default config
		config := DefaultConfig()
		if err := SaveConfig(config); err != nil {
			return config, fmt.Errorf("could not save default config: %w", err)
		}
		return config, nil
	}

	// Load from path
	config, err := LoadConfigFromPath(configPath)
	if err != nil {
		return DefaultConfig(), err
	}

	return config, nil
}

// SaveConfigToPath saves the configuration to a specific path
func SaveConfigToPath(config *Config, configPath string) error {
	// Ensure directory exists
	configDir := filepath.Dir(configPath)
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return fmt.Errorf("could not create config directory: %w", err)
	}

	var data []byte
	var err error

	// Marshal the config based on file extension
	if strings.HasSuffix(configPath, ".yaml") || strings.HasSuffix(configPath, ".yml") {
		data, err = yaml.Marshal(config)
		if err != nil {
			return fmt.Errorf("could not marshal YAML config: %w", err)
		}
	} else {
		data, err = json.MarshalIndent(config, "", "  ")
		if err != nil {
			return fmt.Errorf("could not marshal JSON config: %w", err)
		}
	}

	// Write the file
	if err := os.WriteFile(configPath, data, 0644); err != nil {
		return fmt.Errorf("could not write config file: %w", err)
	}

	return nil
}

// SaveConfig saves the configuration to disk
func SaveConfig(config *Config) error {
	configPath, err := ConfigPath()
	if err != nil {
		return err
	}

	return SaveConfigToPath(config, configPath)
}

// GetConfigFileType determines the type of configuration file from its path
func GetConfigFileType(path string) ConfigFileType {
	if strings.HasSuffix(path, ".yaml") || strings.HasSuffix(path, ".yml") {
		return ConfigFileTypeYAML
	}
	return ConfigFileTypeJSON
}

// Schema version for data files
const CurrentSchemaVersion = 1

// SchemaVersion represents the version of the data schema
type SchemaVersion struct {
	Version int `json:"version"`
}

// GetSchemaVersion returns the current schema version
func GetSchemaVersion() int {
	return CurrentSchemaVersion
}
