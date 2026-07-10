package domain

// Phase is the league's SEASON PHASE — the coarse calendar state that gates which
// transactions are legal at a given point in a league-year (Vision-2026 D3). The rulebook
// does not enumerate named phases; it scatters temporal boundaries (§5 season structure,
// §14 Week-9 trade deadline, §6 free-agency windows). v1 seeds only the three phases the
// rulebook justifies — the machinery (append-only transitions + a declarative op→phase
// gate) is built to take finer phases later as one constant + one gate-map row, with no
// schema change.
//
// The load-bearing invariant (locked 2026-07-10, expert-panel unanimous): the loaded
// season int is the season the OFFSEASON belongs to — offseason sits at the START of its
// season's lifecycle. Cycle: OFFSEASON(N) → REGULAR_SEASON(N) → PLAYOFFS(N) → [rollover to
// N+1] → OFFSEASON(N+1). So an offseason buyout counts against, and charges dead cap to,
// season N — the upcoming managed season it clears cap for. Season-rollover machinery is a
// separate carry-forward (shared with §11's in-season restructure unlock); v1 correctness
// holds because a fresh DB is seeded in OFFSEASON at the loaded season int.
type Phase string

const (
	// PhaseOffseason is the contract-management window: buyouts (§12), tags (§9),
	// extensions (§10), restructures (§11), and free agency happen here.
	PhaseOffseason Phase = "OFFSEASON"
	// PhaseRegularSeason is Weeks 1..13 (§5). No offseason-only op is legal here.
	PhaseRegularSeason Phase = "REGULAR_SEASON"
	// PhasePlayoffs is the postseason (§5).
	PhasePlayoffs Phase = "PLAYOFFS"
)

// Valid reports whether p is one of the known season phases. A value read back from
// storage that fails this is drift — callers fail loud rather than gate on an unknown
// phase (an unrecognized phase must never silently allow or deny an op).
func (p Phase) Valid() bool {
	switch p {
	case PhaseOffseason, PhaseRegularSeason, PhasePlayoffs:
		return true
	default:
		return false
	}
}
