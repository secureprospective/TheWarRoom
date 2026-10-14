package history

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"

	"github.com/secureprospective/TheWarRoom/internal/fantasyscore/source"
)

// PlayerWeekStats reads the original weekly files, not the reduced floating-point observations.
// Play-by-play is not loaded by Signals; Missing keeps those required inputs explicit.
func (s *Store) PlayerWeekStats(ctx context.Context, scope source.RawStatScope) ([]source.HeldWeek, error) {
	if err := scope.Validate(); err != nil {
		return nil, fmt.Errorf("read held player statistics: %w", err)
	}
	feed, ok := s.reg.Feed("stats_player_week")
	if !ok {
		return nil, fmt.Errorf("weekly NFL statistics feed is not registered")
	}
	if scope.Season < feed.FirstSeason {
		return nil, fmt.Errorf("weekly NFL statistics are not loaded for this season")
	}
	wanted := feed.URLFor(scope.Season)
	out := []source.HeldWeek{}
	seen := make(map[string]bool)
	var parseErr error
	_, err := s.ArchivedBodies(ctx, wanted, func(fetchedURL string, body []byte, at time.Time) bool {
		if fetchedURL != wanted {
			return false
		}
		hash := sha256.Sum256(body)
		sha := hex.EncodeToString(hash[:])
		if seen[sha] {
			return false
		}
		seen[sha] = true
		weeks, err := source.ReadWeeks(bytes.NewReader(body))
		if err != nil {
			parseErr = fmt.Errorf("read held weekly statistics %s: %w", sha, err)
			return true
		}
		out = append(out, source.SelectHeldWeek(scope, weeks, sha, at))
		return false
	})
	if err != nil {
		return nil, fmt.Errorf("read weekly statistics archive: %w", err)
	}
	if parseErr != nil {
		return nil, parseErr
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
