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

// signSeed serves the free-agency fixtures. Seed season 2026.
//
//	0001 = 0011 exp 2026 ($5M, expires after 2026 → promoted on rollover to 2027)
//	     + 0012 exp 2028 ($5M)
//	0002 = 0021 exp 2029 ($4M) — the signing destination franchise
type signSeed struct{ t *testing.T }

func (s signSeed) Rosters(context.Context) ([]domain.Roster, error) {
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
	return []domain.Roster{
		{FranchiseID: "0001", Players: []domain.PlayerRecord{mk("0011", 5*mil, 2026), mk("0012", 5*mil, 2028)}},
		{FranchiseID: "0002", Players: []domain.PlayerRecord{mk("0021", 4*mil, 2029)}},
	}, nil
}

func signStore(t *testing.T) (*statepkg.Store, *transactions.Coordinator) {
	t.Helper()
	pools, err := db.Open(context.Background(), filepath.Join(t.TempDir(), "sign.db"))
	if err != nil {
		t.Fatalf("db.Open: %v", err)
	}
	t.Cleanup(func() { _ = pools.Close() })
	s := statepkg.New(pools, "14432", 2026)
	if err := s.Initialize(context.Background(), signSeed{t}); err != nil {
		t.Fatalf("Initialize: %v", err)
	}
	c, err := transactions.New(s.Writer())
	if err != nil {
		t.Fatalf("New coordinator: %v", err)
	}
	return s, c
}

func poolOf(t *testing.T, s *statepkg.Store) []string {
	t.Helper()
	ids, err := s.FreeAgents(context.Background())
	if err != nil {
		t.Fatalf("FreeAgents: %v", err)
	}
	return ids
}

func contains(ids []string, id string) bool {
	for _, x := range ids {
		if x == id {
			return true
		}
	}
	return false
}

// TestIntegration_WaivedPlayerEntersPoolThenSigns is the core happy path: a §8 cut drops a player
// into the FREE_AGENT pool, and a SIGN rosters him on a new franchise with a fresh flat contract —
// his cap lands on the signing franchise and he leaves the pool.
func TestIntegration_WaivedPlayerEntersPoolThenSigns(t *testing.T) {
	s, c := signStore(t)
	ctx := context.Background()

	if _, err := c.Execute(ctx, transactions.Waiver{MFLID: "0011"}); err != nil {
		t.Fatalf("waive 0011: %v", err)
	}
	if pool := poolOf(t, s); !contains(pool, "0011") {
		t.Fatalf("after cut, pool = %v, want to contain 0011", pool)
	}
	// Sign 0011 to 0002 for 3 years at $3M.
	if _, err := c.Execute(ctx, transactions.Sign{MFLID: "0011", FranchiseID: "0002", Salary: 3 * mil, Years: 3}); err != nil {
		t.Fatalf("sign 0011: %v", err)
	}
	// He is rostered on 0002 now, and out of the pool.
	ps, ok := s.Reader().Player("0011")
	if !ok || ps.FranchiseID != "0002" {
		t.Fatalf("after sign, Player(0011) = %+v ok=%v, want on 0002", ps, ok)
	}
	if pool := poolOf(t, s); contains(pool, "0011") {
		t.Fatalf("after sign, pool = %v, still contains 0011", pool)
	}
	// 0002's cap = its own 0021 ($4M) + the new 0011 cell ($3M) = $7M.
	if got, ok := s.Reader().CapUsed("0002"); !ok || got != 7*mil {
		t.Fatalf("CapUsed(0002) = %v ok=%v, want $7M (0021 $4M + signed 0011 $3M)", got, ok)
	}
}

// TestIntegration_SignRejectsNonFreeAgent proves the eligibility gate: a rostered player (never
// released) is not signable, and a RETIRED player (off-roster but barred) is not signable either.
func TestIntegration_SignRejectsNonFreeAgent(t *testing.T) {
	_, c := signStore(t)
	ctx := context.Background()

	// 0012 is rostered — no status event, not a free agent.
	if _, err := c.Execute(ctx, transactions.Sign{MFLID: "0012", FranchiseID: "0002", Salary: 3 * mil, Years: 2}); err == nil {
		t.Fatal("signed a rostered player, want not-a-free-agent rejection")
	}
	// Retire 0021 → RETIRED, off-roster but barred from signing.
	if _, err := c.Execute(ctx, transactions.Retirement{MFLID: "0021"}); err != nil {
		t.Fatalf("retire 0021: %v", err)
	}
	if _, err := c.Execute(ctx, transactions.Sign{MFLID: "0021", FranchiseID: "0001", Salary: 2 * mil, Years: 1}); err == nil {
		t.Fatal("signed a RETIRED player, want barred rejection")
	}
}

// TestIntegration_BuyoutLockoutBlocksSign proves the derived §12 lockout: a bought-out player is a
// free agent but cannot be signed in the buyout season, and the lockout lifts after the season
// rolls (the dead_cap row's league_year drops below the new current season).
func TestIntegration_BuyoutLockoutBlocksSign(t *testing.T) {
	s, c := signStore(t)
	ctx := context.Background()

	// Buy out 0012 (exp 2028 → 2 remaining after 2026, a valid §12 buyout). OFFSEASON at seed.
	if _, err := c.Execute(ctx, transactions.Buyout{MFLID: "0012"}); err != nil {
		t.Fatalf("buyout 0012: %v", err)
	}
	if pool := poolOf(t, s); !contains(pool, "0012") {
		t.Fatalf("after buyout, pool = %v, want to contain 0012", pool)
	}
	// Same season, OFFSEASON: locked.
	if _, err := c.Execute(ctx, transactions.Sign{MFLID: "0012", FranchiseID: "0002", Salary: 2 * mil, Years: 1}); err == nil {
		t.Fatal("signed a bought-out player in the buyout season (offseason), want §12 lockout rejection")
	}
	// Still season N, now REGULAR_SEASON (SIGN is in-window here) — the lockout must STILL hold. A
	// regression that lifted it before the season rolled would slip past the offseason check above,
	// so this in-window attempt gives the lockout real teeth (GLM L7).
	if _, err := c.Execute(ctx, transactions.AdvancePhase{To: domain.PhaseRegularSeason}); err != nil {
		t.Fatalf("advance to REGULAR_SEASON: %v", err)
	}
	if _, err := c.Execute(ctx, transactions.Sign{MFLID: "0012", FranchiseID: "0002", Salary: 2 * mil, Years: 1}); err == nil {
		t.Fatal("signed a bought-out player in REGULAR_SEASON of the buyout season, want §12 lockout still in force")
	}
	// Roll to 2027 (PLAYOFFS-gated). Already in REGULAR_SEASON, so advance only to PLAYOFFS.
	if _, err := c.Execute(ctx, transactions.AdvancePhase{To: domain.PhasePlayoffs}); err != nil {
		t.Fatalf("advance to PLAYOFFS: %v", err)
	}
	if _, err := c.Execute(ctx, transactions.RolloverSeason{Note: "roll"}); err != nil {
		t.Fatalf("rollover: %v", err)
	}
	if _, err := c.Execute(ctx, transactions.Sign{MFLID: "0012", FranchiseID: "0002", Salary: 2 * mil, Years: 1}); err != nil {
		t.Fatalf("sign 0012 after rollover: %v (lockout must lift the following offseason)", err)
	}
}

// TestIntegration_SignBlockedInPlayoffs proves the §6 signing window (Q4): a SIGN is rejected in
// PLAYOFFS (the window is offseason + regular season), even for a legitimate free agent.
func TestIntegration_SignBlockedInPlayoffs(t *testing.T) {
	_, c := signStore(t)
	ctx := context.Background()

	if _, err := c.Execute(ctx, transactions.Waiver{MFLID: "0011"}); err != nil {
		t.Fatalf("waive 0011: %v", err)
	}
	for _, to := range []domain.Phase{domain.PhaseRegularSeason, domain.PhasePlayoffs} {
		if _, err := c.Execute(ctx, transactions.AdvancePhase{To: to}); err != nil {
			t.Fatalf("advance to %s: %v", to, err)
		}
	}
	if _, err := c.Execute(ctx, transactions.Sign{MFLID: "0011", FranchiseID: "0002", Salary: 3 * mil, Years: 2}); err == nil {
		t.Fatal("signed in PLAYOFFS, want out-of-window rejection")
	}
}

// TestIntegration_RolloverPromotesExpiredContractToPool proves the §14 UFA promotion: a player
// whose contract expires (no PAID cell in the upcoming season) is released into the pool at $0 dead
// cap when the season rolls — and is then signable. 0011 (exp 2026) is rostered through 2026 and
// promoted on the roll to 2027.
func TestIntegration_RolloverPromotesExpiredContractToPool(t *testing.T) {
	s, c := signStore(t)
	ctx := context.Background()

	if pool := poolOf(t, s); contains(pool, "0011") {
		t.Fatalf("pre-rollover pool = %v, must not contain still-rostered 0011", pool)
	}
	for _, to := range []domain.Phase{domain.PhaseRegularSeason, domain.PhasePlayoffs} {
		if _, err := c.Execute(ctx, transactions.AdvancePhase{To: to}); err != nil {
			t.Fatalf("advance to %s: %v", to, err)
		}
	}
	if _, err := c.Execute(ctx, transactions.RolloverSeason{Note: "roll to 2027"}); err != nil {
		t.Fatalf("rollover: %v", err)
	}
	// 0011 expired → promoted to the pool; 0012 (exp 2028) survived on the roster.
	if pool := poolOf(t, s); !contains(pool, "0011") {
		t.Fatalf("post-rollover pool = %v, want to contain the expired 0011", pool)
	}
	if _, ok := s.Reader().Player("0011"); ok {
		t.Fatal("expired 0011 still rostered after rollover, want released to pool")
	}
	if _, ok := s.Reader().Player("0012"); !ok {
		t.Fatal("0012 (exp 2028) vanished after rollover, want kept on roster")
	}
	// Rollover charges no dead cap for an expiry: 0001's cap is just 0012's $5M 2027 cell.
	if got, ok := s.Reader().CapUsed("0001"); !ok || got != 5*mil {
		t.Fatalf("CapUsed(0001) = %v ok=%v, want $5M (expiry adds no dead cap)", got, ok)
	}
	// And the promoted free agent is signable.
	if _, err := c.Execute(ctx, transactions.Sign{MFLID: "0011", FranchiseID: "0002", Salary: 2 * mil, Years: 2}); err != nil {
		t.Fatalf("sign promoted 0011: %v", err)
	}
}

// TestIntegration_SignValidationRejectsBadShape proves the pre-tx validation: an empty franchise, a
// non-positive salary, and an out-of-range year count are rejected before any state changes.
func TestIntegration_SignValidationRejectsBadShape(t *testing.T) {
	_, c := signStore(t)
	ctx := context.Background()
	if _, err := c.Execute(ctx, transactions.Waiver{MFLID: "0011"}); err != nil {
		t.Fatalf("waive 0011: %v", err)
	}
	bad := []transactions.Sign{
		{MFLID: "0011", FranchiseID: "", Salary: 3 * mil, Years: 2},
		{MFLID: "0011", FranchiseID: "0002", Salary: 0, Years: 2},
		{MFLID: "0011", FranchiseID: "0002", Salary: 3 * mil, Years: 0},
		{MFLID: "0011", FranchiseID: "0002", Salary: 3 * mil, Years: 5},
	}
	for i, req := range bad {
		if _, err := c.Execute(ctx, req); err == nil {
			t.Fatalf("bad sign %d (%+v) succeeded, want validation rejection", i, req)
		}
	}
}

// TestIntegration_SignRejectsUnknownFranchise proves the phantom-franchise guard (GLM L2): signing
// a free agent to a franchise id that fields no roster is rejected, so a typo can't strand a player
// in a franchise CapUsed/GetFranchiseState will never surface.
func TestIntegration_SignRejectsUnknownFranchise(t *testing.T) {
	_, c := signStore(t)
	ctx := context.Background()
	if _, err := c.Execute(ctx, transactions.Waiver{MFLID: "0011"}); err != nil {
		t.Fatalf("waive 0011: %v", err)
	}
	if _, err := c.Execute(ctx, transactions.Sign{MFLID: "0011", FranchiseID: "9X9X", Salary: 3 * mil, Years: 2}); err == nil {
		t.Fatal("signed to an unknown franchise, want rejection (phantom-franchise guard)")
	}
}

// TestIntegration_SignRejectsSubGridSalary proves the post-round $0 guard (GLM L4): a salary that
// is positive but rounds below the $10k grid to $0 is rejected, not laid as a free contract.
func TestIntegration_SignRejectsSubGridSalary(t *testing.T) {
	_, c := signStore(t)
	ctx := context.Background()
	if _, err := c.Execute(ctx, transactions.Waiver{MFLID: "0011"}); err != nil {
		t.Fatalf("waive 0011: %v", err)
	}
	// $100 (10_000 cents) is > 0 at validate but snaps to $0 on the $10k grid.
	if _, err := c.Execute(ctx, transactions.Sign{MFLID: "0011", FranchiseID: "0002", Salary: domain.Money(10_000), Years: 1}); err == nil {
		t.Fatal("signed at a sub-$10k salary that rounds to $0, want rejection")
	}
}
