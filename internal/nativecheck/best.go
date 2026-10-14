package nativecheck

import (
	"cmp"
	"slices"
	"strings"

	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
)

// BestLineup is the highest-scoring lineup that keeps to the league's position and group limits.
type BestLineup struct {
	Starters []Starter
	Points   leaguerules.Decimal
	Found    bool
	Words    string
}

// slot is one limited position: the players who could fill it, best first, and how many may start.
type slot struct {
	position leaguerules.Position
	side     leaguerules.LineupGroup
	players  []Starter
	prefix   []leaguerules.Decimal
	low      int
	high     int
}

// counts is how many starters a lineup holds on offense, on defense and at the kicker.
type counts struct{ offense, defense, other int }

func (c counts) total() int { return c.offense + c.defense + c.other }

func (c counts) add(side leaguerules.LineupGroup, n int) counts {
	switch side {
	case leaguerules.Offense:
		c.offense += n
	case leaguerules.Defense:
		c.defense += n
	case leaguerules.Total:
		c.other += n
	}
	return c
}

// SelectBestLineup picks the starters that score the most while meeting the same limits JudgeLineup
// checks. For a fixed number of starters at each position the best are the top scorers there, so the
// search is over how many start at each position, and a lineup that cannot be built says so.
func SelectBestLineup(rules leaguerules.LineupRules, candidates []Starter) BestLineup {
	if rules.Positions.State != leaguerules.IsKnown {
		return BestLineup{Words: "Needs a ruling: starting position limits"}
	}
	slots := buildSlots(rules.Positions.Value, candidates)
	groups := map[leaguerules.LineupGroup]leaguerules.Bounds{}
	if rules.Groups.State == leaguerules.IsKnown {
		groups = rules.Groups.Value
	}
	layers := searchLayers(slots, groups)
	end, found := bestEnd(layers[len(slots)], groups)
	if !found {
		return BestLineup{Words: "No lineup can meet this league's limits with these players"}
	}
	best := BestLineup{Found: true, Words: "Best lineup under this league's limits", Starters: []Starter{}}
	for i := len(slots) - 1; i >= 0; i-- {
		k := layers[i+1].choice[end]
		end = end.add(slots[i].side, -k)
		best.Points += slots[i].prefix[k]
		best.Starters = append(slices.Clone(slots[i].players[:k]), best.Starters...)
	}
	return best
}

func buildSlots(limits map[leaguerules.Position]leaguerules.Bounds, candidates []Starter) []slot {
	slots := []slot{}
	for _, position := range sortedPositions(limits) {
		s := slot{position: position, side: sideOf(position)}
		for _, c := range candidates {
			if c.Position == position {
				s.players = append(s.players, c)
			}
		}
		slices.SortFunc(s.players, func(a, b Starter) int {
			if a.Points != b.Points {
				return cmp.Compare(b.Points, a.Points)
			}
			return strings.Compare(a.ID, b.ID)
		})
		s.prefix = make([]leaguerules.Decimal, len(s.players)+1)
		for i, p := range s.players {
			s.prefix[i+1] = s.prefix[i] + p.Points
		}
		bounds := limits[position]
		s.low, s.high = 0, len(s.players)
		if bounds.Min.State == leaguerules.IsKnown {
			s.low = bounds.Min.Value
		}
		if bounds.Max.State == leaguerules.IsKnown {
			s.high = min(s.high, bounds.Max.Value)
		}
		slots = append(slots, s)
	}
	return slots
}

// layer holds, for every reachable (offense, defense, other) count after some positions, the best
// points and how many started at the last position to get there.
type layer struct {
	points map[counts]leaguerules.Decimal
	choice map[counts]int
}

// searchLayers builds the layers one position at a time, dropping any count already over a group's
// maximum. Counts are visited in a fixed order so ties always resolve the same way.
func searchLayers(slots []slot, groups map[leaguerules.LineupGroup]leaguerules.Bounds) []layer {
	layers := []layer{{points: map[counts]leaguerules.Decimal{{}: 0}, choice: map[counts]int{}}}
	for _, s := range slots {
		prev, next := layers[len(layers)-1], layer{points: map[counts]leaguerules.Decimal{}, choice: map[counts]int{}}
		for _, from := range orderedCounts(prev.points) {
			for k := s.low; k <= s.high; k++ {
				to := from.add(s.side, k)
				if overMaximum(to, groups) {
					break
				}
				if score := prev.points[from] + s.prefix[k]; !hasKey(next.points, to) || score > next.points[to] {
					next.points[to], next.choice[to] = score, k
				}
			}
		}
		layers = append(layers, next)
	}
	return layers
}

func hasKey(m map[counts]leaguerules.Decimal, c counts) bool {
	_, ok := m[c]
	return ok
}

func orderedCounts(m map[counts]leaguerules.Decimal) []counts {
	keys := make([]counts, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	slices.SortFunc(keys, func(a, b counts) int {
		if a.offense != b.offense {
			return a.offense - b.offense
		}
		if a.defense != b.defense {
			return a.defense - b.defense
		}
		return a.other - b.other
	})
	return keys
}

// overMaximum is true when any group already has more starters than its maximum allows.
func overMaximum(c counts, groups map[leaguerules.LineupGroup]leaguerules.Bounds) bool {
	for group, n := range map[leaguerules.LineupGroup]int{leaguerules.Offense: c.offense,
		leaguerules.Defense: c.defense, leaguerules.Total: c.total()} {
		if b, ok := groups[group]; ok && b.Max.State == leaguerules.IsKnown && n > b.Max.Value {
			return true
		}
	}
	return false
}

// bestEnd picks the best final count that also meets every group's minimum.
func bestEnd(final layer, groups map[leaguerules.LineupGroup]leaguerules.Bounds) (counts, bool) {
	var best counts
	var found bool
	for _, c := range orderedCounts(final.points) {
		if underMinimum(c, groups) {
			continue
		}
		if !found || final.points[c] > final.points[best] {
			best, found = c, true
		}
	}
	return best, found
}

func underMinimum(c counts, groups map[leaguerules.LineupGroup]leaguerules.Bounds) bool {
	for group, n := range map[leaguerules.LineupGroup]int{leaguerules.Offense: c.offense,
		leaguerules.Defense: c.defense, leaguerules.Total: c.total()} {
		if b, ok := groups[group]; ok && b.Min.State == leaguerules.IsKnown && n < b.Min.Value {
			return true
		}
	}
	return false
}
