package rankings

import (
	"github.com/secureprospective/TheWarRoom/internal/playerid"
	"github.com/secureprospective/TheWarRoom/internal/scouting"
)

// ScoutingDirectory resolves a rostered mfl id to its scouting Profile. A
// narrow injected port mirroring Directory: the orchestrator stays decoupled
// from how profiles are built (the scouting assemblers in internal/scouting/
// assembly). A miss (ok=false) is ordinary — the player has no scouting signal
// at all this pass, and every rubric takes its Data-Parity neutral path.
//
// PER-FIELD PRESENCE (S-Phase 1 onward — supersedes the S-Phase 0 shortcut): a
// Profile can now carry MORE THAN ONE signal (S-Phase 0 RAS + S-Phase 1
// SchoolTier), so "present in the directory" no longer implies "has a RAS". The
// consumer must gate EACH field on that field's own presence signal:
//   - RAS is a bare float with no absent-sentinel → gate on Profile.HasRAS.
//   - SchoolTier has the SchoolUnset sentinel → gate on tier != SchoolUnset.
//
// A player present for one signal but absent for another (school tier known, no
// combine) copies only the present field; the missing one stays neutral. "Absent
// from the directory" and "present but that field unset" are treated identically
// per field — both fall through to the rubric's Data-Parity imputation (RAS →
// DefaultRASFallback = 5.0; SchoolTier → HasSchoolTier false).
type ScoutingDirectory interface {
	Profile(mflID string) (scouting.Profile, bool)
}

// MapScoutingDirectory is the concrete map-backed ScoutingDirectory the app
// wires over the merged scouting-assembler output (RAS + SchoolTier, and later
// signals folded into the same per-player Profile). A nil or empty map is legal (an
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
