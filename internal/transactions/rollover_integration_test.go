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

// rollSeed serves the §14 rollover fixtures. Seed season is 2026; the ledger flat-fills PAID cells
// from 2026 through each player's expiration year.
//
//	0001 = 0011 exp 2026 ($5M, last year) + 0012 exp 2028 ($5M) → cap-DRIFT fixture: CapUsed is
//	       $10M at 2026 and $5M at 2027 (0011's only cell is 2026), so a rollover is observable.
//	0009 = 0091/0092 exp 2028 + 0093 exp 2029 (all $6M) → op-count-reset + dead-cap-persistence
//	       fixture (two buyouts hit the per-season limit; the third rejects until the season rolls).
//	0005 = 0051 exp 2029 ($6M) → restructure-flag-persistence fixture (D4).
type rollSeed struct{ t *testing.T }

func (s rollSeed) Rosters(context.Context) ([]domain.Roster, error) {
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
		{FranchiseID: "0009", Players: []domain.PlayerRecord{mk("0091", 6*mil, 2028), mk("0092", 6*mil, 2028), mk("0093", 6*mil, 2029)}},
		{FranchiseID: "0005", Players: []domain.PlayerRecord{mk("0051", 6*mil, 2029)}},
	}, nil
}

func rollStore(t *testing.T) (*statepkg.Store, *transactions.Coordinator) {
	t.Helper()
	pools, err := db.Open(context.Background(), filepath.Join(t.TempDir(), "roll.db"))
	if err != nil {
		t.Fatalf("db.Open: %v", err)
	}
	t.Cleanup(func() { _ = pools.Close() })
	s := statepkg.New(pools, "14432", 2026)
	if err := s.Initialize(context.Background(), rollSeed{t}); err != nil {
		t.Fatalf("Initialize: %v", err)
	}
	c, err := transactions.New(s.Writer())
	if err != nil {
		t.Fatalf("New coordinator: %v", err)
	}
	return s, c
}

// toPlayoffs advances OFFSEASON → REGULAR_SEASON → PLAYOFFS, the legal from-phase for a rollover.
func toPlayoffs(t *testing.T, c *transactions.Coordinator) {
	t.Helper()
	ctx := context.Background()
	for _, to := range []domain.Phase{domain.PhaseRegularSeason, domain.PhasePlayoffs} {
		if _, err := c.Execute(ctx, transactions.AdvancePhase{To: to}); err != nil {
			t.Fatalf("advance to %s: %v", to, err)
		}
	}
}

func capOf(t *testing.T, s *statepkg.Store, fr string) domain.Money {
	t.Helper()
	got, ok := s.Reader().CapUsed(fr)
	if !ok {
		t.Fatalf("CapUsed(%s): franchise not found", fr)
	}
	return got
}

// TestIntegration_RolloverAdvancesSeasonPhaseAndCap is the centerpiece drift test (D5/D6): a rollover
// from PLAYOFFS moves the league to OFFSEASON of the NEXT season, and the cap — derived through the
// store's own post-commit reload — follows the new year. Franchise 0001 loses player 0011's 2026-only
// cell, so its CapUsed drops $10M → $5M. The assertion has teeth: if the season failed to advance
// while the snapshot rolled to N+1, load()'s roster join (WHERE r.season = staleN) would match zero
// rows and CapUsed would fail loud as "franchise not found" — either way, not the $5M this expects.
func TestIntegration_RolloverAdvancesSeasonPhaseAndCap(t *testing.T) {
	s, c := rollStore(t)
	ctx := context.Background()

	if got := capOf(t, s, "0001"); got != 10*mil {
		t.Fatalf("pre-rollover CapUsed(0001) = %v, want $10M", got)
	}
	toPlayoffs(t, c)

	rec, err := c.Execute(ctx, transactions.RolloverSeason{Note: "end of 2026"})
	if err != nil {
		t.Fatalf("rollover: %v", err)
	}
	if rec.Kind != transactions.KindRolloverSeason || rec.PlayersAffected != 0 {
		t.Fatalf("receipt = %+v, want KindRolloverSeason/0", rec)
	}
	if ph, err := s.CurrentPhase(ctx); err != nil || ph != domain.PhaseOffseason {
		t.Fatalf("post-rollover phase = %q (err=%v), want OFFSEASON", ph, err)
	}
	// The drift: 0011 (exp 2026) has no 2027 cell, so only 0012's $5M cell counts at 2027.
	if got := capOf(t, s, "0001"); got != 5*mil {
		t.Fatalf("post-rollover CapUsed(0001) = %v, want $5M (cap follows the derived season)", got)
	}
	// Neither player was released — a rollover moves the calendar, not the roster.
	if _, ok := s.Reader().Player("0012"); !ok {
		t.Fatal("player 0012 vanished after a rollover (must not release anyone)")
	}
}

// TestIntegration_RolloverSurvivesReboot closes the make-or-break loop (GLM L1): it runs the REAL
// ROLLOVER_SEASON op, then closes and reopens the store from the SAME DB with the ORIGINAL config
// season (2026), and asserts the reopened store DERIVES the rolled season — phase OFFSEASON, cap
// tracks N+1, and it does not re-seed. The boot-derivation test and the op test each covered only
// one half of this seam; this joins them through the actual write primitive + a genuine reboot.
func TestIntegration_RolloverSurvivesReboot(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "reboot.db")
	ctx := context.Background()

	pools, err := db.Open(ctx, path)
	if err != nil {
		t.Fatalf("db.Open: %v", err)
	}
	s := statepkg.New(pools, "14432", 2026)
	if err := s.Initialize(ctx, rollSeed{t}); err != nil {
		t.Fatalf("Initialize: %v", err)
	}
	c, err := transactions.New(s.Writer())
	if err != nil {
		t.Fatalf("New coordinator: %v", err)
	}
	for _, to := range []domain.Phase{domain.PhaseRegularSeason, domain.PhasePlayoffs} {
		if _, err := c.Execute(ctx, transactions.AdvancePhase{To: to}); err != nil {
			t.Fatalf("advance to %s: %v", to, err)
		}
	}
	if _, err := c.Execute(ctx, transactions.RolloverSeason{Note: "reboot"}); err != nil {
		t.Fatalf("rollover: %v", err)
	}
	if got, ok := s.Reader().CapUsed("0001"); !ok || got != 5*mil {
		t.Fatalf("pre-reboot CapUsed(0001) = %v ok=%v, want $5M", got, ok)
	}
	if err := pools.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}

	// Reboot from the same DB with the UNCHANGED config season — must derive 2027 from the log.
	pools2, err := db.Open(ctx, path)
	if err != nil {
		t.Fatalf("db.Open reboot: %v", err)
	}
	t.Cleanup(func() { _ = pools2.Close() })
	s2 := statepkg.New(pools2, "14432", 2026)
	if err := s2.Initialize(ctx, rollSeed{t}); err != nil {
		t.Fatalf("reboot Initialize: %v", err)
	}
	if ph, err := s2.CurrentPhase(ctx); err != nil || ph != domain.PhaseOffseason {
		t.Fatalf("rebooted phase = %q (err=%v), want OFFSEASON", ph, err)
	}
	if got, ok := s2.Reader().CapUsed("0001"); !ok || got != 5*mil {
		t.Fatalf("rebooted CapUsed(0001) = %v ok=%v, want $5M (derived season survived reboot)", got, ok)
	}
	if got := s2.Reader().Franchises(); len(got) != 3 {
		t.Fatalf("rebooted Franchises = %v, want the 3 seeded (must NOT re-seed)", got)
	}
}

// TestIntegration_RolloverResetsOpCountsDropsDeadCap proves D2 + D3: the per-season op counters
// reset across the boundary (a franchise at its 2-buyout limit can buy out again next season) and
// this-season dead cap does NOT carry — the year-N dead-cap rows stay put as audit but sum to 0 at
// N+1. Franchise 0009 buys out two players (hitting the limit), rolls over, and the dead cap is gone.
func TestIntegration_RolloverResetsOpCountsDropsDeadCap(t *testing.T) {
	s, c := rollStore(t)
	ctx := context.Background()

	// OFFSEASON(2026): two buyouts land dead cap; the third is rejected by the per-season limit.
	for _, p := range []string{"0091", "0092"} {
		if _, err := c.Execute(ctx, transactions.Buyout{MFLID: p}); err != nil {
			t.Fatalf("buyout %s: %v", p, err)
		}
	}
	if _, err := c.Execute(ctx, transactions.Buyout{MFLID: "0093"}); err == nil {
		t.Fatal("third buyout succeeded in one season, want per-season-limit rejection")
	}
	// Dead cap from the two buyouts is on the books at 2026 (0093's $6M cell + two dead-cap charges).
	if got := capOf(t, s, "0009"); got <= 6*mil {
		t.Fatalf("pre-rollover CapUsed(0009) = %v, want > $6M (carries dead cap)", got)
	}

	toPlayoffs(t, c)
	if _, err := c.Execute(ctx, transactions.RolloverSeason{Note: "roll"}); err != nil {
		t.Fatalf("rollover: %v", err)
	}

	// At 2027 the 2026 dead-cap rows no longer match: only 0093's $6M 2027 cell counts. Dead cap dropped.
	if got := capOf(t, s, "0009"); got != 6*mil {
		t.Fatalf("post-rollover CapUsed(0009) = %v, want $6M (year-N dead cap must not carry)", got)
	}
	// The buyout counter reset: 0093 (remaining 2 at 2027) can now be bought out despite the prior
	// season's two buyouts.
	if _, err := c.Execute(ctx, transactions.Buyout{MFLID: "0093"}); err != nil {
		t.Fatalf("post-rollover buyout 0093: %v (per-season op count must reset)", err)
	}
}

// TestIntegration_RolloverPreservesRestructureFlag proves D4: is_restructured is a per-CONTRACT
// lifetime guard, NOT reset by a rollover. Franchise 0005 restructures player 0051 in 2026; after the
// season rolls, a second restructure of the SAME contract is rejected — and because 0005's per-season
// restructure count reset (D3), the ONLY thing blocking it is the persistent lifetime flag.
func TestIntegration_RolloverPreservesRestructureFlag(t *testing.T) {
	_, c := rollStore(t)
	ctx := context.Background()

	if _, err := c.Execute(ctx, transactions.Restructure{MFLID: "0051", Move: 1 * mil}); err != nil {
		t.Fatalf("restructure 0051 in 2026: %v", err)
	}
	toPlayoffs(t, c)
	if _, err := c.Execute(ctx, transactions.RolloverSeason{Note: "roll"}); err != nil {
		t.Fatalf("rollover: %v", err)
	}
	// OFFSEASON(2027): the per-season count is fresh, but 0051's lifetime restructure flag persists.
	if _, err := c.Execute(ctx, transactions.Restructure{MFLID: "0051", Move: 1 * mil}); err == nil {
		t.Fatal("re-restructured the same contract after a rollover, want lifetime-guard rejection (D4)")
	}
}

// TestIntegration_RolloverRejectedOutsidePlayoffs proves the PLAYOFFS-only gate: a rollover in
// OFFSEASON and in REGULAR_SEASON is rejected, and the season does not move (0001's cap is unchanged).
func TestIntegration_RolloverRejectedOutsidePlayoffs(t *testing.T) {
	s, c := rollStore(t)
	ctx := context.Background()

	if _, err := c.Execute(ctx, transactions.RolloverSeason{Note: "x"}); err == nil {
		t.Fatal("rollover succeeded in OFFSEASON, want PLAYOFFS-only rejection")
	}
	if _, err := c.Execute(ctx, transactions.AdvancePhase{To: domain.PhaseRegularSeason}); err != nil {
		t.Fatalf("advance to REGULAR_SEASON: %v", err)
	}
	if _, err := c.Execute(ctx, transactions.RolloverSeason{Note: "x"}); err == nil {
		t.Fatal("rollover succeeded in REGULAR_SEASON, want PLAYOFFS-only rejection")
	}
	if got := capOf(t, s, "0001"); got != 10*mil {
		t.Fatalf("CapUsed(0001) = %v after rejected rollovers, want unchanged $10M", got)
	}
}

// TestIntegration_RolloverIsMonotonic proves D5's one-way rule: after a rollover, rolling the PHASE
// backward to PLAYOFFS (a legal ADVANCE_PHASE correction) does NOT rewind the season — the cap stays
// at the new year, and a second rollover moves forward again rather than back.
func TestIntegration_RolloverIsMonotonic(t *testing.T) {
	s, c := rollStore(t)
	ctx := context.Background()

	toPlayoffs(t, c)
	if _, err := c.Execute(ctx, transactions.RolloverSeason{Note: "to 2027"}); err != nil {
		t.Fatalf("first rollover: %v", err)
	}
	if got := capOf(t, s, "0001"); got != 5*mil {
		t.Fatalf("after rollover CapUsed(0001) = %v, want $5M (2027)", got)
	}
	// Phase rollback to PLAYOFFS is allowed, but the season must stay at 2027 (append carries the
	// current season, never a decrement) — the cap does not revert to the 2026 $10M.
	if _, err := c.Execute(ctx, transactions.AdvancePhase{To: domain.PhasePlayoffs}); err != nil {
		t.Fatalf("phase rollback to PLAYOFFS: %v", err)
	}
	if got := capOf(t, s, "0001"); got != 5*mil {
		t.Fatalf("after phase rollback CapUsed(0001) = %v, want unchanged $5M (season is monotonic)", got)
	}
	// A second rollover goes forward to 2028: 0012 (exp 2028) still has a 2028 cell → $5M holds.
	if _, err := c.Execute(ctx, transactions.RolloverSeason{Note: "to 2028"}); err != nil {
		t.Fatalf("second rollover: %v", err)
	}
	if got := capOf(t, s, "0001"); got != 5*mil {
		t.Fatalf("after second rollover CapUsed(0001) = %v, want $5M (2028, 0012's last cell)", got)
	}
}
