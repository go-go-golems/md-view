package commands

import (
	"context"

	"github.com/go-go-golems/glazed/pkg/cli"
	"github.com/go-go-golems/glazed/pkg/cmds"
	"github.com/go-go-golems/glazed/pkg/cmds/values"
	"github.com/go-go-golems/glazed/pkg/middlewares"
	"github.com/go-go-golems/glazed/pkg/settings"
	"github.com/go-go-golems/glazed/pkg/types"
)

type StopCommand struct {
	*cmds.CommandDescription
}

func NewStopCommand() (*StopCommand, error) {
	glazedSection, err := settings.NewGlazedSchema()
	if err != nil {
		return nil, err
	}

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
		cmds.WithSections(glazedSection, commandSettingsSection),
	)

	return &StopCommand{CommandDescription: cmdDesc}, nil
}

func (c *StopCommand) RunIntoGlazeProcessor(
	ctx context.Context,
	vals *values.Values,
	gp middlewares.Processor,
) error {
	err := RunStop(ctx)
	if err != nil {
		return err
	}

	row := types.NewRow(
		types.MRP("status", "stopped"),
	)
	return gp.AddRow(ctx, row)
}
