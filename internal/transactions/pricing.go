package transactions

import (
	"sort"

	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/normalize"
	"github.com/secureprospective/TheWarRoom/internal/store/state"
)

// Directory resolves a rostered mfl id to its players-DB facts (position is the one this
// package needs). normalize.Lookup satisfies it; a fake drops in for tests. This mirrors
// rankings.Directory — the store is deliberately position-blind, so §9's top-5-by-position
// average joins to the players DB here, not in the state store.
type Directory interface {
	Facts(mflID string) (normalize.PlayerFacts, bool)
}

// tagTopN is the §9 pool size: the tag price is the average of the top-N salaries at the
// position, league-wide.
const tagTopN = 5

// tagFloorNum / tagFloorDen express the §9 floor "120% of the player's previous-year
// salary" as an exact integer ratio (no float money).
const (
	tagFloorNum = 120
	tagFloorDen = 100
)

// tagPrice computes the §9 franchise-tag price for a position: the average of the top-5
// (tagTopN) current salaries at that position across every roster, league-wide. It is a
// pure read over authoritative state — the Coordinator calls it BEFORE opening the write
// transaction (the single-writer law means committed state can't shift underneath it), so
// the resulting figure is authoritative and no money crosses the IPC boundary. Salaries are
// exact cents; the average rounds half-up to the cent. Returns 0 only if NO rostered player
// plays the position (impossible for a real tag, since the tagged player himself is one).
func tagPrice(r state.Reader, dir Directory, pos domain.Position) domain.Money {
	var salaries []domain.Money
	for _, fid := range r.Franchises() {
		roster, ok := r.Roster(fid)
		if !ok {
			continue
		}
		for _, ps := range roster {
			facts, ok := dir.Facts(ps.MFLID)
			if !ok || facts.Position != pos {
				continue
			}
			salaries = append(salaries, ps.Salary)
		}
	}
	if len(salaries) == 0 {
		return 0
	}
	// Top-N by salary, descending.
	sort.Slice(salaries, func(i, j int) bool { return salaries[i] > salaries[j] })
	n := tagTopN
	if len(salaries) < n {
		n = len(salaries)
	}
	var sum domain.Money
	for i := 0; i < n; i++ {
		sum += salaries[i]
	}
	// Average, rounded half-up to the cent: (sum + n/2) / n on exact-cents integers.
	return (sum + domain.Money(n)/2) / domain.Money(n)
}

// extMillion is $1,000,000 in exact cents — the unit of the §10 position-floor table.
const extMillion = domain.Money(100_000_000)

// PositionFloor returns the §10 extension salary floor for a position — the minimum an
// extension year may be priced at (the greater of this and 150% of the highest remaining year)
// — and whether the position has a floor at all. It is the rulebook §10 table encoded once, as
// a pure step function. An unclassified position (FLAG) or any position off the table returns
// (0, false) so the Coordinator fails loud rather than extending with no floor.
func PositionFloor(pos domain.Position) (domain.Money, bool) {
	switch pos {
	case domain.PosQB:
		return 15 * extMillion, true
	case domain.PosWR:
		return 10 * extMillion, true
	case domain.PosRB, domain.PosTE, domain.PosLB:
		return 8 * extMillion, true
	case domain.PosDE:
		return 7 * extMillion, true
	case domain.PosS:
		return 5 * extMillion, true
	case domain.PosDT:
		return 4 * extMillion, true
	case domain.PosCB, domain.PosK:
		return 3 * extMillion, true
	case domain.PosFlag:
		return 0, false // unclassified — admin must resolve the position before an extension
	default:
		return 0, false
	}
}

// tagFloorPrice applies the §9 floor: the tag is the greater of the top-5 average and 120%
// of the player's previous-year salary. At tag time (the season boundary) a player's CURRENT
// annual salary IS his previous year's, so priorSalary is ps.Salary — no history store (v1
// decision, Christopher 2026-07-04). The 120% is exact integer ratio math (round half-up).
func tagFloorPrice(topFive, priorSalary domain.Money) domain.Money {
	floor := (priorSalary*tagFloorNum + tagFloorDen/2) / tagFloorDen
	if floor > topFive {
		return floor
	}
	return topFive
}
