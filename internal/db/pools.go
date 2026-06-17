// Package db owns SQLite access for TheWarRoom. It is one of only two packages
// (with internal/store) permitted to import database/sql and the sqlite driver
// — the G0 overlay's depguard `sql-confined-to-data-layer` rule enforces this.
//
// The architecture is the split read/write pool (companion plan Section 3.4),
// which enforces SQLite's single-writer model at the driver level:
//
//   - Write pool: exactly ONE connection (SetMaxOpenConns(1)), opened with
//     _txlock=immediate so every transaction takes the write lock up front.
//     This serializes all writes and removes the SQLITE_BUSY race entirely.
//     Only the B7 transaction coordinator is given the write pool.
//   - Read pool: many connections, opened read-only (mode=ro). Every other
//     consumer — the engine, modules, handlers — gets the read pool.
//
// Both run against one WAL-mode file, so readers never block the writer and
// vice versa. WAL is a file-level property, set once by the write connection
// and then seen by the read pool.
//
// modernc.org/sqlite (pure Go, CGo-free) is used over mattn/go-sqlite3 so Wails
// cross-compilation needs no C toolchain. Its DSN uses repeatable `_pragma=`
// parameters, NOT mattn's `_busy_timeout=`/`_journal_mode=` style.
package db

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	_ "modernc.org/sqlite" // registers the "sqlite" driver
)

// busyTimeoutMS is how long a connection waits on a locked database before
// returning SQLITE_BUSY. With the single-writer pool this should never be hit,
// but it is a cheap safety margin against momentary WAL checkpoint contention.
const busyTimeoutMS = 5000

// Pools holds the split read/write connection pools for one SQLite database.
// Construct it with Open; release it with Close. The zero value is not usable.
type Pools struct {
	read  *sql.DB
	write *sql.DB
}

// Open opens the write and read pools against the file at path, verifies that
// WAL mode actually took effect, and returns the ready Pools. The write pool is
// opened first so the database file and its WAL header exist before the
// read-only pool attaches.
func Open(ctx context.Context, path string) (*Pools, error) {
	writeDSN := fmt.Sprintf(
		"file:%s?_pragma=busy_timeout(%d)&_pragma=journal_mode(WAL)&_txlock=immediate",
		path, busyTimeoutMS,
	)
	write, err := sql.Open("sqlite", writeDSN)
	if err != nil {
		return nil, fmt.Errorf("db: open write pool: %w", err)
	}
	write.SetMaxOpenConns(1)    // ONE writer — serializes writes, no SQLITE_BUSY race.
	write.SetMaxIdleConns(1)    // keep the single writer warm — no open/close churn.
	write.SetConnMaxLifetime(0) // never recycle the writer connection.

	// Force the connection (and thus the file + WAL header) into existence, then
	// confirm WAL is genuinely on. A pragma that silently fails to apply is
	// invisible until it deadlocks under load — so we assert it here, at startup.
	//
	// ORDERING DEPENDENCY (do not reorder): this query also MATERIALIZES the
	// database file. It must run BEFORE the read pool opens with mode=ro, since a
	// read-only open fails if the file does not yet exist (Gemini Round-2 #6).
	mode, err := journalMode(ctx, write)
	if err != nil {
		_ = write.Close()
		return nil, fmt.Errorf("db: verify journal mode: %w", err)
	}
	if mode != "wal" {
		_ = write.Close()
		return nil, fmt.Errorf("db: expected WAL journal mode, got %q — pragma did not apply", mode)
	}

	readDSN := fmt.Sprintf(
		"file:%s?_pragma=busy_timeout(%d)&mode=ro",
		path, busyTimeoutMS,
	)
	read, err := sql.Open("sqlite", readDSN)
	if err != nil {
		_ = write.Close()
		return nil, fmt.Errorf("db: open read pool: %w", err)
	}
	read.SetMaxOpenConns(10) // many concurrent readers (Wails IPC goroutines).
	read.SetMaxIdleConns(10) // match idle to open — no connection thrashing under burst.

	if err := read.PingContext(ctx); err != nil {
		_ = read.Close()
		_ = write.Close()
		return nil, fmt.Errorf("db: ping read pool: %w", err)
	}

	return &Pools{read: read, write: write}, nil
}

// Read returns the read-only pool. Every consumer except the transaction
// coordinator uses this.
func (p *Pools) Read() *sql.DB { return p.read }

// Write returns the single-connection write pool. Only the B7 transaction
// coordinator is given this.
func (p *Pools) Write() *sql.DB { return p.write }

// Health pings both pools. It backs the B0 IPC ping-pong: a round trip that
// proves the Go<->JS bridge AND that the data layer opened correctly.
func (p *Pools) Health(ctx context.Context) error {
	if err := p.write.PingContext(ctx); err != nil {
		return fmt.Errorf("db: write pool unhealthy: %w", err)
	}
	if err := p.read.PingContext(ctx); err != nil {
		return fmt.Errorf("db: read pool unhealthy: %w", err)
	}
	return nil
}

// JournalMode returns the database's active journal mode (expected "wal").
// Exposed for the B0 health check / ping-pong proof.
func (p *Pools) JournalMode(ctx context.Context) (string, error) {
	return journalMode(ctx, p.write)
}

// Close releases both pools. Safe to call on a partially-open Pools.
func (p *Pools) Close() error {
	var errs []error
	if p.read != nil {
		if err := p.read.Close(); err != nil {
			errs = append(errs, fmt.Errorf("db: close read pool: %w", err))
		}
	}
	if p.write != nil {
		if err := p.write.Close(); err != nil {
			errs = append(errs, fmt.Errorf("db: close write pool: %w", err))
		}
	}
	return errors.Join(errs...)
}

// journalMode reads PRAGMA journal_mode from a pool. Executing it on the write
// connection both materializes the file and reports the effective mode.
func journalMode(ctx context.Context, pool *sql.DB) (string, error) {
	var mode string
	if err := pool.QueryRowContext(ctx, "PRAGMA journal_mode").Scan(&mode); err != nil {
		return "", fmt.Errorf("query journal_mode: %w", err)
	}
	return mode, nil
}
