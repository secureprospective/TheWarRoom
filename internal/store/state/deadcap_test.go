package state

import (
	"context"
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/domain"
)

// addDeadCap is a test helper that appends one charge through the real WriteTx path.
func addDeadCap(t *testing.T, s *Store, e DeadCapEntry) {
	t.Helper()
	if err := s.WriteTx(context.Background(), func(w TxWriter) error {
		return w.AddDeadCap(context.Background(), e)
	}); err != nil {
		t.Fatalf("AddDeadCap(%+v): %v", e, err)
	}
}

// TestAddDeadCap_MovesCapUsed proves a ledger charge for the current season is folded
// into the franchise's derived CapUsed after the WriteTx reload.
func TestAddDeadCap_MovesCapUsed(t *testing.T) {
	s := newStore(t, &fakeSource{rosters: baseRosters(t)})

	before, ok := s.CapUsed("0001")
	if !ok {
		t.Fatal("franchise 0001 missing")
	}

	addDeadCap(t, s, DeadCapEntry{
		FranchiseID: "0001", MFLID: "0001", LeagueYear: testSeason,
		DeadCap: domain.Money(350_000_000), Reason: "waiver-cut §8",
	})

	after, ok := s.CapUsed("0001")
	if !ok {
		t.Fatal("franchise 0001 missing after dead cap")
	}
	if after-before != domain.Money(350_000_000) {
		t.Errorf("CapUsed delta = %d, want 350000000", after-before)
	}
}

// TestAddDeadCap_OnlyCurrentSeasonCounts proves CapUsed is a current-season figure: a
// charge keyed to a DIFFERENT absolute league year does not move this season's cap.
func TestAddDeadCap_OnlyCurrentSeasonCounts(t *testing.T) {
	s := newStore(t, &fakeSource{rosters: baseRosters(t)})

	before, _ := s.CapUsed("0001")
	addDeadCap(t, s, DeadCapEntry{
		FranchiseID: "0001", MFLID: "0001", LeagueYear: testSeason + 1,
		DeadCap: domain.Money(200_000_000), Reason: "future charge",
	})
	after, _ := s.CapUsed("0001")
	if after != before {
		t.Errorf("next-year dead cap moved this season's CapUsed: %d → %d", before, after)
	}
}

// TestAddDeadCap_LedgerIsAppendOnly proves the DB triggers reject a raw UPDATE and a raw
// DELETE even though they bypass the Go API (the B6 double-immutability idiom).
func TestAddDeadCap_LedgerIsAppendOnly(t *testing.T) {
	ctx := context.Background()
	s := newStore(t, &fakeSource{rosters: baseRosters(t)})
	addDeadCap(t, s, DeadCapEntry{
		FranchiseID: "0001", MFLID: "0001", LeagueYear: testSeason,
		DeadCap: domain.Money(100_000_000), Reason: "waiver-cut §8",
	})

	if _, err := s.pools.Write().ExecContext(ctx,
		`UPDATE dead_cap_ledger SET dead_cap_cents = 0`); err == nil {
		t.Error("raw UPDATE on dead_cap_ledger succeeded — append-only trigger missing")
	}
	if _, err := s.pools.Write().ExecContext(ctx,
		`DELETE FROM dead_cap_ledger`); err == nil {
		t.Error("raw DELETE on dead_cap_ledger succeeded — append-only trigger missing")
	}
}

// TestAddDeadCap_DuplicateRejected proves the UNIQUE key stops a second charge for the
// same player+franchise+year (a double-cut), and the failed tx leaves cap unchanged.
func TestAddDeadCap_DuplicateRejected(t *testing.T) {
	s := newStore(t, &fakeSource{rosters: baseRosters(t)})
	e := DeadCapEntry{
		FranchiseID: "0001", MFLID: "0001", LeagueYear: testSeason,
		DeadCap: domain.Money(100_000_000), Reason: "waiver-cut §8",
	}
	addDeadCap(t, s, e)
	mid, _ := s.CapUsed("0001")

	if err := s.WriteTx(context.Background(), func(w TxWriter) error {
		return w.AddDeadCap(context.Background(), e)
	}); err == nil {
		t.Fatal("duplicate dead-cap charge accepted")
	}
	after, _ := s.CapUsed("0001")
	if after != mid {
		t.Errorf("rejected duplicate still moved CapUsed: %d → %d", mid, after)
	}
}

// TestAddDeadCap_Validation proves each field guard fails loud before any write.
func TestAddDeadCap_Validation(t *testing.T) {
	for _, tc := range []struct {
		name string
		e    DeadCapEntry
	}{
		{"negative", DeadCapEntry{FranchiseID: "0001", MFLID: "0001", LeagueYear: testSeason, DeadCap: -1, Reason: "x"}},
		{"no franchise", DeadCapEntry{MFLID: "0001", LeagueYear: testSeason, DeadCap: 1, Reason: "x"}},
		{"no player", DeadCapEntry{FranchiseID: "0001", LeagueYear: testSeason, DeadCap: 1, Reason: "x"}},
		{"no reason", DeadCapEntry{FranchiseID: "0001", MFLID: "0001", LeagueYear: testSeason, DeadCap: 1}},
		{"no year", DeadCapEntry{FranchiseID: "0001", MFLID: "0001", DeadCap: 1, Reason: "x"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s := newStore(t, &fakeSource{rosters: baseRosters(t)})
			if err := s.WriteTx(context.Background(), func(w TxWriter) error {
				return w.AddDeadCap(context.Background(), tc.e)
			}); err == nil {
				t.Fatal("invalid dead-cap entry accepted")
			}
		})
	}
}
