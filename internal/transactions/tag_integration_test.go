package transactions_test

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/db"
	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/normalize"
	"github.com/secureprospective/TheWarRoom/internal/playerid"
	statepkg "github.com/secureprospective/TheWarRoom/internal/store/state"
	"github.com/secureprospective/TheWarRoom/internal/transactions"
)

// tagDir is a fake transactions.Directory mapping the seeded ids to positions (the store is
// position-blind; §9's top-5-by-position join lives in the players DB).
type tagDir map[string]domain.Position

func (d tagDir) Facts(mflID string) (normalize.PlayerFacts, bool) {
	pos, ok := d[mflID]
	if !ok {
		return normalize.PlayerFacts{}, false
	}
	return normalize.PlayerFacts{Position: pos}, true
}

// tagSeed: five WRs whose top-5 average is a clean $6M ((10+8+6+4+2)/5), spread across two
// franchises so the aggregation genuinely crosses rosters.
type tagSeed struct{ t *testing.T }

func (s tagSeed) Rosters(context.Context) ([]domain.Roster, error) {
	id := func(raw string) playerid.PlayerID {
		p, err := playerid.New(raw)
		if err != nil {
			s.t.Fatalf("playerid.New(%q): %v", raw, err)
		}
		return p
	}
	mk := func(raw string, sal domain.Money) domain.PlayerRecord {
		return domain.PlayerRecord{MFLID: id(raw), Salary: sal, ContractYear: 2028,
			RosterStatus: domain.RosterActive, ContractStatus: domain.CStatusUFA}
	}
	return []domain.Roster{
		{FranchiseID: "0001", Players: []domain.PlayerRecord{mk("0010", 10*mil), mk("0011", 8*mil)}},
		{FranchiseID: "0002", Players: []domain.PlayerRecord{mk("0020", 6*mil), mk("0021", 4*mil), mk("0022", 2*mil)}},
	}, nil
}

func tagStore(t *testing.T) (*statepkg.Store, *transactions.Coordinator, tagDir) {
	t.Helper()
	pools, err := db.Open(context.Background(), filepath.Join(t.TempDir(), "tag.db"))
	if err != nil {
		t.Fatalf("db.Open: %v", err)
	}
	t.Cleanup(func() { _ = pools.Close() })
	s := statepkg.New(pools, "14432", 2026)
	if err := s.Initialize(context.Background(), tagSeed{t}); err != nil {
		t.Fatalf("Initialize: %v", err)
	}
	c, err := transactions.New(s.Writer())
	if err != nil {
		t.Fatalf("New coordinator: %v", err)
	}
	dir := tagDir{"0010": domain.PosWR, "0011": domain.PosWR, "0020": domain.PosWR, "0021": domain.PosWR, "0022": domain.PosWR}
	return s, c, dir
}

// TestIntegration_TagSetsCapToTopFiveAverage proves the §9 happy path: tagging the $2M WR
// sets his salary to the $6M top-5 average (the floor of 120%×$2M=$2.4M loses), and the
// franchise cap rises by exactly the difference — cap is the sum of contracts.
func TestIntegration_TagSetsCapToTopFiveAverage(t *testing.T) {
	s, c, dir := tagStore(t)
	ctx := context.Background()

	if before, _ := s.CapUsed("0002"); before != 12*mil { // $6M + $4M + $2M
		t.Fatalf("pre-tag cap = %s, want $12M", before)
	}

	if _, err := c.ExecuteTag(ctx, "0022", dir); err != nil {
		t.Fatalf("ExecuteTag: %v", err)
	}

	p, _ := s.Player("0022")
	if !p.IsTagged {
		t.Fatal("player 0022 not flagged tagged")
	}
	if p.AdjustedSalary != 6*mil { // the $6M top-5 WR average
		t.Fatalf("tagged cap-counting salary = %s, want $6M (top-5 avg)", p.AdjustedSalary)
	}
	if got, _ := s.CapUsed("0002"); got != 16*mil { // $6M + $4M + $6M (was $2M, now $6M)
		t.Fatalf("post-tag cap = %s, want $16M", got)
	}
}

// TestIntegration_TagFloorLifts proves the §9 120%-of-prior floor: tagging the $10M WR whose
// 120% ($12M) exceeds the $6M average sets his salary to $12M, not the average.
func TestIntegration_TagFloorLifts(t *testing.T) {
	s, c, dir := tagStore(t)

	if _, err := c.ExecuteTag(context.Background(), "0010", dir); err != nil {
		t.Fatalf("ExecuteTag: %v", err)
	}
	if p, _ := s.Player("0010"); p.AdjustedSalary != 12*mil { // 120% × $10M beats the $6M avg
		t.Fatalf("tagged salary = %s, want $12M (120%% floor lifts above the average)", p.AdjustedSalary)
	}
	if got, _ := s.CapUsed("0001"); got != 20*mil { // $12M + $8M
		t.Fatalf("post-tag cap = %s, want $20M", got)
	}
}

// TestIntegration_TagRejectsAndRollsBack proves §9's limits and that a rejected tag mutates
// nothing: one tag per team per season, and one tag per contract.
func TestIntegration_TagRejectsAndRollsBack(t *testing.T) {
	s, c, dir := tagStore(t)
	ctx := context.Background()

	if _, err := c.ExecuteTag(ctx, "0022", dir); err != nil {
		t.Fatalf("first tag failed: %v", err)
	}
	// Second tag, same franchise (0021 also on 0002), same season → rejected.
	if _, err := c.ExecuteTag(ctx, "0021", dir); err == nil {
		t.Fatal("a second tag for the same franchise in one season was accepted")
	}
	if p, _ := s.Player("0021"); p.IsTagged {
		t.Fatal("the rejected second tag still flagged 0021")
	}
	// One per contract: re-tagging the already-tagged 0022 → rejected.
	if _, err := c.ExecuteTag(ctx, "0022", dir); err == nil {
		t.Fatal("a second tag of the same contract was accepted")
	}
	// An unknown player id → rejected before any tx.
	if _, err := c.ExecuteTag(ctx, "9999", dir); err == nil {
		t.Fatal("tagging an unrostered player was accepted")
	}
}

// TestIntegration_TagResetsRestructureFlag proves the GLM-M4 fix: tagging a
// previously-restructured player is allowed (a tag is a fresh contract) and RESETS the
// restructure flag, so a later §8 cut of the tagged player charges the standard 35% dead
// cap — NOT the restructured 50% on the tag figure.
func TestIntegration_TagResetsRestructureFlag(t *testing.T) {
	s, c, dir := tagStore(t)
	ctx := context.Background()

	// 0010 ($10M base) is restructure-eligible. Restructure, then tag him.
	if _, err := c.Execute(ctx, transactions.Restructure{MFLID: "0010", Move: 1 * mil}); err != nil {
		t.Fatalf("restructure 0010: %v", err)
	}
	if p, _ := s.Player("0010"); !p.IsRestructured {
		t.Fatal("0010 not flagged restructured after restructure")
	}
	// Tag is a different op_kind, so it is NOT blocked by the restructure per-season counter.
	if _, err := c.ExecuteTag(ctx, "0010", dir); err != nil {
		t.Fatalf("tag of a restructured player was rejected: %v", err)
	}
	p, _ := s.Player("0010")
	if p.IsRestructured {
		t.Fatal("tag did NOT reset the restructure flag — a later cut would wrongly charge 50%")
	}
	if !p.IsTagged {
		t.Fatal("0010 not flagged tagged")
	}
}

// TestIntegration_RestructureRejectsTaggedPlayer proves the GLM-M4a guard: a
// franchise-tagged contract is a fixed one-year deal and cannot be restructured.
func TestIntegration_RestructureRejectsTaggedPlayer(t *testing.T) {
	s, c, dir := tagStore(t)
	ctx := context.Background()

	if _, err := c.ExecuteTag(ctx, "0010", dir); err != nil {
		t.Fatalf("tag 0010: %v", err)
	}
	before, _ := s.CapUsed("0001")
	if _, err := c.Execute(ctx, transactions.Restructure{MFLID: "0010", Move: 1 * mil}); err == nil {
		t.Fatal("restructuring a franchise-tagged player was accepted")
	}
	if after, _ := s.CapUsed("0001"); after != before {
		t.Fatalf("rejected restructure of a tagged player moved the cap: %s -> %s", before, after)
	}
}

// TestIntegration_TagDualWritesCellAndHoldsParity proves Ship 2 (2/4): a tag SETS the
// current-season ledger cell to the resolved price (dual-write alongside the legacy column),
// and derived-cap parity stays green because both models took the same figure. The $2M WR
// (0022) tags to the $6M top-5 average, so his 2026 cell becomes $6M while 2027/2028 stay $2M
// (non-conserving — a tag replaces the season salary, it does not move money between years).
func TestIntegration_TagDualWritesCellAndHoldsParity(t *testing.T) {
	s, c, dir := tagStore(t)
	ctx := context.Background()

	if err := s.CheckLedgerParity(ctx); err != nil {
		t.Fatalf("pre-tag parity should hold: %v", err)
	}
	if _, err := c.ExecuteTag(ctx, "0022", dir); err != nil {
		t.Fatalf("ExecuteTag: %v", err)
	}

	cells, err := s.LedgerCells(ctx, "0022")
	if err != nil {
		t.Fatalf("LedgerCells: %v", err)
	}
	if cells[2026] != 6*mil {
		t.Fatalf("2026 cell = %s, want $6M (tag price)", cells[2026])
	}
	for _, y := range []int{2027, 2028} {
		if cells[y] != 2*mil {
			t.Fatalf("cell %d = %s, want $2M (untouched — tag is non-conserving)", y, cells[y])
		}
	}
	if err := s.CheckLedgerParity(ctx); err != nil {
		t.Fatalf("post-tag parity broke — legacy column and cell disagree on derived cap: %v", err)
	}
}

// TestIntegration_TagThenCutVoidsCellsAndHoldsParity proves the GLM carry-forward from the
// §9 tag review: after a player is tagged (his 2026 cell set to $6M) and then §8-cut, the
// cut VOIDs the tagged player's ledger cells so no orphan $6M cell survives, and derived-cap
// parity holds (both the legacy release and the ledger void drop him from the cap together).
func TestIntegration_TagThenCutVoidsCellsAndHoldsParity(t *testing.T) {
	s, c, dir := tagStore(t)
	ctx := context.Background()

	if _, err := c.ExecuteTag(ctx, "0022", dir); err != nil {
		t.Fatalf("ExecuteTag: %v", err)
	}
	// Sanity: the tag set the season cell to the $6M price before the cut.
	if pre, err := s.LedgerCells(ctx, "0022"); err != nil || pre[2026] != 6*mil {
		t.Fatalf("pre-cut 2026 cell = %s (err %v), want $6M", pre[2026], err)
	}

	if _, err := c.Execute(ctx, transactions.Waiver{MFLID: "0022"}); err != nil {
		t.Fatalf("Execute Waiver: %v", err)
	}

	// Every cap-bearing cell of the cut player is gone (voided, not deleted) — no orphan.
	cells, err := s.LedgerCells(ctx, "0022")
	if err != nil {
		t.Fatalf("LedgerCells after cut: %v", err)
	}
	if len(cells) != 0 {
		t.Fatalf("cut player still has PAID cells %v, want none (all voided)", cells)
	}
	// Parity holds: the legacy release and the ledger void dropped him from the cap together.
	if err := s.CheckLedgerParity(ctx); err != nil {
		t.Fatalf("post-cut parity broke: %v", err)
	}
}

// offGridSeed's five WRs average to an OFF-$10k-grid price: (10 + 8 + 6 + 4 + 2.005)/5 =
// $6,001,000, whose cents (600_100_000) are NOT a multiple of $10k (1_000_000 cents). The
// $2,005,000 WR (0022) is the tag target.
type offGridSeed struct{ t *testing.T }

func (s offGridSeed) Rosters(context.Context) ([]domain.Roster, error) {
	id := func(raw string) playerid.PlayerID {
		p, err := playerid.New(raw)
		if err != nil {
			s.t.Fatalf("playerid.New(%q): %v", raw, err)
		}
		return p
	}
	mk := func(raw string, sal domain.Money) domain.PlayerRecord {
		return domain.PlayerRecord{MFLID: id(raw), Salary: sal, ContractYear: 2028,
			RosterStatus: domain.RosterActive, ContractStatus: domain.CStatusUFA}
	}
	return []domain.Roster{
		{FranchiseID: "0001", Players: []domain.PlayerRecord{mk("0010", 10*mil), mk("0011", 8*mil)}},
		// 0022 = $2,005,000 (off-$10k-grid on its own, but seeded cells snap to $2,000,000);
		// the tag price it resolves to is the off-grid $6,001,000 average.
		{FranchiseID: "0002", Players: []domain.PlayerRecord{mk("0020", 6*mil), mk("0021", 4*mil), mk("0022", 200_500_000)}},
	}, nil
}

// TestIntegration_TagOffGridPriceHoldsParity is the deferred GLM carry-forward pin: derived-cap
// parity holds because both models snap the SAME per-player figure to $10k (snap(x)==snap(x)),
// NOT because the figures are $10k multiples. Tagging 0022 resolves an off-grid $6,001,000
// price; the tag writes that exact off-grid figure to BOTH the legacy adjusted-salary column
// and the 2026 ledger cell, so the in-tx parity gate must PASS (the WriteTx commits) even though
// neither side is on the $10k grid. If parity were (wrongly) implemented to require or force
// $10k-multiple values, this tag would either be rejected by the gate or drift the two models.
func TestIntegration_TagOffGridPriceHoldsParity(t *testing.T) {
	pools, err := db.Open(context.Background(), filepath.Join(t.TempDir(), "offgrid.db"))
	if err != nil {
		t.Fatalf("db.Open: %v", err)
	}
	t.Cleanup(func() { _ = pools.Close() })
	s := statepkg.New(pools, "14432", 2026)
	if err := s.Initialize(context.Background(), offGridSeed{t}); err != nil {
		t.Fatalf("Initialize: %v", err)
	}
	c, err := transactions.New(s.Writer())
	if err != nil {
		t.Fatalf("New coordinator: %v", err)
	}
	dir := tagDir{"0010": domain.PosWR, "0011": domain.PosWR, "0020": domain.PosWR, "0021": domain.PosWR, "0022": domain.PosWR}
	ctx := context.Background()

	// The tag commits — the in-tx parity gate passed on an off-grid price.
	if _, err := c.ExecuteTag(ctx, "0022", dir); err != nil {
		t.Fatalf("ExecuteTag (off-grid price rejected by the parity gate?): %v", err)
	}

	// The resolved price is genuinely OFF the $10k grid — otherwise this test proves nothing.
	const offGrid = domain.Money(600_100_000) // $6,001,000
	if domain.RoundToNearest10k(offGrid) == offGrid {
		t.Fatalf("test scaffolding broken: %s is already a $10k multiple", offGrid)
	}
	p, _ := s.Player("0022")
	if p.AdjustedSalary != offGrid {
		t.Fatalf("tagged salary = %s, want %s (off-grid top-5 average)", p.AdjustedSalary, offGrid)
	}
	// The ledger cell took the same off-grid figure (dual-write), and parity still holds.
	cells, err := s.LedgerCells(ctx, "0022")
	if err != nil {
		t.Fatalf("LedgerCells: %v", err)
	}
	if cells[2026] != offGrid {
		t.Fatalf("2026 cell = %s, want %s (off-grid tag price)", cells[2026], offGrid)
	}
	if err := s.CheckLedgerParity(ctx); err != nil {
		t.Fatalf("parity broke on an off-grid tag price — it must hold by symmetric snap: %v", err)
	}
}
