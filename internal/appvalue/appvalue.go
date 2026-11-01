// Package appvalue assembles league-scored production for the existing model pass.
package appvalue

import (
	"context"
	"fmt"
	"maps"
	"math"
	"sort"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/fantasyscore"
	"github.com/secureprospective/TheWarRoom/internal/fantasyscore/source"
	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
	"github.com/secureprospective/TheWarRoom/internal/lineupdelta"
	"github.com/secureprospective/TheWarRoom/internal/m2service"
	"github.com/secureprospective/TheWarRoom/internal/model"
	"github.com/secureprospective/TheWarRoom/internal/modelrun"
	"github.com/secureprospective/TheWarRoom/internal/playerrecord"
	"github.com/secureprospective/TheWarRoom/internal/store/history"
	"github.com/secureprospective/TheWarRoom/internal/store/params"
	"github.com/secureprospective/TheWarRoom/internal/store/state"
)

// Facts is the player record: every per-player fact the model reads, keyed by our permanent id.
type Facts interface {
	Book(context.Context, playerrecord.Query) (playerrecord.Book, error)
}

// Inputs is one league as the valuation sees it: the season valued, the league's own scoring and
type Inputs struct {
	League   string
	Season   int
	AsOf     time.Time
	LastWeek int // the league's last scoring week; weeks after it are not played yet
	Scoring  leaguerules.ScoringRules
	Lineup   leaguerules.LineupRules
	// Owner is every rostered player: the league's player id to the franchise that holds him.
	Owner     map[string]string
	Contracts map[string]state.PlayerState
	// Weeks are the held NFL weekly stat lines by season, as internal/store/history holds them.
	Weeks map[int][]source.PlayerWeek
	// GSIS maps the league's player id to his NFL statistics id.
	GSIS       map[string]string
	Franchises []string
	Notes      []string
}

// Player is one player's value in this league, in its own points per game.
type Player struct {
	Player    string  `json:"player"`
	Franchise string  `json:"franchise"`
	Position  string  `json:"position,omitempty"`
	NowPPG    float64 `json:"nowPPG"`
	DynastyPP float64 `json:"dynastyPPG"`
	Age       float64 `json:"age"`
	AgeKnown  bool    `json:"ageKnown"`
	Available bool    `json:"available"`
	Words     string  `json:"words"`
}

// Best is one franchise's best-N total and where it stands in the league. N is the league's own
type Best struct {
	Franchise string  `json:"franchise"`
	StarterN  int     `json:"starterN"`
	Value     float64 `json:"value"`
	Age       float64 `json:"age"`
	AgeKnown  bool    `json:"ageKnown"`
	Rank      int     `json:"rank"`
	Of        int     `json:"of"`
	Available bool    `json:"available"`
	Words     string  `json:"words"`
}

// Band is one bar of the value-by-age histogram (WR-006): the roster's value held by players of
type Band struct {
	From    int     `json:"from"`
	To      int     `json:"to"`
	Open    bool    `json:"open"`
	Value   float64 `json:"value"`
	Players int     `json:"players"`
}

// Reading is the one league-scoped value read.
type Reading struct {
	League    string   `json:"league"`
	Season    int      `json:"season"`
	Franchise string   `json:"franchise"`
	Players   []Player `json:"players"`
	Best      Best     `json:"best"`
	Age       []Band   `json:"age"`
	Notes     []string `json:"notes"`
}

// bandFrom are the first ages of the value-by-age bars; the last is open-ended.
func bandFrom() []int { return []int{0, 24, 26, 28, 30, 32} }

// Value reads every rostered player's value in the league, then franchise's best N and its value
func Value(ctx context.Context, facts Facts, set params.Set, franchise string, in Inputs) (Reading, error) {
	out := Reading{League: in.League, Season: in.Season, Franchise: franchise, Players: []Player{},
		Age: []Band{}, Notes: append([]string{}, in.Notes...)}
	if facts == nil {
		return out, fmt.Errorf("league values: the player record is not open")
	}
	if in.Season < 1920 || in.Season > 9999 || in.LastWeek < 1 || in.LastWeek > 22 {
		return out, fmt.Errorf("league values: choose a season and the league's last scoring week")
	}
	spec := modelrun.Spec{Season: in.Season, AsOf: in.AsOf, Params: set, LastWeek: in.LastWeek}
	book, err := facts.Book(ctx, modelrun.BookQuery(spec))
	if err != nil {
		return out, fmt.Errorf("league values: read the player record: %w", err)
	}
	spec.FreeAgents = freeAgents(book, in)
	scored, weeks := leagueSeasons(in, book)
	book.Data.Players = maps.Clone(book.Data.Players)
	for id := range scored {
		pl := book.Player(id)
		book.Data.Players[id] = &pl
	}
	book.Data.Seasons = scored
	if weeks == 0 {
		out.Notes = append(out.Notes, "No NFL weekly statistics are held yet, so this league has scored no season")
	}
	valuation, err := modelrun.NewValuation(book, spec)
	if err != nil {
		return out, fmt.Errorf("league values: %w", err)
	}
	scores, err := valuation.Scores(rosterState{in}, spec)
	if err != nil {
		return out, fmt.Errorf("league values: %w", err)
	}
	all := readPlayers(valuation, scores, in)
	out.Players = all
	if franchise != "" {
		out.Players = make([]Player, 0, len(all))
		for _, p := range all {
			if p.Franchise == franchise {
				out.Players = append(out.Players, p)
			}
		}
	}
	out.Best = best(all, in, franchise)
	out.Age = bands(out.Players)
	return out, nil
}

// leagueSeasons is every rostered-or-not player's league points by season: his held NFL weeks,
func leagueSeasons(in Inputs, book playerrecord.Book) (map[string]map[int]*model.Season, int) {
	ours := map[string]string{}
	for league, gsis := range in.GSIS {
		if gsis == "" {
			continue
		}
		if id, ok := book.ID(league); ok && id != "" {
			ours[gsis] = id
		}
	}
	out := map[string]map[int]*model.Season{}
	counted := 0
	for _, weeks := range in.Weeks {
		for _, w := range weeks {
			id := ours[w.GSIS]
			if id == "" || w.SeasonType != "REG" || w.Week < 1 || w.Week > in.LastWeek {
				continue
			}
			if addWeek(out, in.Scoring, book, id, w) {
				counted++
			}
		}
	}

	for _, years := range out {
		for _, s := range years {
			s.Total = 0
			for _, week := range weekOrder(s) {
				s.Total += s.Points[week]
			}
		}
	}
	return out, counted
}

func addWeek(out map[string]map[int]*model.Season, rules leaguerules.ScoringRules, book playerrecord.Book, id string, w source.PlayerWeek) bool {
	pos := scoringPosition(book.Records[id].Position)
	if pos == "" {
		return false
	}
	points, _, err := fantasyscore.Score(rules, pos, w.Line)
	if err != nil || points.State != leaguerules.IsKnown {
		return false
	}
	value := float64(points.Value) / 10000
	// As in model.Build, a game needs a snap or a nonzero league score.
	prior := book.Data.Seasons[id][w.Season]
	played := false
	if prior != nil {
		_, played = prior.Points[w.Week]
	}
	if value == 0 && !played {
		return false
	}
	if out[id] == nil {
		out[id] = map[int]*model.Season{}
	}
	season := out[id][w.Season]
	if season == nil {
		season = &model.Season{Player: id, Year: w.Season, Points: map[int]float64{}, HasTotal: true}
		out[id][w.Season] = season
	}
	season.Points[w.Week] += value
	return true
}

// weekOrder sums a season's weeks in week order, so the same season gives the same bits every time.
func weekOrder(s *model.Season) []int {
	out := make([]int, 0, len(s.Points))
	for w := range s.Points {
		out = append(out, w)
	}
	sort.Ints(out)
	return out
}

// scoringPosition is the league rules' code for the position the record lists a player at.
func scoringPosition(p domain.Position) leaguerules.Position {
	switch p {
	case domain.PosFlag:
		return ""
	case domain.PosK:
		return leaguerules.PK
	case domain.PosQB, domain.PosRB, domain.PosWR, domain.PosTE, domain.PosDT, domain.PosDE,
		domain.PosLB, domain.PosCB, domain.PosS:
		return leaguerules.Position(p)
	}
	return ""
}

// readPlayers values every rostered player, in the league's player-id order so the read is stable.
func readPlayers(v modelrun.Valuation, scores []history.ModelScore, in Inputs) []Player {
	byID := map[string]history.ModelScore{}
	for _, sc := range scores {
		byID[sc.MFLID] = sc
	}
	ids := make([]string, 0, len(scores))
	for _, sc := range scores {
		ids = append(ids, sc.MFLID)
	}
	sort.Strings(ids)
	out := make([]Player, 0, len(ids))
	book := v.Book()
	for _, league := range ids {
		row := Player{Player: league, Franchise: in.Owner[league]}
		ours, _ := book.ID(league)
		score, ok := byID[league]
		if !ok || score.Unavailable {
			row.Words = score.Reason
			out = append(out, row)
			continue
		}
		pl := book.Player(ours)
		if age := pl.AgeAt(in.Season); !math.IsNaN(age) {
			row.Age, row.AgeKnown = age, true
		}
		row.Position, row.NowPPG, row.DynastyPP = score.Position, score.NowPPG, score.DynastyPPG
		row.Available, row.Words = true, score.Reason
		out = append(out, row)
	}
	return out
}

// best counts each franchise's top N by dynasty value with the board's own counting rule
func best(all []Player, in Inputs, franchise string) Best {
	out := Best{Franchise: franchise}
	rules, _ := lineupdelta.Rules(in.Lineup)
	total, ok := in.Lineup.Groups.Value[leaguerules.Total]
	if in.Lineup.Groups.State != leaguerules.IsKnown || !ok ||
		total.Max.State != leaguerules.IsKnown || total.Max.Value <= 0 {
		out.Words = "This league's starter count is not set, so there is no best N to count"
		return out
	}
	rules.Total = total.Max.Value
	out.StarterN = rules.Total
	byFranchise := countable(all)
	counted := map[string]float64{}
	for _, fid := range franchises(in) {
		value, age := m2service.Count(m2service.AggTopN, rules, byFranchise[fid])
		counted[fid] = value
		if fid != franchise {
			continue
		}
		out.Value = value
		if !math.IsNaN(age) {
			out.Age, out.AgeKnown = age, true
		}
	}
	out.Of = len(counted)
	if franchise == "" {
		out.Words = "Choose a franchise to read its best N"
		return out
	}
	if len(byFranchise[franchise]) == 0 {
		out.Words = "No player on this roster could be valued in this league yet"
		return out
	}
	out.Rank = 1
	for fid, value := range counted {
		if fid != franchise && value > out.Value {
			out.Rank++
		}
	}
	out.Available = true
	rostered := 0
	for _, owner := range in.Owner {
		if owner == franchise {
			rostered++
		}
	}
	out.Words = fmt.Sprintf("%d of %d roster players valued", len(byFranchise[franchise]), rostered)
	return out
}

// franchises is every franchise the league rosters a player for, in a stable order.
func franchises(in Inputs) []string {
	seen := map[string]bool{}
	var out []string
	for _, fid := range in.Franchises {
		if !seen[fid] {
			seen[fid] = true
			out = append(out, fid)
		}
	}
	for _, fid := range in.Owner {
		if fid != "" && !seen[fid] {
			seen[fid] = true
			out = append(out, fid)
		}
	}
	sort.Strings(out)
	return out
}

// bands is the value-by-age histogram over the players read: each bar the value held by players
func bands(players []Player) []Band {
	first := bandFrom()
	out := make([]Band, len(first))
	for i, from := range first {
		out[i] = Band{From: from, To: from + 1, Open: i == len(first)-1}
	}
	for _, p := range players {
		if !p.Available || !p.AgeKnown {
			continue
		}
		i := bandOf(p.Age, first)
		if i < 0 {
			continue
		}
		out[i].Value += p.DynastyPP
		out[i].Players++
	}
	return out
}

func bandOf(age float64, first []int) int {
	if age < float64(first[0]) {
		return -1
	}
	for i := len(first) - 1; i >= 0; i-- {
		if age >= float64(first[i]) {
			return i
		}
	}
	return -1
}

func countable(all []Player) map[string][]m2service.PlayerValue {
	byFranchise := map[string][]m2service.PlayerValue{}
	for _, p := range all {
		if !p.Available || p.Franchise == "" {
			continue
		}
		age := math.NaN()
		if p.AgeKnown {
			age = p.Age
		}
		byFranchise[p.Franchise] = append(byFranchise[p.Franchise], m2service.PlayerValue{
			MFLID: p.Player, Position: p.Position, Value: p.DynastyPP, Age: age})
	}
	return byFranchise
}

// rosterState supplies the selected league's contracts to the existing run adjustment.
type rosterState struct{ in Inputs }

func (r rosterState) Franchises() []string { return franchises(r.in) }
func (r rosterState) Roster(fid string) ([]state.PlayerState, bool) {
	var out []state.PlayerState
	for _, id := range sortedOwners(r.in.Owner) {
		if r.in.Owner[id] != fid {
			continue
		}
		p := r.in.Contracts[id]
		p.MFLID = id
		out = append(out, p)
	}
	return out, true
}
func (rosterState) FranchiseState(string) (state.FranchiseState, bool) {
	return state.FranchiseState{}, false
}
func (rosterState) CapUsed(string) (domain.Money, bool)     { return 0, false }
func (rosterState) Player(string) (state.PlayerState, bool) { return state.PlayerState{}, false }
func sortedOwners(owner map[string]string) []string {
	ids := make([]string, 0, len(owner))
	for id := range owner {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	return ids
}

func freeAgents(book playerrecord.Book, in Inputs) []string {
	var ids []string
	for _, rec := range book.Records {
		if rec.Listed && in.Owner[rec.MFL] == "" {
			ids = append(ids, rec.MFL)
		}
	}
	sort.Strings(ids)
	return ids
}
