package state

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/db"
	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/playerid"
)

const (
	testLeague = "14432"
	testSeason = 2026
)

// capUnit is $10,000 in cents — the universal rounding grain (domain.RoundToNearest10k).
// Fixtures use whole multiples of it so each cell's $10k snap is a no-op and the cell-derived
// cap (Ship 3) equals the sum of the seeded salaries, keeping the cap arithmetic legible.
const capUnit domain.Money = 1_000_000

// fakeSource is a controllable seed Source: tests set the rosters (or an error) it
// yields, with no normalize/network dependency.
type fakeSource struct {
	rosters []domain.Roster
	err     error
}

func (f *fakeSource) Rosters(_ context.Context) ([]domain.Roster, error) {
	return f.rosters, f.err
}

func mustID(t *testing.T, raw string) playerid.PlayerID {
	t.Helper()
	id, err := playerid.New(raw)
	if err != nil {
		t.Fatalf("playerid.New(%q): %v", raw, err)
	}
	return id
}

// baseRosters builds two franchises: 0001 with two players, 0002 with one.
func baseRosters(t *testing.T) []domain.Roster {
	t.Helper()
	return []domain.Roster{
		{FranchiseID: "0001", Players: []domain.PlayerRecord{
			{MFLID: mustID(t, "0001"), Salary: 10 * capUnit, ContractYear: 2028,
				RosterStatus: domain.RosterActive, ContractStatus: domain.CStatusUFA},
			{MFLID: mustID(t, "0002"), Salary: 5 * capUnit, ContractYear: 2027,
				RosterStatus: domain.RosterTaxi, ContractStatus: domain.CStatusRFA},
		}},
		{FranchiseID: "0002", Players: []domain.PlayerRecord{
			{MFLID: mustID(t, "0003"), Salary: 7 * capUnit, ContractYear: 2029,
				RosterStatus: domain.RosterActive, ContractStatus: domain.CStatusFT1},
		}},
	}
}

func newStore(t *testing.T, src Source) *Store {
	t.Helper()
	pools, err := db.Open(context.Background(), filepath.Join(t.TempDir(), "state.db"))
	if err != nil {
		t.Fatalf("db.Open: %v", err)
	}
	t.Cleanup(func() { _ = pools.Close() })
	s := New(pools, testLeague, testSeason)
	if err := s.Initialize(context.Background(), src); err != nil {
		t.Fatalf("Initialize: %v", err)
	}
	return s
}

func TestInitializeAndReads(t *testing.T) {
	s := newStore(t, &fakeSource{rosters: baseRosters(t)})

	if got := s.Franchises(); len(got) != 2 || got[0] != "0001" || got[1] != "0002" {
		t.Fatalf("Franchises = %v, want [0001 0002]", got)
	}
	fs, ok := s.FranchiseState("0001")
	if !ok || len(fs.Players) != 2 {
		t.Fatalf("FranchiseState(0001) = %+v ok=%v", fs, ok)
	}
	if fs.CapUsed != 15*capUnit { // cell-derived: 10 + 5 units
		t.Fatalf("CapUsed(0001) = %v, want %v", fs.CapUsed, 15*capUnit)
	}
	p, ok := s.Player("0003")
	if !ok || p.FranchiseID != "0002" || p.ExpirationYear != 2029 {
		t.Fatalf("Player(0003) = %+v ok=%v", p, ok)
	}
}

func TestEmptySeedFailsLoud(t *testing.T) {
	pools, err := db.Open(context.Background(), filepath.Join(t.TempDir(), "s.db"))
	if err != nil {
		t.Fatalf("db.Open: %v", err)
	}
	t.Cleanup(func() { _ = pools.Close() })

	for _, tc := range []struct {
		name string
		src  Source
	}{
		{"nil rosters", &fakeSource{rosters: nil}},
		{"empty rosters", &fakeSource{rosters: []domain.Roster{{FranchiseID: "0001"}}}},
	} {
		s := New(pools, testLeague, testSeason)
		if err := s.Initialize(context.Background(), tc.src); err == nil {
			t.Fatalf("%s: Initialize succeeded, want errEmptySeed", tc.name)
		}
	}
}

// TestReaderCannotMutate is the planted proof of the single-writer boundary: the
// value Reader() returns must NOT be assertable back to Writer, while Writer()
// must be. If a refactor lets a reader reach a mutation, this fails.
func TestReaderCannotMutate(t *testing.T) {
	s := newStore(t, &fakeSource{rosters: baseRosters(t)})

	if _, ok := s.Reader().(Writer); ok {
		t.Fatal("Reader() is assertable to Writer — the single-writer boundary leaks")
	}
	if _, ok := s.Writer().(Reader); !ok {
		t.Fatal("Writer() must also satisfy Reader (B7 reads before it writes)")
	}
}

func TestMovePlayerRecomputesCap(t *testing.T) {
	s := newStore(t, &fakeSource{rosters: baseRosters(t)})
	ctx := context.Background()

	if err := s.MovePlayer(ctx, "0001", "0002"); err != nil {
		t.Fatalf("MovePlayer: %v", err)
	}
	from, _ := s.CapUsed("0001")
	to, _ := s.CapUsed("0002")
	if from != 5*capUnit { // lost the 10-unit player, 5-unit taxi remains
		t.Fatalf("CapUsed(0001) = %v, want %v", from, 5*capUnit)
	}
	if to != 17*capUnit { // 7 + the moved 10
		t.Fatalf("CapUsed(0002) = %v, want %v", to, 17*capUnit)
	}
	p, ok := s.Player("0001")
	if !ok || p.FranchiseID != "0002" {
		t.Fatalf("moved player franchise = %+v ok=%v", p, ok)
	}
}

// TestApplyContractSetsTermsNotCap: after the Ship 3 read-flip, cap usage is DERIVED from the
// ledger cells, so ApplyContract — which writes only the base contract terms + flags — does
// NOT move CapUsed. The cap changes only when a cell op (SetCell/MoveCellMoney/VoidCells) runs.
// This proves the separation the pivot exists to enforce: the buttons carry no money math.
func TestApplyContractSetsTermsNotCap(t *testing.T) {
	s := newStore(t, &fakeSource{rosters: baseRosters(t)})
	ctx := context.Background()

	before, _ := s.CapUsed("0001")
	err := s.ApplyContract(ctx, "0001", ContractChange{
		AnnualSalary: 10 * capUnit, ContractYears: 3,
		ExpirationYear: 2029, ContractStatus: domain.CStatusFT1, IsTagged: true,
	})
	if err != nil {
		t.Fatalf("ApplyContract: %v", err)
	}
	after, _ := s.CapUsed("0001")
	if after != before {
		t.Fatalf("CapUsed changed on ApplyContract: before %v, after %v (cap is cell-derived, unaffected)", before, after)
	}
	p, _ := s.Player("0001")
	if !p.IsTagged || p.ContractYears != 3 || p.ExpirationYear != 2029 {
		t.Fatalf("contract terms not applied: %+v", p)
	}
}

func TestSetRosterStatus(t *testing.T) {
	s := newStore(t, &fakeSource{rosters: baseRosters(t)})
	ctx := context.Background()

	if err := s.SetRosterStatus(ctx, "0001", domain.RosterTaxi); err != nil {
		t.Fatalf("SetRosterStatus: %v", err)
	}
	p, _ := s.Player("0001")
	if p.RosterStatus != domain.RosterTaxi {
		t.Fatalf("RosterStatus = %v, want TAXI_SQUAD", p.RosterStatus)
	}
	if err := s.SetRosterStatus(ctx, "0001", domain.RosterStatus("BOGUS")); err == nil {
		t.Fatal("SetRosterStatus accepted an unknown status")
	}
}

func TestMutationsFailLoudOnUnknownPlayer(t *testing.T) {
	s := newStore(t, &fakeSource{rosters: baseRosters(t)})
	ctx := context.Background()

	if err := s.MovePlayer(ctx, "9999", "0002"); err == nil {
		t.Fatal("MovePlayer accepted an unknown player")
	}
	if err := s.SetRosterStatus(ctx, "9999", domain.RosterActive); err == nil {
		t.Fatal("SetRosterStatus accepted an unknown player")
	}
	if err := s.ApplyContract(ctx, "9999", ContractChange{ContractStatus: domain.CStatusUFA}); err == nil {
		t.Fatal("ApplyContract accepted an unknown player")
	}
}

// TestInitializeDoesNotReseed proves the B3c divergence from rulebook's Reload:
// runtime state is NOT re-pulled. A second Initialize over an existing DB loads the
// stored (already-mutated) state, ignoring a source that now disagrees.
func TestInitializeDoesNotReseed(t *testing.T) {
	dir := t.TempDir()
	pools, err := db.Open(context.Background(), filepath.Join(dir, "state.db"))
	if err != nil {
		t.Fatalf("db.Open: %v", err)
	}
	t.Cleanup(func() { _ = pools.Close() })
	ctx := context.Background()

	s1 := New(pools, testLeague, testSeason)
	if err := s1.Initialize(ctx, &fakeSource{rosters: baseRosters(t)}); err != nil {
		t.Fatalf("first Initialize: %v", err)
	}
	if err := s1.MovePlayer(ctx, "0001", "0002"); err != nil {
		t.Fatalf("MovePlayer: %v", err)
	}

	// A fresh store over the SAME db with a DIFFERENT source must NOT reseed.
	s2 := New(pools, testLeague, testSeason)
	if err := s2.Initialize(ctx, &fakeSource{rosters: []domain.Roster{
		{FranchiseID: "0001", Players: []domain.PlayerRecord{
			{MFLID: mustID(t, "0001"), Salary: 99 * capUnit, RosterStatus: domain.RosterActive,
				ContractStatus: domain.CStatusUFA},
		}},
	}}); err != nil {
		t.Fatalf("second Initialize: %v", err)
	}
	p, ok := s2.Player("0001")
	if !ok || p.FranchiseID != "0002" || p.Salary != 10*capUnit {
		t.Fatalf("state was reseeded: %+v ok=%v (want franchise 0002, salary 10)", p, ok)
	}
}

// TestMutationFailsLoudOnDBDrift plants memory/DB drift: a player's rows are deleted
// out of band while the store's memory still holds him. The mutation's UPDATE then
// matches 0 rows and must fail loud (requireOneRow), not silently no-op.
func TestMutationFailsLoudOnDBDrift(t *testing.T) {
	pools, err := db.Open(context.Background(), filepath.Join(t.TempDir(), "drift.db"))
	if err != nil {
		t.Fatalf("db.Open: %v", err)
	}
	t.Cleanup(func() { _ = pools.Close() })
	ctx := context.Background()

	s := New(pools, testLeague, testSeason)
	if err := s.Initialize(ctx, &fakeSource{rosters: baseRosters(t)}); err != nil {
		t.Fatalf("Initialize: %v", err)
	}
	if _, err := pools.Write().ExecContext(ctx,
		`DELETE FROM rosters WHERE mfl_id = ?`, "0003"); err != nil {
		t.Fatalf("out-of-band delete: %v", err)
	}
	// Memory still holds 0003, so exists() passes; the UPDATE matches 0 rows.
	if err := s.SetRosterStatus(ctx, "0003", domain.RosterActive); err == nil {
		t.Fatal("SetRosterStatus silently no-op'd a 0-row update (memory/DB drift)")
	}
}

// TestLoadFailsLoudOnOrphanRoster plants an orphan roster row (its contract mate
// deleted). The inner join would silently drop it; the reconcile guard must fail.
func TestLoadFailsLoudOnOrphanRoster(t *testing.T) {
	pools, err := db.Open(context.Background(), filepath.Join(t.TempDir(), "orphan.db"))
	if err != nil {
		t.Fatalf("db.Open: %v", err)
	}
	t.Cleanup(func() { _ = pools.Close() })
	ctx := context.Background()

	s := New(pools, testLeague, testSeason)
	if err := s.Initialize(ctx, &fakeSource{rosters: baseRosters(t)}); err != nil {
		t.Fatalf("Initialize: %v", err)
	}
	if _, err := pools.Write().ExecContext(ctx,
		`DELETE FROM contracts WHERE mfl_id = ?`, "0003"); err != nil {
		t.Fatalf("out-of-band delete: %v", err)
	}
	// A fresh load over the orphaned DB must fail, not serve 2 of 3 players.
	s2 := New(pools, testLeague, testSeason)
	if err := s2.Initialize(ctx, &fakeSource{rosters: baseRosters(t)}); err == nil {
		t.Fatal("load served a roster row short of its contract without failing")
	}
}

// TestWriteTxTradeCommitsAtomically runs a multi-step trade in one WriteTx: swap the
// $10 player (0001) and the $7 player (0003) between the two franchises. Both moves
// must land together and the derived cap must reflect the swap on both sides.
func TestWriteTxTradeCommitsAtomically(t *testing.T) {
	s := newStore(t, &fakeSource{rosters: baseRosters(t)})
	ctx := context.Background()

	err := s.WriteTx(ctx, func(w TxWriter) error {
		if err := w.MovePlayer(ctx, "0001", "0002"); err != nil {
			return err
		}
		return w.MovePlayer(ctx, "0003", "0001")
	})
	if err != nil {
		t.Fatalf("WriteTx trade: %v", err)
	}

	p1, _ := s.Player("0001")
	p3, _ := s.Player("0003")
	if p1.FranchiseID != "0002" || p3.FranchiseID != "0001" {
		t.Fatalf("trade did not swap: 0001@%s 0003@%s", p1.FranchiseID, p3.FranchiseID)
	}
	from, _ := s.CapUsed("0001") // lost 10, gained 7, kept the 5 taxi
	to, _ := s.CapUsed("0002")   // lost 7, gained 10
	if from != 12*capUnit || to != 10*capUnit {
		t.Fatalf("cap after trade: 0001=%v (want %v) 0002=%v (want %v)", from, 12*capUnit, to, 10*capUnit)
	}
}

// TestWriteTxRollsBackOnStepError is the atomicity proof: a WriteTx whose FIRST step
// succeeds but whose SECOND step fails must commit NOTHING. After the error, the first
// move must be invisible — no half-applied trade. This is the whole point of the
// spanning-tx surface over chaining self-contained single-op writes.
func TestWriteTxRollsBackOnStepError(t *testing.T) {
	s := newStore(t, &fakeSource{rosters: baseRosters(t)})
	ctx := context.Background()

	err := s.WriteTx(ctx, func(w TxWriter) error {
		if err := w.MovePlayer(ctx, "0001", "0002"); err != nil { // valid, executes in the tx
			return err
		}
		return w.MovePlayer(ctx, "9999", "0001") // unknown player → fails the transaction
	})
	if err == nil {
		t.Fatal("WriteTx succeeded despite an unknown-player step")
	}

	p1, _ := s.Player("0001")
	if p1.FranchiseID != "0001" {
		t.Fatalf("first move was NOT rolled back: 0001@%s (want 0001)", p1.FranchiseID)
	}
	if used, _ := s.CapUsed("0001"); used != 15*capUnit { // untouched: 10 + 5
		t.Fatalf("cap changed after a rolled-back trade: 0001=%v (want %v)", used, 15*capUnit)
	}
}

// TestWriteTxPoisonsOnPostCommitReloadFailure is the GLM-B7a MAJOR gate: when a tx
// COMMITS but the follow-up reload fails, the DB is mutated yet memory is stale. The
// store must (a) surface a distinct error, (b) NOT silently serve the stale view, and
// (c) refuse further writes. We inject the reload failure via the reload seam.
func TestWriteTxPoisonsOnPostCommitReloadFailure(t *testing.T) {
	s := newStore(t, &fakeSource{rosters: baseRosters(t)})
	ctx := context.Background()

	// Force both the reload and its one retry to fail AFTER the commit lands.
	s.reload = func(context.Context) error { return fmt.Errorf("injected reload failure") }

	err := s.MovePlayer(ctx, "0001", "0002")
	if err == nil {
		t.Fatal("MovePlayer returned nil despite a failed post-commit reload")
	}
	if !strings.Contains(err.Error(), "COMMITTED") {
		t.Fatalf("error must flag the committed-but-stale condition, got: %v", err)
	}

	// (a) the DB DID commit — read the row directly, bypassing stale memory.
	var franchise string
	if qerr := s.pools.Read().QueryRowContext(ctx,
		`SELECT franchise_id FROM rosters WHERE mfl_id = ?`, "0001").Scan(&franchise); qerr != nil {
		t.Fatalf("direct DB read: %v", qerr)
	}
	if franchise != "0002" {
		t.Fatalf("DB franchise = %q, want 0002 (the tx should have committed)", franchise)
	}
	// (b) the store is poisoned and (c) refuses further writes rather than serving stale.
	if s.Err() == nil {
		t.Fatal("store not poisoned after a committed-but-stale reload")
	}
	if werr := s.SetRosterStatus(ctx, "0003", domain.RosterActive); werr == nil {
		t.Fatal("a poisoned store accepted a further write")
	}
}

// TestWriteTxRecoversOnReloadRetry proves the transient case: a reload that fails once
// then succeeds is NOT poisoned and the write is visible.
func TestWriteTxRecoversOnReloadRetry(t *testing.T) {
	s := newStore(t, &fakeSource{rosters: baseRosters(t)})
	ctx := context.Background()

	calls := 0
	realLoad := s.load
	s.reload = func(c context.Context) error {
		calls++
		if calls == 1 {
			return fmt.Errorf("transient reload hiccup")
		}
		return realLoad(c)
	}
	if err := s.MovePlayer(ctx, "0001", "0002"); err != nil {
		t.Fatalf("MovePlayer should recover on retry: %v", err)
	}
	if s.Err() != nil {
		t.Fatalf("store poisoned despite a successful retry: %v", s.Err())
	}
	if p, _ := s.Player("0001"); p.FranchiseID != "0002" {
		t.Fatalf("memory not refreshed after retry: 0001@%s", p.FranchiseID)
	}
}

// TestMovePlayerRejectsSelfMove is the GLM-B7a no-silent-no-op gate: moving a player to
// the franchise it already holds must fail, not report a phantom 1-player move.
func TestMovePlayerRejectsSelfMove(t *testing.T) {
	s := newStore(t, &fakeSource{rosters: baseRosters(t)})
	if err := s.MovePlayer(context.Background(), "0001", "0001"); err == nil {
		t.Fatal("MovePlayer accepted a no-op move to the player's own franchise")
	}
}

// TestWriteTxNilFuncFails guards the entry point: a nil transaction function is a
// programmer error, surfaced immediately, not a silent no-op commit.
func TestWriteTxNilFuncFails(t *testing.T) {
	s := newStore(t, &fakeSource{rosters: baseRosters(t)})
	if err := s.WriteTx(context.Background(), nil); err == nil {
		t.Fatal("WriteTx accepted a nil transaction function")
	}
}

func TestReadsDoNotAlias(t *testing.T) {
	s := newStore(t, &fakeSource{rosters: baseRosters(t)})
	players, _ := s.Roster("0001")
	players[0].Salary = 999

	fresh, _ := s.Roster("0001")
	if fresh[0].Salary == 999 {
		t.Fatal("Roster() aliases store memory — a caller mutated internal state")
	}
}

func TestConcurrentReadsAndWrites(t *testing.T) {
	s := newStore(t, &fakeSource{rosters: baseRosters(t)})
	ctx := context.Background()
	var wg sync.WaitGroup

	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 25; j++ {
				_, _ = s.FranchiseState("0001")
				_ = s.Franchises()
				_, _ = s.CapUsed("0002")
			}
		}()
	}
	for i := 0; i < 4; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 25; j++ {
				_ = s.SetRosterStatus(ctx, "0001", domain.RosterActive)
			}
		}()
	}
	wg.Wait()
}
