package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

const (
	AuthTypeToken             = "token"
	AuthTypeOAuth             = "oauth"
	AuthTypeClientCredentials = "client_credentials"
)

// DefaultOAuthScope restricts tokens to the Help Center API, which is all zgsync needs.
const DefaultOAuthScope = "hc:read hc:write"

type Config struct {
	Subdomain                string `yaml:"subdomain" description:"Zendesk subdomain" required:"true"`
	AuthType                 string `yaml:"auth_type" description:"Authentication type (token, oauth, or client_credentials)" default:"token"`
	Email                    string `yaml:"email" description:"Zendesk email (required for auth_type: token)"`
	Token                    string `yaml:"token" description:"Zendesk API token (required for auth_type: token)"`
	OAuthClientID            string `yaml:"oauth_client_id" description:"OAuth client ID (required for auth_type: oauth and client_credentials)"`
	OAuthClientSecret        string `yaml:"-"` // secret is only accepted via ZGSYNC_OAUTH_CLIENT_SECRET so it never lives in a config file
	OAuthScope               string `yaml:"oauth_scope" description:"OAuth scope" default:"hc:read hc:write"`
	DefaultCommentsDisabled  bool   `yaml:"default_comments_disabled" description:"Default comments disabled" default:"false"`
	DefaultLocale            string `yaml:"default_locale" description:"Default locale for articles" required:"true"`
	DefaultPermissionGroupID int    `yaml:"default_permission_group_id" description:"Default permission group ID" required:"true"`
	DefailtUserSegmentID     *int   `yaml:"default_user_segment_id" description:"Default user segment ID"`
	NotifySubscribers        bool   `yaml:"notify_subscribers" description:"Notify subscribers when creating or updating articles" default:"false"`
	ContentsDir              string `yaml:"contents_dir" description:"Path to the contents directory" default:"."`
	EnableLinkTargetBlank    bool   `yaml:"enable_link_target_blank" description:"This makes links open in a new tab" default:"false"`
}

func (c *Config) Validation() error {
	if c.Subdomain == "" {
		return fmt.Errorf("subdomain is required")
	}
	switch c.AuthType {
	case "", AuthTypeToken:
		if c.Email == "" {
			return fmt.Errorf("email is required")
		}
		if c.Token == "" {
			return fmt.Errorf("token is required")
		}
	case AuthTypeOAuth:
		if c.OAuthClientID == "" {
			return fmt.Errorf("oauth_client_id is required for auth_type: oauth")
		}
	case AuthTypeClientCredentials:
		if c.OAuthClientID == "" {
			return fmt.Errorf("oauth_client_id is required for auth_type: client_credentials")
		}
		if c.OAuthClientSecret == "" {
			return fmt.Errorf("the ZGSYNC_OAUTH_CLIENT_SECRET environment variable is required for auth_type: client_credentials")
		}
	default:
		return fmt.Errorf("auth_type must be one of %s, %s, or %s", AuthTypeToken, AuthTypeOAuth, AuthTypeClientCredentials)
	}
	if c.DefaultLocale == "" {
		return fmt.Errorf("default_locale is required")
	}
	if c.DefaultPermissionGroupID == 0 {
		return fmt.Errorf("default_permission_group_id is required")
	}
	return nil
}

// applyDefaultsAndEnv fills in default values and applies environment variable
// overrides so that secrets can be injected via CI secrets instead of the config file.
func (c *Config) applyDefaultsAndEnv() {
	if c.AuthType == "" {
		c.AuthType = AuthTypeToken
	}
	if c.OAuthScope == "" {
		c.OAuthScope = DefaultOAuthScope
	}
	if v := os.Getenv("ZGSYNC_OAUTH_CLIENT_ID"); v != "" {
		c.OAuthClientID = v
	}
	if v := os.Getenv("ZGSYNC_OAUTH_CLIENT_SECRET"); v != "" {
		c.OAuthClientSecret = v
	}
}

// zgsyncConfigDir is the single source of the config directory path, shared
// with the credential store so the two files can never end up in different places.
func zgsyncConfigDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".config", "zgsync"), nil
}

func (g *Global) LoadConfig() error {
	if err := g.readConfigFile(); err != nil {
		return err
	}
	return g.Config.Validation()
}

// LoadConfigForAuthLogin loads the config but only validates what `auth login`
// actually needs. Content settings and other auth types' requirements (API
// tokens, the client_credentials secret) are irrelevant to logging in, and
// requiring them would block users migrating to OAuth.
func (g *Global) LoadConfigForAuthLogin() error {
	if err := g.readConfigFile(); err != nil {
		return err
	}
	if g.Config.Subdomain == "" {
		return fmt.Errorf("subdomain is required")
	}
	return nil
}

func (g *Global) readConfigFile() error {
	if g.ConfigPath == "" {
		dir, err := zgsyncConfigDir()
		if err != nil {
			return err
		}
		g.ConfigPath = filepath.Join(dir, "config.yaml")
	}
	b, err := os.ReadFile(g.ConfigPath)
	if err != nil {
		return fmt.Errorf("failed to read config file %s: %w", g.ConfigPath, err)
	}
	if err := yaml.Unmarshal(b, &g.Config); err != nil {
		return err
	}
	if g.Config.ContentsDir == "" {
		g.Config.ContentsDir = "."
	}
	g.Config.applyDefaultsAndEnv()
	return nil
}

func (g *Global) ConfigExists() error {
	abs := g.AbsConfig()
	if _, err := os.Stat(abs); os.IsNotExist(err) {
		return fmt.Errorf("config file %s does not exist", abs)
	}
	return nil
}

func (g *Global) AbsConfig() string {
	if abs, err := filepath.Abs(g.ConfigPath); err != nil {
		return g.ConfigPath
	} else {
		return abs
	}
}
