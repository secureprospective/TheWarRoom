package main

import (
	"encoding/json"
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/governance"
)

type tradeFixture struct{ *waiverFixture }

func tradeApp(t *testing.T) *tradeFixture {
	t.Helper()
	f := &tradeFixture{waiverApp(t)}
	f.set(t, "calendar.windows", `{"native":{"start":{"month":1,"day":1},"opens":["trade"]}}`, "now")
	f.set(t, "calendar.tradeDeadline", `"2026-12-01T00:00:00Z"`, "now")
	f.set(t, "trades.dotSeats", `4`, "now")
	for _, who := range []string{"owner.two", "dot.a", "dot.b", "dot.c"} {
		f.role(t, who, governance.DOT)
	}
	f.role(t, "cc", governance.CoCommissioner)
	return f
}

func (f *tradeFixture) role(t *testing.T, who string, role governance.Role) {
	t.Helper()
	f.command(t, "commish.role.grant", commissioner.RawArgs{People: &commissioner.RawPeople{Principal: who, Role: role}}, "private")
}

func tradePlayerOffer() commissioner.RawTradeArgs {
	return commissioner.RawTradeArgs{Franchise: "franchise.02", Parties: []string{"franchise.02", "franchise.01"},
		Assets: []commissioner.TradeAsset{{From: "franchise.02", To: "franchise.01", Player: "1000"},
			{From: "franchise.01", To: "franchise.02", Player: "4000"}}}
}

func (f *tradeFixture) action(t *testing.T, who, command string, args commissioner.RawTradeArgs) (commissioner.Receipt, error) {
	t.Helper()
	f.app.core.CommissionerSession = testCommissionerSession{who}
	raw := f.raw(t, command, commissioner.RawArgs{Trade: &args}, "league")
	if commissioner.OwnerTradeAction(command) != "" {
		return newCommissionerAPI(f.app).TargetNativeTradeAction(raw)
	}
	body, err := json.Marshal(raw)
	if err != nil {
		t.Fatal(err)
	}
	return newCommissionerAPI(f.app).TargetCommissionerExecute(string(body))
}

func (f *tradeFixture) mustAction(t *testing.T, who, command string, args commissioner.RawTradeArgs) commissioner.Receipt {
	t.Helper()
	receipt, err := f.action(t, who, command, args)
	if err != nil {
		t.Fatalf("%s as %s: %v", command, who, err)
	}
	return receipt
}

func (f *tradeFixture) offer(t *testing.T) commissioner.Receipt {
	t.Helper()
	return f.mustAction(t, "owner.two", commissioner.TradeOffer, tradePlayerOffer())
}

func (f *tradeFixture) accept(t *testing.T, offer commissioner.Receipt) commissioner.Receipt {
	t.Helper()
	return f.mustAction(t, "owner.one", commissioner.TradeAccept, commissioner.RawTradeArgs{ID: offer.Entry, Franchise: "franchise.01"})
}

func (f *tradeFixture) vote(t *testing.T, offer commissioner.Receipt, who, decision string) commissioner.Receipt {
	t.Helper()
	return f.mustAction(t, who, commissioner.TradeReview, commissioner.RawTradeArgs{ID: offer.Entry, Decision: decision})
}

func (f *tradeFixture) veto(t *testing.T, offer commissioner.Receipt) {
	t.Helper()
	for _, who := range []string{"dot.a", "dot.b", "dot.c"} {
		f.vote(t, offer, who, "veto")
	}
}

func (f *tradeFixture) trades(t *testing.T, who, team string) commissioner.NativeTrades {
	t.Helper()
	f.app.core.CommissionerSession = testCommissionerSession{who}
	reading, err := newCommissionerAPI(f.app).TargetNativeTrades(legalityLeague, team)
	if err != nil {
		t.Fatal(err)
	}
	return reading
}

func (f *tradeFixture) only(t *testing.T) commissioner.NativeTradeOffer {
	t.Helper()
	reading := f.trades(t, "owner.one", "franchise.01")
	if len(reading.Offers) != 1 {
		t.Fatalf("offers: %+v", reading)
	}
	return reading.Offers[0]
}

func (f *tradeFixture) ownerOf(t *testing.T, player string) string {
	t.Helper()
	assets, err := newCommissionerAPI(f.app).TargetNativeAssets(legalityLeague)
	if err != nil {
		t.Fatal(err)
	}
	for _, p := range assets.Snapshot.Players {
		if p.ID == player {
			return p.Franchise
		}
	}
	t.Fatalf("player disappeared: %s", player)
	return ""
}

func tradePickApp(t *testing.T) *tradeFixture {
	t.Helper()
	d := draftApp(t)
	f := &tradeFixture{&waiverFixture{legalityFixture: &legalityFixture{app: d.app, path: d.path, pools: d.pools}, now: d.now}}
	f.clock()
	f.set(t, "roster.limits", `{"min":0,"max":100}`, "now")
	f.set(t, "cap.amount", `100000000000`, "now")
	f.set(t, "calendar.windows", `{"native":{"start":{"month":1,"day":1},"opens":["trade"]}}`, "now")
	f.set(t, "calendar.tradeDeadline", `"2027-12-01T00:00:00Z"`, "now")
	f.set(t, "trades.dotSeats", `4`, "now")
	for _, owner := range []struct{ id, who string }{{"franchise.01", "owner.one"}, {"franchise.02", "owner.two"}, {"franchise.03", "owner.three"}} {
		f.command(t, "commish.seat.replace", commissioner.RawArgs{People: &commissioner.RawPeople{Franchise: owner.id, Principal: owner.who}}, "league")
	}
	for _, who := range []string{"owner.two", "dot.a", "dot.b", "dot.c"} {
		f.role(t, who, governance.DOT)
	}
	return f
}
