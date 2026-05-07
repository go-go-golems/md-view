package commands

import (
	"context"
	"fmt"
	"time"

	"github.com/go-go-golems/glazed/pkg/cli"
	"github.com/go-go-golems/glazed/pkg/cmds"
	"github.com/go-go-golems/glazed/pkg/cmds/values"
	"github.com/go-go-golems/glazed/pkg/middlewares"
	"github.com/go-go-golems/glazed/pkg/settings"
	"github.com/go-go-golems/glazed/pkg/types"
	"github.com/go-go-golems/md-view/pkg/daemon"
)

type StatusCommand struct {
	*cmds.CommandDescription
}

func NewStatusCommand() (*StatusCommand, error) {
	glazedSection, err := settings.NewGlazedSchema()
	if err != nil {
		return nil, err
	}

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
		cmds.WithSections(glazedSection, commandSettingsSection),
	)

	return &StatusCommand{CommandDescription: cmdDesc}, nil
}

func (c *StatusCommand) RunIntoGlazeProcessor(
	ctx context.Context,
	vals *values.Values,
	gp middlewares.Processor,
) error {
	status, err := daemon.GetStatus()
	if err != nil {
		row := types.NewRow(
			types.MRP("running", false),
			types.MRP("error", err.Error()),
		)
		return gp.AddRow(ctx, row)
	}

	row := types.NewRow(
		types.MRP("running", status.Running),
		types.MRP("pid", status.PID),
		types.MRP("port", status.Port),
	)
	if status.Running && !status.StartTime.IsZero() {
		row.Set("uptime", time.Since(status.StartTime).Round(time.Second).String())
	} else {
		row.Set("uptime", "")
	}
	if status.SocketPath != "" {
		row.Set("socket", status.SocketPath)
	}

	return gp.AddRow(ctx, row)
}

func RunStatus() error {
	status, err := daemon.GetStatus()
	if err != nil {
		fmt.Printf("md-view daemon: not running (error: %v)\n", err)
		return nil
	}
	if !status.Running {
		fmt.Println("md-view daemon: not running")
		return nil
	}
	fmt.Printf("md-view daemon: running (PID %d, port %d)\n", status.PID, status.Port)
	if !status.StartTime.IsZero() {
		fmt.Printf("  uptime: %s\n", time.Since(status.StartTime).Round(time.Second))
	}
	return nil
}
