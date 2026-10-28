package market

import (
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
	"github.com/secureprospective/TheWarRoom/internal/nativecheck"
)

func TestBidChoicesDistinguishSides(t *testing.T) {
	book := commissioner.BidBook{Auctions: []commissioner.BidAuction{
		{Player: "player.1", Side: "AFC", Bids: []commissioner.BidRecord{{ID: "a"}}},
		{Player: "player.1", Side: "NFC", Bids: []commissioner.BidRecord{{ID: "n"}}},
	}}
	var out Market
	out.readBids(book, bidViewer{open: true, zone: time.UTC}, leaguerules.Value[time.Duration]{}, time.Now())
	if len(out.Players) != 2 || out.Players[0].ID == out.Players[1].ID || out.Players[0].ID == "player.1" {
		t.Fatalf("side choices must identify both player and side: %+v", out.Players)
	}
}

func TestStartingOLineLotsAreScopedAndCloseHidesThem(t *testing.T) {
	for _, side := range []string{"", "AFC", "NFC"} {
		t.Run(side, func(t *testing.T) {
			seed := &commissioner.RawNativeSnapshot{}
			sides := []string{""}
			if side != "" {
				seed.Sides = map[string]string{"a": "AFC", "n": "NFC"}
				sides = []string{"AFC", "NFC"}
			}
			for _, team := range nativecheck.LineCardTeams() {
				id, _ := nativecheck.LineCardID(team)
				for _, pool := range sides {
					seed.Pool = append(seed.Pool, commissioner.NativePoolEntry{Player: id, Side: pool, Name: team + " O-Line"})
				}
			}
			records := []commissioner.Record{
				{Command: commissioner.RawCommand{Command: commissioner.Create, Args: commissioner.RawArgs{Source: commissioner.SourceNative}}},
				{Command: commissioner.RawCommand{Command: commissioner.NativeSeed, Args: commissioner.RawArgs{Native: seed}}},
			}
			assets := commissioner.NativeAssets{Seeded: true, Snapshot: seed}
			book := commissioner.MarketBook{}
			if rows := unbidStartingLots(assets, book, records, side); len(rows) != 32 {
				t.Fatalf("side lots: %+v", rows)
			}
			book.Bids.Auctions = []commissioner.BidAuction{{Player: "990001", Side: side}}
			if rows := unbidStartingLots(assets, book, records, side); len(rows) != 31 {
				t.Fatal("live auction shown as unbid", rows)
			}
			book.Bids.Auctions = nil
			book.OLine = &commissioner.OLineOutcome{}
			if rows := unbidStartingLots(assets, book, records, side); len(rows) != 0 {
				t.Fatal("closed auction still offers lots", rows)
			}
		})
	}
}
