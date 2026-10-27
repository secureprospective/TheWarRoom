package commissioner

import (
	"testing"
)

func TestCompensatoryTradePickIsNeverTradable(t *testing.T) {
	for _, reverse := range []bool{false, true} {
		seed := ledgerSnapshot()
		seed.Picks[0].Compensatory = true
		asset := TradeAsset{From: "franchise.02", To: "franchise.01", Season: 2028, Round: 1, Original: "franchise.01"}
		if reverse {
			asset.From, asset.To = asset.To, asset.From
		}
		if err := MoveTradeAssets(seed, []TradeAsset{asset}, reverse); err == nil {
			t.Fatal("a compensatory pick was traded")
		}
		if seed.Picks[0].Owner != "franchise.02" {
			t.Fatal("refused pick moved")
		}
	}
}

func TestTradeBoundaryRefusesForeignAndForgedDetails(t *testing.T) {
	args := RawTradeArgs{ID: "trade", Decision: "approve"}
	raw := RawCommand{League: "native.test", Command: TradeOverride, Args: RawArgs{Trade: &args}, Request: "override", Reason: "Correct decision", Effective: "now", Visibility: "league"}
	if err := raw.Validate(); err != nil {
		t.Fatal(err)
	}
	for _, edit := range []func(*RawCommand){
		func(r *RawCommand) { r.Reason = "" }, func(r *RawCommand) { r.Visibility = "private" },
		func(r *RawCommand) { r.Gravity = "G1" }, func(r *RawCommand) { r.Effective = "next_rollover" },
		func(r *RawCommand) { r.Args.Name = "foreign" },
	} {
		bad := raw
		edit(&bad)
		if err := bad.Validate(); err == nil {
			t.Fatalf("accepted malformed override: %+v", bad)
		}
	}
	for _, bad := range []RawTradeArgs{{ID: "trade", Decision: "maybe"}, {ID: "trade", Parties: []string{"a", "b"}},
		{Franchise: "a", Parties: []string{"a", "a"}, Assets: []TradeAsset{{From: "a", To: "b", Player: "0001"}}}} {
		command := TradeReview
		if bad.ID == "" {
			command = TradeOffer
		}
		if err := bad.Validate(command); err == nil {
			t.Fatalf("accepted malformed trade: %+v", bad)
		}
	}
	if _, err := Decode(`{"league":"native.test","command":"trade.override","args":{"trade":{"id":"trade","decision":"approve"},"tradeOutcome":{}},"reason":"correct","request":"x","effective":"now","visibility":"league"}`); err == nil {
		t.Fatal("accepted a forged server outcome")
	}
}
