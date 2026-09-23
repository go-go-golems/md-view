package main

import (
	"embed"
	"fmt"
	"log"

	"github.com/go-go-golems/md-view/internal/launch"
	"net/http"

	"github.com/spf13/cobra"
	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
)

// assets holds the embedded frontend (HTML/CSS/JS). Wails serves these to
// the WebView. In `wails dev` they load from disk for hot reload; in
// `wails build` they are baked into this binary.
//
//go:embed all:frontend/dist
var assets embed.FS

// singleInstanceID is the UniqueId for Wails' SingleInstanceLock. A second
// `md-view` process with this id is caught by the lock; its os.Args are
// forwarded to instance #1 via OnSecondInstanceLaunch. This replaces the
// daemon's "reuse running server over a Unix socket" with zero filesystem state.
//
// The id must be a single path segment: on macOS Wails turns it into
// <TMPDIR>/<id>.lock and does not create intermediate directories, so a
// slash-containing id (e.g. a Go import path) fails with ENOENT and the app
// exits as though a second instance were already running.
const singleInstanceID = "com.go-go-golems.md-view"

func main() {
	rootCmd := newRootCommand(runDesktop, func(file string, dark bool) error {
		result, err := launch.Start(file, dark)
		if err != nil {
			return err
		}
		fmt.Printf("Started md-view process %d; log: %s\n", result.PID, result.LogPath)
		return nil
	})
	if err := rootCmd.Execute(); err != nil {
		log.Fatal(err)
	}
}

// newRootCommand separates argument dispatch from native desktop startup.
func newRootCommand(desktop, background func(string, bool) error) *cobra.Command {
	var viewDark, viewForeground bool
	// `md-view view [file] [--dark]` — the primary, drop-in command.
	viewCmd := &cobra.Command{
		Use:   "view [file]",
		Short: "View a markdown file in the md-view window",
		Long: `View a markdown file rendered as HTML in the md-view desktop window.

Runs in the background by default and prints the child PID and private log path.
Success means the process started, not that a window is ready. Use --foreground
to stay attached and see desktop diagnostics.

If the app is already running, Wails attempts to reuse the existing window.

Examples:
  md-view view ./README.md
  md-view view --dark ./notes.md
  md-view view --foreground ./doc.md  # wait until the desktop exits`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			file := ""
			if len(args) == 1 {
				file = args[0]
			}
			if viewForeground {
				return desktop(file, viewDark)
			}
			return background(file, viewDark)
		},
	}
	viewCmd.Flags().BoolVar(&viewDark, "dark", false, "Use the dark theme")
	viewCmd.Flags().BoolVar(&viewForeground, "foreground", false, "Stay in the foreground (do not detach)")

	// Bare `md-view` (no subcommand) opens an empty window — this is also what
	// happens when the binary is double-clicked. (Wails single-instance: a 2nd
	// bare launch just focuses the existing window.)
	rootCmd := &cobra.Command{
		Use:   "md-view",
		Short: "A markdown viewer desktop application",
		RunE: func(cmd *cobra.Command, args []string) error {
			file := ""
			if len(args) > 0 {
				file = args[0]
			}
			return desktop(file, false)
		},
	}
	rootCmd.AddCommand(viewCmd)

	return rootCmd
}

// runDesktop starts the Wails app, opening `file` (if non-empty) with the
// given initial theme. It blocks until the window closes.
func runDesktop(file string, dark bool) error {
	// Resolve a relative path to absolute NOW, in the invoking process, so:
	//   - the first instance opens the right file (PendingOpen is absolute,
	//     resolved against the caller's cwd), and
	//   - if Wails' SingleInstanceLock forwards this process to an already-
	//     running instance, the forwarded os.Args carry an absolute path.
	//     This matters on macOS, where Wails sets the forwarded
	//     SecondInstanceData.WorkingDirectory to the executable's directory
	//     (filepath.Dir(os.Executable())) rather than the caller's cwd, so a
	//     relative path joined against it in OnSecondInstanceLaunch would
	//     resolve to the wrong location. (Linux/Windows forward os.Getwd().)
	file = absolutizeFileArg(file)

	app := NewApp()
	app.PendingOpen = file
	app.PendingDark = dark
	if dark {
		app.theme = "dark"
	}

	return wails.Run(&options.App{
		Title:     "md-view",
		Width:     1024,
		Height:    768,
		MinWidth:  480,
		MinHeight: 360,
		AssetServer: &assetserver.Options{
			Assets:  assets,
			Handler: http.HandlerFunc(app.ServeReferencedFile), // serves /file/<abs> images (DR-5)
		},
		Menu:       buildMenu(app),
		OnStartup:  app.Startup,
		OnDomReady: app.OnDomReady,
		OnShutdown: app.Shutdown,
		Bind: []interface{}{
			app,
		},
		DragAndDrop: &options.DragAndDrop{
			EnableFileDrop: true,
		},
		SingleInstanceLock: &options.SingleInstanceLock{
			UniqueId:               singleInstanceID,
			OnSecondInstanceLaunch: app.OnSecondInstanceLaunch,
		},
	})
}
