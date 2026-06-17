package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/db"
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
	startupErr error // captured at startup; surfaced via Ping (Wails OnStartup cannot fail).
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
