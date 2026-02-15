package config

import (
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Config holds the application configuration
type Config struct {
	// UI settings
	UI UIConfig `yaml:"ui"`

	// Default namespace (empty = all namespaces)
	DefaultNamespace string `yaml:"defaultNamespace"`

	// Refresh interval in seconds
	RefreshInterval int `yaml:"refreshInterval"`

	// Database path
	DatabasePath string `yaml:"databasePath"`
}

// UIConfig holds UI-specific settings
type UIConfig struct {
	// Theme name
	Theme string `yaml:"theme"`

	// Show all namespaces by default
	ShowAllNamespaces bool `yaml:"showAllNamespaces"`

	// Table settings
	Table TableConfig `yaml:"table"`
}

// TableConfig holds table display settings
type TableConfig struct {
	// Show line numbers
	ShowLineNumbers bool `yaml:"showLineNumbers"`

	// Compact mode (less padding)
	CompactMode bool `yaml:"compactMode"`
}

// DefaultConfig returns the default configuration
func DefaultConfig() *Config {
	return &Config{
		UI: UIConfig{
			Theme:             "default",
			ShowAllNamespaces: true,
			Table: TableConfig{
				ShowLineNumbers: false,
				CompactMode:     false,
			},
		},
		DefaultNamespace: "",
		RefreshInterval:  30,
		DatabasePath:     getDefaultDatabasePath(),
	}
}

// Load loads configuration from the default path
func Load() (*Config, error) {
	configPath := getConfigPath()

	// If config doesn't exist, return defaults
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return DefaultConfig(), nil
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		return DefaultConfig(), nil
	}

	config := DefaultConfig()
	if err := yaml.Unmarshal(data, config); err != nil {
		return DefaultConfig(), err
	}

	return config, nil
}

// Save saves configuration to the default path
func (c *Config) Save() error {
	configPath := getConfigPath()

	// Ensure directory exists
	dir := filepath.Dir(configPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	data, err := yaml.Marshal(c)
	if err != nil {
		return err
	}

	return os.WriteFile(configPath, data, 0644)
}

func getConfigPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ".kview/config.yaml"
	}
	return filepath.Join(home, ".kview", "config.yaml")
}

func getDefaultDatabasePath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ".kview/data.db"
	}
	return filepath.Join(home, ".kview", "data.db")
}

// GetDataDir returns the kview data directory
func GetDataDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ".kview"
	}
	return filepath.Join(home, ".kview")
}

// EnsureDataDir creates the data directory if it doesn't exist
func EnsureDataDir() error {
	dir := GetDataDir()
	return os.MkdirAll(dir, 0755)
}
