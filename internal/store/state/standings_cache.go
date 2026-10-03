package state

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"
)

// standings_cache keeps the last good MFL standings so an MFL outage degrades M2 to stale data
// instead of blanking it. It is a cache, not a ledger, so it is the one kind of table here that is
// not append-only: it holds the newest copy of MFL's data per (league, season). Standings history
// would be a different table. payload is the fetcher's validated raw records as JSON, so a
// downstream schema change re-parses the cached bytes. fetched_at (RFC3339 UTC) is shown to the
// user as the board's age.
const standingsCacheDDL = `
CREATE TABLE IF NOT EXISTS standings_cache (
	league_id  TEXT NOT NULL,
	season     INTEGER NOT NULL,
	payload    TEXT NOT NULL,
	fetched_at TEXT NOT NULL,
	PRIMARY KEY (league_id, season)
);`

// initStandingsCacheSchema creates the standings cache table.
func (s *Store) initStandingsCacheSchema(ctx context.Context) error {
	if _, err := s.pools.Write().ExecContext(ctx, standingsCacheDDL); err != nil {
		return fmt.Errorf("state: init standings-cache schema: %w", err)
	}
	return nil
}

// ErrNoCachedStandings means nothing was ever cached: the caller reports "fail", not "stale".
var ErrNoCachedStandings = errors.New("state: no cached standings for this league and season")

// PutStandings replaces the cached standings after a successful fetch. An empty payload is
// refused: a cached "[]" would later render a blank board that claims to be cached data.
func (s *Store) PutStandings(ctx context.Context, payload string, fetchedAt time.Time) error {
	if p := strings.TrimSpace(payload); p == "" || p == "[]" || p == "null" {
		return fmt.Errorf("state: refusing to cache an empty standings payload (%q)", p)
	}
	s.wmu.Lock()
	defer s.wmu.Unlock()
	if _, err := s.pools.Write().ExecContext(ctx, `
INSERT INTO standings_cache (league_id, season, payload, fetched_at)
VALUES (?, ?, ?, ?)
ON CONFLICT (league_id, season) DO UPDATE SET
	payload    = excluded.payload,
	fetched_at = excluded.fetched_at`,
		s.leagueID, s.season, payload, fetchedAt.UTC().Format(time.RFC3339)); err != nil {
		return fmt.Errorf("state: cache standings: %w", err)
	}
	return nil
}

// CachedStandings returns the cached payload and when it was fetched, or ErrNoCachedStandings.
// An unparseable fetched_at is an error: stale data must always state its age.
func (s *Store) CachedStandings(ctx context.Context) (string, time.Time, error) {
	var payload, at string
	row := s.pools.Read().QueryRowContext(ctx,
		`SELECT payload, fetched_at FROM standings_cache WHERE league_id = ? AND season = ?`,
		s.leagueID, s.season)
	switch err := row.Scan(&payload, &at); {
	case errors.Is(err, sql.ErrNoRows):
		return "", time.Time{}, fmt.Errorf("season %d: %w", s.season, ErrNoCachedStandings)
	case err != nil:
		return "", time.Time{}, fmt.Errorf("state: read cached standings: %w", err)
	}
	ts, err := time.Parse(time.RFC3339, at)
	if err != nil {
		return "", time.Time{}, fmt.Errorf(
			"state: cached standings has unparseable fetched_at %q: %w", at, err)
	}
	return payload, ts, nil
}
