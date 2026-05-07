package commands

import (
	"context"

	"github.com/go-go-golems/glazed/pkg/cli"
	"github.com/go-go-golems/glazed/pkg/cmds"
	"github.com/go-go-golems/glazed/pkg/cmds/fields"
	"github.com/go-go-golems/glazed/pkg/cmds/schema"
	"github.com/go-go-golems/glazed/pkg/cmds/values"
	"github.com/go-go-golems/glazed/pkg/middlewares"
	"github.com/go-go-golems/glazed/pkg/settings"
)

type ServeCommand struct {
	*cmds.CommandDescription
}

type ServeSettings struct {
	Port int `glazed:"port"`
}

func NewServeCommand() (*ServeCommand, error) {
	glazedSection, err := settings.NewGlazedSchema()
	if err != nil {
		return nil, err
	}

	commandSettingsSection, err := cli.NewCommandSettingsSection()
	if err != nil {
		return nil, err
	}

	cmdDesc := cmds.NewCommandDescription(
		"serve",
		cmds.WithShort("Start the markdown viewer server in the foreground"),
		cmds.WithLong(`
Start the md-view HTTP server in the foreground.

This is normally called internally by the daemon, but can be run directly
for debugging.

Examples:
  md-view serve
  md-view serve --port 8080
`),
		cmds.WithFlags(
			fields.New(
				"port",
				fields.TypeInteger,
				fields.WithDefault(0),
				fields.WithHelp("HTTP port (0 = random available)"),
			),
		),
		cmds.WithSections(glazedSection, commandSettingsSection),
	)

	return &ServeCommand{CommandDescription: cmdDesc}, nil
}

func (c *ServeCommand) RunIntoGlazeProcessor(
	ctx context.Context,
	vals *values.Values,
	gp middlewares.Processor,
) error {
	s := &ServeSettings{}
	if err := vals.DecodeSectionInto(schema.DefaultSlug, s); err != nil {
		return err
	}

	err := RunServe(ctx, s, gp)
	if err != nil {
		return err
	}

	return nil
}
