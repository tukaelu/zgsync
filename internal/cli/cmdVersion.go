package cli

import (
	"fmt"

	"github.com/tukaelu/zgsync"
)

type CommandVersion struct{}

func (c *CommandVersion) Run() error {
	fmt.Printf("version %s (rev: %s)\n", zgsync.Version, zgsync.Revision)
	return nil
}
