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
	seeded := false
	for _, record := range records {
		seeded = seeded || record.Command.Command == commissioner.NativeSeed
	}
	if !seeded {
		reading, err := commissioner.NativePlayerScoreReading(league, scope, records, source.NativeEvidence{}, s.now())
		if err != nil {
			return reading, fmt.Errorf("read native score: %w", err)
		}
		return reading, nil
	}
	evidence, err := source.ReadNativeEvidence(ctx, s, s, scope)
	if err != nil {
		return commissioner.NativePlayerScore{}, fmt.Errorf("read native score evidence: %w", err)
	}
	reading, err := commissioner.NativePlayerScoreReading(league, scope, records, evidence, s.now())
	if err != nil {
		return commissioner.NativePlayerScore{}, fmt.Errorf("read native score: %w", err)
	}
	return reading, nil
}
