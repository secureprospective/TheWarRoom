// Package pickvalue is Pick Value v1: what a league rookie-draft pick is worth, in the same
// units as a player's native value (dynasty points per game, internal/modelrun), from the records
// of the players typically taken at its slot.
//
// The chain is slot → NFL draft-capital band → record values:
//  1. The league's own rookie drafts (slots_v1.csv) say which NFL draft capital goes at a slot:
//     the overall NFL picks of the players taken within two selections of it, 2017–2026.
//  2. Their middle half (25th–75th percentile) is the slot's band; the share of them who went
//     undrafted is kept beside it.
//  3. The slot's value is the mean native value of the recent NFL classes' players whose record
//     puts them in that band, blended with the undrafted players' mean by that share. A player
//     the run did not value (out of the league) counts as zero: that is what the pick returns.
//  4. A pick in a later draft is discounted by the dynasty discount for each year out.
//
// It is pure: callers hand it the records and their values.
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
const Version = "Pick Value v1"

// Classes is how many NFL classes, the current one included, a slot's record values come from.
const Classes = 4

// window is how many selections either side of a slot its band is drawn from; minPicks the
// fewest history picks a band is drawn from (a slot past the league's history widens it).
const (
	window   = 2
	minPicks = 20
)

//go:embed slots_v1.csv
var slotsV1 []byte

// Pick is one rookie-draft selection in the league's history and the NFL capital of the player
// taken: his overall NFL pick, 0 when he went undrafted.
type Pick struct {
	Year, Round, Slot, Selection, NFLOverall int
}

// History is the league's rookie drafts Pick Value v1 was built on.
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

// Model is Pick Value v1 fitted on one run's values.
type Model struct {
	season    int
	discount  float64
	history   []Pick
	drafted   []capital // the recent classes' drafted players, by overall pick
	undrafted float64   // their undrafted players' mean value
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
	Players   int     // drafted players in the band the value is the mean of
	YearsOut  int
}

// Fit fits the model: the players of the Classes NFL classes up to season, and the dynasty
// discount per year.
func Fit(history []Pick, players []Player, season int, discount float64) (Model, error) {
	if len(history) == 0 || discount <= 0 || discount > 1 {
		return Model{}, fmt.Errorf("pickvalue: need draft history and a discount in (0, 1], got %d picks and %v", len(history), discount)
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
	m.undrafted = mean(undrafted)
	return m, nil
}

// Selection values the n-th selection of a rookie draft held yearsOut drafts from now.
func (m Model) Selection(n, yearsOut int) Value {
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
	v := Value{YearsOut: max(yearsOut, 0), Undrafted: float64(undrafted) / float64(max(len(caps)+undrafted, 1))}
	if len(caps) > 0 {
		v.BandLow, v.BandHigh = quantile(caps, 0.25), quantile(caps, 0.75)
	}
	var in []float64
	for _, c := range m.drafted {
		if float64(c.overall) >= v.BandLow && float64(c.overall) <= v.BandHigh {
			in = append(in, c.value)
		}
	}
	v.Players = len(in)
	v.PPG = ((1-v.Undrafted)*mean(in) + v.Undrafted*m.undrafted) * math.Pow(m.discount, float64(v.YearsOut))
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
	}
	sum.Players /= teams
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
