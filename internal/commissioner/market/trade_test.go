package market

import (
	"strings"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/governance"
	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
)

func TestTradeNoEligibleReviewFailsClosed(t *testing.T) {
	events := []governance.Event{governance.LeagueCreated{Creator: "creator", Franchises: []governance.FranchiseID{"a", "b", "c", "d"}}}
	b := newLedger(t, commissioner.PresetLegacy)
	parties := []string{"a", "b", "c", "d"}
	for _, p := range parties {
		events = append(events, governance.SeatReplaced{Franchise: governance.FranchiseID(p), Principal: governance.Principal(p)},
			governance.RoleGranted{Principal: governance.Principal(p), Role: governance.DOT})
		b.add(commissioner.RawCommand{Command: roleGrant, Args: commissioner.RawArgs{People: &commissioner.RawPeople{Principal: p, Role: governance.DOT}}}, seasonMoment(2))
	}
	state, err := governance.Replay(events)
	if err != nil {
		t.Fatal(err)
	}
	trade := commissioner.TradeState{Parties: parties, Policy: commissioner.TradePolicy{Seats: 4}}
	if _, err := tradeReviewers(&trade, state, b.records, seasonMoment(3)); err == nil || trade.Status != "failed" || !strings.Contains(trade.Words, "No eligible") {
		t.Fatalf("all-recused trade did not fail closed: %+v %v", trade, err)
	}
	if len(trade.Recusals) != 4 {
		t.Fatal("recusals missing")
	}
}

func TestTradePolicyNeedsRulingsInsteadOfDefaults(t *testing.T) {
	if _, err := resolveTradePolicy(leaguerules.TradesRules{}); err == nil {
		t.Fatal("unset rules defaulted")
	}
	b := newLedger(t, commissioner.PresetLegacy)
	rules, err := commissioner.RulesNow(b.records, seasonMoment(2))
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
