package composition

import (
	"context"
	"fmt"
	"io"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"time"

	catalog "github.com/secureprospective/TheWarRoom/internal/commandcatalog"

	commissionerapp "github.com/secureprospective/TheWarRoom/internal/app/commissioner"
	platformapp "github.com/secureprospective/TheWarRoom/internal/app/platform"

	"github.com/secureprospective/TheWarRoom/internal/serverclient"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	wailsruntime "github.com/wailsapp/wails/v2/pkg/runtime"
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
	return entryPoints(embedded).Run(args, os.Stdout, os.Stderr)
}

func entryPoints(embedded Embedded) platformapp.EntryPoints {
	return platformapp.EntryPoints{
		Smoke:   func(directory string) int { return packageSmoke(directory, embedded, os.Stderr) },
		Native:  func(address, league string) int { return runNative(address, league, embedded, os.Stderr) },
		Desktop: func() int { return runDesktop(embedded) }, Probe: probe, Backups: listBackups, Restore: restoreBackup, Label: buildLabel(),
	}
}

func runDesktop(embedded Embedded) int {
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
	return entryPoints(Embedded{}).RunFlag(args, stdout, stderr)
}

func probe() int {
	app := NewApp()
	return platformapp.Probe(app.startup, app.shutdown, func() error { return app.core.StartupErr })
}

// packageSmoke initializes the installed WebView without databases, credentials or network
// workers. The caller supplies a disposable existing directory and kills the ready process.
func packageSmoke(directory string, embedded Embedded, stderr io.Writer) int {
	return platformapp.SmokeWindow(directory, stderr, func() error {
		return wails.Run(&options.App{
			Title: "The War Room — installation check", Width: 640, Height: 480,
			AssetServer: &assetserver.Options{Assets: embedded.Assets},
			OnDomReady: func(_ context.Context) {
				if err := os.WriteFile(filepath.Join(directory, "ready"), []byte("webview ready\n"), 0o600); err != nil {
					log.Printf("package smoke ready: %v", err)
				}
			},
		})
	})
}

// bindings is the runtime surface also inspected by the binding manifest test.
func bindings(app *App) []interface{} {
	apis := newCommandAPIs(app)
	registry, err := catalog.NewRegistry(apis.registryInputs(app))
	if err != nil {
		panic(fmt.Sprintf("invalid command registry: %v", err))
	}
	app.commands = registry
	return generatedWailsBindings(apis)
}

// helperCommands is every catalog command an AI helper's token may carry.
func helperCommands() []string {
	return platformapp.DelegableCommands(commissionerapp.NativeCommands())
}

// runNative constructs only the hosted client and WebView, never the local App.
func runNative(address, league string, embedded Embedded, stderr io.Writer) int {
	var desktop *serverclient.Desktop
	var window context.Context
	return (platformapp.NativeWindow{
		Open: func() (platformapp.NativeDesktop, error) {
			var err error
			desktop, err = serverclient.NewDesktop(address, league, nil)
			return desktop, err //nolint:wrapcheck // NativeWindow preserves the public startup refusal.
		},
		Commands: helperCommands, Directory: os.UserConfigDir, Lock: nativeUpdateLock,
		Quit: func() { wailsruntime.Quit(window) }, Version: version, Practice: func() string { return os.Getenv("TWR_PRACTICE_UPDATE") },
		Run: func() error {
			return wails.Run(&options.App{
				Title: "The War Room — Native league", Width: 960, Height: 800,
				OnStartup:        func(ctx context.Context) { window = ctx },
				BackgroundColour: &options.RGBA{R: 9, G: 10, B: 12, A: 255},
				AssetServer:      &assetserver.Options{Assets: embedded.Assets, Handler: desktop.Handler()},
			})
		},
	}).RunWindow(stderr)
}

func nativeUpdateLock(directory string) (*os.File, error) {
	return platformapp.NativeUpdateLock(directory, acquireInstanceLock, time.Sleep) //nolint:wrapcheck // Preserve update-lock refusal text.
}

func backupCLI() platformapp.BackupCLI {
	return platformapp.BackupCLI{Directory: configDir, Backups: newDatabaseBackups, DevBuild: isDevBuild()}
}
func listBackups(stdout, stderr io.Writer) int { return backupCLI().List(stdout, stderr) }
func restoreBackup(set string, stdout, stderr io.Writer) int {
	return backupCLI().Restore(set, stdout, stderr)
}
