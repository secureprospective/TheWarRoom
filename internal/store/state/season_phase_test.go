package state

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/domain"
)

// readPhase reads the current phase through a WriteTx (the surface the op-gate uses).
func readPhase(t *testing.T, s *Store) domain.Phase {
	t.Helper()
	var got domain.Phase
	if err := s.WriteTx(context.Background(), func(w TxWriter) error {
		p, err := w.CurrentPhase(context.Background())
		got = p
		return err
	}); err != nil {
		t.Fatalf("WriteTx/CurrentPhase: %v", err)
	}
	return got
}

// TestSeedInitialPhaseIsOffseason: a fresh store seeds the genesis phase row → OFFSEASON,
// realizing the "offseason = start of season N" invariant.
func TestSeedInitialPhaseIsOffseason(t *testing.T) {
	s := newStore(t, &fakeSource{rosters: baseRosters(t)})
	if got := readPhase(t, s); got != domain.PhaseOffseason {
		t.Fatalf("current phase = %q, want %q", got, domain.PhaseOffseason)
	}
}

// TestSeedInitialPhaseIdempotent: once a transition history exists, re-running the seed must
// NOT append a second genesis row or revert the phase — the seed only fires on an empty log.
func TestSeedInitialPhaseIdempotent(t *testing.T) {
	s := newStore(t, &fakeSource{rosters: baseRosters(t)})
	ctx := context.Background()

	// Simulate a later ADVANCE_PHASE (the op lands in commit 2) by appending a transition row.
	now := time.Now().UTC().Format(time.RFC3339)
	if _, err := s.pools.Write().ExecContext(ctx, `
INSERT INTO season_phases (league_id, season, from_phase, to_phase, note, meta, at)
VALUES (?, ?, ?, ?, 'test-advance', '', ?)`,
		s.leagueID, s.season, string(domain.PhaseOffseason), string(domain.PhaseRegularSeason), now); err != nil {
		t.Fatalf("append transition: %v", err)
	}
	if got := readPhase(t, s); got != domain.PhaseRegularSeason {
		t.Fatalf("after advance, phase = %q, want %q", got, domain.PhaseRegularSeason)
	}

	// Re-seed: must be a no-op (history exists), leaving the phase at REGULAR_SEASON.
	if err := s.seedInitialPhase(ctx); err != nil {
		t.Fatalf("re-seed: %v", err)
	}
	if got := readPhase(t, s); got != domain.PhaseRegularSeason {
		t.Fatalf("re-seed clobbered history: phase = %q, want %q", got, domain.PhaseRegularSeason)
	}
}

// advance runs AppendPhaseTransition through a WriteTx.
func advance(t *testing.T, s *Store, to domain.Phase, note string) error {
	t.Helper()
	return s.WriteTx(context.Background(), func(w TxWriter) error {
		return w.AppendPhaseTransition(context.Background(), to, note)
	})
}

// TestAppendPhaseTransition: a valid target advances the current phase and records from→to;
// a no-op (to == current) and an unknown target are both rejected, leaving the phase unchanged.
func TestAppendPhaseTransition(t *testing.T) {
	s := newStore(t, &fakeSource{rosters: baseRosters(t)})

	if err := advance(t, s, domain.PhaseRegularSeason, "kickoff"); err != nil {
		t.Fatalf("advance to REGULAR_SEASON: %v", err)
	}
	if got := readPhase(t, s); got != domain.PhaseRegularSeason {
		t.Fatalf("phase = %q, want REGULAR_SEASON", got)
	}

	// No-op: advancing to the current phase is rejected.
	if err := advance(t, s, domain.PhaseRegularSeason, "again"); err == nil {
		t.Fatal("no-op transition succeeded, want rejection")
	} else if !strings.Contains(err.Error(), "no-op") {
		t.Fatalf("no-op error = %v, want a no-op rejection", err)
	}

	// Unknown target is rejected.
	if err := advance(t, s, domain.Phase("PRESEASON"), "bad"); err == nil {
		t.Fatal("unknown target phase succeeded, want rejection")
	}

	// The from→to pair was recorded on the advancing row.
	var from, to string
	if err := s.pools.Read().QueryRowContext(context.Background(),
		`SELECT from_phase, to_phase FROM season_phases WHERE note = 'kickoff'`).Scan(&from, &to); err != nil {
		t.Fatalf("read transition row: %v", err)
	}
	if from != string(domain.PhaseOffseason) || to != string(domain.PhaseRegularSeason) {
		t.Fatalf("transition row = %s→%s, want OFFSEASON→REGULAR_SEASON", from, to)
	}

	// Rollback is allowed (v1 commissioner correction).
	if err := advance(t, s, domain.PhaseOffseason, "rollback"); err != nil {
		t.Fatalf("rollback to OFFSEASON: %v", err)
	}
	if got := readPhase(t, s); got != domain.PhaseOffseason {
		t.Fatalf("after rollback, phase = %q, want OFFSEASON", got)
	}
}

// TestSeasonPhaseAppendOnlyTriggers proves the double-immutability: a raw UPDATE or DELETE
// that bypasses the (write-only) Go API still aborts at the DB. A gate that never sees a
// planted violation proves nothing — these are the plants.
func TestSeasonPhaseAppendOnlyTriggers(t *testing.T) {
	s := newStore(t, &fakeSource{rosters: baseRosters(t)})
	ctx := context.Background()

	if _, err := s.pools.Write().ExecContext(ctx,
		`UPDATE season_phases SET to_phase = ? WHERE league_id = ?`,
		string(domain.PhasePlayoffs), s.leagueID); err == nil {
		t.Fatal("raw UPDATE on season_phases succeeded — append-only trigger did not fire")
	} else if !strings.Contains(err.Error(), "append-only") {
		t.Fatalf("UPDATE error = %v, want an append-only abort", err)
	}

	if _, err := s.pools.Write().ExecContext(ctx,
		`DELETE FROM season_phases WHERE league_id = ?`, s.leagueID); err == nil {
		t.Fatal("raw DELETE on season_phases succeeded — append-only trigger did not fire")
	} else if !strings.Contains(err.Error(), "append-only") {
		t.Fatalf("DELETE error = %v, want an append-only abort", err)
	}

	// The seed row survives both blocked mutations.
	if got := readPhase(t, s); got != domain.PhaseOffseason {
		t.Fatalf("after blocked mutations, phase = %q, want %q", got, domain.PhaseOffseason)
	}
}
