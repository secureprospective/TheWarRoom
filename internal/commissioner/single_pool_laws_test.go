package commissioner

import (
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/domain"
)

func TestSinglePoolLaws(t *testing.T) {
	// Provenance without a franchise is covered by TestCorrectionProjectionNeverRewritesTheOriginalEntry.
	for _, row := range []struct {
		name string
		run  func(*testing.T)
	}{
		{"duplicate seed refused", func(t *testing.T) {
			p := NativePlayer{ID: "1000", Franchise: "a", Status: domain.RosterActive}
			s := RawNativeSnapshot{Players: []NativePlayer{p, p}}
			err := s.validatePlayers(map[string]bool{"a": true})
			if err == nil || err.Error() != "a held player has repeated or invalid contract details" {
				t.Fatalf("duplicate: %v", err)
			}
		}},
		{"open and latest auction", func(t *testing.T) {
			b := MarketBook{Bids: BidBook{Auctions: []BidAuction{{Player: "1000", Decided: true}, {Player: "1000"}}}}
			if b.Bids.Open("1000") != &b.Bids.Auctions[1] || b.Bids.Latest("1000") != &b.Bids.Auctions[1] {
				t.Fatal("open or latest did not find newest auction")
			}
			b.Bids.Auctions[1].Decided = true
			if b.Bids.Open("1000") != nil || b.Bids.Latest("1000") != &b.Bids.Auctions[1] {
				t.Fatal("decided auction state incorrect")
			}
		}},
		{"roster correction finds ID", func(t *testing.T) {
			s := RawNativeSnapshot{Franchises: []NativeFranchise{{ID: "a"}}, Players: []NativePlayer{{ID: "1000", Franchise: "a"}, {ID: "2000", Franchise: "a"}}}
			r := Record{Command: RawCommand{Command: CorrectRoster, Args: RawArgs{Correction: &RawCorrection{Player: "2000", Franchise: "a", Action: "remove"}}}}
			if err := applyAssetCorrection(&s, nil, r); err != nil {
				t.Fatal(err)
			}
			if len(s.Players) != 1 || s.Players[0].ID != "1000" {
				t.Fatalf("players: %+v", s.Players)
			}
		}},
		{"draft replaces rookie ID", func(t *testing.T) {
			s := RawNativeSnapshot{Players: []NativePlayer{{ID: "1000", Franchise: "a"}}, Picks: []NativePick{{Player: "1000"}}}
			r := Record{Command: RawCommand{Args: RawArgs{Draft: &RawDraftArgs{Act: &RawDraftAct{Player: "2000", Replaces: "1000"}}}}}
			applyDraftChange(&s, r)
			if s.Players[0].ID != "2000" || s.Players[0].Franchise != "a" || s.Picks[0].Player != "2000" {
				t.Fatalf("snapshot: %+v", s)
			}
		}},
	} {
		t.Run(row.name, row.run)
	}
}
