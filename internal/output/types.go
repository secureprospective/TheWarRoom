package output

import (
	"context"

	"github.com/secureprospective/TheWarRoom/internal/engine"
)

// ScoreRecord pairs a player id with the engine Result the pipeline produced for him.
// engine.Result is position/score data only — it carries NO player id (the engine is
// pure and never sees an mfl id), so the caller supplies the id alongside each Result
// at write time. MFLID is validated as an unsigned-digit MFL id by the Writer before
// it is persisted verbatim (TEXT, leading zeros preserved).
type ScoreRecord struct {
	MFLID  string
	Result engine.Result
}

// SeasonScore is one PERSISTED engine output: a flattened, queryable projection of
// engine.Result tagged by (Season, ScoringConfigID, MFLID). The scalars are stored in
// their own columns (not a JSON blob) so M1 can rank by AdjustedScore in SQL; the
// AdjustedScore is held at full float64 precision because the L6 tiebreak depends on
// exact equality. Confidence values are NOT present — a Hard Constraint already honored
// by engine.Result. CreatedAt records when the score was written, never participates in
// identity, and is purely informational.
type SeasonScore struct {
	Season          int
	ScoringConfigID int
	MFLID           string

	BasePoints   float64
	AgePull      float64
	Layer4Output engine.Layer4Output

	ScoutingAdjusted float64
	CapMultiplier    float64
	CapTier          engine.CapTier
	AdjustedScore    float64
	Tiebreaker       engine.TiebreakerKey

	CreatedAt string
}

// Reader is the READ-ONLY surface handed to the ranking modules (M1) and IPC handlers.
// It exposes no write path — a consumer holding a Reader cannot persist or mutate a
// score, even by type assertion (Store.Reader returns a wrapper that does not embed
// *Store). This is the read half of the single-writer law (AD-02/AD-05) for the output
// store; on top of it, the records themselves are immutable both ways (AD-04).
type Reader interface {
	// Scores returns every persisted score for one (season, scoring config), ordered by
	// AdjustedScore descending then mfl id — the deterministic ranking order M1 reads.
	Scores(ctx context.Context, season, scoringConfigID int) ([]SeasonScore, error)
	// Score returns one player's persisted score for a (season, scoring config). ok is
	// false when no such record exists.
	Score(ctx context.Context, season, scoringConfigID int, mflID string) (SeasonScore, bool, error)
	// PriorRanks returns the ranking positions of the most recent EARLIER scoring config
	// for this season, as mflID → 1-based rank. It backs the §1 rank-delta display: the
	// board is append-only per (season, config), so a previous config's rows ARE the
	// previous board — no separate history table is needed to know where a player ranked
	// before. ok is false when no earlier scored config exists (the first-ever run), which
	// is a legitimate "no delta to show" and NOT an error.
	PriorRanks(ctx context.Context, season, beforeConfigID int) (map[string]int, bool, error)
}

// Writer is the APPEND-ONLY mutation surface, injected to the score/transaction layer
// ONLY (mirrors B3c's StateWriter DI). It deliberately exposes NO Update or Delete —
// immutability is enforced in the Go API by the ABSENCE of those methods AND in the DB
// by a BEFORE UPDATE/DELETE trigger that ABORTs (AD-04, double immutability). It embeds
// Reader because a writer may read what is already persisted before appending.
type Writer interface {
	Reader
	// Write appends a batch of scores for one (season, scoring config) in a single
	// transaction. ScoringConfigID is STAMPED by the caller from the rulebook active
	// version (B3b) — the output store never mints it (DECISION-010: a record is frozen
	// under the config it was scored with). A re-write of an existing
	// (season, scoring_config_id, mfl_id) is rejected as drift (append-only, not upsert);
	// a non-finite score field or a malformed id fails loud and rolls back the whole batch.
	Write(ctx context.Context, season, scoringConfigID int, recs []ScoreRecord) error
}
