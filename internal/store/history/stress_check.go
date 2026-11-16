package history

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/url"
	"os"

	"github.com/secureprospective/TheWarRoom/internal/db"
)

// StressEntry is the identity and revision of a durable event, without its payload.
type StressEntry struct {
	League, Stream, Actor, Command string
	Revision                       int64
}

// InspectStressDatabase reads integrity findings and ledger identities without migrations.
// A stopped database without a WAL is immutable so SQLite cannot create sidecar files.
func InspectStressDatabase(ctx context.Context, path string) (findings db.CheckFindings, entries []StressEntry, err error) {
	uri := url.URL{Scheme: "file", Path: path}
	query := url.Values{"mode": {"ro"}}
	if _, statErr := os.Stat(path + "-wal"); errors.Is(statErr, os.ErrNotExist) {
		query.Set("immutable", "1")
	} else if statErr != nil {
		return findings, nil, fmt.Errorf("stat stress WAL: %w", statErr)
	}
	uri.RawQuery = query.Encode()
	pool, err := sql.Open("sqlite", uri.String())
	if err != nil {
		return findings, nil, fmt.Errorf("open stress database: %w", err)
	}
	defer func() { err = errors.Join(err, pool.Close()) }()
	findings, err = db.Check(ctx, pool)
	if err != nil {
		return findings, nil, fmt.Errorf("check stress database: %w", err)
	}
	var ledger bool
	if err = pool.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM sqlite_master WHERE type='table' AND name='league_event')`).Scan(&ledger); err != nil {
		return findings, nil, fmt.Errorf("find stress ledger: %w", err)
	}
	if ledger {
		entries, err = stressEntries(ctx, pool)
	}
	return findings, entries, err
}

func stressEntries(ctx context.Context, pool *sql.DB) ([]StressEntry, error) {
	rows, err := pool.QueryContext(ctx, `SELECT league,stream,actor,correlation_id,revision FROM league_event ORDER BY league,stream,revision`)
	if err != nil {
		return nil, fmt.Errorf("read stress ledger: %w", err)
	}
	defer func() { _ = rows.Close() }()
	var entries []StressEntry
	for rows.Next() {
		var entry StressEntry
		if err := rows.Scan(&entry.League, &entry.Stream, &entry.Actor, &entry.Command, &entry.Revision); err != nil {
			return nil, fmt.Errorf("scan stress ledger: %w", err)
		}
		entries = append(entries, entry)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate stress ledger: %w", err)
	}
	return entries, nil
}
