package config_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/google/uuid"
	"github.com/iamoeg/bootdev-capstone/pkg/config"
	"github.com/stretchr/testify/require"
)

// ============================================================================
// Config Struct Tests
// ============================================================================

func TestDefault(t *testing.T) {
	t.Parallel()

	cfg := config.Default()

	require.NotNil(t, cfg)
	require.Equal(t, "data.db", cfg.DatabasePath)
	require.Equal(t, "", cfg.DisplayName)
	require.Equal(t, "", cfg.DefaultOrgID)
}

// ============================================================================
// Validation Tests
// ============================================================================

func TestConfig_Validate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		config  *config.Config
		wantErr error
	}{
		{
			name: "valid config with all fields",
			config: &config.Config{
				DatabasePath: "data.db",
				DisplayName:  "Ahmed Hassan",
				DefaultOrgID: uuid.New().String(),
			},
			wantErr: nil,
		},
		{
			name: "valid config with minimal fields",
			config: &config.Config{
				DatabasePath: "data.db",
			},
			wantErr: nil,
		},
		{
			name: "valid config with absolute path",
			config: &config.Config{
				DatabasePath: "/tmp/test.db",
			},
			wantErr: nil,
		},
		{
			name: "empty database path",
			config: &config.Config{
				DatabasePath: "",
			},
			wantErr: config.ErrInvalidDatabasePath,
		},
		{
			name: "whitespace-only database path",
			config: &config.Config{
				DatabasePath: "   ",
			},
			wantErr: config.ErrInvalidDatabasePath,
		},
		{
			name: "invalid default_org_id UUID",
			config: &config.Config{
				DatabasePath: "data.db",
				DefaultOrgID: "not-a-uuid",
			},
			wantErr: config.ErrInvalidConfig,
		},
		{
			name: "empty default_org_id is valid",
			config: &config.Config{
				DatabasePath: "data.db",
				DefaultOrgID: "",
			},
			wantErr: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := tt.config.Validate()

			if tt.wantErr == nil {
				require.NoError(t, err)
			} else {
				require.ErrorIs(t, err, tt.wantErr)
			}
		})
	}
}

// ============================================================================
// Path Resolution Tests
// ============================================================================

func TestConfig_ResolveDatabasePath(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		dbPath      string
		checkResult func(t *testing.T, result string)
	}{
		{
			name:   "empty path uses default",
			dbPath: "",
			checkResult: func(t *testing.T, result string) {
				require.Contains(t, result, "finmgmt")
				require.Contains(t, result, "data.db")
			},
		},
		{
			name:   "absolute path unchanged",
			dbPath: "/tmp/test.db",
			checkResult: func(t *testing.T, result string) {
				require.Equal(t, "/tmp/test.db", result)
			},
		},
		{
			name:   "relative path resolved",
			dbPath: "custom.db",
			checkResult: func(t *testing.T, result string) {
				require.True(t, filepath.IsAbs(result))
				require.Contains(t, result, "finmgmt")
				require.Contains(t, result, "custom.db")
			},
		},
		{
			name:   "relative path with subdirs",
			dbPath: "backups/archive.db",
			checkResult: func(t *testing.T, result string) {
				require.True(t, filepath.IsAbs(result))
				require.Contains(t, result, "finmgmt")
				require.Contains(t, result, "backups")
				require.Contains(t, result, "archive.db")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			cfg := &config.Config{DatabasePath: tt.dbPath}
			result, err := cfg.ResolveDatabasePath()

			require.NoError(t, err)
			tt.checkResult(t, result)
		})
	}
}

// ============================================================================
// Save/Load Tests
// ============================================================================

func TestConfig_SaveAndLoad(t *testing.T) {
	t.Parallel()

	// Create temp directory for test
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	// Create test config
	original := &config.Config{
		DatabasePath: "test.db",
		DisplayName:  "Test User",
		DefaultOrgID: uuid.New().String(),
	}

	// Save
	err := original.Save(configPath)
	require.NoError(t, err)

	// Verify file exists
	_, err = os.Stat(configPath)
	require.NoError(t, err)

	// Load
	loaded, err := config.Load(configPath)
	require.NoError(t, err)

	// Verify values match
	require.Equal(t, original.DatabasePath, loaded.DatabasePath)
	require.Equal(t, original.DisplayName, loaded.DisplayName)
	require.Equal(t, original.DefaultOrgID, loaded.DefaultOrgID)
}

func TestConfig_SaveWithInvalidConfig(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	// Invalid config (empty database path)
	cfg := &config.Config{
		DatabasePath: "",
	}

	// Save should fail validation
	err := cfg.Save(configPath)
	require.ErrorIs(t, err, config.ErrInvalidDatabasePath)

	// File should not have been created
	_, err = os.Stat(configPath)
	require.True(t, os.IsNotExist(err))
}

func TestConfig_SaveCreatesDirectories(t *testing.T) {
	t.Parallel()

	// Create temp directory
	tmpDir := t.TempDir()

	// Path with nested directories
	configPath := filepath.Join(tmpDir, "nested", "dirs", "config.yaml")

	cfg := config.Default()

	// Save should create directories
	err := cfg.Save(configPath)
	require.NoError(t, err)

	// Verify file exists
	_, err = os.Stat(configPath)
	require.NoError(t, err)
}

func TestLoad_ConfigNotFound(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	nonExistentPath := filepath.Join(tmpDir, "does-not-exist.yaml")

	_, err := config.Load(nonExistentPath)
	require.Error(t, err)
	require.ErrorIs(t, err, config.ErrConfigNotFound)
}

func TestLoad_ConfigNotFoundInXDGSearch(t *testing.T) {
	// Override XDG paths to non-existent directories
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	// Load with empty path (triggers XDG search)
	_, err := config.Load("")
	require.Error(t, err)
	require.ErrorIs(t, err, config.ErrConfigNotFound)
}

func TestLoad_InvalidYAML(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	// Write invalid YAML
	err := os.WriteFile(configPath, []byte("invalid: yaml: content:"), 0o644)
	require.NoError(t, err)

	_, err = config.Load(configPath)
	require.Error(t, err)
}

// ============================================================================
// LoadOrCreate Tests
// ============================================================================

func TestLoadOrCreate_LoadsExisting(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	// Create existing config
	original := &config.Config{
		DatabasePath: "existing.db",
		DisplayName:  "Existing User",
	}
	err := original.Save(configPath)
	require.NoError(t, err)

	// LoadOrCreate should load existing
	cfg, err := config.LoadOrCreate(configPath)
	require.NoError(t, err)
	require.Equal(t, original.DatabasePath, cfg.DatabasePath)
	require.Equal(t, original.DisplayName, cfg.DisplayName)
}

func TestLoadOrCreate_CreatesDefault(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	// LoadOrCreate should create default when file doesn't exist
	cfg, err := config.LoadOrCreate(configPath)
	require.NoError(t, err)

	// Should have default values
	require.Equal(t, "data.db", cfg.DatabasePath)
	require.Equal(t, "", cfg.DisplayName)

	// File should now exist
	_, err = os.Stat(configPath)
	require.NoError(t, err)

	// Should be able to load it again
	cfg2, err := config.Load(configPath)
	require.NoError(t, err)
	require.Equal(t, cfg.DatabasePath, cfg2.DatabasePath)
}

// ============================================================================
// Environment Variable Tests
// ============================================================================

func TestLoad_EnvironmentVariableOverride(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	// Create config file
	cfg := &config.Config{
		DatabasePath: "file.db",
		DisplayName:  "File User",
	}
	err := cfg.Save(configPath)
	require.NoError(t, err)

	// Set environment variable override
	t.Setenv("FINMGMT_DATABASE_PATH", "/env/override.db")
	t.Setenv("FINMGMT_DISPLAY_NAME", "Env User")

	// Load should use environment variable
	loaded, err := config.Load(configPath)
	require.NoError(t, err)

	require.Equal(t, "/env/override.db", loaded.DatabasePath)
	require.Equal(t, "Env User", loaded.DisplayName)
}

// ============================================================================
// XDG Helper Tests
// ============================================================================

func TestConfigDir(t *testing.T) {
	t.Parallel()

	dir := config.ConfigDir()

	// Should end with "finmgmt"
	require.Contains(t, dir, "finmgmt")

	// Should be an absolute path
	require.True(t, filepath.IsAbs(dir))
}

func TestDataDir(t *testing.T) {
	t.Parallel()

	dir := config.DataDir()

	// Should end with "finmgmt"
	require.Contains(t, dir, "finmgmt")

	// Should be an absolute path
	require.True(t, filepath.IsAbs(dir))
}

// ============================================================================
// YAML Format Tests
// ============================================================================

func TestConfig_SavedYAMLFormat(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	cfg := &config.Config{
		DatabasePath: "test.db",
		DisplayName:  "Test User",
		DefaultOrgID: "550e8400-e29b-41d4-a716-446655440000",
	}

	err := cfg.Save(configPath)
	require.NoError(t, err)

	// Read raw file content
	content, err := os.ReadFile(configPath)
	require.NoError(t, err)

	contentStr := string(content)

	// Verify YAML format
	require.Contains(t, contentStr, "database_path: test.db")
	require.Contains(t, contentStr, "display_name: Test User")
	require.Contains(t, contentStr, "default_org_id: 550e8400-e29b-41d4-a716-446655440000")
}
