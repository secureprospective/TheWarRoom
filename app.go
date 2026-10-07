package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/archive"
	"github.com/secureprospective/TheWarRoom/internal/db"
	"github.com/secureprospective/TheWarRoom/internal/envelope"
	"github.com/secureprospective/TheWarRoom/internal/ingestion"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/players"
	"github.com/secureprospective/TheWarRoom/internal/leagueweek"
	"github.com/secureprospective/TheWarRoom/internal/measures"
	"github.com/secureprospective/TheWarRoom/internal/mfl"
	"github.com/secureprospective/TheWarRoom/internal/mflkey"
	"github.com/secureprospective/TheWarRoom/internal/normalize"
	"github.com/secureprospective/TheWarRoom/internal/snapshot"
	"github.com/secureprospective/TheWarRoom/internal/store/history"
	"github.com/secureprospective/TheWarRoom/internal/store/params"
	"github.com/secureprospective/TheWarRoom/internal/store/rulebook"
	"github.com/secureprospective/TheWarRoom/internal/store/state"
	"github.com/secureprospective/TheWarRoom/internal/transactions"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// App is the Wails root and the backend's composition root: it wires the stores and routes
// IPC calls. Business logic lives in the engine, store and transaction packages, never here.
type App struct {
	keyMu         sync.Mutex
	keyStore      *mflkey.Store
	keyVerifiedAt string
	//nolint:containedctx // Wails supplies an app-lifetime context; bindings derive bounded contexts.
	ctx         context.Context
	pools       *db.Pools // thewarroom.db: the MFL mirror and app settings
	histPools   *db.Pools // history.db: everything that cannot be rebuilt
	whatifPools *db.Pools // whatif.db: moves made in the app, apart from MFL truth (R2)
	params      *params.Store
	rulebook    *rulebook.Store
	league      *state.Mirror // the league as MFL states it; every score surface reads it
	whatif      *state.Store  // the what-if league the transaction screens work on
	history     *history.Store
	coordinator *transactions.Coordinator // the only holder of the what-if Writer
	mflClient   *mfl.Client               // shared, so rate limit and host discovery are process-wide
	fetches     *archive.Transport        // every outbound HTTP request goes through it
	season      int                       // mirror season at startup; rollover is picked up next launch
	startupErr  error                     // startup failure; shown in the shell through AppInfo
	started     chan struct{}             // closed when startup returns; read the fields above through ready
	lockFile    *os.File                  // single-instance lock, held until shutdown

	// The players directory is fetched at most once per process (MFL allows the endpoint once a
	// day). Wails runs IPC calls concurrently, hence the mutex.
	lookupMu  sync.Mutex
	lookup    normalize.Lookup
	hasLookup bool
	lookupAt  time.Time

	// Drafts check against the last good TargetSnapshot build, so a draft never refetches.
	targetSnapshotMu  sync.Mutex
	targetSnapshot    snapshot.Snapshot
	hasTargetSnapshot bool
	targetMoveLog     *envelope.MemoryLog // in memory until ring 1 persists the audit log

	weekMu           sync.Mutex
	week             leagueweek.Week
	weekFetchedAt    time.Time
	weekRefreshError string
	clockChanged     func(context.Context)
	weekCancel       context.CancelFunc
	weekWorkers      sync.WaitGroup
	weekStart        sync.Once // domReady fires again on a webview reload; one worker only

	refreshMu        sync.Mutex // one MFL refresh at a time
	launchRefreshDue bool       // startup did not refresh, so domReady does

	crosswalkMu sync.Mutex
	crosswalk   CrosswalkReport // the latest crosswalk load's report, held since launch

	loadingSignals sync.Mutex    // one signals load at a time
	signalsMu      sync.Mutex    // guards signals
	signals        SignalsReport // the latest signals load, held since launch
}

// directory returns the cached players Lookup for the season held, fetching it on first use.
func (a *App) directory(ctx context.Context) (normalize.Lookup, error) {
	a.lookupMu.Lock()
	defer a.lookupMu.Unlock()
	if a.hasLookup {
		return a.lookup, nil
	}
	raws, err := players.Fetch(ctx, a.mflClient, strconv.Itoa(a.season), ingestion.LeagueID)
	if err != nil {
		return normalize.Lookup{}, fmt.Errorf("app: fetch players db: %w", err)
	}
	lk, err := normalize.NewLookup(raws)
	if err != nil {
		return normalize.Lookup{}, fmt.Errorf("app: build players lookup: %w", err)
	}
	a.lookup, a.hasLookup, a.lookupAt = lk, true, time.Now().UTC()
	return lk, nil
}

// NewApp is cheap; resources are acquired in startup.
func NewApp() *App {
	return &App{
		started:       make(chan struct{}),
		targetMoveLog: envelope.NewMemoryLog(),
		clockChanged:  func(ctx context.Context) { runtime.EventsEmit(ctx, "target:clock") },
	}
}

// ready waits for startup to finish and returns its failure, if any. On Linux, Wails runs startup
// alongside the page load, so every IPC method calls ready before it reads a store.
func (a *App) ready() error {
	<-a.started
	return a.startupErr
}

// startup is the Wails OnStartup hook. OnStartup cannot return an error, so a failure is
// kept in startupErr, logged, and shown by the shell's banner through AppInfo.
func (a *App) startup(ctx context.Context) {
	defer close(a.started)
	a.ctx = ctx
	defer func() {
		if a.startupErr != nil {
			log.Printf("the war room: startup failed: %v", a.startupErr)
		}
	}()

	// Logging starts first so a lock or open failure below reaches the disk log. A logging
	// failure is not fatal.
	dir, err := configDir()
	if err != nil {
		a.startupErr = fmt.Errorf("startup: resolve config dir: %w", err)
		return
	}
	if lerr := setupLogging(dir); lerr != nil {
		log.Printf("the war room: WARNING disk logging unavailable: %v", lerr)
	}

	path, hist, err := a.openDatabases(ctx, dir)
	if err != nil {
		a.startupErr = fmt.Errorf("startup: %w", err)
		return
	}

	client, err := mfl.New("api", 2, mfl.WithTransport(a.fetches),
		mfl.WithKeySource(func(ctx context.Context) (mflkey.Key, error) {
			if a.keyStore == nil {
				return "", mflkey.ErrUnavailable
			}
			return a.keyStore.Get(ctx)
		}))
	if err != nil {
		a.startupErr = fmt.Errorf("startup: mfl client: %w", err)
		return
	}
	a.mflClient = client

	log.Printf("the war room: starting %s, db %s", buildLabel(), path)
	began := time.Now()
	refreshed, err := a.initStoreFloor(ctx, hist)
	if err != nil {
		a.startupErr = err
		return
	}
	log.Printf("the war room: season %d ready in %s", a.season, time.Since(began).Round(time.Millisecond))
	a.keyStore = mflkey.New(ingestion.LeagueID, strconv.Itoa(a.season))
	a.launchRefreshDue = !refreshed
}

// domReady is the Wails OnDomReady hook: the window is up, so the launch refresh and the signals
// load run now, off the startup path. A windowless -probe never reaches it.
func (a *App) domReady(ctx context.Context) {
	a.refreshInBackground(ctx)
	a.signalsInBackground(ctx)
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
	a.lockFile = lock
	if a.pools, err = db.Open(ctx, path); err != nil {
		return "", nil, fmt.Errorf("open database: %w", err)
	}
	reg, err := measures.Embedded()
	if err != nil {
		return "", nil, err //nolint:wrapcheck // the registry's errors name the CSV line
	}
	if a.histPools, err = db.Open(ctx, filepath.Join(dir, dbFileName("history", isDevBuild()))); err != nil {
		return "", nil, fmt.Errorf("open history database: %w", err)
	}
	if a.whatifPools, err = db.Open(ctx, filepath.Join(dir, dbFileName("whatif", isDevBuild()))); err != nil {
		return "", nil, fmt.Errorf("open what-if database: %w", err)
	}
	hist := history.New(a.histPools, reg)
	a.fetches = &archive.Transport{Sink: hist}
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

	pstore := params.New(a.pools)
	rb := rulebook.New(a.pools)
	mirror := state.NewMirror(a.pools, rb)
	var whatif *state.Store
	var coord *transactions.Coordinator
	steps := []struct {
		name string
		run  func(context.Context) error
	}{
		{"history", hist.Initialize},
		{"params", pstore.Initialize},
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
			a.season = mirror.Season()
			whatif = state.New(a.whatifPools, ingestion.LeagueID, a.season, rb)
			return whatif.Initialize(c, mirror)
		}},
		// The coordinator is the only holder of the what-if Writer.
		{"transaction coordinator", func(context.Context) error {
			var err error
			coord, err = transactions.New(whatif.Writer(), &rosterPolicyAdapter{rb: rb, app: a},
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
	a.params, a.rulebook, a.league, a.whatif, a.coordinator, a.history = pstore, rb, mirror, whatif, coord, hist
	return refreshed, nil
}

// shutdown releases the database and the instance lock.
func (a *App) shutdown(_ context.Context) {
	if a.weekCancel != nil {
		a.weekCancel()
	}
	a.weekWorkers.Wait()
	for _, p := range []*db.Pools{a.pools, a.histPools, a.whatifPools} {
		if p != nil {
			_ = p.Close()
		}
	}
	releaseInstanceLock(a.lockFile)
}

// configDir returns the app's data directory (~/.config/TheWarRoom on Linux), creating it.
func configDir() (string, error) {
	cfg, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("user config dir: %w", err)
	}
	dir := filepath.Join(cfg, "TheWarRoom")
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return "", fmt.Errorf("create data dir %q: %w", dir, err)
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
