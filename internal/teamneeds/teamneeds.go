// Package teamneeds reads each franchise's needs and surplus against the league (D27): a need is
// a position where the franchise's Best N holds fewer players than the league's typical Best N,
// and surplus is every valued player outside its Best N. Value is the model's dynasty value, so
// a need is about the roster's shape, never about price.
package teamneeds

import (
	"cmp"
	"slices"
)

// Player is one rostered player as the model valued him.
type Player struct {
	ID, Position string
	Value        float64
	Valued       bool
}

// Need is a position the franchise is short at; Grade is "critical" at two or more short,
// otherwise "thin".
type Need struct {
	Position string
	Short    int
	Grade    string
}

// Surplus is a position's valued players outside the Best N, most valuable first.
type Surplus struct {
	Position string
	Players  []string
	Value    float64
}

// Team is one franchise's read.
type Team struct {
	Best     []string // the Best N, most valuable first
	Count    map[string]int
	Needs    []Need
	Surplus  []Surplus
	Unvalued []string // rostered players the model could not value
}

// League is every franchise's read and the template it was read against.
type League struct {
	N        int
	Template map[string]int
	Teams    map[string]Team
}

// Grades.
const (
	Critical = "critical"
	Thin     = "thin"
)

// criticalShort is the shortfall at which a need is critical.
const criticalShort = 2

// Compute reads every roster against the league's Best n. The template at a position is the
// lower median of the franchises' Best-n counts there: with an even league the lower middle
// value, so a need is never manufactured from a split league.
func Compute(rosters map[string][]Player, n int) League {
	out := League{N: n, Template: map[string]int{}, Teams: map[string]Team{}}
	positions := map[string]bool{}
	for fid, roster := range rosters {
		t := best(roster, n)
		out.Teams[fid] = t
		for _, p := range roster {
			if p.Valued {
				positions[p.Position] = true
			}
		}
	}
	for pos := range positions {
		counts := make([]int, 0, len(out.Teams))
		for _, t := range out.Teams {
			counts = append(counts, t.Count[pos])
		}
		slices.Sort(counts)
		if m := counts[(len(counts)-1)/2]; m > 0 {
			out.Template[pos] = m
		}
	}
	for fid, t := range out.Teams {
		for _, pos := range sortedKeys(out.Template) {
			if short := out.Template[pos] - t.Count[pos]; short > 0 {
				grade := Thin
				if short >= criticalShort {
					grade = Critical
				}
				t.Needs = append(t.Needs, Need{Position: pos, Short: short, Grade: grade})
			}
		}
		out.Teams[fid] = t
	}
	return out
}

// best splits one roster into its Best n, its surplus and its unvalued players.
func best(roster []Player, n int) Team {
	valued := make([]Player, 0, len(roster))
	t := Team{Count: map[string]int{}}
	for _, p := range roster {
		if p.Valued {
			valued = append(valued, p)
		} else {
			t.Unvalued = append(t.Unvalued, p.ID)
		}
	}
	slices.SortFunc(valued, func(a, b Player) int {
		return cmp.Or(cmp.Compare(b.Value, a.Value), cmp.Compare(a.ID, b.ID))
	})
	slices.Sort(t.Unvalued)
	surplus := map[string]*Surplus{}
	for i, p := range valued {
		if i < n {
			t.Best = append(t.Best, p.ID)
			t.Count[p.Position]++
			continue
		}
		if p.Value <= 0 {
			continue
		}
		s := surplus[p.Position]
		if s == nil {
			s = &Surplus{Position: p.Position}
			surplus[p.Position] = s
		}
		s.Players = append(s.Players, p.ID)
		s.Value += p.Value
	}
	for _, pos := range sortedKeys(surplus) {
		t.Surplus = append(t.Surplus, *surplus[pos])
	}
	slices.SortStableFunc(t.Surplus, func(a, b Surplus) int { return cmp.Compare(b.Value, a.Value) })
	return t
}

// SurplusAt is the franchise's surplus value at a position.
func (t Team) SurplusAt(pos string) float64 {
	for _, s := range t.Surplus {
		if s.Position == pos {
			return s.Value
		}
	}
	return 0
}

// InBest reports whether the player is in the franchise's Best N.
func (t Team) InBest(id string) bool { return slices.Contains(t.Best, id) }

func sortedKeys[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	slices.Sort(keys)
	return keys
}
