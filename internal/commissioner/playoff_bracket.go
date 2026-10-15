package commissioner

import (
	"slices"

	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
)

// Where a game stands. A game with both teams and both scores has a result; nothing is ever guessed.
const (
	gameWaiting = "Waiting for teams"
	gameByHand  = "Set by the commissioner"
)

// BracketSide is one team in a bracket game, with the seed it holds in its own conference and where it
// finished in the whole league's standings.
type BracketSide struct {
	Franchise string `json:"franchise"`
	Name      string `json:"name"`
	Seed      int    `json:"seed"`
	Rank      int    `json:"rank"`
	ByHand    bool   `json:"byHand"`
}

// BracketGame is one playoff game. HomeSlot and AwaySlot name the places a team can be put by hand.
type BracketGame struct {
	ID         string               `json:"id"`
	Node       string               `json:"node"`
	Conference string               `json:"conference,omitempty"`
	Week       int                  `json:"week"`
	HomeSlot   string               `json:"homeSlot"`
	AwaySlot   string               `json:"awaySlot"`
	Home       *BracketSide         `json:"home,omitempty"`
	Away       *BracketSide         `json:"away,omitempty"`
	Status     string               `json:"status"`
	HomePoints *leaguerules.Decimal `json:"homePoints,omitempty"`
	AwayPoints *leaguerules.Decimal `json:"awayPoints,omitempty"`
	HomeBonus  leaguerules.Decimal  `json:"homeBonus"`
	HomeTotal  *leaguerules.Decimal `json:"homeTotal,omitempty"`
	AwayTotal  *leaguerules.Decimal `json:"awayTotal,omitempty"`
	Winner     string               `json:"winner,omitempty"`
	ByHand     bool                 `json:"byHand"`
	Words      string               `json:"words"`
}

// BracketRound is the games of one playoff week.
type BracketRound struct {
	Round int           `json:"round"`
	Week  int           `json:"week"`
	Name  string        `json:"name"`
	Games []BracketGame `json:"games"`
}

// pointsFunc is a franchise's complete score for a week, or why there is none yet.
type pointsFunc func(franchise string, week int) (*leaguerules.Decimal, string)

type bracketInput struct {
	rules   PlayoffRules
	fields  []ConferenceField
	rank    map[string]int
	names   map[string]string
	points  pointsFunc
	edits   map[string]string
	advance map[string]string
}

// bracketBuilder follows the shape round by round. A round's pairings come from the teams that reached it,
// so a winner changes who plays next; that is why each round is built after the one before it is read.
type bracketBuilder struct {
	in        bracketInput
	seeds     map[string]map[int]string
	seedOf    map[string]int
	games     map[string]bool
	winners   map[string]*BracketSide
	advancers map[string][]*BracketSide
	pending   map[string]bool
}

func newBracketBuilder(in bracketInput) *bracketBuilder {
	b := &bracketBuilder{in: in, seeds: map[string]map[int]string{}, seedOf: map[string]int{}, games: map[string]bool{},
		winners: map[string]*BracketSide{}, advancers: map[string][]*BracketSide{}, pending: map[string]bool{}}
	for _, f := range in.fields {
		b.seeds[f.Conference] = map[int]string{}
		for _, s := range f.Seeds {
			b.seeds[f.Conference][s.Seed], b.seedOf[s.Franchise] = s.Franchise, s.Seed
		}
		for _, n := range in.rules.Shape {
			if !n.leagueWide() {
				b.games[gameID(f.Conference, n.ID)] = true
			}
		}
	}
	for _, n := range in.rules.Shape {
		if n.leagueWide() {
			b.games[gameID("", n.ID)] = true
		}
	}
	return b
}

func gameID(scope, node string) string {
	if scope == "" {
		return node
	}
	return scope + "/" + node
}

// buildBracket pairs every round of the shape for the seeded fields. It returns plain words instead when
// the shape cannot be followed with these teams, and the champion once the last game has a winner.
func buildBracket(in bracketInput) ([]BracketRound, *BracketSide, string) {
	b := newBracketBuilder(in)
	rounds := []BracketRound{}
	for i, nodes := range in.rules.rounds() {
		round := BracketRound{Round: i + 1, Week: nodes[0].Week, Name: roundName(nodes), Games: []BracketGame{}}
		used := map[string]bool{}
		for _, scope := range b.scopes(nodes) {
			games, words := b.round(i, scope, nodes, used)
			if words != "" {
				return nil, nil, words
			}
			round.Games = append(round.Games, games...)
		}
		b.reachNext(round, nodes)
		rounds = append(rounds, round)
	}
	last := rounds[len(rounds)-1].Games[0]
	return rounds, b.winners[last.ID], ""
}

// scopes are the conferences that play this round's conference games, then the whole league when a
// game takes conference winners.
func (b *bracketBuilder) scopes(nodes []shapeNode) []string {
	var out []string
	if slices.ContainsFunc(nodes, func(n shapeNode) bool { return !n.leagueWide() }) {
		for _, f := range b.in.fields {
			out = append(out, f.Conference)
		}
	}
	if slices.ContainsFunc(nodes, shapeNode.leagueWide) {
		out = append(out, "")
	}
	return out
}

// reachNext records who reached the next round, from each scope's winners in the order its games were listed.
func (b *bracketBuilder) reachNext(round BracketRound, nodes []shapeNode) {
	next := map[string][]*BracketSide{}
	pending := map[string]bool{}
	for _, g := range round.Games {
		winner := b.winners[g.ID]
		if winner == nil {
			pending[g.Conference], pending[""] = true, true
			continue
		}
		next[g.Conference] = append(next[g.Conference], winner)
	}
	league := []*BracketSide{}
	if slices.ContainsFunc(nodes, shapeNode.leagueWide) {
		league = next[""]
	} else {
		for _, f := range b.in.fields {
			league = append(league, next[f.Conference]...)
		}
	}
	next[""] = league
	b.advancers, b.pending = next, pending
}

func (b *bracketBuilder) side(franchise string) *BracketSide {
	return &BracketSide{Franchise: franchise, Name: b.in.names[franchise], Seed: b.seedOf[franchise],
		Rank: b.in.rank[franchise]}
}
