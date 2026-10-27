package trades

import (
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/envelope"
	"github.com/secureprospective/TheWarRoom/internal/leaguefeed"
	"github.com/secureprospective/TheWarRoom/internal/playerid"
)

func TestTradeRequestMapsOnlyAKnownHost(t *testing.T) {
	t.Parallel()
	host := ""
	id, err := playerid.New("0099")
	if err != nil {
		t.Fatal(err)
	}
	offer := leaguefeed.PendingTrade{ID: "77", Offering: "0001", OfferedTo: "0025", Gives: []leaguefeed.Asset{{Player: &id}, {FuturePick: &leaguefeed.FuturePick{Year: 2027}}}, Gets: []leaguefeed.Asset{{Player: &id}}, Expires: time.Now()}
	req := Request("0025", offer, host, 2026)
	if req.Target.Kind != envelope.Unmapped || req.FranchiseID != "0025" || req.Trade.TradeID != "77" ||
		req.Trade.Offering != "0001" || req.Trade.Accepting != "0025" || req.Deadline == nil || !req.Deadline.Equal(offer.Expires) ||
		len(req.Trade.OfferingGives) != 2 || req.Trade.AcceptingGives[0] != "0099" {
		t.Fatal(req)
	}
	offer.Expires = time.Time{}
	if req := Request("0025", offer, host, 2026); req.Deadline != nil {
		t.Fatal("zero expiry became a deadline")
	}
	host = "www47.myfantasyleague.com"
	if req := Request("0025", offer, host, 2026); req.Target.Kind != envelope.Mapped ||
		req.Target.URL != "https://www47.myfantasyleague.com/2026/options?L=14432&O=05" {
		t.Fatal(req.Target)
	}
}
