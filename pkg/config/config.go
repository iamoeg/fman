// Package config provides configuration management for the fman application.
//
// It handles loading, saving, and validating application configuration using
// the XDG Base Directory specification and supports environment variable overrides.
package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/adrg/xdg"
	"github.com/google/uuid"
	"github.com/spf13/viper"
	"gopkg.in/yaml.v3"
)

// ============================================================================
// Errors
// ============================================================================

var (
	// ErrConfigNotFound indicates the config file could not be found in any
	// search path.
	ErrConfigNotFound = errors.New("config: configuration file not found")

	// ErrInvalidConfig indicates the config contains invalid values.
	ErrInvalidConfig = errors.New("config: invalid configuration")

	// ErrInvalidDatabasePath indicates the database path is empty or invalid.
	ErrInvalidDatabasePath = errors.New("config: invalid database path")
)

// ============================================================================
// Config Structure
// ============================================================================

// Config represents the application configuration.
//
// Configuration priority (highest to lowest):
//  1. Environment variables (FINMGMT_*)
//  2. Config file values
//  3. Default values
type Config struct {
	// DatabasePath is the path to the SQLite database file.
	// Can be absolute or relative (resolved against XDG data directory).
	// Default: "data.db"
	DatabasePath string `mapstructure:"database_path" yaml:"database_path"`

	// DisplayName is the user's display name shown in the TUI.
	// Optional.
	DisplayName string `mapstructure:"display_name" yaml:"display_name"`

	// DefaultOrgID is the UUID of the default organization to use.
	// Optional - if set, must be a valid UUID.
	DefaultOrgID string `mapstructure:"default_org_id" yaml:"default_org_id"`
}

// Default returns a Config with default values.
func Default() *Config {
	return &Config{
		DatabasePath: "data.db", // Relative, will resolve to XDG data dir
		DisplayName:  "",
		DefaultOrgID: "",
	}
}

// ============================================================================
// Loading Configuration
// ============================================================================

// Load reads configuration from file using Viper.
//
// If configPath is empty, searches for config.yaml in XDG config directories.
// Environment variables with FINMGMT_ prefix override config file values.
//
// Priority (highest to lowest):
//  1. Environment variables (FINMGMT_DATABASE_PATH, etc.)
//  2. Config file at configPath (if provided)
//  3. Config file in XDG directories (if configPath empty)
//
// Returns ErrConfigNotFound if the config file doesn't exist.
func Load(configPath string) (*Config, error) {
	// Create a new Viper instance (NOT using global)
	v := viper.New()

	// Determine which config file to use
	var configFileToUse string
	if configPath != "" {
		// Explicit path provided - use it directly
		configFileToUse = configPath
	} else {
		// Reload XDG paths from the current environment before searching.
		// The xdg package caches paths at init time, so this ensures changes
		// to XDG_CONFIG_HOME (e.g. in tests) are respected.
		xdg.Reload()

		// Search for config file in XDG paths
		// (searches XDG_CONFIG_HOME and XDG_CONFIG_DIRS)
		foundPath, err := xdg.SearchConfigFile("fman/config.yaml")
		if err != nil {
			// Not found - return our custom error
			return nil, fmt.Errorf("%w: searched XDG config directories", ErrConfigNotFound)
		}
		configFileToUse = foundPath
	}

	v.SetConfigFile(configFileToUse)

	// Environment variable overrides
	v.SetEnvPrefix("FINMGMT")
	v.AutomaticEnv()
	if err := v.BindEnv("database_path", "FINMGMT_DATABASE_PATH"); err != nil {
		return nil, fmt.Errorf("failed to bind env var: %w", err)
	}
	if err := v.BindEnv("display_name", "FINMGMT_DISPLAY_NAME"); err != nil {
		return nil, fmt.Errorf("failed to bind env var: %w", err)
	}
	if err := v.BindEnv("default_org_id", "FINMGMT_DEFAULT_ORG_ID"); err != nil {
		return nil, fmt.Errorf("failed to bind env var: %w", err)
	}

	// Read the config file
	if err := v.ReadInConfig(); err != nil {
		// return nil, fmt.Errorf("failed to read config from %s: %w", configFileToUse, err)
		// Check for Viper's "not found" error (when searching paths)
		var viperNotFound viper.ConfigFileNotFoundError
		if errors.As(err, &viperNotFound) {
			return nil, fmt.Errorf("%w: %s", ErrConfigNotFound, configFileToUse)
		}
		// Check for OS-level "not found" error (when using explicit path)
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("%w: %s", ErrConfigNotFound, configFileToUse)
		}
		// Other errors (permission denied, invalid YAML, etc.)
		return nil, fmt.Errorf("failed to read config from %s: %w", configFileToUse, err)
	}

	// Unmarshal into our Config struct
	cfg := &Config{}
	if err := v.Unmarshal(cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	return cfg, nil
}

// LoadOrCreate loads existing config or creates a default config if not found.
//
// If configPath is empty, uses XDG config directory.
// If the config file doesn't exist, creates it with default values.
//
// This is the recommended way to initialize configuration on first run.
func LoadOrCreate(configPath string) (*Config, error) {
	// Try to load existing config
	cfg, err := Load(configPath)
	if err == nil {
		return cfg, nil
	}

	// If not found, create default
	if errors.Is(err, ErrConfigNotFound) {
		cfg = Default()

		// Determine where to save
		savePath := configPath
		if savePath == "" {
			// Use xdg.ConfigFile to get proper path (creates dirs automatically)
			savePath, err = xdg.ConfigFile("fman/config.yaml")
			if err != nil {
				return nil, fmt.Errorf("failed to determine config file path: %w", err)
			}
		}

		// Save the default config
		if saveErr := cfg.Save(savePath); saveErr != nil {
			return nil, fmt.Errorf("failed to save default config: %w", saveErr)
		}

		return cfg, nil
	}

	// Other error (not just "not found")
	return nil, err
}

// ============================================================================
// Saving Configuration
// ============================================================================

// Save writes the config to a file in YAML format.
//
// If configPath is empty, uses XDG config directory (~/.config/fman/).
// Creates parent directories automatically if they don't exist.
//
// The config is validated before saving. Returns an error if validation fails.
func (c *Config) Save(configPath string) error {
	// Validate before saving
	if err := c.Validate(); err != nil {
		return fmt.Errorf("cannot save invalid config: %w", err)
	}

	// If no path specified, use XDG location (creates dirs automatically)
	if configPath == "" {
		var err error
		configPath, err = xdg.ConfigFile("fman/config.yaml")
		if err != nil {
			return fmt.Errorf("failed to determine config file path: %w", err)
		}
	} else {
		// Explicit path - ensure parent directory exists
		configDir := filepath.Dir(configPath)
		if err := ensureDir(configDir); err != nil {
			return err
		}
	}

	// Ensure the data directory exists (since config references it)
	if err := ensureDataDir(); err != nil {
		return err
	}

	// Marshal config to YAML
	data, err := yaml.Marshal(c)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	// Write to file
	if err := os.WriteFile(configPath, data, 0o644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// ============================================================================
// Validation
// ============================================================================

// Validate checks if the config contains valid values.
//
// Validation rules:
//   - database_path must not be empty
//   - default_org_id must be a valid UUID (if provided)
func (c *Config) Validate() error {
	// Database path must not be empty
	if strings.TrimSpace(c.DatabasePath) == "" {
		return fmt.Errorf("%w: database_path cannot be empty", ErrInvalidDatabasePath)
	}

	// Default org ID must be valid UUID if provided
	if c.DefaultOrgID != "" {
		if _, err := uuid.Parse(c.DefaultOrgID); err != nil {
			return fmt.Errorf("%w: invalid default_org_id UUID: %s", ErrInvalidConfig, c.DefaultOrgID)
		}
	}

	// Display name is optional, no validation needed

	return nil
}

// ============================================================================
// Path Resolution
// ============================================================================

// ResolveDatabasePath returns the absolute database file path.
//
// Path resolution rules:
//   - Empty path: uses default (XDG data dir + "data.db")
//   - Absolute path: uses as-is
//   - Relative path: resolves against XDG data directory
//
// Creates parent directories automatically if they don't exist.
func (c *Config) ResolveDatabasePath() (string, error) {
	dbPath := c.DatabasePath

	// If empty, use default
	if dbPath == "" {
		return xdg.DataFile("fman/data.db")
	}

	// If absolute, use as-is
	if filepath.IsAbs(dbPath) {
		return dbPath, nil
	}

	// Relative path - resolve against XDG data directory
	// (xdg.DataFile creates parent directories automatically)
	return xdg.DataFile(filepath.Join("fman", dbPath))
}

// ============================================================================
// Helper Functions
// ============================================================================

// ensureDir creates a directory if it doesn't exist.
func ensureDir(dirPath string) error {
	if _, err := os.Stat(dirPath); err == nil {
		return nil // Already exists
	}

	if err := os.MkdirAll(dirPath, 0o750); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", dirPath, err)
	}

	return nil
}

// ensureDataDir creates the data directory if it doesn't exist.
func ensureDataDir() error {
	// Use xdg.DataFile to ensure the directory exists
	// We create a dummy path just to trigger directory creation
	_, err := xdg.DataFile("fman/.keep")
	if err != nil {
		return fmt.Errorf("failed to ensure data directory: %w", err)
	}
	return nil
}

// ============================================================================
// XDG Path Helpers
// ============================================================================
// These functions are provided for informational and debugging purposes.
// The actual config loading uses xdg.SearchConfigFile() and xdg.ConfigFile().

// ConfigDir returns the XDG config directory for the application.
// Typically: ~/.config/fman
func ConfigDir() string {
	return filepath.Join(xdg.ConfigHome, "fman")
}

// DataDir returns the XDG data directory for the application.
// Typically: ~/.local/share/fman
func DataDir() string {
	return filepath.Join(xdg.DataHome, "fman")
}
