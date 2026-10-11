package contractacts

import (
	"fmt"
	"slices"

	"github.com/secureprospective/TheWarRoom/internal/domain"
)

const Million domain.Money = 100_000_000

type Cell struct {
	Season int          `json:"season"`
	Salary domain.Money `json:"salary"`
}

type Terms struct {
	Cells          []Cell       `json:"cells"`
	Price          domain.Money `json:"price"`
	How            string       `json:"how"`
	DeadCapPercent int          `json:"deadCapPercent"`
}

func percent(base domain.Money, rate int) domain.Money {
	return domain.RoundToNearest10k((base*domain.Money(rate) + 50) / 100)
}

// MaxResignYears and MaxExtendYears are the longest re-sign term and the most years one extension adds.
const (
	MaxResignYears = 4
	MaxExtendYears = 3
)

// TenderLevel is one RFA tender level: the draft compensation it names and its floor price.
type TenderLevel struct {
	Round string
	Price domain.Money
}

func TenderLevels() []TenderLevel {
	return []TenderLevel{{"5th", 60 * Million / 100}, {"4th", Million}, {"3rd", 15 * Million / 10}, {"2nd", 25 * Million / 10}, {"1st", 5 * Million}, {"1st + 2nd", 65 * Million / 10}}
}

func Tender(previous domain.Money, level int) (domain.Money, error) {
	levels := TenderLevels()
	if previous < 0 || level < 1 || level > len(levels) {
		return 0, fmt.Errorf("tender: previous salary and level 1–%d required", len(levels))
	}
	return max(levels[level-1].Price, percent(previous, 110)), nil
}

func Resign(tender domain.Money, years int) (domain.Money, error) {
	if tender <= 0 || years < 1 || years > MaxResignYears {
		return 0, fmt.Errorf("re-sign: positive tender and 1–%d years required", MaxResignYears)
	}
	return percent(tender, 100+15*(years-1)), nil
}

// Tag compares the unsnapped average with prior BASE salary before its conditional uplift.
func Tag(salaries []domain.Money, previous, firstTag domain.Money, consecutive int) (domain.Money, error) {
	if previous < 0 || consecutive < 0 || consecutive >= 2 {
		return 0, fmt.Errorf("tag: maximum two consecutive tags")
	}
	if consecutive == 1 {
		if firstTag <= 0 {
			return 0, fmt.Errorf("tag: first tag price missing")
		}
		return percent(firstTag, 120), nil
	}
	if len(salaries) < 5 {
		return 0, fmt.Errorf("tag: five eligible BASE salaries required")
	}
	pool := slices.Clone(salaries)
	slices.SortFunc(pool, func(a, b domain.Money) int {
		if a > b {
			return -1
		}
		if a < b {
			return 1
		}
		return 0
	})
	var sum domain.Money
	for _, v := range pool[:5] {
		if v <= 0 {
			return 0, fmt.Errorf("tag: positive BASE salaries required")
		}
		sum += v
	}
	average := (sum + 2) / 5
	if average < previous {
		return percent(previous, 120), nil
	}
	return domain.RoundToNearest10k(average), nil
}

func PositionFloor(position string) (domain.Money, bool) {
	floors := map[string]int{"QB": 15, "WR": 10, "RB": 8, "TE": 8, "LB": 8, "DE": 7, "S": 5, "DT": 4, "CB": 3, "K": 3}
	floor, ok := floors[position]
	return domain.Money(floor) * Million, ok
}

func Extend(cells []Cell, season, added int, position string, tagPrice domain.Money, extended bool) (Terms, error) {
	floor, ok := PositionFloor(position)
	if !ok || added < 1 || added > MaxExtendYears || len(cells)+added > 6 || extended {
		return Terms{}, fmt.Errorf("extension: known position, 1–3 added years, at most 6 total years and no prior extension required")
	}
	var highest domain.Money
	last := season
	for _, c := range cells {
		if c.Season > season {
			highest = max(highest, c.Salary)
		}
		last = max(last, c.Season)
	}
	if highest <= 0 && tagPrice <= 0 {
		return Terms{}, fmt.Errorf("extension: at least one remaining paid year after this season; UFAs ineligible")
	}
	price := max(percent(highest, 150), floor)
	how := "§10: added years = greater of 150% of highest remaining paid cell and position floor; exact cents then nearest $10,000. Existing cells unchanged."
	if tagPrice > 0 {
		price = percent(tagPrice, 120)
		how = "§9: tagged extension = 120% of tag price per added year, exact cents then nearest $10,000; existing cells unchanged."
	}
	out := slices.Clone(cells)
	for year := last + 1; year <= last+added; year++ {
		out = append(out, Cell{year, price})
	}
	return Terms{Cells: out, Price: price, How: how, DeadCapPercent: 35}, nil
}

func MaxMove(salary domain.Money) domain.Money {
	switch {
	case salary >= 12*Million:
		return 3 * Million
	case salary >= 6*Million:
		return 2 * Million
	case salary >= 3*Million:
		return Million
	default:
		return 0
	}
}

func Restructure(cells []Cell, source, destination int, move domain.Money, alreadyUsed bool) (Terms, error) {
	out := slices.Clone(cells)
	from := slices.IndexFunc(out, func(c Cell) bool { return c.Season == source })
	to := slices.IndexFunc(out, func(c Cell) bool { return c.Season == destination })
	move = domain.RoundToNearest10k(move)
	if alreadyUsed || from < 0 || to < 0 || from == to || move <= 0 {
		return Terms{}, fmt.Errorf("restructure: unused allowance, two distinct existing paid years and positive move required")
	}
	if move > MaxMove(out[from].Salary) || move > out[from].Salary {
		return Terms{}, fmt.Errorf("restructure: move exceeds source-year tier or would make a cell negative")
	}
	out[from].Salary -= move
	out[to].Salary += move
	return Terms{Cells: out, Price: move, How: "§11 and Math Rules §7: owner-selected existing source and destination years; total conserved. Source tiers: $3M → $1M, $6M → $2M, $12M → $3M maximum. Move snaps to $10,000. One per team/year and per contract; each extension unlocks one more. Later waiver charges 50% of remaining paid cells.", DeadCapPercent: 50}, nil
}

func Flat(season, years int, price domain.Money, how string) Terms {
	cells := make([]Cell, years)
	for i := range cells {
		cells[i] = Cell{season + i, price}
	}
	return Terms{Cells: cells, Price: price, How: how, DeadCapPercent: 35}
}

// NextSeason is the first season after the held contract's last cell, and never this season or earlier.
func NextSeason(cells []Cell, season int) int {
	last := season
	for _, c := range cells {
		last = max(last, c.Season)
	}
	return last + 1
}

// Eligible offers only acts whose held contract can take them; history is checked separately.
func Eligible(intent string, status domain.ContractStatus, cells []Cell, season int, position string) bool {
	if len(cells) == 0 {
		return false
	}
	expires := cells[len(cells)-1].Season
	tagged := status == domain.CStatusFT1 || status == domain.CStatusFT2
	switch intent {
	case "rfa.tender", "fa.resign":
		return status == domain.CStatusRFA && expires == season
	case "contract.tag":
		return expires == season
	case "contract.extend":
		_, known := PositionFloor(position)
		return known && (expires > season || tagged) && len(cells) < 6
	case "contract.restructure":
		return !tagged && expires > season && MaxMove(cells[0].Salary) > 0
	default:
		return false
	}
}
