package market

import (
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
	"github.com/secureprospective/TheWarRoom/internal/nativecheck"
)

// Starting cards belong to the auction, never to franchise creation order.
func TestDefault32LeagueStartsWithUnassignedAuctionLots(t *testing.T) {
	b := newLedger(t, commissioner.PresetDefault32)
	raw, err := commissioner.Default32SeedCommand(b.records, "native.test", "start", "Start", 1, seasonMoment(2))
	if err != nil {
		t.Fatal(err)
	}
	if err := commissioner.CheckNativeAppend(raw, b.records); err != nil {
		t.Fatal(err)
	}
	b.add(raw, seasonMoment(2))
	assets, err := commissioner.NativeAssetsReading("native.test", b.records, seasonMoment(3))
	if err != nil || !assets.Seeded || len(assets.Snapshot.Players) != 0 || len(assets.Snapshot.Pool) != 32 {
		t.Fatalf("%+v %v", assets, err)
	}
	again, err := commissioner.Default32SeedCommand(b.records, "native.test", "start", "Start", 1, seasonMoment(4))
	if err != nil || again.Args.Native.RosterAt != raw.Args.Native.RosterAt {
		t.Fatalf("retry: %v", err)
	}
	b.add(commissioner.RawCommand{League: "native.test", Command: "commish.seat.replace", Args: commissioner.RawArgs{People: &commissioner.RawPeople{Franchise: "franchise.01", Principal: "creator"}}}, seasonMoment(3))
	lots, err := NativeBidsReading("native.test", "franchise.01", "creator", b.records, seasonMoment(3), time.UTC, nativecheck.Facts{})
	if err != nil || len(lots.Players) != 32 {
		t.Fatalf("auction lots: %+v %v", lots, err)
	}
	for _, lot := range lots.Players {
		if lot.Mode != string(leaguerules.Open) || lot.Leader != nil {
			t.Fatalf("not unassigned open lot: %+v", lot)
		}
	}
}
