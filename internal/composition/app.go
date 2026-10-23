// Package composition is the backend's composition root: it wires the stores and routes each feature's
// IPC calls. Business logic lives in the engine, store and transaction packages, never here.
package composition

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"time"

	archiveapp "github.com/secureprospective/TheWarRoom/internal/app/archive"
	contractsapp "github.com/secureprospective/TheWarRoom/internal/app/contracts"
	controlapp "github.com/secureprospective/TheWarRoom/internal/app/control"
	"github.com/secureprospective/TheWarRoom/internal/app/core"
	marketapp "github.com/secureprospective/TheWarRoom/internal/app/market"
	nowapp "github.com/secureprospective/TheWarRoom/internal/app/now"
	raceapp "github.com/secureprospective/TheWarRoom/internal/app/race"
	researchapp "github.com/secureprospective/TheWarRoom/internal/app/research"
	sourcesapp "github.com/secureprospective/TheWarRoom/internal/app/sources"

	"github.com/secureprospective/TheWarRoom/internal/archive"
	"github.com/secureprospective/TheWarRoom/internal/db"
	"github.com/secureprospective/TheWarRoom/internal/ingestion"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/players"
	"github.com/secureprospective/TheWarRoom/internal/lab"
	"github.com/secureprospective/TheWarRoom/internal/measures"
	"github.com/secureprospective/TheWarRoom/internal/mfl"
	"github.com/secureprospective/TheWarRoom/internal/mflkey"
	"github.com/secureprospective/TheWarRoom/internal/normalize"
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
	core         *core.State
	archiveCache archiveapp.Snapshot
	controlHeld  *controlapp.Held
	researchHeld *researchapp.Held
	raceHeld     *raceapp.Held
	now          nowSchedules
	nowHeld      *nowapp.Held
	sourcesHeld  *sourcesapp.Held
	marketHeld   *marketapp.Held
}

// directory returns the cached players Lookup for the season held, fetching it on first use.
func (a *App) directory(ctx context.Context) (normalize.Lookup, error) {
	a.core.LookupMu.Lock()
	defer a.core.LookupMu.Unlock()
	if a.core.HasLookup {
		return a.core.Lookup, nil
	}
	raws, err := players.Fetch(ctx, a.core.MFLClient, strconv.Itoa(a.core.Season), ingestion.LeagueID)
	if err != nil {
		return normalize.Lookup{}, fmt.Errorf("app: fetch players db: %w", err)
	}
	lk, err := normalize.NewLookup(raws)
	if err != nil {
		return normalize.Lookup{}, fmt.Errorf("app: build players lookup: %w", err)
	}
	a.core.Lookup, a.core.HasLookup, a.core.LookupAt = lk, true, time.Now().UTC()
	return lk, nil
}

// NewApp is cheap; resources are acquired in startup.
func NewApp() *App {
	var desktop core.Runtime = desktopRuntime{}
	a := &App{controlHeld: &controlapp.Held{}, researchHeld: &researchapp.Held{}, marketHeld: &marketapp.Held{}, nowHeld: &nowapp.Held{}, raceHeld: &raceapp.Held{}, sourcesHeld: &sourcesapp.Held{}, core: &core.State{Started: make(chan struct{}),
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
	defer close(a.core.Started)
	a.core.Ctx = ctx
	a.core.WorkCtx, a.core.WorkCancel = context.WithCancel(ctx)
	defer func() {
		if a.core.StartupErr != nil {
			log.Printf("the war room: startup failed: %v", a.core.StartupErr)
		}
	}()

	// Logging starts first so a lock or open failure below reaches the disk log. A logging
	// failure is not fatal.
	dir, err := configDir()
	if err != nil {
		a.core.StartupErr = fmt.Errorf("startup: resolve config dir: %w", err)
		return
	}
	a.core.Paths.DataDir = dir
	if lerr := setupLogging(dir); lerr != nil {
		log.Printf("the war room: WARNING disk logging unavailable: %v", lerr)
	}

	path, hist, err := a.openDatabases(ctx, dir)
	if err != nil {
		a.core.StartupErr = fmt.Errorf("startup: %w", err)
		return
	}

	client, err := mfl.New("api", 2, mfl.WithTransport(a.core.Fetches),
		mfl.WithKeySource(func(ctx context.Context) (mflkey.Key, error) {
			if a.core.KeyStore == nil {
				return "", mflkey.ErrUnavailable
			}
			return a.core.KeyStore.Get(ctx)
		}))
	if err != nil {
		a.core.StartupErr = fmt.Errorf("startup: mfl client: %w", err)
		return
	}
	a.core.MFLClient = client
	a.startHeadshots()

	log.Printf("the war room: starting %s, db %s", buildLabel(), path)
	began := time.Now()
	refreshed, err := a.initStoreFloor(ctx, hist)
	if err != nil {
		a.core.StartupErr = err
		return
	}
	log.Printf("the war room: season %d ready in %s", a.core.Season, time.Since(began).Round(time.Millisecond))
	a.core.KeyStore = mflkey.New(ingestion.LeagueID, strconv.Itoa(a.core.Season))
	a.lookupMyFranchise(ctx)
	a.core.LaunchRefreshDue = !refreshed
}

// domReady is the Wails OnDomReady hook: the window is up, so the launch refresh and the signals
// load run now, off the startup path. A windowless -probe never reaches it.
func (a *App) domReady(ctx context.Context) {
	a.refreshInBackground(ctx)
	a.startDatabaseChecks()      //nolint:contextcheck // The app lifetime cancels and joins database checks on shutdown.
	a.compileFactsInBackground() //nolint:contextcheck // The app lifetime cancels and joins facts work on shutdown.
}

// openDatabases takes the instance lock and opens the three databases. Every fetch after this goes
// through the history archive. It returns the league database's path for the startup log.
func (a *App) openDatabases(ctx context.Context, dir string) (string, *history.Store, error) {
	path := filepath.Join(dir, dbFileName("thewarroom", isDevBuild()))
	// One running copy per database: two processes writing one ledger is the hazard.
	lock, err := acquireInstanceLock(path)
	if err != nil {
		return "", nil, err
	}
	a.core.LockFile = lock
	if a.core.Backups, err = newDatabaseBackups(dir); err != nil {
		return "", nil, err
	}
	if err := a.core.Backups.Recover(ctx); err != nil {
		return "", nil, fmt.Errorf("recover databases: %w", err)
	}
	if _, err := a.core.Backups.BeforeMigrations(ctx, time.Now().UTC()); err != nil {
		return "", nil, fmt.Errorf("capture before migrations: %w", err)
	}
	if a.core.Pools, err = db.Open(ctx, path); err != nil {
		return "", nil, fmt.Errorf("open database: %w", err)
	}
	reg, err := measures.Embedded()
	if err != nil {
		return "", nil, err //nolint:wrapcheck // the registry's errors name the CSV line
	}
	if a.core.HistPools, err = db.Open(ctx, filepath.Join(dir, dbFileName("history", isDevBuild())), history.SchemaVersion()); err != nil {
		return "", nil, fmt.Errorf("open history database: %w", err)
	}
	if a.core.WhatifPools, err = db.Open(ctx, filepath.Join(dir, dbFileName("whatif", isDevBuild())), state.SchemaVersion()); err != nil {
		return "", nil, fmt.Errorf("open what-if database: %w", err)
	}
	hist := history.New(a.core.HistPools, reg)
	a.core.Fetches = &archive.Transport{Sink: hist, Judge: ingestion.MFLRefusal}
	return path, hist, nil
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
	steps := []struct {
		name string
		run  func(context.Context) error
	}{
		{"history", hist.Initialize},
		{"params", pstore.Initialize},
		{"moves", moveStore.Initialize},
		{"personal lists", personal.New(a.core.Pools).Initialize},
		{"rulebook", func(c context.Context) error { return rb.Initialize(c, discoverSource{app: a}) }},
		{"league mirror", mirror.Initialize},
		{"first MFL refresh", func(c context.Context) error {
			if mirror.Season() != 0 {
				return nil
			}
			refreshed = true
			_, err := a.refreshLeague(c, rb, mirror)
			return err
		}},
		{"what-if league", func(c context.Context) error {
			a.core.Season = mirror.Season()
			whatif = state.New(a.core.WhatifPools, ingestion.LeagueID, a.core.Season, rb)
			return whatif.Initialize(c, mirror)
		}},
		// The coordinator is the only holder of the what-if Writer.
		{"transaction coordinator", func(context.Context) error {
			var err error
			coord, err = transactions.New(whatif.Writer(), &contractsapp.RosterPolicy{Rulebook: rb, Directory: a.directory},
				func(c context.Context) (transactions.Directory, error) { return a.directory(c) })
			return err //nolint:wrapcheck // wrapped below with the step name
		}},
	}
	for _, s := range steps {
		began := time.Now()
		if err := s.run(ctx); err != nil {
			return false, fmt.Errorf("startup: initialize %s: %w", s.name, err)
		}
		log.Printf("the war room: %s ready in %s", s.name, time.Since(began).Round(time.Millisecond))
	}
	a.logMigrationFindings()
	a.core.Params, a.core.Rulebook, a.core.League, a.core.Whatif, a.core.Coordinator, a.core.History = pstore, rb, mirror, whatif, coord, hist
	a.core.Moves = moveStore
	hist.SetLineupEnvironment(appLineupEnv{app: a})
	hist.SetBidEnvironment(appBidEnv{app: a})
	a.seedClockSchedules(ctx)
	return refreshed, nil
}

func (a *App) logMigrationFindings() {
	for _, file := range []struct {
		name  string
		pools *db.Pools
	}{
		{"thewarroom", a.core.Pools}, {"history", a.core.HistPools}, {"whatif", a.core.WhatifPools},
	} {
		if file.pools != nil {
			findings := file.pools.MigrationFindings()
			log.Printf("the war room: %s migration checks: before=%+v after=%+v", file.name, findings.Before, findings.After)
		}
	}
}

// shutdown releases the database and the instance lock.
func (a *App) shutdown(_ context.Context) {
	a.core.ShutdownOnce.Do(func() {
		a.core.WorkerMu.Lock()
		a.core.Stopping = true
		if a.core.WeekCancel != nil {
			a.core.WeekCancel()
		}
		if a.core.WorkCancel != nil {
			a.core.WorkCancel()
		}
		a.core.WorkerMu.Unlock()
		a.core.WeekWorkers.Wait()
		for _, p := range []*db.Pools{a.core.Pools, a.core.HistPools, a.core.WhatifPools} {
			if p != nil {
				_ = p.Close()
			}
		}
		releaseInstanceLock(a.core.LockFile)
	})
}

func (a *App) startWorker(work func()) {
	a.core.WorkerMu.Lock()
	defer a.core.WorkerMu.Unlock()
	if a.core.Stopping {
		return
	}
	a.core.WeekWorkers.Add(1)
	go func() {
		defer a.core.WeekWorkers.Done()
		work()
	}()
}

// configDir returns the app's data directory (~/.config/TheWarRoom on Linux), creating it
// owner-only. A dir left by an older build (0750) is tightened, since it holds the ledger.
func configDir() (string, error) {
	cfg, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("user config dir: %w", err)
	}
	dir := filepath.Join(cfg, "TheWarRoom")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "", fmt.Errorf("create data dir %q: %w", dir, err)
	}
	if err := os.Chmod(dir, 0o700); err != nil { //nolint:gosec // a directory needs its x bit; 0700 is owner-only
		return "", fmt.Errorf("restrict data dir %q: %w", dir, err)
	}
	return dir, nil
}

// isDevBuild reports an un-stamped build (plain go build or wails dev).
func isDevBuild() bool { return version == "dev" }

// dbFileName gives dev builds their own databases, so development can never migrate or
// corrupt the real league ledger or its history.
func dbFileName(base string, devBuild bool) string {
	if devBuild {
		return base + "-dev.db"
	}
	return base + ".db"
}

type desktopRuntime struct{}

func (desktopRuntime) EventsEmit(ctx context.Context, event string) { runtime.EventsEmit(ctx, event) }
func (desktopRuntime) BrowserOpenURL(ctx context.Context, target string) {
	runtime.BrowserOpenURL(ctx, target)
}
