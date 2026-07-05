package main

import (
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
	// Headless startup diagnostic: `thewarroom -probe` runs the real store-floor
	// init chain with per-step timing + timeout and exits, no Wails/UI. Used to see
	// exactly where startup stalls or errors without a GUI (Ship-4 hang triage).
	if len(os.Args) > 1 && os.Args[1] == "-probe" {
		runProbe()
		return
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
