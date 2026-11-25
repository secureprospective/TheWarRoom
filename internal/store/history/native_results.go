package history

import (
	"context"
	"fmt"
	"slices"
	"sync"
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
	overlay LiveView
	store   *Store
	league  string
	season  int
	now     time.Time
	held    map[source.RawNativeScope]source.NativeEvidence
	files   []heldFile
	gsis    map[string]string
	rules   *commissioner.RulesMemo
	facts   nativecheck.Facts
	// revision is the season score cache's name for the ledger last scored, worked out once per ledger.
	revision *scoreRevisionKey
	// canon, when set, is the league's own ledger read at the start of this request; the score revision
	// uses it instead of reading the ledger again.
	canon []commissioner.Record
}

func (s *Store) nativeSources(league string, season int) *nativeSources {
	return &nativeSources{overlay: s.liveSnapshot(), store: s, league: league, season: season, now: s.now(),
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
	n.playerFacts()
	for i, starter := range lineup.Starters {
		if starter.Name == "" {
			lineup.Starters[i].Name = n.facts[starter.Player].Name
		}
		if starter.Team == "" {
			lineup.Starters[i].Team = n.facts[starter.Player].Team
		}
	}
	return lineup, nil
}

func (n *nativeSources) playerFacts() nativecheck.Facts {
	if n.facts == nil && n.store.lineupEnv != nil {
		n.facts = n.store.lineupEnv.PlayerFacts()
	}
	return n.facts
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

// RulesInForce lets a reading that prices many proposals resolve the held ledger's rules once.
func (n *nativeSources) RulesInForce() commissioner.RulesFunc { return n.rules.InForce }

// PlayerWeekStats selects the player's week from the season's held files, read on first use.
func (n *nativeSources) PlayerWeekStats(ctx context.Context, scope source.RawStatScope) ([]source.HeldWeek, error) {
	if scope.Season != n.season {
		return n.store.PlayerWeekStats(ctx, scope)
	}
	if err := scope.Validate(); err != nil {
		return nil, fmt.Errorf("read held player statistics: %w", err)
	}
	if n.files == nil {
		files, err := n.store.indexedHeldSeason(ctx, scope.Season)
		if err != nil {
			return nil, err
		}
		n.files = files
	}
	return selectHeldWeeks(scope, n.files), nil
}

// NativeResults scores one week of the published schedule.
func (s *Store) NativeResults(ctx context.Context, league string, season, week int) (commissioner.NativeResults, error) {
	// The ledger is read once, as in NativeStandings: its mark names the flight, the reading folds these
	// records, and the score revision checks them as the league's own entries (nativeSources.canon).
	records, err := s.nativeRecords(ctx, league)
	if err != nil {
		return commissioner.NativeResults{}, err
	}
	mark := recordsMark(records)
	src := s.nativeSources(league, season)
	src.canon = records
	args := src.liveKey() + "|" + mark + "|" + fmt.Sprintf("%s|%d|%d", league, season, week)
	starters := league + "|" + mark + "|" + src.weekKey(season, week) + fmt.Sprintf("|%d|%d", season, week)
	return shareReading(ctx, &s.flights, "NativeResults", args,
		func(ctx context.Context) (commissioner.NativeResults, error) {
			return s.readNativeResults(ctx, league, season, week, src, records, starters, mark)
		})
}

func (s *Store) readNativeResults(ctx context.Context, league string, season, week int, src *nativeSources,
	records []commissioner.Record, startersKey, mark string) (commissioner.NativeResults, error) {
	read := func() (commissioner.NativeResults, error) {
		out, err := commissioner.NativeResultsReading(ctx, league, records, src, season, week, src.now)
		if err != nil {
			return commissioner.NativeResults{}, fmt.Errorf("read native results: %w", err)
		}
		s.recordedStarters.apply(ctx, startersKey, s.now(), &out, func() {
			commissioner.RecordedStarters(ctx, records, src, &out)
		})
		return out, nil
	}
	out, err := sharedReading(ctx, s, &s.results, src, records, mark, season, week, read)
	if err != nil {
		return commissioner.NativeResults{}, err
	}
	out.Evidence = src.scoreEvidence(season, week)
	out.Revision = commissioner.ViewRevision(ctx, records, out.Revision)
	return out, nil
}

// NativeStandings ranks a season's regular-season results.
func (s *Store) NativeStandings(ctx context.Context, league string, season int) (commissioner.NativeStandings, error) {
	// The ledger is read once: its mark names the flight, the reading folds these records, and the score
	// revision checks them as the league's own entries (nativeSources.canon) instead of reading them again.
	records, err := s.nativeRecords(ctx, league)
	if err != nil {
		return commissioner.NativeStandings{}, err
	}
	mark := recordsMark(records)
	src := s.nativeSources(league, season)
	src.canon = records
	args := src.liveKey() + "|" + mark + "|" + fmt.Sprintf("%s|%d", league, season)
	return shareReading(ctx, &s.flights, "NativeStandings", args,
		func(ctx context.Context) (commissioner.NativeStandings, error) {
			return s.readNativeStandings(ctx, league, season, src, records, mark)
		})
}

func (s *Store) readNativeStandings(ctx context.Context, league string, season int, src *nativeSources,
	records []commissioner.Record, mark string) (commissioner.NativeStandings, error) {
	read := func() (commissioner.NativeStandings, error) {
		out, err := commissioner.NativeStandingsReading(ctx, league, records, src, season, src.now)
		if err != nil {
			return commissioner.NativeStandings{}, fmt.Errorf("read native standings: %w", err)
		}
		return out, nil
	}
	out, err := sharedReading(ctx, s, &s.standings, src, records, mark, season, 0, read)
	if err != nil {
		return commissioner.NativeStandings{}, err
	}
	out.Evidence = src.scoreEvidence(season, 0)
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
	mark, err := s.recordsMarkFor(ctx, league)
	if err != nil {
		return commissioner.NativeBracket{}, err
	}
	args := mark + "|" + fmt.Sprintf("%s|%d", league, season)
	return shareReading(ctx, &s.flights, "NativeBracket", args,
		func(ctx context.Context) (commissioner.NativeBracket, error) {
			return s.readNativeBracket(ctx, league, season)
		})
}

func (s *Store) readNativeBracket(ctx context.Context, league string, season int) (commissioner.NativeBracket, error) {
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

// NativeOdds reads win chance, power and playoff odds from the league's own results and the values given.
func (s *Store) NativeOdds(ctx context.Context, league string, season int, values commissioner.OddsValues) (commissioner.NativeOdds, error) {
	mark, err := s.recordsMarkFor(ctx, league)
	if err != nil {
		return commissioner.NativeOdds{}, err
	}
	src := s.nativeSources(league, season)
	args := src.liveKey() + "|" + mark + "|" + oddsArgs(league, season, values)
	return shareReading(ctx, &s.flights, "NativeOdds", args,
		func(ctx context.Context) (commissioner.NativeOdds, error) {
			return s.readNativeOdds(ctx, league, season, values, src)
		})
}

func (s *Store) readNativeOdds(ctx context.Context, league string, season int, values commissioner.OddsValues, src *nativeSources) (commissioner.NativeOdds, error) {
	records, err := s.nativeRecords(ctx, league)
	if err != nil {
		return commissioner.NativeOdds{}, err
	}
	out, err := commissioner.NativeOddsReadingCached(ctx, league, records, src, season, src.now, values, &s.oddsCache)
	if err != nil {
		return commissioner.NativeOdds{}, fmt.Errorf("read native odds: %w", err)
	}
	out.Evidence = src.scoreEvidence(season, 0)
	out.Revision = commissioner.ViewRevision(ctx, records, out.Revision)
	return out, nil
}

// recordedStartersTTL bounds how long a finished week's starters are reused, so a late NFL stat
// correction still reaches the screen.
const recordedStartersTTL = time.Hour

// recordedStartersCache keeps each finished (recorded) week's starters. Rebuilding them reloads every
// franchise's lineup and rescores every starter: ~4 s a week on the lab, which made the phone League
// tab take 40 s. The key holds the league records mark and the week's live generation, so a ledger
// change or new scores for that week read fresh.
type recordedStartersCache struct {
	mu      sync.Mutex
	entries map[string]recordedStartersEntry
	flights map[string]*startersFlight
	hits    int
	misses  int
}

type recordedStartersEntry struct {
	at       time.Time
	starters map[string][]commissioner.StarterScore
}

type startersFlight struct {
	done     chan struct{}
	starters map[string][]commissioner.StarterScore
	ok       bool
}

func (c *recordedStartersCache) stats() (hits, misses int) {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.hits, c.misses
}

func hasRecordedResult(scores []commissioner.FranchiseWeekScore) bool {
	return slices.ContainsFunc(scores, func(score commissioner.FranchiseWeekScore) bool {
		return score.RecordedResult != ""
	})
}

func copyRecordedStarters(
	out *commissioner.NativeResults, held map[string][]commissioner.StarterScore,
) {
	for i := range out.Scores {
		score := &out.Scores[i]
		if starters, ok := held[score.Franchise]; ok && score.RecordedResult != "" {
			score.Starters = slices.Clone(starters)
		}
	}
}

func (c *recordedStartersCache) apply(
	ctx context.Context, key string, now time.Time, out *commissioner.NativeResults, build func(),
) {
	if !hasRecordedResult(out.Scores) || ctx.Err() != nil {
		return
	}
	flight := c.begin(ctx, key, now, out)
	if flight == nil {
		return
	}
	finished := false
	defer func() {
		if !finished {
			c.mu.Lock()
			delete(c.flights, key)
			close(flight.done)
			c.mu.Unlock()
		}
	}()
	build()
	if ctx.Err() != nil {
		return
	}
	kept := map[string][]commissioner.StarterScore{}
	for _, score := range out.Scores {
		if score.RecordedResult != "" {
			kept[score.Franchise] = slices.Clone(score.Starters)
		}
	}
	c.mu.Lock()
	if c.entries == nil || len(c.entries) >= 256 {
		c.entries = map[string]recordedStartersEntry{}
	}
	c.entries[key] = recordedStartersEntry{at: now, starters: kept}
	flight.starters, flight.ok = kept, true
	delete(c.flights, key)
	close(flight.done)
	finished = true
	c.mu.Unlock()
}

func (c *recordedStartersCache) begin(
	ctx context.Context, key string, now time.Time, out *commissioner.NativeResults,
) *startersFlight {
	for {
		c.mu.Lock()
		if hit, ok := c.entries[key]; ok && now.Sub(hit.at) < recordedStartersTTL {
			c.hits++
			c.mu.Unlock()
			copyRecordedStarters(out, hit.starters)
			return nil
		}
		if flight := c.flights[key]; flight != nil {
			c.mu.Unlock()
			select {
			case <-flight.done:
				if flight.ok {
					c.mu.Lock()
					c.hits++
					c.mu.Unlock()
					copyRecordedStarters(out, flight.starters)
					return nil
				}
				continue
			case <-ctx.Done():
				return nil
			}
		}
		c.misses++
		flight := &startersFlight{done: make(chan struct{})}
		if c.flights == nil {
			c.flights = map[string]*startersFlight{}
		}
		c.flights[key] = flight
		c.mu.Unlock()
		return flight
	}
}

// WarmCacheStats reports hit and miss counts for the caches warmed at server startup.
type WarmCacheStats struct {
	ScoreHits     int
	ScoreMisses   int
	StarterHits   int
	StarterMisses int
	OddsHits      int
	OddsMisses    int
}

// WarmCacheStats returns cache reuse counters for results, standings, odds and bracket reads.
func (s *Store) WarmCacheStats() WarmCacheStats {
	s.scores.mu.Lock()
	scoreHits, scoreMisses := s.scores.hits, s.scores.misses
	s.scores.mu.Unlock()
	starterHits, starterMisses := s.recordedStarters.stats()
	oddsHits, oddsMisses := s.oddsCache.Stats()
	return WarmCacheStats{
		ScoreHits:     scoreHits,
		ScoreMisses:   scoreMisses,
		StarterHits:   starterHits,
		StarterMisses: starterMisses,
		OddsHits:      oddsHits,
		OddsMisses:    oddsMisses,
	}
}
