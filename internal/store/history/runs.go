package history

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"time"
)

// RunKind says what a scoring run is for. The model run is the only kind written now; the board
// and rebalance runs that existing databases hold in scoring_runs are no longer read.
type RunKind string

// RunModel is the measurables (internal/model), kept in model_runs.
const RunModel RunKind = "model"

// ParamSet is the parameters a run scored with, and the measures its model reads.
type ParamSet struct {
	Params   map[string]float64
	Measures []string
}

// canonical encodes the set (params keys sorted, measures sorted and unique) and returns its id,
// the sha256 of that encoding.
func (p ParamSet) canonical() (id string, params, measures []byte, err error) {
	ms := slices.Clone(p.Measures)
	slices.Sort(ms)
	ms = slices.Compact(ms)
	if ms == nil {
		ms = []string{}
	}
	if measures, err = json.Marshal(ms); err != nil {
		return "", nil, nil, fmt.Errorf("history: encode param set measures: %w", err)
	}
	if params, err = json.Marshal(p.Params); err != nil {
		return "", nil, nil, fmt.Errorf("history: encode param set: %w", err)
	}
	sum := sha256.Sum256(append(append(measures, '\n'), params...))
	return hex.EncodeToString(sum[:]), params, measures, nil
}

// Run is a stored scoring run. MissingMeasures are the measures its param set reads that were
// not flowing when it ran.
type Run struct {
	ID              int64
	Kind            RunKind
	Season          int
	AsOf            time.Time
	ParamSetID      string
	Engine          string
	InputsHash      string
	ScoresHash      string
	MissingMeasures []string
	CreatedAt       time.Time
}

// pendingRun is a NewModelRun checked and reduced to what identifies it.
type pendingRun struct {
	refresh          bool
	season           int
	asOf             time.Time
	engine           string
	inputsHash       string
	paramSetID       string
	params, measures []byte
	scoresHash       string
	missing          []string
}

// sameAs reports whether prev already holds this run.
func (p pendingRun) sameAs(prev Run) bool {
	return !p.refresh && p.asOf.Sub(prev.AsOf) < 24*time.Hour && prev.ParamSetID == p.paramSetID && prev.Engine == p.engine && prev.InputsHash == p.inputsHash &&
		prev.ScoresHash == p.scoresHash && slices.Equal(prev.MissingMeasures, p.missing)
}

// writeRun stores p unless the latest run of its kind and season already holds it; scores writes
// its score rows inside the run's transaction.
func (s *Store) writeRun(ctx context.Context, p pendingRun, scores func(tx *sql.Tx, runID int64) error) (Run, bool, error) {
	s.wmu.Lock()
	defer s.wmu.Unlock()
	if prev, ok, err := s.LatestRun(ctx, p.season, RunModel); err != nil {
		return Run{}, false, err
	} else if ok && p.sameAs(prev) {
		return prev, false, nil
	}
	runID, err := s.insertRun(ctx, p, scores)
	if err != nil {
		return Run{}, false, err
	}
	stored, _, err := s.run(ctx, `WHERE run_id = ?`, runID)
	return stored, true, err
}

// pending reduces a checked run to what identifies it: its param set and the measures it reads
// that are not flowing.
func (s *Store) pending(ctx context.Context, nr NewModelRun, scoresHash string) (pendingRun, error) {
	p := pendingRun{refresh: nr.Refresh, season: nr.Season, asOf: nr.AsOf, engine: nr.Engine,
		inputsHash: nr.InputsHash, scoresHash: scoresHash}
	var err error
	if p.paramSetID, p.params, p.measures, err = nr.Params.canonical(); err != nil {
		return pendingRun{}, err
	}
	flowing, err := s.FlowingMeasures(ctx)
	if err != nil {
		return pendingRun{}, err
	}
	var need []string
	if err := json.Unmarshal(p.measures, &need); err != nil {
		return pendingRun{}, fmt.Errorf("history: decode measures: %w", err)
	}
	p.missing = missingFrom(need, flowing)
	return p, nil
}

// insertRun writes the param set, the run and its scores in one transaction. The caller holds wmu.
func (s *Store) insertRun(ctx context.Context, p pendingRun, scores func(tx *sql.Tx, runID int64) error) (int64, error) {
	now := formatTime(s.now())
	missingJSON, err := json.Marshal(p.missing)
	if err != nil {
		return 0, fmt.Errorf("history: encode missing measures: %w", err)
	}
	tx, err := s.pools.BeginWrite(ctx)
	if err != nil {
		return 0, fmt.Errorf("history: write run: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	if _, err := tx.ExecContext(ctx, `
INSERT OR IGNORE INTO param_sets (param_set_id, measures, params, created_at) VALUES (?, ?, ?, ?)`,
		p.paramSetID, string(p.measures), string(p.params), now); err != nil {
		return 0, fmt.Errorf("history: write param set: %w", err)
	}
	res, err := tx.ExecContext(ctx, `INSERT INTO model_runs (kind, season, as_of, param_set_id, engine,
	inputs_hash, scores_hash, missing_measures, created_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		string(RunModel), p.season, formatTime(p.asOf), p.paramSetID, p.engine, p.inputsHash, p.scoresHash,
		string(missingJSON), now)
	if err != nil {
		return 0, fmt.Errorf("history: write run: %w", err)
	}
	runID, err := res.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("history: run id: %w", err)
	}
	if err := scores(tx, runID); err != nil {
		return 0, err
	}
	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("history: write run commit: %w", err)
	}
	return runID, nil
}

// LatestRun returns the newest run of a kind for a season. ok is false when there is none, or
// when no store is open yet (a fresh profile).
func (s *Store) LatestRun(ctx context.Context, season int, kind RunKind) (Run, bool, error) {
	if s == nil {
		return Run{}, false, nil
	}
	return s.run(ctx, `WHERE season = ? AND kind = ? ORDER BY run_id DESC LIMIT 1`, season, string(kind))
}

// PreviousRun returns the newest run of the same kind and season before r. ok is false when r
// is the first.
func (s *Store) PreviousRun(ctx context.Context, r Run) (Run, bool, error) {
	return s.run(ctx, `WHERE season = ? AND kind = ? AND run_id < ? ORDER BY run_id DESC LIMIT 1`,
		r.Season, string(r.Kind), r.ID)
}

func (s *Store) run(ctx context.Context, where string, args ...any) (Run, bool, error) {
	r, err := scanRun(s.pools.Read().QueryRowContext(ctx, `SELECT run_id, kind, season, as_of, param_set_id, engine,
	inputs_hash, scores_hash, missing_measures, created_at FROM model_runs `+where, args...))
	if errors.Is(err, sql.ErrNoRows) {
		return Run{}, false, nil
	}
	if err != nil {
		return Run{}, false, err
	}
	return r, true, nil
}

func scanRun(row interface{ Scan(dest ...any) error }) (Run, error) {
	var r Run
	var kind, asOf, missing, created string
	if err := row.Scan(&r.ID, &kind, &r.Season, &asOf, &r.ParamSetID, &r.Engine, &r.InputsHash, &r.ScoresHash,
		&missing, &created); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Run{}, err //nolint:wrapcheck // callers test for sql.ErrNoRows
		}
		return Run{}, fmt.Errorf("history: scan run: %w", err)
	}
	r.Kind = RunKind(kind)
	var err error
	if r.AsOf, err = parseTime(asOf); err != nil {
		return Run{}, err
	}
	if r.CreatedAt, err = parseTime(created); err != nil {
		return Run{}, err
	}
	if err := json.Unmarshal([]byte(missing), &r.MissingMeasures); err != nil {
		return Run{}, fmt.Errorf("history: run %d missing measures: %w", r.ID, err)
	}
	return r, nil
}
