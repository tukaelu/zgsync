package cli

import (
	"fmt"
	"path/filepath"
	"strconv"

	"github.com/tukaelu/zgsync/internal/zendesk"
)

type CommandEmpty struct {
	Locale            string         `name:"locale" short:"l" help:"locale"`
	PermissionGroupID int            `name:"permission-group-id" short:"p" help:"permission group ID"`
	SaveArticle       bool           `name:"save-article" help:"save article to the contents directory"`
	SectionID         int            `name:"section-id" short:"s" help:"section ID" required:""`
	Title             string         `name:"title" short:"t" help:"title" required:""`
	UserSegmentID     int            `name:"user-segment-id" short:"u" help:"user segment ID"`
	client            zendesk.Client `kong:"-"`
}

func (c *CommandEmpty) AfterApply(g *Global) error {
	c.client = zendesk.NewClient(g.Config.Subdomain, g.Config.Email, g.Config.Token)
	return nil
}

func (c *CommandEmpty) Run(g *Global) error {
	if c.Locale == "" {
		c.Locale = g.Config.DefaultLocale
	}
	if c.PermissionGroupID == 0 {
		c.PermissionGroupID = g.Config.DefaultPermissionGroupID
	}
	if c.UserSegmentID == 0 {
		c.UserSegmentID = g.Config.DefailtUserSegmentID
	}

	a := &zendesk.Article{
		Draft:             true,
		Locale:            c.Locale,
		PermissionGroupID: c.PermissionGroupID,
		SectionID:         c.SectionID,
		Title:             c.Title,
		UserSegmentID:     c.UserSegmentID,
		Body:              "",
	}
	payload, err := a.ToPayload(g.Config.NotifySubscribers)
	if err != nil {
		return err
	}

	res, err := c.client.CreateArticle(c.Locale, c.SectionID, payload)
	if err != nil {
		return err
	}

	if err = a.FromJson(res); err != nil {
		return err
	}

	if c.SaveArticle {
		aPath := filepath.Join(g.Config.ContentsDir, strconv.Itoa(a.ID), ".md")
		if err = a.Save(aPath); err != nil {
			return fmt.Errorf("failed to save the article to %s: %w", aPath, err)
		}
	}

	res, err = c.client.ShowTranslation(a.ID, c.Locale)
	if err != nil {
		return err
	}

	t := &zendesk.Translation{}
	if err = t.FromJson(res); err != nil {
		return err
	}
	tPath := filepath.Join(g.Config.ContentsDir, strconv.Itoa(t.SourceID), "-", t.Locale, ".md")
	if err = t.Save(tPath); err != nil {
		return fmt.Errorf("failed to save the translation to %s: %w", tPath, err)
	}
	return nil
}
