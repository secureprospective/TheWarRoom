package competition

import (
	"fmt"
	"slices"
	"strings"
)

// picked is what one place in a game resolved to: a team, or nobody yet; the team the place consumed from
// the round's pool, even if a hand edit then replaced it; or the words saying why the shape cannot be followed.
type picked struct {
	side     *BracketSide
	consumed string
	words    string
}

func (s ShapeSlot) pass() int {
	switch {
	case s.Seed > 0:
		return 0
	case strings.HasPrefix(s.From, fromWinnerOf):
		return 1
	}
	return 2
}

// round pairs one scope's games for a round. Fixed seeds are placed first, then the games a place names,
// and the places that take whoever is left last, so the teams left are exactly those nobody else claimed.
func (b *bracketBuilder) round(index int, scope string, nodes []ShapeNode, used map[string]bool) ([]BracketGame, string) {
	mine := slices.DeleteFunc(slices.Clone(nodes), func(n ShapeNode) bool { return n.LeagueWide() != (scope == "") })
	games := make([]BracketGame, len(mine))
	for i, n := range mine {
		id := GameID(scope, n.ID)
		games[i] = BracketGame{ID: id, Node: n.ID, Conference: scope, Week: n.Week, HomeSlot: id + "/home",
			AwaySlot: id + "/away", HomeBonus: b.in.Rules.HomeBonus}
	}
	for pass := range 3 {
		for i, n := range mine {
			for _, home := range []bool{true, false} {
				slot, slotID, place := n.Home, games[i].HomeSlot, &games[i].Home
				if !home {
					slot, slotID, place = n.Away, games[i].AwaySlot, &games[i].Away
				}
				if slot.pass() != pass {
					continue
				}
				side, words := b.fill(index, scope, slot, slotID, used)
				if words != "" {
					return nil, words
				}
				*place = side
			}
		}
	}
	if words := b.unplaced(index, scope, used, mine[0].Week); words != "" {
		return nil, words
	}
	for i := range games {
		b.decide(&games[i])
	}
	return games, ""
}

// fill resolves one place and applies a hand edit to it. A team may play only once a round.
func (b *bracketBuilder) fill(index int, scope string, slot ShapeSlot, slotID string, used map[string]bool) (*BracketSide, string) {
	p := b.resolve(index, scope, slot, used)
	if p.words != "" {
		return nil, p.words
	}
	if p.consumed != "" {
		used[p.consumed] = true
	}
	team, edited := b.in.Edits[slotID]
	if !edited {
		return p.side, ""
	}
	if used[team] && team != p.consumed {
		return nil, fmt.Sprintf("%s is placed in two games of the same round", b.in.Names[team])
	}
	used[team] = true
	side := b.side(team)
	side.ByHand = true
	return side, ""
}

func (b *bracketBuilder) resolve(index int, scope string, slot ShapeSlot, used map[string]bool) picked {
	switch {
	case slot.Seed > 0:
		return b.seeded(scope, slot.Seed, used)
	case strings.HasPrefix(slot.From, fromWinnerOf):
		return b.followed(scope, strings.TrimSpace(strings.TrimPrefix(slot.From, fromWinnerOf)))
	}
	return b.fromPool(index, scope, slot, used)
}

func (b *bracketBuilder) seeded(scope string, seed int, used map[string]bool) picked {
	team := b.seeds[scope][seed]
	switch {
	case scope == "":
		return picked{words: fmt.Sprintf("The bracket names seed %d in a game that takes conference winners", seed)}
	case team == "":
		return picked{words: fmt.Sprintf("The %s conference has no seed %d", scope, seed)}
	case used[team]:
		return picked{words: fmt.Sprintf("Seed %d of the %s conference is placed twice in the bracket", seed, scope)}
	}
	return picked{side: b.side(team), consumed: team}
}

// followed is the winner of a named game, or nobody while that game is undecided.
func (b *bracketBuilder) followed(scope, target string) picked {
	id := GameID(scope, target)
	if strings.Contains(target, "/") {
		id = target
	}
	if !b.games[id] {
		return picked{words: fmt.Sprintf("The bracket follows the winner of %s, which is not a game in it", target)}
	}
	winner := b.winners[id]
	if winner == nil {
		return picked{}
	}
	carried := *winner
	return picked{side: &carried, consumed: carried.Franchise}
}

// fromPool takes a team from those who reached this round and have no place yet. A reseeded round orders
// them by seed (by league standing across conferences); an unseeded one keeps the order they arrived in.
func (b *bracketBuilder) fromPool(index int, scope string, slot ShapeSlot, used map[string]bool) picked {
	if index == 0 {
		return picked{words: "The first round cannot pair teams by who is left; name a seed in each place"}
	}
	if b.pending[scope] {
		return picked{}
	}
	pool := slices.DeleteFunc(slices.Clone(b.advancers[scope]), func(s *BracketSide) bool { return used[s.Franchise] })
	if len(pool) == 0 {
		return picked{words: "The bracket has no team left for a place that takes whoever is left"}
	}
	if b.in.Rules.Reseed {
		slices.SortStableFunc(pool, func(x, y *BracketSide) int { return b.order(scope, x) - b.order(scope, y) })
	}
	choice := *pool[0]
	if slot.From == fromLowest {
		choice = *pool[len(pool)-1]
	}
	return picked{side: &choice, consumed: choice.Franchise}
}

// order is the number a team is reseeded by: its seed in its conference, or its place in the league's
// standings when conferences meet.
func (b *bracketBuilder) order(scope string, s *BracketSide) int {
	if scope == "" {
		return s.Rank
	}
	return s.Seed
}

// unplaced refuses a shape that leaves a team that reached the round without a game.
func (b *bracketBuilder) unplaced(index int, scope string, used map[string]bool, week int) string {
	if index == 0 || b.pending[scope] {
		return ""
	}
	for _, s := range b.advancers[scope] {
		if !used[s.Franchise] {
			return fmt.Sprintf("The bracket leaves %s without a game in week %d", s.Name, week)
		}
	}
	return ""
}
