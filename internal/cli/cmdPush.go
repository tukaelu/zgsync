package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/tukaelu/zgsync/internal/converter"
	"github.com/tukaelu/zgsync/internal/zendesk"
)

type CommandPush struct {
	Article   bool                `name:"article" help:"Specify when posting an article. If not specified, the translation will be pushed."`
	DryRun    bool                `name:"dry-run" help:"dry run"`
	Raw       bool                `name:"raw" help:"It pushes raw data without converting it from Markdown to HTML."`
	Files     []string            `arg:"" help:"Specify the files to push." type:"existingfile"`
	client    zendesk.Client      `kong:"-"`
	converter converter.Converter `kong:"-"`
}

func (c *CommandPush) AfterApply(g *Global) error {
	c.client = zendesk.NewClient(g.Config.Subdomain, g.Config.Email, g.Config.Token)
	c.converter = converter.NewConverter()
	return nil
}

func (c *CommandPush) Run(g *Global) error {
	var err error
	for _, file := range c.Files {
		if !filepath.IsAbs(file) {
			if file, err = filepath.Abs(file); err != nil {
				return err
			}
		}

		if _, err = os.Stat(file); os.IsNotExist(err) {
			return fmt.Errorf("file %s does not exist", file)
		}

		if c.Article {
			if err := c.pushArticle(g, file); err != nil {
				return err
			}
			continue
		}

		if err = c.pushTranslation(g, file); err != nil {
			return err
		}
	}
	return nil
}

func (c *CommandPush) pushArticle(g *Global, file string) error {
	a := &zendesk.Article{}
	if err := a.FromFile(file); err != nil {
		return err
	}

	if c.DryRun {
		dryRun(a, file)
		return nil
	}

	payload, err := a.ToPayload(g.Config.NotifySubscribers)
	if err != nil {
		return err
	}

	var locale string
	if a.Locale == "" {
		locale = g.Config.DefaultLocale
	} else {
		locale = a.Locale
	}

	_, err = c.client.UpdateArticle(locale, a.ID, payload)
	if err != nil {
		return err
	}

	return nil
}

func (c *CommandPush) pushTranslation(g *Global, file string) error {
	t := &zendesk.Translation{}
	err := t.FromFile(file)
	if err != nil {
		return err
	}

	if !c.Raw {
		if t.Body, err = c.converter.ConvertToHTML(t.Body); err != nil {
			return err
		}
	}

	if c.DryRun {
		dryRun(t, file)
		return nil
	}

	payload, err := t.ToPayload()
	if err != nil {
		return err
	}

	var locale string
	if t.Locale == "" {
		locale = g.Config.DefaultLocale
	} else {
		locale = t.Locale
	}

	_, err = c.client.UpdateTranslation(t.SourceID, locale, payload)
	if err != nil {
		return err
	}

	return nil
}

func dryRun(v interface{}, file string) {
	prettyPayload, _ := json.MarshalIndent(v, "", "  ")
	fmt.Printf("file: %s\n", file)
	fmt.Println(string(prettyPayload))
}
