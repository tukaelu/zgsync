package cli

import "github.com/alecthomas/kong"

type Global struct {
	ConfigPath string `name:"config" help:"path to the configuration file" default:"~/.config/zgsync/config.yaml" type:"path"`
	Config     Config `kong:"-"`
}

type cli struct {
	Global
	Push    CommandPush    `cmd:"push" help:"push articles/translations to remote"`
	Pull    CommandPull    `cmd:"pull" help:"pull articles/translations from remote"`
	Empty   CommandEmpty   `cmd:"empty" help:"create an empty draft article"`
	Version CommandVersion `cmd:"version" help:"show version"`
}

func (c *cli) AfterApply(kCtx *kong.Context) error {
	if kCtx.Command() == "version" {
		return nil
	}
	if err := c.Global.ConfigExists(); err != nil {
		return err
	}
	if err := c.Global.LoadConfig(); err != nil {
		return err
	}
	return nil
}

func Bind() {
	c := &cli{}
	kCtx := kong.Parse(c,
		kong.Name("zgsync"),
		kong.Description("zgsync is a command-line tool for posting Markdown files as articles to Zendesk Guide."),
		kong.UsageOnError(),
		kong.Bind(&c.Global),
	)
	err := kCtx.Run()
	kCtx.FatalIfErrorf(err)
}
