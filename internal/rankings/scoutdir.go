package rankings

import (
	"github.com/secureprospective/TheWarRoom/internal/playerid"
	"github.com/secureprospective/TheWarRoom/internal/scouting"
)

// ScoutingDirectory resolves a rostered mfl id to its scouting Profile. A
// narrow injected port mirroring Directory: the orchestrator stays decoupled
// from how profiles are built (today: the RAS assembler in internal/scouting/
// assembly; later: the full scouting fetch+assemble pipeline). A miss
// (ok=false) is ordinary — the player has no scouting signal this pass, L1
// imputes the per-signal fallback (RAS → DefaultRASFallback = 5.0). A profile
// present in the directory is a player the assembler populated; in S-Phase 0
// the only field set is RAS.
//
// PRESENCE = HAS-RAS (S-Phase 0 convention): scouting.Profile has a flat
// `RAS float64` but no `HasRAS bool` — the source of truth is the
// composition.PlayerSpec the orchestrator builds, not the Profile. A profile
// in the directory with RAS populated means "this player has a RAS signal";
// absence means "no signal". The rankings consumer therefore treats "absent
// from the directory" identically to "HasRAS=false" — both fall through to
// L1's DefaultRASFallback imputation.
type ScoutingDirectory interface {
	Profile(mflID string) (scouting.Profile, bool)
}

// MapScoutingDirectory is the concrete map-backed ScoutingDirectory the app
// wires over the RAS assembler's output. A nil or empty map is legal (an
// explicitly-empty directory is a real condition — every player misses); the
// orchestrator's New nil-guards the directory ITSELF, not the map's contents.
type MapScoutingDirectory struct {
	profiles map[playerid.PlayerID]scouting.Profile
}

// NewMapScoutingDirectory wraps a profile map. The map is held read-only from
// the caller's perspective after handoff; the orchestrator never mutates it.
func NewMapScoutingDirectory(profiles map[playerid.PlayerID]scouting.Profile) MapScoutingDirectory {
	return MapScoutingDirectory{profiles: profiles}
}

// Profile resolves an mfl id (canonicalized through playerid.New) to its
// scouting Profile. A malformed id or a miss returns (_, false) — ordinary.
func (m MapScoutingDirectory) Profile(mflID string) (scouting.Profile, bool) {
	pid, err := playerid.New(mflID)
	if err != nil {
		return scouting.Profile{}, false
	}
	p, ok := m.profiles[pid]
	return p, ok
}
