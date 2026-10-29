package offers

import (
	"strings"
	"testing"

	bidrules "github.com/secureprospective/TheWarRoom/internal/mflref/bid"

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
	p := a.TargetBidPreview(raw)
	if !p.Blocked || !strings.Contains(p.Note, "$500,000 over your $100,000 of room") {
		t.Fatalf("over cap preview: %+v", p)
	}
	blocked, err := a.TargetBidDraft(raw)
	if err != nil || blocked.Receipt.State != envelope.Blocked {
		t.Fatalf("over cap went Ready: %+v, %v", blocked, err)
	}
	if _, err := a.TargetBidHandOff(blocked.Receipt.CorrelationID); err == nil {
		t.Fatal("over cap plan handed off")
	}
	raw.ConditionalDrop = "0042"
	p = a.TargetBidPreview(raw)
	var drop BidDropOption
	for _, option := range p.ConditionalDrops {
		if option.ID == "0042" {
			drop = option
		}
	}
	cells := []state.LedgerCell{{Year: 2026, Salary: 2_000_000_000}, {Year: 2027, Salary: 2_000_000_000}}
	charge := transactions.CutCharge(cells, 2026, false)
	if p.Blocked || drop.DeadCap != charge || drop.Freed != drop.Salary-charge || !drop.Covers {
		t.Fatalf("covering cut not priced by shared rule: %+v, %+v, charge %d", p, drop, charge)
	}
	if !strings.Contains(p.Post, "Conditional drop if I win: Rostered, Rick (QB, BUF), commissioner to drop to cover the cap.") {
		t.Fatalf("conditional post: %s", p.Post)
	}
	ready, err := a.TargetBidRevise(blocked.Receipt.CorrelationID, raw)
	if err != nil || ready.Receipt.State != envelope.Ready || ready.Facts.ConditionalDrop.String() != "0042" {
		t.Fatalf("revised plan: %+v, %v", ready, err)
	}
	stored, err := a.core.Moves.BidFacts(a.core.Ctx, ready.Receipt.CorrelationID)
	if err != nil || stored.ConditionalDrop != ready.Facts.ConditionalDrop || stored.ConditionalName != "Rostered, Rick (QB, BUF)" {
		t.Fatalf("condition lost in store: %+v, %v", stored, err)
	}
	raw.ConditionalDrop = ""
	cleared, err := a.TargetBidRevise(ready.Receipt.CorrelationID, raw)
	if err != nil || cleared.Receipt.State != envelope.Blocked || strings.Contains(cleared.Post, "Conditional drop") {
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

func TestConditionalOptionsCoveringFirstAndShortfall(t *testing.T) {
	t.Parallel()
	a := bidTestApp(t)
	setRosters(t, a, state.MirrorPlayer{MFLID: "13595", FranchiseID: "0001", Salary: 10_490_000_000,
		ContractYear: 2030}, state.MirrorPlayer{MFLID: "13594", FranchiseID: "0001", Salary: 1_000_000,
		ContractYear: 2026})
	raw := rawBid("0.6", 1)
	raw.ConditionalDrop = "13594"
	p := a.TargetBidPreview(raw)
	if !p.Blocked || !strings.Contains(p.Note, "still ") || !strings.Contains(p.Note, "short") {
		t.Fatalf("insufficient drop passes: %+v", p)
	}
	if p.ConditionalDrops[0].ID != "0042" || !p.ConditionalDrops[0].Covers {
		t.Fatalf("covering options not first: %+v", p.ConditionalDrops)
	}
	for _, r := range p.Rules {
		if r.Rule == bidrules.RuleCapRoom && r.Verdict != bidrules.Fail {
			t.Fatal(r)
		}
	}
	for _, o := range p.ConditionalDrops {
		want := map[string]string{"13594": " · 1 year · ", "13595": " · 5 years · "}[o.ID]
		if want != "" && !strings.Contains(o.Label, want) {
			t.Fatalf("option %s label %q, want %q", o.ID, o.Label, want)
		}
	}
}
