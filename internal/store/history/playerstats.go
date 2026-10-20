package history

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"maps"
	"slices"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"

	"github.com/secureprospective/TheWarRoom/internal/fantasyscore/source"
)

// heldWeeklyFiles offers each held copy of a season's weekly file to accept, newest first, parsed once. A
// refetch of the body just offered is not offered again; a return to an earlier body is, because the
// feed really did change back.
func (s *Store) heldWeeklyFiles(ctx context.Context, season int,
	accept func(sha string, at time.Time, weeks []source.PlayerWeek)) error {
	feed, ok := s.reg.Feed("stats_player_week")
	if !ok {
		return fmt.Errorf("weekly NFL statistics feed is not registered")
	}
	if season < feed.FirstSeason {
		return fmt.Errorf("%w", source.ErrSeasonNotLoaded)
	}
	wanted := feed.URLFor(season)
	var parseErr error
	var previous string
	_, err := s.ExactArchivedBodies(ctx, wanted, func(fetchedURL string, body []byte, at time.Time) bool {
		if fetchedURL != wanted {
			return false
		}
		hash := sha256.Sum256(body)
		sha := hex.EncodeToString(hash[:])
		if sha == previous {
			return false
		}
		previous = sha
		weeks, err := s.decodedWeeks(sha, body)
		if err != nil {
			parseErr = fmt.Errorf("read held weekly statistics %s: %w", sha, err)
			return true
		}
		accept(sha, at, weeks)
		return false
	})
	if err != nil {
		return fmt.Errorf("read weekly statistics archive: %w", err)
	}
	return parseErr
}

// PlayerWeekStats reads the original weekly files, not the reduced floating-point observations.
// Play-by-play is not loaded by Signals; Missing keeps those required inputs explicit.
func (s *Store) PlayerWeekStats(ctx context.Context, scope source.RawStatScope) ([]source.HeldWeek, error) {
	if err := scope.Validate(); err != nil {
		return nil, fmt.Errorf("read held player statistics: %w", err)
	}
	out := []source.HeldWeek{}
	err := s.heldWeeklyFiles(ctx, scope.Season, func(sha string, at time.Time, weeks []source.PlayerWeek) {
		held := source.SelectHeldWeek(scope, weeks, sha, at)
		if held.Week != nil {
			week := clonePlayerWeek(*held.Week)
			held.Week = &week
		}
		out = append(out, held)
	})
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (s *Store) NativePlayerScore(ctx context.Context, league string,
	scope source.RawNativeScope) (commissioner.NativePlayerScore, error) {
	if err := scope.Validate(); err != nil {
		return commissioner.NativePlayerScore{}, fmt.Errorf("read player score: %w", err)
	}
	records, err := s.nativeRecords(ctx, league)
	if err != nil {
		return commissioner.NativePlayerScore{}, err
	}
	return s.scorePlayer(ctx, league, records, scope)
}

// scorePlayer scores one player under the given ledger. An unseeded league reads no NFL statistics, so
// an ordinal season is never taken for an NFL one.
func (s *Store) scorePlayer(ctx context.Context, league string, records []commissioner.Record,
	scope source.RawNativeScope) (commissioner.NativePlayerScore, error) {
	evidence := source.NativeEvidence{}
	if nativeSeeded(records) {
		var err error
		if evidence, err = s.nativeEvidence(ctx, scope); err != nil {
			return commissioner.NativePlayerScore{}, fmt.Errorf("read native score evidence: %w", err)
		}
	}
	reading, err := commissioner.NativePlayerScoreReading(league, scope, records, evidence, s.now())
	if err != nil {
		return commissioner.NativePlayerScore{}, fmt.Errorf("read native score: %w", err)
	}
	return reading, nil
}

func nativeSeeded(records []commissioner.Record) bool {
	for _, record := range records {
		if record.Command.Command == commissioner.NativeSeed {
			return true
		}
	}
	return false
}

// nativeEvidence reads the held NFL evidence for a player's week; tests replace it with supplied evidence.
func (s *Store) nativeEvidence(ctx context.Context, scope source.RawNativeScope) (source.NativeEvidence, error) {
	if s.evidence != nil {
		return s.evidence(ctx, scope)
	}
	out, err := source.ReadNativeEvidence(ctx, s, s, scope)
	if err != nil {
		return out, fmt.Errorf("read held native evidence: %w", err)
	}
	return out, nil
}

type weeklyFile struct {
	weeks []source.PlayerWeek
	err   error
}

func (s *Store) decodedWeeks(sha string, body []byte) ([]source.PlayerWeek, error) {
	s.weeklyMu.Lock()
	defer s.weeklyMu.Unlock()
	if file, ok := s.weekly[sha]; ok {
		return file.weeks, file.err
	}
	weeks, err := source.ReadWeeks(bytes.NewReader(body))
	if err != nil {
		err = fmt.Errorf("decode weekly statistics: %w", err)
	}
	if s.weekly == nil {
		s.weekly = map[string]weeklyFile{}
	}
	s.weekly[sha] = weeklyFile{weeks: weeks, err: err}
	return weeks, err
}

func clonePlayerWeek(week source.PlayerWeek) source.PlayerWeek {
	week.Line.Quantities = maps.Clone(week.Line.Quantities)
	week.Line.Labels = maps.Clone(week.Line.Labels)
	week.Line.Events = maps.Clone(week.Line.Events)
	for stat, events := range week.Line.Events {
		week.Line.Events[stat] = slices.Clone(events)
	}
	return week
}
