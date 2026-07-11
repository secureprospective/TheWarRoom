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

// buySeed puts each buyout target on its OWN franchise so that after a buyout the franchise's
// CapUsed equals exactly the §12 dead-cap charge (the player's salary leaves, only his dead cap
// remains) — a clean assertion of the rate math. Seed season is 2026; the ledger flat-fills PAID
// cells from 2026 through the expiration year, so expiration sets "remaining years" (years after
// 2026). All salaries $6M so the average remaining salary is $6M and the charge is rate × $6M.
//
//	0001 exp 2028 → remaining 2 → 60% → $3.6M
//	0002 exp 2029 → remaining 3 → 75% → $4.5M
//	0003 exp 2030 → remaining 4 → 90% → $5.4M
//	0004 exp 2027 → remaining 1 → outside §12 (reject → §13)
//	0005 exp 2031 → remaining 5 → outside §12 (reject → §13)
//	0009 holds 0091/0092/0093 (all exp 2028, remaining 2) → the per-season-limit fixture.
type buySeed struct{ t *testing.T }

func (s buySeed) Rosters(context.Context) ([]domain.Roster, error) {
	id := func(raw string) playerid.PlayerID {
		p, err := playerid.New(raw)
		if err != nil {
			s.t.Fatalf("playerid.New(%q): %v", raw, err)
		}
		return p
	}
	mk := func(raw string, exp int) domain.PlayerRecord {
		return domain.PlayerRecord{MFLID: id(raw), Salary: 6 * mil, ContractYear: exp,
			RosterStatus: domain.RosterActive, ContractStatus: domain.CStatusUFA}
	}
	one := func(fid, raw string, exp int) domain.Roster {
		return domain.Roster{FranchiseID: fid, Players: []domain.PlayerRecord{mk(raw, exp)}}
	}
	return []domain.Roster{
		one("0001", "0001", 2028),
		one("0002", "0002", 2029),
		one("0003", "0003", 2030),
		one("0004", "0004", 2027),
		one("0005", "0005", 2031),
		{FranchiseID: "0009", Players: []domain.PlayerRecord{mk("0091", 2028), mk("0092", 2028), mk("0093", 2028)}},
	}, nil
}

func buyStore(t *testing.T) (*statepkg.Store, *transactions.Coordinator) {
	t.Helper()
	pools, err := db.Open(context.Background(), filepath.Join(t.TempDir(), "buy.db"))
	if err != nil {
		t.Fatalf("db.Open: %v", err)
	}
	t.Cleanup(func() { _ = pools.Close() })
	s := statepkg.New(pools, "14432", 2026)
	if err := s.Initialize(context.Background(), buySeed{t}); err != nil {
		t.Fatalf("Initialize: %v", err)
	}
	c, err := transactions.New(s.Writer())
	if err != nil {
		t.Fatalf("New coordinator: %v", err)
	}
	return s, c
}

// TestIntegration_BuyoutHappyPathAndRates proves the §12 rate table end to end: the whole charge
// lands as dead cap in the current season, the player is released, and 2/3/4 remaining years map
// to 60/75/90% of the $6M average.
func TestIntegration_BuyoutHappyPathAndRates(t *testing.T) {
	s, c := buyStore(t)
	ctx := context.Background()

	for _, tc := range []struct {
		fr, player string
		want       domain.Money
	}{
		{"0001", "0001", 36 * mil / 10}, // 60% × $6M = $3.6M
		{"0002", "0002", 45 * mil / 10}, // 75% × $6M = $4.5M
		{"0003", "0003", 54 * mil / 10}, // 90% × $6M = $5.4M
	} {
		rec, err := c.Execute(ctx, transactions.Buyout{MFLID: tc.player})
		if err != nil {
			t.Fatalf("buyout %s: %v", tc.player, err)
		}
		if rec.Kind != transactions.KindBuyout || rec.PlayersAffected != 1 {
			t.Fatalf("receipt = %+v, want KindBuyout/1", rec)
		}
		if _, ok := s.Reader().Player(tc.player); ok {
			t.Fatalf("player %s still rostered after buyout", tc.player)
		}
		got, ok := s.Reader().CapUsed(tc.fr)
		if !ok || got != tc.want {
			t.Fatalf("franchise %s dead cap = %v (ok=%v), want %v", tc.fr, got, ok, tc.want)
		}
	}
}

// TestIntegration_BuyoutRejectsOutOfRangeRemaining proves the fail-loud edge (locked GQ3 / panel
// A6): 1 and 5 remaining years have no §12 rate, so the buyout is rejected and the player stays.
func TestIntegration_BuyoutRejectsOutOfRangeRemaining(t *testing.T) {
	s, c := buyStore(t)
	ctx := context.Background()
	for _, p := range []string{"0004" /*1 remaining*/, "0005" /*5 remaining*/} {
		if _, err := c.Execute(ctx, transactions.Buyout{MFLID: p}); err == nil {
			t.Fatalf("buyout of %s (out-of-range remaining) succeeded, want rejection", p)
		}
		if _, ok := s.Reader().Player(p); !ok {
			t.Fatalf("player %s was released despite a rejected buyout", p)
		}
	}
}

// TestIntegration_BuyoutOffseasonOnly proves the phase-RESTRICTION branch of the gate (the proof
// deferred from commit 3): a buyout is rejected in REGULAR_SEASON and permitted in OFFSEASON.
func TestIntegration_BuyoutOffseasonOnly(t *testing.T) {
	s, c := buyStore(t)
	ctx := context.Background()

	if _, err := c.Execute(ctx, transactions.AdvancePhase{To: domain.PhaseRegularSeason}); err != nil {
		t.Fatalf("advance to REGULAR_SEASON: %v", err)
	}
	if _, err := c.Execute(ctx, transactions.Buyout{MFLID: "0001"}); err == nil {
		t.Fatal("buyout succeeded in REGULAR_SEASON, want offseason-only rejection")
	}
	if _, ok := s.Reader().Player("0001"); !ok {
		t.Fatal("player released by a buyout that the phase gate should have blocked")
	}

	// Back to OFFSEASON: the same buyout now succeeds.
	if _, err := c.Execute(ctx, transactions.AdvancePhase{To: domain.PhaseOffseason}); err != nil {
		t.Fatalf("advance back to OFFSEASON: %v", err)
	}
	if _, err := c.Execute(ctx, transactions.Buyout{MFLID: "0001"}); err != nil {
		t.Fatalf("buyout in OFFSEASON: %v", err)
	}
}

// TestIntegration_BuyoutTwoPerSeasonLimit proves §12's "two per team per season": franchise 0009
// may buy out two players, and the third is rejected.
func TestIntegration_BuyoutTwoPerSeasonLimit(t *testing.T) {
	_, c := buyStore(t)
	ctx := context.Background()

	for _, p := range []string{"0091", "0092"} {
		if _, err := c.Execute(ctx, transactions.Buyout{MFLID: p}); err != nil {
			t.Fatalf("buyout %s: %v", p, err)
		}
	}
	if _, err := c.Execute(ctx, transactions.Buyout{MFLID: "0093"}); err == nil {
		t.Fatal("third buyout in one season succeeded, want the §12 two-per-season rejection")
	}
}
