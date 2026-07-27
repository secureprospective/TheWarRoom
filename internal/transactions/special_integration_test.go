package transactions_test

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/db"
	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/playerid"
	statepkg "github.com/secureprospective/TheWarRoom/internal/store/state"
	"github.com/secureprospective/TheWarRoom/internal/transactions"
)

// specialSeed puts each §13 target on its OWN franchise so a franchise's CapUsed after the op is
// exactly the charge (the player's salary leaves, only his dead cap remains) — a clean assertion.
// Seed season is 2026; the ledger flat-fills PAID cells from 2026 through the expiration year.
//
//	0001 $6M exp 2028 → 2 remaining → retirement 30% × (6M+6M) = $3.6M
//	0002 $10M exp 2027 → death (Gaines Adams) → removed, $0 dead cap
//	0003 $10M exp 2026 → cap-relief fixture (CapUsed = $10M current cell, no remaining years)
//	0004 $8M exp 2027 → 1 remaining → retirement 30% × 8M = $2.4M
type specialSeed struct{ t *testing.T }

func (s specialSeed) Rosters(context.Context) ([]domain.Roster, error) {
	id := func(raw string) playerid.PlayerID {
		p, err := playerid.New(raw)
		if err != nil {
			s.t.Fatalf("playerid.New(%q): %v", raw, err)
		}
		return p
	}
	mk := func(raw string, salary domain.Money, exp int) domain.PlayerRecord {
		return domain.PlayerRecord{MFLID: id(raw), Salary: salary, ContractYear: exp,
			RosterStatus: domain.RosterActive, ContractStatus: domain.CStatusUFA}
	}
	one := func(fid, raw string, salary domain.Money, exp int) domain.Roster {
		return domain.Roster{FranchiseID: fid, Players: []domain.PlayerRecord{mk(raw, salary, exp)}}
	}
	return []domain.Roster{
		one("0001", "0001", 6*mil, 2028),
		one("0002", "0002", 10*mil, 2027),
		one("0003", "0003", 10*mil, 2026),
		one("0004", "0004", 8*mil, 2027),
	}, nil
}

func specialStore(t *testing.T) (*statepkg.Store, *transactions.Coordinator) {
	t.Helper()
	pools, err := db.Open(context.Background(), filepath.Join(t.TempDir(), "special.db"))
	if err != nil {
		t.Fatalf("db.Open: %v", err)
	}
	t.Cleanup(func() { _ = pools.Close() })
	s := statepkg.New(pools, "14432", 2026, nil)
	if err := s.Initialize(context.Background(), specialSeed{t}); err != nil {
		t.Fatalf("Initialize: %v", err)
	}
	c, err := transactions.New(s.Writer(), nil)
	if err != nil {
		t.Fatalf("New coordinator: %v", err)
	}
	return s, c
}

// TestIntegration_Retirement proves §13 retirement end to end: the player is released and the
// franchise carries a dead-cap charge of 30% of his remaining contract (years after 2026).
func TestIntegration_Retirement(t *testing.T) {
	s, c := specialStore(t)
	ctx := context.Background()

	for _, tc := range []struct {
		fr, player string
		want       domain.Money
	}{
		{"0001", "0001", 36 * mil / 10}, // 30% × (6M+6M) = $3.6M
		{"0004", "0004", 24 * mil / 10}, // 30% × 8M (one remaining year, 2027) = $2.4M
	} {
		rec, err := c.Execute(ctx, transactions.Retirement{MFLID: tc.player})
		if err != nil {
			t.Fatalf("retire %s: %v", tc.player, err)
		}
		if rec.Kind != transactions.KindRetirement || rec.PlayersAffected != 1 {
			t.Fatalf("receipt = %+v, want KindRetirement/1", rec)
		}
		if _, ok := s.Reader().Player(tc.player); ok {
			t.Fatalf("player %s still rostered after retirement", tc.player)
		}
		got, ok := s.Reader().CapUsed(tc.fr)
		if !ok || got != tc.want {
			t.Fatalf("franchise %s dead cap = %v (ok=%v), want %v", tc.fr, got, ok, tc.want)
		}
	}
}

// TestIntegration_Death proves the §13 Gaines Adams Rule: the player is removed with ZERO cap
// penalty — his salary leaves and NO dead cap lands, so the franchise's CapUsed is $0.
func TestIntegration_Death(t *testing.T) {
	s, c := specialStore(t)
	ctx := context.Background()

	rec, err := c.Execute(ctx, transactions.Death{MFLID: "0002"})
	if err != nil {
		t.Fatalf("death 0002: %v", err)
	}
	if rec.Kind != transactions.KindDeath || rec.PlayersAffected != 1 {
		t.Fatalf("receipt = %+v, want KindDeath/1", rec)
	}
	if _, ok := s.Reader().Player("0002"); ok {
		t.Fatal("player 0002 still rostered after death")
	}
	got, ok := s.Reader().CapUsed("0002")
	if !ok || got != 0 {
		t.Fatalf("franchise 0002 CapUsed = %v (ok=%v), want $0 (no penalty)", got, ok)
	}
}

// TestIntegration_CapRelief proves §13 Cap Relief: a commissioner credit reduces the franchise's
// CapUsed by the relief amount, and multiple reliefs accumulate.
func TestIntegration_CapRelief(t *testing.T) {
	s, c := specialStore(t)
	ctx := context.Background()

	// Franchise 0003 starts at $10M cap (its one player's current-season cell).
	if got, _ := s.Reader().CapUsed("0003"); got != 10*mil {
		t.Fatalf("pre-relief CapUsed = %v, want $10M", got)
	}
	rec, err := c.Execute(ctx, transactions.CapRelief{FranchiseID: "0003", Amount: 3 * mil, Reason: "career-ending injury"})
	if err != nil {
		t.Fatalf("cap relief: %v", err)
	}
	if rec.Kind != transactions.KindCapRelief || rec.PlayersAffected != 0 {
		t.Fatalf("receipt = %+v, want KindCapRelief/0", rec)
	}
	if got, _ := s.Reader().CapUsed("0003"); got != 7*mil {
		t.Fatalf("post-relief CapUsed = %v, want $7M", got)
	}
	// A second relief accumulates: $3M + $2M = $5M off → CapUsed $5M.
	if _, err := c.Execute(ctx, transactions.CapRelief{FranchiseID: "0003", Amount: 2 * mil, Reason: "recurring injury"}); err != nil {
		t.Fatalf("second cap relief: %v", err)
	}
	if got, _ := s.Reader().CapUsed("0003"); got != 5*mil {
		t.Fatalf("post-second-relief CapUsed = %v, want $5M", got)
	}
}

// TestIntegration_CapReliefFloorsAtZero proves an over-generous relief (more than the current
// hit) floors CapUsed at $0 rather than going negative — a franchise can't use negative cap.
func TestIntegration_CapReliefFloorsAtZero(t *testing.T) {
	s, c := specialStore(t)
	ctx := context.Background()

	if _, err := c.Execute(ctx, transactions.CapRelief{FranchiseID: "0003", Amount: 25 * mil, Reason: "behavioral suspension"}); err != nil {
		t.Fatalf("cap relief: %v", err)
	}
	if got, ok := s.Reader().CapUsed("0003"); !ok || got != 0 {
		t.Fatalf("over-relieved CapUsed = %v (ok=%v), want $0 floor", got, ok)
	}
}

// TestIntegration_CapReliefSnapsToGrid proves the universal FLAT $10k rounding invariant reaches
// the commissioner's discretionary relief figure (GLM §13 review): an OFF-grid relief of $3,006,000
// snaps to $3,010,000, so franchise 0003's CapUsed lands on the $10k grid at $6.99M. A non-snapping
// impl would subtract the raw $3,006,000 → an off-grid $6,994,000, which this assertion rejects.
func TestIntegration_CapReliefSnapsToGrid(t *testing.T) {
	s, c := specialStore(t)
	ctx := context.Background()

	offGrid := 3*mil + 600_000 // $3,006,000 — not a $10k multiple
	if _, err := c.Execute(ctx, transactions.CapRelief{FranchiseID: "0003", Amount: offGrid, Reason: "career-ending injury"}); err != nil {
		t.Fatalf("cap relief: %v", err)
	}
	want := 699 * mil / 100 // $10M − snapped $3.01M = $6.99M (on-grid)
	if got, ok := s.Reader().CapUsed("0003"); !ok || got != want {
		t.Fatalf("post-relief CapUsed = %v (ok=%v), want %v (off-grid relief snapped to $10k)", got, ok, want)
	}
}

// TestIntegration_CapReliefRejectsBadShape proves the request-level shape gate: a non-positive
// amount, an empty franchise, and an empty reason are all rejected before any write.
func TestIntegration_CapReliefRejectsBadShape(t *testing.T) {
	_, c := specialStore(t)
	ctx := context.Background()
	for _, tc := range []struct {
		name string
		req  transactions.CapRelief
	}{
		{"zero amount", transactions.CapRelief{FranchiseID: "0003", Amount: 0, Reason: "x"}},
		{"negative amount", transactions.CapRelief{FranchiseID: "0003", Amount: -mil, Reason: "x"}},
		{"empty franchise", transactions.CapRelief{FranchiseID: "", Amount: mil, Reason: "x"}},
		{"empty reason", transactions.CapRelief{FranchiseID: "0003", Amount: mil, Reason: ""}},
	} {
		if _, err := c.Execute(ctx, tc.req); err == nil {
			t.Fatalf("%s: cap relief accepted a bad-shape request, want rejection", tc.name)
		}
	}
}

// TestIntegration_SpecialSituationsUnknownPlayer proves retirement and death fail loud on a
// player who is on no roster (nothing released, no charge written).
func TestIntegration_SpecialSituationsUnknownPlayer(t *testing.T) {
	_, c := specialStore(t)
	ctx := context.Background()
	if _, err := c.Execute(ctx, transactions.Retirement{MFLID: "9999"}); err == nil {
		t.Fatal("retirement of an unrostered player succeeded, want fail-loud")
	}
	if _, err := c.Execute(ctx, transactions.Death{MFLID: "9999"}); err == nil {
		t.Fatal("death of an unrostered player succeeded, want fail-loud")
	}
}
