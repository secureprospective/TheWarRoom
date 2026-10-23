package history

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
)

const closedBooksDDL = `
CREATE TABLE league_season_summary (
 league TEXT NOT NULL,
 revision INTEGER NOT NULL CHECK (revision > 0),
 season INTEGER NOT NULL CHECK (season > 0),
 payload BLOB NOT NULL,
 payload_hash TEXT NOT NULL,
 PRIMARY KEY (league, revision)
);`

func addClosedBooks(ctx context.Context, tx *sql.Tx) error {
	if _, err := tx.ExecContext(ctx, closedBooksDDL); err != nil {
		return fmt.Errorf("history: add closed season summaries: %w", err)
	}
	return nil
}

func saveClosedBooks(ctx context.Context, tx *sql.Tx, records []commissioner.Record) error {
	raw, err := commissioner.CloseBooks(records)
	if err != nil {
		return fmt.Errorf("close league books: %w", err)
	}
	body, err := json.Marshal(raw)
	if err != nil {
		return fmt.Errorf("encode closed league books: %w", err)
	}
	hash := sha256.Sum256(body)
	league := records[len(records)-1].Command.League
	if _, err := tx.ExecContext(ctx, `INSERT INTO league_season_summary (league,revision,season,payload,payload_hash) VALUES (?,?,?,?,?)`,
		league, raw.Revision, raw.Season, body, hex.EncodeToString(hash[:])); err != nil {
		return fmt.Errorf("store closed league books: %w", err)
	}
	return nil
}

// An absent, corrupt, unsupported or stale derived row is not a ledger error.
// Only database failures escape; rejection retains the complete replay path.
func loadClosedBooks(ctx context.Context, tx *sql.Tx, league string, records []commissioner.Record) ([]commissioner.Record, error) {
	records = commissioner.WithoutClosedBooks(records)
	boundary := latestClosedRevision(records)
	if boundary == 0 {
		return records, nil
	}
	var body []byte
	var expected string
	var revision, season int
	err := tx.QueryRowContext(ctx, `SELECT revision,season,payload,payload_hash FROM league_season_summary WHERE league=? AND revision=?`,
		league, boundary).Scan(&revision, &season, &body, &expected)
	if errors.Is(err, sql.ErrNoRows) {
		return records, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read closed league books: %w", err)
	}
	hash := sha256.Sum256(body)
	if hex.EncodeToString(hash[:]) != expected {
		return records, nil
	}
	var raw commissioner.RawClosedBooks
	if json.Unmarshal(body, &raw) != nil || raw.Validate() != nil || raw.Revision != revision || raw.Season != season {
		return records, nil //nolint:nilerr // Invalid derived payloads intentionally fall back to authoritative replay.
	}
	folded, _ := commissioner.WithClosedBooks(records, raw)
	return folded, nil
}

func (s *Store) eventsInTransaction(ctx context.Context, tx *sql.Tx, league string) ([]commissioner.Record, error) {
	rows, err := tx.QueryContext(ctx, eventsQuery, league, commissioner.Stream, 0)
	if err != nil {
		return nil, fmt.Errorf("read closing league records: %w", err)
	}
	events, err := scanEvents(rows)
	if err != nil {
		return nil, err
	}
	return commissionerRecords(events)
}

func (s *Store) closeRolloverBooks(ctx context.Context, tx *sql.Tx, raw commissioner.RawCommand) error {
	if raw.Command != commissioner.SeasonRollover {
		return nil
	}
	// Reading the new receipt here keeps the rollover and its derived state atomic.
	records, err := s.eventsInTransaction(ctx, tx, raw.League)
	if err != nil {
		return err
	}
	return saveClosedBooks(ctx, tx, records)
}

func latestClosedRevision(records []commissioner.Record) int64 {
	for i := len(records) - 1; i >= 0; i-- {
		if records[i].Command.Command == commissioner.SeasonRollover {
			return records[i].Receipt.Revision
		}
	}
	return 0
}
