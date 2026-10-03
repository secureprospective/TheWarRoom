package state

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"
)

// league_schedule_cache keeps the last good MFL league schedule (the fantasy matchups), for the
// same reasons and in the same shape as standings_cache.
const leagueScheduleCacheDDL = `
CREATE TABLE IF NOT EXISTS league_schedule_cache (
	league_id  TEXT NOT NULL,
	season     INTEGER NOT NULL,
	payload    TEXT NOT NULL,
	fetched_at TEXT NOT NULL,
	PRIMARY KEY (league_id, season)
);`

// initLeagueScheduleCacheSchema creates the schedule cache table.
func (s *Store) initLeagueScheduleCacheSchema(ctx context.Context) error {
	if _, err := s.pools.Write().ExecContext(ctx, leagueScheduleCacheDDL); err != nil {
		return fmt.Errorf("state: init league-schedule-cache schema: %w", err)
	}
	return nil
}

// ErrNoCachedLeagueSchedule means nothing was ever cached.
var ErrNoCachedLeagueSchedule = errors.New("state: no cached league schedule for this league and season")

// PutLeagueSchedule replaces the cached schedule after a successful fetch; an empty payload is
// refused.
func (s *Store) PutLeagueSchedule(ctx context.Context, payload string, fetchedAt time.Time) error {
	if p := strings.TrimSpace(payload); p == "" || p == "[]" || p == "null" {
		return fmt.Errorf("state: refusing to cache an empty league-schedule payload (%q)", p)
	}
	s.wmu.Lock()
	defer s.wmu.Unlock()
	if _, err := s.pools.Write().ExecContext(ctx, `
INSERT INTO league_schedule_cache (league_id, season, payload, fetched_at)
VALUES (?, ?, ?, ?)
ON CONFLICT (league_id, season) DO UPDATE SET
	payload    = excluded.payload,
	fetched_at = excluded.fetched_at`,
		s.leagueID, s.season, payload, fetchedAt.UTC().Format(time.RFC3339)); err != nil {
		return fmt.Errorf("state: cache league schedule: %w", err)
	}
	return nil
}

// CachedLeagueSchedule returns the cached payload and its fetch time, or
// ErrNoCachedLeagueSchedule.
func (s *Store) CachedLeagueSchedule(ctx context.Context) (string, time.Time, error) {
	var payload, at string
	row := s.pools.Read().QueryRowContext(ctx,
		`SELECT payload, fetched_at FROM league_schedule_cache WHERE league_id = ? AND season = ?`,
		s.leagueID, s.season)
	switch err := row.Scan(&payload, &at); {
	case errors.Is(err, sql.ErrNoRows):
		return "", time.Time{}, fmt.Errorf("season %d: %w", s.season, ErrNoCachedLeagueSchedule)
	case err != nil:
		return "", time.Time{}, fmt.Errorf("state: read cached league schedule: %w", err)
	}
	ts, err := time.Parse(time.RFC3339, at)
	if err != nil {
		return "", time.Time{}, fmt.Errorf(
			"state: cached league schedule has unparseable fetched_at %q: %w", at, err)
	}
	return payload, ts, nil
}
