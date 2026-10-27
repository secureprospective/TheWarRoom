package nativecheck

import (
	"fmt"
	"slices"
	"strconv"

	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
)

// LineCardTeams is the fixed nflverse team list that numbers the O-Line cards (DEFAULT-32 spec §3).
// It is append-only: a card keeps its id by keeping its place, so a relocated team keeps its place.
func LineCardTeams() []string {
	return []string{"ARI", "ATL", "BAL", "BUF", "CAR", "CHI", "CIN", "CLE", "DAL", "DEN", "DET",
		"GB", "HOU", "IND", "JAX", "KC", "LA", "LAC", "LV", "MIA", "MIN", "NE", "NO", "NYG", "NYJ", "PHI", "PIT",
		"SEA", "SF", "TB", "TEN", "WAS"}
}

// lineCardBlock is the id block of the cards, 9900NN; no MFL player id uses it.
const lineCardBlock = 990000

// LineCardID is the player id of an NFL team's O-Line card.
func LineCardID(team string) (string, bool) {
	i := slices.Index(LineCardTeams(), team)
	if i < 0 {
		return "", false
	}
	return strconv.Itoa(lineCardBlock + i + 1), true
}

// LineCardTeam is the NFL team whose line a card id stands for; ok is false for every other id.
func LineCardTeam(id string) (string, bool) {
	n, err := strconv.Atoi(id)
	if err != nil || strconv.Itoa(n) != id || n <= lineCardBlock || n > lineCardBlock+len(LineCardTeams()) {
		return "", false
	}
	return LineCardTeams()[n-lineCardBlock-1], true
}

// LineCardFacts are a card's facts: they follow from its id, so no player feed has to hold them.
func LineCardFacts(id string) (PlayerFacts, bool) {
	team, ok := LineCardTeam(id)
	if !ok {
		return PlayerFacts{}, false
	}
	return PlayerFacts{Name: team + " O-Line", Position: leaguerules.OL, Team: team}, true
}

// Of reads a player's facts; an O-Line card's facts are its own.
func (f Facts) Of(id string) (PlayerFacts, bool) {
	if card, ok := LineCardFacts(id); ok {
		return card, true
	}
	p, ok := f[id]
	return p, ok
}

// AddLineCards puts every O-Line card into a facts map, so a reading that looks a player up by id names,
// places and teams a card like any player (the cards join the player pool).
func AddLineCards(f Facts) {
	for _, team := range LineCardTeams() {
		id, _ := LineCardID(team)
		f[id], _ = LineCardFacts(id)
	}
}

// LineCardsHeld checks the one-card rule on everything a franchise holds, reserve included: a league
// that bounds OL holds each franchise to exactly one card, whatever other gate is turned off.
func LineCardsHeld(rules leaguerules.RosterRules, players []RosterPlayer) error {
	if rules.PositionBounds.State != leaguerules.IsKnown {
		return nil
	}
	if _, bounded := rules.PositionBounds.Value[leaguerules.OL]; !bounded {
		return nil
	}
	cards := 0
	for _, p := range players {
		if _, ok := LineCardTeam(p.ID); ok {
			cards++
		}
	}
	if cards != 1 {
		return fmt.Errorf("a franchise holds exactly one O-Line card; this move leaves it with %d", cards)
	}
	return nil
}
