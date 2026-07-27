package state

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"
)

// league_schedule_cache is the LAST-KNOWN-GOOD store for MFL `schedule` payloads — the
// fantasy league's own weekly matchups (who plays whom), not the standings. It follows the
// exact same shape as standings_cache.go for the exact same reason: a re-fetchable copy of
// someone ELSE's data (MFL's) that we did not author and cannot be asked to account for.
//
// THIS TABLE IS DELIBERATELY NOT APPEND-ONLY, matching standings_cache — it is the second
// mutable table in this package, both cache-not-ledger. Keeping only the newest row per
// (league, season) is correct: there is no audit value in a schedule pull from an hour ago,
// and an append-only cache would grow without bound on every refresh.
//
// payload is the fetcher's raw []RawScheduleWeek re-encoded as JSON, stored VERBATIM at the
// shape-validated level, same as standings_cache. fetched_at is RFC3339 UTC and is the only
// freshness signal.
const leagueScheduleCacheDDL = `
CREATE TABLE IF NOT EXISTS league_schedule_cache (
	league_id  TEXT NOT NULL,
	season     INTEGER NOT NULL,
	payload    TEXT NOT NULL,
	fetched_at TEXT NOT NULL,
	PRIMARY KEY (league_id, season)
);`

// initLeagueScheduleCacheSchema creates the league-schedule cache table. Called from
// initSchema alongside the other per-feature DDL (own file, store-no-siblings + the
// 400-line cap).
func (s *Store) initLeagueScheduleCacheSchema(ctx context.Context) error {
	if _, err := s.pools.Write().ExecContext(ctx, leagueScheduleCacheDDL); err != nil {
		return fmt.Errorf("state: init league-schedule-cache schema: %w", err)
	}
	return nil
}

// ErrNoCachedLeagueSchedule reports that nothing has ever been cached for this league+season.
// A legitimate, expected state (first run, or a first-ever fetch that failed), not a storage
// fault — callers distinguish "we have older data" (stale) from "we have nothing" (fail) on
// this, mirroring ErrNoCachedStandings.
var ErrNoCachedLeagueSchedule = errors.New("state: no cached league schedule for this league and season")

// PutLeagueSchedule replaces the cached schedule payload for this league+season. Called only
// after a SUCCESSFUL live fetch, so the cache never holds a partial or failed read. An empty
// or zero-row payload is REFUSED for the same reason PutStandings refuses one: a cache holding
// "[]" would arm a later outage with a board that renders blank while truthfully claiming to
// be cached data.
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

// CachedLeagueSchedule returns the last successfully fetched schedule payload for this
// league+season and the time it was fetched. Returns ErrNoCachedLeagueSchedule (wrapped) when
// nothing has been cached, so the caller can tell "stale data available" from "no data at
// all" — the two degrade to different UI states and must not collapse into one.
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
