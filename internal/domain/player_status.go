package domain

// PlayerStatus is a player's AVAILABILITY state once he is off every roster — the
// free-agency pool's membership marker. It exists because ReleasePlayer (the sole
// roster-removal primitive) is shared by four terminal paths (waiver-cut §8, buyout §12,
// retirement §13, death §13) plus the §14 rollover UFA-expiry — all of which leave the
// player with the SAME database footprint (no rosters/contracts rows). Without an explicit
// marker a retired or deceased player is indistinguishable from a signable free agent. The
// pool is exactly the set of players whose latest status is FREE_AGENT.
//
// Status is recorded as an APPEND-ONLY event (player_status_events) — the current status is
// the latest row — matching the house double-immutable ledger idiom (dead_cap / cap_relief /
// season_phases). A player cycles FREE_AGENT → rostered (a SIGN clears him from the pool) →
// FREE_AGENT across seasons; the log carries every transition with its reason.
type PlayerStatus string

const (
	// PlayerFreeAgent is a signable free agent — released via waiver-cut or rollover expiry,
	// or bought out (a buyout also makes him a free agent; the "no re-bid until next offseason"
	// lockout is DERIVED from the dead_cap buyout row, not stored here).
	PlayerFreeAgent PlayerStatus = "FREE_AGENT"
	// PlayerRetired is a retired player (§13) — off every roster and NOT signable.
	PlayerRetired PlayerStatus = "RETIRED"
	// PlayerDeceased is a deceased player (§13 Gaines-Adams Rule) — off every roster and NOT
	// signable.
	PlayerDeceased PlayerStatus = "DECEASED"
)

// Valid reports whether s is one of the known player statuses. A value read back from storage
// that fails this is drift — callers fail loud rather than treat an unknown status as signable
// or unsignable (an unrecognized status must never silently pool or bar a player).
func (s PlayerStatus) Valid() bool {
	switch s {
	case PlayerFreeAgent, PlayerRetired, PlayerDeceased:
		return true
	default:
		return false
	}
}

// Signable reports whether a player at this status may be signed out of the pool — true only
// for a free agent. Retired/deceased players are off-roster but barred.
func (s PlayerStatus) Signable() bool { return s == PlayerFreeAgent }
