package state

import (
	"context"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/db"
	"github.com/secureprospective/TheWarRoom/internal/domain"
)

// simulateRollover fakes a committed ROLLOVER_SEASON op at the DB level (the op itself lands in a
// later commit): it appends the PLAYOFFS→OFFSEASON transition carrying the NEW season and advances
// the roster/contract snapshot to it (D7). Lets the boot-order derivation (D1) be proven before the
// op exists.
func simulateRollover(t *testing.T, pools *db.Pools, league string, from, to int) {
	t.Helper()
	ctx := context.Background()
	now := time.Now().UTC().Format(time.RFC3339)
	// One tx so the fixture never lands in the split-brain state (phase log at N+1, rosters at N)
	// the production primitive is built to avoid (GLM L4).
	tx, err := pools.Write().BeginTx(ctx, nil)
	if err != nil {
		t.Fatalf("simulate rollover begin: %v", err)
	}
	defer func() { _ = tx.Rollback() }()
	if _, err := tx.ExecContext(ctx, `
INSERT INTO season_phases (league_id, season, from_phase, to_phase, note, meta, at)
VALUES (?, ?, ?, ?, 'test-rollover', '', ?)`,
		league, to, string(domain.PhasePlayoffs), string(domain.PhaseOffseason), now); err != nil {
		t.Fatalf("simulate rollover phase row: %v", err)
	}
	if _, err := tx.ExecContext(ctx,
		`UPDATE rosters SET season = ? WHERE league_id = ? AND season = ?`, to, league, from); err != nil {
		t.Fatalf("simulate rollover advance rosters: %v", err)
	}
	if _, err := tx.ExecContext(ctx,
		`UPDATE contracts SET season = ? WHERE league_id = ? AND season = ?`, to, league, from); err != nil {
		t.Fatalf("simulate rollover advance contracts: %v", err)
	}
	if err := tx.Commit(); err != nil {
		t.Fatalf("simulate rollover commit: %v", err)
	}
}

// TestSeasonDerivedFromPhaseLogOnReboot proves the D1/D7 boot-order fix: after a rollover has moved
// the phase log (and the roster snapshot) to a new season, a fresh Store constructed with the
// ORIGINAL config season must DERIVE the current season from the phase log — it must NOT re-seed at
// the stale config year, and its cap must follow the derived season. This is the make-or-break
// rollover seam (D6): a stale s.season silently computes the cap against the wrong year.
func TestSeasonDerivedFromPhaseLogOnReboot(t *testing.T) {
	dir := t.TempDir()
	pools, err := db.Open(context.Background(), filepath.Join(dir, "rollover.db"))
	if err != nil {
		t.Fatalf("db.Open: %v", err)
	}
	t.Cleanup(func() { _ = pools.Close() })
	ctx := context.Background()

	// Seed at the config season (2026): franchise 0001 = player 0001 (exp 2028) + 0002 (exp 2027).
	s1 := New(pools, testLeague, testSeason, nil)
	if err := s1.Initialize(ctx, &fakeSource{rosters: baseRosters(t)}); err != nil {
		t.Fatalf("seed Initialize: %v", err)
	}
	if fs, ok := s1.FranchiseState("0001"); !ok || fs.CapUsed != 15*capUnit {
		t.Fatalf("seed CapUsed(0001) = %v ok=%v, want %v", fs.CapUsed, ok, 15*capUnit)
	}

	// Simulate a rollover to 2028: player 0002 (exp 2027) then has no PAID cell, so 0001's cap
	// must drop to player 0001's 2028 cell alone (10 units) IF the derived season is honored.
	rolled := testSeason + 2
	simulateRollover(t, pools, testLeague, testSeason, rolled)

	// Reboot with the UNCHANGED config season — must derive 2028 from the phase log.
	s2 := New(pools, testLeague, testSeason, nil)
	if err := s2.Initialize(ctx, &fakeSource{rosters: baseRosters(t)}); err != nil {
		t.Fatalf("reboot Initialize: %v", err)
	}
	if s2.season != rolled {
		t.Fatalf("derived season = %d, want %d (from phase log)", s2.season, rolled)
	}
	if got := readPhase(t, s2); got != domain.PhaseOffseason {
		t.Fatalf("post-rollover phase = %q, want OFFSEASON", got)
	}
	if fs, ok := s2.FranchiseState("0001"); !ok || fs.CapUsed != 10*capUnit {
		t.Fatalf("post-rollover CapUsed(0001) = %v ok=%v, want %v (cap follows derived season)", fs.CapUsed, ok, 10*capUnit)
	}
	if got := s2.Franchises(); len(got) != 2 {
		t.Fatalf("Franchises after reboot = %v, want the 2 seeded (must NOT re-seed)", got)
	}
}

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
