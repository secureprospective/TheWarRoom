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

// extSeed rosters one franchise (0001) with players covering every §10 path:
//
//	0001: $6M, expires 2028 — the standard extend target (3 paid years remaining).
//	0002: $6M, expires 2026 — final year this season → no year remaining (UFA-ineligible).
//	0003: $8M, expires 2028 — 150%×$8M=$12M beats the $10M WR floor (price-above-floor).
//	0004: $6M, expires 2031 — already 6 paid years (2026..2031) → the ≤6-total ceiling.
//	0005: $6M, expires 2028 — mapped to FLAG (no §10 floor).
type extSeed struct{ t *testing.T }

func (s extSeed) Rosters(context.Context) ([]domain.Roster, error) {
	id := func(raw string) playerid.PlayerID {
		p, err := playerid.New(raw)
		if err != nil {
			s.t.Fatalf("playerid.New(%q): %v", raw, err)
		}
		return p
	}
	mk := func(raw string, sal domain.Money, exp int) domain.PlayerRecord {
		return domain.PlayerRecord{MFLID: id(raw), Salary: sal, ContractYear: exp,
			RosterStatus: domain.RosterActive, ContractStatus: domain.CStatusUFA}
	}
	return []domain.Roster{
		{FranchiseID: "0001", Players: []domain.PlayerRecord{
			mk("0001", 6*mil, 2028), mk("0002", 6*mil, 2026), mk("0003", 8*mil, 2028),
			mk("0004", 6*mil, 2031), mk("0005", 6*mil, 2028),
		}},
	}, nil
}

func extStore(t *testing.T) (*statepkg.Store, *transactions.Coordinator, tagDir) {
	t.Helper()
	pools, err := db.Open(context.Background(), filepath.Join(t.TempDir(), "ext.db"))
	if err != nil {
		t.Fatalf("db.Open: %v", err)
	}
	t.Cleanup(func() { _ = pools.Close() })
	s := statepkg.New(pools, "14432", 2026, nil)
	if err := s.Initialize(context.Background(), extSeed{t}); err != nil {
		t.Fatalf("Initialize: %v", err)
	}
	c, err := transactions.New(s.Writer())
	if err != nil {
		t.Fatalf("New coordinator: %v", err)
	}
	dir := tagDir{
		"0001": domain.PosWR, "0002": domain.PosWR, "0003": domain.PosWR,
		"0004": domain.PosWR, "0005": domain.PosFlag,
	}
	return s, c, dir
}

// TestIntegration_ExtendAppendsYearsAtFlooredPrice proves the §10 happy path: a +2 extension of
// the $6M WR appends two new paid years at the $10M WR floor (150%×$6M=$9M loses to it),
// lengthens the term to 2030, promotes the old UFA slot and lays a fresh one at 2031, and leaves
// the CURRENT-season cap untouched (an extension adds only FUTURE cap).
func TestIntegration_ExtendAppendsYearsAtFlooredPrice(t *testing.T) {
	s, c, dir := extStore(t)
	ctx := context.Background()

	before, _ := s.CapUsed("0001")
	if _, err := c.ExecuteExtension(ctx, "0001", 2, dir); err != nil {
		t.Fatalf("ExecuteExtension: %v", err)
	}

	cells, err := s.LedgerCells(ctx, "0001")
	if err != nil {
		t.Fatalf("LedgerCells: %v", err)
	}
	// Original paid years untouched; two new years at the $10M floor.
	for _, y := range []int{2026, 2027, 2028} {
		if cells[y] != 6*mil {
			t.Fatalf("original cell %d = %s, want $6M (untouched)", y, cells[y])
		}
	}
	for _, y := range []int{2029, 2030} {
		if cells[y] != 10*mil {
			t.Fatalf("extension cell %d = %s, want $10M (WR floor)", y, cells[y])
		}
	}
	if _, ok := cells[2031]; ok {
		t.Fatalf("2031 should be the new UFA slot (no cap), but appears as a PAID cell: %s", cells[2031])
	}
	// Term lengthened; the current-season cap is unchanged (future years don't count yet).
	if p, _ := s.Player("0001"); p.ExpirationYear != 2030 {
		t.Fatalf("expiration = %d, want 2030", p.ExpirationYear)
	}
	if after, _ := s.CapUsed("0001"); after != before {
		t.Fatalf("current-season cap moved on an extension: %s -> %s (only future years were added)", before, after)
	}
}

// TestIntegration_ExtendPricesAt150AboveFloor proves the 150% branch wins when it exceeds the
// floor: extending the $8M WR prices the new years at $12M (150%×$8M), above the $10M WR floor.
func TestIntegration_ExtendPricesAt150AboveFloor(t *testing.T) {
	s, c, dir := extStore(t)
	ctx := context.Background()

	if _, err := c.ExecuteExtension(ctx, "0003", 1, dir); err != nil {
		t.Fatalf("ExecuteExtension: %v", err)
	}
	cells, _ := s.LedgerCells(ctx, "0003")
	if cells[2029] != 12*mil { // 150%×$8M = $12M > $10M floor
		t.Fatalf("extension cell 2029 = %s, want $12M (150%% of $8M beats the floor)", cells[2029])
	}
}

// TestIntegration_ExtendResetsRestructureFlag proves the §10 unlock: an extension resets
// is_restructured, re-allowing one §11 restructure off the extended contract.
func TestIntegration_ExtendResetsRestructureFlag(t *testing.T) {
	s, c, dir := extStore(t)
	ctx := context.Background()

	if _, err := c.Execute(ctx, transactions.Restructure{MFLID: "0001", Move: 1 * mil}); err != nil {
		t.Fatalf("restructure: %v", err)
	}
	if p, _ := s.Player("0001"); !p.IsRestructured {
		t.Fatal("0001 not flagged restructured before extension")
	}
	if _, err := c.ExecuteExtension(ctx, "0001", 1, dir); err != nil {
		t.Fatalf("ExecuteExtension: %v", err)
	}
	if p, _ := s.Player("0001"); p.IsRestructured {
		t.Fatal("extension did NOT reset is_restructured — the §10 restructure unlock is missing")
	}
}

// TestIntegration_ExtendRejectsAndRollsBack proves every §10 rejection leaves state untouched:
// UFA-ineligible, unknown player, out-of-range years, the ≤6-total ceiling, no-floor position,
// a tagged player, the one-per-season limit, and no second extension off a prior one.
func TestIntegration_ExtendRejectsAndRollsBack(t *testing.T) {
	s, c, dir := extStore(t)
	ctx := context.Background()

	// A final-year player has no year remaining (UFAs ineligible).
	if _, err := c.ExecuteExtension(ctx, "0002", 1, dir); err == nil {
		t.Fatal("extending a final-year (UFA-ineligible) player was accepted")
	}
	// An unknown player is rejected before any tx.
	if _, err := c.ExecuteExtension(ctx, "9999", 1, dir); err == nil {
		t.Fatal("extending an unrostered player was accepted")
	}
	// Out-of-range added years (validate gate).
	if _, err := c.ExecuteExtension(ctx, "0001", 4, dir); err == nil {
		t.Fatal("a 4-year extension was accepted (max 3)")
	}
	if _, err := c.ExecuteExtension(ctx, "0001", 0, dir); err == nil {
		t.Fatal("a 0-year extension was accepted")
	}
	// The ≤6-total ceiling: 0004 already has 6 paid years.
	if _, err := c.ExecuteExtension(ctx, "0004", 1, dir); err == nil {
		t.Fatal("extending past 6 total contract years was accepted")
	}
	// A position with no §10 floor is rejected in the Coordinator.
	if _, err := c.ExecuteExtension(ctx, "0005", 1, dir); err == nil {
		t.Fatal("extending a FLAG-position player (no floor) was accepted")
	}
	// A franchise-tagged player is out of scope in v1 (120%-of-tag pricing deferred).
	if _, err := c.ExecuteTag(ctx, "0003", dir); err != nil {
		t.Fatalf("tag setup failed: %v", err)
	}
	if _, err := c.ExecuteExtension(ctx, "0003", 1, dir); err == nil {
		t.Fatal("extending a franchise-tagged player was accepted (out of scope in v1)")
	}

	// Nothing above mutated a contract term: 0001 still expires 2028, 0004 still 2031.
	if p, _ := s.Player("0001"); p.ExpirationYear != 2028 {
		t.Fatalf("0001 term changed by a rejected extension: expiration %d, want 2028", p.ExpirationYear)
	}
	if p, _ := s.Player("0004"); p.ExpirationYear != 2031 {
		t.Fatalf("0004 term changed by a rejected extension: expiration %d, want 2031", p.ExpirationYear)
	}
}

// TestIntegration_ExtendLimits proves the one-per-season and no-second-extension guards on a
// clean store: the first extension of a franchise succeeds, a second extension (any player, same
// franchise, same season) is rejected, and re-extending the already-extended player is rejected.
func TestIntegration_ExtendLimits(t *testing.T) {
	s, c, dir := extStore(t)
	ctx := context.Background()

	if _, err := c.ExecuteExtension(ctx, "0001", 1, dir); err != nil {
		t.Fatalf("first extension: %v", err)
	}
	// Second extension, same franchise (0003 also on 0001), same season → rejected.
	if _, err := c.ExecuteExtension(ctx, "0003", 1, dir); err == nil {
		t.Fatal("a second extension for the same franchise in one season was accepted")
	}
	// The rejected second extension left 0003 untouched.
	if p, _ := s.Player("0003"); p.ExpirationYear != 2028 {
		t.Fatalf("rejected second extension changed 0003: expiration %d, want 2028", p.ExpirationYear)
	}
	// Re-extending the already-extended 0001 is rejected (no second extension off a prior one),
	// independent of the per-season counter.
	if _, err := c.ExecuteExtension(ctx, "0001", 1, dir); err == nil {
		t.Fatal("a second extension off a prior extension was accepted")
	}
}
