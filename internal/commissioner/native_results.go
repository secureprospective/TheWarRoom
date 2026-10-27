package commissioner

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/engine/competition"
	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
	"github.com/secureprospective/TheWarRoom/internal/nativecheck"
)

// NativeGame is one published head-to-head game and, when both scores are complete, its result.
type NativeGame struct {
	Week       int                  `json:"week"`
	Home       string               `json:"home"`
	Away       string               `json:"away"`
	HomeName   string               `json:"homeName"`
	AwayName   string               `json:"awayName"`
	Status     string               `json:"status"`
	HomePoints *leaguerules.Decimal `json:"homePoints,omitempty"`
	AwayPoints *leaguerules.Decimal `json:"awayPoints,omitempty"`
	Winner     string               `json:"winner,omitempty"`
	Words      string               `json:"words"`
	Why        []string             `json:"why"`
}

// NativeResults is one week of the published schedule scored under the league's own rules.
type NativeResults struct {
	League    string               `json:"league"`
	Season    int                  `json:"season"`
	Week      int                  `json:"week"`
	Revision  int64                `json:"revision"`
	Published bool                 `json:"published"`
	Words     string               `json:"words"`
	Games     []NativeGame         `json:"games"`
	Bye       string               `json:"bye,omitempty"`
	ByeName   string               `json:"byeName,omitempty"`
	Scores    []FranchiseWeekScore `json:"scores"`
	// Missed lists the franchises with no lineup this week, flagged to the commissioner when the league
	// flags them. Nothing is taken from them.
	Missed []MissedLineup `json:"missed"`
}

// MissedLineup is a franchise that set no lineup for a week.
type MissedLineup struct {
	Franchise string `json:"franchise"`
	Name      string `json:"name"`
	Flagged   bool   `json:"flagged"`
}

// NativeResultsReading scores a published week. Nothing is guessed: a game missing either score has no
// result yet and says why.
func NativeResultsReading(ctx context.Context, league string, records []Record, src ResultSources, season, week int,
	now time.Time) (NativeResults, error) {
	if week < 1 || week > 22 || season < 1920 || season > 9999 {
		return NativeResults{}, fmt.Errorf("choose a season and week")
	}
	directory, err := NativeDirectoryReading(league, records, now)
	if err != nil {
		return NativeResults{}, err
	}
	published, err := resultSchedule(records, season, now)
	if err != nil {
		return NativeResults{}, err
	}
	rules, err := RulesInForce(records, season, ReadingInstant(records, now))
	if err != nil {
		return NativeResults{}, err
	}
	return weekResults(ctx, league, records, src, rules.Lineup, directory, published, season, week)
}

func weekResults(ctx context.Context, league string, records []Record, src ResultSources, lineup leaguerules.LineupRules,
	directory NativeDirectory, published []ScheduleWeek, season, week int) (NativeResults, error) {
	out := NativeResults{League: league, Season: season, Week: week, Revision: int64(len(records)),
		Published: len(published) != 0, Games: []NativeGame{}, Scores: []FranchiseWeekScore{}, Missed: []MissedLineup{}}
	if !out.Published {
		out.Words = "The league schedule is not published yet, so there are no games to score"
		return out, nil
	}
	index := slices.IndexFunc(published, func(w ScheduleWeek) bool { return w.Week == week })
	if index < 0 {
		out.Words = fmt.Sprintf("Week %d is not in the published schedule", week)
		return out, nil
	}
	names := directory.Seats()
	scored := map[string]FranchiseWeekScore{}
	for _, game := range published[index].Games {
		for _, id := range []string{game.Home, game.Away} {
			score, err := franchiseWeekScore(ctx, records, src, lineup, names.Of(id), season, week)
			if err != nil {
				return NativeResults{}, err
			}
			scored[id] = score
			out.Scores = append(out.Scores, score)
		}
		out.Games = append(out.Games, gameResult(week, names.Of(game.Home), names.Of(game.Away),
			scored[game.Home], scored[game.Away]))
	}
	if bye := published[index].Bye; bye != "" {
		out.Bye, out.ByeName = bye, names.Of(bye).Name
		score, err := franchiseWeekScore(ctx, records, src, lineup, names.Of(bye), season, week)
		if err != nil {
			return NativeResults{}, err
		}
		out.Scores = append(out.Scores, score)
	}
	for _, score := range out.Scores {
		if score.Lineup.Status == nativecheck.LineupMissed {
			out.Missed = append(out.Missed, MissedLineup{Franchise: score.Franchise, Name: score.Name, Flagged: score.Lineup.Flagged})
		}
	}
	out.Words = resultsWords(out)
	return out, nil
}

func resultsWords(r NativeResults) string {
	decided := 0
	for _, g := range r.Games {
		if g.Status != GameNoResult {
			decided++
		}
	}
	return fmt.Sprintf("%d of %d games in week %d have a result", decided, len(r.Games), r.Week)
}

// gameResult turns two franchise scores into a result. Either score being incomplete leaves the game
// without one.
func gameResult(week int, home, away NativeDirectorySeat, hs, as FranchiseWeekScore) NativeGame {
	game := NativeGame{Week: week, Home: home.Franchise, Away: away.Franchise, HomeName: home.Name,
		AwayName: away.Name, Status: GameNoResult, Why: []string{}}
	for _, s := range []FranchiseWeekScore{hs, as} {
		if s.Points == nil {
			game.Why = append(game.Why, s.Words)
		}
	}
	if len(game.Why) != 0 {
		game.Words = "No result yet: " + strings.Join(game.Why, " ")
		return game
	}
	game.HomePoints, game.AwayPoints = hs.Points, as.Points
	switch {
	case *hs.Points == *as.Points:
		game.Status = GameTie
		game.Words = fmt.Sprintf("%s and %s tied at %s", home.Name, away.Name, scorePointsWords(*hs.Points))
	case *hs.Points > *as.Points:
		game.Status, game.Winner = GameFinal, home.Franchise
		game.Words = finalWords(home.Name, away.Name, *hs.Points, *as.Points)
	default:
		game.Status, game.Winner = GameFinal, away.Franchise
		game.Words = finalWords(away.Name, home.Name, *as.Points, *hs.Points)
	}
	return game
}

// A closed season keeps its published schedule; later seasons' edits do not rewrite its games.
func resultSchedule(records []Record, season int, now time.Time) ([]ScheduleWeek, error) {
	at := ReadingInstant(records, now)
	if _, err := RulesInForce(records, season, at); err != nil {
		return nil, err
	}
	starts, err := RolloverInstants(records)
	if err != nil {
		return nil, err
	}
	first, err := firstSeason(records)
	if err != nil {
		return nil, err
	}
	if index := season - first; index < len(starts) && !at.Before(starts[index]) {
		at = starts[index].Add(-time.Nanosecond)
	}
	_, published, err := replaySchedule(RecordsThrough(records, at))
	return published, err
}

// The competition judgment (tie-breakers, playoff fields and brackets, schedule generation) lives in
// internal/engine/competition. These names keep the commissioner API and its wire shapes unchanged.
type (
	ScheduleWeek    = competition.ScheduleWeek
	ScheduleGame    = competition.ScheduleGame
	PlayoffRules    = competition.PlayoffRules
	BracketRound    = competition.BracketRound
	BracketSide     = competition.BracketSide
	BracketGame     = competition.BracketGame
	ConferenceField = competition.ConferenceField
	FieldSeed       = competition.FieldSeed
)

// Game statuses: a result needs both scores complete; a level score is a tie, never a coin toss.
const (
	GameFinal    = competition.GameFinal
	GameTie      = competition.GameTie
	GameNoResult = competition.GameNoResult
)

// ResolvePlayoffRules reads a season's playoff rules from its rules and bracket shape.
func ResolvePlayoffRules(rules leaguerules.RuleSet, graph []leaguerules.BracketNode) PlayoffRules {
	return competition.ResolvePlayoffRules(rules, graph)
}

func addScoreDelta(a, b leaguerules.Decimal) (leaguerules.Decimal, error) {
	return competition.AddPoints(a, b) //nolint:wrapcheck // Same package family; the refusal text is the message.
}

func scorePointsWords(points leaguerules.Decimal) string { return competition.PointsWords(points) }

func finalWords(winner, loser string, high, low leaguerules.Decimal) string {
	return competition.FinalWords(winner, loser, high, low)
}
