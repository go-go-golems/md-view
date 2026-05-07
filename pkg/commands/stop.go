package commands

import (
	"context"
	"fmt"

	"github.com/go-go-golems/glazed/pkg/cli"
	"github.com/go-go-golems/glazed/pkg/cmds"
	"github.com/go-go-golems/glazed/pkg/cmds/values"
)

type StopCommand struct {
	*cmds.CommandDescription
}

func NewStopCommand() (*StopCommand, error) {
	commandSettingsSection, err := cli.NewCommandSettingsSection()
	if err != nil {
		return nil, err
	}

	cmdDesc := cmds.NewCommandDescription(
		"stop",
		cmds.WithShort("Stop the running md-view daemon"),
		cmds.WithLong(`
Stop the md-view daemon by sending SIGTERM to its process.

Examples:
  md-view stop
`),
		cmds.WithSections(commandSettingsSection),
	)

	return &StopCommand{CommandDescription: cmdDesc}, nil
}

func (c *StopCommand) Run(
	ctx context.Context,
	_ *values.Values,
) error {
	err := RunStop(ctx)
	if err != nil {
		return err
	}

	fmt.Println("Daemon stopped.")
	return nil
}
