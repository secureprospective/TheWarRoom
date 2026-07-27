package transactions

import (
	"context"
	"errors"
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/store/state"
)

// This file exercises the roster/position/taxi/IR enforcement THROUGH the real Coordinator.Execute
// path with a non-nil policy — the gap DeepSeek's Session 2 review flagged as MAJOR (every existing
// integration test wires policy=nil, so a bug in the enforcement math is invisible to them). It is
// exactly this gap that hid the real BLOCKER the same review found: Trade's per-position net-delta
// double-subtracted outgoingByPos, undercounting the post-trade position count by outgoingByPos[pos]
// and letting an over-limit trade commit. TestTrade_RejectsOverPositionLimit_NetDelta pins that fix.

// posPolicy is a RosterPolicy with a fixed position resolution map, for tests that need real
// per-position gating (fakeRosterPolicy in policy_test.go always reports PosFlag/unresolved).
type posPolicy struct {
	fakeRosterPolicy
	positions map[string]domain.Position
}

func (p *posPolicy) Position(_ context.Context, mflID string) (domain.Position, bool) {
	pos, ok := p.positions[mflID]
	return pos, ok
}

// seededWriter is a state.Writer + state.TxWriter fake seeded with a fixed roster, for exercising
// enforcement through Coordinator.Execute. It embeds *fakeTxWriter for every op the shared fake
// already stubs (CapLedgerWriter/LedgerWriter/SeasonScope/StatusWriter/CalendarWriter, etc.) and
// overrides only Roster/Player/MovePlayer/SetRosterStatus/CurrentPhase/LogTradeNote — the shared
// fakeWriter/fakeTxWriter always report Roster/Player as empty, which is fine for the tests that
// predate Session 2 but hides the position/roster-size math this file needs to actually exercise.
type seededWriter struct {
	*fakeTxWriter
	rosters map[string][]state.PlayerState // franchiseID -> roster
	players map[string]state.PlayerState   // mflID -> player (for Player() lookups)
}

func newSeededWriter(rosters map[string][]state.PlayerState, players map[string]state.PlayerState) *seededWriter {
	return &seededWriter{fakeTxWriter: &fakeTxWriter{}, rosters: rosters, players: players}
}

func (w *seededWriter) WriteTx(_ context.Context, fn func(state.TxWriter) error) error {
	return fn(w)
}
func (w *seededWriter) FranchiseState(string) (state.FranchiseState, bool) {
	return state.FranchiseState{}, false
}
func (w *seededWriter) Roster(franchiseID string) ([]state.PlayerState, bool) {
	r, ok := w.rosters[franchiseID]
	return r, ok
}
func (w *seededWriter) CapUsed(string) (domain.Money, bool) { return 0, false }
func (w *seededWriter) Player(mflID string) (state.PlayerState, bool) {
	p, ok := w.players[mflID]
	return p, ok
}
func (w *seededWriter) Franchises() []string { return nil }

func (w *seededWriter) CurrentPhase(context.Context) (domain.Phase, error) {
	return domain.PhaseOffseason, nil
}
func (w *seededWriter) MovePlayer(_ context.Context, mflID, to string) error {
	p := w.players[mflID]
	p.FranchiseID = to
	w.players[mflID] = p
	return nil
}
func (w *seededWriter) SetRosterStatus(_ context.Context, mflID string, s domain.RosterStatus) error {
	p := w.players[mflID]
	p.RosterStatus = s
	w.players[mflID] = p
	return nil
}
func (w *seededWriter) LogTradeNote(context.Context, string, string, []string) error { return nil }

func newSeededPolicy(size, taxi, ir int, posCaps map[domain.Position]int, positions map[string]domain.Position) *posPolicy {
	return &posPolicy{
		fakeRosterPolicy: fakeRosterPolicy{rosterSize: size, taxiSquad: taxi, ir: ir, positionCaps: posCaps},
		positions:        positions,
	}
}

// TestTrade_RejectsOverPositionLimit_NetDelta pins the DeepSeek-flagged BLOCKER fix: franchise 0001
// holds 3 QBs (cap 3), and a trade sends 1 QB out while receiving 2 QBs in — net position count
// after the trade is 3-1+2=4, which must be rejected. The buggy formula computed
// (committedAtPos-out)+(inc-out) = 3, which incorrectly passed.
func TestTrade_RejectsOverPositionLimit_NetDelta(t *testing.T) {
	roster0001 := []state.PlayerState{
		{MFLID: "q1", FranchiseID: "0001", RosterStatus: domain.RosterActive},
		{MFLID: "q2", FranchiseID: "0001", RosterStatus: domain.RosterActive},
		{MFLID: "qOut", FranchiseID: "0001", RosterStatus: domain.RosterActive},
	}
	w := newSeededWriter(
		map[string][]state.PlayerState{
			"0001": roster0001,
			"0002": {},
		},
		map[string]state.PlayerState{
			"q1":   {MFLID: "q1", FranchiseID: "0001", RosterStatus: domain.RosterActive},
			"q2":   {MFLID: "q2", FranchiseID: "0001", RosterStatus: domain.RosterActive},
			"qOut": {MFLID: "qOut", FranchiseID: "0001", RosterStatus: domain.RosterActive},
			"qIn1": {MFLID: "qIn1", FranchiseID: "0002", RosterStatus: domain.RosterActive},
			"qIn2": {MFLID: "qIn2", FranchiseID: "0002", RosterStatus: domain.RosterActive},
		},
	)
	policy := newSeededPolicy(0, 0, 0,
		map[domain.Position]int{domain.PosQB: 3},
		map[string]domain.Position{"q1": domain.PosQB, "q2": domain.PosQB, "qOut": domain.PosQB, "qIn1": domain.PosQB, "qIn2": domain.PosQB},
	)

	c, err := New(w, policy)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	trade := Trade{
		Moves: []PlayerMove{
			{MFLID: "qOut", ToFranchiseID: "0002"},
			{MFLID: "qIn1", ToFranchiseID: "0001"},
			{MFLID: "qIn2", ToFranchiseID: "0001"},
		},
		PicksNote: "test",
		Rationale: "test",
	}
	_, err = c.Execute(context.Background(), trade)
	if err == nil {
		t.Fatal("trade would leave franchise 0001 with 4 QBs against a cap of 3 — must be rejected")
	}
	if !isRosterLimitReject(err) {
		t.Fatalf("rejection should be an errRosterLimit, got %T: %v", err, err)
	}
}

// TestTrade_AllowsWithinPositionLimit_NetDelta is the sibling proof that the fix does not
// over-reject: a like-for-like QB swap (1 out, 1 in) at the cap must still commit.
func TestTrade_AllowsWithinPositionLimit_NetDelta(t *testing.T) {
	w := newSeededWriter(
		map[string][]state.PlayerState{
			"0001": {
				{MFLID: "q1", FranchiseID: "0001", RosterStatus: domain.RosterActive},
				{MFLID: "q2", FranchiseID: "0001", RosterStatus: domain.RosterActive},
				{MFLID: "qOut", FranchiseID: "0001", RosterStatus: domain.RosterActive},
			},
			"0002": {},
		},
		map[string]state.PlayerState{
			"q1":   {MFLID: "q1", FranchiseID: "0001", RosterStatus: domain.RosterActive},
			"q2":   {MFLID: "q2", FranchiseID: "0001", RosterStatus: domain.RosterActive},
			"qOut": {MFLID: "qOut", FranchiseID: "0001", RosterStatus: domain.RosterActive},
			"qIn1": {MFLID: "qIn1", FranchiseID: "0002", RosterStatus: domain.RosterActive},
		},
	)
	policy := newSeededPolicy(0, 0, 0,
		map[domain.Position]int{domain.PosQB: 3},
		map[string]domain.Position{"q1": domain.PosQB, "q2": domain.PosQB, "qOut": domain.PosQB, "qIn1": domain.PosQB},
	)
	c, err := New(w, policy)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	trade := Trade{
		Moves: []PlayerMove{
			{MFLID: "qOut", ToFranchiseID: "0002"},
			{MFLID: "qIn1", ToFranchiseID: "0001"},
		},
		PicksNote: "test",
		Rationale: "test",
	}
	if _, err := c.Execute(context.Background(), trade); err != nil {
		t.Fatalf("a like-for-like QB swap at the position cap must still be legal, got: %v", err)
	}
}

// TestRosterStatusChange_RejectsOverTaxiCap pins the taxi-squad slot enforcement through the real
// Coordinator path: a franchise already at its taxi cap must reject one more move into taxi.
func TestRosterStatusChange_RejectsOverTaxiCap(t *testing.T) {
	w := newSeededWriter(
		map[string][]state.PlayerState{
			"0001": {
				{MFLID: "t1", FranchiseID: "0001", RosterStatus: domain.RosterTaxi},
				{MFLID: "p1", FranchiseID: "0001", RosterStatus: domain.RosterActive},
			},
		},
		map[string]state.PlayerState{
			"t1": {MFLID: "t1", FranchiseID: "0001", RosterStatus: domain.RosterTaxi},
			"p1": {MFLID: "p1", FranchiseID: "0001", RosterStatus: domain.RosterActive},
		},
	)
	policy := newSeededPolicy(0, 1 /* taxi cap */, 0, nil, nil)
	c, err := New(w, policy)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	change := RosterStatusChange{MFLID: "p1", Status: domain.RosterTaxi}
	_, err = c.Execute(context.Background(), change)
	if err == nil {
		t.Fatal("moving a second player to taxi against a cap of 1 must be rejected")
	}
	if !isRosterLimitReject(err) {
		t.Fatalf("rejection should be an errRosterLimit, got %T: %v", err, err)
	}
}

func TestIsRosterLimitReject_UnwrapsThroughFmtErrorf(t *testing.T) {
	base := &errRosterLimit{detail: "roster limit: test"}
	wrapped := errors.New("outer: " + base.Error())
	if isRosterLimitReject(wrapped) {
		t.Fatal("a plain-string-wrapped error (not errors.Wrap-style) must not be misdetected as errRosterLimit")
	}
}
