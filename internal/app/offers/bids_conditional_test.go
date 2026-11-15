package offers

import (
	"strings"
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/mflref/envelope"
	"github.com/secureprospective/TheWarRoom/internal/store/state"
	"github.com/secureprospective/TheWarRoom/internal/transactions"
)

func TestConditionalBidUsesSharedChargeAndVersionsThePlan(t *testing.T) {
	t.Parallel()
	a := bidTestApp(t)
	setRosters(t, a, state.MirrorPlayer{MFLID: "13595", FranchiseID: "0001",
		Salary: 10_490_000_000, ContractYear: 2026, RosterStatus: domain.RosterActive})
	raw := rawBid("0.6", 1)
	plan, err := a.TargetBidDraft(raw)
	if err != nil || plan.Receipt.State != envelope.Ready {
		t.Fatalf("plan: %+v, %v", plan, err)
	}
	raw.ConditionalDrop = "0042"
	p := a.TargetBidPreview(raw)
	var drop BidDropOption
	for _, option := range p.ConditionalDrops {
		if option.ID == "0042" {
			drop = option
		}
	}
	cells := []state.LedgerCell{{Year: 2026, Salary: 2_000_000_000}, {Year: 2027, Salary: 2_000_000_000}}
	charge := transactions.CutCharge(cells, 2026, false)
	if p.Problem != "" || drop.DeadCap != charge || drop.Freed != drop.Salary-charge || !drop.Covers {
		t.Fatalf("covering cut not priced by shared rule: %+v, %+v, charge %d", p, drop, charge)
	}
	if !strings.Contains(p.Post, "Conditional drop if I win: Rostered, Rick (QB, BUF), commissioner to drop to cover the cap.") {
		t.Fatalf("conditional post: %s", p.Post)
	}
	ready, err := a.TargetBidRevise(plan.Receipt.CorrelationID, raw)
	if err != nil || ready.Receipt.State != envelope.Ready || ready.Facts.ConditionalDrop.String() != "0042" {
		t.Fatalf("revised plan: %+v, %v", ready, err)
	}
	stored, err := a.core.Moves.BidFacts(a.core.Ctx, ready.Receipt.CorrelationID)
	if err != nil || stored.ConditionalDrop != ready.Facts.ConditionalDrop || stored.ConditionalName != "Rostered, Rick (QB, BUF)" {
		t.Fatalf("condition lost in store: %+v, %v", stored, err)
	}
	raw.ConditionalDrop = ""
	cleared, err := a.TargetBidRevise(ready.Receipt.CorrelationID, raw)
	if err != nil || cleared.Receipt.State != envelope.Ready || strings.Contains(cleared.Post, "Conditional drop") {
		t.Fatalf("cleared condition: %+v, %v", cleared, err)
	}
}

func TestConditionalBidRejectsForeignMissingOrUnpricedDrop(t *testing.T) {
	t.Parallel()
	a := bidTestApp(t)
	setRosters(t, a, state.MirrorPlayer{MFLID: "13595", FranchiseID: "0002", Salary: 1_000_000,
		ContractYear: 2026}, state.MirrorPlayer{MFLID: "13594", FranchiseID: "0001", Salary: 1_000_000})
	for _, id := range []string{"13595", "99999", "bad", "13594"} {
		raw := rawBid("5", 1)
		raw.ConditionalDrop = id
		if p := a.TargetBidPreview(raw); p.Problem == "" {
			t.Errorf("accepted drop %s: %+v", id, p)
		}
	}
	raw := rawBid("5", 1)
	if strings.Contains(a.TargetBidPreview(raw).Post, "Conditional drop") {
		t.Fatal("unconditional post contains a drop")
	}
}
