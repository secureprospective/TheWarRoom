package commissioner

import (
	"strings"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/governance"
	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
)

func TestCompensatoryTradePickIsNeverTradable(t *testing.T) {
	for _, reverse := range []bool{false, true} {
		seed := ledgerSnapshot()
		seed.Picks[0].Compensatory = true
		asset := TradeAsset{From: "franchise.02", To: "franchise.01", Season: 2028, Round: 1, Original: "franchise.01"}
		if reverse {
			asset.From, asset.To = asset.To, asset.From
		}
		if err := moveTradeAssets(seed, []TradeAsset{asset}, reverse); err == nil {
			t.Fatal("a compensatory pick was traded")
		}
		if seed.Picks[0].Owner != "franchise.02" {
			t.Fatal("refused pick moved")
		}
	}
}

func TestTradeNoEligibleReviewFailsClosed(t *testing.T) {
	events := []governance.Event{governance.LeagueCreated{Creator: "creator", Franchises: []governance.FranchiseID{"a", "b", "c", "d"}}}
	b := newLedger(t, PresetLegacy)
	parties := []string{"a", "b", "c", "d"}
	for _, p := range parties {
		events = append(events, governance.SeatReplaced{Franchise: governance.FranchiseID(p), Principal: governance.Principal(p)},
			governance.RoleGranted{Principal: governance.Principal(p), Role: governance.DOT})
		b.add(RawCommand{Command: roleGrant, Args: RawArgs{People: &RawPeople{Principal: p, Role: governance.DOT}}}, seasonMoment(2))
	}
	state, err := governance.Replay(events)
	if err != nil {
		t.Fatal(err)
	}
	trade := TradeState{Parties: parties, Policy: TradePolicy{Seats: 4}}
	if _, err := tradeReviewers(&trade, state, b.records, seasonMoment(3)); err == nil || trade.Status != "failed" || !strings.Contains(trade.Words, "No eligible") {
		t.Fatalf("all-recused trade did not fail closed: %+v %v", trade, err)
	}
	if len(trade.Recusals) != 4 {
		t.Fatal("recusals missing")
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

func TestTradePolicyNeedsRulingsInsteadOfDefaults(t *testing.T) {
	if _, err := resolveTradePolicy(leaguerules.TradesRules{}); err == nil {
		t.Fatal("unset rules defaulted")
	}
	b := newLedger(t, PresetLegacy)
	rules, err := RulesNow(b.records, seasonMoment(2))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := resolveTradePolicy(rules.Trades); err == nil {
		t.Fatal("unset DOT seats defaulted")
	}
	rules.Trades.DOTSeats = leaguerules.Known(4)
	p, err := resolveTradePolicy(rules.Trades)
	if err != nil {
		t.Fatal(err)
	}
	if p.DecideAt != 3 || p.FutureYears != 2 || p.MinimumRound != 1 || p.MaximumRound != 5 || p.Expiry != 7*24*time.Hour || p.MaximumParties != 3 || !p.RosterGate {
		t.Fatalf("wrong preset policy: %+v", p)
	}
}
