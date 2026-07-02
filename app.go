package main

import (
	"context"
	"fmt"
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
	ctx        context.Context
	pools      *db.Pools
	params     *params.Store   // B4 calibration store; backs the harness admin panel
	rulebook   *rulebook.Store // B3b league config; active version stamps B6 (M1)
	state      *state.Store    // B3c runtime rosters/contracts; the M1 roster source
	output     *output.Store   // B6 per-season engine output; the M1 board reads it
	mflClient  *mfl.Client     // shared transport; rate limit + host cache live here
	season     int             // ingestion.SeasonYear parsed once at startup
	startupErr error           // captured at startup; surfaced via Ping (Wails OnStartup cannot fail).

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

	path, err := databasePath()
	if err != nil {
		a.startupErr = fmt.Errorf("startup: resolve database path: %w", err)
		return
	}
	pools, err := db.Open(ctx, path)
	if err != nil {
		a.startupErr = fmt.Errorf("startup: open database: %w", err)
		return
	}
	a.pools = pools

	// Construct and seed the B4 params store: the harness reads cap-tier % and decay
	// rate from it and the admin panel tunes them live. Initialize seeds the shipped
	// defaults once on a fresh DB and loads existing calibration on restart.
	pstore := params.New(pools)
	if err := pstore.Initialize(ctx); err != nil {
		a.startupErr = fmt.Errorf("startup: initialize params store: %w", err)
		return
	}
	a.params = pstore

	// M1 store floor. The MFL client is shared so rate limiting and the
	// discovered league host are process-wide. Season is parsed once — the
	// canonical ingestion.SeasonYear is a string for URL building.
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

	// Each store seeds from live MFL ONLY on a fresh DB and loads from SQLite
	// after (their Initialize contracts) — so first launch needs the network,
	// every later launch comes up offline.
	rb := rulebook.New(pools)
	if err := rb.Initialize(ctx, league.APISource{Client: client, Year: ingestion.SeasonYear, LeagueID: ingestion.LeagueID}); err != nil {
		a.startupErr = fmt.Errorf("startup: initialize rulebook: %w", err)
		return
	}
	a.rulebook = rb

	st := state.New(pools, ingestion.LeagueID, season)
	if err := st.Initialize(ctx, rosterSeedSource{app: a}); err != nil {
		a.startupErr = fmt.Errorf("startup: initialize league state: %w", err)
		return
	}
	a.state = st

	out := output.New(pools)
	if err := out.Initialize(ctx); err != nil {
		a.startupErr = fmt.Errorf("startup: initialize output store: %w", err)
		return
	}
	a.output = out
}

// shutdown is the Wails OnShutdown hook. It releases the SQLite pools.
func (a *App) shutdown(_ context.Context) {
	if a.pools != nil {
		_ = a.pools.Close()
	}
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

// databasePath returns the on-disk location of the SQLite file, under the user
// config dir (e.g. ~/.config/TheWarRoom/thewarroom.db), creating the directory
// if needed.
func databasePath() (string, error) {
	cfg, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("user config dir: %w", err)
	}
	dir := filepath.Join(cfg, "TheWarRoom")
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return "", fmt.Errorf("create data dir %q: %w", dir, err)
	}
	return filepath.Join(dir, "thewarroom.db"), nil
}
