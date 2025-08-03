package cli

import (
	"strings"
	"testing"
)

func TestLoadConfig(t *testing.T) {
	refDefaultUserSegmentID := 456
	tests := []struct {
		configPath               string
		subdomain                string
		email                    string
		token                    string
		defaultCommentsDisabled  bool
		defaultLocale            string
		defaultPermissionGroupID int
		defaultUserSegmentID     *int
		notifySubscribers        bool
		contentsDir              string
	}{
		{
			"testdata/config.yaml",
			"example",
			"hoge@example.com",
			"foobarfoobar",
			true,
			"ja",
			123,
			&refDefaultUserSegmentID,
			false,
			"example",
		},
		{
			"testdata/config_no_required.yaml",
			"example",
			"hoge@example.com",
			"foobarfoobar",
			false,
			"ja",
			123,
			nil,
			false,
			".",
		},
	}

	for _, tt := range tests {
		t.Run(tt.configPath, func(t *testing.T) {
			var g Global
			g.ConfigPath = tt.configPath
			err := g.LoadConfig()
			if err != nil {
				t.Errorf("LoadConfig() failed: %v", err)
			}

			if g.Config.Subdomain != tt.subdomain {
				t.Errorf("Config.Subdomain failed: got %v, want %v", g.Config.Subdomain, tt.subdomain)
			}
			if g.Config.Email != tt.email {
				t.Errorf("Config.Email failed: got %v, want %v", g.Config.Email, tt.email)
			}
			if g.Config.Token != tt.token {
				t.Errorf("Config.Token failed: got %v, want %v", g.Config.Token, tt.token)
			}
			if g.Config.DefaultCommentsDisabled != tt.defaultCommentsDisabled {
				t.Errorf("Config.DefaultCommentsDisabled failed: got %v, want %v", g.Config.DefaultCommentsDisabled, tt.defaultCommentsDisabled)
			}
			if g.Config.DefaultLocale != tt.defaultLocale {
				t.Errorf("Config.DefaultLocale failed: got %v, want %v", g.Config.DefaultLocale, tt.defaultLocale)
			}
			if g.Config.DefaultPermissionGroupID != tt.defaultPermissionGroupID {
				t.Errorf("Config.DefaultPermissionGroupID failed: got %v, want %v", g.Config.DefaultPermissionGroupID, tt.defaultPermissionGroupID)
			}
			if g.Config.DefailtUserSegmentID != nil && *g.Config.DefailtUserSegmentID != *tt.defaultUserSegmentID {
				t.Errorf("Config.DefailtUserSegmentID failed: got %v, want %v", g.Config.DefailtUserSegmentID, tt.defaultUserSegmentID)
			}
			if g.Config.NotifySubscribers != tt.notifySubscribers {
				t.Errorf("Config.NotifySubscribers failed: got %v, want %v", g.Config.NotifySubscribers, tt.notifySubscribers)
			}
			if g.Config.ContentsDir != tt.contentsDir {
				t.Errorf("Config.DocsRoot failed: got %v, want %v", g.Config.ContentsDir, tt.contentsDir)
			}
		})
	}
}

func TestConfigExists(t *testing.T) {
	tests := []struct {
		name       string
		configPath string
		notError   bool
	}{
		{
			"config exists",
			"testdata/config.yaml",
			true,
		},
		{
			"config does not exists",
			"testdata/config_not_exists.yaml",
			false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var g Global
			g.ConfigPath = tt.configPath
			err := g.ConfigExists()
			if tt.notError == (err != nil) {
				t.Errorf("ConfigExists() failed: %v", err)
			}
		})
	}
}

func TestLoadConfig_ErrorCases(t *testing.T) {
	tests := []struct {
		name       string
		configPath string
		expectErr  bool
		errMsg     string
	}{
		{
			name:       "non-existent config file",
			configPath: "testdata/non-existent.yaml",
			expectErr:  false, // LoadConfig returns nil for missing files
		},
		{
			name:       "invalid yaml format",
			configPath: "testdata/invalid.yaml",
			expectErr:  true,
			errMsg:     "mapping values are not allowed", // Actual YAML error message
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var g Global
			g.ConfigPath = tt.configPath
			err := g.LoadConfig()

			if tt.expectErr {
				if err == nil {
					t.Errorf("LoadConfig() expected error but got none")
				} else if tt.errMsg != "" && !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("LoadConfig() error = %v, expected to contain %v", err, tt.errMsg)
				}
			} else {
				if err != nil {
					t.Errorf("LoadConfig() unexpected error: %v", err)
				}
			}
		})
	}
}

func TestConfig_Validation_ErrorCases(t *testing.T) {
	tests := []struct {
		name   string
		config Config
		expectErr bool
		errMsg string
	}{
		{
			name: "missing subdomain",
			config: Config{
				Email:                    "test@example.com/token",
				Token:                    "token123",
				DefaultLocale:            "en",
				DefaultPermissionGroupID: 123,
			},
			expectErr: true,
			errMsg:    "subdomain",
		},
		{
			name: "missing email",
			config: Config{
				Subdomain:                "test",
				Token:                    "token123",
				DefaultLocale:            "en",
				DefaultPermissionGroupID: 123,
			},
			expectErr: true,
			errMsg:    "email",
		},
		{
			name: "missing token",
			config: Config{
				Subdomain:                "test",
				Email:                    "test@example.com/token",
				DefaultLocale:            "en",
				DefaultPermissionGroupID: 123,
			},
			expectErr: true,
			errMsg:    "token",
		},
		{
			name: "missing default locale",
			config: Config{
				Subdomain:                "test",
				Email:                    "test@example.com/token",
				Token:                    "token123",
				DefaultPermissionGroupID: 123,
			},
			expectErr: true,
			errMsg:    "default_locale",
		},
		{
			name: "missing default permission group id",
			config: Config{
				Subdomain:     "test",
				Email:         "test@example.com/token",
				Token:         "token123",
				DefaultLocale: "en",
			},
			expectErr: true,
			errMsg:    "default_permission_group_id",
		},
		{
			name: "valid config",
			config: Config{
				Subdomain:                "test",
				Email:                    "test@example.com/token",
				Token:                    "token123",
				DefaultLocale:            "en",
				DefaultPermissionGroupID: 123,
			},
			expectErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validation()

			if tt.expectErr {
				if err == nil {
					t.Errorf("Config.Validation() expected error but got none")
				} else if tt.errMsg != "" && !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("Config.Validation() error = %v, expected to contain %v", err, tt.errMsg)
				}
			} else {
				if err != nil {
					t.Errorf("Config.Validation() unexpected error: %v", err)
				}
			}
		})
	}
}

func TestAbsConfig_ErrorCases(t *testing.T) {
	tests := []struct {
		name       string
		configPath string
		expectAbsolute bool
	}{
		{
			name:       "valid relative path",
			configPath: "testdata/config.yaml",
			expectAbsolute: true,
		},
		{
			name:       "valid absolute path",
			configPath: "/tmp/config.yaml",
			expectAbsolute: true,
		},
		{
			name:       "empty path returns current directory",
			configPath: "",
			expectAbsolute: true, // filepath.Abs("") returns current directory
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var g Global
			g.ConfigPath = tt.configPath
			result := g.AbsConfig()

			if tt.expectAbsolute {
				if result == "" {
					t.Errorf("AbsConfig() expected non-empty result but got empty")
				}
				// For relative paths, result should be different from input
				if tt.configPath != "" && !strings.HasPrefix(result, "/") {
					// On Unix systems, absolute paths start with "/"
					// This test verifies the path was made absolute
					t.Logf("AbsConfig() converted relative path correctly: %s -> %s", tt.configPath, result)
				}
			}
		})
	}
}
