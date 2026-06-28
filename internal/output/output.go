// Package output is the B6 Per-Season Output Store: the FIRST persistence layer for
// engine OUTPUT. The three prior Layer-2 stores hold config (rulebook B3b, params B4)
// or mutable runtime state (state B3c); this one holds the scores the pure engine
// pipeline (B5a) computes but never persists. It clones the store template — a
// Reader/Writer split enforcing the single-writer law (AD-02/AD-05), db.Pools read/write
// separation (MaxOpenConns=1 on the write pool), the two-lock idiom (wmu outer write
// mutex + mu reader RWMutex), parameterized SQL, and fail-loud on drift — but adds the
// mechanic that DEFINES it: DOUBLE immutability (AD-04, DECISION-010).
//
//   - DECISION-010: a season's scores are frozen under the scoring config they were
//     computed with. Each record carries a scoring_config_id (stamped by the caller from
//     B3b's active version, never minted here). Re-tuning params produces NEW records
//     under a new id; old records are never re-scored.
//   - AD-04: immutability is enforced BOTH ways. The Go API is APPEND-ONLY (the Writer
//     has no Update/Delete method), and a SQLite BEFORE UPDATE/DELETE trigger RAISE(ABORT)s
//     so the database itself rejects mutation even if a future caller bypasses the Go API.
//
// It is PURE DATA ACCESS: it persists and serves engine.Result projections and runs no
// scoring logic. It imports the engine ONLY for the pure Result/Layer4Output/Tiebreaker
// value types; it imports NO other store (depguard store-output-no-siblings).
package output

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/db"
)

// ErrDuplicate is returned when a Write would re-persist an existing
// (season, scoring_config_id, mfl_id). Append-only means a duplicate key is drift, not
// an upsert — fail loud rather than silently overwrite a frozen score. It is EXPORTED so
// the caller can react to drift specifically via errors.Is, rather than string-matching a
// wrapped message (GLM review).
var ErrDuplicate = errors.New("output: score already persisted for this (season, scoring_config_id, mfl_id)")

// Store is the per-season output store. Construct with New; prepare with Initialize.
// Appends serialize under wmu so each batch insert is one atomic step (the single-writer
// half of the store template). UNLIKE B3c/B4 there is NO reader RWMutex: this store keeps
// no in-memory cache — reads go straight to the read pool (sql.DB is concurrency-safe),
// and the DB is the durable, trigger-immutable source of truth. There is no shared memory
// for a second lock to guard (GLM review — the second lock would be cargo-culted here).
type Store struct {
	pools *db.Pools

	wmu sync.Mutex // serializes appends end to end (one writer)
}

// New constructs an unprepared store over the given pools. Call Initialize before any
// read or write.
func New(pools *db.Pools) *Store {
	return &Store{pools: pools}
}

// Writer returns the append-only mutation surface. It is intended for the
// score/transaction layer's dependency injection ONLY — wire it exactly where the single
// writer lives, nowhere else.
func (s *Store) Writer() Writer { return s }

// Reader returns the read-only surface for the ranking modules and IPC handlers. The
// returned value does NOT embed *Store, so a consumer cannot recover the Writer by type
// assertion — the boundary is real, not a naming convention (proven by a planted test).
func (s *Store) Reader() Reader { return readerView{s: s} }

// Initialize ensures the schema and the immutability triggers exist. It is idempotent
// (CREATE ... IF NOT EXISTS) and seeds nothing — an output store starts empty and grows
// only by Write.
func (s *Store) Initialize(ctx context.Context) error {
	s.wmu.Lock()
	defer s.wmu.Unlock()
	return s.initSchema(ctx)
}

// initSchema creates the season_scores table plus the two BEFORE UPDATE/DELETE triggers
// that make rows immutable at the database level (AD-04). The (season, scoring_config_id,
// mfl_id) primary key is the DECISION-010 identity: a new config writes new rows, never
// touches old ones; a duplicate key is a constraint error the Writer reports as drift.
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

// Write appends a batch of scores for one (season, scoring config) in a single
// transaction. It validates EVERY record up front — id well-formed, cap tier known, all
// score fields finite — before any insert, so a malformed batch rolls back whole (the
// engine multiplies straight into rankings; a single non-finite field would round-trip
// into a frozen, immutable record). A duplicate key surfaces as errDuplicate.
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
		// A duplicate id WITHIN one batch is a self-contradictory input, distinct from
		// drift against already-persisted rows — report it as such, not as ErrDuplicate.
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

// Scores returns every persisted score for one (season, scoring config) in final ranking
// order. The ORDER BY encodes the L6 tiebreaker (engine.TiebreakerKey.RanksAbove) the
// store persists for exactly this purpose: AdjustedScore desc, then veteran, then RAS,
// then positional scarcity — with mfl id as the final stable key so the order is fully
// deterministic even when the whole tiebreaker key ties (GLM review — an arbitrary mfl-id
// tiebreak would ignore the persisted L6 fields).
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
