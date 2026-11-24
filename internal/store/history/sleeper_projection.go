package history

import (
	"context"
	"fmt"
	"math"
	"sort"
	"strings"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/fantasyscore"
	"github.com/secureprospective/TheWarRoom/internal/fantasyscore/source"
	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
	"github.com/secureprospective/TheWarRoom/internal/nativecheck"
)

const sleeperProjectionURL = "https://api.sleeper.app/projections/nfl/%d/%d?season_type=regular"

// SleeperProjectionStats describes the rows not included in Points and the no-snapshot case.
type SleeperProjectionStats struct {
	Scored   int    `json:"scored"`
	Unmapped int    `json:"unmapped"`
	Note     string `json:"note,omitempty"`
}

// SleeperProjectedPoints scores the newest held Sleeper snapshot under this league's rules, in the engine's
// fixed-point points (10,000 per point). A row that cannot be scored is counted in Unmapped, never an error (R11).
// The archived body remains neutral; points are recalculated on every read.
func (s *Store) SleeperProjectedPoints(ctx context.Context, league string, season, week int) (map[string]leaguerules.Decimal, SleeperProjectionStats, error) {
	points := map[string]leaguerules.Decimal{}
	stats := SleeperProjectionStats{}
	if season < 1920 || season > 9999 || week < 1 || week > 22 {
		return nil, stats, fmt.Errorf("sleeper projections: choose an NFL season and week")
	}
	url := fmt.Sprintf(sleeperProjectionURL, season, week)
	var body []byte
	found, err := s.ReferenceBodies(ctx, url, func(got string, candidate []byte, _ time.Time) bool {
		if got != url {
			return false
		}
		body = candidate
		return true
	})
	if err != nil {
		return nil, stats, fmt.Errorf("sleeper projections: read archive: %w", err)
	}
	if !found {
		stats.Note = fmt.Sprintf("no forecast held for week %d", week)
		return points, stats, nil
	}
	ids, err := s.SourceIDs(ctx, "sleeper")
	if err != nil {
		return nil, stats, fmt.Errorf("sleeper projections: read player identities: %w", err)
	}
	bySleeper := make(map[string]string, len(ids))
	for player, sleeperID := range ids {
		bySleeper[sleeperID] = player
	}
	rows, _, unmapped, err := source.ParseSleeperProjections(body, bySleeper)
	if err != nil {
		return nil, stats, fmt.Errorf("sleeper projections: parse archive: %w", err)
	}
	stats.Unmapped = unmapped
	facts := s.PinnedPlayerFacts()
	records, err := s.nativeRecords(ctx, league)
	if err != nil {
		return nil, stats, err
	}
	rules, err := commissioner.RulesInForce(records, season, s.now())
	if err != nil {
		return nil, stats, fmt.Errorf("sleeper projections: rules: %w", err)
	}
	points, scored, err := scoreSleeperRows(rows, facts, rules.Scoring)
	if err != nil {
		return nil, stats, err
	}
	stats.Scored += scored.Scored
	stats.Unmapped += scored.Unmapped
	return points, stats, nil
}

func scoreSleeperRows(rows []source.SleeperProjection, facts map[string]nativecheck.PlayerFacts, rules leaguerules.ScoringRules) (map[string]leaguerules.Decimal, SleeperProjectionStats, error) {
	points := make(map[string]leaguerules.Decimal, len(rows))
	stats := SleeperProjectionStats{}
	for _, row := range rows {
		position, ok := sleeperScoringPosition(row, facts)
		if !ok {
			stats.Unmapped++
			continue
		}
		score, _, err := fantasyscore.Score(rules, position, row.Line)
		if err != nil || score.State != leaguerules.IsKnown {
			stats.Unmapped++ // one player the rules cannot score never loses the others
			continue
		}
		points[row.CanonicalID] = score.Value
		stats.Scored++
	}
	return points, stats, nil
}

func sleeperScoringPosition(row source.SleeperProjection, facts map[string]nativecheck.PlayerFacts) (leaguerules.Position, bool) {
	if strings.HasPrefix(row.CanonicalID, "DEF:") {
		_, held := facts[row.CanonicalID]
		return "DEF", held
	}
	// The league's own player record wins; the feed's label is only a fallback.
	if position := facts[row.CanonicalID].Position; position != "" {
		return position, true
	}
	return row.Position, row.Position != ""
}

// SleeperMiss compares a Sleeper projection with the league-scored result.
type SleeperMiss struct {
	Player    string
	Projected float64
	Actual    float64
}

// SleeperAccuracyReport summarizes projections for players with actual scores.
type SleeperAccuracyReport struct {
	Players      int
	NoActual     int
	MeanAbsError float64
	Bias         float64
	Misses       []SleeperMiss
}

// SleeperAccuracy compares projections to actual league scores. Decimal units are 1/10000 point.
func (s *Store) SleeperAccuracy(ctx context.Context, league string, season, week int) (SleeperAccuracyReport, error) {
	projected, _, err := s.SleeperProjectedPoints(ctx, league, season, week)
	if err != nil {
		return SleeperAccuracyReport{}, err
	}
	records, err := s.nativeRecords(ctx, league)
	if err != nil {
		return SleeperAccuracyReport{}, err
	}
	src := s.nativeSources(league, season)
	return sleeperAccuracy(projected, func(player string) (*leaguerules.Decimal, error) {
		score, err := src.PlayerScore(ctx, records, player, week)
		if err != nil {
			return nil, fmt.Errorf("sleeper accuracy score: %w", err)
		}
		return score.Points, nil
	})
}

// sleeperAccuracy compares each projection with actual(player); a nil actual means the player did not play.
func sleeperAccuracy(projected map[string]leaguerules.Decimal,
	actual func(string) (*leaguerules.Decimal, error)) (SleeperAccuracyReport, error) {
	report := SleeperAccuracyReport{}
	var absTotal, biasTotal float64
	for player, points := range projected {
		got, err := actual(player)
		if err != nil {
			return SleeperAccuracyReport{}, err
		}
		if got == nil {
			report.NoActual++
			continue
		}
		p, a := float64(points)/10000, float64(*got)/10000
		report.Players++
		absTotal += math.Abs(p - a)
		biasTotal += p - a
		report.Misses = append(report.Misses, SleeperMiss{Player: player, Projected: p, Actual: a})
	}
	if report.Players > 0 {
		report.MeanAbsError = math.Round(absTotal/float64(report.Players)*100) / 100
		report.Bias = math.Round(biasTotal/float64(report.Players)*100) / 100
	}
	sort.Slice(report.Misses, func(i, j int) bool {
		a := math.Abs(report.Misses[i].Projected - report.Misses[i].Actual)
		b := math.Abs(report.Misses[j].Projected - report.Misses[j].Actual)
		if a == b {
			return report.Misses[i].Player < report.Misses[j].Player
		}
		return a > b
	})
	if len(report.Misses) > 10 {
		report.Misses = report.Misses[:10]
	}
	return report, nil
}
