// Package db owns SQLite access (only db and store may import database/sql). SQLite allows
// one writer, so the write pool is one connection with _txlock=immediate (writes never race
// into SQLITE_BUSY) and the read pool is many read-only connections, over one WAL file. The
// driver is modernc.org/sqlite: pure Go, so Wails needs no C toolchain.
package db

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	_ "modernc.org/sqlite" // registers the "sqlite" driver
)

// busyTimeoutMS is a safety margin for WAL checkpoint contention; the single writer should
// never hit it.
const busyTimeoutMS = 5000

const connectionPragmas = "&_pragma=foreign_keys(ON)&_pragma=recursive_triggers(ON)&_pragma=synchronous(FULL)"

// Pools is the read/write pool pair for one database. Construct with Open.
type Pools struct {
	read             *sql.DB
	write            *sql.DB
	path             string
	checkRunMu       sync.Mutex
	checkMu          sync.Mutex
	lastCheck        RoutineCheck
	migrationMu      sync.Mutex
	findings         Findings
	checkedVersion   int
	migrationChecked bool
}

// Open refuses newer files before opening a writer. The optional version is the file's
// newest known migration; the default is MainVersion (also the current what-if version).
func Open(ctx context.Context, path string, versions ...int) (*Pools, error) {
	maxVersion := MainVersion()
	if len(versions) > 1 || (len(versions) == 1 && versions[0] < 0) {
		return nil, fmt.Errorf("db: expected one nonnegative schema version")
	}
	if len(versions) == 1 {
		maxVersion = versions[0]
	}
	if err := checkUserVersion(ctx, path, maxVersion); err != nil {
		return nil, err
	}
	if err := tightenOwned(path); err != nil {
		return nil, err
	}
	if err := createOwnerOnly(path); err != nil {
		return nil, err
	}
	writeDSN := fmt.Sprintf(
		"%s?_pragma=busy_timeout(%d)&_pragma=journal_mode(WAL)&_txlock=immediate",
		fileURI(path), busyTimeoutMS,
	) + connectionPragmas
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
		"%s?_pragma=busy_timeout(%d)&mode=ro",
		fileURI(path), busyTimeoutMS,
	) + connectionPragmas
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

	return &Pools{read: read, write: write, path: path}, nil
}

// createOwnerOnly creates a missing file empty and 0600 (an empty file is a valid empty
// database). SQLite would create it 0644, and it gives the WAL, shared-memory and journal
// files the main file's mode, so this keeps all of them owner-only. An existing file is
// left as it is.
func createOwnerOnly(path string) error {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		return fmt.Errorf("db: create %q: %w", path, err)
	}
	if err := f.Close(); err != nil {
		return fmt.Errorf("db: create %q: %w", path, err)
	}
	return nil
}

// tightenOwned makes the database, its sidecars and its backups owner-only where an older build
// left them group- or other-readable. Only regular files under the database's own names are
// changed; a symlink is never followed, so nothing outside the data dir is touched.
func tightenOwned(path string) error {
	base := filepath.Base(path)
	entries, err := os.ReadDir(filepath.Dir(path))
	if errors.Is(err, os.ErrNotExist) {
		return nil
	} else if err != nil {
		return fmt.Errorf("db: restrict %q: %w", path, err)
	}
	for _, e := range entries {
		name := e.Name()
		owned := name == base || name == base+"-wal" || name == base+"-shm" || name == base+"-journal" ||
			strings.HasPrefix(name, base+".premigration-") ||
			isMigrationBackup(name, base)
		if !owned || !e.Type().IsRegular() {
			continue
		}
		if err := tightenFile(filepath.Join(filepath.Dir(path), name)); err != nil {
			return err
		}
	}
	return nil
}

// tightenFile changes the mode through the opened file, and only when it is the regular file
// the directory listed: a file swapped for a symlink after the listing is left alone.
func tightenFile(path string) error {
	listed, err := os.Lstat(path)
	if err != nil || !listed.Mode().IsRegular() || listed.Mode().Perm()&0o077 == 0 {
		return nil // a file gone or replaced since the listing is not ours to change
	}
	f, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("db: restrict %q: %w", path, err)
	}
	defer func() { _ = f.Close() }()
	opened, err := f.Stat()
	if err != nil {
		return fmt.Errorf("db: restrict %q: %w", path, err)
	}
	if !os.SameFile(listed, opened) {
		return nil
	}
	if err := f.Chmod(listed.Mode().Perm() &^ 0o077); err != nil {
		return fmt.Errorf("db: restrict %q: %w", path, err)
	}
	return nil
}

func checkUserVersion(ctx context.Context, path string, maxVersion int) (err error) {
	if _, err := os.Stat(path); errors.Is(err, os.ErrNotExist) {
		return nil
	} else if err != nil {
		return fmt.Errorf("db: stat database: %w", err)
	}
	probe, err := sql.Open("sqlite", fileURI(path)+"?mode=ro")
	if err != nil {
		return fmt.Errorf("db: open version probe: %w", err)
	}
	defer func() {
		if closeErr := probe.Close(); closeErr != nil {
			err = errors.Join(err, fmt.Errorf("db: close version probe: %w", closeErr))
		}
	}()
	var version int
	if err := probe.QueryRowContext(ctx, "PRAGMA user_version").Scan(&version); err != nil {
		return fmt.Errorf("db: read user_version: %w", err)
	}
	return refuseNewer(version, maxVersion)
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

// fileURI makes a SQLite file: URI for path. SQLite reads ? and # as URI delimiters and %
// as an escape, so a folder named "Bob#2" would otherwise open a different file.
func fileURI(path string) string {
	return "file:" + strings.NewReplacer("%", "%25", "?", "%3f", "#", "%23").Replace(path)
}
