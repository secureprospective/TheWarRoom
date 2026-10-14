package history

import (
	"context"
	"fmt"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/fantasyscore/source"
)

// LineupSource says which players count for a franchise in a week. No native lineup is recorded yet, so
// until one is supplied every franchise has none and no game has a result.
type LineupSource interface {
	WeekLineup(ctx context.Context, league, franchise string, season, week int) (commissioner.WeekLineup, error)
}

// SetNativeLineups supplies the lineups native results are scored from. Set it at startup, before readings.
func (s *Store) SetNativeLineups(lineups LineupSource) { s.lineups = lineups }

// nativeSources is one reading's view of lineups and player scores. Held evidence does not depend on the
// ledger, so a player's week is read once however many ledgers it is scored under.
type nativeSources struct {
	store  *Store
	league string
	season int
	held   map[source.RawNativeScope]source.NativeEvidence
}

func (s *Store) nativeSources(league string, season int) *nativeSources {
	return &nativeSources{store: s, league: league, season: season,
		held: map[source.RawNativeScope]source.NativeEvidence{}}
}

func (n *nativeSources) Lineup(ctx context.Context, franchise string, week int) (commissioner.WeekLineup, error) {
	if n.store.lineups == nil {
		return commissioner.WeekLineup{}, nil
	}
	lineup, err := n.store.lineups.WeekLineup(ctx, n.league, franchise, n.season, week)
	if err != nil {
		return commissioner.WeekLineup{}, fmt.Errorf("read the franchise lineup: %w", err)
	}
	return lineup, nil
}

func (n *nativeSources) PlayerScore(ctx context.Context, records []commissioner.Record, player string,
	week int) (commissioner.NativePlayerScore, error) {
	scope := source.RawNativeScope{Player: player, Season: n.season, Week: week}
	if err := scope.Validate(); err != nil {
		return commissioner.NativePlayerScore{}, fmt.Errorf("score a starter: %w", err)
	}
	evidence := source.NativeEvidence{}
	if nativeSeeded(records) {
		held, ok := n.held[scope]
		if !ok {
			var err error
			if held, err = n.store.nativeEvidence(ctx, scope); err != nil {
				return commissioner.NativePlayerScore{}, fmt.Errorf("read native score evidence: %w", err)
			}
			n.held[scope] = held
		}
		evidence = held
	}
	reading, err := commissioner.NativePlayerScoreReading(n.league, scope, records, evidence, n.store.now())
	if err != nil {
		return commissioner.NativePlayerScore{}, fmt.Errorf("read native score: %w", err)
	}
	return reading, nil
}

// NativeResults scores one week of the published schedule.
func (s *Store) NativeResults(ctx context.Context, league string, season, week int) (commissioner.NativeResults, error) {
	records, err := s.nativeRecords(ctx, league)
	if err != nil {
		return commissioner.NativeResults{}, err
	}
	out, err := commissioner.NativeResultsReading(ctx, league, records, s.nativeSources(league, season), season, week, s.now())
	if err != nil {
		return commissioner.NativeResults{}, fmt.Errorf("read native results: %w", err)
	}
	return out, nil
}

// NativeStandings ranks a season's regular-season results.
func (s *Store) NativeStandings(ctx context.Context, league string, season int) (commissioner.NativeStandings, error) {
	records, err := s.nativeRecords(ctx, league)
	if err != nil {
		return commissioner.NativeStandings{}, err
	}
	out, err := commissioner.NativeStandingsReading(ctx, league, records, s.nativeSources(league, season), season, s.now())
	if err != nil {
		return commissioner.NativeStandings{}, fmt.Errorf("read native standings: %w", err)
	}
	return out, nil
}

// NativeScorePreview warns when a score change would turn over a game's result.
func (s *Store) NativeScorePreview(ctx context.Context, raw commissioner.RawCommand) (commissioner.ScorePreview, error) {
	records, err := s.nativeRecords(ctx, raw.League)
	if err != nil {
		return commissioner.ScorePreview{}, err
	}
	if raw.Args.Score == nil {
		return commissioner.ScorePreview{}, fmt.Errorf("preview a score change")
	}
	out, err := commissioner.ScoreChangePreview(ctx, raw.League, records, s.nativeSources(raw.League, raw.Args.Score.Season),
		raw, s.now())
	if err != nil {
		return commissioner.ScorePreview{}, fmt.Errorf("preview score change: %w", err)
	}
	return out, nil
}
