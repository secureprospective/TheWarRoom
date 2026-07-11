package state

import (
	"context"
	"strings"
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/domain"
)

// readWindowClosed reads the commissioner UFA-calendar state through a WriteTx (the surface the
// SIGN op-gate uses).
func readWindowClosed(t *testing.T, s *Store) bool {
	t.Helper()
	var got bool
	if err := s.WriteTx(context.Background(), func(w TxWriter) error {
		c, err := w.SigningWindowClosed(context.Background())
		got = c
		return err
	}); err != nil {
		t.Fatalf("WriteTx/SigningWindowClosed: %v", err)
	}
	return got
}

// setWindow runs AppendSigningWindow through a WriteTx.
func setWindow(t *testing.T, s *Store, open bool, note string) error {
	t.Helper()
	return s.WriteTx(context.Background(), func(w TxWriter) error {
		return w.AppendSigningWindow(context.Background(), open, note)
	})
}

// TestSigningWindow_DefaultOpen: a fresh league with no commissioner directive reports the window
// OPEN (closed=false) — the v1 posture where SIGN's only phase restriction is the playoffs block.
func TestSigningWindow_DefaultOpen(t *testing.T) {
	s := newStore(t, &fakeSource{rosters: baseRosters(t)})
	if readWindowClosed(t, s) {
		t.Fatal("fresh league reports the signing window CLOSED, want open by default")
	}
}

// TestSigningWindow_CloseReopen: closing sets the window closed; reopening clears it. Each toggle
// is a from==to directive row (the phase is unchanged) carrying the ufa_window meta.
func TestSigningWindow_CloseReopen(t *testing.T) {
	s := newStore(t, &fakeSource{rosters: baseRosters(t)})

	if err := setWindow(t, s, false, "super bowl kickoff"); err != nil {
		t.Fatalf("close window: %v", err)
	}
	if !readWindowClosed(t, s) {
		t.Fatal("after close, window reports open, want closed")
	}

	// The directive row keeps the standing phase (from == to == OFFSEASON) and carries the meta.
	var from, to, meta string
	if err := s.pools.Read().QueryRowContext(context.Background(),
		`SELECT from_phase, to_phase, meta FROM season_phases WHERE note = 'super bowl kickoff'`).
		Scan(&from, &to, &meta); err != nil {
		t.Fatalf("read directive row: %v", err)
	}
	if from != string(domain.PhaseOffseason) || to != string(domain.PhaseOffseason) {
		t.Fatalf("directive row = %s→%s, want a no-phase-change OFFSEASON→OFFSEASON", from, to)
	}
	if !strings.Contains(meta, "ufa_window") || !strings.Contains(meta, "closed") {
		t.Fatalf("directive meta = %q, want a closed ufa_window directive", meta)
	}

	if err := setWindow(t, s, true, "free agency opens"); err != nil {
		t.Fatalf("reopen window: %v", err)
	}
	if readWindowClosed(t, s) {
		t.Fatal("after reopen, window reports closed, want open")
	}
}

// TestSigningWindow_RedundantToggleRejected: setting the window to a state it is already in is a
// no-op and is rejected (the no-silent-no-op house rule), leaving the state unchanged.
func TestSigningWindow_RedundantToggleRejected(t *testing.T) {
	s := newStore(t, &fakeSource{rosters: baseRosters(t)})

	// Already open (default) → opening again is rejected.
	if err := setWindow(t, s, true, "redundant open"); err == nil {
		t.Fatal("redundant OPEN succeeded, want a no-op rejection")
	} else if !strings.Contains(err.Error(), "no-op") {
		t.Fatalf("redundant-open error = %v, want a no-op rejection", err)
	}

	if err := setWindow(t, s, false, "close"); err != nil {
		t.Fatalf("close window: %v", err)
	}
	// Already closed → closing again is rejected.
	if err := setWindow(t, s, false, "redundant close"); err == nil {
		t.Fatal("redundant CLOSE succeeded, want a no-op rejection")
	}
	if !readWindowClosed(t, s) {
		t.Fatal("a rejected redundant close changed the state, want still closed")
	}
}

// TestSigningWindow_PersistsAcrossPhaseTransition: a closed window STAYS closed across an ordinary
// phase transition (which writes meta=”) — the directive persists until the commissioner reopens
// it, per Free_Agency_Design Q4 ("stays closed until the commissioner reopens it").
func TestSigningWindow_PersistsAcrossPhaseTransition(t *testing.T) {
	s := newStore(t, &fakeSource{rosters: baseRosters(t)})

	if err := setWindow(t, s, false, "close"); err != nil {
		t.Fatalf("close window: %v", err)
	}
	// An ordinary phase advance (meta='') must NOT reset the window.
	if err := advance(t, s, domain.PhaseRegularSeason, "kickoff"); err != nil {
		t.Fatalf("advance phase: %v", err)
	}
	if !readWindowClosed(t, s) {
		t.Fatal("a phase transition reset the signing window, want it to persist closed")
	}
}
