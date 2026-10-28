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
