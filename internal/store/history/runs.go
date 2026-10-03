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
	"strings"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/engine"
	"github.com/secureprospective/TheWarRoom/internal/numeric"
	"github.com/secureprospective/TheWarRoom/internal/playerid"
)

// RunKind says what a scoring run is for. The board is the latest board run; a rebalance run is
// a proposal shown beside it and never changes it.
type RunKind string

const (
	RunBoard     RunKind = "board"
	RunRebalance RunKind = "rebalance"
)

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

// Score is one player's engine result in a run.
type Score struct {
	MFLID string
	engine.Result
}

// NewRun is a scoring run to write. Engine is the build that scored it; InputsHash identifies
// every engine input the run read.
type NewRun struct {
	Kind       RunKind
	Season     int
	AsOf       time.Time
	Params     ParamSet
	Engine     string
	InputsHash string
	Scores     []Score
}

// Run is a stored scoring run. MissingMeasures are the measures its param set reads that were
// not flowing when it ran; a board run with any is running on a reduced set.
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

// pendingRun is a NewRun checked and reduced to what identifies it.
type pendingRun struct {
	NewRun
	paramSetID       string
	params, measures []byte
	scoresHash       string
	missing          []string
}

// sameAs reports whether prev already holds this run.
func (p pendingRun) sameAs(prev Run) bool {
	return prev.ParamSetID == p.paramSetID && prev.Engine == p.Engine && prev.InputsHash == p.InputsHash &&
		prev.ScoresHash == p.scoresHash && slices.Equal(prev.MissingMeasures, p.missing)
}

// WriteRun stores a run and its scores. If the latest run of the same kind and season matches it
// in param set, engine, inputs, scores and missing measures, nothing is written and that run is
// returned with written false. The previous board is then always a different board.
func (s *Store) WriteRun(ctx context.Context, nr NewRun) (run Run, written bool, err error) {
	p, err := s.prepareRun(ctx, nr)
	if err != nil {
		return Run{}, false, err
	}
	s.wmu.Lock()
	defer s.wmu.Unlock()
	if prev, ok, err := s.LatestRun(ctx, nr.Season, nr.Kind); err != nil {
		return Run{}, false, err
	} else if ok && p.sameAs(prev) {
		return prev, false, nil
	}
	runID, err := s.insertRun(ctx, p)
	if err != nil {
		return Run{}, false, err
	}
	stored, _, err := s.run(ctx, `WHERE run_id = ?`, runID)
	return stored, true, err
}

// prepareRun validates nr and computes its param set, scores hash and missing measures.
func (s *Store) prepareRun(ctx context.Context, nr NewRun) (pendingRun, error) {
	switch {
	case nr.Kind != RunBoard && nr.Kind != RunRebalance:
		return pendingRun{}, fmt.Errorf("history: unknown run kind %q", nr.Kind)
	case len(nr.Scores) == 0:
		return pendingRun{}, fmt.Errorf("history: run for season %d has no scores", nr.Season)
	case nr.InputsHash == "" || nr.Engine == "":
		return pendingRun{}, fmt.Errorf("history: run for season %d needs its inputs hash and engine", nr.Season)
	}
	p := pendingRun{NewRun: nr}
	var err error
	if p.scoresHash, err = validScores(nr.Scores); err != nil {
		return pendingRun{}, err
	}
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
func (s *Store) insertRun(ctx context.Context, p pendingRun) (int64, error) {
	now := formatTime(s.now())
	missingJSON, err := json.Marshal(p.missing)
	if err != nil {
		return 0, fmt.Errorf("history: encode missing measures: %w", err)
	}
	tx, err := s.pools.Write().BeginTx(ctx, nil)
	if err != nil {
		return 0, fmt.Errorf("history: write run: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	if _, err := tx.ExecContext(ctx, `
INSERT OR IGNORE INTO param_sets (param_set_id, measures, params, created_at) VALUES (?, ?, ?, ?)`,
		p.paramSetID, string(p.measures), string(p.params), now); err != nil {
		return 0, fmt.Errorf("history: write param set: %w", err)
	}
	res, err := tx.ExecContext(ctx, `
INSERT INTO scoring_runs (kind, season, as_of, param_set_id, engine, inputs_hash, scores_hash,
	missing_measures, created_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		string(p.Kind), p.Season, formatTime(p.AsOf), p.paramSetID, p.Engine, p.InputsHash, p.scoresHash,
		string(missingJSON), now)
	if err != nil {
		return 0, fmt.Errorf("history: write run: %w", err)
	}
	runID, err := res.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("history: run id: %w", err)
	}
	for _, sc := range p.Scores {
		if _, err := tx.ExecContext(ctx, insertScoreSQL, scoreArgs(runID, sc)...); err != nil {
			return 0, fmt.Errorf("history: write score %s in run %d: %w", sc.MFLID, runID, err)
		}
	}
	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("history: write run commit: %w", err)
	}
	return runID, nil
}

// validScores rejects a malformed or repeated id, an unknown cap tier and any non-finite field,
// so a NaN never freezes into an append-only row. It returns the sha256 of the scores in id
// order.
func validScores(scores []Score) (string, error) {
	seen := make(map[string]bool, len(scores))
	for _, sc := range scores {
		id, err := playerid.New(sc.MFLID)
		if err != nil || id.String() != sc.MFLID {
			return "", fmt.Errorf("history: score id %q is not a canonical MFL id", sc.MFLID)
		}
		if seen[sc.MFLID] {
			return "", fmt.Errorf("history: %s is scored twice in one run", sc.MFLID)
		}
		seen[sc.MFLID] = true
		switch sc.CapTier {
		case engine.CapTierCold, engine.CapTierNeutral, engine.CapTierHot:
		default:
			return "", fmt.Errorf("history: %s has unknown cap tier %q", sc.MFLID, sc.CapTier)
		}
		l4 := sc.Layer4Output
		if !numeric.Finite(sc.BasePoints, sc.AgePull, l4.FilmEffective, l4.FilmRaw, l4.RASEffective,
			l4.BreakoutEffective, l4.Combined, sc.ScoutingAdjusted, sc.CapMultiplier, sc.AdjustedScore,
			sc.Tiebreaker.RAS) {
			return "", fmt.Errorf("history: %s has a non-finite score field", sc.MFLID)
		}
	}
	sorted := slices.Clone(scores)
	slices.SortFunc(sorted, func(a, b Score) int { return strings.Compare(a.MFLID, b.MFLID) })
	enc, err := json.Marshal(sorted)
	if err != nil {
		return "", fmt.Errorf("history: encode scores: %w", err)
	}
	sum := sha256.Sum256(enc)
	return hex.EncodeToString(sum[:]), nil
}

const scoreCols = `mfl_id, base_points, age_pull,
	film_effective, film_raw, ras_effective, breakout_effective, combined,
	scouting_adjusted, cap_multiplier, cap_tier, adjusted_score,
	tb_is_veteran, tb_ras, tb_scarcity_rank`

const insertScoreSQL = `INSERT INTO run_scores (run_id, ` + scoreCols + `)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

func scoreArgs(runID int64, sc Score) []any {
	l4 := sc.Layer4Output
	return []any{runID, sc.MFLID, sc.BasePoints, sc.AgePull,
		l4.FilmEffective, l4.FilmRaw, l4.RASEffective, l4.BreakoutEffective, l4.Combined,
		sc.ScoutingAdjusted, sc.CapMultiplier, string(sc.CapTier), sc.AdjustedScore,
		numeric.BoolToInt(sc.Tiebreaker.IsVeteran), sc.Tiebreaker.RAS, sc.Tiebreaker.ScarcityRank}
}

// LatestRun returns the newest run of a kind for a season. ok is false when there is none.
func (s *Store) LatestRun(ctx context.Context, season int, kind RunKind) (Run, bool, error) {
	return s.run(ctx, `WHERE season = ? AND kind = ? ORDER BY run_id DESC LIMIT 1`, season, string(kind))
}

// PreviousRun returns the newest run of the same kind and season before r. ok is false when r
// is the first.
func (s *Store) PreviousRun(ctx context.Context, r Run) (Run, bool, error) {
	return s.run(ctx, `WHERE season = ? AND kind = ? AND run_id < ? ORDER BY run_id DESC LIMIT 1`,
		r.Season, string(r.Kind), r.ID)
}

// Runs returns every run for a season, newest first.
func (s *Store) Runs(ctx context.Context, season int) ([]Run, error) {
	rows, err := s.pools.Read().QueryContext(ctx, runSelect+`WHERE season = ? ORDER BY run_id DESC`, season)
	if err != nil {
		return nil, fmt.Errorf("history: runs: %w", err)
	}
	defer func() { _ = rows.Close() }()
	var out []Run
	for rows.Next() {
		r, err := scanRun(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("history: runs rows: %w", err)
	}
	return out, nil
}

const runSelect = `SELECT run_id, kind, season, as_of, param_set_id, engine, inputs_hash, scores_hash,
	missing_measures, created_at FROM scoring_runs `

func (s *Store) run(ctx context.Context, where string, args ...any) (Run, bool, error) {
	r, err := scanRun(s.pools.Read().QueryRowContext(ctx, runSelect+where, args...))
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

// RunScores returns a run's scores in ranking order: the ORDER BY is the L6 tiebreaker
// (engine.TiebreakerKey.RanksAbove), with mfl_id last so the order is total.
func (s *Store) RunScores(ctx context.Context, runID int64) ([]Score, error) {
	return s.scores(ctx, `WHERE run_id = ?
ORDER BY adjusted_score DESC, tb_is_veteran DESC, tb_ras DESC, tb_scarcity_rank DESC, mfl_id ASC`, runID)
}

// RunScore returns one player's score in a run. ok is false when the run did not score them.
func (s *Store) RunScore(ctx context.Context, runID int64, mflID string) (Score, bool, error) {
	got, err := s.scores(ctx, `WHERE run_id = ? AND mfl_id = ?`, runID, mflID)
	if err != nil || len(got) == 0 {
		return Score{}, false, err
	}
	return got[0], true, nil
}

func (s *Store) scores(ctx context.Context, where string, args ...any) ([]Score, error) {
	rows, err := s.pools.Read().QueryContext(ctx, `SELECT `+scoreCols+` FROM run_scores `+where, args...)
	if err != nil {
		return nil, fmt.Errorf("history: scores: %w", err)
	}
	defer func() { _ = rows.Close() }()
	var out []Score
	for rows.Next() {
		var sc Score
		var tier string
		var vet int
		l4 := &sc.Layer4Output
		if err := rows.Scan(&sc.MFLID, &sc.BasePoints, &sc.AgePull,
			&l4.FilmEffective, &l4.FilmRaw, &l4.RASEffective, &l4.BreakoutEffective, &l4.Combined,
			&sc.ScoutingAdjusted, &sc.CapMultiplier, &tier, &sc.AdjustedScore,
			&vet, &sc.Tiebreaker.RAS, &sc.Tiebreaker.ScarcityRank); err != nil {
			return nil, fmt.Errorf("history: scan score: %w", err)
		}
		sc.CapTier = engine.CapTier(tier)
		sc.Tiebreaker.IsVeteran = vet != 0
		out = append(out, sc)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("history: scores rows: %w", err)
	}
	return out, nil
}
