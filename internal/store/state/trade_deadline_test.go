package state

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/domain"
)

// readTradeDeadlinePassed reads the §14 commissioner trade-deadline state through a WriteTx (the
// surface the TRADE op-gate uses).
func readTradeDeadlinePassed(t *testing.T, s *Store) bool {
	t.Helper()
	var got bool
	if err := s.WriteTx(context.Background(), func(w TxWriter) error {
		p, err := w.TradeDeadlinePassed(context.Background())
		got = p
		return err
	}); err != nil {
		t.Fatalf("WriteTx/TradeDeadlinePassed: %v", err)
	}
	return got
}

// setTradeDeadline runs AppendTradeDeadline through a WriteTx.
func setTradeDeadline(t *testing.T, s *Store, deadline time.Time, note string) error {
	t.Helper()
	return s.WriteTx(context.Background(), func(w TxWriter) error {
		return w.AppendTradeDeadline(context.Background(), deadline, note)
	})
}

// TestTradeDeadline_DefaultNotPassed: a fresh league with no commissioner directive reports the
// deadline NOT passed — trades are never blocked until the commissioner sets one.
func TestTradeDeadline_DefaultNotPassed(t *testing.T) {
	s := newStore(t, &fakeSource{rosters: baseRosters(t)})
	if readTradeDeadlinePassed(t, s) {
		t.Fatal("fresh league reports the trade deadline PASSED, want not-set/not-passed")
	}
}

// TestTradeDeadline_PastDeadlineBlocks: a deadline set in the past reports passed=true.
func TestTradeDeadline_PastDeadlineBlocks(t *testing.T) {
	s := newStore(t, &fakeSource{rosters: baseRosters(t)})

	past := time.Now().Add(-24 * time.Hour)
	if err := setTradeDeadline(t, s, past, "week 9 deadline"); err != nil {
		t.Fatalf("set deadline: %v", err)
	}
	if !readTradeDeadlinePassed(t, s) {
		t.Fatal("a past deadline reports not-passed, want passed")
	}

	// The directive row keeps the standing phase (from == to) and carries the meta.
	var from, to, meta string
	if err := s.pools.Read().QueryRowContext(context.Background(),
		`SELECT from_phase, to_phase, meta FROM season_phases WHERE note = 'week 9 deadline'`).
		Scan(&from, &to, &meta); err != nil {
		t.Fatalf("read directive row: %v", err)
	}
	if from != string(domain.PhaseOffseason) || to != string(domain.PhaseOffseason) {
		t.Fatalf("directive row = %s→%s, want a no-phase-change OFFSEASON→OFFSEASON", from, to)
	}
	if !strings.Contains(meta, "trade_deadline") {
		t.Fatalf("directive meta = %q, want a trade_deadline directive", meta)
	}
}

// TestTradeDeadline_FutureDeadlineDoesNotBlock: a deadline set in the future reports not-passed.
func TestTradeDeadline_FutureDeadlineDoesNotBlock(t *testing.T) {
	s := newStore(t, &fakeSource{rosters: baseRosters(t)})

	future := time.Now().Add(24 * time.Hour)
	if err := setTradeDeadline(t, s, future, "week 9 deadline"); err != nil {
		t.Fatalf("set deadline: %v", err)
	}
	if readTradeDeadlinePassed(t, s) {
		t.Fatal("a future deadline reports passed, want not-passed")
	}
}

// TestTradeDeadline_ClearRestoresNotPassed: appending a zero Deadline clears a standing past
// deadline — the commissioner's "reopen trades" action.
func TestTradeDeadline_ClearRestoresNotPassed(t *testing.T) {
	s := newStore(t, &fakeSource{rosters: baseRosters(t)})

	past := time.Now().Add(-24 * time.Hour)
	if err := setTradeDeadline(t, s, past, "week 9 deadline"); err != nil {
		t.Fatalf("set deadline: %v", err)
	}
	if !readTradeDeadlinePassed(t, s) {
		t.Fatal("past deadline did not block")
	}

	if err := setTradeDeadline(t, s, time.Time{}, "commissioner reopens trades"); err != nil {
		t.Fatalf("clear deadline: %v", err)
	}
	if readTradeDeadlinePassed(t, s) {
		t.Fatal("a cleared deadline still reports passed")
	}
}

// TestTradeDeadline_PersistsAcrossPhaseTransition: a passed deadline STAYS passed across an
// ordinary phase transition (which writes meta=”) — mirrors the signing window's persistence.
func TestTradeDeadline_PersistsAcrossPhaseTransition(t *testing.T) {
	s := newStore(t, &fakeSource{rosters: baseRosters(t)})

	past := time.Now().Add(-24 * time.Hour)
	if err := setTradeDeadline(t, s, past, "week 9 deadline"); err != nil {
		t.Fatalf("set deadline: %v", err)
	}
	if err := advance(t, s, domain.PhaseRegularSeason, "kickoff"); err != nil {
		t.Fatalf("advance phase: %v", err)
	}
	if !readTradeDeadlinePassed(t, s) {
		t.Fatal("a phase transition reset the trade deadline, want it to persist passed")
	}
}
