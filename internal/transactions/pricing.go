package transactions

import (
	"sort"

	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/normalize"
	"github.com/secureprospective/TheWarRoom/internal/store/state"
)

// Directory resolves a rostered player to his players-directory facts (position). The state store
// is position-blind, so §9's by-position average joins here. normalize.Lookup satisfies it.
type Directory interface {
	Facts(mflID string) (normalize.PlayerFacts, bool)
}

// tagTopN is the §9 pool: the tag is the average of the top N salaries at the position.
const tagTopN = 5

// The §9 floor, 120% of last year's salary, as an exact ratio.
const (
	tagFloorNum = 120
	tagFloorDen = 100
)

// tagPrice is the §9 tag price: the average of the top-5 current salaries at the position,
// league-wide, rounded half-up to the cent. It reads committed state before the transaction
// opens; nothing can change it underneath, and no money crosses the IPC boundary.
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
	sort.Slice(salaries, func(i, j int) bool { return salaries[i] > salaries[j] })
	n := tagTopN
	if len(salaries) < n {
		n = len(salaries)
	}
	var sum domain.Money
	for i := 0; i < n; i++ {
		sum += salaries[i]
	}
	// Round half-up on exact cents.
	return (sum + domain.Money(n)/2) / domain.Money(n)
}

// extMillion is $1M in cents, the unit of the §10 floor table.
const extMillion = domain.Money(100_000_000)

// PositionFloor is the §10 extension floor for a position (a year is priced at the greater of
// this and 150% of the top remaining year). An unclassified position has no floor, and the
// Coordinator then refuses the extension.
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
		return 0, false // unclassified: resolve the position first
	default:
		return 0, false
	}
}

// tagFloorPrice is the greater of the top-5 average and 120% of last year's salary. At tag time a
// player's current salary is last year's, so no salary history is needed (Christopher's ruling).
func tagFloorPrice(topFive, priorSalary domain.Money) domain.Money {
	floor := (priorSalary*tagFloorNum + tagFloorDen/2) / tagFloorDen
	if floor > topFive {
		return floor
	}
	return topFive
}
