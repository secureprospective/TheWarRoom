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
	calls   []recordedMove
	failOn  int // 0 = never
	failErr error
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
func (w *fakeWriter) CapUsed(string) (float64, bool)            { return 0, false }
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
