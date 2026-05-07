package main

import (
	"github.com/go-go-golems/glazed/pkg/cli"
	"github.com/go-go-golems/glazed/pkg/cmds/logging"
	"github.com/go-go-golems/glazed/pkg/help"
	help_cmd "github.com/go-go-golems/glazed/pkg/help/cmd"
	"github.com/go-go-golems/md-view/pkg/commands"
	"github.com/spf13/cobra"
)

var version = "dev"

var rootCmd = &cobra.Command{
	Use:     "md-view",
	Short:   "md-view is a markdown viewer daemon that renders .md files in your browser",
	Version: version,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		return logging.InitLoggerFromCobra(cmd)
	},
}

func main() {
	err := logging.AddLoggingSectionToRootCommand(rootCmd, "md-view")
	cobra.CheckErr(err)

	helpSystem := help.NewHelpSystem()
	help_cmd.SetupCobraRootCommand(helpSystem, rootCmd)

	// View command (default verb)
	viewCmd, err := commands.NewViewCommand()
	cobra.CheckErr(err)
	viewCobraCmd, err := cli.BuildCobraCommand(viewCmd,
		cli.WithParserConfig(cli.CobraParserConfig{AppName: "md-view"}),
	)
	cobra.CheckErr(err)
	rootCmd.AddCommand(viewCobraCmd)

	// Serve command (foreground server)
	serveCmd, err := commands.NewServeCommand()
	cobra.CheckErr(err)
	serveCobraCmd, err := cli.BuildCobraCommand(serveCmd,
		cli.WithParserConfig(cli.CobraParserConfig{AppName: "md-view"}),
	)
	cobra.CheckErr(err)
	rootCmd.AddCommand(serveCobraCmd)

	// Stop command
	stopCmd, err := commands.NewStopCommand()
	cobra.CheckErr(err)
	stopCobraCmd, err := cli.BuildCobraCommand(stopCmd,
		cli.WithParserConfig(cli.CobraParserConfig{AppName: "md-view"}),
	)
	cobra.CheckErr(err)
	rootCmd.AddCommand(stopCobraCmd)

	// Status command
	statusCmd, err := commands.NewStatusCommand()
	cobra.CheckErr(err)
	statusCobraCmd, err := cli.BuildCobraCommand(statusCmd,
		cli.WithParserConfig(cli.CobraParserConfig{AppName: "md-view"}),
	)
	cobra.CheckErr(err)
	rootCmd.AddCommand(statusCobraCmd)

	cobra.CheckErr(rootCmd.Execute())
}
