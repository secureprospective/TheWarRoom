package domain

// Phase is the league's season phase, which gates the transactions legal at a point in the
// league-year. The rulebook scatters these boundaries (§5 season, §6 free agency, §14 Week-9
// deadline); only the three it justifies exist, and a finer phase is one constant plus one
// gate-map row.
//
// The loaded season is the season its offseason belongs to: OFFSEASON(N) → REGULAR_SEASON(N)
// → PLAYOFFS(N) → OFFSEASON(N+1). An offseason buyout therefore charges season N, the season
// it clears cap for. A fresh DB starts in OFFSEASON at the loaded season.
type Phase string

const (
	// PhaseOffseason is the contract window: buyouts (§12), tags (§9), extensions (§10),
	// restructures (§11) and free agency.
	PhaseOffseason Phase = "OFFSEASON"
	// PhaseRegularSeason is Weeks 1..13 (§5). No offseason-only op is legal here.
	PhaseRegularSeason Phase = "REGULAR_SEASON"
	// PhasePlayoffs is the postseason (§5).
	PhasePlayoffs Phase = "PLAYOFFS"
)

// Valid reports whether p is a known phase. A stored value that fails is drift; callers fail
// rather than gate on it.
func (p Phase) Valid() bool {
	switch p {
	case PhaseOffseason, PhaseRegularSeason, PhasePlayoffs:
		return true
	default:
		return false
	}
}
