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

// seedSource yields a fixed two-franchise league for the integration wiring test.
type seedSource struct{ t *testing.T }

func (s seedSource) Rosters(context.Context) ([]domain.Roster, error) {
	id := func(raw string) playerid.PlayerID {
		p, err := playerid.New(raw)
		if err != nil {
			s.t.Fatalf("playerid.New(%q): %v", raw, err)
		}
		return p
	}
	return []domain.Roster{
		{FranchiseID: "0001", Players: []domain.PlayerRecord{
			{MFLID: id("0001"), Salary: 10, ContractYear: 2028,
				RosterStatus: domain.RosterActive, ContractStatus: domain.CStatusUFA},
		}},
		{FranchiseID: "0002", Players: []domain.PlayerRecord{
			{MFLID: id("0003"), Salary: 7, ContractYear: 2029,
				RosterStatus: domain.RosterActive, ContractStatus: domain.CStatusFT1},
		}},
	}, nil
}

func realStore(t *testing.T) *state.Store {
	t.Helper()
	pools, err := db.Open(context.Background(), filepath.Join(t.TempDir(), "txn.db"))
	if err != nil {
		t.Fatalf("db.Open: %v", err)
	}
	t.Cleanup(func() { _ = pools.Close() })
	s := state.New(pools, "14432", 2026)
	if err := s.Initialize(context.Background(), seedSource{t}); err != nil {
		t.Fatalf("Initialize: %v", err)
	}
	return s
}

// TestIntegration_TradePersists wires the real Coordinator to the real state store and
// executes a two-leg swap. It proves the reshaped Writer interface, the spanning tx, and
// the handler dispatch all connect end to end: after Execute, both players have swapped
// franchises and the derived cap reflects it.
func TestIntegration_TradePersists(t *testing.T) {
	s := realStore(t)
	c, err := transactions.New(s.Writer())
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	rec, err := c.Execute(context.Background(), transactions.Trade{Moves: []transactions.PlayerMove{
		{MFLID: "0001", ToFranchiseID: "0002"},
		{MFLID: "0003", ToFranchiseID: "0001"},
	}})
	if err != nil {
		t.Fatalf("Execute trade: %v", err)
	}
	if rec.PlayersAffected != 2 {
		t.Fatalf("receipt affected = %d, want 2", rec.PlayersAffected)
	}

	p1, _ := s.Player("0001")
	p3, _ := s.Player("0003")
	if p1.FranchiseID != "0002" || p3.FranchiseID != "0001" {
		t.Fatalf("trade did not persist: 0001@%s 0003@%s", p1.FranchiseID, p3.FranchiseID)
	}
	if used, _ := s.CapUsed("0001"); used != 7 {
		t.Fatalf("cap after trade: 0001=%v, want 7", used)
	}
}

// TestIntegration_BadLegRollsBackWholeTrade plants an unknown player on the second leg.
// The whole trade must roll back: the first, valid leg must NOT persist. This is the
// atomicity guarantee, proven through the real Coordinator + store together.
func TestIntegration_BadLegRollsBackWholeTrade(t *testing.T) {
	s := realStore(t)
	c, err := transactions.New(s.Writer())
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	_, err = c.Execute(context.Background(), transactions.Trade{Moves: []transactions.PlayerMove{
		{MFLID: "0001", ToFranchiseID: "0002"}, // valid
		{MFLID: "9999", ToFranchiseID: "0001"}, // unknown → fails the transaction
	}})
	if err == nil {
		t.Fatal("Execute succeeded despite an unknown-player leg")
	}

	// The first leg must be invisible — no half-applied trade.
	if p1, _ := s.Player("0001"); p1.FranchiseID != "0001" {
		t.Fatalf("first leg was not rolled back: 0001@%s (want 0001)", p1.FranchiseID)
	}
}
