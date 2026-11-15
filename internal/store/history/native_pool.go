package history

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
	"github.com/secureprospective/TheWarRoom/internal/leagueweek"
)

// Pool totals retain only one revision; changed rules, evidence or finished weeks replace it.
func (s *Store) nativePoolPoints(ctx context.Context, reading *commissioner.NativeAssets, records []commissioner.Record) error {
	season, finished, err := s.poolFinishedWeeks(ctx, reading.Snapshot.Season)
	if err != nil {
		return err
	}
	if len(finished) == 0 {
		return nil
	}
	src := s.nativeSources(reading.League, season)
	key, err := src.poolScoreKey(ctx, records, finished)
	if err != nil {
		return err
	}
	s.poolMu.Lock()
	defer s.poolMu.Unlock()
	if s.poolKey != key || s.poolPoints == nil {
		totals, err := s.scorePoolPlayers(ctx, reading, records, src, finished)
		if err != nil {
			return err
		}
		s.poolKey, s.poolPoints = key, totals
	}
	for i, player := range reading.Pool {
		if points, held := s.poolPoints[player.Player]; held {
			reading.Pool[i].Points = fmt.Sprintf("%.1f", float64(points)/10000)
		}
	}
	return nil
}

func (s *Store) scorePoolPlayers(ctx context.Context, reading *commissioner.NativeAssets, records []commissioner.Record, src *nativeSources, weeks []int) (map[string]leaguerules.Decimal, error) {
	files, err := s.indexedHeldSeason(ctx, src.season)
	if pinned, ok := s.lineupEnv.(*PinnedFootball); ok {
		files, err = pinned.stats, nil
	}
	if err != nil {
		return nil, fmt.Errorf("pool points NFL statistics: %w", err)
	}
	gsis, canonical := map[string]bool{}, map[string]bool{}
	for _, file := range files {
		for _, week := range file.weeks {
			if week.SeasonType == "REG" {
				gsis[week.GSIS] = true
			}
		}
		for key := range file.canonical {
			canonical[key.player] = true
		}
	}
	totals := map[string]leaguerules.Decimal{}
	for _, player := range reading.Pool {
		id, err := src.SourceID(ctx, player.Player, "gsis")
		if err != nil {
			return nil, fmt.Errorf("pool points identity: %w", err)
		}
		if !gsis[id] && !canonical[player.Player] {
			continue
		}
		for _, week := range weeks {
			score, err := src.PlayerScore(ctx, records, player.Player, week)
			if err != nil {
				return nil, fmt.Errorf("pool points scoring: %w", err)
			}
			if score.Points != nil {
				totals[player.Player] += *score.Points
			}
		}
	}
	return totals, nil
}

func (s *Store) poolFinishedWeeks(ctx context.Context, season int) (int, []int, error) {
	schedule, ok := s.lineupEnv.(interface {
		ScheduleDates(context.Context, int) ([]leagueweek.Week, error)
	})
	if !ok {
		return season, nil, nil
	}
	weeks, err := schedule.ScheduleDates(ctx, season)
	if err != nil {
		return season, nil, fmt.Errorf("pool points schedule: %w", err)
	}
	started := false
	for _, week := range weeks {
		for _, game := range week.Games {
			started = started || !game.Kickoff.After(s.now())
		}
	}
	if !started {
		season--
		weeks, err = schedule.ScheduleDates(ctx, season)
		if err != nil {
			return season, nil, fmt.Errorf("pool points previous season: %w", err)
		}
	}
	finished := []int{}
	for _, week := range weeks {
		if len(week.Games) == 0 {
			continue
		}
		if _, open := leagueweek.LineupWeek(s.now(), week); !open {
			finished = append(finished, week.Number)
		}
	}
	return season, finished, nil
}

// Assets supplies a durable ledger, so the database revision covers archive and identity changes
// without the trial-ledger admission required by the shared franchise-week score cache.
func (n *nativeSources) poolScoreKey(ctx context.Context, records []commissioner.Record, weeks []int) (string, error) {
	revision, err := n.store.pools.Revision(ctx)
	if err != nil {
		return "", fmt.Errorf("pool points revision: %w", err)
	}
	rules, err := n.rules.InForce(records, n.season, commissioner.ReadingInstant(records, n.now))
	if err != nil {
		return "", fmt.Errorf("pool points rules: %w", err)
	}
	body, err := json.Marshal(rules)
	if err != nil {
		return "", fmt.Errorf("pool points rules digest: %w", err)
	}
	return fmt.Sprintf("%s|%d|%d|%s|%x|%s|%v", n.league, n.season, revision, recordsMark(records), sha256.Sum256(body), factsDigest(n.store.lineupEnv, n.playerFacts()), weeks), nil
}
