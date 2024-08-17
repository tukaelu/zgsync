package cli

import (
	"fmt"
	"path/filepath"
	"strconv"

	"github.com/tukaelu/zgsync/internal/converter"
	"github.com/tukaelu/zgsync/internal/zendesk"
)

type CommandPull struct {
	Locale         string              `name:"locale" short:"l" help:"Specify the locale to pull. If not specified, the default locale will be used."`
	Raw            bool                `name:"raw" help:"It pulls raw data without converting it from HTML to Markdown."`
	SaveArticle    bool                `name:"save-article" short:"a" help:"It pulls and saves the article in addition to the translation."`
	WithSectionDir bool                `name:"with-section-dir" short:"S" help:"A .md file will be created in the section ID directory."`
	ArticleIDs     []int               `arg:"" help:"Specify the article IDs to pull." type:"int"`
	client         zendesk.Client      `kong:"-"`
	converter      converter.Converter `kong:"-"`
}

func (c *CommandPull) AfterApply(g *Global) error {
	c.client = zendesk.NewClient(g.Config.Subdomain, g.Config.Email, g.Config.Token)
	c.converter = converter.NewConverter()
	return nil
}

func (c *CommandPull) Run(g *Global) error {
	if c.Locale == "" {
		c.Locale = g.Config.DefaultLocale
	}

	for _, articleID := range c.ArticleIDs {
		res, err := c.client.ShowArticle(c.Locale, articleID)
		if err != nil {
			return err
		}
		a := &zendesk.Article{}
		if err := a.FromJson(res); err != nil {
			return err
		}

		saveDirPath := g.Config.ContentsDir
		if c.WithSectionDir {
			saveDirPath = filepath.Join(g.Config.ContentsDir, strconv.Itoa(a.SectionID))
		}

		if c.SaveArticle {
			if err = a.Save(saveDirPath, true); err != nil {
				return fmt.Errorf("failed to save the article: %w", err)
			}
		}

		res, err = c.client.ShowTranslation(articleID, c.Locale)
		if err != nil {
			return err
		}
		t := &zendesk.Translation{}
		if err := t.FromJson(res); err != nil {
			return err
		}
		t.SectionID = a.SectionID

		if !c.Raw {
			if t.Body, err = c.converter.ConvertToMarkdown(t.Body); err != nil {
				return err
			}
		}

		if err = t.Save(saveDirPath, true); err != nil {
			return fmt.Errorf("failed to save the translation: %w", err)
		}
	}
	return nil
}
