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

// readerView is the concrete type Store.Reader hands out. It embeds NO *Store, only a
// private pointer, and implements ONLY Reader by delegation — so a consumer cannot
// type-assert it back to Writer. This is the compile/runtime half of the single-writer
// law (proven by a planted test); the DB triggers are the other half (AD-04).
type readerView struct{ s *Store }

func (r readerView) Scores(ctx context.Context, season, cfg int) ([]SeasonScore, error) {
	return r.s.Scores(ctx, season, cfg)
}

func (r readerView) Score(ctx context.Context, season, cfg int, mflID string) (SeasonScore, bool, error) {
	return r.s.Score(ctx, season, cfg, mflID)
}

// insertSQL appends one season_scores row. The column order matches rowValues.args.
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

// rowValues is a validated, persist-ready projection of one ScoreRecord. Building it via
// newRowValues is the single validation gate — every field that reaches the DB passes
// through here.
type rowValues struct {
	season, scoringConfigID int
	mflID                   string
	r                       engine.Result
	createdAt               string
}

// newRowValues validates one record and returns its persist-ready form. It rejects a
// malformed id (must be unsigned digits — the playerid signed-string gap, audit #4), an
// unknown cap tier, and any non-finite score field (the engine multiplies straight into
// rankings; a NaN/Inf would freeze into an immutable record — audit #1, L5 does not
// re-check accumulator finiteness).
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
		mflID:           rec.MFLID, // persisted verbatim (audit #2)
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

// validMFLID is a FORM gate: a non-empty, UNSIGNED, all-digit id. It is not a full MFL-id
// validity check (it accepts "0" and unbounded-length digit strings) — canonical ids
// arrive from playerid upstream; this guard exists because playerid.New would accept a
// leading '-' (strconv.Atoi parses signed), the audit #4 gap. The value is persisted
// verbatim, not re-canonicalized.
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

// isUniqueViolation reports whether err is a PRIMARY KEY / UNIQUE constraint failure —
// the duplicate-key drift signal. It matches modernc's TYPED extended result code, not a
// message substring: a substring like "constraint failed" also matches CHECK/NOT NULL/FK
// failures, which would misclassify an unrelated write error as drift (GLM review).
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
