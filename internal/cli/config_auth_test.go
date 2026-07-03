package cli

import (
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestConfig_Validation_AuthTypes(t *testing.T) {
	base := Config{
		Subdomain:                "test",
		DefaultLocale:            "en",
		DefaultPermissionGroupID: 123,
	}

	tests := []struct {
		name      string
		mutate    func(*Config)
		expectErr string
	}{
		{
			name: "oauth requires client id",
			mutate: func(c *Config) {
				c.AuthType = AuthTypeOAuth
			},
			expectErr: "oauth_client_id",
		},
		{
			name: "oauth with client id is valid",
			mutate: func(c *Config) {
				c.AuthType = AuthTypeOAuth
				c.OAuthClientID = "myclient"
			},
		},
		{
			name: "client_credentials requires client id",
			mutate: func(c *Config) {
				c.AuthType = AuthTypeClientCredentials
				c.OAuthClientSecret = "secret"
			},
			expectErr: "oauth_client_id",
		},
		{
			name: "client_credentials requires client secret",
			mutate: func(c *Config) {
				c.AuthType = AuthTypeClientCredentials
				c.OAuthClientID = "myclient"
			},
			expectErr: "ZGSYNC_OAUTH_CLIENT_SECRET",
		},
		{
			name: "client_credentials with id and secret is valid",
			mutate: func(c *Config) {
				c.AuthType = AuthTypeClientCredentials
				c.OAuthClientID = "myclient"
				c.OAuthClientSecret = "secret"
			},
		},
		{
			name: "unknown auth_type is rejected",
			mutate: func(c *Config) {
				c.AuthType = "bogus"
			},
			expectErr: "auth_type",
		},
		{
			name: "empty auth_type falls back to token and requires email",
			mutate: func(c *Config) {
				c.Token = "token123"
			},
			expectErr: "email",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := base
			tt.mutate(&cfg)

			err := cfg.Validation()
			if tt.expectErr == "" {
				if err != nil {
					t.Errorf("Validation() error = %v, want nil", err)
				}
				return
			}
			if err == nil {
				t.Fatalf("Validation() = nil, want error containing %q", tt.expectErr)
			}
			if !strings.Contains(err.Error(), tt.expectErr) {
				t.Errorf("Validation() error = %v, want error containing %q", err, tt.expectErr)
			}
		})
	}
}

func TestConfig_ApplyDefaultsAndEnv(t *testing.T) {
	t.Run("defaults", func(t *testing.T) {
		cfg := Config{}
		cfg.applyDefaultsAndEnv()

		if cfg.AuthType != AuthTypeToken {
			t.Errorf("AuthType = %s, want %s", cfg.AuthType, AuthTypeToken)
		}
		if cfg.OAuthScope != DefaultOAuthScope {
			t.Errorf("OAuthScope = %s, want %s", cfg.OAuthScope, DefaultOAuthScope)
		}
	})

	t.Run("env overrides client id and secret", func(t *testing.T) {
		t.Setenv("ZGSYNC_OAUTH_CLIENT_ID", "env-client")
		t.Setenv("ZGSYNC_OAUTH_CLIENT_SECRET", "env-secret")

		cfg := Config{OAuthClientID: "file-client", OAuthClientSecret: "file-secret"}
		cfg.applyDefaultsAndEnv()

		if cfg.OAuthClientID != "env-client" {
			t.Errorf("OAuthClientID = %s, want env-client", cfg.OAuthClientID)
		}
		if cfg.OAuthClientSecret != "env-secret" {
			t.Errorf("OAuthClientSecret = %s, want env-secret", cfg.OAuthClientSecret)
		}
	})

	t.Run("file values kept without env", func(t *testing.T) {
		cfg := Config{OAuthClientID: "file-client", OAuthScope: "hc:read"}
		cfg.applyDefaultsAndEnv()

		if cfg.OAuthClientID != "file-client" {
			t.Errorf("OAuthClientID = %s, want file-client", cfg.OAuthClientID)
		}
		if cfg.OAuthScope != "hc:read" {
			t.Errorf("OAuthScope = %s, want hc:read", cfg.OAuthScope)
		}
	})
}

func TestConfig_OAuthClientSecretIgnoredInYAML(t *testing.T) {
	t.Parallel()

	var cfg Config
	if err := yaml.Unmarshal([]byte("oauth_client_secret: leaked-secret"), &cfg); err != nil {
		t.Fatalf("yaml.Unmarshal() error = %v", err)
	}
	if cfg.OAuthClientSecret != "" {
		t.Errorf("OAuthClientSecret = %s, want empty (must only be set via env)", cfg.OAuthClientSecret)
	}
}
