// Package lineupdelta says what a trade does to a franchise's best lineup: who enters and leaves
// the starters, who is benched or promoted, and the points per week it adds. It reuses the
// powerrankings lineup fill; it holds no optimizer of its own (MODEL_METHODS_DRAFT §6, D33).
package lineupdelta

import (
	"sort"

	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
	"github.com/secureprospective/TheWarRoom/internal/powerrankings"
)

// Player is one candidate starter: his projected points per game, whether a projection exists,
// and whether he plays in the coming week (false on a bye or when ruled out).
type Player struct {
	ID        string
	Position  string
	PPG       float64
	Projected bool
	Plays     bool
}

// Starter is one player in a best lineup.
type Starter struct {
	Player   string  `json:"player"`
	Position string  `json:"position"`
	PPG      float64 `json:"ppg"`
}

// Change is one starter-list change a trade causes.
type Change struct {
	// Kind is enters (incoming player starts), leaves (outgoing starter), benched (a kept starter
	// loses his place) or promoted (a kept bench player gains one).
	Kind     string `json:"kind"`
	Player   string `json:"player"`
	Position string `json:"position"`
}

// Side is one party's before and after.
type Side struct {
	Franchise string    `json:"franchise"`
	Before    []Starter `json:"before"`
	After     []Starter `json:"after"`
	// PerWeek is the change in the lineup's projected points per game, rest of season.
	PerWeek float64 `json:"perWeek"`
	// ThisWeek is the same change for the coming week only, without players who do not play in it.
	ThisWeek float64  `json:"thisWeek"`
	Changes  []Change `json:"changes"`
	// Unprojected names players counted at zero because no projection exists for them.
	Unprojected []string `json:"unprojected"`
}

// Rules reads the league's starting limits into the lineup fill's form; ok is false when any
// limit it needs is not known.
func Rules(lr leaguerules.LineupRules) (powerrankings.LineupRules, bool) {
	var out powerrankings.LineupRules
	if lr.Positions.State != leaguerules.IsKnown || lr.Groups.State != leaguerules.IsKnown {
		return out, false
	}
	total, ok := lr.Groups.Value[leaguerules.Total]
	if !ok || total.Max.State != leaguerules.IsKnown || total.Max.Value <= 0 {
		return out, false
	}
	out.Total = total.Max.Value
	positions := make([]string, 0, len(lr.Positions.Value))
	for p := range lr.Positions.Value {
		positions = append(positions, string(p))
	}
	sort.Strings(positions)
	defenders := false
	for _, p := range positions {
		b := lr.Positions.Value[leaguerules.Position(p)]
		if b.Min.State != leaguerules.IsKnown || b.Max.State != leaguerules.IsKnown {
			return out, false
		}
		out.Slots = append(out.Slots, powerrankings.Slot{Position: p, Min: b.Min.Value, Max: b.Max.Value})
		defenders = defenders || defensive(p)
	}
	if def, ok := lr.Groups.Value[leaguerules.Defense]; ok && def.Max.State == leaguerules.IsKnown {
		out.Defense = def.Max.Value
	} else if defenders {
		return out, false
	}
	return out, true
}

func defensive(pos string) bool {
	switch pos {
	case "DT", "DE", "LB", "CB", "S":
		return true
	}
	return false
}

// Compute is one franchise's side of a trade: its roster loses give and gains get.
func Compute(franchise string, roster []Player, give []string, get []Player, rules powerrankings.LineupRules) Side {
	gone := map[string]bool{}
	for _, id := range give {
		gone[id] = true
	}
	after := make([]Player, 0, len(roster)+len(get))
	for _, p := range roster {
		if !gone[p.ID] {
			after = append(after, p)
		}
	}
	after = append(after, get...)
	beforeTotal, before := best(roster, rules, false)
	afterTotal, afterStarters := best(after, rules, false)
	beforeWeek, _ := best(roster, rules, true)
	afterWeek, _ := best(after, rules, true)
	out := Side{Franchise: franchise, Before: before, After: afterStarters,
		PerWeek: round(afterTotal - beforeTotal), ThisWeek: round(afterWeek - beforeWeek),
		Changes: changes(before, afterStarters, gone, get), Unprojected: []string{}}
	seen := map[string]bool{}
	for _, p := range append(append([]Player{}, roster...), get...) {
		if !p.Projected && !seen[p.ID] {
			seen[p.ID] = true
			out.Unprojected = append(out.Unprojected, p.ID)
		}
	}
	sort.Strings(out.Unprojected)
	return out
}

// best is the lineup fill over the players; week drops those who do not play this week.
func best(players []Player, rules powerrankings.LineupRules, week bool) (float64, []Starter) {
	pool := make([]Player, 0, len(players))
	for _, p := range players {
		if !week || p.Plays {
			pool = append(pool, p)
		}
	}
	cands := make([]powerrankings.Candidate, len(pool))
	for i, p := range pool {
		cands[i] = powerrankings.Candidate{Position: p.Position, Value: p.PPG}
	}
	total, idx := powerrankings.Lineup(cands, rules)
	starters := make([]Starter, 0, len(idx))
	for _, i := range idx {
		starters = append(starters, Starter{Player: pool[i].ID, Position: pool[i].Position, PPG: pool[i].PPG})
	}
	sort.SliceStable(starters, func(a, b int) bool {
		if starters[a].Position != starters[b].Position {
			return starters[a].Position < starters[b].Position
		}
		return starters[a].PPG > starters[b].PPG
	})
	return total, starters
}

func changes(before, after []Starter, gone map[string]bool, get []Player) []Change {
	in := map[string]Starter{}
	for _, s := range after {
		in[s.Player] = s
	}
	was := map[string]bool{}
	incoming := map[string]bool{}
	for _, p := range get {
		incoming[p.ID] = true
	}
	out := []Change{}
	for _, s := range before {
		was[s.Player] = true
		if _, ok := in[s.Player]; ok {
			continue
		}
		kind := "benched"
		if gone[s.Player] {
			kind = "leaves"
		}
		out = append(out, Change{Kind: kind, Player: s.Player, Position: s.Position})
	}
	for _, s := range after {
		if was[s.Player] {
			continue
		}
		kind := "promoted"
		if incoming[s.Player] {
			kind = "enters"
		}
		out = append(out, Change{Kind: kind, Player: s.Player, Position: s.Position})
	}
	return out
}

func round(v float64) float64 {
	if v < 0 {
		return -float64(int64(-v*10+0.5)) / 10
	}
	return float64(int64(v*10+0.5)) / 10
}
