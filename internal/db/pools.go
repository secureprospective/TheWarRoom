// Package db owns SQLite access. Only db and store may import database/sql (depguard
// sql-confined-to-data-layer).
//
// SQLite allows one writer, so the pools are split:
//   - Write: one connection opened with _txlock=immediate, so every transaction takes the
//     write lock up front and writes never race into SQLITE_BUSY.
//   - Read: many read-only (mode=ro) connections.
//
// Both use one WAL-mode file, so readers and the writer never block each other. The driver
// is modernc.org/sqlite (pure Go, no C toolchain for Wails); its DSN takes repeated _pragma=
// parameters.
package db

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	_ "modernc.org/sqlite" // registers the "sqlite" driver
)

// busyTimeoutMS is a safety margin for WAL checkpoint contention; the single writer should
// never hit it.
const busyTimeoutMS = 5000

// Pools is the read/write pool pair for one database. Construct with Open.
type Pools struct {
	read  *sql.DB
	write *sql.DB
}

// Open opens both pools and checks that WAL mode took effect. The write pool opens first so the
// file exists before the read-only pool attaches.
func Open(ctx context.Context, path string) (*Pools, error) {
	writeDSN := fmt.Sprintf(
		"file:%s?_pragma=busy_timeout(%d)&_pragma=journal_mode(WAL)&_txlock=immediate",
		path, busyTimeoutMS,
	)
	write, err := sql.Open("sqlite", writeDSN)
	if err != nil {
		return nil, fmt.Errorf("db: open write pool: %w", err)
	}
	write.SetMaxOpenConns(1)
	write.SetMaxIdleConns(1)
	write.SetConnMaxLifetime(0)

	// This query creates the file and confirms WAL; a pragma that silently fails shows up only
	// as a deadlock under load. It must run before the read pool opens, because mode=ro fails on
	// a missing file.
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
	read.SetMaxOpenConns(10)
	read.SetMaxIdleConns(10)

	if err := read.PingContext(ctx); err != nil {
		_ = read.Close()
		_ = write.Close()
		return nil, fmt.Errorf("db: ping read pool: %w", err)
	}

	return &Pools{read: read, write: write}, nil
}

// Read returns the read-only pool.
func (p *Pools) Read() *sql.DB { return p.read }

// Write returns the single-connection write pool, for the stores' write paths.
func (p *Pools) Write() *sql.DB { return p.write }

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
