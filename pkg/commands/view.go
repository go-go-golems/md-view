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
	"github.com/go-go-golems/glazed/pkg/types"
)

type ViewCommand struct {
	*cmds.CommandDescription
}

type ViewSettings struct {
	File     string `glazed:"file"`
	NoReload bool  `glazed:"no-reload"`
	Browser  string `glazed:"browser"`
	Port     int    `glazed:"port"`
	Dark     bool   `glazed:"dark"`
}

func NewViewCommand() (*ViewCommand, error) {
	glazedSection, err := settings.NewGlazedSchema()
	if err != nil {
		return nil, err
	}

	commandSettingsSection, err := cli.NewCommandSettingsSection()
	if err != nil {
		return nil, err
	}

	cmdDesc := cmds.NewCommandDescription(
		"view",
		cmds.WithShort("View a markdown file in your browser"),
		cmds.WithLong(`
View a markdown file rendered as HTML in your browser.

The md-view daemon is started automatically if not already running.

Examples:
  md-view view ./README.md
  md-view view --no-reload ./notes.md
  md-view view --browser firefox ./doc.md
  md-view view --dark ./doc.md
`),
		cmds.WithArguments(
			fields.New(
				"file",
				fields.TypeString,
				fields.WithHelp("Path to the markdown file to view"),
				fields.WithIsArgument(true),
			),
		),
		cmds.WithFlags(
			fields.New(
				"no-reload",
				fields.TypeBool,
				fields.WithDefault(false),
				fields.WithHelp("Disable live reload for this view"),
			),
			fields.New(
				"browser",
				fields.TypeString,
				fields.WithDefault(""),
				fields.WithHelp("Override browser command (default: xdg-open or $BROWSER)"),
			),
			fields.New(
				"port",
				fields.TypeInteger,
				fields.WithDefault(0),
				fields.WithHelp("HTTP port for the daemon (0 = random available)"),
			),
			fields.New(
				"dark",
				fields.TypeBool,
				fields.WithDefault(false),
				fields.WithHelp("Use dark theme"),
			),
		),
		cmds.WithSections(glazedSection, commandSettingsSection),
	)

	return &ViewCommand{CommandDescription: cmdDesc}, nil
}

func (c *ViewCommand) RunIntoGlazeProcessor(
	ctx context.Context,
	vals *values.Values,
	gp middlewares.Processor,
) error {
	s := &ViewSettings{}
	if err := vals.DecodeSectionInto(schema.DefaultSlug, s); err != nil {
		return err
	}

	url, err := RunView(ctx, s)
	if err != nil {
		return err
	}

	row := types.NewRow(
		types.MRP("url", url),
		types.MRP("file", s.File),
	)
	return gp.AddRow(ctx, row)
}
