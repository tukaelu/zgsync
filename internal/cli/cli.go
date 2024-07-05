package cli

import "github.com/alecthomas/kong"

type Global struct {
	ConfigPath string `name:"config" help:"path to the configuration file" default:"~/.config/zgsync/config.yaml" type:"path"`
	Config     Config `kong:"-"`
}

type cli struct {
	Global
	Push    CommandPush    `cmd:"push" help:"Push translations or articles to the remote."`
	Pull    CommandPull    `cmd:"pull" help:"Pull translations or articles from the remote."`
	Empty   CommandEmpty   `cmd:"empty" help:"Creates an empty draft article remotely and saves it locally."`
	Version CommandVersion `cmd:"version" help:"Show version."`
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
