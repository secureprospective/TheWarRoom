// Package pickvalue estimates rookie picks in dynasty points per game from league draft slots
// and recent players' NFL draft capital. Missing drafted records count as zero; a running minimum
// makes the slot curve non-increasing. Future drafts use their own per-year discount.
package pickvalue

import (
	"bytes"
	_ "embed"
	"encoding/csv"
	"fmt"
	"math"
	"slices"
	"strconv"

	"github.com/secureprospective/TheWarRoom/internal/playerrecord"
)

// Version names the method; a change to the chain or its data is a new version.
const Version = "Pick Value v2"

// Classes is how many NFL classes, the current one included, a slot's record values come from.
const Classes = 4

// window is how many selections either side of a slot its band is drawn from; minPicks the
// fewest history picks a band is drawn from (a slot past the league's history widens it).
const (
	window   = 2
	minPicks = 20
)

// curveLength is how many selections the fitted curve holds: 512 is sixteen rounds of a 32-team
// league. A selection past it takes the last value, which is the curve's floor.
const curveLength = 512

//go:embed slots_v1.csv
var slotsV1 []byte

// Pick is one rookie-draft selection in the league's history and the NFL capital of the player
// taken: his overall NFL pick, 0 when he went undrafted.
type Pick struct {
	Year, Round, Slot, Selection, NFLOverall int
}

// History is the league's rookie drafts Pick Value was built on.
func History() ([]Pick, error) {
	rows, err := csv.NewReader(bytes.NewReader(slotsV1)).ReadAll()
	if err != nil || len(rows) < 2 {
		return nil, fmt.Errorf("pickvalue: read slots_v1.csv: %w", err)
	}
	out := make([]Pick, 0, len(rows)-1)
	for _, row := range rows[1:] {
		var n [5]int
		for i, cell := range row {
			if n[i], err = strconv.Atoi(cell); err != nil || n[i] < 0 {
				return nil, fmt.Errorf("pickvalue: slots_v1.csv row %v: bad number %q", row, cell)
			}
		}
		out = append(out, Pick{Year: n[0], Round: n[1], Slot: n[2], Selection: n[3], NFLOverall: n[4]})
	}
	return out, nil
}

// Player is one player's record and native value; Valued is false when the run did not value
// him.
type Player struct {
	Draft  playerrecord.Draft
	Value  float64
	Valued bool
}

// Model is Pick Value fitted on one run's values.
type Model struct {
	season    int
	discount  float64
	history   []Pick
	drafted   []capital // the recent classes' drafted players, by overall pick
	classes   int       // number of NFL classes, including missing records
	undrafted float64   // their undrafted players' mean value
	curve     []Value   // selection 1..curveLength at yearsOut 0, non-increasing in PPG
}

type capital struct {
	overall int
	value   float64
}

// Value is a pick's value and how it was reached.
type Value struct {
	PPG       float64 // dynasty points per game, discounted for years out
	BandLow   float64 // the band's overall NFL picks
	BandHigh  float64
	Undrafted float64 // share of the history picks at the slot who went undrafted
	Players   int     // drafted players in the band a record was found for
	Expected  int     // NFL picks the band holds across the classes fitted: the denominator
	// Clamped is true when the band's own mean rose above an earlier selection's and the curve
	// held this slot down to it. The band columns still describe this slot's own band.
	Clamped  bool
	YearsOut int
}

// Fit fits the model: the players of the Classes NFL classes up to season, and the discount
// applied to a pick for each draft it is out. A discount of 0 makes a future pick worth nothing.
func Fit(history []Pick, players []Player, season int, discount float64) (Model, error) {
	if len(history) == 0 || math.IsNaN(discount) || discount < 0 || discount > 1 {
		return Model{}, fmt.Errorf("pickvalue: need draft history and a discount in [0, 1], got %d picks and %v", len(history), discount)
	}
	m := Model{season: season, discount: discount, history: history}
	var undrafted []float64
	for _, p := range players {
		d := p.Draft
		if d.Year < season-Classes+1 || d.Year > season {
			continue
		}
		v := 0.0
		if p.Valued {
			v = p.Value
		}
		switch {
		case d.Status == playerrecord.Drafted && d.Overall > 0:
			m.drafted = append(m.drafted, capital{d.Overall, v})
		case d.Status == playerrecord.Undrafted:
			undrafted = append(undrafted, v)
		}
	}
	if len(m.drafted) == 0 {
		return Model{}, fmt.Errorf("pickvalue: no drafted player in the %d classes to %d", Classes, season)
	}
	slices.SortFunc(m.drafted, func(a, b capital) int { return a.overall - b.overall })
	m.classes, m.undrafted = Classes, mean(undrafted)
	m.curve = m.fitCurve()
	return m, nil
}

// fitCurve values every selection once and holds the curve down to a running minimum, so a later
// pick is never worth more than an earlier one. The raw band means are not monotone: one league's
// drafts overlap heavily at neighbouring slots, and the percentile band itself moves about.
func (m Model) fitCurve() []Value {
	curve := make([]Value, curveLength)
	floor := math.Inf(1)
	for i := range curve {
		v := m.rawSelection(i + 1)
		if v.PPG > floor {
			v.PPG, v.Clamped = floor, true
		}
		floor = v.PPG
		curve[i] = v
	}
	return curve
}

// rawSelection is the n-th selection's band and its mean value, before the curve is made
// non-increasing and before any years-out discount.
func (m Model) rawSelection(n int) Value {
	var caps []float64
	undrafted := 0
	for w := window; len(caps)+undrafted < minPicks && w < 1000; w *= 2 {
		caps, undrafted = caps[:0], 0
		for _, p := range m.history {
			if abs(p.Selection-max(n, 1)) > w {
				continue
			}
			if p.NFLOverall == 0 {
				undrafted++
			} else {
				caps = append(caps, float64(p.NFLOverall))
			}
		}
	}
	slices.Sort(caps)
	v := Value{Undrafted: float64(undrafted) / float64(max(len(caps)+undrafted, 1))}
	if len(caps) > 0 {
		v.BandLow, v.BandHigh = quantile(caps, 0.25), quantile(caps, 0.75)
	}
	var sum float64
	for _, c := range m.drafted {
		if float64(c.overall) >= v.BandLow && float64(c.overall) <= v.BandHigh {
			sum += c.value
			v.Players++
		}
	}
	// Missing records still occupy their NFL draft slots in every fitted class.
	slots := int(math.Floor(v.BandHigh)) - int(math.Ceil(v.BandLow)) + 1
	v.Expected = max(m.classes*max(slots, 0), v.Players)
	drafted := 0.0
	if v.Expected > 0 {
		drafted = sum / float64(v.Expected)
	}
	v.PPG = (1-v.Undrafted)*drafted + v.Undrafted*m.undrafted
	return v
}

// Selection values the n-th selection of a rookie draft held yearsOut drafts from now.
func (m Model) Selection(n, yearsOut int) Value {
	if len(m.curve) == 0 {
		return Value{YearsOut: max(yearsOut, 0)}
	}
	v := m.curve[min(max(n, 1), len(m.curve))-1]
	v.YearsOut = max(yearsOut, 0)
	v.PPG *= math.Pow(m.discount, float64(v.YearsOut))
	return v
}

// Pick values a pick in a draft year: round and slot within it, in a league of teams franchises.
// Slot 0 (not yet known) is the mean over the round's slots.
func (m Model) Pick(year, round, slot, teams int) (Value, error) {
	if round < 1 || slot < 0 || teams < 1 || slot > teams {
		return Value{}, fmt.Errorf("pickvalue: round %d slot %d of %d is no pick", round, slot, teams)
	}
	out := year - m.season
	if slot > 0 {
		return m.Selection((round-1)*teams+slot, out), nil
	}
	var sum Value
	for s := 1; s <= teams; s++ {
		v := m.Selection((round-1)*teams+s, out)
		sum.PPG += v.PPG / float64(teams)
		sum.BandLow += v.BandLow / float64(teams)
		sum.BandHigh += v.BandHigh / float64(teams)
		sum.Undrafted += v.Undrafted / float64(teams)
		sum.Players += v.Players
		sum.Expected += v.Expected
	}
	sum.Players /= teams
	sum.Expected /= teams
	sum.YearsOut = max(out, 0)
	return sum, nil
}

// quantile is the q-th quantile of sorted xs, interpolating between ranks.
func quantile(xs []float64, q float64) float64 {
	pos := q * float64(len(xs)-1)
	lo := int(math.Floor(pos))
	if lo+1 >= len(xs) {
		return xs[len(xs)-1]
	}
	return xs[lo] + (pos-float64(lo))*(xs[lo+1]-xs[lo])
}

func mean(xs []float64) float64 {
	if len(xs) == 0 {
		return 0
	}
	s := 0.0
	for _, x := range xs {
		s += x
	}
	return s / float64(len(xs))
}

func abs(n int) int { return max(n, -n) }
