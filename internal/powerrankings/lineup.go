package powerrankings

import (
	"math"
	"sort"
)

// Slot is one position's starter bounds from the league's lineup rules: MFL's "1-3" is Min 1,
// Max 3, and "1" is both.
type Slot struct {
	Position string
	Min, Max int
}

// LineupRules are the league's starters: the per-position bounds, how many start in all, and how
// many of those are defenders.
type LineupRules struct {
	Slots   []Slot
	Total   int
	Defense int
}

// Candidate is one rostered player who could start: his position and his value.
type Candidate struct {
	Position string
	Value    float64
}

// defensive reports whether a position fills the defensive starter count.
func defensive(pos string) bool {
	switch pos {
	case "DT", "DE", "LB", "CB", "S":
		return true
	}
	return false
}

// Lineup is the most valuable lineup the rules allow, and the indexes of the candidates it
// starts. Each position first starts its minimum, its best players; the remaining offensive and
// defensive slots then go to the best players left, up to each position's maximum. Taking the
// minimums from the top makes this greedy fill optimal. A position the rules do not list never
// starts.
func Lineup(cands []Candidate, rules LineupRules) (float64, []int) {
	byPos := map[string][]int{}
	for i, c := range cands {
		byPos[c.Position] = append(byPos[c.Position], i)
	}
	for _, idx := range byPos {
		sort.SliceStable(idx, func(a, b int) bool { return cands[idx[a]].Value > cands[idx[b]].Value })
	}
	open := map[bool]int{false: rules.Total - rules.Defense, true: rules.Defense}
	var started, rest []int
	for _, s := range rules.Slots {
		idx := byPos[s.Position]
		n := min(s.Min, len(idx))
		started = append(started, idx[:n]...)
		open[defensive(s.Position)] -= n
		rest = append(rest, idx[n:min(max(s.Max, n), len(idx))]...)
	}
	sort.SliceStable(rest, func(a, b int) bool { return cands[rest[a]].Value > cands[rest[b]].Value })
	for _, i := range rest {
		if d := defensive(cands[i].Position); open[d] > 0 {
			started = append(started, i)
			open[d]--
		}
	}
	var total float64
	for _, i := range started {
		total += cands[i].Value
	}
	sort.Ints(started)
	return total, started
}

// ExpectedPoints is a team's expected score in a game: its lineup's points per game, blended
// with its points per game so far at the automatic roster weight.
func ExpectedPoints(lineup, pointsFor float64, games int) float64 {
	if games <= 0 {
		return lineup
	}
	w := AutoRosterWeight(games)
	return w*lineup + (1-w)*pointsFor/float64(games)
}

// WinProbability is the chance a team expected to score a beats one expected to score b.
func WinProbability(a, b float64) float64 {
	return 0.5 * (1 + math.Erf((a-b)/(GameSpread*math.Sqrt2)))
}

// Game is one remaining matchup between two franchises.
type Game struct {
	Home, Away string
}

// ExpectedWins is each franchise's expected wins over the games, from its expected score;
// a franchise without one is skipped, as are its games.
func ExpectedWins(games []Game, expected map[string]float64) map[string]float64 {
	out := map[string]float64{}
	for _, g := range games {
		a, okA := expected[g.Home]
		b, okB := expected[g.Away]
		if !okA || !okB {
			continue
		}
		p := WinProbability(a, b)
		out[g.Home] += p
		out[g.Away] += 1 - p
	}
	return out
}
