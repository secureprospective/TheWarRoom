package mflseed

import (
	"fmt"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
)

// MirrorAssets answers native.assets.read for the MFL-mirror league from the held rosters, in the
// same shape a native league returns. Franchises keep their MFL IDs, so a mirror owner's franchise
// is the one the rest of the app already knows. The revision is the held roster time, so every
// mirror read taken from the same refresh agrees.
func MirrorAssets(source SeedSource, league string) (commissioner.NativeAssets, error) {
	held, err := CaptureSeed(source)
	if err != nil {
		return commissioner.NativeAssets{}, fmt.Errorf("mirror league assets: %w", err)
	}
	keepHeldIDs(&held)
	out := commissioner.NativeAssets{League: league, Revision: held.RosterAt.Unix(), Seeded: true,
		Source: "MFL mirror; held rosters as of the last refresh", Snapshot: &held}
	positions := map[string]string{}
	for _, p := range source.Directory() {
		positions[p.ID.String()] = string(p.Position)
	}
	for _, p := range held.Players {
		out.Identities = append(out.Identities, commissioner.NativeAssetIdentity{Player: p.ID,
			Franchise: p.Franchise, Name: p.Name, Position: positions[p.ID]})
	}
	return out, nil
}

func keepHeldIDs(s *commissioner.RawNativeSnapshot) {
	held := make(map[string]string, len(s.Franchises))
	for i, f := range s.Franchises {
		held[f.ID] = f.HeldID
		s.Franchises[i].ID = f.HeldID
	}
	for i := range s.Players {
		s.Players[i].Franchise = held[s.Players[i].Franchise]
	}
	for i := range s.Picks {
		s.Picks[i].Original, s.Picks[i].Owner = held[s.Picks[i].Original], held[s.Picks[i].Owner]
	}
	for i := range s.Adjustments {
		s.Adjustments[i].Franchise = held[s.Adjustments[i].Franchise]
	}
}
