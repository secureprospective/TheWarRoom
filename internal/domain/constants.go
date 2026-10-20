// Package domain holds the typed records and value sets the engine and stores consume. It is a
// leaf: it imports only playerid. Mapping from raw MFL lives in normalize.
package domain

// Position is the normalized engine position set; normalize maps MFL's raw codes onto it.
// PosFlag marks an unclassified record an admin must resolve before scoring.
type Position string

const (
	PosQB   Position = "QB"
	PosRB   Position = "RB"
	PosWR   Position = "WR"
	PosTE   Position = "TE"
	PosK    Position = "K"  // MFL "PK" normalizes here
	PosDE   Position = "DE" // MFL "EDGE" maps here (OQ-004)
	PosDT   Position = "DT"
	PosLB   Position = "LB"
	PosCB   Position = "CB"
	PosS    Position = "S"
	PosFlag Position = "FLAG" // MFL "XX" or unknown
)

// ContractStatus is the normalized contract state. MFL's field is dirty; CStatusFlag marks a
// value no known prefix matched.
type ContractStatus string

const (
	CStatusUFA  ContractStatus = "UFA"
	CStatusRFA  ContractStatus = "RFA"
	CStatusFT1  ContractStatus = "FT1"
	CStatusFT2  ContractStatus = "FT2"
	CStatusFlag ContractStatus = "FLAG"
)

// RosterStatus is where a player sits on a franchise.
type RosterStatus string

const (
	RosterActive RosterStatus = "ROSTER"
	RosterTaxi   RosterStatus = "TAXI_SQUAD"
	RosterIR     RosterStatus = "IR"
)

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

// PlayerStatus is an off-roster player's availability. ReleasePlayer serves waiver cuts (§8),
// buyouts (§12), retirement and death (§13) and rollover expiry (§14), which all leave the same
// database footprint; without this marker a retired player would look like a signable free
// agent. Status is an append-only event log (player_status_events); the latest row is current.
type PlayerStatus string

const (
	// PlayerFreeAgent is signable. A buyout also lands here; its "no re-bid until next
	// offseason" lockout is derived from the dead_cap row, not stored.
	PlayerFreeAgent PlayerStatus = "FREE_AGENT"
	// PlayerRetired and PlayerDeceased (§13, the Gaines-Adams Rule) are not signable.
	PlayerRetired  PlayerStatus = "RETIRED"
	PlayerDeceased PlayerStatus = "DECEASED"
)

// Valid reports whether s is a known status. A stored value that fails is drift; callers fail
// rather than pool or bar the player.
func (s PlayerStatus) Valid() bool {
	switch s {
	case PlayerFreeAgent, PlayerRetired, PlayerDeceased:
		return true
	default:
		return false
	}
}

// Signable is true only for a free agent.
func (s PlayerStatus) Signable() bool { return s == PlayerFreeAgent }
