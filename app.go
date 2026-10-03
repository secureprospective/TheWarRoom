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

	"github.com/secureprospective/TheWarRoom/internal/db"
	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/ingestion"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/league"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/players"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/rosters"
	"github.com/secureprospective/TheWarRoom/internal/mfl"
	"github.com/secureprospective/TheWarRoom/internal/normalize"
	"github.com/secureprospective/TheWarRoom/internal/output"
	"github.com/secureprospective/TheWarRoom/internal/store/params"
	"github.com/secureprospective/TheWarRoom/internal/store/rulebook"
	"github.com/secureprospective/TheWarRoom/internal/store/state"
	"github.com/secureprospective/TheWarRoom/internal/transactions"
)

// App is the Wails root and the backend's composition root: it wires the stores and routes
// IPC calls. Business logic lives in the engine, store and transaction packages, never here.
type App struct {
	//nolint:containedctx // Wails IPC methods get no per-call context; this is the app-lifetime one, and each method derives a bounded context from it
	ctx         context.Context
	pools       *db.Pools
	params      *params.Store
	rulebook    *rulebook.Store
	state       *state.Store
	output      *output.Store
	coordinator *transactions.Coordinator // the only holder of the state Writer
	mflClient   *mfl.Client               // shared, so rate limit and host discovery are process-wide
	season      int
	startupErr  error    // startup failure; shown in the shell through AppInfo
	lockFile    *os.File // single-instance lock, held until shutdown

	// The players directory is fetched at most once per process (MFL allows the endpoint once a
	// day). Wails runs IPC calls concurrently, hence the mutex.
	lookupMu  sync.Mutex
	lookup    normalize.Lookup
	hasLookup bool
}

// directory returns the cached players Lookup, fetching it on first use.
func (a *App) directory(ctx context.Context) (normalize.Lookup, error) {
	a.lookupMu.Lock()
	defer a.lookupMu.Unlock()
	if a.hasLookup {
		return a.lookup, nil
	}
	raws, err := players.Fetch(ctx, a.mflClient, ingestion.SeasonYear, ingestion.LeagueID)
	if err != nil {
		return normalize.Lookup{}, fmt.Errorf("app: fetch players db: %w", err)
	}
	lk, err := normalize.NewLookup(raws)
	if err != nil {
		return normalize.Lookup{}, fmt.Errorf("app: build players lookup: %w", err)
	}
	a.lookup, a.hasLookup = lk, true
	return lk, nil
}

// rosterSeedSource seeds league state from MFL rosters; state.Initialize calls it only on a
// fresh DB.
type rosterSeedSource struct{ app *App }

func (s rosterSeedSource) Rosters(ctx context.Context) ([]domain.Roster, error) {
	lk, err := s.app.directory(ctx)
	if err != nil {
		return nil, err
	}
	raws, err := rosters.Fetch(ctx, s.app.mflClient, ingestion.SeasonYear, ingestion.LeagueID)
	if err != nil {
		return nil, fmt.Errorf("app: fetch rosters seed: %w", err)
	}
	seed, err := normalize.Rosters(raws, lk)
	if err != nil {
		return nil, fmt.Errorf("app: normalize roster seed: %w", err)
	}
	return seed, nil
}

// NewApp is cheap; resources are acquired in startup.
func NewApp() *App {
	return &App{}
}

// startup is the Wails OnStartup hook. OnStartup cannot return an error, so a failure is
// kept in startupErr, logged, and shown by the shell's banner through AppInfo.
func (a *App) startup(ctx context.Context) {
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

	path := filepath.Join(dir, dbFileName(isDevBuild()))
	// One running copy per database: two processes writing one ledger is the hazard.
	lock, err := acquireInstanceLock(path)
	if err != nil {
		a.startupErr = fmt.Errorf("startup: %w", err)
		return
	}
	a.lockFile = lock

	pools, err := db.Open(ctx, path)
	if err != nil {
		a.startupErr = fmt.Errorf("startup: open database: %w", err)
		return
	}
	a.pools = pools

	season, err := strconv.Atoi(ingestion.SeasonYear)
	if err != nil {
		a.startupErr = fmt.Errorf("startup: parse season %q: %w", ingestion.SeasonYear, err)
		return
	}
	a.season = season
	client, err := mfl.New("api", 2)
	if err != nil {
		a.startupErr = fmt.Errorf("startup: mfl client: %w", err)
		return
	}
	a.mflClient = client

	log.Printf("the war room: starting %s (%s) season %d, db %s", version, commit, season, path)
	began := time.Now()
	if err := a.initStoreFloor(ctx); err != nil {
		a.startupErr = err
		return
	}
	log.Printf("the war room: ready in %s", time.Since(began).Round(time.Millisecond))
}

// startupBudget bounds store-floor init. A fresh DB seeds from MFL on this thread, so an
// unreachable MFL would otherwise hang the window black forever.
const startupBudget = 2 * time.Minute

// initStoreFloor brings up the stores and the transaction coordinator, logging each step's
// time so a slow or failed launch shows where it stopped. Stores seed from MFL only on a fresh
// DB. Fields are assigned only once every step succeeds: IPC methods treat a nil store as
// "not initialized".
func (a *App) initStoreFloor(parent context.Context) error {
	ctx, cancel := context.WithTimeout(parent, startupBudget)
	defer cancel()

	pstore := params.New(a.pools)
	rb := rulebook.New(a.pools)
	st := state.New(a.pools, ingestion.LeagueID, a.season, rb)
	out := output.New(a.pools)
	var coord *transactions.Coordinator
	steps := []struct {
		name string
		run  func(context.Context) error
	}{
		{"params", pstore.Initialize},
		{"rulebook", func(c context.Context) error {
			return rb.Initialize(c, league.APISource{Client: a.mflClient, Year: ingestion.SeasonYear, LeagueID: ingestion.LeagueID})
		}},
		{"league state", func(c context.Context) error { return st.Initialize(c, rosterSeedSource{app: a}) }},
		// The coordinator is the only holder of the state Writer.
		{"transaction coordinator", func(context.Context) error {
			var err error
			coord, err = transactions.New(st.Writer(), &rosterPolicyAdapter{rb: rb, app: a},
				func(c context.Context) (transactions.Directory, error) { return a.directory(c) })
			return err //nolint:wrapcheck // wrapped below with the step name
		}},
		{"output", out.Initialize},
	}
	for _, s := range steps {
		began := time.Now()
		if err := s.run(ctx); err != nil {
			return fmt.Errorf("startup: initialize %s: %w", s.name, err)
		}
		log.Printf("the war room: %s ready in %s", s.name, time.Since(began).Round(time.Millisecond))
	}
	a.params, a.rulebook, a.state, a.coordinator, a.output = pstore, rb, st, coord, out
	return nil
}

// shutdown releases the database and the instance lock.
func (a *App) shutdown(_ context.Context) {
	if a.pools != nil {
		_ = a.pools.Close()
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

// dbFileName gives dev builds their own database, so development can never migrate or
// corrupt the real league ledger.
func dbFileName(devBuild bool) string {
	if devBuild {
		return "thewarroom-dev.db"
	}
	return "thewarroom.db"
}
