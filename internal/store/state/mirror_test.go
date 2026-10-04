package state

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/db"
	"github.com/secureprospective/TheWarRoom/internal/domain"
)

func mirrorSnap() MirrorSnapshot {
	return MirrorSnapshot{
		Season: 2026,
		Players: []MirrorPlayer{
			{MFLID: "2002", FranchiseID: "0001", RosterStatus: domain.RosterTaxi, Salary: 4 * capUnit, ContractYear: 2027},
			{MFLID: "2001", FranchiseID: "0001", RosterStatus: domain.RosterActive, Salary: 10 * capUnit, ContractYear: 2028},
			{MFLID: "0042", FranchiseID: "0002", RosterStatus: domain.RosterIR, Salary: 6 * capUnit, ContractYear: 2026},
		},
		Adjustments: []Adjustment{
			{ID: "7", FranchiseID: "0002", Amount: 2 * capUnit, Description: "dead cap"},
			{ID: "8", FranchiseID: "0003", Amount: -1 * capUnit, Description: "credit"},
		},
	}
}

func openMirror(t *testing.T, path string) *Mirror {
	t.Helper()
	pools, err := db.Open(context.Background(), path)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = pools.Close() })
	m := NewMirror(pools, fakeDiscounts{taxiPct: 50, irPct: 0})
	if err := m.Initialize(context.Background()); err != nil {
		t.Fatal(err)
	}
	return m
}

func TestMirrorCapFollowsMFLAndRefreshIsIdempotent(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "league.db")
	m := openMirror(t, path)
	if changed, err := m.Replace(ctx, mirrorSnap()); err != nil || !changed {
		t.Fatalf("first Replace = %t, %v", changed, err)
	}
	r := m.Reader()
	// 0001: 10 active + 4 taxi at 50%; 0002: 6 on IR at 0% plus 2 dead; 0003: a 1 credit only.
	for fid, want := range map[string]domain.Money{"0001": 12 * capUnit, "0002": 2 * capUnit, "0003": -1 * capUnit} {
		if got, ok := r.CapUsed(fid); !ok || got != want {
			t.Errorf("CapUsed(%s) = %d, %t; want %d", fid, got, ok, want)
		}
	}
	if dc := m.DeadCap(); len(dc) != 2 || dc["0002"] != 2*capUnit || dc["0003"] != -1*capUnit {
		t.Errorf("DeadCap = %v, want 0002 charged 2 and 0003 credited 1", dc)
	}
	if p, ok := r.Player("0042"); !ok || p.FranchiseID != "0002" || p.CapSalary != 6*capUnit {
		t.Errorf("Player(0042) = %+v, %t", p, ok)
	}
	shuffled := mirrorSnap()
	shuffled.Players[0], shuffled.Players[2] = shuffled.Players[2], shuffled.Players[0]
	if changed, err := m.Replace(ctx, shuffled); err != nil || changed {
		t.Errorf("same league in another order: Replace = %t, %v; want unchanged", changed, err)
	}
	if _, err := m.Replace(ctx, MirrorSnapshot{Season: 2026}); err == nil {
		t.Error("an empty league replaced the mirror")
	}

	reopened := openMirror(t, path)
	if reopened.Season() != 2026 || len(reopened.Reader().Franchises()) != 3 {
		t.Errorf("reopened mirror: season %d, franchises %v", reopened.Season(), reopened.Reader().Franchises())
	}
	rosters, err := reopened.Rosters(ctx)
	if err != nil || len(rosters) != 2 || rosters[0].Players[0].MFLID.String() != "2001" {
		t.Errorf("Rosters = %+v, %v", rosters, err)
	}
}

// A what-if league seeded from the mirror starts with MFL's cap: salaries and salary adjustments,
// so the Transact screens and the boards agree on every franchise.
func TestWhatIfSeededFromMirrorHasTheMirrorsCap(t *testing.T) {
	ctx := context.Background()
	m := openMirror(t, filepath.Join(t.TempDir(), "league.db"))
	if _, err := m.Replace(ctx, mirrorSnap()); err != nil {
		t.Fatal(err)
	}
	whatif := newStoreWithDiscounts(t, m, fakeDiscounts{taxiPct: 50, irPct: 0})
	for _, fid := range m.Reader().Franchises() {
		want, _ := m.Reader().CapUsed(fid)
		if got, ok := whatif.Reader().CapUsed(fid); !ok || got != want {
			t.Errorf("CapUsed(%s): what-if %d (%t), mirror %d", fid, got, ok, want)
		}
	}
}
