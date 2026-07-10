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
