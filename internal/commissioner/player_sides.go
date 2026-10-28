package commissioner

import (
	"fmt"
	"slices"
)

// The side seam. In a league whose player pools are per side, each side holds its own copy of every player:
// the same player id can be on a roster, in a free-agent pool, at auction or on waivers once on each side.
// Every lookup of a player's copy therefore names the side, or the franchise whose side it is; nothing
// outside this file finds a copy by the player's id alone (TestPlayerLookupsNameASide holds it). In a league
// with one pool every side is "", so each lookup reads exactly as a lookup by id did.

// errNameTheSide refuses a player-scoped change that names no franchise in a league with a pool per side.
var errNameTheSide = fmt.Errorf("each side has its own copy of this player: name the franchise or side")

// SideOf is the franchise's side; "" in a league with one player pool.
func (r RawNativeSnapshot) SideOf(franchise string) string { return r.Sides[franchise] }

// PerSide says the league holds a copy of every player for each side.
func (r RawNativeSnapshot) PerSide() bool { return len(r.Sides) != 0 }

// PlayerOn is the index of the player's copy on a roster of the side, or -1.
func (r RawNativeSnapshot) PlayerOn(side, id string) int {
	return slices.IndexFunc(r.Players, func(p NativePlayer) bool { return p.ID == id && r.SideOf(p.Franchise) == side })
}

// OnSide says the player's copy on the side is on a roster.
func (r RawNativeSnapshot) OnSide(side, id string) bool { return r.PlayerOn(side, id) >= 0 }

// PlayerOf is the index of the player on the franchise's roster, or -1.
func (r RawNativeSnapshot) PlayerOf(franchise, id string) int {
	return slices.IndexFunc(r.Players, func(p NativePlayer) bool { return p.ID == id && p.Franchise == franchise })
}

// PoolOn is the index of the player's free-agent lot on the side, or -1.
func (r RawNativeSnapshot) PoolOn(side, id string) int {
	return slices.IndexFunc(r.Pool, func(p NativePoolEntry) bool { return p.Player == id && p.Side == side })
}

// TargetSide is the side a player-scoped correction acts on: the named franchise's, or in a league with one
// pool the only side there is. A per-side league refuses a correction that names neither.
func (r RawNativeSnapshot) TargetSide(franchise string) (string, error) {
	if franchise != "" {
		return r.SideOf(franchise), nil
	}
	if r.PerSide() {
		return "", errNameTheSide
	}
	return "", nil
}

// SideNames are the league's sides in order; nil for one pool.
func (r RawNativeSnapshot) SideNames() []string {
	var out []string
	for _, side := range r.Sides {
		if !slices.Contains(out, side) {
			out = append(out, side)
		}
	}
	slices.Sort(out)
	return out
}

// bidSideOf is the side of the franchise that placed a bid, followed back from a raise or withdrawal; ""
// in a league with one pool, which never reads its rules for it.
func bidSideOf(r Record, records []Record) (string, error) {
	sides, err := snapshotSides(records)
	if err != nil || len(sides) == 0 {
		return "", err
	}
	if r.Command.Args.Bid == nil {
		return "", nil
	}
	return sides[r.Command.Args.Bid.Franchise], nil
}
