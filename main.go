package main

import (
	"context"
	"embed"
	"fmt"
	"io"
	"log"
	"os"

	archiveapp "github.com/secureprospective/TheWarRoom/internal/app/archive"
	"github.com/secureprospective/TheWarRoom/internal/app/comms"
	"github.com/secureprospective/TheWarRoom/internal/app/counterparties"
	"github.com/secureprospective/TheWarRoom/internal/app/draft"
	raceapp "github.com/secureprospective/TheWarRoom/internal/app/race"
	"github.com/secureprospective/TheWarRoom/internal/app/sources"

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
		Title: "The War Room",
		// Opens maximised; 1280x800 is the restore size and the size every render gate uses.
		// The shell needs it: rail, inspector and comms leave the workspace 718px at 1280.
		Width:            1280,
		Height:           800,
		WindowStartState: options.Maximised,
		// Wails on Linux turns a zero maximum into the monitor size at launch, so after a resolution change the
		// window could not grow past the old screen. An explicit ceiling no display reaches keeps it unbounded.
		MaxWidth:  16384,
		MaxHeight: 16384,
		AssetServer: &assetserver.Options{
			Assets: assets,
			// Anything not in the bundle: player photos at /headshot/<mfl id>.
			Handler: app.photoHandler(),
		},
		// --surface-0 (hsl(220 14% 4%)), opaque, so the window never flashes another colour.
		BackgroundColour: &options.RGBA{R: 9, G: 10, B: 12, A: 255},
		OnStartup:        app.startup,
		OnDomReady:       app.domReady,
		OnShutdown:       app.shutdown,
		Bind:             bindings(app),
	})

	if err != nil {
		log.Fatalf("the war room: %v", err)
	}
}

// runFlag runs a windowless mode and returns the exit code. `-probe` runs the real startup and
// exits 0 if it came up; the log shows each step's time, so a hang is the last step logged.
// `-version` prints the build label. `-backups` lists the backup sets and `-restore <set>` restores
// one, both with the app closed.
func runFlag(args []string, stdout, stderr io.Writer) int {
	switch {
	case len(args) == 1 && args[0] == "-probe":
		return probe()
	case len(args) == 1 && args[0] == "-version":
		_, _ = fmt.Fprintln(stdout, buildLabel())
		return 0
	case len(args) == 1 && args[0] == "-backups":
		return listBackups(stdout, stderr)
	case len(args) == 2 && args[0] == "-restore":
		return restoreBackup(args[1], stdout, stderr)
	default:
		_, _ = fmt.Fprintf(stderr, "thewarroom: unknown arguments %q\nusage: thewarroom [-probe | -version | -backups | -restore <set>]\n", args)
		return 2
	}
}

func probe() int {
	app := NewApp()
	app.startup(context.Background())
	defer app.shutdown(context.Background())
	if app.core.StartupErr != nil {
		return 1
	}
	return 0
}

// bindings is the runtime surface also inspected by the binding manifest test.
func bindings(app *App) []interface{} {
	return []interface{}{
		app,
		archiveapp.New(app.core),
		comms.New(app.core),
		counterparties.New(app.core),
		draft.New(draftHost{State: app.core, app: app}),
		raceapp.New(app.core, app.raceHeld),
		sources.New(app.core, app.sourcesHeld),
	}
}
