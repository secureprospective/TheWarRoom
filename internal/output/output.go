// Package output persists engine scores. A season's scores are frozen under the scoring config
// they were computed with: each row carries a scoring_config_id stamped by the caller from the
// rulebook's active version, and re-tuning writes new rows under a new id rather than rescoring
// old ones.
//
// Rows are immutable twice over: the Writer has no update or delete method, and SQLite triggers
// abort any UPDATE or DELETE that bypasses it. The package imports no other store.
package output

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/db"
)

// ErrDuplicate means a Write tried to re-persist an existing (season, scoring_config_id,
// mfl_id). Append-only makes that drift, not an upsert.
var ErrDuplicate = errors.New("output: score already persisted for this (season, scoring_config_id, mfl_id)")

// Store is the output store. Construct with New, prepare with Initialize. Writes serialize
// under wmu. There is no in-memory cache, so reads need no lock.
type Store struct {
	pools *db.Pools

	wmu sync.Mutex // serializes appends end to end (one writer)
}

// New constructs an unprepared store over the given pools. Call Initialize before any
// read or write.
func New(pools *db.Pools) *Store {
	return &Store{pools: pools}
}

// Writer returns the append-only surface. Wire it only where the single writer lives.
func (s *Store) Writer() Writer { return s }

// Reader returns the read-only surface. The value does not embed *Store, so it cannot be
// type-asserted back to a Writer.
func (s *Store) Reader() Reader { return readerView{s: s} }

// Initialize creates the table and the immutability triggers if absent. It seeds nothing.
func (s *Store) Initialize(ctx context.Context) error {
	s.wmu.Lock()
	defer s.wmu.Unlock()
	return s.initSchema(ctx)
}

// initSchema creates season_scores and the triggers that make its rows immutable. The primary
// key (season, scoring_config_id, mfl_id) means a new config writes new rows and a repeat is a
// constraint error.
func (s *Store) initSchema(ctx context.Context) error {
	const ddl = `
CREATE TABLE IF NOT EXISTS season_scores (
	season            INTEGER NOT NULL,
	scoring_config_id INTEGER NOT NULL,
	mfl_id            TEXT NOT NULL,
	base_points       REAL NOT NULL,
	age_pull          REAL NOT NULL,
	film_effective    REAL NOT NULL,
	film_raw          REAL NOT NULL,
	ras_effective     REAL NOT NULL,
	breakout_effective REAL NOT NULL,
	combined          REAL NOT NULL,
	scouting_adjusted REAL NOT NULL,
	cap_multiplier    REAL NOT NULL,
	cap_tier          TEXT NOT NULL,
	adjusted_score    REAL NOT NULL,
	tb_is_veteran     INTEGER NOT NULL,
	tb_ras            REAL NOT NULL,
	tb_scarcity_rank  INTEGER NOT NULL,
	created_at        TEXT NOT NULL,
	PRIMARY KEY (season, scoring_config_id, mfl_id)
);
CREATE TRIGGER IF NOT EXISTS season_scores_no_update
BEFORE UPDATE ON season_scores
BEGIN
	SELECT RAISE(ABORT, 'season_scores is append-only: UPDATE forbidden (AD-04)');
END;
CREATE TRIGGER IF NOT EXISTS season_scores_no_delete
BEFORE DELETE ON season_scores
BEGIN
	SELECT RAISE(ABORT, 'season_scores is append-only: DELETE forbidden (AD-04)');
END;`
	if _, err := s.pools.Write().ExecContext(ctx, ddl); err != nil {
		return fmt.Errorf("output: init schema: %w", err)
	}
	return nil
}

// Write appends one (season, scoring config) batch in a single transaction. Every record is
// validated before any insert, so one bad record rolls back the batch instead of freezing a
// NaN into an immutable row. A repeat key returns ErrDuplicate.
func (s *Store) Write(ctx context.Context, season, scoringConfigID int, recs []ScoreRecord) error {
	if len(recs) == 0 {
		return fmt.Errorf("output: Write got no records for season %d config %d", season, scoringConfigID)
	}
	rows := make([]rowValues, len(recs))
	seen := make(map[string]struct{}, len(recs))
	now := time.Now().UTC().Format(time.RFC3339)
	for i, rec := range recs {
		rv, err := newRowValues(season, scoringConfigID, rec, now)
		if err != nil {
			return err
		}
		// A repeat id inside one batch is bad input, not drift against stored rows.
		if _, dup := seen[rv.mflID]; dup {
			return fmt.Errorf("output: duplicate mfl id %q within one Write batch (season %d config %d)",
				rv.mflID, season, scoringConfigID)
		}
		seen[rv.mflID] = struct{}{}
		rows[i] = rv
	}

	s.wmu.Lock()
	defer s.wmu.Unlock()

	tx, err := s.pools.Write().BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("output: write begin: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	for _, rv := range rows {
		if _, err := tx.ExecContext(ctx, insertSQL, rv.args()...); err != nil {
			if isUniqueViolation(err) {
				return fmt.Errorf("output: write %q (season %d config %d): %w",
					rv.mflID, season, scoringConfigID, ErrDuplicate)
			}
			return fmt.Errorf("output: write %q: %w", rv.mflID, err)
		}
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("output: write commit: %w", err)
	}
	return nil
}

// Scores returns one (season, scoring config) in ranking order: the ORDER BY is the L6
// tiebreaker (engine.TiebreakerKey.RanksAbove), with mfl_id last so the order is total.
func (s *Store) Scores(ctx context.Context, season, scoringConfigID int) ([]SeasonScore, error) {
	rows, err := s.pools.Read().QueryContext(ctx, selectCols+`
WHERE season = ? AND scoring_config_id = ?
ORDER BY adjusted_score DESC, tb_is_veteran DESC, tb_ras DESC, tb_scarcity_rank DESC, mfl_id ASC`,
		season, scoringConfigID)
	if err != nil {
		return nil, fmt.Errorf("output: scores query: %w", err)
	}
	defer func() { _ = rows.Close() }()
	return scanScores(rows)
}

// Score returns one player's persisted score for a (season, scoring config). ok is false
// when no such record exists.
func (s *Store) Score(ctx context.Context, season, scoringConfigID int, mflID string) (SeasonScore, bool, error) {
	rows, err := s.pools.Read().QueryContext(ctx, selectCols+`
WHERE season = ? AND scoring_config_id = ? AND mfl_id = ?`, season, scoringConfigID, mflID)
	if err != nil {
		return SeasonScore{}, false, fmt.Errorf("output: score query: %w", err)
	}
	defer func() { _ = rows.Close() }()
	got, err := scanScores(rows)
	if err != nil {
		return SeasonScore{}, false, err
	}
	if len(got) == 0 {
		return SeasonScore{}, false, nil
	}
	return got[0], true, nil
}
