package history

import (
	"context"
	"fmt"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/fantasyscore/source"
	"github.com/secureprospective/TheWarRoom/internal/nativecheck"
)

// LineupSource says which players count for a franchise in a week, from the ledger it is given so that a
// proposed change can be read as it would stand.
type LineupSource interface {
	WeekLineup(ctx context.Context, records []commissioner.Record, league, franchise string, season, week int) (commissioner.WeekLineup, error)
}

// ledgerLineups is the durable source: the lineups the league recorded, owners' and commissioners', folded
// from its ledger. A franchise that recorded none has none, and nothing is read from any outside league.
type ledgerLineups struct{}

func (ledgerLineups) WeekLineup(_ context.Context, records []commissioner.Record, _, franchise string, season,
	week int) (commissioner.WeekLineup, error) {
	lineup, err := commissioner.LedgerWeekLineup(records, franchise, season, week)
	if err != nil {
		return commissioner.WeekLineup{}, fmt.Errorf("read recorded lineup: %w", err)
	}
	return lineup, nil
}

// SetNativeLineups replaces the lineup source native results are scored from; nil restores the ledger.
// Tests use it to score a league without recording lineups.
func (s *Store) SetNativeLineups(lineups LineupSource) {
	if lineups == nil {
		lineups = ledgerLineups{}
	}
	s.lineups = lineups
}

// nativeSources is one reading's view of lineups and player scores. Held evidence does not depend on the
// ledger, so a player's week is read once however many ledgers it is scored under.
type nativeSources struct {
	store  *Store
	league string
	season int
	held   map[source.RawNativeScope]source.NativeEvidence
	facts  nativecheck.Facts
}

func (s *Store) nativeSources(league string, season int) *nativeSources {
	return &nativeSources{store: s, league: league, season: season,
		held: map[source.RawNativeScope]source.NativeEvidence{}}
}

func (n *nativeSources) Lineup(ctx context.Context, records []commissioner.Record, franchise string,
	week int) (commissioner.WeekLineup, error) {
	lineup, err := n.store.lineups.WeekLineup(ctx, records, n.league, franchise, n.season, week)
	if err != nil {
		return commissioner.WeekLineup{}, fmt.Errorf("read the franchise lineup: %w", err)
	}
	if n.store.lineupEnv == nil {
		return lineup, nil
	}
	if n.facts == nil {
		n.facts = n.store.lineupEnv.PlayerFacts()
	}
	for i, starter := range lineup.Starters {
		if starter.Name == "" {
			lineup.Starters[i].Name = n.facts[starter.Player].Name
		}
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

// NativeScorePreview warns when a score change or a lineup correction would turn over a game's result.
func (s *Store) NativeScorePreview(ctx context.Context, raw commissioner.RawCommand) (commissioner.ScorePreview, error) {
	records, err := s.nativeRecords(ctx, raw.League)
	if err != nil {
		return commissioner.ScorePreview{}, err
	}
	if raw.Command == commissioner.LineupCorrect {
		return s.lineupCorrectionPreview(ctx, records, raw)
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
