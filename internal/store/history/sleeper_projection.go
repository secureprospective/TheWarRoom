package history

import (
	"context"
	"fmt"
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

// SleeperProjectedPoints scores the newest held Sleeper snapshot under this league's rules.
// The archived body remains neutral; points are recalculated on every read.
func (s *Store) SleeperProjectedPoints(ctx context.Context, league string, season, week int) (map[string]float64, SleeperProjectionStats, error) {
	points := map[string]float64{}
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

func scoreSleeperRows(rows []source.SleeperProjection, facts map[string]nativecheck.PlayerFacts, rules leaguerules.ScoringRules) (map[string]float64, SleeperProjectionStats, error) {
	points := make(map[string]float64, len(rows))
	stats := SleeperProjectionStats{}
	for _, row := range rows {
		position, ok := sleeperScoringPosition(row, facts)
		if !ok {
			stats.Unmapped++
			continue
		}
		score, _, err := fantasyscore.Score(rules, position, row.Line)
		if err != nil {
			return nil, stats, fmt.Errorf("sleeper projections: score %s: %w", row.CanonicalID, err)
		}
		if score.State != leaguerules.IsKnown {
			return nil, stats, fmt.Errorf("sleeper projections: score %s: %s", row.CanonicalID, score.Citation)
		}
		points[row.CanonicalID] = float64(score.Value) / 10000
		stats.Scored++
	}
	return points, stats, nil
}

func sleeperScoringPosition(row source.SleeperProjection, facts map[string]nativecheck.PlayerFacts) (leaguerules.Position, bool) {
	if strings.HasPrefix(row.CanonicalID, "DEF:") {
		_, held := facts[row.CanonicalID]
		return "DEF", held
	}
	if row.Position != "" {
		return row.Position, true
	}
	position := facts[row.CanonicalID].Position
	return position, position != ""
}
