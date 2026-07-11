package transactions

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/store/state"
)

// --- fakes -------------------------------------------------------------------

// recordedMove is one TxWriter call the fake captured, in order.
type recordedMove struct {
	op     string // "move" | "status"
	mflID  string
	target string // franchise for a move, status string for a status change
}

// fakeTxWriter records every op and can be told to fail on the Nth call (1-indexed),
// so a test can plant a mid-transaction failure.
type fakeTxWriter struct {
	calls     []recordedMove
	failOn    int // 0 = never
	failErr   error
	player    state.PlayerState             // the one player Player() resolves (for waiver tests)
	season    int                           // the league year Season() reports
	opCounts  map[string]int                // per-(franchise/op_kind) counters for OpCount/IncOpCount
	paidCells map[string][]state.LedgerCell // per-player PAID cells PaidCells() returns
	phase     domain.Phase                  // current season phase CurrentPhase() reports (defaults OFFSEASON)

	status      domain.PlayerStatus // the status CurrentStatus() reports for statusMFLID
	statusMFLID string              // the one player CurrentStatus() resolves (for SIGN eligibility tests)
	lockedMFLID string              // the one player ActiveBuyoutLockout() reports locked

	windowClosed bool // SigningWindowClosed() reports this (the §6 commissioner UFA-calendar state)
}

func (f *fakeTxWriter) maybeFail() error {
	if f.failOn != 0 && len(f.calls) == f.failOn {
		return f.failErr
	}
	return nil
}

func (f *fakeTxWriter) MovePlayer(_ context.Context, mflID, to string) error {
	f.calls = append(f.calls, recordedMove{op: "move", mflID: mflID, target: to})
	return f.maybeFail()
}

func (f *fakeTxWriter) SetRosterStatus(_ context.Context, mflID string, s domain.RosterStatus) error {
	f.calls = append(f.calls, recordedMove{op: "status", mflID: mflID, target: string(s)})
	return f.maybeFail()
}

func (f *fakeTxWriter) ApplyContract(_ context.Context, _ string, _ state.ContractChange) error {
	f.calls = append(f.calls, recordedMove{op: "contract"})
	return f.maybeFail()
}

func (f *fakeTxWriter) AddDeadCap(_ context.Context, e state.DeadCapEntry) error {
	f.calls = append(f.calls, recordedMove{op: "deadcap", mflID: e.MFLID, target: e.FranchiseID})
	return f.maybeFail()
}

func (f *fakeTxWriter) AddCapRelief(_ context.Context, e state.CapReliefEntry) error {
	f.calls = append(f.calls, recordedMove{op: "caprelief", target: e.FranchiseID})
	return f.maybeFail()
}

func (f *fakeTxWriter) ReleasePlayer(_ context.Context, mflID string, status domain.PlayerStatus, _ string) error {
	f.calls = append(f.calls, recordedMove{op: "release", mflID: mflID, target: string(status)})
	return f.maybeFail()
}

func (f *fakeTxWriter) RecordStatus(_ context.Context, mflID string, status domain.PlayerStatus, _ string) error {
	f.calls = append(f.calls, recordedMove{op: "status", mflID: mflID, target: string(status)})
	return f.maybeFail()
}

// CurrentStatus reports the fake's status for one player, defaulting to found=false (never
// released) so a test that doesn't set it exercises the not-a-free-agent path.
func (f *fakeTxWriter) CurrentStatus(_ context.Context, mflID string) (domain.PlayerStatus, bool, error) {
	if f.status != "" && f.statusMFLID == mflID {
		return f.status, true, nil
	}
	return "", false, nil
}

func (f *fakeTxWriter) SignContract(_ context.Context, mflID, franchiseID string, _ domain.Money, _ int, _, _ string) error {
	f.calls = append(f.calls, recordedMove{op: "sign", mflID: mflID, target: franchiseID})
	return f.maybeFail()
}

// ActiveBuyoutLockout reports the fake's lockout flag for one player (default false = not locked).
func (f *fakeTxWriter) ActiveBuyoutLockout(_ context.Context, mflID, _ string, _ int) (bool, error) {
	return f.lockedMFLID == mflID, nil
}

// fakePlayer is the one player the fake will resolve for a waiver; the zero value's
// ok=false lets a test exercise the unknown-player path.
func (f *fakeTxWriter) Player(mflID string) (state.PlayerState, bool) {
	if f.player.MFLID == mflID {
		return f.player, true
	}
	return state.PlayerState{}, false
}

func (f *fakeTxWriter) Season() int { return f.season }

// CurrentPhase reports the fake's phase, defaulting to OFFSEASON when unset so a test that
// doesn't care about the phase gate still exercises the offseason-legal ops.
func (f *fakeTxWriter) CurrentPhase(_ context.Context) (domain.Phase, error) {
	if f.phase == "" {
		return domain.PhaseOffseason, nil
	}
	return f.phase, nil
}

// AppendPhaseTransition records the transition and updates the fake's current phase, rejecting
// a no-op (mirrors the store primitive's guard so handler-level tests see the same behavior).
func (f *fakeTxWriter) AppendPhaseTransition(ctx context.Context, to domain.Phase, _ string) error {
	cur, _ := f.CurrentPhase(ctx)
	if to == cur {
		return fmt.Errorf("already in phase %q (no-op rejected)", to)
	}
	f.calls = append(f.calls, recordedMove{op: "advancephase", target: string(to)})
	if err := f.maybeFail(); err != nil {
		return err
	}
	f.phase = to
	return nil
}

// RolloverSeason records the rollover and advances the fake to OFFSEASON of the next season,
// rejecting a from-phase other than PLAYOFFS (mirrors the store primitive's guard).
func (f *fakeTxWriter) RolloverSeason(ctx context.Context, _ string) error {
	cur, _ := f.CurrentPhase(ctx)
	if cur != domain.PhasePlayoffs {
		return fmt.Errorf("only legal from PLAYOFFS (current phase is %q)", cur)
	}
	f.calls = append(f.calls, recordedMove{op: "rolloverseason"})
	if err := f.maybeFail(); err != nil {
		return err
	}
	f.season++
	f.phase = domain.PhaseOffseason
	return nil
}

// SigningWindowClosed reports the fake's commissioner UFA-calendar state (default open).
func (f *fakeTxWriter) SigningWindowClosed(_ context.Context) (bool, error) {
	return f.windowClosed, nil
}

// AppendSigningWindow records the toggle and flips the fake's window, rejecting a redundant toggle
// (mirrors the store primitive's no-op guard so handler-level tests see the same behavior).
func (f *fakeTxWriter) AppendSigningWindow(_ context.Context, open bool, _ string) error {
	if open == !f.windowClosed {
		word := "closed"
		if !f.windowClosed {
			word = "open"
		}
		return fmt.Errorf("signing window already %s (no-op rejected)", word)
	}
	f.calls = append(f.calls, recordedMove{op: "setwindow", target: fmt.Sprintf("%v", open)})
	if err := f.maybeFail(); err != nil {
		return err
	}
	f.windowClosed = !open
	return nil
}

func (f *fakeTxWriter) MoveCellMoney(_ context.Context, mflID string, _, _ int, _ domain.Money, _ string) error {
	f.calls = append(f.calls, recordedMove{op: "movecell", mflID: mflID})
	return f.maybeFail()
}

func (f *fakeTxWriter) SetCell(_ context.Context, mflID string, _ int, _ domain.Money, _ string) error {
	f.calls = append(f.calls, recordedMove{op: "setcell", mflID: mflID})
	return f.maybeFail()
}

func (f *fakeTxWriter) VoidCells(_ context.Context, mflID string, _ string) error {
	f.calls = append(f.calls, recordedMove{op: "voidcells", mflID: mflID})
	return f.maybeFail()
}

func (f *fakeTxWriter) PaidCells(_ context.Context, mflID string) ([]state.LedgerCell, error) {
	return f.paidCells[mflID], f.maybeFail()
}

func (f *fakeTxWriter) AppendExtensionYears(_ context.Context, mflID string, _ int, _ domain.Money, _ string) error {
	f.calls = append(f.calls, recordedMove{op: "appendext", mflID: mflID})
	return f.maybeFail()
}

func (f *fakeTxWriter) OpCount(_ context.Context, franchiseID, opKind string) (int, error) {
	return f.opCounts[franchiseID+"/"+opKind], nil
}

func (f *fakeTxWriter) IncOpCount(_ context.Context, franchiseID, opKind string) error {
	f.calls = append(f.calls, recordedMove{op: "incop", mflID: opKind, target: franchiseID})
	if f.opCounts == nil {
		f.opCounts = map[string]int{}
	}
	f.opCounts[franchiseID+"/"+opKind]++
	return f.maybeFail()
}

// fakeWriter is a state.Writer whose WriteTx drives the fakeTxWriter. It records whether
// WriteTx ran (to prove validation short-circuits before a transaction is opened) and
// whether the callback returned an error (to prove rollback intent propagates).
type fakeWriter struct {
	tw          *fakeTxWriter
	writeTxRan  bool
	callbackErr error
}

func (w *fakeWriter) WriteTx(_ context.Context, fn func(state.TxWriter) error) error {
	w.writeTxRan = true
	w.callbackErr = fn(w.tw)
	return w.callbackErr // the real store would roll back on a non-nil return
}

// Reader half — unused by these tests; present so *fakeWriter satisfies state.Writer.
func (w *fakeWriter) FranchiseState(string) (state.FranchiseState, bool) {
	return state.FranchiseState{}, false
}
func (w *fakeWriter) Roster(string) ([]state.PlayerState, bool) { return nil, false }
func (w *fakeWriter) CapUsed(string) (domain.Money, bool)       { return 0, false }
func (w *fakeWriter) Player(string) (state.PlayerState, bool)   { return state.PlayerState{}, false }
func (w *fakeWriter) Franchises() []string                      { return nil }

func newFake() *fakeWriter { return &fakeWriter{tw: &fakeTxWriter{}} }

// makeMoves builds n distinct trade legs (for the leg-count boundary test).
func makeMoves(n int) []PlayerMove {
	m := make([]PlayerMove, n)
	for i := range m {
		m[i] = PlayerMove{MFLID: fmt.Sprintf("p%d", i), ToFranchiseID: "0002"}
	}
	return m
}

func newCoord(t *testing.T, w state.Writer) *Coordinator {
	t.Helper()
	c, err := New(w)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	return c
}

// --- tests -------------------------------------------------------------------

// TestNew_NilWriterFails is the fail-loud construction gate: the sole mutator must be
// wired with a real writer, never a nil that would no-op every transaction.
func TestNew_NilWriterFails(t *testing.T) {
	if _, err := New(nil); err == nil {
		t.Fatal("New(nil) succeeded — a nil writer must fail at construction")
	}
}

func TestExecute_NilRequestFails(t *testing.T) {
	w := newFake()
	c := newCoord(t, w)
	if _, err := c.Execute(context.Background(), nil); err == nil {
		t.Fatal("Execute(nil) succeeded")
	}
	if w.writeTxRan {
		t.Fatal("a nil request opened a transaction — it must be rejected first")
	}
}

// TestExecute_TradeDispatchesEveryLegInOrder proves the Coordinator hands each leg of a
// trade to the tx writer, in order, and reports the count on the Receipt.
func TestExecute_TradeDispatchesEveryLegInOrder(t *testing.T) {
	w := newFake()
	c := newCoord(t, w)

	rec, err := c.Execute(context.Background(), Trade{Moves: []PlayerMove{
		{MFLID: "0001", ToFranchiseID: "0002"},
		{MFLID: "0003", ToFranchiseID: "0001"},
	}})
	if err != nil {
		t.Fatalf("Execute trade: %v", err)
	}
	if rec.Kind != KindTrade || rec.PlayersAffected != 2 {
		t.Fatalf("receipt = %+v, want KindTrade/2", rec)
	}
	got := w.tw.calls
	if len(got) != 2 ||
		got[0] != (recordedMove{"move", "0001", "0002"}) ||
		got[1] != (recordedMove{"move", "0003", "0001"}) {
		t.Fatalf("trade legs dispatched wrong/out of order: %+v", got)
	}
}

// TestExecute_AdvancePhase runs the ADVANCE_PHASE op: a valid target appends a transition
// (0 players affected), a no-op (target == current) is rejected inside the tx, and an unknown
// target is rejected before a tx is even opened.
func TestExecute_AdvancePhase(t *testing.T) {
	w := newFake()
	w.tw.phase = domain.PhaseOffseason
	c := newCoord(t, w)

	rec, err := c.Execute(context.Background(), AdvancePhase{To: domain.PhaseRegularSeason, Note: "kickoff"})
	if err != nil {
		t.Fatalf("advance to REGULAR_SEASON: %v", err)
	}
	if rec.Kind != KindAdvancePhase || rec.PlayersAffected != 0 {
		t.Fatalf("receipt = %+v, want KindAdvancePhase/0", rec)
	}
	if len(w.tw.calls) != 1 || w.tw.calls[0] != (recordedMove{op: "advancephase", target: string(domain.PhaseRegularSeason)}) {
		t.Fatalf("advance-phase call not recorded: %+v", w.tw.calls)
	}

	// No-op: advancing to the now-current phase is rejected (inside the tx).
	if _, err := c.Execute(context.Background(), AdvancePhase{To: domain.PhaseRegularSeason}); err == nil {
		t.Fatal("no-op advance succeeded, want rejection")
	}

	// Unknown target is rejected at validate — no tx opened.
	w2 := newFake()
	c2 := newCoord(t, w2)
	if _, err := c2.Execute(context.Background(), AdvancePhase{To: domain.Phase("PRESEASON")}); err == nil {
		t.Fatal("unknown target phase succeeded, want rejection")
	}
	if w2.writeTxRan {
		t.Fatal("an invalid advance-phase opened a transaction, want short-circuit before tx")
	}
}

// bogusReq is a Request with a Kind absent from opPhaseGate — the PLANT that proves the phase
// gate is default-deny. apply records a call so the test can assert it never ran.
type bogusReq struct{ tw *fakeTxWriter }

func (bogusReq) Kind() Kind      { return Kind("BOGUS") }
func (bogusReq) validate() error { return nil }
func (bogusReq) sealed()         {}
func (b bogusReq) apply(_ context.Context, _ state.TxWriter) (int, error) {
	b.tw.calls = append(b.tw.calls, recordedMove{op: "bogus-apply-ran"})
	return 1, nil
}

// TestExecute_DefaultDenyUnknownKind: an op_kind with no phase policy is rejected by the gate,
// and its apply NEVER runs (the gate is the first step inside the tx). This is the planted
// violation that proves default-deny — a gate that only ever passes proves nothing.
func TestExecute_DefaultDenyUnknownKind(t *testing.T) {
	w := newFake()
	c := newCoord(t, w)

	if _, err := c.Execute(context.Background(), bogusReq{tw: w.tw}); err == nil {
		t.Fatal("an unmapped op_kind was permitted — default-deny did not fire")
	}
	for _, cl := range w.tw.calls {
		if cl.op == "bogus-apply-ran" {
			t.Fatal("apply ran despite the phase gate denying the op")
		}
	}
}

// TestExecute_GateAllowsMappedOpEveryPhase: a shipped op mapped to all phases passes the gate
// in OFFSEASON, REGULAR_SEASON, and PLAYOFFS (the v1 no-restriction policy). The phase-
// RESTRICTION branch (current phase not in the allow-list) is proven by the §12 buyout in the
// wrong phase, in its own commit.
func TestExecute_GateAllowsMappedOpEveryPhase(t *testing.T) {
	for _, ph := range []domain.Phase{domain.PhaseOffseason, domain.PhaseRegularSeason, domain.PhasePlayoffs} {
		w := newFake()
		w.tw.phase = ph
		c := newCoord(t, w)
		if _, err := c.Execute(context.Background(), RosterStatusChange{MFLID: "0001", Status: domain.RosterTaxi}); err != nil {
			t.Fatalf("mapped op denied in phase %q: %v", ph, err)
		}
	}
}

// TestExecute_BuyoutAppliesThenBumps pins the §12 atomic order the rollback guarantee depends
// on: release → dead-cap charge → void cells → bump the per-season counter, all in one tx, with
// the counter bump LAST (a rolled-back buyout leaves no orphan increment).
func TestExecute_BuyoutAppliesThenBumps(t *testing.T) {
	w := newFake()
	w.tw.season = 2026
	w.tw.player = state.PlayerState{MFLID: "0001", FranchiseID: "0001"}
	w.tw.paidCells = map[string][]state.LedgerCell{ // remaining after 2026 = 2 → §12 rate applies
		"0001": {{Year: 2027, Salary: 6 * 100_000_000}, {Year: 2028, Salary: 6 * 100_000_000}},
	}
	c := newCoord(t, w)

	if _, err := c.Execute(context.Background(), Buyout{MFLID: "0001"}); err != nil {
		t.Fatalf("Execute buyout: %v", err)
	}
	got := w.tw.calls
	if len(got) != 4 || got[0].op != "release" || got[1].op != "deadcap" || got[2].op != "voidcells" || got[3].op != "incop" {
		t.Fatalf("buyout did not release→deadcap→void→bump: %+v", got)
	}
	if got[3].mflID != "BUYOUT" || got[3].target != "0001" {
		t.Fatalf("counter bumped with wrong op/franchise: %+v", got[3])
	}
}

// TestExecute_BuyoutCounterFailRollsBack plants a failure on the counter bump (the LAST step).
// The transaction must fail with a zero receipt — the release/charge/void applied earlier must
// NOT be reported as committed (the real store rolls the whole WriteTx back; here we prove the
// error propagates and no partial success leaks). Parity with the restructure rollback proof.
func TestExecute_BuyoutCounterFailRollsBack(t *testing.T) {
	w := newFake()
	w.tw.season = 2026
	w.tw.player = state.PlayerState{MFLID: "0001", FranchiseID: "0001"}
	w.tw.paidCells = map[string][]state.LedgerCell{
		"0001": {{Year: 2027, Salary: 6 * 100_000_000}, {Year: 2028, Salary: 6 * 100_000_000}},
	}
	w.tw.failOn = 4 // release(1) deadcap(2) voidcells(3) incop(4)
	w.tw.failErr = errors.New("counter boom")
	c := newCoord(t, w)

	rec, err := c.Execute(context.Background(), Buyout{MFLID: "0001"})
	if err == nil {
		t.Fatal("buyout succeeded despite a failing counter bump")
	}
	if rec != (Receipt{}) {
		t.Fatalf("failed buyout returned a non-zero receipt: %+v", rec)
	}
}

// TestExecute_StepErrorPropagates plants a failure on the SECOND leg. Execute must
// return the error and a ZERO receipt — never a partial success. (The actual rollback
// is the state layer's job, proven in its own tests; here we prove the error surfaces.)
func TestExecute_StepErrorPropagates(t *testing.T) {
	w := newFake()
	w.tw.failOn = 2
	w.tw.failErr = errors.New("boom")
	c := newCoord(t, w)

	rec, err := c.Execute(context.Background(), Trade{Moves: []PlayerMove{
		{MFLID: "0001", ToFranchiseID: "0002"},
		{MFLID: "0003", ToFranchiseID: "0001"},
	}})
	if err == nil {
		t.Fatal("Execute succeeded despite a failing leg")
	}
	if rec != (Receipt{}) {
		t.Fatalf("a failed transaction returned a non-zero receipt: %+v", rec)
	}
	if !w.writeTxRan {
		t.Fatal("WriteTx did not run for a valid-but-failing trade")
	}
}

// TestExecute_InvalidTradeRejectedBeforeTx proves validation short-circuits: a
// structurally broken request never opens a transaction.
func TestExecute_InvalidTradeRejectedBeforeTx(t *testing.T) {
	for _, tc := range []struct {
		name string
		req  Request
	}{
		{"no moves", Trade{}},
		{"empty target", Trade{Moves: []PlayerMove{{MFLID: "0001"}}}},
		{"duplicate player", Trade{Moves: []PlayerMove{
			{MFLID: "0001", ToFranchiseID: "0002"},
			{MFLID: "0001", ToFranchiseID: "0003"},
		}}},
		{"too many legs", Trade{Moves: makeMoves(maxTradeLegs + 1)}},
		{"empty status player", RosterStatusChange{Status: domain.RosterActive}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			w := newFake()
			c := newCoord(t, w)
			if _, err := c.Execute(context.Background(), tc.req); err == nil {
				t.Fatal("invalid request accepted")
			}
			if w.writeTxRan {
				t.Fatal("invalid request opened a transaction (validation must run first)")
			}
		})
	}
}

func TestExecute_RosterStatusChange(t *testing.T) {
	w := newFake()
	c := newCoord(t, w)

	rec, err := c.Execute(context.Background(), RosterStatusChange{
		MFLID: "0001", Status: domain.RosterTaxi,
	})
	if err != nil {
		t.Fatalf("Execute status: %v", err)
	}
	if rec.Kind != KindRosterStatus || rec.PlayersAffected != 1 {
		t.Fatalf("receipt = %+v, want KindRosterStatus/1", rec)
	}
	if len(w.tw.calls) != 1 || w.tw.calls[0] != (recordedMove{"status", "0001", string(domain.RosterTaxi)}) {
		t.Fatalf("status not dispatched: %+v", w.tw.calls)
	}
}

// TestExecute_WaiverReleasesThenCharges proves a waiver dispatches the roster release
// AND the dead-cap charge, in that order, through the shared tx writer.
func TestExecute_WaiverReleasesThenCharges(t *testing.T) {
	w := newFake()
	w.tw.season = 2026
	w.tw.player = state.PlayerState{
		MFLID: "0001", FranchiseID: "0001", Salary: 10 * 100_000_000, CapSalary: 10 * 100_000_000,
		ExpirationYear: 2028,
	}
	c := newCoord(t, w)

	rec, err := c.Execute(context.Background(), Waiver{MFLID: "0001"})
	if err != nil {
		t.Fatalf("Execute waiver: %v", err)
	}
	if rec.Kind != KindWaiver || rec.PlayersAffected != 1 {
		t.Fatalf("receipt = %+v, want KindWaiver/1", rec)
	}
	got := w.tw.calls
	if len(got) != 3 || got[0].op != "release" || got[1].op != "deadcap" || got[2].op != "voidcells" {
		t.Fatalf("waiver did not release-then-charge-then-void: %+v", got)
	}
	if got[0].mflID != "0001" || got[1].mflID != "0001" || got[1].target != "0001" || got[2].mflID != "0001" {
		t.Fatalf("waiver dispatched wrong player/franchise: %+v", got)
	}
}

// TestExecute_WaiverUnknownPlayerFails proves a cut of a non-rostered player fails the
// transaction (and, since the release never runs, nothing is dispatched).
func TestExecute_WaiverUnknownPlayerFails(t *testing.T) {
	w := newFake()
	w.tw.season = 2026 // no player set on the fake → Player() returns ok=false
	c := newCoord(t, w)

	if _, err := c.Execute(context.Background(), Waiver{MFLID: "9999"}); err == nil {
		t.Fatal("waiver of an unknown player succeeded")
	}
	for _, call := range w.tw.calls {
		if call.op == "release" || call.op == "deadcap" {
			t.Fatalf("unknown-player waiver still dispatched %q", call.op)
		}
	}
}

func TestExecute_WaiverEmptyPlayerRejectedBeforeTx(t *testing.T) {
	w := newFake()
	c := newCoord(t, w)
	if _, err := c.Execute(context.Background(), Waiver{MFLID: "  "}); err == nil {
		t.Fatal("empty-id waiver accepted")
	}
	if w.writeTxRan {
		t.Fatal("empty-id waiver opened a transaction (validation must run first)")
	}
}

// TestExecute_RestructureAppliesThenBumps proves the §11 handler dispatches in the
// atomic order the rollback guarantee depends on: it applies the contract change FIRST,
// then bumps the per-season counter — both inside the one transaction.
func TestExecute_RestructureAppliesThenBumps(t *testing.T) {
	w := newFake()
	w.tw.season = 2026
	w.tw.player = state.PlayerState{
		MFLID: "0001", FranchiseID: "0001", Salary: 6 * 100_000_000, CapSalary: 6 * 100_000_000, // $6M → tier max $2M
		ExpirationYear: 2028, // a future paid year exists to absorb the move
	}
	c := newCoord(t, w)

	if _, err := c.Execute(context.Background(), Restructure{MFLID: "0001", Move: 2 * 100_000_000}); err != nil {
		t.Fatalf("Execute restructure: %v", err)
	}
	got := w.tw.calls
	// Dual-write order: apply the legacy contract, move money between cells, then bump the
	// per-season counter — all inside the one transaction.
	if len(got) != 3 || got[0].op != "contract" || got[1].op != "movecell" || got[2].op != "incop" {
		t.Fatalf("restructure did not apply→move→bump: %+v", got)
	}
	if got[2].mflID != "RESTRUCTURE" || got[2].target != "0001" {
		t.Fatalf("counter bumped with wrong op/franchise: %+v", got[2])
	}
}

// TestExecute_RestructureCounterFailRollsBack plants a failure on the counter bump (the
// LAST step). The transaction must fail and return a zero receipt — the contract change
// applied one step earlier must NOT be reported as committed (the real store rolls the
// whole WriteTx back; here we prove the error propagates and no partial success leaks).
func TestExecute_RestructureCounterFailRollsBack(t *testing.T) {
	w := newFake()
	w.tw.season = 2026
	w.tw.player = state.PlayerState{MFLID: "0001", FranchiseID: "0001", Salary: 6 * 100_000_000, CapSalary: 6 * 100_000_000, ExpirationYear: 2028}
	w.tw.failOn = 3 // fail the incop (call 1 = contract, call 2 = movecell, call 3 = incop)
	w.tw.failErr = errors.New("counter boom")
	c := newCoord(t, w)

	rec, err := c.Execute(context.Background(), Restructure{MFLID: "0001", Move: 1 * 100_000_000})
	if err == nil {
		t.Fatal("restructure succeeded despite a failing counter bump")
	}
	if rec != (Receipt{}) {
		t.Fatalf("failed restructure returned a non-zero receipt: %+v", rec)
	}
}

func TestRestructure_ValidateRejectsBeforeTx(t *testing.T) {
	for _, tc := range []struct {
		name string
		req  Restructure
	}{
		{"empty id", Restructure{MFLID: "  ", Move: 1 * 100_000_000}},
		{"zero move", Restructure{MFLID: "0001", Move: 0}},
		{"negative move", Restructure{MFLID: "0001", Move: -100}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			w := newFake()
			c := newCoord(t, w)
			if _, err := c.Execute(context.Background(), tc.req); err == nil {
				t.Fatal("invalid restructure accepted")
			}
			if w.writeTxRan {
				t.Fatal("invalid restructure opened a transaction (validation must run first)")
			}
		})
	}
}

// TestExecute_ReceiptTimestampUsesClock pins the Receipt time to the injected clock.
func TestExecute_ReceiptTimestampUsesClock(t *testing.T) {
	w := newFake()
	c := newCoord(t, w)
	fixed := time.Date(2026, 7, 3, 12, 0, 0, 0, time.UTC)
	c.now = func() time.Time { return fixed }

	rec, err := c.Execute(context.Background(), RosterStatusChange{MFLID: "0001", Status: domain.RosterActive})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if !rec.At.Equal(fixed) {
		t.Fatalf("receipt time = %v, want %v", rec.At, fixed)
	}
}
