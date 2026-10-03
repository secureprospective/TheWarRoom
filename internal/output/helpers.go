package output

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/secureprospective/TheWarRoom/internal/engine"
	"github.com/secureprospective/TheWarRoom/internal/numeric"
	sqlite "modernc.org/sqlite"
	sqlite3 "modernc.org/sqlite/lib"
)

// readerView holds a private *Store and implements only Reader, so it cannot be asserted back
// to a Writer.
type readerView struct{ s *Store }

func (r readerView) Scores(ctx context.Context, season, cfg int) ([]SeasonScore, error) {
	return r.s.Scores(ctx, season, cfg)
}

func (r readerView) Score(ctx context.Context, season, cfg int, mflID string) (SeasonScore, bool, error) {
	return r.s.Score(ctx, season, cfg, mflID)
}

func (r readerView) PriorRanks(ctx context.Context, season, before int) (map[string]int, bool, error) {
	return r.s.PriorRanks(ctx, season, before)
}

// PriorRanks ranks the highest scoring_config_id below before that has rows this season, in
// the same order the live board uses, so the delta is real and not a tiebreak artifact.
// Config ids are not dense (a version can exist unscored), so "before minus one" would be
// wrong.
//
// ok=false must render as absent, not as zero: zero says "held position", absent says "no
// earlier board".
func (s *Store) PriorRanks(ctx context.Context, season, before int) (map[string]int, bool, error) {
	var prior int
	// MAX over no rows returns one NULL row, so absence is detected by validity.
	var priorNull sql.NullInt64
	if err := s.pools.Read().QueryRowContext(ctx, `
SELECT MAX(scoring_config_id) FROM season_scores
WHERE season = ? AND scoring_config_id < ?`, season, before).Scan(&priorNull); err != nil {
		return nil, false, fmt.Errorf("output: find prior scoring config: %w", err)
	}
	if !priorNull.Valid {
		return nil, false, nil
	}
	prior = int(priorNull.Int64)

	rows, err := s.pools.Read().QueryContext(ctx, `
SELECT mfl_id FROM season_scores
WHERE season = ? AND scoring_config_id = ?
ORDER BY adjusted_score DESC, mfl_id`, season, prior)
	if err != nil {
		return nil, false, fmt.Errorf("output: read prior board: %w", err)
	}
	defer func() { _ = rows.Close() }()

	ranks := make(map[string]int)
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, false, fmt.Errorf("output: scan prior board: %w", err)
		}
		ranks[id] = len(ranks) + 1 // 1-based, in the same order the live board ranks
	}
	if err := rows.Err(); err != nil {
		return nil, false, fmt.Errorf("output: prior board rows: %w", err)
	}
	if len(ranks) == 0 {
		return nil, false, nil // config id existed but yielded nothing — same as no prior
	}
	return ranks, true, nil
}

// insertSQL appends one season_scores row in rowValues.args order.
const insertSQL = `
INSERT INTO season_scores (
	season, scoring_config_id, mfl_id,
	base_points, age_pull,
	film_effective, film_raw, ras_effective, breakout_effective, combined,
	scouting_adjusted, cap_multiplier, cap_tier, adjusted_score,
	tb_is_veteran, tb_ras, tb_scarcity_rank, created_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

// selectCols lists the columns scanScores reads, in scan order.
const selectCols = `
SELECT season, scoring_config_id, mfl_id,
	base_points, age_pull,
	film_effective, film_raw, ras_effective, breakout_effective, combined,
	scouting_adjusted, cap_multiplier, cap_tier, adjusted_score,
	tb_is_veteran, tb_ras, tb_scarcity_rank, created_at
FROM season_scores`

// rowValues is a validated, persist-ready record. newRowValues is the only way to build one.
type rowValues struct {
	season, scoringConfigID int
	mflID                   string
	r                       engine.Result
	createdAt               string
}

// newRowValues rejects a malformed id, an unknown cap tier and any non-finite score field.
func newRowValues(season, scoringConfigID int, rec ScoreRecord, now string) (rowValues, error) {
	if err := validMFLID(rec.MFLID); err != nil {
		return rowValues{}, err
	}
	if !validCapTier(rec.Result.CapTier) {
		return rowValues{}, fmt.Errorf("output: %q has unknown cap tier %q", rec.MFLID, rec.Result.CapTier)
	}
	l4 := rec.Result.Layer4Output
	if !numeric.Finite(
		rec.Result.BasePoints, rec.Result.AgePull,
		l4.FilmEffective, l4.FilmRaw, l4.RASEffective, l4.BreakoutEffective, l4.Combined,
		rec.Result.ScoutingAdjusted, rec.Result.CapMultiplier, rec.Result.AdjustedScore,
		rec.Result.Tiebreaker.RAS,
	) {
		return rowValues{}, fmt.Errorf("output: %q has a non-finite score field (NaN/Inf)", rec.MFLID)
	}
	return rowValues{
		season:          season,
		scoringConfigID: scoringConfigID,
		mflID:           rec.MFLID,
		r:               rec.Result,
		createdAt:       now,
	}, nil
}

// args returns the INSERT placeholder values in insertSQL column order.
func (rv rowValues) args() []any {
	l4 := rv.r.Layer4Output
	return []any{
		rv.season, rv.scoringConfigID, rv.mflID,
		rv.r.BasePoints, rv.r.AgePull,
		l4.FilmEffective, l4.FilmRaw, l4.RASEffective, l4.BreakoutEffective, l4.Combined,
		rv.r.ScoutingAdjusted, rv.r.CapMultiplier, string(rv.r.CapTier), rv.r.AdjustedScore,
		numeric.BoolToInt(rv.r.Tiebreaker.IsVeteran), rv.r.Tiebreaker.RAS, rv.r.Tiebreaker.ScarcityRank,
		rv.createdAt,
	}
}

// scanScores reads a season_scores result set into SeasonScore values, in selectCols order.
func scanScores(rows *sql.Rows) ([]SeasonScore, error) {
	var out []SeasonScore
	for rows.Next() {
		var s SeasonScore
		var tier string
		var vet int
		if err := rows.Scan(
			&s.Season, &s.ScoringConfigID, &s.MFLID,
			&s.BasePoints, &s.AgePull,
			&s.Layer4Output.FilmEffective, &s.Layer4Output.FilmRaw, &s.Layer4Output.RASEffective,
			&s.Layer4Output.BreakoutEffective, &s.Layer4Output.Combined,
			&s.ScoutingAdjusted, &s.CapMultiplier, &tier, &s.AdjustedScore,
			&vet, &s.Tiebreaker.RAS, &s.Tiebreaker.ScarcityRank, &s.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("output: scan: %w", err)
		}
		s.CapTier = engine.CapTier(tier)
		s.Tiebreaker.IsVeteran = vet != 0
		out = append(out, s)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("output: iterate: %w", err)
	}
	return out, nil
}

// validMFLID checks form only: non-empty, unsigned digits. Ids arrive canonical from playerid;
// this is the last check before an immutable write.
func validMFLID(id string) error {
	if id == "" {
		return fmt.Errorf("output: empty mfl id")
	}
	for _, c := range id {
		if c < '0' || c > '9' {
			return fmt.Errorf("output: mfl id %q is not unsigned digits", id)
		}
	}
	return nil
}

// validCapTier accepts only the three L5 cap tiers the engine emits.
func validCapTier(t engine.CapTier) bool {
	switch t {
	case engine.CapTierCold, engine.CapTierNeutral, engine.CapTierHot:
		return true
	default:
		return false
	}
}

// isUniqueViolation matches modernc's typed PRIMARY KEY / UNIQUE result codes. A message
// substring would also match CHECK and NOT NULL failures and misreport them as drift.
func isUniqueViolation(err error) bool {
	var se *sqlite.Error
	if !errors.As(err, &se) {
		return false
	}
	switch se.Code() {
	case sqlite3.SQLITE_CONSTRAINT_PRIMARYKEY, sqlite3.SQLITE_CONSTRAINT_UNIQUE:
		return true
	default:
		return false
	}
}
