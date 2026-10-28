package leaguerules

import (
	"fmt"
	"maps"
	"slices"
)

const (
	teamsPath     = "league.teams"
	divisionsPath = "league.divisions"
)

// EvenDivisions splits the franchises, in order, into count divisions named "Division 1"... The first
// divisions take one more team when the split is uneven.
func EvenDivisions(franchises []string, count int) map[string]string {
	out := make(map[string]string, len(franchises))
	if count < 1 || count > len(franchises) {
		return out
	}
	size, extra := len(franchises)/count, len(franchises)%count
	at := 0
	for d := range count {
		n := size
		if d < extra {
			n++
		}
		for _, id := range franchises[at : at+n] {
			out[id] = fmt.Sprintf("Division %d", d+1)
		}
		at += n
	}
	return out
}

// DivisionCount is the number of distinct named divisions in an alignment.
func DivisionCount(alignment map[string]string) int {
	names := map[string]bool{}
	for _, name := range alignment {
		if name != "" {
			names[name] = true
		}
	}
	return len(names)
}

// alignTo keeps the division alignment to a Teams or Divisions change's franchises. A Teams change drops
// the departed franchises; a Divisions change that differs from the alignment's count divides the
// franchises evenly. Either way a conference entry for a division that no longer exists is dropped.
func alignTo(r *RuleSet, c Change) {
	if c.Franchises == nil || (c.Field != teamsPath && c.Field != divisionsPath) {
		return
	}
	alignment := &r.League.Alignment
	if alignment.State == IsKnown {
		kept := maps.Clone(alignment.Value)
		maps.DeleteFunc(kept, func(id, _ string) bool { return !slices.Contains(c.Franchises, id) })
		if len(kept) != len(alignment.Value) {
			alignment.Value = kept
		}
	}
	divisions := r.League.Divisions
	if c.Field == divisionsPath && divisions.State == IsKnown &&
		(alignment.State != IsKnown || DivisionCount(alignment.Value) != divisions.Value) {
		*alignment = cited(EvenDivisions(c.Franchises, divisions.Value),
			fmt.Sprintf("Divided evenly when Divisions was set to %d", divisions.Value))
	}
	conferences := &r.League.DivisionConferences
	if conferences.State != IsKnown || alignment.State != IsKnown {
		return
	}
	named := map[string]bool{}
	for _, name := range alignment.Value {
		named[name] = true
	}
	kept := maps.Clone(conferences.Value)
	maps.DeleteFunc(kept, func(division, _ string) bool { return !named[division] })
	if len(kept) != len(conferences.Value) {
		conferences.Value = kept
	}
}

// PoolsPerSide says the league holds a copy of every player for each side.
func PoolsPerSide(r RuleSet) bool {
	return r.League.Pools.State == IsKnown && r.League.Pools.Value == PerSide
}

// CountedSides is each franchise's side when the league's player pools are per side: the conference of
// its division. A franchise with no division, or a division with no conference, has no side yet.
func CountedSides(r RuleSet) map[string]string {
	out := map[string]string{}
	if r.League.Alignment.State != IsKnown || r.League.DivisionConferences.State != IsKnown {
		return out
	}
	for franchise, division := range r.League.Alignment.Value {
		if side := r.League.DivisionConferences.Value[division]; side != "" {
			out[franchise] = side
		}
	}
	return out
}
