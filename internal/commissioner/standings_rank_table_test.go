package commissioner

import (
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/engine/competition"

	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
)

func TestRankStandingsTable(t *testing.T) {
	base := rankFixture()
	base.Teams = base.Teams[:2]
	cases := []struct {
		name  string
		in    competition.RankInput
		order []string
		line  string
		level [][]string
	}{}
	add := func(name string, in competition.RankInput, order []string, line string, level [][]string) {
		cases = append(cases, struct {
			name  string
			in    competition.RankInput
			order []string
			line  string
			level [][]string
		}{name, in, order, line, level})
	}
	in := base
	in.Criteria = []leaguerules.Criterion{leaguerules.WinPercentage}
	in.Stands = map[string]competition.Tally{"a": {W: 1, L: 3}, "b": {W: 3, L: 1}}
	add("winning percentage", in, []string{"b", "a"}, "Winning percentage put Beta (3-1) over Alpha (1-3).", nil)
	in = base
	in.Criteria = []leaguerules.Criterion{leaguerules.WinPercentage, leaguerules.HeadToHeadRecord, leaguerules.Points}
	add("criterion order head-to-head before points", in, []string{"a", "b"}, "Alpha and Beta were level on winning percentage. Head-to-head record put Alpha (1-0) over Beta (0-1).", nil)
	in = base
	in.Criteria = []leaguerules.Criterion{leaguerules.DivisionPercentage}
	add("division", in, []string{"a", "b"}, "Division record put Alpha (1-0) over Beta (0-1).", nil)
	in = base
	in.Criteria = []leaguerules.Criterion{leaguerules.ConferencePercentage}
	add("conference", in, []string{"a", "b"}, "Conference record put Alpha (1-0) over Beta (0-1).", nil)
	in = base
	in.Criteria = []leaguerules.Criterion{leaguerules.AllPlayPercentage}
	add("all-play", in, []string{"b", "a"}, "All-play record put Beta (3-1) over Alpha (1-3).", nil)
	in = base
	in.Criteria = []leaguerules.Criterion{leaguerules.Points}
	add("points", in, []string{"b", "a"}, "Points scored put Beta (0.001 points scored) over Alpha (0.0001 points scored).", nil)
	in = base
	in.Criteria = []leaguerules.Criterion{leaguerules.PowerRank}
	add("power rank unavailable", in, []string{"a", "b"}, "Power rank did not apply: no power rank is held for this league yet. Alpha and Beta remain level on the available tie-breakers, so they are listed in the league's franchise order.", [][]string{{"a", "b"}})
	in = base
	in.Stands = map[string]competition.Tally{"a": {W: 1}, "b": {W: 1}}
	in.Criteria = []leaguerules.Criterion{leaguerules.WinPercentage}
	add("undecided equal records", in, []string{"a", "b"}, "Alpha and Beta remain level on the available tie-breakers, so they are listed in the league's franchise order.", [][]string{{"a", "b"}})
	// A three-way head-to-head is skipped for the entire group unless every pair has met.
	in = competition.RankInput{Teams: []competition.RankTeam{{ID: "a", Name: "Alpha"}, {ID: "b", Name: "Beta"}, {ID: "c", Name: "Charlie"}}, Games: []competition.RankGame{{Home: "a", Away: "b", HomePoints: 1}, {Home: "b", Away: "c", HomePoints: 1}}, Points: map[string]leaguerules.Decimal{"a": 10000, "b": 20000, "c": 30000}, Criteria: []leaguerules.Criterion{leaguerules.HeadToHeadRecord, leaguerules.Points}}
	add("three-way incomplete head-to-head", in, []string{"c", "b", "a"}, "Head-to-head record did not apply: not every tied team has played every other. Points scored put Charlie (3 points scored) over Beta (2 points scored) over Alpha (1 points scored).", nil)
	records := default32League(t)
	ids := records[0].Command.Args.Franchises
	if len(ids) != 32 {
		t.Fatalf("default32 franchises: %d", len(ids))
	}
	in = competition.RankInput{Stands: map[string]competition.Tally{}, Criteria: []leaguerules.Criterion{leaguerules.WinPercentage}}
	for i, id := range ids {
		conference := "AFC"
		if i >= 16 {
			conference = "NFC"
		}
		team := competition.RankTeam{ID: id, Name: fmt.Sprintf("Team %02d", i+1), Division: fmt.Sprintf("%s-%d", conference, (i%16)/4+1), Conference: conference}
		if conference == "AFC" {
			in.Teams = append(in.Teams, team)
			in.Stands[id] = competition.Tally{W: 16 - i, L: i}
		}
	}
	order := make([]string, 16)
	copy(order, ids[:16])
	add("default32 AFC", in, order, "Winning percentage put", nil)
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := competition.RankStandings(tc.in)
			if !reflect.DeepEqual(got.Order, tc.order) || !reflect.DeepEqual(got.Level, tc.level) {
				t.Errorf("order/level = %v / %v, want %v / %v", got.Order, got.Level, tc.order, tc.level)
			}
			for _, id := range tc.order {
				if !strings.Contains(strings.Join(got.Why[id], " "), tc.line) {
					t.Errorf("%s explanation = %q, want %q", id, got.Why[id], tc.line)
				}
			}
		})
	}
}

func TestDefault32BracketShape(t *testing.T) {
	records := default32League(t)
	rules, err := RulesInForce(records, 2026, noChangeTime())
	if err != nil {
		t.Fatal(err)
	}
	field := competition.FieldInput{Teams: map[string]competition.FieldTeam{}, Conferences: []string{"AFC", "NFC"}, Rules: ResolvePlayoffRules(rules, rules.Playoffs.Graph.Value), Hand: map[string]map[int]string{}}
	rank := map[string]int{}
	names := map[string]string{}
	for i, id := range records[0].Command.Args.Franchises {
		conference := "AFC"
		if i >= 16 {
			conference = "NFC"
		}
		field.Order = append(field.Order, id)
		field.Teams[id] = competition.FieldTeam{Name: id, Conference: conference, Division: fmt.Sprintf("%s-%d", conference, (i%16)/4)}
		rank[id] = (i + 1) * 10
		names[id] = id
	}
	fields, words := competition.SeedField(field)
	if words != "" {
		t.Fatal(words)
	}
	in := competition.BracketInput{Rules: field.Rules, Fields: fields, Rank: rank, Names: names, Points: scripted{}.points(rank), Edits: map[string]string{}, Advance: map[string]string{}}
	rounds, champion, words := competition.BuildBracket(in)
	if words != "" {
		t.Fatal(words)
	}
	if len(rounds) != 4 || len(rounds[0].Games) != 6 {
		t.Fatalf("rounds: %+v", rounds)
	}
	final := gameOf(rounds, "championship")
	if pair(final) != "franchise.01 v franchise.17" || final.Week != 17 || champion == nil || champion.Franchise != "franchise.01" {
		t.Fatalf("final: %+v champion: %+v rounds: %+v", final, champion, rounds)
	}
	for _, conf := range []string{"AFC", "NFC"} {
		for _, suffix := range []string{"wildcard_2", "wildcard_3", "wildcard_4"} {
			g := gameOf(rounds, conf+"/"+suffix)
			if g.Home == nil || g.Away == nil || field.Teams[g.Home.Franchise].Conference != conf || field.Teams[g.Away.Franchise].Conference != conf {
				t.Fatalf("%s/%s: %+v", conf, suffix, g)
			}
		}
	}
}
