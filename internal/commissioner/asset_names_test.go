package commissioner

import (
	"strings"
	"testing"
)

// scanAssetName is the pre-11-08 per-lookup scan, kept as the reference the index must match.
func scanAssetName(records []Record, id string, player bool) string {
	for _, r := range records {
		if r.Command.Args.Native == nil {
			continue
		}
		if player {
			for _, p := range r.Command.Args.Native.Players {
				if p.ID == id && p.Name != "" {
					return p.Name
				}
			}
		} else {
			for _, f := range r.Command.Args.Native.Franchises {
				if f.ID == id {
					return f.Name
				}
			}
		}
	}
	if player {
		return "Player " + id
	}
	return "Franchise " + strings.TrimPrefix(id, "franchise.")
}

// The first snapshot in record order that names an asset wins: an unnamed copy is skipped for players,
// a franchise's first copy wins even with an empty name, and unknown ids fall back to the id.
func TestAssetNamesMatchPerLookupScan(t *testing.T) {
	snap := func(players []NativePlayer, teams []NativeFranchise) Record {
		return Record{Command: RawCommand{Args: RawArgs{Native: &RawNativeSnapshot{Players: players, Franchises: teams}}}}
	}
	records := []Record{
		{},
		snap([]NativePlayer{{ID: "0042"}, {ID: "1001", Name: "First"}}, []NativeFranchise{{ID: "franchise.01", Name: ""}}),
		snap([]NativePlayer{{ID: "0042", Name: "Later"}, {ID: "1001", Name: "Second"}, {ID: "0042", Name: "Third"}},
			[]NativeFranchise{{ID: "franchise.01", Name: "Renamed"}, {ID: "franchise.02", Name: "Two"}}),
	}
	ids := []string{"0042", "1001", "42", "9999", "", "franchise.01", "franchise.02", "franchise.09"}
	for _, prefix := range []int{0, 1, 2, 3} {
		names := newAssetNames(records[:prefix])
		for _, id := range ids {
			for _, player := range []bool{true, false} {
				if got, want := names.name(id, player), scanAssetName(records[:prefix], id, player); got != want {
					t.Fatalf("records[:%d] %q player=%v: got %q want %q", prefix, id, player, got, want)
				}
			}
		}
	}
}
