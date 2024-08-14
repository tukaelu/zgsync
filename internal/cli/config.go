package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Subdomain                string `yaml:"subdomain" description:"Zendesk subdomain" required:"true"`
	Email                    string `yaml:"email" description:"Zendesk email" required:"true"`
	Token                    string `yaml:"token" description:"Zendesk API token" required:"true"`
	DefaultCommentsDisabled  bool   `yaml:"default_comments_disabled" description:"Default comments disabled" default:"false"`
	DefaultLocale            string `yaml:"default_locale" description:"Default locale for articles" required:"true"`
	DefaultPermissionGroupID int    `yaml:"default_permission_group_id" description:"Default permission group ID" required:"true"`
	DefailtUserSegmentID     *int   `yaml:"default_user_segment_id" description:"Default user segment ID"`
	NotifySubscribers        bool   `yaml:"notify_subscribers" description:"Notify subscribers when creating or updating articles" default:"false"`
	ContentsDir              string `yaml:"contents_dir" description:"Path to the contents directory" default:"."`
}

func (c *Config) Validation() error {
	if c.Subdomain == "" {
		return fmt.Errorf("subdomain is required")
	}
	if c.Email == "" {
		return fmt.Errorf("email is required")
	}
	if c.Token == "" {
		return fmt.Errorf("token is required")
	}
	if c.DefaultLocale == "" {
		return fmt.Errorf("default_locale is required")
	}
	if c.DefaultPermissionGroupID == 0 {
		return fmt.Errorf("default_permission_group_id is required")
	}
	return nil
}

func (g *Global) LoadConfig() error {
	if g.ConfigPath == "" {
		home, _ := os.UserHomeDir()
		g.ConfigPath = filepath.Join(home, ".config", "zgsync", "config.yaml")
	}
	b, err := os.ReadFile(g.ConfigPath)
	if err != nil {
		return nil
	}
	if err := yaml.Unmarshal(b, &g.Config); err != nil {
		return err
	}
	if g.Config.ContentsDir == "" {
		g.Config.ContentsDir = "."
	}
	return g.Config.Validation()
}

func (g *Global) ConfigExists() error {
	abs := g.AbsConfig()
	if _, err := os.Stat(abs); os.IsNotExist(err) {
		return fmt.Errorf("config file %s does not exists.", abs)
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
