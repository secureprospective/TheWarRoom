package transactions

import (
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/normalize"
	"github.com/secureprospective/TheWarRoom/internal/store/state"
)

const tm = domain.Money(100_000_000) // $1M in cents

// fakeReader is a minimal state.Reader: only Franchises/Roster feed tagPrice; the rest are
// stubbed so the fake satisfies the interface.
type fakeReader struct {
	rosters map[string][]state.PlayerState
}

func (f fakeReader) Franchises() []string {
	out := make([]string, 0, len(f.rosters))
	for fid := range f.rosters {
		out = append(out, fid)
	}
	return out
}
func (f fakeReader) Roster(fid string) ([]state.PlayerState, bool) {
	r, ok := f.rosters[fid]
	return r, ok
}
func (f fakeReader) Player(mflID string) (state.PlayerState, bool) {
	for _, r := range f.rosters {
		for _, p := range r {
			if p.MFLID == mflID {
				return p, true
			}
		}
	}
	return state.PlayerState{}, false
}
func (f fakeReader) FranchiseState(string) (state.FranchiseState, bool) {
	return state.FranchiseState{}, false
}
func (f fakeReader) CapUsed(string) (domain.Money, bool) { return 0, false }

// fakeDir maps a player id to its position.
type fakeDir map[string]domain.Position

func (d fakeDir) Facts(mflID string) (normalize.PlayerFacts, bool) {
	pos, ok := d[mflID]
	if !ok {
		return normalize.PlayerFacts{}, false
	}
	return normalize.PlayerFacts{Position: pos}, true
}

// TestTagPrice covers the §9 top-5-by-position average: it averages ONLY the target
// position, takes the top 5 when more exist, averages all when fewer than 5 exist, and
// rounds half-up on exact cents.
func TestTagPrice(t *testing.T) {
	// Six WRs (so top-5 drops the cheapest) plus a QB that must be ignored.
	r := fakeReader{rosters: map[string][]state.PlayerState{
		"0001": {
			{MFLID: "w1", Salary: 10 * tm},
			{MFLID: "w2", Salary: 8 * tm},
			{MFLID: "q1", Salary: 30 * tm}, // QB — must NOT count toward the WR tag
		},
		"0002": {
			{MFLID: "w3", Salary: 6 * tm},
			{MFLID: "w4", Salary: 4 * tm},
			{MFLID: "w5", Salary: 2 * tm},
			{MFLID: "w6", Salary: 1 * tm}, // 6th-cheapest WR — dropped by top-5
		},
	}}
	dir := fakeDir{
		"w1": domain.PosWR, "w2": domain.PosWR, "w3": domain.PosWR,
		"w4": domain.PosWR, "w5": domain.PosWR, "w6": domain.PosWR,
		"q1": domain.PosQB,
	}
	// Top-5 WR salaries: 10,8,6,4,2 → sum 30M / 5 = $6M. w6 ($1M) and the $30M QB excluded.
	got := tagPrice(r, dir, domain.PosWR)
	if got != 6*tm {
		t.Fatalf("tagPrice(WR) = %s, want %s", got, 6*tm)
	}

	// Fewer than 5 at a position → average them all. Two TEs at $5M and $2M → $3.5M.
	r2 := fakeReader{rosters: map[string][]state.PlayerState{
		"0001": {{MFLID: "t1", Salary: 5 * tm}, {MFLID: "t2", Salary: 2 * tm}},
	}}
	dir2 := fakeDir{"t1": domain.PosTE, "t2": domain.PosTE}
	got2 := tagPrice(r2, dir2, domain.PosTE)
	if want := domain.Money(350_000_000); got2 != want { // $3.5M
		t.Fatalf("tagPrice(TE, 2 players) = %s, want %s", got2, want)
	}

	// Rounding half-up: three players 1,1,2 cents → sum 4 / 3 = 1.33 → rounds to 1 cent.
	r3 := fakeReader{rosters: map[string][]state.PlayerState{
		"0001": {{MFLID: "a", Salary: 2}, {MFLID: "b", Salary: 1}, {MFLID: "c", Salary: 1}},
	}}
	dir3 := fakeDir{"a": domain.PosLB, "b": domain.PosLB, "c": domain.PosLB}
	if got3 := tagPrice(r3, dir3, domain.PosLB); got3 != 1 {
		t.Fatalf("tagPrice rounding = %d cents, want 1", got3)
	}

	// No player at the position → 0.
	if got4 := tagPrice(r, dir, domain.PosK); got4 != 0 {
		t.Fatalf("tagPrice(no K) = %s, want 0", got4)
	}
}

// TestTagFloorPrice pins the §9 floor: the tag is the greater of the top-5 average and 120%
// of the prior-year salary, exact integer ratio, round half-up.
func TestTagFloorPrice(t *testing.T) {
	// Average $5M dominates a $3M prior (120% = $3.6M) → $5M.
	if got := tagFloorPrice(5*tm, 3*tm); got != 5*tm {
		t.Fatalf("floor (avg wins) = %s, want %s", got, 5*tm)
	}
	// Prior $10M (120% = $12M) beats a $5M average → $12M.
	if got := tagFloorPrice(5*tm, 10*tm); got != 12*tm {
		t.Fatalf("floor (120%% prior wins) = %s, want %s", got, 12*tm)
	}
	// Half-up rounding on the 120%: prior 5 cents → 6.0; prior 7 cents → 8.4 → 8.
	if got := tagFloorPrice(0, 7); got != 8 {
		t.Fatalf("floor rounding = %d cents, want 8", got)
	}
}
