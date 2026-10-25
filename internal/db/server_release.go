package db

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"os"
	"time"
)

// ErrNotDatabase marks a file that is not an SQLite database at all.
var ErrNotDatabase = errors.New("db: not an SQLite database")

// FileState is what a read-only look at a database file found. Release is the newest server
// release that stamped the file; zero means none has.
type FileState struct {
	Empty       bool
	Schema      int
	Release     int
	IntegrityOK bool
}

// CreateServerRelease adds the one-row table naming the newest server release that served a file.
// Desktop files keep it empty.
func CreateServerRelease(ctx context.Context, tx *sql.Tx) error {
	const ddl = `CREATE TABLE server_release (
 id INTEGER PRIMARY KEY CHECK(id = 1),
 release INTEGER NOT NULL CHECK(release > 0),
 stamped_at TEXT NOT NULL
)`
	if _, err := tx.ExecContext(ctx, ddl); err != nil {
		return fmt.Errorf("db: create server release: %w", err)
	}
	return nil
}

// StampServerRelease records release as the file's server release. It never lowers an existing
// stamp: an older server cannot claim a file a newer one has served.
func (p *Pools) StampServerRelease(ctx context.Context, release int, at time.Time) error {
	if release <= 0 {
		return fmt.Errorf("db: server release must be positive")
	}
	var stamped int
	err := p.write.QueryRowContext(ctx, `INSERT INTO server_release(id, release, stamped_at) VALUES(1, ?, ?)
 ON CONFLICT(id) DO UPDATE SET release = excluded.release, stamped_at = excluded.stamped_at
 WHERE excluded.release >= server_release.release
 RETURNING release`, release, at.UTC().Format(time.RFC3339Nano)).Scan(&stamped)
	if errors.Is(err, sql.ErrNoRows) {
		return fmt.Errorf("db: a newer server release already stamped this file")
	}
	if err != nil {
		return fmt.Errorf("db: stamp server release: %w", err)
	}
	return nil
}

// Inspect reads path without writing to it, whether or not a server has it open. The schema
// version comes from the file header first, so a file a newer build wrote is judged before SQLite
// opens it at all; newerThan is the newest schema the caller understands. A newer file returns
// its header schema and nothing else.
func Inspect(ctx context.Context, path string, newerThan int) (FileState, error) {
	header, err := readHeader(path)
	if err != nil {
		return FileState{}, err
	}
	if header == nil {
		return FileState{Empty: true}, nil
	}
	state := FileState{Schema: int(binary.BigEndian.Uint32(header[60:64]))}
	if state.Schema > newerThan {
		return state, nil
	}
	query := "?mode=ro"
	if _, err := os.Lstat(path + "-wal"); errors.Is(err, os.ErrNotExist) {
		// No WAL file: no server has it open. An immutable read takes no locks and, unlike a plain
		// read-only open, leaves no -wal or -shm file behind.
		query += "&immutable=1"
	}
	probe, err := sql.Open("sqlite", fileURI(path)+query)
	if err != nil {
		return state, fmt.Errorf("db: open inspection: %w", err)
	}
	defer func() { _ = probe.Close() }()
	// The header may predate frames still in the WAL; SQLite's own answer is authoritative.
	if err := probe.QueryRowContext(ctx, "PRAGMA user_version").Scan(&state.Schema); err != nil {
		return state, fmt.Errorf("db: read schema version: %w", err)
	}
	if state.Schema > newerThan {
		return state, nil
	}
	var check string
	if err := probe.QueryRowContext(ctx, "PRAGMA quick_check(1)").Scan(&check); err != nil {
		return state, fmt.Errorf("db: integrity check: %w", err)
	}
	state.IntegrityOK = check == "ok"
	var marked bool
	if err := probe.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM sqlite_master WHERE type = 'table' AND name = 'server_release')`).Scan(&marked); err != nil {
		return state, fmt.Errorf("db: look for server release: %w", err)
	}
	if !marked {
		return state, nil
	}
	err = probe.QueryRowContext(ctx, `SELECT release FROM server_release WHERE id = 1`).Scan(&state.Release)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return state, fmt.Errorf("db: read server release: %w", err)
	}
	return state, nil
}

// readHeader returns the 100-byte SQLite header, or nil for an empty file.
func readHeader(path string) ([]byte, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("db: open %q: %w", path, err)
	}
	defer func() { _ = f.Close() }()
	header := make([]byte, 100)
	n, err := io.ReadFull(f, header)
	if n == 0 && errors.Is(err, io.EOF) {
		return nil, nil
	}
	if err != nil || !bytes.HasPrefix(header, []byte("SQLite format 3\x00")) {
		return nil, ErrNotDatabase
	}
	return header, nil
}
