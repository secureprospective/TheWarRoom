package output

import (
	"context"

	"github.com/secureprospective/TheWarRoom/internal/engine"
)

// ScoreRecord pairs an engine Result with its player id; the engine never sees ids. The
// Writer checks the id is unsigned digits and stores it verbatim, leading zeros included.
type ScoreRecord struct {
	MFLID  string
	Result engine.Result
}

// SeasonScore is one persisted score, flattened into columns so rankings can sort in SQL.
// AdjustedScore keeps full float64 precision because the L6 tiebreak tests exact equality.
// CreatedAt is informational and not part of the identity.
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

// Reader is the read-only surface for the ranking modules and IPC handlers.
type Reader interface {
	// Scores returns every score for one (season, scoring config) in ranking order.
	Scores(ctx context.Context, season, scoringConfigID int) ([]SeasonScore, error)
	// Score returns one player's persisted score for a (season, scoring config). ok is
	// false when no such record exists.
	Score(ctx context.Context, season, scoringConfigID int, mflID string) (SeasonScore, bool, error)
	// PriorRanks returns mflID -> 1-based rank from the latest earlier scoring config of this
	// season, for the rank-delta display. ok is false when there is no earlier scored config,
	// which is a normal first run, not an error.
	PriorRanks(ctx context.Context, season, beforeConfigID int) (map[string]int, bool, error)
}

// Writer is the append-only surface, injected only where scores are written. It embeds Reader
// so a writer can read what is already stored.
type Writer interface {
	Reader
	// Write appends one (season, scoring config) batch in a single transaction; see Store.Write.
	Write(ctx context.Context, season, scoringConfigID int, recs []ScoreRecord) error
}
