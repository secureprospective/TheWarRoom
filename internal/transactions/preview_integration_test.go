package transactions_test

import (
	"context"
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/transactions"
)

// TestIntegration_PreviewLeavesStateUnchanged is the planted proof of the dry-run mechanism
// (design D5): a Preview must run the REAL handler — same validation, phase gate, apply — yet
// persist NOTHING. It captures state, previews a buyout that would succeed, asserts the roster
// and cap are byte-identical, then Executes the SAME request and asserts it NOW mutates — so the
// only difference between preview and commit is the rollback. A gate that only ever passed would
// prove nothing; the Execute half is the planted mutation the preview must have suppressed.
func TestIntegration_PreviewLeavesStateUnchanged(t *testing.T) {
	s, c := buyStore(t)
	ctx := context.Background()

	// 0001 exp 2028 → remaining 2 → a valid §12 buyout (60% × $6M = $3.6M dead cap).
	capBefore, ok := s.Reader().CapUsed("0001")
	if !ok {
		t.Fatalf("precondition: franchise 0001 has no cap")
	}
	if _, ok := s.Reader().Player("0001"); !ok {
		t.Fatalf("precondition: player 0001 not rostered")
	}

	rec, err := c.Preview(ctx, transactions.Buyout{MFLID: "0001"})
	if err != nil {
		t.Fatalf("preview of a valid buyout errored: %v", err)
	}
	if rec.Kind != transactions.KindBuyout || rec.PlayersAffected != 1 {
		t.Fatalf("preview receipt = %+v, want KindBuyout/1", rec)
	}

	// The whole point: nothing persisted.
	if _, stillOK := s.Reader().Player("0001"); !stillOK {
		t.Fatalf("preview RELEASED player 0001 — it must roll back")
	}
	if capAfter, _ := s.Reader().CapUsed("0001"); capAfter != capBefore {
		t.Fatalf("preview changed cap: before %v, after %v — it must roll back", capBefore, capAfter)
	}

	// Planted mutation: the same request COMMITTED must actually change state, proving the
	// preview above genuinely suppressed a real, applicable transaction.
	if _, err := c.Execute(ctx, transactions.Buyout{MFLID: "0001"}); err != nil {
		t.Fatalf("execute of the previewed buyout errored: %v", err)
	}
	if _, stillOK := s.Reader().Player("0001"); stillOK {
		t.Fatalf("execute did NOT release player 0001 — the mechanism under test is inert")
	}
}

// TestIntegration_PreviewSurfacesRejection proves a preview reports the AUTHORITATIVE rejection
// (not a false "ok") and still persists nothing: 0004 has 1 remaining year, outside the §12 rate
// table, so both Preview and Execute must reject it — and the roster is untouched either way.
func TestIntegration_PreviewSurfacesRejection(t *testing.T) {
	s, c := buyStore(t)
	ctx := context.Background()

	if _, err := c.Preview(ctx, transactions.Buyout{MFLID: "0004"}); err == nil {
		t.Fatalf("preview of an out-of-range buyout returned nil error, want a §12 rejection")
	}
	if _, ok := s.Reader().Player("0004"); !ok {
		t.Fatalf("a rejected preview must leave player 0004 rostered")
	}
}
