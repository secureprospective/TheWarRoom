package leaguerules

import (
	"fmt"
	"maps"
)

const (
	mirror16Citation = "16-Team Mirror: two sides, each with its own copy of every player, ruling 2026-10-27"
	mirror16Layout   = "16-Team Mirror: two conferences of two divisions"
	mirror16Bracket  = "16-Team Mirror: four teams a side, side champions meet in one final"
)

// Mirror16 is the 16-Team Mirror: the 32-Team Default's rules for 16 franchises in two conferences, each
// conference a side holding its own copy of every player. The sides meet only on the field and in the final.
// It has no O-Line card. Every value is a commissioner setting the league can change afterwards.
func Mirror16() RuleSet {
	r := Default32()
	r.League.Teams = cited(16, mirror16Layout)
	r.League.Divisions = cited(4, mirror16Layout)
	r.League.Alignment = cited(mirror16Alignment(), mirror16Layout)
	r.League.DivisionConferences = cited(map[string]string{"AFC East": "AFC", "AFC West": "AFC",
		"NFC East": "NFC", "NFC West": "NFC"}, mirror16Layout)
	r.League.Pools = cited(PerSide, mirror16Citation)
	// O-Line off: no card on the roster or in the lineup, and no unit scored.
	positions := maps.Clone(r.Lineup.Positions.Value)
	delete(positions, OL)
	r.Lineup.Positions = cited(positions, mirror16Citation)
	r.Lineup.Groups = cited(map[LineupGroup]Bounds{Offense: bounds(5, 5, mirror16Citation),
		Defense: bounds(11, 11, mirror16Citation), Total: bounds(16, 16, mirror16Citation)}, mirror16Citation)
	r.Roster.PositionBounds = cited(map[Position]Bounds{}, mirror16Citation)
	r.Scoring.LineUnit = cited(LineUnit{}, mirror16Citation)
	rounds, _ := StartupRoundsFor(r)
	r.Draft.StartupRounds = cited(rounds, mirror16Citation+"; "+startupRuling)
	r.Playoffs.PerConference = cited(4, mirror16Bracket)
	r.Playoffs.DivisionWinners = cited(2, mirror16Bracket)
	r.Playoffs.WildCards = cited(2, mirror16Bracket)
	r.Playoffs.ByeSeed = cited(0, mirror16Bracket)
	r.Playoffs.StartWeek = cited(14, mirror16Bracket)
	r.Playoffs.FinalWeek = cited(16, mirror16Bracket)
	r.Playoffs.Graph = cited(mirror16Graph(), mirror16Bracket)
	r.Trades.MaximumParties = cited(3, mirror16Citation)
	return r
}

// mirror16Alignment puts four franchises in each division, in franchise order.
func mirror16Alignment() map[string]string {
	alignment := map[string]string{}
	for i, division := range []string{"AFC East", "AFC West", "NFC East", "NFC West"} {
		for seat := 1; seat <= 4; seat++ {
			alignment[fmt.Sprintf("franchise.%02d", i*4+seat)] = division
		}
	}
	return alignment
}

// mirror16Graph is each side's two semifinals and side final, then the one final between the side champions.
func mirror16Graph() []BracketNode {
	node := func(id string, week, home, away int, homeFrom, awayFrom string) BracketNode {
		n := BracketNode{ID: cited(id, mirror16Bracket), Week: cited(week, mirror16Bracket),
			HomeSeed: Needs[int]("16-Team Mirror: the winner arrives"), AwaySeed: Needs[int]("16-Team Mirror: the winner arrives"),
			HomeWinner: cited(homeFrom, mirror16Bracket), AwayWinner: cited(awayFrom, mirror16Bracket)}
		if home != 0 {
			n.HomeSeed, n.AwaySeed = cited(home, mirror16Bracket), cited(away, mirror16Bracket)
		}
		return n
	}
	return []BracketNode{node("semi_1", 14, 1, 4, "", ""), node("semi_2", 14, 2, 3, "", ""),
		node("conference", 15, 0, 0, "winner of semi_1", "winner of semi_2"),
		node("championship", 16, 0, 0, "conference winner", "conference winner")}
}
