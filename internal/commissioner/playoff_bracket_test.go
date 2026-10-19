package commissioner

import (
	"fmt"
	"strings"
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
)

// twoConferences is 16 teams in two conferences of four divisions: a1..a8 and b1..b8, ranked a1, b1, a2, b2...
// Teams 2k-1 and 2k share a division, so the second of each pair is a wild card, never a division winner.
func twoConferences() (fieldInput, map[string]string) {
	in := fieldInput{teams: map[string]fieldTeam{}, conferences: []string{"AFC", "NFC"},
		rules: legacyPlayoffRules(), hand: map[string]map[int]string{}}
	names := map[string]string{}
	for i := 1; i <= 8; i++ {
		for _, c := range []struct{ id, prefix string }{{"AFC", "a"}, {"NFC", "b"}} {
			id := fmt.Sprintf("%s%d", c.prefix, i)
			in.order = append(in.order, id)
			in.teams[id] = fieldTeam{Name: "Team " + id, Division: fmt.Sprintf("%s-%d", c.id, (i+1)/2), Conference: c.id}
			names[id] = "Team " + id
		}
	}
	return in, names
}

func legacyPlayoffRules() PlayoffRules {
	rules := leaguerules.LegacyNFL()
	return ResolvePlayoffRules(rules, rules.Playoffs.Graph.Value)
}

// scripted scores a team by how it finished (better is higher) unless the test names its score for a week.
type scripted map[string]map[int]leaguerules.Decimal

func (s scripted) points(rank map[string]int) pointsFunc {
	return func(franchise string, week int) (*leaguerules.Decimal, string) {
		if v, ok := s[franchise][week]; ok {
			return &v, ""
		}
		v := leaguerules.Decimal((100 - rank[franchise]) * 10000)
		return &v, ""
	}
}

func points(n int) leaguerules.Decimal { return leaguerules.Decimal(n * 10000) }

func legacyBracketFixture(t *testing.T, s scripted) (bracketInput, []BracketRound, *BracketSide, string) {
	t.Helper()
	field, names := twoConferences()
	fields, words := seedField(field)
	if words != "" {
		t.Fatal(words)
	}
	rank := map[string]int{}
	for i, id := range field.order {
		rank[id] = i + 1
	}
	in := bracketInput{rules: field.rules, fields: fields, rank: rank, names: names, points: s.points(rank),
		edits: map[string]string{}, advance: map[string]string{}}
	rounds, champion, words := buildBracket(in)
	return in, rounds, champion, words
}

func seedOf(fields []ConferenceField, conference, team string) int {
	for _, f := range fields {
		if f.Conference != conference {
			continue
		}
		for _, s := range f.Seeds {
			if s.Franchise == team {
				return s.Seed
			}
		}
	}
	return 0
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

func TestLegacyFieldSeedsDivisionWinnersBeforeWildCards(t *testing.T) {
	field, _ := twoConferences()
	fields, words := seedField(field)
	if words != "" {
		t.Fatal(words)
	}
	// a2 finished third in the league but shares a division with a1, so it is a wild card.
	got := []string{}
	for _, s := range fields[0].Seeds {
		got = append(got, fmt.Sprintf("%d:%s:%s", s.Seed, s.Franchise, s.Kind))
	}
	want := "1:a1:Division winner 2:a3:Division winner 3:a5:Division winner 4:a7:Division winner " +
		"5:a2:Wild card 6:a4:Wild card 7:a6:Wild card"
	if strings.Join(got, " ") != want {
		t.Fatalf("field = %s", strings.Join(got, " "))
	}
	if !fields[0].Seeds[0].Bye || fields[0].Seeds[1].Bye {
		t.Fatal("only the first seed has a bye")
	}
}

func TestLegacyFirstRoundLeavesTheTopSeedOnABye(t *testing.T) {
	_, rounds, _, words := legacyBracketFixture(t, scripted{})
	if words != "" {
		t.Fatal(words)
	}
	if len(rounds) != 4 || rounds[0].Week != 14 || rounds[3].Week != 17 {
		t.Fatalf("rounds: %+v", rounds)
	}
	wantRound1 := map[string]string{"AFC/wildcard_2": "a3 v a6", "AFC/wildcard_3": "a5 v a4", "AFC/wildcard_4": "a7 v a2",
		"NFC/wildcard_2": "b3 v b6", "NFC/wildcard_3": "b5 v b4", "NFC/wildcard_4": "b7 v b2"}
	if len(rounds[0].Games) != len(wantRound1) {
		t.Fatalf("round one has %d games", len(rounds[0].Games))
	}
	for _, g := range rounds[0].Games {
		if pair(g) != wantRound1[g.ID] {
			t.Fatalf("%s = %s", g.ID, pair(g))
		}
		for _, side := range []*BracketSide{g.Home, g.Away} {
			if side.Seed == 1 {
				t.Fatalf("the top seed plays in the bye round: %s", g.ID)
			}
		}
	}
	if g := gameOf(rounds, "AFC/division_1"); g.Home == nil || g.Home.Franchise != "a1" || g.Week != 15 {
		t.Fatalf("the top seed enters in week 15: %+v", g)
	}
}

func TestLegacyReseedsEveryRoundByTheSeedsLeft(t *testing.T) {
	// The 7th seed and 3rd seed win, the 5th seed beats the 4th: the 1st seed must host the lowest seed left.
	_, rounds, _, words := legacyBracketFixture(t, scripted{"a6": {14: points(200)}})
	if words != "" {
		t.Fatal(words)
	}
	if got := pair(gameOf(rounds, "AFC/division_1")); got != "a1 v a6" {
		t.Fatalf("top seed plays %s, want the lowest seed left (a6)", got)
	}
	if got := pair(gameOf(rounds, "AFC/division_2")); got != "a5 v a2" {
		t.Fatalf("the other two seeds play %s, higher seed hosting", got)
	}
	// The NFC played no upset, so its top seed hosts the 7th seed's slot winner b3's lower seed.
	if got := pair(gameOf(rounds, "NFC/division_1")); got != "b1 v b2" {
		t.Fatalf("NFC divisional = %s", got)
	}
}

func TestHomeFieldBonusGoesToTheHomeTeamOnly(t *testing.T) {
	// a3 hosts a6. a6 scores two points more, so only the three home field points save a3.
	s := scripted{"a3": {14: points(100)}, "a6": {14: points(102)}}
	_, rounds, _, _ := legacyBracketFixture(t, s)
	g := gameOf(rounds, "AFC/wildcard_2")
	if g.Winner != "a3" || g.HomeTotal == nil || *g.HomeTotal != points(103) || *g.AwayTotal != points(102) {
		t.Fatalf("home field: %+v", g)
	}
	if !strings.Contains(g.Words, "home field points for Team a3") {
		t.Fatalf("words: %s", g.Words)
	}
	s["a6"][14] = points(104)
	_, rounds, _, _ = legacyBracketFixture(t, s)
	if g := gameOf(rounds, "AFC/wildcard_2"); g.Winner != "a6" {
		t.Fatalf("a bigger score beats the bonus: %+v", g)
	}
}

func TestAnAdvancementOverrideRepairsTheNextRound(t *testing.T) {
	in, rounds, _, _ := legacyBracketFixture(t, scripted{})
	// a5 hosts a4 and wins by the bonus, a2 beats a7: the lowest seed left is a2.
	if got := pair(gameOf(rounds, "AFC/division_1")); got != "a1 v a2" {
		t.Fatalf("without an override: %s", got)
	}
	// Computed: a3 beats a6. The commissioner advances a6.
	in.advance["AFC/wildcard_2"] = "a6"
	rounds, _, words := buildBracket(in)
	if words != "" {
		t.Fatal(words)
	}
	over := gameOf(rounds, "AFC/wildcard_2")
	if over.Winner != "a6" || !over.ByHand || over.Status != gameByHand || !strings.Contains(over.Words, "despite the computed result") {
		t.Fatalf("override: %+v", over)
	}
	if got := pair(gameOf(rounds, "AFC/division_1")); got != "a1 v a6" {
		t.Fatalf("the next round did not re-pair from a6: %s", got)
	}
	in.advance["AFC/wildcard_2"] = "a1"
	rounds, _, _ = buildBracket(in)
	if g := gameOf(rounds, "AFC/wildcard_2"); g.Winner != "a3" || !strings.Contains(g.Words, "set aside") {
		t.Fatalf("a team outside the game cannot advance from it: %+v", g)
	}
}

func TestATiedPlayoffGameHasNoWinnerUntilTheCommissionerDecides(t *testing.T) {
	// a5 hosts a4: 100 + 3 against 103 is level.
	s := scripted{"a5": {14: points(100)}, "a4": {14: points(103)}}
	in, rounds, _, _ := legacyBracketFixture(t, s)
	g := gameOf(rounds, "AFC/wildcard_3")
	if g.Status != GameTie || g.Winner != "" || !strings.Contains(g.Words, "commissioner decides") {
		t.Fatalf("tie: %+v", g)
	}
	if got := pair(gameOf(rounds, "AFC/division_1")); got != "waiting" {
		t.Fatalf("the divisional round must wait for the tie: %s", got)
	}
	in.advance["AFC/wildcard_3"] = "a4"
	rounds, _, _ = buildBracket(in)
	if got := pair(gameOf(rounds, "AFC/division_1")); got == "waiting" {
		t.Fatalf("a hand decision should release the next round: %s", got)
	}
}

func TestTheChampionIsTheWinnerOfTheFinalAcrossConferences(t *testing.T) {
	_, rounds, champion, words := legacyBracketFixture(t, scripted{})
	if words != "" || champion == nil || champion.Franchise != "a1" {
		t.Fatalf("champion %+v %q", champion, words)
	}
	final := gameOf(rounds, "championship")
	if final.Conference != "" || pair(final) != "a1 v b1" || final.Week != 17 {
		t.Fatalf("final: %+v", final)
	}
}

func TestAHandEditReplacesOnePlaceAndAHandSeedOneSlot(t *testing.T) {
	in, _, _, _ := legacyBracketFixture(t, scripted{})
	in.edits["AFC/wildcard_2/away"] = "a8"
	rounds, _, words := buildBracket(in)
	if words != "" {
		t.Fatal(words)
	}
	g := gameOf(rounds, "AFC/wildcard_2")
	if g.Away.Franchise != "a8" || !g.Away.ByHand || g.Home.ByHand {
		t.Fatalf("edit: %+v", g)
	}
	in.edits["AFC/wildcard_3/home"] = "a8"
	if _, _, words = buildBracket(in); !strings.Contains(words, "two games") {
		t.Fatalf("a team in two games of one round: %q", words)
	}

	field, _ := twoConferences()
	field.hand["AFC"] = map[int]string{1: "a8"}
	fields, words := seedField(field)
	if words != "" {
		t.Fatal(words)
	}
	if seedOf(fields, "AFC", "a8") != 1 || !fields[0].Seeds[0].ByHand || fields[0].Seeds[0].Kind != kindByHand ||
		seedOf(fields, "AFC", "a1") != 2 || seedOf(fields, "NFC", "b1") != 1 {
		t.Fatalf("hand seed: %+v", fields[0].Seeds)
	}
	field.hand["AFC"] = map[int]string{1: "b1"}
	if _, words := seedField(field); !strings.Contains(words, "no longer fits") {
		t.Fatalf("a hand seed from another conference: %q", words)
	}
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
	in := fieldInput{teams: map[string]fieldTeam{}, conferences: []string{singleConference}, rules: pr}
	names := map[string]string{}
	for i := 1; i <= 8; i++ {
		id := fmt.Sprintf("t%d", i)
		in.order = append(in.order, id)
		in.teams[id] = fieldTeam{Name: id, Division: fmt.Sprintf("d%d", (i+1)/2), Conference: singleConference}
		names[id] = id
	}
	fields, words := seedField(in)
	if words != "" {
		t.Fatal(words)
	}
	rank := map[string]int{}
	for i, id := range in.order {
		rank[id] = i + 1
	}
	rounds, champion, words := buildBracket(bracketInput{rules: pr, fields: fields, rank: rank, names: names,
		points: scripted{}.points(rank), edits: map[string]string{}, advance: map[string]string{}})
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

func TestPlayoffRulesSayWhatTheyStillNeed(t *testing.T) {
	rules := leaguerules.LegacyNFL()
	graph := rules.Playoffs.Graph.Value
	for name, edit := range map[string]func(*leaguerules.RuleSet){
		"per":    func(r *leaguerules.RuleSet) { r.Playoffs.PerConference = leaguerules.Needs[int]("x") },
		"sum":    func(r *leaguerules.RuleSet) { r.Playoffs.WildCards = leaguerules.Known(2) },
		"weeks":  func(r *leaguerules.RuleSet) { r.Playoffs.FinalWeek = leaguerules.Known(18) },
		"reseed": func(r *leaguerules.RuleSet) { r.Playoffs.Reseed = leaguerules.Known(false) },
		"policy": func(r *leaguerules.RuleSet) { r.Playoffs.Seeding = leaguerules.Known(leaguerules.SeedPolicy("best")) },
	} {
		r := leaguerules.LegacyNFL()
		edit(&r)
		got := ResolvePlayoffRules(r, graph)
		if !strings.HasPrefix(got.Words, "Needs a ruling: ") {
			t.Fatalf("%s: %q", name, got.Words)
		}
	}
	if got := ResolvePlayoffRules(rules, nil); !strings.HasPrefix(got.Words, "Needs a ruling: the playoff bracket shape") {
		t.Fatalf("no shape: %q", got.Words)
	}
	if got := legacyPlayoffRules(); got.Words != "" || got.PerConference != 7 || got.HomeBonus != points(3) || got.ByeSeeds != 1 {
		t.Fatalf("Legacy: %+v", got)
	}
}

func TestPlayoffRulesAreReadFromTheSeasonsOwnRules(t *testing.T) {
	r := leaguerules.LegacyNFL()
	r.Playoffs.HomeBonus = leaguerules.Known(leaguerules.Decimal(50000))
	r.Playoffs.ByeSeed = leaguerules.Known(2)
	got := ResolvePlayoffRules(r, r.Playoffs.Graph.Value)
	if got.Words != "" || got.HomeBonus != points(5) || got.ByeSeeds != 2 || got.StartWeek != 14 || got.FinalWeek != 17 ||
		!got.Reseed || got.DivisionWinners != 4 || got.WildCards != 3 || len(got.Shape) != 7 {
		t.Fatalf("%+v", got)
	}
	field, _ := twoConferences()
	field.rules = got
	fields, words := seedField(field)
	if words != "" || !fields[0].Seeds[1].Bye || fields[0].Seeds[2].Bye {
		t.Fatalf("two byes: %q %+v", words, fields[0].Seeds)
	}
}
