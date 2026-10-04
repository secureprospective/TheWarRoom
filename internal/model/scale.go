package model

import (
	"math"
	"slices"
	"sort"

	"github.com/secureprospective/TheWarRoom/internal/domain"
)

// RegularGames is how many games make a season one of the regulars a position's scale is drawn
// from.
const RegularGames = 4

// Scale maps points per game to a within-position percentile for one season: the share of that
// season's regulars below it, ties counted half, kept inside (0, 1).
type Scale struct {
	sorted []float64
}

// NewScale builds a scale from the regulars' points per game.
func NewScale(ppg []float64) Scale {
	s := slices.Clone(ppg)
	slices.Sort(s)
	return Scale{sorted: s}
}

// Pct is x's percentile. An empty scale or a non-finite x is the midpoint.
func (s Scale) Pct(x float64) float64 {
	n := len(s.sorted)
	if n == 0 || math.IsNaN(x) || math.IsInf(x, 0) {
		return 0.5
	}
	below := sort.SearchFloat64s(s.sorted, x)
	upTo := sort.Search(n, func(i int) bool { return s.sorted[i] > x })
	p := (float64(below) + 0.5*float64(upTo-below)) / float64(n)
	half := 0.5 / float64(n)
	return min(max(p, half), 1-half)
}

// PPG is the points per game at percentile pct: the inverse of Pct, interpolated between the
// regulars. An empty scale gives NaN.
func (s Scale) PPG(pct float64) float64 {
	n := len(s.sorted)
	if n == 0 {
		return math.NaN()
	}
	x := min(max(pct*float64(n)-0.5, 0), float64(n-1))
	i := int(x)
	if i == n-1 {
		return s.sorted[i]
	}
	return s.sorted[i] + (x-float64(i))*(s.sorted[i+1]-s.sorted[i])
}

// Len is the number of regulars the scale was drawn from.
func (s Scale) Len() int { return len(s.sorted) }

// Scales returns each position's scale for each season.
func (d Data) Scales() map[domain.Position]map[int]Scale {
	ppg := map[domain.Position]map[int][]float64{}
	for id, seasons := range d.Seasons {
		pos := d.Players[id].position()
		if pos == "" {
			continue
		}
		for year, s := range seasons {
			if s.Games() < RegularGames {
				continue
			}
			if ppg[pos] == nil {
				ppg[pos] = map[int][]float64{}
			}
			ppg[pos][year] = append(ppg[pos][year], s.PPG())
		}
	}
	out := map[domain.Position]map[int]Scale{}
	for pos, years := range ppg {
		out[pos] = map[int]Scale{}
		for year, v := range years {
			out[pos][year] = NewScale(v)
		}
	}
	return out
}

// position is the player's position, "" for an unknown player.
func (p *Player) position() domain.Position {
	if p == nil {
		return ""
	}
	return p.Position
}
