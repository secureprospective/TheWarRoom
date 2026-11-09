// Package projection is the D11 native projection range: pure math, no I/O.
package projection

import (
	"math"
	"sort"
)

// Params are the D11 native-projection dials. DefaultParams holds the values the
// replay chose (docs/ui/data-catalog/D11-REPLAY.md); changing one needs a fresh replay.
type Params struct {
	HalfLifeWeeks float64 // this season's weeks lose half their weight every HalfLifeWeeks
	PriorWeight   float64 // flat weight of each week from the two seasons before
	MinScores     float64 // games at which a player's own weeks carry half the blend
	FloorQ, CeilQ float64 // floor and ceiling quantiles
}

// DefaultParams is the replay-proven method default.
func DefaultParams() Params {
	return Params{HalfLifeWeeks: 8, PriorWeight: 1.0 / 3, MinScores: 8, FloorQ: 0.10, CeilQ: 0.90}
}

// WeekScore is one finished week's fantasy points in the league's scoring.
type WeekScore struct {
	Season, Week int
	Points       float64
}

// Range is one player's projected week: floor, median and ceiling, and how many of his own
// weeks fed it. Games 0 means the position's shape alone.
type Range struct {
	Floor, Median, Ceiling float64
	Games                  int
}

type weighted struct{ v, w float64 }

// Shape is a weighted, sorted sample of weeks strictly before a projected week.
type Shape struct {
	samples []weighted
	total   float64
}

// NewShape weights the weeks before (season, week): this season's by recency half-life, the two
// seasons before at the flat prior weight, older ones not at all.
func NewShape(weeks []WeekScore, season, week int, p Params) Shape {
	var s Shape
	for _, ws := range weeks {
		if w := p.weight(ws, season, week); w > 0 {
			s.samples = append(s.samples, weighted{ws.Points, w})
			s.total += w
		}
	}
	sort.Slice(s.samples, func(a, b int) bool { return s.samples[a].v < s.samples[b].v })
	return s
}

// Games is how many weeks the shape holds.
func (s Shape) Games() int { return len(s.samples) }

func (p Params) weight(ws WeekScore, season, week int) float64 {
	switch {
	case ws.Season == season && ws.Week < week:
		return math.Pow(0.5, float64(week-ws.Week)/p.HalfLifeWeeks)
	case ws.Season < season && season-ws.Season <= 2:
		return p.PriorWeight
	}
	return 0
}

// Project blends a player's own shape with his position's: his share is games/(games+MinScores),
// so a player with no history is the position shape. ok is false when neither has any weeks.
func Project(player, position Shape, p Params) (Range, bool) {
	games := float64(player.Games())
	share := games / (games + p.MinScores)
	if player.total == 0 {
		share = 0
	}
	if position.total == 0 {
		if share == 0 {
			return Range{}, false
		}
		share = 1
	}
	q := func(target float64) float64 { return blendQuantile(player, position, share, target) }
	return Range{Floor: q(p.FloorQ), Median: q(0.5), Ceiling: q(p.CeilQ), Games: player.Games()}, true
}

// blendQuantile walks the two sorted shapes together, each scaled to its share of the weight.
func blendQuantile(a, b Shape, share, target float64) float64 {
	wa, wb := 0.0, 0.0
	if a.total > 0 {
		wa = share / a.total
	}
	if b.total > 0 {
		wb = (1 - share) / b.total
	}
	i, j, acc, last := 0, 0, 0.0, 0.0
	for i < len(a.samples) || j < len(b.samples) {
		takeA := j >= len(b.samples) || (i < len(a.samples) && a.samples[i].v <= b.samples[j].v)
		if takeA {
			last, acc = a.samples[i].v, acc+a.samples[i].w*wa
			i++
		} else {
			last, acc = b.samples[j].v, acc+b.samples[j].w*wb
			j++
		}
		if acc >= target-1e-12 {
			return last
		}
	}
	return last
}
