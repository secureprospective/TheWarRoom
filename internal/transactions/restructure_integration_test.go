package transactions_test

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/db"
	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/playerid"
	"github.com/secureprospective/TheWarRoom/internal/store/state"
	"github.com/secureprospective/TheWarRoom/internal/transactions"
)

const mil = domain.Money(100_000_000) // $1M in cents

// richSeed is a $-scale league that reaches the §11 restructure tiers (the shared
// integration seed uses cents-scale salaries too small to be eligible). Franchise 0001
// holds a $12M and a $6M player; 0002 holds a $2M player (below the $3M restructure floor).
type richSeed struct{ t *testing.T }

func (s richSeed) Rosters(context.Context) ([]domain.Roster, error) {
	id := func(raw string) playerid.PlayerID {
		p, err := playerid.New(raw)
		if err != nil {
			s.t.Fatalf("playerid.New(%q): %v", raw, err)
		}
		return p
	}
	return []domain.Roster{
		{FranchiseID: "0001", Players: []domain.PlayerRecord{
			{MFLID: id("0010"), Salary: 12 * mil, ContractYear: 2028,
				RosterStatus: domain.RosterActive, ContractStatus: domain.CStatusUFA},
			{MFLID: id("0011"), Salary: 6 * mil, ContractYear: 2029,
				RosterStatus: domain.RosterActive, ContractStatus: domain.CStatusUFA},
		}},
		{FranchiseID: "0002", Players: []domain.PlayerRecord{
			{MFLID: id("0020"), Salary: 2 * mil, ContractYear: 2028,
				RosterStatus: domain.RosterActive, ContractStatus: domain.CStatusUFA},
		}},
	}, nil
}

func richStore(t *testing.T) (*state.Store, *transactions.Coordinator) {
	t.Helper()
	pools, err := db.Open(context.Background(), filepath.Join(t.TempDir(), "rx.db"))
	if err != nil {
		t.Fatalf("db.Open: %v", err)
	}
	t.Cleanup(func() { _ = pools.Close() })
	s := state.New(pools, "14432", 2026)
	if err := s.Initialize(context.Background(), richSeed{t}); err != nil {
		t.Fatalf("Initialize: %v", err)
	}
	c, err := transactions.New(s.Writer())
	if err != nil {
		t.Fatalf("New coordinator: %v", err)
	}
	return s, c
}

// TestIntegration_RestructureLowersCapFlat proves the §11 happy path end to end: the move
// lowers the cap by EXACTLY the move amount (flat math — cap is the sum of contracts), the
// contract is flagged restructured, and a subsequent §8 cut then charges 50% dead cap on
// the reduced salary (the whole B7b→B7c dead-cap story wired together).
func TestIntegration_RestructureLowersCapFlat(t *testing.T) {
	s, c := richStore(t)

	before, _ := s.CapUsed("0001")
	if before != 18*mil { // $12M + $6M
		t.Fatalf("pre-restructure cap = %s, want $18M", before)
	}

	// $12M salary → tier max $3M. Move the full $3M.
	if _, err := c.Execute(context.Background(), transactions.Restructure{MFLID: "0010", Move: 3 * mil}); err != nil {
		t.Fatalf("Execute restructure: %v", err)
	}

	p, _ := s.Player("0010")
	if !p.IsRestructured {
		t.Fatal("player 0010 not flagged restructured")
	}
	if p.AdjustedSalary != 9*mil { // $12M − $3M
		t.Fatalf("adjusted salary = %s, want $9M", p.AdjustedSalary)
	}
	if got, _ := s.CapUsed("0001"); got != 15*mil { // $18M − $3M move
		t.Fatalf("post-restructure cap = %s, want $15M (dropped by exactly the move)", got)
	}

	// A §8 cut now charges 50% (restructured) × $9M effective × 2 remaining years = $9M.
	if _, err := c.Execute(context.Background(), transactions.Waiver{MFLID: "0010"}); err != nil {
		t.Fatalf("Execute waiver: %v", err)
	}
	// 0001 cap: $6M live (0011) + $9M dead (0010) = $15M — the $9M live salary became $9M dead.
	if got, _ := s.CapUsed("0001"); got != 15*mil {
		t.Fatalf("post-cut cap = %s, want $15M ($6M live + $9M @ 50%% dead cap)", got)
	}
}

// TestIntegration_RestructureRejectsAndRollsBack proves every §11 boundary is enforced and
// that a rejected restructure changes NOTHING (the Coordinator rolls back): over-max move,
// below-floor salary, one-per-team-per-year, and one-per-contract.
func TestIntegration_RestructureRejectsAndRollsBack(t *testing.T) {
	s, c := richStore(t)
	ctx := context.Background()

	// Over the $3M tier max for a $12M salary.
	if _, err := c.Execute(ctx, transactions.Restructure{MFLID: "0010", Move: 4 * mil}); err == nil {
		t.Fatal("over-max move ($4M) was accepted")
	}
	if p, _ := s.Player("0010"); p.IsRestructured || p.AdjustedSalary != 0 {
		t.Fatal("rejected over-max move still mutated the contract")
	}
	if got, _ := s.CapUsed("0001"); got != 18*mil {
		t.Fatalf("cap moved on a rejected restructure: %s, want $18M", got)
	}

	// $2M salary is below the $3M restructure floor.
	if _, err := c.Execute(ctx, transactions.Restructure{MFLID: "0020", Move: 1 * mil}); err == nil {
		t.Fatal("below-floor restructure ($2M salary) was accepted")
	}

	// A valid restructure, then a SECOND on the same franchise this season → rejected.
	if _, err := c.Execute(ctx, transactions.Restructure{MFLID: "0010", Move: 1 * mil}); err != nil {
		t.Fatalf("first valid restructure failed: %v", err)
	}
	if _, err := c.Execute(ctx, transactions.Restructure{MFLID: "0011", Move: 1 * mil}); err == nil {
		t.Fatal("second restructure for the same franchise in one season was accepted")
	}
	if p, _ := s.Player("0011"); p.IsRestructured {
		t.Fatal("the rejected second restructure still flagged 0011")
	}

	// One per contract: re-restructuring 0010 (already restructured) → rejected.
	if _, err := c.Execute(ctx, transactions.Restructure{MFLID: "0010", Move: 1 * mil}); err == nil {
		t.Fatal("a second restructure of the same contract was accepted")
	}
}
