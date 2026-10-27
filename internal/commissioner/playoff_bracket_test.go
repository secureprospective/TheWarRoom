package commissioner

import (
	"fmt"
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/engine/competition"
	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
)

// scripted scores a team by how it finished (better is higher) unless the test names its score for a week.
type scripted map[string]map[int]leaguerules.Decimal

func (s scripted) points(rank map[string]int) competition.PointsFunc {
	return func(franchise string, week int) (*leaguerules.Decimal, string) {
		if v, ok := s[franchise][week]; ok {
			return &v, ""
		}
		v := leaguerules.Decimal((100 - rank[franchise]) * 10000)
		return &v, ""
	}
}

func gameOf(rounds []BracketRound, id string) BracketGame {
	for _, r := range rounds {
		for _, g := range r.Games {
			if g.ID == id {
				return g
			}
		}
	}
	return BracketGame{}
}

func pair(g BracketGame) string {
	if g.Home == nil || g.Away == nil {
		return "waiting"
	}
	return g.Home.Franchise + " v " + g.Away.Franchise
}

func TestASmallerLeagueFollowsItsOwnShape(t *testing.T) {
	// One conference of eight teams: four make it, the top two seeds skip nothing, and the shape names winners.
	rules := leaguerules.LegacyNFL()
	rules.Playoffs.PerConference = leaguerules.Known(4)
	rules.Playoffs.DivisionWinners = leaguerules.Known(2)
	rules.Playoffs.WildCards = leaguerules.Known(2)
	rules.Playoffs.ByeSeed = leaguerules.Known(0)
	rules.Playoffs.StartWeek, rules.Playoffs.FinalWeek = leaguerules.Known(15), leaguerules.Known(16)
	node := func(id string, week, home, away int, homeFrom, awayFrom string) leaguerules.BracketNode {
		n := leaguerules.BracketNode{ID: leaguerules.Known(id), Week: leaguerules.Known(week),
			HomeSeed: leaguerules.Known(home), AwaySeed: leaguerules.Known(away),
			HomeWinner: leaguerules.Known(homeFrom), AwayWinner: leaguerules.Known(awayFrom)}
		return n
	}
	graph := []leaguerules.BracketNode{node("semi_1", 15, 1, 4, "", ""), node("semi_2", 15, 2, 3, "", ""),
		node("final", 16, 0, 0, "winner of semi_1", "winner of semi_2")}
	pr := ResolvePlayoffRules(rules, graph)
	if pr.Words != "" {
		t.Fatal(pr.Words)
	}
	in := competition.FieldInput{Teams: map[string]competition.FieldTeam{}, Conferences: []string{singleConference}, Rules: pr}
	names := map[string]string{}
	for i := 1; i <= 8; i++ {
		id := fmt.Sprintf("t%d", i)
		in.Order = append(in.Order, id)
		in.Teams[id] = competition.FieldTeam{Name: id, Division: fmt.Sprintf("d%d", (i+1)/2), Conference: singleConference}
		names[id] = id
	}
	fields, words := competition.SeedField(in)
	if words != "" {
		t.Fatal(words)
	}
	rank := map[string]int{}
	for i, id := range in.Order {
		rank[id] = i + 1
	}
	rounds, champion, words := competition.BuildBracket(competition.BracketInput{Rules: pr, Fields: fields, Rank: rank, Names: names,
		Points: scripted{}.points(rank), Edits: map[string]string{}, Advance: map[string]string{}})
	if words != "" || champion == nil {
		t.Fatalf("%q %+v", words, champion)
	}
	// Seeds 1 to 4 are the two division winners, t1 and t3, then the wild cards t2 and t4.
	if got := pair(gameOf(rounds, "League/semi_1")); got != "t1 v t4" {
		t.Fatalf("semi 1 = %s, field %+v", got, fields[0].Seeds)
	}
	if final := gameOf(rounds, "League/final"); final.Conference != "League" || champion.Franchise != "t1" {
		t.Fatalf("final %+v champion %+v", final, champion)
	}
}
