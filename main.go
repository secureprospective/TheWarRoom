package main

import (
	"context"
	"embed"
	"log"
	"os"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
)

//go:embed all:frontend/dist
var assets embed.FS // populated by the //go:embed directive above; gochecknoglobals exempts embed vars (no nolint needed).

func main() {
	// `thewarroom -probe` runs the real startup with no window and exits 0 if it came up.
	// The log shows each step's time, so a hang is the last step logged.
	if len(os.Args) > 1 && os.Args[1] == "-probe" {
		os.Exit(probe())
	}

	// Create an instance of the app structure
	app := NewApp()

	// Create application with options
	err := wails.Run(&options.App{
		Title:  "The War Room",
		Width:  1024,
		Height: 768,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		BackgroundColour: &options.RGBA{R: 27, G: 38, B: 54, A: 1},
		OnStartup:        app.startup,
		OnShutdown:       app.shutdown,
		Bind: []interface{}{
			app,
		},
	})

	if err != nil {
		log.Fatalf("the war room: %v", err)
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
