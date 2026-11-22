package commissioner

import (
	"fmt"
	"slices"
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/domain"
)

// twinPoolLedger is nativeLedger (league_teams_test.go) with the preset named: the same create, native start
// and seed records, so the rules in force carry the player pools the preset gives the league.
func twinPoolLedger(t *testing.T, preset string, teams int) (*ledgerBuilder, RawNativeSnapshot) {
	t.Helper()
	create, err := RawCommand{League: "native.test", Command: Create, Args: RawArgs{Name: "Ours", Preset: preset,
		Source: SourceNative, Season: 2026, FranchiseCount: teams}, Request: "create", Effective: "now",
		Visibility: "league"}.Prepared()
	if err != nil {
		t.Fatal(err)
	}
	b := &ledgerBuilder{t: t}
	b.add(create, seasonMoment(1))
	seed, err := NativeStart("native.test", b.records, seasonMoment(1))
	if err != nil {
		t.Fatal(err)
	}
	b.add(RawCommand{League: "native.test", Command: NativeSeed, Args: RawArgs{Native: &seed}, Request: "seed",
		Effective: "now", Visibility: "league"}, seasonMoment(1))
	return b, seed
}

// twinPoolAdd records a roster add of the player to the franchise, with the contract the add would sign.
func twinPoolAdd(b *ledgerBuilder, player, franchise string) {
	b.t.Helper()
	contract := NativePlayer{ID: player, Franchise: franchise, Name: "Player " + player,
		Status: domain.RosterActive, Salary: 500, ExpirationYear: 2026, ContractStatus: domain.CStatusUFA}
	b.add(RawCommand{League: "native.test", Command: RosterAdd, Request: fmt.Sprintf("add%d", len(b.records)),
		Effective: "now", Visibility: "league", Reason: "Agreed",
		Args: RawArgs{Waiver: &RawWaiverArgs{Player: player, Franchise: franchise},
			WaiverOutcome: &WaiverOutcome{Player: &contract, Season: 2026}}}, seasonMoment(2))
}

// twinPoolCopies is the player's copies on the rosters: the franchises that hold him, in ledger order.
func twinPoolCopies(t *testing.T, b *ledgerBuilder, player string) []string {
	t.Helper()
	assets, err := NativeAssetsReading("native.test", b.records, seasonMoment(5))
	if err != nil {
		t.Fatalf("read rosters: %v", err)
	}
	var out []string
	for _, p := range assets.Snapshot.Players {
		if p.ID == player {
			out = append(out, p.Franchise)
		}
	}
	return out
}

// twinPoolSideTeams are the franchises of the side, in the league's order.
func twinPoolSideTeams(seed RawNativeSnapshot, side string) []string {
	var out []string
	for _, f := range seed.Franchises {
		if seed.SideOf(f.ID) == side {
			out = append(out, f.ID)
		}
	}
	return out
}

func TestTwinPoolProof(t *testing.T) {
	const player = "1000"

	t.Run("a PerSide league takes one player on an AFC team and an NFC team", func(t *testing.T) {
		b, seed := twinPoolLedger(t, PresetMirror16, 16)
		sides := seed.SideNames()
		if !slices.Equal(sides, []string{"AFC", "NFC"}) {
			t.Fatalf("sides %v, want [AFC NFC]", sides)
		}
		afc, nfc := twinPoolSideTeams(seed, "AFC")[0], twinPoolSideTeams(seed, "NFC")[0]
		twinPoolAdd(b, player, afc)
		twinPoolAdd(b, player, nfc)
		copies := twinPoolCopies(t, b, player)
		if !slices.Equal(copies, []string{afc, nfc}) {
			t.Fatalf("copies %v, want %v (one per side)", copies, []string{afc, nfc})
		}
	})

	t.Run("a PerSide league refuses a second AFC team for the same player", func(t *testing.T) {
		b, seed := twinPoolLedger(t, PresetMirror16, 16)
		afc := twinPoolSideTeams(seed, "AFC")
		twinPoolAdd(b, player, afc[0])
		twinPoolAdd(b, player, afc[1])
		if _, err := NativeAssetsReading("native.test", b.records, seasonMoment(5)); err == nil {
			t.Fatalf("second AFC copy accepted: %v", twinPoolCopies(t, b, player))
		} else {
			t.Logf("refused: %v", err)
		}
	})

	t.Run("a OnePool league refuses a second team for the same player", func(t *testing.T) {
		b, _ := twinPoolLedger(t, PresetLegacy, 32)
		twinPoolAdd(b, player, "franchise.01")
		twinPoolAdd(b, player, "franchise.02")
		if _, err := NativeAssetsReading("native.test", b.records, seasonMoment(5)); err == nil {
			t.Fatalf("second copy accepted in a one-pool league: %v", twinPoolCopies(t, b, player))
		} else {
			t.Logf("refused: %v", err)
		}
	})
}
