package composition

import (
	"context"
	"fmt"
	"io"
	"io/fs"
	"log"
	"os"

	archiveapp "github.com/secureprospective/TheWarRoom/internal/app/archive"
	"github.com/secureprospective/TheWarRoom/internal/app/comms"
	contractsapp "github.com/secureprospective/TheWarRoom/internal/app/contracts"
	controlapp "github.com/secureprospective/TheWarRoom/internal/app/control"
	"github.com/secureprospective/TheWarRoom/internal/app/counterparties"
	"github.com/secureprospective/TheWarRoom/internal/app/draft"
	nowapp "github.com/secureprospective/TheWarRoom/internal/app/now"
	"github.com/secureprospective/TheWarRoom/internal/app/offers"
	raceapp "github.com/secureprospective/TheWarRoom/internal/app/race"
	"github.com/secureprospective/TheWarRoom/internal/app/research"
	"github.com/secureprospective/TheWarRoom/internal/app/trades"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
)

// Embedded is what the root binary embeds and hands in: embed cannot reach above a package
// directory, and the frontend bundle, the rulebook and the endpoint registry live at the repo root.
type Embedded struct {
	Assets           fs.FS
	Rulebook         []byte
	EndpointRegistry string
}

// Main wires and runs the app and returns the process exit code. No argument opens the window. Any
// argument must be a known flag: an unknown one once fell through to a window opened on the real
// database.
func Main(args []string, embedded Embedded) int {
	rulebookMarkdown, commissionerPreviewRegistry = embedded.Rulebook, embedded.EndpointRegistry
	if len(args) > 0 {
		return runFlag(args, os.Stdout, os.Stderr)
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
			Assets: embedded.Assets,
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
		log.Printf("the war room: %v", err)
		return 1
	}
	return 0
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
		archiveapp.New(app.core),
		newCommissionerAPI(app),
		comms.New(app.core),
		controlapp.New(controlHost{State: app.core, app: app}, app.controlHeld),
		contractsapp.New(contractsHost{State: app.core, app: app}),
		counterparties.New(app.core),
		draft.New(draftHost{State: app.core, app: app}),
		newMarketAPI(app),
		nowapp.New(nowHost{State: app.core, app: app}, app.nowHeld),
		offers.New(offersHost{State: app.core, app: app}),
		app.platformAPI(),
		raceapp.New(app.core, app.raceHeld),
		research.New(researchHost{app: app}, app.researchHeld),
		app.sourcesAPI(),
		trades.New(tradesHost{State: app.core, app: app}),
	}
}
