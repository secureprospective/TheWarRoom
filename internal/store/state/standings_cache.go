package state

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"
)

// standings_cache is the LAST-KNOWN-GOOD store for MFL leagueStandings payloads (B-5
// degradation contract). It exists so an MFL outage degrades M2 Power Rankings to stale
// data instead of blanking the module: the all-play component is 40% of the blend and had
// no persisted fallback, which made a standings failure fatal (the old m2_app.go comment
// said so explicitly).
//
// THIS TABLE IS DELIBERATELY NOT APPEND-ONLY — and that makes it the ONLY mutable table in
// this package. Every sibling here (season_phases, calendar_events, dead_cap_ledger,
// cap_relief_ledger, contract_year_changes) is a LEDGER: it records things that happened,
// so history is the point and the immutability triggers defend it. This is a CACHE: it
// records the latest copy of someone ELSE's data, which we did not author and cannot be
// asked to account for. Keeping only the newest row per (league, season) is correct here —
// there is no audit value in the standings we saw an hour ago, and an append-only cache
// would grow without bound on every refresh. If a future feature ever wants standings
// HISTORY (e.g. week-over-week movement), that is a different table with a different
// justification; do not quietly turn this one into a log.
//
// payload is the fetcher's raw records re-encoded as JSON. It is stored VERBATIM at the
// RawStanding level — pre-parse, post-shape-validation — so a schema change downstream
// re-parses the cached bytes rather than being stuck with an old normalization. fetched_at
// is RFC3339 UTC and is the ONLY freshness signal: the UI shows it literally, so a stale
// board always states how stale it is rather than implying it is current.
const standingsCacheDDL = `
CREATE TABLE IF NOT EXISTS standings_cache (
	league_id  TEXT NOT NULL,
	season     INTEGER NOT NULL,
	payload    TEXT NOT NULL,
	fetched_at TEXT NOT NULL,
	PRIMARY KEY (league_id, season)
);`

// initStandingsCacheSchema creates the standings cache table. Called from initSchema
// alongside the other per-feature DDL (own file, store-no-siblings + the 400-line cap).
func (s *Store) initStandingsCacheSchema(ctx context.Context) error {
	if _, err := s.pools.Write().ExecContext(ctx, standingsCacheDDL); err != nil {
		return fmt.Errorf("state: init standings-cache schema: %w", err)
	}
	return nil
}

// ErrNoCachedStandings reports that nothing has ever been cached for this league+season.
// It is a legitimate, expected state (first run, or a first-ever fetch that failed), NOT a
// storage fault — callers distinguish "we have older data" from "we have nothing" on it,
// which is the difference between the stale and fail freshness states.
var ErrNoCachedStandings = errors.New("state: no cached standings for this league and season")

// PutStandings replaces the cached standings payload for this league+season. It is called
// only after a SUCCESSFUL live fetch, so the cache never holds a partial or failed read.
// fetchedAt is normalized to RFC3339 UTC on the way in — one timestamp format in the
// column means the reader never has to guess how to parse it.
//
// It takes wmu to serialize with the rest of this package's mutations, matching the
// convention every other writer here follows. Note that wmu is NOT what makes the
// (payload, fetched_at) pair atomic — the single upsert statement below does that on its
// own, since SQLite applies it as one row-level operation. Do not read the lock as the
// source of that guarantee (GLM review, B-5: the previous comment here overclaimed it).
//
// An empty or zero-row payload is REFUSED. The fetcher already rejects a zero-franchise
// response (errEmptyStandings), so this is the second line of a defense that matters: a
// cache holding "[]" would arm a later outage with a board that renders blank while
// truthfully claiming to be cached data — a blank screen wearing a freshness label, which
// is worse than an honest failure. Guarding at THIS boundary means the promise holds for
// any future caller, not just the one that exists today.
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

// CachedStandings returns the last successfully fetched standings payload for this
// league+season and the time it was fetched. It returns ErrNoCachedStandings (wrapped)
// when nothing has been cached, so the caller can tell "stale data available" from "no
// data at all" — the two degrade to different UI states and must not collapse into one.
//
// A row whose fetched_at will not parse is reported as an error rather than served with a
// zero time: an unlabelled stale board is worse than an honest failure, because the whole
// contract is that stale data always states its age.
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
