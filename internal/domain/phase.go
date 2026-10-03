package domain

// Phase is the league's season phase, which gates the transactions legal at a point in the
// league-year. Only the three phases the rulebook justifies exist (§5, §6, §14); a finer one
// is a constant plus a gate-map row.
//
// The loaded season is the season its offseason belongs to: OFFSEASON(N) → REGULAR_SEASON(N)
// → PLAYOFFS(N) → OFFSEASON(N+1), so an offseason buyout charges the season it clears cap for.
// A fresh DB starts in OFFSEASON at the loaded season.
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
