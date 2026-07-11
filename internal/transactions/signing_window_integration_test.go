package transactions_test

import (
	"context"
	"strings"
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/transactions"
)

// TestIntegration_UFAWindowGatesSigning proves the COMMISSIONER UFA-CALENDAR (§6, Free_Agency_Design
// Q4) end-to-end: a commissioner-CLOSED window blocks a signing that is otherwise legal (right phase,
// signable free agent, salary above the floor), and reopening the window lets the same signing land.
// The window rides season_phases.meta via SET_SIGNING_WINDOW and does not change the phase.
func TestIntegration_UFAWindowGatesSigning(t *testing.T) {
	s, c := signStore(t)
	ctx := context.Background()
	dir := draftDir{} // 0011 absent → rookie floor ($330k); $700k clears it

	// Cut 0011 into the pool so he is a signable free agent.
	waiveIntoPool(t, c, "0011")
	if !contains(poolOf(t, s), "0011") {
		t.Fatal("0011 not in the free-agent pool after waiver")
	}

	// Commissioner CLOSES the window (e.g. Super Bowl kickoff).
	if _, err := c.Execute(ctx, transactions.SetSigningWindow{Open: false, Note: "super bowl kickoff"}); err != nil {
		t.Fatalf("close signing window: %v", err)
	}

	// A signing that clears every OTHER gate is now blocked purely by the closed window.
	sign := transactions.Sign{MFLID: "0011", FranchiseID: "0002", Salary: signUSD(700_000), Years: 2}
	_, err := c.ExecuteSign(ctx, sign, dir)
	if err == nil {
		t.Fatal("signing succeeded with the window CLOSED, want a rejection")
	}
	if !strings.Contains(err.Error(), "signing window is closed") {
		t.Fatalf("closed-window rejection = %v, want the §6 UFA-calendar window error", err)
	}
	if !contains(poolOf(t, s), "0011") {
		t.Fatal("0011 left the pool despite the signing being rejected")
	}

	// Commissioner REOPENS the window — the same signing now lands.
	if _, err := c.Execute(ctx, transactions.SetSigningWindow{Open: true, Note: "free agency opens"}); err != nil {
		t.Fatalf("reopen signing window: %v", err)
	}
	if _, err := c.ExecuteSign(ctx, sign, dir); err != nil {
		t.Fatalf("signing rejected with the window REOPENED: %v", err)
	}
	if contains(poolOf(t, s), "0011") {
		t.Fatal("0011 still in the pool after a successful signing")
	}
}

// TestIntegration_UFAWindowRedundantToggleRejected proves the no-silent-no-op guard surfaces through
// the Coordinator: toggling the window to a state it is already in is rejected.
func TestIntegration_UFAWindowRedundantToggleRejected(t *testing.T) {
	_, c := signStore(t)
	ctx := context.Background()

	// Default state is OPEN → opening again is a no-op.
	if _, err := c.Execute(ctx, transactions.SetSigningWindow{Open: true}); err == nil {
		t.Fatal("redundant OPEN succeeded, want a no-op rejection")
	}

	if _, err := c.Execute(ctx, transactions.SetSigningWindow{Open: false, Note: "close"}); err != nil {
		t.Fatalf("close window: %v", err)
	}
	// Now closed → closing again is a no-op.
	if _, err := c.Execute(ctx, transactions.SetSigningWindow{Open: false}); err == nil {
		t.Fatal("redundant CLOSE succeeded, want a no-op rejection")
	}
}
