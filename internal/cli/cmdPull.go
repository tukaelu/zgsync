package cli

import (
	"fmt"
	"path/filepath"
	"strconv"

	"github.com/tukaelu/zgsync/internal/converter"
	"github.com/tukaelu/zgsync/internal/zendesk"
)

type CommandPull struct {
	Locale      string              `name:"locale" short:"l" help:"Specify the locale to pull. If not specified, the default locale will be used."`
	Raw         bool                `name:"raw" help:"It pulls raw data without converting it from HTML to Markdown."`
	SaveArticle bool                `name:"save-article" short:"a" help:"It pulls and saves the article in addition to the translation."`
	ArticleIDs  []int               `arg:"" help:"Specify the article IDs to pull." type:"int"`
	client      zendesk.Client      `kong:"-"`
	converter   converter.Converter `kong:"-"`
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
		if c.SaveArticle {
			resPayload, err := c.client.ShowArticle(c.Locale, articleID)
			if err != nil {
				return err
			}
			a := &zendesk.Article{}
			a.FromJson(resPayload)
			aPath := filepath.Join(g.Config.ContentsDir, strconv.Itoa(a.ID)+".md")
			if err = a.Save(aPath); err != nil {
				return fmt.Errorf("failed to save the article to %s: %w", aPath, err)
			}
		}

		resPayload, err := c.client.ShowTranslation(articleID, c.Locale)
		if err != nil {
			return err
		}
		t := &zendesk.Translation{}
		t.FromJson(resPayload)

		if !c.Raw {
			if t.Body, err = c.converter.ConvertToMarkdown(t.Body); err != nil {
				return err
			}
		}

		tPath := filepath.Join(g.Config.ContentsDir, strconv.Itoa(t.SourceID)+"-"+t.Locale+".md")
		if err = t.Save(tPath); err != nil {
			return fmt.Errorf("failed to save the translation to %s: %w", tPath, err)
		}
	}
	return nil
}
