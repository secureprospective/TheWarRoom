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

// App is the Wails application root and the composition root for the backend.
// It owns process-lifetime resources (the SQLite pools) and exposes the
// IPC-bound methods the frontend calls. Per the three-layer law it stays a thin
// adapter: it wires dependencies and routes calls — business logic lives in the
// engine, stores, and transaction packages, never here.
type App struct {
	//nolint:containedctx // Wails binds IPC methods with no per-call context; the
	// app-lifetime context captured at OnStartup is the sanctioned source for
	// backend calls (B0 first-instance decision — see SYSTEM_MAP.md).
	ctx      context.Context
	pools    *db.Pools
	params   *params.Store   // B4 calibration store; backs the harness admin panel
	rulebook *rulebook.Store // B3b league config; active version stamps B6 (M1)
	state    *state.Store    // B3c runtime rosters/contracts; the M1 roster source
	output   *output.Store   // B6 per-season engine output; the M1 board reads it
	//nolint:lll // field comment
	coordinator *transactions.Coordinator // B7a sole runtime mutator; holds the ONLY state.Writer
	mflClient   *mfl.Client               // shared transport; rate limit + host cache live here
	season      int                       // ingestion.SeasonYear parsed once at startup
	startupErr  error                     // captured at startup; surfaced via Ping (Wails OnStartup cannot fail).
	lockFile    *os.File                  // single-instance advisory lock; held for process lifetime, released at shutdown.

	// players-DB directory, fetched at most once per process (MFL caps the
	// endpoint at once/day): the state seed (fresh DB only) and every M1
	// name/position/birthdate resolution share this one cached Lookup.
	// Guarded by lookupMu — Wails runs IPC calls concurrently.
	lookupMu  sync.Mutex
	lookup    normalize.Lookup
	hasLookup bool
}

// directory returns the cached players-DB Lookup, fetching and normalizing it on
// first use. The mutex makes concurrent first calls collapse into one fetch.
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

// rosterSeedSource adapts the Layer-1 rosters fetch + normalize join into the
// state.Source seam. It is invoked by state.Initialize ONLY on a fresh DB — an
// existing database loads as-is with no network (B3c seed-once law).
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

// NewApp creates a new App. Resources are acquired in startup, not here, so the
// struct is cheap to construct and test.
func NewApp() *App {
	return &App{}
}

// PingResult is the IPC ping-pong payload: a typed, JSON-serializable round
// trip that proves the Go<->JS bridge works AND that the data layer came up.
// No interface{}/any fields — the boundary stays fully typed (ifaceguard).
type PingResult struct {
	OK          bool   `json:"ok"`
	Message     string `json:"message"`
	JournalMode string `json:"journalMode"`
	Detail      string `json:"detail"`
}

// startup is the Wails OnStartup hook. It saves the context and opens the
// SQLite pools. OnStartup has no error return, so a failure is captured in
// startupErr and reported through Ping rather than silently swallowed.
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx

	// Wails OnStartup cannot return an error, so a failure here is captured in
	// startupErr and surfaced through Ping. But nothing in the shell pings on load,
	// so a broken startup shows only as a downstream "store not initialized" from a
	// panel — masking the real cause. Log it to stderr too so a terminal launch
	// prints the underlying error directly (Ship-4 diagnostic).
	defer func() {
		if a.startupErr != nil {
			log.Printf("the war room: startup failed: %v", a.startupErr)
		}
	}()

	// Resolve the data dir ONCE (it creates the dir); logging and the DB path both
	// derive from it. Disk logging first, so anything below (a lock or open failure)
	// is captured on disk as well as stderr. A logging failure is non-fatal.
	dir, err := configDir()
	if err != nil {
		a.startupErr = fmt.Errorf("startup: resolve config dir: %w", err)
		return
	}
	if lerr := setupLogging(dir); lerr != nil {
		log.Printf("the war room: WARNING disk logging unavailable: %v", lerr)
	}

	path := filepath.Join(dir, dbFileName(isDevBuild()))
	// Single-instance guard BEFORE opening the DB: a second copy must not attach to
	// the same ledger (a migration or write from two processes is the hazard).
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

	// The MFL client is shared so rate limiting and the discovered league host are
	// process-wide. Season is parsed once — canonical ingestion.SeasonYear is a string
	// for URL building.
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

	if err := a.initStoreFloor(ctx); err != nil {
		a.startupErr = err
		return
	}
}

// initStoreFloor constructs and initializes the full store floor plus the B7a
// transaction coordinator. Each store seeds from live MFL ONLY on a fresh DB and loads
// from SQLite after (their Initialize contracts) — so first launch needs the network,
// every later launch comes up offline. Split out of startup to keep that hook within
// the function-length budget and to give the data layer one wiring site.
func (a *App) initStoreFloor(ctx context.Context) error {
	// B4 params: the harness reads cap-tier % and decay rate from it and the admin
	// panel tunes them live. Initialize seeds shipped defaults once, loads calibration
	// on restart.
	pstore := params.New(a.pools)
	if err := pstore.Initialize(ctx); err != nil {
		return fmt.Errorf("startup: initialize params store: %w", err)
	}
	a.params = pstore

	rb := rulebook.New(a.pools)
	if err := rb.Initialize(ctx, league.APISource{Client: a.mflClient, Year: ingestion.SeasonYear, LeagueID: ingestion.LeagueID}); err != nil {
		return fmt.Errorf("startup: initialize rulebook: %w", err)
	}
	a.rulebook = rb

	st := state.New(a.pools, ingestion.LeagueID, a.season, rb)
	if err := st.Initialize(ctx, rosterSeedSource{app: a}); err != nil {
		return fmt.Errorf("startup: initialize league state: %w", err)
	}
	a.state = st

	// B7a: the transaction Coordinator is the SOLE holder of the state Writer in the
	// whole process (AD-02). Wired here, once, right after the state store comes up —
	// nothing else calls st.Writer(). The Session 2 roster-policy adapter (rulebook +
	// players-DB) supplies the roster/position/taxi/IR enforcement gate; composed here
	// because depguard forbids the transactions package from importing either store.
	policy := &rosterPolicyAdapter{rb: rb, app: a}
	coord, err := transactions.New(st.Writer(), policy)
	if err != nil {
		return fmt.Errorf("startup: initialize transaction coordinator: %w", err)
	}
	a.coordinator = coord

	out := output.New(a.pools)
	if err := out.Initialize(ctx); err != nil {
		return fmt.Errorf("startup: initialize output store: %w", err)
	}
	a.output = out
	return nil
}

// shutdown is the Wails OnShutdown hook. It releases the SQLite pools.
func (a *App) shutdown(_ context.Context) {
	if a.pools != nil {
		_ = a.pools.Close()
	}
	releaseInstanceLock(a.lockFile)
}

// Ping is the IPC ping-pong method bound to the frontend. It round-trips a
// typed result and reports data-layer health — the B0 functional-verification
// target.
func (a *App) Ping() PingResult {
	if a.startupErr != nil {
		return PingResult{OK: false, Message: "pong", Detail: a.startupErr.Error()}
	}
	if a.pools == nil {
		return PingResult{OK: false, Message: "pong", Detail: "database not initialized"}
	}
	// Derive a bounded context from the app-lifetime ctx: an IPC method must never
	// block the frontend indefinitely if the data layer stalls. Every IPC method
	// inherits this pattern (Gemini Round-2 finding #1).
	ctx, cancel := context.WithTimeout(a.ctx, 3*time.Second)
	defer cancel()
	if err := a.pools.Health(ctx); err != nil {
		return PingResult{OK: false, Message: "pong", Detail: err.Error()}
	}
	mode, err := a.pools.JournalMode(ctx)
	if err != nil {
		return PingResult{OK: false, Message: "pong", Detail: err.Error()}
	}
	return PingResult{
		OK:          true,
		Message:     "pong",
		JournalMode: mode,
		Detail:      "Go<->JS bridge live; SQLite read/write pools healthy.",
	}
}

// configDir returns the app's on-disk data directory (e.g. ~/.config/TheWarRoom),
// creating it if needed. The database, the instance lockfile, and the logs dir all
// live under it.
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

// isDevBuild reports whether this is an un-stamped DEV binary (plain `go build` or
// `wails dev`), where the link-time version default "dev" survives (see version.go).
func isDevBuild() bool { return version == "dev" }

// dbFileName is the SQLite filename for this build. A DEV build uses a SEPARATE
// -dev database so development (`wails dev`, un-stamped binaries) can never open —
// and never migrate or corrupt — the real dynasty ledger (Tier 3 dev-build guard).
// A stamped release uses the real ledger.
func dbFileName(devBuild bool) string {
	if devBuild {
		return "thewarroom-dev.db"
	}
	return "thewarroom.db"
}

// databasePath returns the on-disk location of the SQLite file under the config dir,
// selecting the -dev database for an un-stamped build.
func databasePath() (string, error) {
	dir, err := configDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, dbFileName(isDevBuild())), nil
}
