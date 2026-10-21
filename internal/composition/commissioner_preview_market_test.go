package composition

import (
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
)

func TestEveryBuiltWaiverAndBidCommandHasARealPreview(t *testing.T) {
	t.Parallel()
	f := recordedStandingsApp(t)
	apply := func(command string, args commissioner.RawWaiverArgs) {
		t.Helper()
		f.app.core.CommissionerSession = testCommissionerSession{"creator"}
		previewAndExecute(t, f.app, f.raw(t, command, commissioner.RawArgs{Waiver: &args}, "league"))
	}
	apply(commissioner.WaiverOrderSet, commissioner.RawWaiverArgs{Order: waiverOrder()})
	apply(commissioner.WaiverOrderAuto, commissioner.RawWaiverArgs{})
	cut := f.cut(t, "1000")
	f.claim(t, cut.Entry, "owner.one", "franchise.01")
	third := f.claim(t, cut.Entry, "owner.three", "franchise.03")
	f.now = f.now.Add(24 * time.Hour)
	apply(commissioner.WaiverProcess, commissioner.RawWaiverArgs{Window: cut.Entry})
	apply(commissioner.WaiverOverride, commissioner.RawWaiverArgs{Claim: third.Entry, Franchise: "franchise.03"})
	b := bidApp(t)
	one := b.place(t, "owner.one", "franchise.01", "5004", 5*million, 1)
	b.place(t, "owner.three", "franchise.03", "5004", 6*million, 2)
	b.app.core.CommissionerSession = testCommissionerSession{"creator"}
	previewAndExecute(t, b.app, b.raw(t, commissioner.BidsProcess,
		commissioner.RawArgs{Bid: &commissioner.RawBidArgs{Player: "5004"}}, "league"))
	previewAndExecute(t, b.app, b.raw(t, commissioner.BidsOverride,
		commissioner.RawArgs{Bid: &commissioner.RawBidArgs{Bid: one.Entry, Franchise: "franchise.01"}}, "league"))
}

func previewTrade(t *testing.T, f *tradeFixture, who, command string, args commissioner.RawTradeArgs) commissioner.Receipt {
	t.Helper()
	f.app.core.CommissionerSession = testCommissionerSession{who}
	return previewAndExecute(t, f.app, f.raw(t, command, commissioner.RawArgs{Trade: &args}, "league"))
}

func TestEveryBuiltTradeCommandHasARealPreview(t *testing.T) {
	t.Parallel()
	f := tradeApp(t)
	offer := f.offer(t)
	f.accept(t, offer)
	for _, who := range []string{"dot.a", "dot.b", "dot.c"} {
		previewTrade(t, f, who, commissioner.TradeReview, commissioner.RawTradeArgs{ID: offer.Entry, Decision: "veto"})
	}
	previewTrade(t, f, "owner.one", commissioner.TradeAppeal,
		commissioner.RawTradeArgs{ID: offer.Entry, Franchise: "franchise.01"})
	previewTrade(t, f, "creator", commissioner.TradeAppealRule,
		commissioner.RawTradeArgs{ID: offer.Entry, Decision: "veto"})
	previewTrade(t, f, "creator", commissioner.TradeOverride,
		commissioner.RawTradeArgs{ID: offer.Entry, Decision: "approve"})
	g := tradeApp(t)
	pending := g.offer(t)
	g.accept(t, pending)
	g.app.core.CommissionerSession = testCommissionerSession{"dot.a"}
	raw := g.raw(t, commissioner.TradeRecuse, commissioner.RawArgs{Trade: &commissioner.RawTradeArgs{ID: pending.Entry}}, "league")
	proveRefusedAct(t, g.app, raw)
}
