package rankings

import (
	"github.com/secureprospective/TheWarRoom/internal/playerid"
	"github.com/secureprospective/TheWarRoom/internal/scouting"
)

// ScoutingDirectory resolves a rostered mfl id to its scouting Profile. A miss is ordinary.
// A Profile can carry some signals and not others, so consumers gate each field on its own
// presence (see applyScouting).
type ScoutingDirectory interface {
	Profile(mflID string) (scouting.Profile, bool)
}

// MapScoutingDirectory is the map-backed ScoutingDirectory the app wires over the assembled
// profiles. A nil or empty map is legal: every player misses.
type MapScoutingDirectory struct {
	profiles map[playerid.PlayerID]scouting.Profile
}

// NewMapScoutingDirectory wraps a profile map; the runner never mutates it.
func NewMapScoutingDirectory(profiles map[playerid.PlayerID]scouting.Profile) MapScoutingDirectory {
	return MapScoutingDirectory{profiles: profiles}
}

// Profile canonicalizes the id through playerid.New. A malformed id or a miss returns false.
func (m MapScoutingDirectory) Profile(mflID string) (scouting.Profile, bool) {
	pid, err := playerid.New(mflID)
	if err != nil {
		return scouting.Profile{}, false
	}
	p, ok := m.profiles[pid]
	return p, ok
}
