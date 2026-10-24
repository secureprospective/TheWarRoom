package history

import (
	"context"
	"fmt"
	"time"

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
// ledger, so a player's week is read once however many ledgers it is scored under. The reading's clock is
// taken once and its rules are resolved once per ledger, however many players are scored; a ledger with a
// proposed change in it is a different ledger and resolves for itself. The season's held weekly files are
// read once for the reading, not once per starter.
type nativeSources struct {
	store  *Store
	league string
	season int
	now    time.Time
	held   map[source.RawNativeScope]source.NativeEvidence
	files  []heldFile
	rules  *commissioner.RulesMemo
	facts  nativecheck.Facts
}

func (s *Store) nativeSources(league string, season int) *nativeSources {
	return &nativeSources{store: s, league: league, season: season, now: s.now(),
		held: map[source.RawNativeScope]source.NativeEvidence{}, rules: commissioner.NewRulesMemo()}
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
			if held, err = n.store.nativeEvidence(ctx, n, scope); err != nil {
				return commissioner.NativePlayerScore{}, fmt.Errorf("read native score evidence: %w", err)
			}
			n.held[scope] = held
		}
		evidence = held
	}
	reading, err := n.rules.Score(n.league, scope, records, evidence, n.now)
	if err != nil {
		return commissioner.NativePlayerScore{}, fmt.Errorf("read native score: %w", err)
	}
	return reading, nil
}

// PlayerWeekStats selects the player's week from the season's held files, read on first use.
func (n *nativeSources) PlayerWeekStats(ctx context.Context, scope source.RawStatScope) ([]source.HeldWeek, error) {
	if scope.Season != n.season {
		return n.store.PlayerWeekStats(ctx, scope)
	}
	if err := scope.Validate(); err != nil {
		return nil, fmt.Errorf("read held player statistics: %w", err)
	}
	if n.files == nil {
		files, err := n.store.heldSeason(ctx, scope.Season)
		if err != nil {
			return nil, err
		}
		for i := range files {
			files[i].index = source.IndexHeldWeeks(files[i].weeks)
		}
		n.files = files
	}
	return selectHeldWeeks(scope, n.files), nil
}

// NativeResults scores one week of the published schedule.
func (s *Store) NativeResults(ctx context.Context, league string, season, week int) (commissioner.NativeResults, error) {
	records, err := s.nativeRecords(ctx, league)
	if err != nil {
		return commissioner.NativeResults{}, err
	}
	src := s.nativeSources(league, season)
	out, err := commissioner.NativeResultsReading(ctx, league, records, src, season, week, src.now)
	if err != nil {
		return commissioner.NativeResults{}, fmt.Errorf("read native results: %w", err)
	}
	out.Revision = commissioner.ViewRevision(ctx, records, out.Revision)
	return out, nil
}

// NativeStandings ranks a season's regular-season results.
func (s *Store) NativeStandings(ctx context.Context, league string, season int) (commissioner.NativeStandings, error) {
	records, err := s.nativeRecords(ctx, league)
	if err != nil {
		return commissioner.NativeStandings{}, err
	}
	src := s.nativeSources(league, season)
	out, err := commissioner.NativeStandingsReading(ctx, league, records, src, season, src.now)
	if err != nil {
		return commissioner.NativeStandings{}, fmt.Errorf("read native standings: %w", err)
	}
	out.Revision = commissioner.ViewRevision(ctx, records, out.Revision)
	return out, nil
}

// NativeScorePreview warns when a score change, a lineup correction or a statistic decision would turn over a
// game's result.
func (s *Store) NativeScorePreview(ctx context.Context, raw commissioner.RawCommand) (commissioner.ScorePreview, error) {
	records, err := s.nativeRecords(ctx, raw.League)
	if err != nil {
		return commissioner.ScorePreview{}, err
	}
	if raw.Command == commissioner.LineupCorrect {
		return s.lineupCorrectionPreview(ctx, records, raw)
	}
	if commissioner.IsStatCommand(raw.Command) {
		return s.statChangePreview(ctx, records, raw)
	}
	if raw.Args.Score == nil {
		return commissioner.ScorePreview{}, fmt.Errorf("preview a score change")
	}
	src := s.nativeSources(raw.League, raw.Args.Score.Season)
	out, err := commissioner.ScoreChangePreview(ctx, raw.League, records, src, raw, src.now)
	if err != nil {
		return commissioner.ScorePreview{}, fmt.Errorf("preview score change: %w", err)
	}
	return out, nil
}

// NativeBracket reads a season's playoffs. The first playoff week's kickoffs are added when the app holds
// them, so the reading can say whether the playoffs have begun.
func (s *Store) NativeBracket(ctx context.Context, league string, season int) (commissioner.NativeBracket, error) {
	records, err := s.nativeRecords(ctx, league)
	if err != nil {
		return commissioner.NativeBracket{}, err
	}
	facts := commissioner.LineupFacts{}
	if week, ok := commissioner.PlayoffStartWeek(records, season, s.now()); ok && s.lineupEnv != nil {
		if facts, err = s.weekFacts(ctx, season, week); err != nil {
			return commissioner.NativeBracket{}, err
		}
	}
	src := s.nativeSources(league, season)
	out, err := commissioner.NativeBracketReading(ctx, league, records, src, season, src.now, facts)
	if err != nil {
		return commissioner.NativeBracket{}, fmt.Errorf("read native bracket: %w", err)
	}
	out.Revision = commissioner.ViewRevision(ctx, records, out.Revision)
	return out, nil
}

// rolloverChampion resolves the final and its history link under the same append lock as the rollover.
func (s *Store) rolloverChampion(ctx context.Context, raw commissioner.RawCommand, records []commissioner.Record,
	now time.Time) (*commissioner.BracketChampion, error) {
	if raw.Command != commissioner.SeasonRollover {
		return nil, nil
	}
	season, err := commissioner.SeasonAt(records, commissioner.ReadingInstant(records, now))
	if err != nil {
		return nil, fmt.Errorf("read closing season: %w", err)
	}
	src := s.nativeSources(raw.League, season)
	bracket, err := commissioner.NativeBracketReading(ctx, raw.League, records, src, season, now, commissioner.LineupFacts{})
	if err != nil {
		return nil, fmt.Errorf("read closing champion: %w", err)
	}
	return bracket.Champion, nil
}

func (s *Store) prepareRollover(ctx context.Context, raw commissioner.RawCommand, records []commissioner.Record,
	now time.Time) (*commissioner.RolloverBefore, *commissioner.RolloverPlan, error) {
	champion, err := s.rolloverChampion(ctx, raw, records, now)
	if err != nil {
		return nil, nil, err
	}
	before, plan, err := commissioner.PrepareRollover(raw, records, now, champion)
	if err != nil {
		return before, plan, fmt.Errorf("prepare season rollover: %w", err)
	}
	return before, plan, nil
}
