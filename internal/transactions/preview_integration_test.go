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

	// The pre-commit dollar breakdown: the §12 dead-cap charge (60% × $6M = $3.6M) must travel
	// OUT of the rolled-back preview as one POSITIVE cap delta — the figure the confirm quote
	// shows before any write. It survives the rollback because it is the handler's RETURN, not a
	// post-apply re-read (the in-tx read wall).
	if len(rec.CapDeltas) != 1 {
		t.Fatalf("preview cap breakdown = %+v, want exactly 1 line (the §12 dead-cap charge)", rec.CapDeltas)
	}
	if got := rec.CapDeltas[0]; got.Cents != 3_600_000*100 || got.FranchiseID == "" || got.Reason == "" {
		t.Fatalf("preview dead-cap delta = %+v, want +$3.6M (360000000c) with a franchise + reason", got)
	}
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

// TestIntegration_PreviewTagLeavesStateUnchanged: a tag preview resolves the same top-5 price as
// the commit (both go through Tag.resolve) and runs the real handler, yet 0022 stays untagged and
// the franchise cap is unchanged. Executing the same request then tags him and moves the cap, so
// the rollback is the only difference.
func TestIntegration_PreviewTagLeavesStateUnchanged(t *testing.T) {
	s, c, dir := tagStore(t)
	ctx := context.Background()

	capBefore, ok := s.Reader().CapUsed("0002")
	if !ok {
		t.Fatalf("precondition: franchise 0002 has no cap")
	}

	rec, err := c.WithDirectory(dir).Preview(ctx, transactions.Tag{MFLID: "0022"})
	if err != nil {
		t.Fatalf("preview of a valid tag errored: %v", err)
	}
	if rec.Kind != transactions.KindTag || rec.PlayersAffected != 1 {
		t.Fatalf("preview receipt = %+v, want KindTag/1", rec)
	}

	// Nothing persisted: 0022 is not tagged and the cap has not moved.
	if p, _ := s.Reader().Player("0022"); p.IsTagged {
		t.Fatalf("preview TAGGED player 0022 — it must roll back")
	}
	if capAfter, _ := s.Reader().CapUsed("0002"); capAfter != capBefore {
		t.Fatalf("preview changed cap: before %v, after %v — it must roll back", capBefore, capAfter)
	}

	// Planted mutation: the same tag COMMITTED must actually flag + move the cap.
	if _, err := c.WithDirectory(dir).Execute(ctx, transactions.Tag{MFLID: "0022"}); err != nil {
		t.Fatalf("execute of the previewed tag errored: %v", err)
	}
	if p, _ := s.Reader().Player("0022"); !p.IsTagged {
		t.Fatalf("execute did NOT tag player 0022 — the mechanism under test is inert")
	}
}

// TestIntegration_PreviewExtensionLeavesStateUnchanged: a +2 extension preview of 0001 resolves
// the same floor as the commit (both go through Extension.resolve), yet the term stays 2028 and no
// cells are appended. Executing the same request then lengthens the term to 2030.
func TestIntegration_PreviewExtensionLeavesStateUnchanged(t *testing.T) {
	s, c, dir := extStore(t)
	ctx := context.Background()

	if p, _ := s.Player("0001"); p.ExpirationYear != 2028 {
		t.Fatalf("precondition: 0001 expiration = %d, want 2028", p.ExpirationYear)
	}

	rec, err := c.WithDirectory(dir).Preview(ctx, transactions.Extension{MFLID: "0001", AddedYears: 2})
	if err != nil {
		t.Fatalf("preview of a valid extension errored: %v", err)
	}
	if rec.Kind != transactions.KindExtension || rec.PlayersAffected != 1 {
		t.Fatalf("preview receipt = %+v, want KindExtension/1", rec)
	}

	// Nothing persisted: the term is unchanged and no extension cells were appended.
	if p, _ := s.Player("0001"); p.ExpirationYear != 2028 {
		t.Fatalf("preview LENGTHENED 0001 to %d — it must roll back to 2028", p.ExpirationYear)
	}
	cells, err := s.LedgerCells(ctx, "0001")
	if err != nil {
		t.Fatalf("LedgerCells: %v", err)
	}
	for _, y := range []int{2029, 2030} {
		if _, exists := cells[y]; exists {
			t.Fatalf("preview appended cell %d — it must roll back", y)
		}
	}

	// Planted mutation: the same extension COMMITTED must actually lengthen the term.
	if _, err := c.WithDirectory(dir).Execute(ctx, transactions.Extension{MFLID: "0001", AddedYears: 2}); err != nil {
		t.Fatalf("execute of the previewed extension errored: %v", err)
	}
	if p, _ := s.Player("0001"); p.ExpirationYear != 2030 {
		t.Fatalf("execute did NOT lengthen 0001 to 2030 (got %d) — the mechanism under test is inert", p.ExpirationYear)
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
