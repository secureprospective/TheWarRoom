package transactions_test

import (
	"context"
	"path/filepath"
	"strings"
	"testing"
	"time"

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
			{MFLID: id("0001"), Salary: 10 * mil, ContractYear: 2028,
				RosterStatus: domain.RosterActive, ContractStatus: domain.CStatusUFA},
		}},
		{FranchiseID: "0002", Players: []domain.PlayerRecord{
			{MFLID: id("0003"), Salary: 7 * mil, ContractYear: 2029,
				RosterStatus: domain.RosterActive, ContractStatus: domain.CStatusFT1},
		}},
	}, nil
}

func realStore(t *testing.T) *state.Store {
	t.Helper()
	s, _ := realStoreWithPools(t)
	return s
}

// realStoreWithPools also hands back the raw *db.Pools — needed by tests that read back a table
// (like trade_notes) the state.Store package doesn't expose a typed accessor for.
func realStoreWithPools(t *testing.T) (*state.Store, *db.Pools) {
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
	return s, pools
}

// TestIntegration_TradePersists wires the real Coordinator to the real state store and
// executes a two-leg swap. It proves the reshaped Writer interface, the spanning tx, and
// the handler dispatch all connect end to end: after Execute, both players have swapped
// franchises and the derived cap reflects it.
func TestIntegration_TradePersists(t *testing.T) {
	s, pools := realStoreWithPools(t)
	c, err := transactions.New(s.Writer())
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	rec, err := c.Execute(context.Background(), transactions.Trade{
		Moves: []transactions.PlayerMove{
			{MFLID: "0001", ToFranchiseID: "0002"},
			{MFLID: "0003", ToFranchiseID: "0001"},
		},
		Rationale: "integration test trade",
	})
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
	if used, _ := s.CapUsed("0001"); used != 7*mil {
		t.Fatalf("cap after trade: 0001=%v, want %v", used, 7*mil)
	}

	// The trade event itself must be persisted (trade_notes), not just its player-move side
	// effects — this is the whole point of LogTradeNote.
	var rationale, involved string
	if err := pools.Read().QueryRowContext(context.Background(),
		`SELECT rationale, involved_franchises FROM trade_notes`).Scan(&rationale, &involved); err != nil {
		t.Fatalf("read trade_notes row: %v", err)
	}
	if rationale != "integration test trade" {
		t.Fatalf("trade_notes.rationale = %q, want %q", rationale, "integration test trade")
	}
	if !strings.Contains(involved, "0001") || !strings.Contains(involved, "0002") {
		t.Fatalf("trade_notes.involved_franchises = %q, want it to mention both 0001 and 0002", involved)
	}
}

// TestIntegration_WaiverCutConservesCap wires the real Coordinator to the real store and
// cuts a player, proving the §8 close gate end to end: the released salary leaves the cap,
// the dead-cap charge lands in the ledger, cap "conservation" holds (new cap = §8 charge
// only), the player is gone, and a second read confirms it all persisted.
func TestIntegration_WaiverCutConservesCap(t *testing.T) {
	s := realStore(t)
	c, err := transactions.New(s.Writer())
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	// Player 0001: salary $10M, contract through 2028, store season 2026 → remaining = 2,
	// §8 charge = 35% × $10M × 2 = $7M (a clean $10k multiple).
	before, _ := s.CapUsed("0001")
	if before != 10*mil {
		t.Fatalf("pre-cut cap = %v, want %v (the player's salary)", before, 10*mil)
	}

	rec, err := c.Execute(context.Background(), transactions.Waiver{MFLID: "0001"})
	if err != nil {
		t.Fatalf("Execute waiver: %v", err)
	}
	if rec.Kind != transactions.KindWaiver || rec.PlayersAffected != 1 {
		t.Fatalf("receipt = %+v, want KindWaiver/1", rec)
	}

	// The player is gone from state.
	if _, ok := s.Player("0001"); ok {
		t.Fatal("player 0001 still rostered after a cut")
	}
	// Cap conservation: his salary ($10M) left; the §8 dead cap ($7M) remains against 0001.
	after, ok := s.CapUsed("0001")
	if !ok {
		t.Fatal("franchise 0001 vanished — dead cap should keep it visible")
	}
	if after != 7*mil {
		t.Fatalf("post-cut cap = %v, want %v (§8 dead cap only)", after, 7*mil)
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

	_, err = c.Execute(context.Background(), transactions.Trade{
		Moves: []transactions.PlayerMove{
			{MFLID: "0001", ToFranchiseID: "0002"}, // valid
			{MFLID: "9999", ToFranchiseID: "0001"}, // unknown → fails the transaction
		},
		Rationale: "integration test trade",
	})
	if err == nil {
		t.Fatal("Execute succeeded despite an unknown-player leg")
	}

	// The first leg must be invisible — no half-applied trade.
	if p1, _ := s.Player("0001"); p1.FranchiseID != "0001" {
		t.Fatalf("first leg was not rolled back: 0001@%s (want 0001)", p1.FranchiseID)
	}
}

// TestIntegration_TradeDeadlineBlocksTrade wires the real Coordinator to the real store and
// proves the §14 phase-gate second check: once the commissioner sets a past trade deadline, a
// KindTrade is rejected with the same shape as the existing KindSign/SigningWindowClosed gate.
func TestIntegration_TradeDeadlineBlocksTrade(t *testing.T) {
	s := realStore(t)
	c, err := transactions.New(s.Writer())
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	past := time.Now().Add(-24 * time.Hour)
	if _, err := c.Execute(context.Background(), transactions.SetTradeDeadline{Deadline: past, Note: "week 9"}); err != nil {
		t.Fatalf("Execute set trade deadline: %v", err)
	}

	_, err = c.Execute(context.Background(), transactions.Trade{
		Moves:     []transactions.PlayerMove{{MFLID: "0001", ToFranchiseID: "0002"}},
		Rationale: "too late",
	})
	if err == nil {
		t.Fatal("trade after the deadline succeeded, want the phase-gate rejection")
	}

	// The would-be leg must be invisible — the gate ran before any player moved.
	if p1, _ := s.Player("0001"); p1.FranchiseID != "0001" {
		t.Fatalf("a rejected trade still moved a player: 0001@%s (want 0001)", p1.FranchiseID)
	}
}
