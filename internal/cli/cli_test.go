package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestCli_AfterApply_Logic tests the logic within AfterApply function
// We test the components directly rather than mocking kong.Context
func TestCli_AfterApply_Logic(t *testing.T) {
	// Test config validation logic for non-version commands
	t.Run("config validation logic", func(t *testing.T) {
		// Create temporary config for testing
		tmpDir, err := os.MkdirTemp("", "zgsync_test_*")
		if err != nil {
			t.Fatalf("Failed to create temp dir: %v", err)
		}
		defer func() { _ = os.RemoveAll(tmpDir) }()

		// Create valid config file
		configContent := `subdomain: test
email: test@example.com/token
token: testtoken
default_locale: en_us
default_permission_group_id: 1
`
		configPath := filepath.Join(tmpDir, ".config", "zgsync")
		if err := os.MkdirAll(configPath, 0755); err != nil {
			t.Fatalf("Failed to create config path: %v", err)
		}

		configFile := filepath.Join(configPath, "config.yaml")
		if err := os.WriteFile(configFile, []byte(configContent), 0644); err != nil {
			t.Fatalf("Failed to create config file: %v", err)
		}

		// Test Global config validation
		originalHome := os.Getenv("HOME")
		_ = os.Setenv("HOME", tmpDir)
		defer func() { _ = os.Setenv("HOME", originalHome) }()

		g := &Global{}

		// Test ConfigExists
		if err := g.ConfigExists(); err != nil {
			t.Errorf("ConfigExists() failed with valid config: %v", err)
		}

		// Test LoadConfig
		if err := g.LoadConfig(); err != nil {
			t.Errorf("LoadConfig() failed with valid config: %v", err)
		}
	})

	// Test config not found error
	t.Run("config not found error", func(t *testing.T) {
		g := &Global{
			ConfigPath: "/nonexistent/path/config.yaml",
		}

		// Test ConfigExists should fail
		err := g.ConfigExists()
		if err == nil {
			t.Error("Expected ConfigExists() to fail with non-existent config")
		} else if !strings.Contains(err.Error(), "does not exist") {
			t.Errorf("Expected 'does not exist' error, got: %v", err)
		}
	})
}
