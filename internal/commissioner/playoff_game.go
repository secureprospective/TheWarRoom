package commissioner

import (
	"fmt"
	"strings"

	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
)

// decide reads a game once both teams are set: each team's complete score for the week, the home team's
// bonus, and the winner. A tie has no winner; only the commissioner can say who advances from it.
func (b *bracketBuilder) decide(g *BracketGame) {
	b.winners[g.ID] = nil
	if g.Home == nil || g.Away == nil {
		g.Status, g.Words = gameWaiting, b.waitingWords(g)
		return
	}
	hp, homeWhy := b.in.points(g.Home.Franchise, g.Week)
	ap, awayWhy := b.in.points(g.Away.Franchise, g.Week)
	g.HomePoints, g.AwayPoints = hp, ap
	var winner *BracketSide
	if hp == nil || ap == nil {
		g.Status, g.Words = GameNoResult, "No result yet: "+strings.TrimSpace(homeWhy+" "+awayWhy)
	} else {
		winner = b.settle(g, *hp, *ap)
	}
	if team, ok := b.in.advance[g.ID]; ok {
		winner = b.advanced(g, team, winner)
	}
	if winner != nil {
		g.Winner = winner.Franchise
		carried := *winner
		carried.ByHand = false
		b.winners[g.ID] = &carried
	}
}

// settle gives the home team its bonus and compares the totals.
func (b *bracketBuilder) settle(g *BracketGame, home, away leaguerules.Decimal) *BracketSide {
	homeTotal, err := addScoreDelta(home, g.HomeBonus)
	if err != nil {
		g.Status, g.Words = GameNoResult, "No result yet: the home field points overflow the score."
		return nil
	}
	g.HomeTotal, g.AwayTotal = &homeTotal, &away
	switch {
	case homeTotal == away:
		g.Status, g.Words = GameTie, fmt.Sprintf("%s and %s tied at %s; the commissioner decides who advances",
			g.Home.Name, g.Away.Name, scorePointsWords(away))
		return nil
	case homeTotal > away:
		g.Status, g.Words = GameFinal, g.finalWords(g.Home, g.Away, homeTotal, away)
		return g.Home
	}
	g.Status, g.Words = GameFinal, g.finalWords(g.Away, g.Home, away, homeTotal)
	return g.Away
}

func (g *BracketGame) finalWords(winner, loser *BracketSide, high, low leaguerules.Decimal) string {
	words := finalWords(winner.Name, loser.Name, high, low)
	if g.HomeBonus != 0 {
		words += fmt.Sprintf(", counting %s home field points for %s", scorePointsWords(g.HomeBonus), g.Home.Name)
	}
	return words
}

// advanced puts the commissioner's choice over the computed result. A team that no longer plays in the
// game cannot advance from it, so the choice is set aside and the game says so.
func (b *bracketBuilder) advanced(g *BracketGame, team string, computed *BracketSide) *BracketSide {
	var chosen *BracketSide
	for _, side := range []*BracketSide{g.Home, g.Away} {
		if side != nil && side.Franchise == team {
			chosen = side
		}
	}
	if chosen == nil {
		g.Words += "; the commissioner's choice of " + b.in.names[team] + " is set aside because that team no longer plays in this game"
		return computed
	}
	g.ByHand = true
	if computed == nil {
		g.Words = fmt.Sprintf("The commissioner advanced %s. %s", chosen.Name, g.Words)
	} else {
		g.Words = fmt.Sprintf("The commissioner advanced %s despite the computed result. %s", chosen.Name, g.Words)
	}
	g.Status = gameByHand
	return chosen
}

func (b *bracketBuilder) waitingWords(g *BracketGame) string {
	switch {
	case g.Home == nil && g.Away == nil:
		return "Waiting for both teams"
	case g.Home == nil:
		return "Waiting for the home team"
	}
	return "Waiting for the away team"
}

// roundName reads a round's name from the game names it holds: wildcard_2 and wildcard_3 are the Wildcard round.
func roundName(nodes []shapeNode) string {
	base := strings.TrimRight(nodes[0].ID, "0123456789")
	if base != nodes[0].ID {
		base = strings.TrimSuffix(base, "_")
	}
	name := strings.ReplaceAll(base, "_", " ")
	if name == "" {
		return nodes[0].ID
	}
	name = strings.ToUpper(name[:1]) + name[1:]
	lower := strings.ToLower(name)
	if strings.HasSuffix(lower, "championship") || strings.HasSuffix(lower, "final") || strings.HasSuffix(lower, "round") {
		return name
	}
	return name + " round"
}
