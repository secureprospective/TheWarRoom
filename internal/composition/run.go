package composition

import (
	"context"
	"fmt"
	"io"
	"io/fs"
	"log"
	"os"
	"path/filepath"

	"github.com/secureprospective/TheWarRoom/internal/accounts"
	archiveapp "github.com/secureprospective/TheWarRoom/internal/app/archive"
	commissionerapp "github.com/secureprospective/TheWarRoom/internal/app/commissioner"
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

	"github.com/secureprospective/TheWarRoom/internal/serverclient"

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
	if len(args) == 2 && args[0] == "-package-smoke" {
		return packageSmoke(args[1], embedded, os.Stderr)
	}
	if len(args) == 3 && args[0] == "-native" {
		return runNative(args[1], args[2], embedded, os.Stderr)
	}
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
		_, _ = fmt.Fprintf(stderr, "thewarroom: unknown arguments %q\nusage: thewarroom [-probe | -version | -backups | -restore <set> | -native <https-origin> <league-id>]\n", args)
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

// packageSmoke initializes the installed WebView without databases, credentials or network
// workers. The caller supplies a disposable existing directory and kills the ready process.
func packageSmoke(directory string, embedded Embedded, stderr io.Writer) int {
	info, err := os.Stat(directory)
	if err != nil || !info.IsDir() || !filepath.IsAbs(directory) {
		_, _ = fmt.Fprintln(stderr, "package smoke requires an existing absolute directory")
		return 2
	}
	lock, err := acquireInstanceLock(filepath.Join(directory, "smoke.db"))
	if err != nil {
		_, _ = fmt.Fprintln(stderr, err)
		return 1
	}
	defer releaseInstanceLock(lock)
	err = wails.Run(&options.App{
		Title: "The War Room — installation check", Width: 640, Height: 480,
		AssetServer: &assetserver.Options{Assets: embedded.Assets},
		OnDomReady: func(_ context.Context) {
			if writeErr := os.WriteFile(filepath.Join(directory, "ready"), []byte("webview ready\n"), 0o600); writeErr != nil {
				log.Printf("package smoke ready: %v", writeErr)
			}
		},
	})
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "package smoke: %v\n", err)
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

// helperCommands is every catalog command an AI helper's token may carry.
func helperCommands() []string {
	var out []string
	for _, command := range commissionerapp.NativeCommands() {
		if accounts.DelegableCommand(command.ID) {
			out = append(out, command.ID)
		}
	}
	return out
}

// runNative never constructs the local App: no MFL key, databases, refresh workers
// or authoritative local mutations are available in a hosted window.
func runNative(address, league string, embedded Embedded, stderr io.Writer) int {
	desktop, err := serverclient.NewDesktop(address, league, nil)
	if err != nil {
		_, _ = fmt.Fprintln(stderr, "native startup requires an HTTPS origin and native league ID")
		return 2
	}
	defer desktop.Close()
	desktop.SetHelperCommands(helperCommands())
	err = wails.Run(&options.App{
		Title: "The War Room — Native league", Width: 960, Height: 800,
		BackgroundColour: &options.RGBA{R: 9, G: 10, B: 12, A: 255},
		AssetServer:      &assetserver.Options{Assets: embedded.Assets, Handler: desktop.Handler()},
	})
	if err != nil {
		_, _ = fmt.Fprintln(stderr, "native window could not start")
		return 1
	}
	return 0
}
