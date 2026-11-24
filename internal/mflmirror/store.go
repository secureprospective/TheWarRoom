// Package mflmirror holds a read-only copy of one MFL league's public data inside the hosted league
// database, and enforces the 5-minute window between two reads of MFL. The reading itself is
// mflmirror/mflread; this package never names an MFL client.
package mflmirror

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/db"
	"github.com/secureprospective/TheWarRoom/internal/domain"
)

// Source reads MFL's public exports; the read-only implementation is mflmirror/mflread.
type Source interface {
	FetchFreeAgents(ctx context.Context, mflLeague string, season int) ([]string, error)
	FetchPlayers(ctx context.Context, mflLeague string, season int) ([]byte, error)
	// Directory reads a held player directory (the bytes FetchPlayers returned) into player facts.
	Directory(playersJSON []byte) (Directory, error)
}

// Directory resolves an MFL player id to name, position, team, age and draft year.
type Directory interface {
	Facts(id string) (domain.PlayerFacts, bool)
}

// Season is the NFL season a date belongs to: January and February still belong to the year before.
func Season(now time.Time) int {
	if now.Month() <= time.February {
		return now.Year() - 1
	}
	return now.Year()
}

// Window is the shortest gap between two reads of MFL for one league.
const Window = 5 * time.Minute

// PlayersTTL is how long the held player directory is reused (MFL allows one call a day).
const PlayersTTL = 24 * time.Hour

// Held is one league's held copy of MFL's public data.
type Held struct {
	MFLLeague   string
	Season      int
	FetchedAt   time.Time
	FreeAgents  []string
	PlayersJSON []byte // the player directory as JSON; normalize.NewLookupJSON reads it
}

// Store holds the mirror in the hosted league's own database. Its tables are created only if
// missing, so nothing already in the database is touched.
type Store struct {
	pools *db.Pools
	now   func() time.Time
	mu    sync.Mutex // single-flight: one refresh at a time
}

func NewStore(pools *db.Pools, now func() time.Time) *Store {
	if now == nil {
		now = time.Now
	}
	return &Store{pools: pools, now: now}
}

const schema = `
CREATE TABLE IF NOT EXISTS mfl_mirror_settings (key TEXT PRIMARY KEY, value TEXT NOT NULL);
CREATE TABLE IF NOT EXISTS mfl_mirror_hold (
  mfl_league TEXT PRIMARY KEY, season INTEGER NOT NULL, last_fetched_at TEXT NOT NULL, free_agents TEXT NOT NULL);
CREATE TABLE IF NOT EXISTS mfl_mirror_players (
  season INTEGER PRIMARY KEY, fetched_at TEXT NOT NULL, body TEXT NOT NULL);`

func (s *Store) ensure(ctx context.Context) error {
	if s == nil || s.pools == nil {
		return errors.New("MFL mirror storage unavailable")
	}
	if _, err := s.pools.Write().ExecContext(ctx, schema); err != nil {
		return fmt.Errorf("mflmirror: create tables: %w", err)
	}
	return nil
}

const (
	keyLeague    = "mfl.league"
	keyFranchise = "mfl.franchise"
)

// Registration is the league's plain settings that say which MFL league it follows and which
// MFL franchise the seated owner chose.
type Registration struct{ MFLLeague, Franchise string }

// Register stores the two settings. They are ordinary league settings, not a league kind.
func (s *Store) Register(ctx context.Context, r Registration) error {
	if err := s.ensure(ctx); err != nil {
		return err
	}
	for k, v := range map[string]string{keyLeague: r.MFLLeague, keyFranchise: r.Franchise} {
		if _, err := s.pools.Write().ExecContext(ctx,
			`INSERT INTO mfl_mirror_settings(key,value) VALUES(?,?) ON CONFLICT(key) DO UPDATE SET value=excluded.value`,
			k, v); err != nil {
			return fmt.Errorf("mflmirror: save setting: %w", err)
		}
	}
	return nil
}

// Registration reads the settings; ok is false when the league follows no MFL league.
func (s *Store) Registration(ctx context.Context) (Registration, bool, error) {
	if err := s.ensure(ctx); err != nil {
		return Registration{}, false, err
	}
	var r Registration
	for k, dst := range map[string]*string{keyLeague: &r.MFLLeague, keyFranchise: &r.Franchise} {
		if _, err := s.queryOne(ctx, `SELECT value FROM mfl_mirror_settings WHERE key=?`, []any{k}, dst); err != nil {
			return Registration{}, false, fmt.Errorf("mflmirror: read setting: %w", err)
		}
	}
	return r, r.MFLLeague != "", nil
}

// Result is a refresh answer: the held copy, its age, and whether MFL was asked this time.
type Result struct {
	Held    Held
	Age     time.Duration
	Fetched bool
}

// Refresh returns the held copy when it is younger than Window, with no call to MFL; otherwise it
// fetches, stores and returns the fresh copy. Calls are single-flight. The window is judged from
// last_fetched_at in the database.
func (s *Store) Refresh(ctx context.Context, f Source, mflLeague string) (Result, error) {
	if err := s.ensure(ctx); err != nil {
		return Result{}, err
	}
	if f == nil {
		return Result{}, errors.New("MFL data source is not available on this host")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	held, ok, err := s.read(ctx, mflLeague)
	if err != nil {
		return Result{}, err
	}
	now := s.now()
	if ok && now.Sub(held.FetchedAt) < Window {
		return Result{Held: held, Age: now.Sub(held.FetchedAt)}, nil
	}
	season := Season(now)
	ids, err := f.FetchFreeAgents(ctx, mflLeague, season)
	if err != nil {
		return Result{}, fmt.Errorf("mflmirror: free agents: %w", err)
	}
	if err := s.refreshPlayers(ctx, f, mflLeague, season, now); err != nil {
		return Result{}, err
	}
	body, err := json.Marshal(ids)
	if err != nil {
		return Result{}, fmt.Errorf("mflmirror: encode free agents: %w", err)
	}
	if _, err := s.pools.Write().ExecContext(ctx, `INSERT INTO mfl_mirror_hold(mfl_league,season,last_fetched_at,free_agents)
VALUES(?,?,?,?) ON CONFLICT(mfl_league) DO UPDATE SET season=excluded.season,
last_fetched_at=excluded.last_fetched_at, free_agents=excluded.free_agents`,
		mflLeague, season, now.UTC().Format(time.RFC3339Nano), string(body)); err != nil {
		return Result{}, fmt.Errorf("mflmirror: hold free agents: %w", err)
	}
	held, _, err = s.read(ctx, mflLeague)
	if err != nil {
		return Result{}, err
	}
	return Result{Held: held, Age: now.Sub(held.FetchedAt), Fetched: true}, nil
}

func (s *Store) refreshPlayers(ctx context.Context, f Source, mflLeague string, season int, now time.Time) error {
	var at string
	found, err := s.queryOne(ctx, `SELECT fetched_at FROM mfl_mirror_players WHERE season=?`, []any{season}, &at)
	if err != nil {
		return fmt.Errorf("mflmirror: read player hold: %w", err)
	}
	if found {
		if t, perr := time.Parse(time.RFC3339Nano, at); perr == nil && now.Sub(t) < PlayersTTL {
			return nil
		}
	}
	raw, err := f.FetchPlayers(ctx, mflLeague, season)
	if err != nil {
		return fmt.Errorf("mflmirror: players: %w", err)
	}
	if !json.Valid(raw) {
		return errors.New("mflmirror: the player directory is not JSON")
	}
	if _, err := s.pools.Write().ExecContext(ctx, `INSERT INTO mfl_mirror_players(season,fetched_at,body) VALUES(?,?,?)
ON CONFLICT(season) DO UPDATE SET fetched_at=excluded.fetched_at, body=excluded.body`,
		season, now.UTC().Format(time.RFC3339Nano), string(raw)); err != nil {
		return fmt.Errorf("mflmirror: hold players: %w", err)
	}
	return nil
}

// Read returns the held copy without any network call.
func (s *Store) Read(ctx context.Context, mflLeague string) (Held, bool, error) {
	if err := s.ensure(ctx); err != nil {
		return Held{}, false, err
	}
	return s.read(ctx, mflLeague)
}

func (s *Store) read(ctx context.Context, mflLeague string) (Held, bool, error) {
	h := Held{MFLLeague: mflLeague}
	var at, fa string
	found, err := s.queryOne(ctx, `SELECT season,last_fetched_at,free_agents FROM mfl_mirror_hold WHERE mfl_league=?`,
		[]any{mflLeague}, &h.Season, &at, &fa)
	if err != nil {
		return Held{}, false, fmt.Errorf("mflmirror: read hold: %w", err)
	}
	if !found {
		return Held{}, false, nil
	}
	if h.FetchedAt, err = time.Parse(time.RFC3339Nano, at); err != nil {
		return Held{}, false, fmt.Errorf("mflmirror: held time: %w", err)
	}
	if err := json.Unmarshal([]byte(fa), &h.FreeAgents); err != nil {
		return Held{}, false, fmt.Errorf("mflmirror: held free agents: %w", err)
	}
	var body string
	hasPlayers, err := s.queryOne(ctx, `SELECT body FROM mfl_mirror_players WHERE season=?`, []any{h.Season}, &body)
	if err != nil {
		return Held{}, false, fmt.Errorf("mflmirror: read players: %w", err)
	}
	if hasPlayers {
		h.PlayersJSON = []byte(body)
	}
	return h, true, nil
}

// queryOne scans the first row into dest; found is false when there is none.
func (s *Store) queryOne(ctx context.Context, query string, args []any, dest ...any) (bool, error) {
	rows, err := s.pools.Read().QueryContext(ctx, query, args...)
	if err != nil {
		return false, fmt.Errorf("query: %w", err)
	}
	defer func() { _ = rows.Close() }()
	if !rows.Next() {
		if err := rows.Err(); err != nil {
			return false, fmt.Errorf("rows: %w", err)
		}
		return false, nil
	}
	if err := rows.Scan(dest...); err != nil {
		return false, fmt.Errorf("scan: %w", err)
	}
	return true, nil
}
