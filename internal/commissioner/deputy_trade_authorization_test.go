package commissioner

import (
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/governance"
)

func deputyTradeAuthorizationRecords(t *testing.T, principal, seat string, permissions []governance.DeputyPermission) ([]Record, string) {
	t.Helper()
	b := rolloverLedger(t)
	b.add(RawCommand{League: "native.test", Command: "commish.role.grant",
		Args: RawArgs{People: &RawPeople{Principal: principal, Role: governance.Deputy}}}, seasonMoment(5))
	if permissions != nil {
		b.add(RawCommand{League: "native.test", Command: DeputyPermissionsSet,
			Args: RawArgs{People: &RawPeople{Principal: principal, Permissions: permissions}}}, seasonMoment(6))
	}
	if seat != "" {
		b.add(RawCommand{League: "native.test", Command: "commish.seat.replace",
			Args: RawArgs{People: &RawPeople{Franchise: seat, Principal: principal}}}, seasonMoment(7))
	}
	tradeID := b.add(RawCommand{League: "native.test", Command: TradeOffer,
		Args: RawArgs{TradeOutcome: &TradeState{Parties: []string{"franchise.01", "franchise.02"}, Status: "offered"}}}, seasonMoment(8))
	return b.records, tradeID
}

func deputyTradeRuling(command, tradeID string) RawCommand {
	return RawCommand{League: "native.test", Command: command,
		Args: RawArgs{Trade: &RawTradeArgs{ID: tradeID, Decision: "approve"}}}
}

func TestDeputyWithRuleOnTradesUsesTradeReviewerAuthorization(t *testing.T) {
	records, tradeID := deputyTradeAuthorizationRecords(t, "deputy", "franchise.03", []governance.DeputyPermission{governance.RuleOnTrades})
	if role, err := AuthorizeRecords(deputyTradeRuling(TradeReview, tradeID), "deputy", records); err != nil || role != "deputy" {
		t.Fatalf("non-party deputy trade review: role=%q err=%v", role, err)
	}
}

func TestDeputyWhoIsTradePartyCannotReviewOverrideOrRule(t *testing.T) {
	records, tradeID := deputyTradeAuthorizationRecords(t, "deputy", "franchise.01", []governance.DeputyPermission{governance.RuleOnTrades})
	for _, command := range []string{TradeReview, TradeRecuse, TradeAppealRule, TradeOverride} {
		if _, err := AuthorizeRecords(deputyTradeRuling(command, tradeID), "deputy", records); err == nil {
			t.Errorf("trade-party deputy ran %s", command)
		}
	}
}

func TestDeputyWithoutRuleOnTradesCannotReviewTrade(t *testing.T) {
	records, tradeID := deputyTradeAuthorizationRecords(t, "unticked-deputy", "", nil)
	if _, err := AuthorizeRecords(deputyTradeRuling(TradeReview, tradeID), "unticked-deputy", records); err == nil {
		t.Fatal("deputy without Rule on trades reviewed a trade")
	}
}
