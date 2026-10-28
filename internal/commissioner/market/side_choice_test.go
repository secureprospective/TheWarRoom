package market

import (
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
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
			seed := &commissioner.RawNativeSnapshot{Sides: map[string]string{"a": "AFC", "n": "NFC"},
				Pool: []commissioner.NativePoolEntry{
					{Player: "990001", Side: "AFC", Name: "ARI O-Line"},
					{Player: "990001", Side: "NFC", Name: "ARI O-Line"},
				}}
			if side == "" {
				seed.Sides = nil
				seed.Pool = seed.Pool[:1]
				seed.Pool[0].Side = ""
			}
			records := []commissioner.Record{
				{Command: commissioner.RawCommand{Command: commissioner.Create, Args: commissioner.RawArgs{Source: commissioner.SourceNative}}},
				{Command: commissioner.RawCommand{Command: commissioner.NativeSeed, Args: commissioner.RawArgs{Native: seed}}},
			}
			assets := commissioner.NativeAssets{Seeded: true, Snapshot: seed}
			book := commissioner.MarketBook{}
			if rows := unbidStartingLots(assets, book, records, side); len(rows) != 1 {
				t.Fatalf("side lots: %+v", rows)
			}
			book.Bids.Auctions = []commissioner.BidAuction{{Player: "990001", Side: side}}
			if rows := unbidStartingLots(assets, book, records, side); len(rows) != 0 {
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
