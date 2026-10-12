package race

import (
	"fmt"
	"slices"
)

// Seed is a playoff place if the season ended today.
type Seed struct {
	Number         int
	Franchise      string
	DivisionWinner bool
}

// Field is one conference's playoff picture: the division leaders, the rest in wild-card order,
// and the seeds they earn. Winners and Wild carry the ties their order broke.
type Field struct {
	Conference string
	Seeds      []Seed
	Winners    Ordering
	Wild       Ordering
}

// Division ranks one division's franchises.
func (t Table) Division(id string) Ordering {
	members := []string{}
	for _, f := range t.ids {
		if t.lines[f].Division == id {
			members = append(members, f)
		}
	}
	return t.Order(members)
}

// Seeding seeds a conference's bracket of teams: every division leader first, in the league's
// order, then the best of the rest as wild cards.
func (t Table) Seeding(conference string, divisions []string, teams int) (Field, error) {
	if len(divisions) > teams {
		return Field{}, fmt.Errorf("race: %d division leaders do not fit a bracket of %d", len(divisions), teams)
	}
	leaders := []string{}
	for _, d := range divisions {
		if order := t.Division(d).Order; len(order) > 0 {
			leaders = append(leaders, order[0])
		}
	}
	rest := []string{}
	for _, f := range t.ids {
		if t.lines[f].Conference == conference && !slices.Contains(leaders, f) {
			rest = append(rest, f)
		}
	}
	field := Field{Conference: conference, Winners: t.Order(leaders), Wild: t.Order(rest), Seeds: []Seed{}}
	for _, f := range field.Winners.Order {
		field.Seeds = append(field.Seeds, Seed{Number: len(field.Seeds) + 1, Franchise: f, DivisionWinner: true})
	}
	for _, f := range field.Wild.Order {
		if len(field.Seeds) == teams {
			break
		}
		field.Seeds = append(field.Seeds, Seed{Number: len(field.Seeds) + 1, Franchise: f})
	}
	return field, nil
}

// FirstRound is a bracket's opening games by seed: the top seeds sit out until the field is a
// power of two, and the rest meet highest against lowest.
func FirstRound(teams int) (byes int, games [][2]int) {
	size := 1
	for size < teams {
		size *= 2
	}
	byes = size - teams
	for hi, lo := byes+1, teams; hi < lo; hi, lo = hi+1, lo-1 {
		games = append(games, [2]int{hi, lo})
	}
	return byes, games
}

// Contender is a franchise in reach of a conference's last seed.
type Contender struct {
	Franchise string
	Seed      int // the seed held today, 0 when outside
	// HalfGamesBack trails the last seed's holder, in half games; 0 for a team holding a wild card.
	HalfGamesBack int
}

// Bubble lists the wild-card holders and every franchise outside the field that can still catch
// the last seed: one that trails by no more games than it has left to play.
func (t Table) Bubble(f Field) []Contender {
	if len(f.Seeds) == 0 {
		return []Contender{}
	}
	last := t.lines[f.Seeds[len(f.Seeds)-1].Franchise].Overall
	seeded := map[string]int{}
	for _, s := range f.Seeds {
		seeded[s.Franchise] = s.Number
	}
	out := []Contender{}
	for _, id := range f.Wild.Order {
		l := t.lines[id]
		if n := seeded[id]; n > 0 {
			out = append(out, Contender{Franchise: id, Seed: n})
			continue
		}
		if back := HalfGamesBack(last, l.Overall); back <= 2*len(l.Remaining) {
			out = append(out, Contender{Franchise: id, HalfGamesBack: back})
		}
	}
	return out
}
