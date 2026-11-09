package commissioner

import (
	"context"
	"fmt"
	"maps"
	"math"
	"math/rand/v2"
	"slices"
	"sync"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/engine/competition"
	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
	"github.com/secureprospective/TheWarRoom/internal/lineupdelta"
	"github.com/secureprospective/TheWarRoom/internal/powerrankings"
)

// OddsRuns is how many times the odds reading plays out the open games: one standard error on a 50% chance is
// 1.1 points. The runs are split into oddsChunks fixed streams, each seeded from the recorded seed and its
// index, so the result is the same on any machine however many cores run it.
const (
	OddsRuns   = 2000
	oddsChunks = 8
)

// OddsPlayer is one rostered player's on-field-now value, from the league's own value read.
type OddsPlayer struct {
	Franchise, Position string
	NowPPG, Age         float64
	AgeKnown, Available bool
}

// OddsValues is the league's value read as the odds reading needs it. Words says why there are none.
type OddsValues struct {
	Players []OddsPlayer
	Words   string
}

// OddsRow is one franchise's power, expected score and playoff chance.
type OddsRow struct {
	Franchise  string  `json:"franchise"`
	Name       string  `json:"name"`
	Rank       int     `json:"rank"`
	Record     string  `json:"record"`
	Power      int     `json:"power"` // 1 is strongest; 0 when not ranked
	PowerScore float64 `json:"powerScore"`
	Expected   float64 `json:"expected"` // expected points in one game
	// PlayoffOdds is the share of played-out seasons that put the franchise in the playoff field, 0 to 1.
	PlayoffOdds *float64 `json:"playoffOdds,omitempty"`
}

// OddsGame is one open game of the coming week and the home side's chance to win it.
type OddsGame struct {
	Week     int     `json:"week"`
	Home     string  `json:"home"`
	Away     string  `json:"away"`
	HomeName string  `json:"homeName"`
	AwayName string  `json:"awayName"`
	HomeWin  float64 `json:"homeWin"`
}

// NativeOdds is win chance, power and playoff odds from the league's own results, schedule, rosters and
// values. Each part says in its words why it is missing; one missing part never blanks the others.
type NativeOdds struct {
	League     string     `json:"league"`
	Name       string     `json:"name"`
	Season     int        `json:"season"`
	Revision   int64      `json:"revision"`
	Status     string     `json:"status"`
	Words      string     `json:"words"`
	PowerWords string     `json:"powerWords"`
	GameWords  string     `json:"gameWords"`
	OddsWords  string     `json:"oddsWords"`
	Week       int        `json:"week"`
	Runs       int        `json:"runs"`
	Seed       uint64     `json:"seed"` // every simulated score and coin flip is drawn from this recorded seed
	Rows       []OddsRow  `json:"rows"`
	Games      []OddsGame `json:"games"`
}

// NativeOddsReading reads the standings and the playoff rules the bracket uses, then wires the existing power
// blend (powerrankings.Blend), expected score (powerrankings.ExpectedPoints) and game chance
// (powerrankings.WinProbability) to them. Playoff odds play the open games out with that same chance, rank each
// season with the league's tie-breakers (competition.RankStandings), break a still-level line with a coin flip
// drawn from the recorded seed, and seed it with the bracket's own field rules (competition.SeedField).
func NativeOddsReading(ctx context.Context, league string, records []Record, src ResultSources, season int,
	now time.Time, values OddsValues) (NativeOdds, error) {
	out := NativeOdds{League: league, Season: season, Revision: int64(len(records)), Status: "Unavailable",
		Rows: []OddsRow{}, Games: []OddsGame{}}
	standings, err := NativeStandingsReading(ctx, league, records, src, season, now)
	if err != nil {
		return out, err
	}
	out.Name = standings.Name
	if standings.Status != "Ranked" {
		out.Words = standings.Words
		out.PowerWords, out.GameWords, out.OddsWords = standings.Words, standings.Words, standings.Words
		return out, nil
	}
	out.Status, out.Words = "Ready", standings.Words
	for _, row := range standings.Computed {
		out.Rows = append(out.Rows, OddsRow{Franchise: row.Franchise, Name: row.Name, Rank: row.Rank, Record: row.Record})
	}
	rules, err := RulesInForce(records, season, ReadingInstant(records, now))
	if err != nil {
		return out, err
	}
	expected, words := expectedPoints(standings, rules.Lineup, values)
	if words != "" {
		out.PowerWords, out.GameWords, out.OddsWords = words, words, words
		return out, nil
	}
	if err := out.power(standings, rules.Lineup, values, expected); err != nil {
		return out, err
	}
	out.games(standings, expected)
	b := bracketReader{league: league, records: records, src: src, season: season, now: now,
		out: &NativeBracket{Status: bracketUnavailable}}
	if err := b.prepare(LineupFacts{}); err != nil {
		return out, err
	}
	// The seed is the league's record count and the season, both held non-negative by the standings read.
	out.Seed = uint64(len(records))<<16 | uint64(season) //nolint:gosec // Checked above: 1920 <= season <= 9999.
	if out.OddsWords = oddsWaiting(standings, rules, b.out); out.OddsWords != "" {
		return out, nil
	}
	field, _, _ := b.fieldInput(standings.Computed)
	field.Hand = b.state.handFields(b.layout)
	made, words := playOutChunks(standings, expected, field, out.Seed)
	if words != "" {
		out.OddsWords = words
		return out, nil
	}
	out.Runs = OddsRuns
	for i := range out.Rows {
		odds := float64(made[out.Rows[i].Franchise]) / OddsRuns
		out.Rows[i].PlayoffOdds = &odds
	}
	out.OddsWords = fmt.Sprintf("%d open games played out %d times under the league's tie-breakers and playoff rules",
		len(standings.counted.open), OddsRuns)
	return out, nil
}

// oddsWaiting says why the playoff odds cannot be read, or nothing when they can.
func oddsWaiting(s NativeStandings, rules leaguerules.RuleSet, bracket *NativeBracket) string {
	switch {
	case bracket.Status != bracketUnavailable:
		return "Playoff odds need the playoff rules: " + bracket.Words
	case s.Through < rules.Schedule.RegularWeeks.Value:
		return fmt.Sprintf("Playoff odds need the whole regular season: the schedule covers only %d of %d weeks",
			s.Through, rules.Schedule.RegularWeeks.Value)
	case s.Override != nil:
		return "The commissioner set the standings by hand, so the season is not played out from the results"
	}
	return ""
}

// expectedPoints is each franchise's expected score in a game: its best legal lineup's on-field-now points
// blended with its points per game so far, as the power board's outlook reads it.
func expectedPoints(s NativeStandings, lineup leaguerules.LineupRules, values OddsValues) (map[string]float64, string) {
	if values.Words != "" {
		return nil, values.Words
	}
	rules, ok := lineupdelta.Rules(lineup)
	if !ok {
		return nil, "This league's starting lineup is not fully set, so no lineup can be counted"
	}
	by := oddsCountable(values.Players)
	out := map[string]float64{}
	for _, row := range s.Computed {
		value, _ := powerrankings.Count("lineup", rules, by[row.Franchise])
		out[row.Franchise] = powerrankings.ExpectedPoints(value, float64(row.PointsFor)/10000, row.Wins+row.Losses+row.Ties)
	}
	return out, ""
}

func oddsCountable(players []OddsPlayer) map[string][]powerrankings.CountValue {
	by := map[string][]powerrankings.CountValue{}
	for _, p := range players {
		if !p.Available || p.Franchise == "" {
			continue
		}
		age := math.NaN()
		if p.AgeKnown {
			age = p.Age
		}
		by[p.Franchise] = append(by[p.Franchise], powerrankings.CountValue{Position: p.Position, Value: p.NowPPG, Age: age})
	}
	return by
}

// power ranks the season view of the power board: lineup value blended with results at the weight the weeks
// played give, results read as all-play when any week was all-played and as points scored otherwise.
func (out *NativeOdds) power(s NativeStandings, lineup leaguerules.LineupRules, values OddsValues,
	expected map[string]float64) error {
	rules, _ := lineupdelta.Rules(lineup)
	by := oddsCountable(values.Players)
	var allPlayGames int
	var maxPF float64
	played := 0
	for _, row := range s.Computed {
		t := s.counted.allPlay[row.Franchise]
		allPlayGames += t.W + t.L + t.T
		maxPF = max(maxPF, float64(row.PointsFor))
		played = max(played, row.Wins+row.Losses+row.Ties)
	}
	inputs := make([]powerrankings.Input, 0, len(s.Computed))
	for _, row := range s.Computed {
		value, age := powerrankings.Count("lineup", rules, by[row.Franchise])
		in := powerrankings.Input{FranchiseID: row.Franchise, RosterValue: value, Age: age}
		switch t := s.counted.allPlay[row.Franchise]; {
		case allPlayGames > 0 && t.W+t.L+t.T > 0:
			in.Performance = (float64(t.W) + float64(t.T)/2) / float64(t.W+t.L+t.T)
		case allPlayGames == 0 && maxPF > 0:
			in.Performance = float64(row.PointsFor) / maxPF
		}
		inputs = append(inputs, in)
	}
	blended, err := powerrankings.Blend(inputs, powerrankings.Weights{Roster: powerrankings.AutoRosterWeight(played)})
	if err != nil {
		return fmt.Errorf("power ranking: %w", err)
	}
	at := map[string]powerrankings.Row{}
	for _, r := range blended {
		at[r.FranchiseID] = r
	}
	for i := range out.Rows {
		r := at[out.Rows[i].Franchise]
		out.Rows[i].Power, out.Rows[i].PowerScore = r.Rank, r.PowerScore
		out.Rows[i].Expected = math.Round(expected[out.Rows[i].Franchise]*100) / 100
	}
	out.PowerWords = fmt.Sprintf("Lineup value blended with results after %d games", played)
	return nil
}

// games gives each game of the first week still open its home side's chance.
func (out *NativeOdds) games(s NativeStandings, expected map[string]float64) {
	if len(s.counted.open) == 0 {
		out.GameWords = "Every regular-season game has a result"
		return
	}
	names := map[string]string{}
	for _, row := range s.Computed {
		names[row.Franchise] = row.Name
	}
	out.Week = s.counted.open[0].Week
	for _, g := range s.counted.open {
		if g.Week != out.Week {
			break
		}
		out.Games = append(out.Games, OddsGame{Week: g.Week, Home: g.Home, Away: g.Away, HomeName: names[g.Home],
			AwayName: names[g.Away], HomeWin: powerrankings.WinProbability(expected[g.Home], expected[g.Away])})
	}
	out.GameWords = fmt.Sprintf("Week %d: each side's expected score from its lineup and its points so far", out.Week)
}

// playOutChunks runs the fixed streams side by side and adds their counts.
func playOutChunks(s NativeStandings, expected map[string]float64, field competition.FieldInput,
	seed uint64) (map[string]int, string) {
	counts := make([]map[string]int, oddsChunks)
	words := make([]string, oddsChunks)
	var wait sync.WaitGroup
	for i := range uint64(oddsChunks) {
		chunk := field
		wait.Go(func() { counts[i], words[i] = playOut(s, expected, chunk, OddsRuns/oddsChunks, seed+i) })
	}
	wait.Wait()
	made := map[string]int{}
	for i := range oddsChunks {
		if words[i] != "" {
			return nil, words[i]
		}
		for id, n := range counts[i] {
			made[id] += n
		}
	}
	return made, ""
}

// playOut counts, for each franchise, the played-out seasons that seed it. Each side's score is drawn around its
// expected score with half the game's variance, so a game's margin has exactly the spread WinProbability reads.
func playOut(s NativeStandings, expected map[string]float64, field competition.FieldInput, runs int,
	seed uint64) (map[string]int, string) {
	rng := newOddsRand(seed)
	spread := powerrankings.GameSpread / math.Sqrt2
	made := map[string]int{}
	games := make([]competition.RankGame, 0, len(s.counted.games)+len(s.counted.open))
	needAllPlay := slices.Contains(s.criteria, leaguerules.AllPlayPercentage)
	for range runs {
		games = append(games[:0], s.counted.games...)
		points, allPlay := maps.Clone(s.counted.points), maps.Clone(s.counted.allPlay)
		var week []FranchiseWeekScore
		for i, g := range s.counted.open {
			home := leaguerules.Decimal(math.Round((expected[g.Home] + rng.NormFloat64()*spread) * 10000))
			away := leaguerules.Decimal(math.Round((expected[g.Away] + rng.NormFloat64()*spread) * 10000))
			games = append(games, competition.RankGame{Home: g.Home, Away: g.Away, HomePoints: home, AwayPoints: away})
			points[g.Home] += home
			points[g.Away] += away
			if needAllPlay {
				week = append(week, FranchiseWeekScore{Franchise: g.Home, Points: &home},
					FranchiseWeekScore{Franchise: g.Away, Points: &away})
				if i+1 == len(s.counted.open) || s.counted.open[i+1].Week != g.Week {
					played := seasonGames{allPlay: allPlay}
					played.addAllPlay(week)
					week = nil
				}
			}
		}
		ranking := competition.RankStandings(competition.RankInput{Teams: s.teams, Games: games, AllPlay: allPlay,
			Points: points, Criteria: s.criteria})
		field.Order = coinFlips(ranking, rng)
		fields, words := competition.SeedField(field)
		if words != "" {
			return nil, words
		}
		for _, f := range fields {
			for _, seed := range f.Seeds {
				made[seed.Franchise]++
			}
		}
	}
	return made, ""
}

// coinFlips breaks each group the full tie-breaker line left level with a coin flip, in place.
func coinFlips(r competition.Ranking, rng *rand.Rand) []string {
	order := slices.Clone(r.Order)
	at := map[string]int{}
	for i, id := range order {
		at[id] = i
	}
	for _, group := range r.Level {
		places := make([]int, 0, len(group))
		for _, id := range group {
			places = append(places, at[id])
		}
		slices.Sort(places)
		flipped := slices.Clone(group)
		rng.Shuffle(len(flipped), func(i, j int) { flipped[i], flipped[j] = flipped[j], flipped[i] })
		for i, place := range places {
			order[place] = flipped[i]
		}
	}
	return order
}

// newOddsRand is the reproducible stream a recorded seed names.
func newOddsRand(seed uint64) *rand.Rand {
	return rand.New(rand.NewPCG(seed, seed^0x9e3779b97f4a7c15)) //nolint:gosec // A recorded, reproducible draw, not a secret.
}
