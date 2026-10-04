package fit

import (
	"math"
	"slices"

	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/model"
)

// season is one player-season at the position being fitted.
type season struct {
	player *model.Player
	s      *model.Season
	year   int
	games  float64
	pct    float64
	age    float64
	exp    int
}

// pair is a season and the player's next one, which may have no games.
type pair struct {
	from season
	next *season
}

// samples is one position's seasons, indexed by player and year.
type samples struct {
	pos     domain.Position
	seasons []season
	byYear  map[string]map[int]season
}

func collect(d model.Data, scales map[domain.Position]map[int]model.Scale, pos domain.Position, first, last int) samples {
	out := samples{pos: pos, byYear: map[string]map[int]season{}}
	for id, years := range d.Seasons {
		p := d.Players[id]
		if p == nil || p.Position != pos {
			continue
		}
		for year, s := range years {
			scale, ok := scales[pos][year]
			if year < first || year > last || s.Games() == 0 || !ok {
				continue
			}
			ss := season{player: p, s: s, year: year, games: float64(s.Games()), pct: scale.Pct(s.PPG()),
				age: p.AgeAt(year), exp: p.Experience(year)}
			out.seasons = append(out.seasons, ss)
			if out.byYear[id] == nil {
				out.byYear[id] = map[int]season{}
			}
			out.byYear[id][year] = ss
		}
	}
	slices.SortFunc(out.seasons, func(a, b season) int {
		if a.player.ID != b.player.ID {
			if a.player.ID < b.player.ID {
				return -1
			}
			return 1
		}
		return a.year - b.year
	})
	return out
}

// pairs returns each season followed by the player's next, for next years up to last.
func (sm samples) pairs(last int) []pair {
	var out []pair
	for _, s := range sm.seasons {
		if s.year+1 > last {
			continue
		}
		p := pair{from: s}
		if n, ok := sm.byYear[s.player.ID][s.year+1]; ok {
			p.next = &n
		}
		out = append(out, p)
	}
	return out
}

// weekly returns a season's weekly points split into odd and even weeks.
func weekly(s *model.Season) (all, odd, even []float64) {
	weeks := make([]int, 0, len(s.Points))
	for w := range s.Points {
		weeks = append(weeks, w)
	}
	slices.Sort(weeks)
	for _, w := range weeks {
		v := s.Points[w]
		all = append(all, v)
		if w%2 == 1 {
			odd = append(odd, v)
		} else {
			even = append(even, v)
		}
	}
	return all, odd, even
}

func avg(v []float64) float64 {
	s := 0.0
	for _, x := range v {
		s += x
	}
	return s / float64(len(v))
}

func finite(v ...float64) bool {
	for _, x := range v {
		if math.IsNaN(x) || math.IsInf(x, 0) {
			return false
		}
	}
	return true
}

// rmse is the root mean squared error of predictions against targets.
func rmse(pred, target []float64) float64 {
	if len(pred) == 0 {
		return math.NaN()
	}
	s := 0.0
	for i := range pred {
		s += (pred[i] - target[i]) * (pred[i] - target[i])
	}
	return math.Sqrt(s / float64(len(pred)))
}
