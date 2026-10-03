package main

import (
	"context"
	"embed"
	"fmt"
	"io"
	"log"
	"os"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	// No argument opens the window. Any argument must be a known flag: an unknown one once fell
	// through to a window opened on the real database.
	if len(os.Args) > 1 {
		os.Exit(runFlag(os.Args[1:], os.Stdout, os.Stderr))
	}

	app := NewApp()

	err := wails.Run(&options.App{
		Title:  "The War Room",
		Width:  1024,
		Height: 768,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		BackgroundColour: &options.RGBA{R: 27, G: 38, B: 54, A: 1},
		OnStartup:        app.startup,
		OnDomReady:       app.domReady,
		OnShutdown:       app.shutdown,
		Bind: []interface{}{
			app,
		},
	})

	if err != nil {
		log.Fatalf("the war room: %v", err)
	}
}

// runFlag runs a windowless mode and returns the exit code. `-probe` runs the real startup and
// exits 0 if it came up; the log shows each step's time, so a hang is the last step logged.
// `-version` prints the build label.
func runFlag(args []string, stdout, stderr io.Writer) int {
	switch {
	case len(args) == 1 && args[0] == "-probe":
		return probe()
	case len(args) == 1 && args[0] == "-version":
		_, _ = fmt.Fprintln(stdout, buildLabel())
		return 0
	default:
		_, _ = fmt.Fprintf(stderr, "thewarroom: unknown arguments %q\nusage: thewarroom [-probe | -version]\n", args)
		return 2
	}
}

func probe() int {
	app := NewApp()
	app.startup(context.Background())
	defer app.shutdown(context.Background())
	if app.startupErr != nil {
		return 1
	}
	return 0
}
