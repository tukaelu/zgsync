package cli

import (
	"fmt"
	"strconv"

	"github.com/tukaelu/zgsync/internal/zendesk"
)

type CommandArchive struct {
	Target string         `arg:"" name:"target" help:"Specify the article ID or file path of the article to archive."`
	client zendesk.Client `kong:"-"`
}

func (c *CommandArchive) AfterApply(g *Global) error {
	c.client = zendesk.NewClient(g.Config.Subdomain, g.Config.Email, g.Config.Token)
	return nil
}

func (c *CommandArchive) Run(g *Global) error {
	articleID, err := c.resolveArticleID()
	if err != nil {
		return err
	}
	if err := c.client.ArchiveArticle(articleID); err != nil {
		return fmt.Errorf("failed to archive article %d: %w", articleID, err)
	}
	return nil
}

func (c *CommandArchive) resolveArticleID() (int, error) {
	if id, err := strconv.Atoi(c.Target); err == nil {
		if id <= 0 {
			return 0, fmt.Errorf("invalid article ID: %d", id)
		}
		return id, nil
	}

	a := &zendesk.Article{}
	if err := a.FromFile(c.Target); err != nil {
		return 0, fmt.Errorf("failed to read article file: %w", err)
	}
	if a.ID == 0 {
		return 0, fmt.Errorf("article ID not found in file: %s", c.Target)
	}
	return a.ID, nil
}
