// Package composition is the backend's composition root: it wires the stores and routes each feature's
// IPC calls. Business logic lives in the engine, store and transaction packages, never here.
package composition

import (
	"context"
	"log"
	"os"
	"strconv"
	"sync"
	"time"

	platformapp "github.com/secureprospective/TheWarRoom/internal/app/platform"

	archiveapp "github.com/secureprospective/TheWarRoom/internal/app/archive"
	contractsapp "github.com/secureprospective/TheWarRoom/internal/app/contracts"
	controlapp "github.com/secureprospective/TheWarRoom/internal/app/control"
	"github.com/secureprospective/TheWarRoom/internal/app/core"
	marketapp "github.com/secureprospective/TheWarRoom/internal/app/market"
	nowapp "github.com/secureprospective/TheWarRoom/internal/app/now"
	raceapp "github.com/secureprospective/TheWarRoom/internal/app/race"
	researchapp "github.com/secureprospective/TheWarRoom/internal/app/research"
	sourcesapp "github.com/secureprospective/TheWarRoom/internal/app/sources"
	catalog "github.com/secureprospective/TheWarRoom/internal/commandcatalog"

	"github.com/secureprospective/TheWarRoom/internal/accounts"
	"github.com/secureprospective/TheWarRoom/internal/archive"
	"github.com/secureprospective/TheWarRoom/internal/db"
	"github.com/secureprospective/TheWarRoom/internal/ingestion"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/players"
	"github.com/secureprospective/TheWarRoom/internal/lab"
	"github.com/secureprospective/TheWarRoom/internal/measures"
	"github.com/secureprospective/TheWarRoom/internal/mfl"
	"github.com/secureprospective/TheWarRoom/internal/mflkey"
	"github.com/secureprospective/TheWarRoom/internal/normalize"
	accountstore "github.com/secureprospective/TheWarRoom/internal/store/accounts"
	"github.com/secureprospective/TheWarRoom/internal/store/history"
	"github.com/secureprospective/TheWarRoom/internal/store/moves"
	"github.com/secureprospective/TheWarRoom/internal/store/params"
	"github.com/secureprospective/TheWarRoom/internal/store/personal"
	"github.com/secureprospective/TheWarRoom/internal/store/rulebook"
	"github.com/secureprospective/TheWarRoom/internal/store/state"
	"github.com/secureprospective/TheWarRoom/internal/transactions"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// App is the Wails root and the backend's composition root: it wires the stores and routes
// IPC calls. Business logic lives in the engine, store and transaction packages, never here.
type App struct {
	commands            *catalog.Registry
	core                *core.State
	tradeBuilderTimeout time.Duration
	archiveCache        archiveapp.Snapshot
	controlHeld         *controlapp.Held
	researchHeld        *researchapp.Held
	raceHeld            *raceapp.Held
	now                 nowSchedules
	nowHeld             *nowapp.Held
	sourcesHeld         *sourcesapp.Held
	marketHeld          *marketapp.Held
	seatStore           *accountstore.Store
	seat                *accounts.LocalSeat
}

// attachSeat keeps the local sign-in beside the ledger, so a matched backup carries it.
func (a *App) attachSeat(hist *history.Store) {
	a.seatStore = accountstore.New(a.core.HistPools)
	a.seat = accounts.NewLocalSeat(accounts.New(a.seatStore), a.seatStore)
	hist.SetVerifiedAliases(a.seat.VerifiedAliases)
}

// directory returns the cached players Lookup for the season held, fetching it on first use.
func (a *App) directory(ctx context.Context) (normalize.Lookup, error) {
	a.core.LookupMu.Lock()
	defer a.core.LookupMu.Unlock()
	return sourcesapp.Directory(ctx, a.core.Lookup, a.core.HasLookup, //nolint:wrapcheck // Preserve directory fetch/normalize errors.
		func(ctx context.Context) ([]players.RawPlayer, error) {
			return players.Fetch(ctx, a.core.MFLClient, strconv.Itoa(a.core.Season), ingestion.LeagueID)
		},
		func(lookup normalize.Lookup, at time.Time) {
			a.core.Lookup, a.core.HasLookup, a.core.LookupAt = lookup, true, at
		})
}

// NewApp is cheap; resources are acquired in startup.
func NewApp() *App {
	var desktop core.Runtime = desktopRuntime{}
	a := &App{tradeBuilderTimeout: m2Timeout, controlHeld: &controlapp.Held{}, researchHeld: &researchapp.Held{}, marketHeld: &marketapp.Held{}, nowHeld: &nowapp.Held{}, raceHeld: &raceapp.Held{}, sourcesHeld: &sourcesapp.Held{}, core: &core.State{Started: make(chan struct{}),
		BuildStamp:    buildLabel() + " " + buildDate,
		LabData:       sync.OnceValues(lab.Load),
		MovesWake:     make(chan struct{}, 1),
		MovesChanged:  func(ctx context.Context) { desktop.EventsEmit(ctx, "target:moves") },
		OpenURL:       desktop.BrowserOpenURL,
		MovesNow:      time.Now,
		MovesTimer:    newMovesTimer,
		MovesLog:      func(line string) { log.Print(line) },
		FallbackLog:   func(line string) { log.Print(line) },
		MovesUnknown:  make(map[string]bool),
		ClockChanged:  func(ctx context.Context) { desktop.EventsEmit(ctx, "target:clock") },
		SeasonChanged: func(ctx context.Context) { desktop.EventsEmit(ctx, "target:season") }},
	}
	a.configureArchive()
	a.core.CaptureRace = a.raceSnapshot
	a.core.RefreshRaceFeeds = a.refreshRace
	a.core.RefreshMarketFeeds = a.refreshMarketFeeds
	a.core.RefreshTransactions = a.refreshSeasonTransactions
	a.wireSources()
	return a
}

// ready waits for startup to finish and returns its failure, if any. On Linux, Wails runs startup
// alongside the page load, so every IPC method calls ready before it reads a store.
func (a *App) ready() error {
	<-a.core.Started
	return a.core.StartupErr
}

// startup is the Wails OnStartup hook. OnStartup cannot return an error, so a failure is
// kept in startupErr, logged, and shown by the shell's banner through AppInfo.
func (a *App) startup(ctx context.Context) {
	var refreshed bool
	defer func() { a.finishStartup(ctx, refreshed) }()
	a.core.Ctx = ctx
	a.core.WorkCtx, a.core.WorkCancel = context.WithCancel(ctx)
	defer a.settleStartupFailure()
	refreshed, a.core.StartupErr = (platformapp.StartupWorkflow{
		Directory: configDir, SetDirectory: func(dir string) { a.core.Paths.DataDir = dir }, Logging: setupLogging,
		Databases: a.openDatabases,
		Client: func() error {
			client, err := mfl.New("api", 2, mfl.WithTransport(a.core.Fetches),
				mfl.WithKeySource(func(ctx context.Context) (mflkey.Key, error) {
					if a.core.KeyStore == nil {
						return "", mflkey.ErrUnavailable
					}
					return a.core.KeyStore.Get(ctx)
				}))
			if err != nil {
				return err //nolint:wrapcheck // The workflow adds the original startup client context.
			}
			a.core.MFLClient = client
			return nil
		}, Photos: a.startHeadshots, Floor: a.initStoreFloor,
		Keys: func(ctx context.Context) {
			a.core.KeyStore = mflkey.New(ingestion.LeagueID, strconv.Itoa(a.core.Season))
			a.lookupMyFranchise(ctx)
		},
		LaunchDue: func(due bool) { a.core.LaunchRefreshDue = due }, Season: func() int { return a.core.Season }, Label: buildLabel(), Log: func(line string) { log.Print(line) },
	}).Run(ctx)
}

func (a *App) settleStartupFailure() {
	a.core.StartupErr = platformapp.StartupFailure(a.core.StartupErr, func(line string) { log.Print(line) })
}

func (a *App) finishStartup(ctx context.Context, refreshed bool) {
	platformapp.FinishStartup(a.core.StartupErr, refreshed, func() { close(a.core.Started) }, func() {
		defer a.sourceHealthChanged(ctx)
		a.core.RefreshMu.Lock()
		defer a.core.RefreshMu.Unlock()
		a.revalidateReadyRosterMoves(ctx)
	})
}

// domReady is the Wails OnDomReady hook: the window is up, so the launch refresh and the signals
// load run now, off the startup path. A windowless -probe never reaches it.
func (a *App) domReady(ctx context.Context) {
	a.refreshInBackground(ctx)
	a.startDatabaseChecks()      //nolint:contextcheck // The app lifetime cancels and joins database checks on shutdown.
	a.compileFactsInBackground() //nolint:contextcheck // The app lifetime cancels and joins facts work on shutdown.
	a.startWorker(func() { a.publishAssetModel(ctx); a.sourceHealthChanged(ctx) })
}

// openDatabases takes the instance lock and opens the three databases. Every fetch after this goes
// through the history archive. It returns the league database's path for the startup log.
func (a *App) openDatabases(ctx context.Context, dir string) (string, *history.Store, error) {
	//nolint:wrapcheck // Preserve startup step and store refusal errors.
	return (platformapp.DatabaseOpening{
		Lock: func(path string) error {
			lock, err := acquireInstanceLock(path)
			if err == nil {
				a.core.LockFile = lock
			}
			return err
		},
		Backups: func(dir string) error { backups, err := newDatabaseBackups(dir); a.core.Backups = backups; return err },
		Recover: func(ctx context.Context) error { return a.core.Backups.Recover(ctx) },
		BeforeMigrations: func(ctx context.Context, now time.Time) error {
			_, err := a.core.Backups.BeforeMigrations(ctx, now)
			return err
		},
		Main: func(ctx context.Context, path string) error {
			pools, err := db.Open(ctx, path)
			a.core.Pools = pools
			return err
		},
		Registry: measures.Embedded,
		History: func(ctx context.Context, path string) error {
			pools, err := db.Open(ctx, path, history.SchemaVersion())
			a.core.HistPools = pools
			return err
		},
		WhatIf: func(ctx context.Context, path string) error {
			pools, err := db.Open(ctx, path, state.SchemaVersion())
			a.core.WhatifPools = pools
			return err
		},
		Attach: func(reg *measures.Registry) *history.Store {
			hist := history.New(a.core.HistPools, reg)
			a.attachSeat(hist)
			return hist
		},
		Archive: func(hist *history.Store) {
			a.core.Fetches = &archive.Transport{Base: a.core.Network, Sink: hist, Judge: ingestion.MFLRefusal}
		}, DevBuild: isDevBuild(),
	}).Open(ctx, dir)
}

// startupBudget bounds store-floor init. An empty mirror is filled from MFL on this thread, so an
// unreachable MFL would otherwise hang the window black forever.
const startupBudget = 2 * time.Minute

// initStoreFloor brings up the stores and the transaction coordinator, logging each step's
// time so a slow or failed launch shows where it stopped. History comes first, because every
// fetch after it is archived there. An empty mirror is refreshed from MFL here, and refreshed
// reports that; otherwise the launch refresh runs in the background. The what-if league seeds
// from the mirror on a fresh what-if database. Fields are assigned only once every step
// succeeds, so a failed startup leaves no half-built store behind ready.
func (a *App) initStoreFloor(parent context.Context, hist *history.Store) (refreshed bool, err error) {
	ctx, cancel := context.WithTimeout(parent, startupBudget)
	defer cancel()

	moveStore := moves.New(a.core.Pools)
	pstore := params.New(a.core.Pools)
	rb := rulebook.New(a.core.Pools)
	mirror := state.NewMirror(a.core.Pools, rb)
	var whatif *state.Store
	var coord *transactions.Coordinator
	//nolint:wrapcheck // The service retains the original step names and failures.
	return (platformapp.StoreFloor{
		History: hist.Initialize, Seat: a.seatStore.Initialize, Params: pstore.Initialize, Moves: moveStore.Initialize,
		Personal: personal.New(a.core.Pools).Initialize,
		Rules:    func(c context.Context) error { return rb.Initialize(c, discoverSource{app: a}) },
		Mirror:   mirror.Initialize, MirrorSeason: mirror.Season,
		Refresh: func(c context.Context) error {
			a.core.RefreshMu.Lock()
			defer a.core.RefreshMu.Unlock()
			_, err := a.refreshLeagueLocked(c, rb, mirror)
			return err
		},
		WhatIf: func(c context.Context) error {
			a.core.Season = mirror.Season()
			whatif = state.New(a.core.WhatifPools, ingestion.LeagueID, a.core.Season, rb)
			return whatif.Initialize(c, mirror)
		},
		Coordinator: func(context.Context) error {
			var err error
			coord, err = transactions.New(whatif.Writer(), &contractsapp.RosterPolicy{Rulebook: rb, Directory: a.directory},
				func(c context.Context) (transactions.Directory, error) { return a.directory(c) })
			return err
		},
		MigrationFindings: a.logMigrationFindings,
		Publish: func() {
			a.core.Params, a.core.Rulebook, a.core.League, a.core.Whatif, a.core.Coordinator, a.core.History = pstore, rb, mirror, whatif, coord, hist
			a.core.Moves = moveStore
			hist.SetLineupEnvironment(appLineupEnv{app: a})
			hist.SetBidEnvironment(appBidEnv{app: a})
		}, DraftReports: a.refreshDraftReportsFeeds, SeedClock: a.seedClockSchedules, Log: func(line string) { log.Print(line) },
	}).Run(ctx)
}

func (a *App) logMigrationFindings() {
	files := []platformapp.DatabaseFile{}
	for _, file := range a.databaseFiles() {
		files = append(files, platformapp.DatabaseFile{Name: file.name, Pools: file.pools})
	}
	platformapp.LogMigrationFindings(files, func(line string) { log.Print(line) })
}

func (a *App) shutdown(_ context.Context) {
	a.core.ShutdownOnce.Do(func() {
		a.workerGroup().Stop(&a.core.WeekCancel, &a.core.WorkCancel)
		platformapp.CloseDatabases([]*db.Pools{a.core.Pools, a.core.HistPools, a.core.WhatifPools})
		releaseInstanceLock(a.core.LockFile)
	})
}

func (a *App) startWorker(work func()) { a.workerGroup().Start(work) }

func (a *App) workerGroup() platformapp.WorkerGroup {
	return platformapp.WorkerGroup{Mutex: &a.core.WorkerMu, Stopping: &a.core.Stopping, Workers: &a.core.WeekWorkers}
}

type desktopRuntime struct{}

func (desktopRuntime) EventsEmit(ctx context.Context, event string) { runtime.EventsEmit(ctx, event) }
func (desktopRuntime) BrowserOpenURL(ctx context.Context, target string) {
	runtime.BrowserOpenURL(ctx, target)
}

func configDir() (string, error) { return platformapp.ConfigDir() } //nolint:wrapcheck // Preserve startup/config directory errors.
func isDevBuild() bool           { return platformapp.IsDevBuild(version) }
func dbFileName(base string, devBuild bool) string {
	return platformapp.DatabaseFileName(base, devBuild)
}

// Build stamp, set at link time by -ldflags -X from git describe (see the Makefile). The
// defaults make an un-stamped binary read as a DEV build, never a release.
//
//nolint:gochecknoglobals // ldflags -X requires package-level string vars.
var (
	version   = "dev"
	commit    = ""
	buildDate = ""
)

func buildLabel() string { return platformapp.BuildLabel(version, commit) }

func setupLogging(dir string) error { return platformapp.SetupLogging(dir) } //nolint:wrapcheck // Logging already provides its resource context.

func acquireInstanceLock(path string) (*os.File, error) {
	return platformapp.AcquireInstanceLock(path) //nolint:wrapcheck // Preserve instance-lock refusal text.
}
func releaseInstanceLock(file *os.File) { platformapp.ReleaseInstanceLock(file) }
