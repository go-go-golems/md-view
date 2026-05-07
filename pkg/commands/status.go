package commands

import (
	"context"

	"github.com/go-go-golems/glazed/pkg/cli"
	"github.com/go-go-golems/glazed/pkg/cmds"
	"github.com/go-go-golems/glazed/pkg/cmds/values"
)

type StatusCommand struct {
	*cmds.CommandDescription
}

func NewStatusCommand() (*StatusCommand, error) {
	commandSettingsSection, err := cli.NewCommandSettingsSection()
	if err != nil {
		return nil, err
	}

	cmdDesc := cmds.NewCommandDescription(
		"status",
		cmds.WithShort("Show the status of the md-view daemon"),
		cmds.WithLong(`
Show whether the md-view daemon is running, its PID, HTTP port, and uptime.

Examples:
  md-view status
`),
		cmds.WithSections(commandSettingsSection),
	)

	return &StatusCommand{CommandDescription: cmdDesc}, nil
}

func (c *StatusCommand) Run(
	_ context.Context,
	_ *values.Values,
) error {
	return RunStatus()
}
