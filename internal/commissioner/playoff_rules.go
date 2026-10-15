package commissioner

import (
	"fmt"
	"slices"
	"strings"

	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
)

// Words a bracket slot may use to say which team plays there instead of naming a seed. The seed words
// pair by the seeds left, which is reseeding; the arrival words take the next team to reach the round, the
// best seed when the round is reseeded and the earlier game's winner when it is not.
const (
	fromLowest   = "lowest remaining seed"
	fromHighest  = "highest remaining seed"
	fromOther    = "other remaining seed"
	fromDivision = "division winner"
	fromConf     = "conference winner"
	fromWinnerOf = "winner of "
)

func (s shapeSlot) seedBased() bool {
	return slices.Contains([]string{fromLowest, fromHighest, fromOther}, s.From)
}

// shapeSlot is where one side of a game comes from: a fixed seed, or a rule for who is left.
type shapeSlot struct {
	Seed int
	From string
}

// shapeNode is one game of the bracket's shape. A conference's games are repeated in each conference,
// except a game that takes conference winners, which is played once.
type shapeNode struct {
	ID         string
	Week       int
	Home, Away shapeSlot
}

func (n shapeNode) leagueWide() bool {
	return n.Home.From == fromConf || n.Away.From == fromConf
}

// PlayoffRules are the playoff rules a season runs under, resolved from the league's rules and a
// bracket shape. Words says what the commissioner still has to settle; it is empty when the rules are whole.
type PlayoffRules struct {
	PerConference   int
	DivisionWinners int
	WildCards       int
	ByeSeeds        int
	Reseed          bool
	StartWeek       int
	FinalWeek       int
	HomeBonus       leaguerules.Decimal
	Shape           []shapeNode
	Words           string
}

func needsRuling(what string) PlayoffRules { return PlayoffRules{Words: "Needs a ruling: " + what} }

// ResolvePlayoffRules reads the season's playoff rules. The shape comes from graph, which is the league's own
// bracket unless a saved or built-in preset replaces it.
func ResolvePlayoffRules(rules leaguerules.RuleSet, graph []leaguerules.BracketNode) PlayoffRules {
	p := rules.Playoffs
	known := []struct {
		what string
		ok   bool
	}{
		{"how many teams per conference make the playoffs", p.PerConference.State == leaguerules.IsKnown},
		{"how many division winners qualify", p.DivisionWinners.State == leaguerules.IsKnown},
		{"how many wild cards qualify", p.WildCards.State == leaguerules.IsKnown},
		{"how many seeds get a first-round bye", p.ByeSeed.State == leaguerules.IsKnown},
		{"how playoff teams are seeded", p.Seeding.State == leaguerules.IsKnown},
		{"whether each round is reseeded", p.Reseed.State == leaguerules.IsKnown},
		{"the first playoff week", p.StartWeek.State == leaguerules.IsKnown},
		{"the final playoff week", p.FinalWeek.State == leaguerules.IsKnown},
		{"the home field points", p.HomeBonus.State == leaguerules.IsKnown},
	}
	for _, k := range known {
		if !k.ok {
			return needsRuling(k.what)
		}
	}
	out := PlayoffRules{PerConference: p.PerConference.Value, DivisionWinners: p.DivisionWinners.Value,
		WildCards: p.WildCards.Value, ByeSeeds: p.ByeSeed.Value, Reseed: p.Reseed.Value, StartWeek: p.StartWeek.Value,
		FinalWeek: p.FinalWeek.Value, HomeBonus: p.HomeBonus.Value}
	if p.Seeding.Value != leaguerules.DivisionWinnersFirst {
		return needsRuling("how playoff teams are seeded; the app seeds division winners first")
	}
	if out.DivisionWinners+out.WildCards != out.PerConference || out.PerConference < 2 {
		return needsRuling(fmt.Sprintf("the playoff field: %d division winners and %d wild cards do not make %d teams per conference",
			out.DivisionWinners, out.WildCards, out.PerConference))
	}
	if out.ByeSeeds < 0 || out.ByeSeeds >= out.PerConference || out.HomeBonus < 0 {
		return needsRuling("the byes or home field points, which cannot be negative or leave no team to play")
	}
	shape, words := shapeFrom(graph)
	if words != "" {
		return PlayoffRules{Words: words}
	}
	out.Shape = shape
	out.Words = out.checkShape()
	return out
}

// shapeFrom reads a bracket graph into nodes, refusing a slot that names neither a seed nor a source.
func shapeFrom(graph []leaguerules.BracketNode) ([]shapeNode, string) {
	if len(graph) == 0 {
		return nil, "Needs a ruling: the playoff bracket shape"
	}
	nodes := make([]shapeNode, 0, len(graph))
	for _, n := range graph {
		id := strings.TrimSpace(n.ID.Value)
		if n.ID.State != leaguerules.IsKnown || id == "" || strings.Contains(id, "/") ||
			n.Week.State != leaguerules.IsKnown || n.Week.Value < 1 {
			return nil, "Needs a ruling: every bracket game needs a name without a slash and a week"
		}
		home, homeOK := slotOf(n.HomeSeed, n.HomeWinner)
		away, awayOK := slotOf(n.AwaySeed, n.AwayWinner)
		if !homeOK || !awayOK {
			return nil, fmt.Sprintf("Needs a ruling: where each team in %s comes from, a seed or a rule", id)
		}
		nodes = append(nodes, shapeNode{ID: id, Week: n.Week.Value, Home: home, Away: away})
	}
	return nodes, ""
}

func slotOf(seed leaguerules.Value[int], from leaguerules.Value[string]) (shapeSlot, bool) {
	word := strings.ToLower(strings.TrimSpace(from.Value))
	fixed := seed.State == leaguerules.IsKnown && seed.Value > 0
	named := from.State == leaguerules.IsKnown && word != ""
	if fixed == named {
		return shapeSlot{}, false
	}
	if fixed {
		return shapeSlot{Seed: seed.Value}, true
	}
	return shapeSlot{From: word}, true
}

// checkShape says what is wrong with the shape beside the rest of the rules, or nothing.
func (r PlayoffRules) checkShape() string {
	seen := map[string]bool{}
	first, last := r.Shape[0].Week, r.Shape[0].Week
	for _, n := range r.Shape {
		if seen[n.ID] {
			return fmt.Sprintf("Needs a ruling: the bracket names %s twice", n.ID)
		}
		seen[n.ID] = true
		first, last = min(first, n.Week), max(last, n.Week)
		for _, slot := range []shapeSlot{n.Home, n.Away} {
			if words := r.checkSlot(slot, n.ID); words != "" {
				return words
			}
		}
	}
	if first != r.StartWeek || last != r.FinalWeek {
		return fmt.Sprintf("Needs a ruling: the bracket runs from week %d to week %d but the playoff weeks are %d to %d",
			first, last, r.StartWeek, r.FinalWeek)
	}
	if len(r.rounds()[len(r.rounds())-1]) != 1 {
		return "Needs a ruling: the last round of the bracket must be one game, the final"
	}
	return ""
}

func (r PlayoffRules) checkSlot(slot shapeSlot, game string) string {
	switch {
	case slot.Seed > r.PerConference:
		return fmt.Sprintf("Needs a ruling: %s names seed %d but only %d teams per conference qualify", game, slot.Seed,
			r.PerConference)
	case strings.HasPrefix(slot.From, fromWinnerOf):
		return ""
	case slot.From == "" || slot.seedBased() || slot.From == fromDivision || slot.From == fromConf:
		if slot.seedBased() && !r.Reseed {
			return fmt.Sprintf("Needs a ruling: %s pairs teams by the seeds left, which is reseeding; turn reseeding on "+
				"or name the game each team follows", game)
		}
		return ""
	}
	return fmt.Sprintf("Needs a ruling: %s says %q, which the app does not know how to pair", game, slot.From)
}

// rounds groups the shape's games by week, earliest first, keeping the order the shape lists them in.
func (r PlayoffRules) rounds() [][]shapeNode {
	weeks := []int{}
	for _, n := range r.Shape {
		if !slices.Contains(weeks, n.Week) {
			weeks = append(weeks, n.Week)
		}
	}
	slices.Sort(weeks)
	out := make([][]shapeNode, len(weeks))
	for i, week := range weeks {
		for _, n := range r.Shape {
			if n.Week == week {
				out[i] = append(out[i], n)
			}
		}
	}
	return out
}
