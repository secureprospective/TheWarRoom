package domain

// PlayerStatus is an off-roster player's availability. ReleasePlayer serves waiver cuts (§8),
// buyouts (§12), retirement and death (§13) and rollover expiry (§14), which all leave the same
// database footprint; without this marker a retired player would look like a signable free
// agent. Status is an append-only event log (player_status_events); the latest row is current.
type PlayerStatus string

const (
	// PlayerFreeAgent is signable. A buyout also lands here; its "no re-bid until next
	// offseason" lockout is derived from the dead_cap row, not stored.
	PlayerFreeAgent PlayerStatus = "FREE_AGENT"
	// PlayerRetired is retired (§13) and not signable.
	PlayerRetired PlayerStatus = "RETIRED"
	// PlayerDeceased is deceased (§13 Gaines-Adams Rule) and not signable.
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
