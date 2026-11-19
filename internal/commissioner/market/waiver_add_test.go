package market

import (
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/nativecheck"
)

func TestRosterAddOnWaiversRequiresAClaim(t *testing.T) {
	const player = "6001"
	name := "Held receiver"
	x := waiverExecution{
		assets: commissioner.NativeAssets{Snapshot: &commissioner.RawNativeSnapshot{Season: 2026,
			Franchises: []commissioner.NativeFranchise{{ID: "franchise.01"}}}},
		state: commissioner.MarketBook{Windows: []commissioner.WaiverWindow{{Player: commissioner.NativePlayer{ID: player}, Side: ""}}},
		now:   time.Now(), facts: nativecheck.Facts{player: {Name: name}},
	}
	raw := &commissioner.RawCommand{Command: commissioner.RosterAdd, Args: commissioner.RawArgs{Waiver: &commissioner.RawWaiverArgs{
		Player: player, Franchise: "franchise.01"}}}
	if err := x.add(raw); err == nil || err.Error() != "on waivers; Use a waiver claim for "+name {
		t.Fatalf("waiver bypass error = %v", err)
	}
}
