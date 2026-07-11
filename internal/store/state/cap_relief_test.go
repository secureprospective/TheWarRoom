package state

import (
	"context"
	"strings"
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/domain"
)

// addRelief runs AddCapRelief through a WriteTx (the surface the §13 handler uses).
func addRelief(t *testing.T, s *Store, e CapReliefEntry) error {
	t.Helper()
	return s.WriteTx(context.Background(), func(w TxWriter) error {
		return w.AddCapRelief(context.Background(), e)
	})
}

// TestAddCapRelief pins the write path and its shape guards: a valid credit lands and is summed
// into loadCapRelief; a non-positive amount, an empty franchise, an empty reason, and a
// non-absolute year are each rejected before any row is written.
func TestAddCapRelief(t *testing.T) {
	s := newStore(t, &fakeSource{rosters: baseRosters(t)})
	ctx := context.Background()

	if err := addRelief(t, s, CapReliefEntry{FranchiseID: "0001", LeagueYear: s.season, Amount: 3 * domain.Money(100_000_000), Reason: "career-ending injury"}); err != nil {
		t.Fatalf("valid relief: %v", err)
	}
	// A second credit accumulates for the same franchise.
	if err := addRelief(t, s, CapReliefEntry{FranchiseID: "0001", LeagueYear: s.season, Amount: 2 * domain.Money(100_000_000), Reason: "recurring injury"}); err != nil {
		t.Fatalf("second relief: %v", err)
	}
	cr, err := s.loadCapRelief(ctx)
	if err != nil {
		t.Fatalf("loadCapRelief: %v", err)
	}
	if got := cr["0001"]; got != 5*domain.Money(100_000_000) {
		t.Fatalf("summed relief = %v, want $5M", got)
	}

	for _, tc := range []struct {
		name string
		e    CapReliefEntry
	}{
		{"zero amount", CapReliefEntry{FranchiseID: "0001", LeagueYear: s.season, Amount: 0, Reason: "x"}},
		{"negative amount", CapReliefEntry{FranchiseID: "0001", LeagueYear: s.season, Amount: -1, Reason: "x"}},
		{"empty franchise", CapReliefEntry{FranchiseID: "", LeagueYear: s.season, Amount: 1, Reason: "x"}},
		{"empty reason", CapReliefEntry{FranchiseID: "0001", LeagueYear: s.season, Amount: 1, Reason: ""}},
		{"non-absolute year", CapReliefEntry{FranchiseID: "0001", LeagueYear: 0, Amount: 1, Reason: "x"}},
	} {
		if err := addRelief(t, s, tc.e); err == nil {
			t.Fatalf("%s: AddCapRelief accepted a bad-shape entry, want rejection", tc.name)
		}
	}
}

// TestCapReliefAppendOnlyTriggers proves the double-immutability: a raw UPDATE or DELETE that
// bypasses the (write-only) Go API still aborts at the DB. A gate that never sees a planted
// violation proves nothing — these are the plants.
func TestCapReliefAppendOnlyTriggers(t *testing.T) {
	s := newStore(t, &fakeSource{rosters: baseRosters(t)})
	ctx := context.Background()

	if err := addRelief(t, s, CapReliefEntry{FranchiseID: "0001", LeagueYear: s.season, Amount: domain.Money(100_000_000), Reason: "seed row"}); err != nil {
		t.Fatalf("seed relief: %v", err)
	}

	if _, err := s.pools.Write().ExecContext(ctx,
		`UPDATE cap_relief_ledger SET relief_cents = 1 WHERE league_id = ?`, s.leagueID); err == nil {
		t.Fatal("raw UPDATE on cap_relief_ledger succeeded — append-only trigger did not fire")
	} else if !strings.Contains(err.Error(), "append-only") {
		t.Fatalf("UPDATE error = %v, want an append-only abort", err)
	}

	if _, err := s.pools.Write().ExecContext(ctx,
		`DELETE FROM cap_relief_ledger WHERE league_id = ?`, s.leagueID); err == nil {
		t.Fatal("raw DELETE on cap_relief_ledger succeeded — append-only trigger did not fire")
	} else if !strings.Contains(err.Error(), "append-only") {
		t.Fatalf("DELETE error = %v, want an append-only abort", err)
	}

	// The seed row survives both blocked mutations.
	cr, err := s.loadCapRelief(ctx)
	if err != nil {
		t.Fatalf("loadCapRelief after blocked mutations: %v", err)
	}
	if cr["0001"] != domain.Money(100_000_000) {
		t.Fatalf("seed relief = %v after blocked mutations, want $1M intact", cr["0001"])
	}
}

// TestCapReliefCheckRejectsNonPositive proves the DB-level CHECK(relief_cents > 0) is a real
// backstop under the Go guard: a raw INSERT of a non-positive credit is aborted by SQLite.
func TestCapReliefCheckRejectsNonPositive(t *testing.T) {
	s := newStore(t, &fakeSource{rosters: baseRosters(t)})
	ctx := context.Background()
	if _, err := s.pools.Write().ExecContext(ctx, `
INSERT INTO cap_relief_ledger (league_id, franchise_id, league_year, relief_cents, reason, created_at)
VALUES (?, '0001', ?, 0, 'raw zero', '2026-01-01T00:00:00Z')`, s.leagueID, s.season); err == nil {
		t.Fatal("raw INSERT of a $0 relief succeeded — CHECK(relief_cents > 0) did not fire")
	}
}
