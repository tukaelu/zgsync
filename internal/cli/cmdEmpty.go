package cli

import (
	"fmt"
	"path/filepath"
	"strconv"

	"github.com/tukaelu/zgsync/internal/zendesk"
)

type CommandEmpty struct {
	SectionID         int            `name:"section-id" short:"s" help:"Specify the section ID of the article." required:""`
	Title             string         `name:"title" short:"t" help:"Specify the title of the article." required:""`
	Locale            string         `name:"locale" short:"l" help:"Specify the locale to pull. If not specified, the default locale will be used."`
	PermissionGroupID int            `name:"permission-group-id" short:"p" help:"Specify the permission group ID. If not specified, the default value will be used."`
	UserSegmentID     *int           `name:"user-segment-id" short:"u" help:"Specify the user segment ID. If not specified, the default value will be used."`
	SaveArticle       bool           `name:"save-article" help:"It saves the article in addition to the translation."`
	WithoutSectionDir bool           `name:"without-section-dir" help:"It doesn't save in a directory named after the section ID."`
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
	if c.UserSegmentID == nil {
		c.UserSegmentID = g.Config.DefailtUserSegmentID
	}

	a := &zendesk.Article{
		Draft:             true,
		CommentsDisabled:  g.Config.DefaultCommentsDisabled,
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

	var saveDirPath string
	if c.WithoutSectionDir {
		saveDirPath = g.Config.ContentsDir
	} else {
		saveDirPath = filepath.Join(g.Config.ContentsDir, strconv.Itoa(a.SectionID))
	}

	if c.SaveArticle {
		if err = a.Save(saveDirPath, true); err != nil {
			return fmt.Errorf("failed to save the article: %w", err)
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
	t.SectionID = a.SectionID

	if err = t.Save(saveDirPath, true); err != nil {
		return fmt.Errorf("failed to save the translation: %w", err)
	}
	return nil
}
