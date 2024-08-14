package cli

import "testing"

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
			&refDefaultUserSegmentID,
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
