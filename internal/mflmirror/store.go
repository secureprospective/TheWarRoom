// Package mflmirror holds a read-only copy of one MFL league's public data inside the hosted league
// database, and enforces the 5-minute window between two reads of MFL. The reading itself is
// mflmirror/mflread; this package never names an MFL client.
package mflmirror

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"sync"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/db"
	"github.com/secureprospective/TheWarRoom/internal/domain"
)

// Source reads MFL's public exports; the read-only implementation is mflmirror/mflread.
type Source interface {
	FetchFreeAgents(ctx context.Context, mflLeague string, season int) ([]string, error)
	FetchPlayers(ctx context.Context, mflLeague string, season int) ([]byte, error)
	// FetchLeagueData reads the league's public franchises, rosters, salary adjustments, rules and NFL bye weeks.
	FetchLeagueData(ctx context.Context, mflLeague string, season int) (LeagueData, error)
	// Directory reads a held player directory (the bytes FetchPlayers returned) into player facts.
	Directory(playersJSON []byte) (Directory, error)
}

// Franchise is one team in the MFL league's public franchise list.
type Franchise struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// RosterRow is one player on one MFL franchise's roster: salary in cents, the final contract year
// and MFL's status (ROSTER, TAXI_SQUAD or INJURED_RESERVE).
type RosterRow struct {
	Franchise    string `json:"franchise"`
	Player       string `json:"player"`
	Salary       int64  `json:"salary"`
	ContractYear int    `json:"contractYear"`
	Status       string `json:"status"`
}

// Adjustment is one literal MFL cap adjustment (dead cap, credit) on a franchise, in cents.
type Adjustment struct {
	Franchise string `json:"franchise"`
	Cents     int64  `json:"cents"`
}

// LeagueData is the public league data the bid and lineup checks read: franchises, every roster,
// the cap adjustments, MFL's league and lineup rules, and each NFL team's bye week.
type LeagueData struct {
	Franchises  []Franchise     `json:"franchises,omitempty"`
	Rosters     []RosterRow     `json:"rosters"`
	Adjustments []Adjustment    `json:"adjustments"`
	Byes        map[string]int  `json:"byes"`
	Config      json.RawMessage `json:"config"` // MFL's league rules; the MFL reader decodes them
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
	Franchises  []Franchise
	League      LeagueData
	HasLeague   bool
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
CREATE TABLE IF NOT EXISTS mfl_mirror_franchises (mfl_league TEXT PRIMARY KEY, body TEXT NOT NULL);
CREATE TABLE IF NOT EXISTS mfl_mirror_league (mfl_league TEXT PRIMARY KEY, body TEXT NOT NULL);
CREATE TABLE IF NOT EXISTS mfl_mirror_team (
  franchise TEXT PRIMARY KEY, mfl_league TEXT NOT NULL, team TEXT NOT NULL);
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

// SetTeam records the MFL team the seated franchise's owner chose. The franchise comes from the
// seat, never from a request, and the team must be one of the held league's franchises.
func (s *Store) SetTeam(ctx context.Context, franchise, mflLeague, team string) error {
	if franchise == "" {
		return errors.New("a seated franchise is required")
	}
	held, ok, err := s.Read(ctx, mflLeague)
	if err != nil {
		return err
	}
	if !ok || len(held.Franchises) == 0 {
		return Refused("MFL teams are not held yet; refresh first")
	}
	if !slices.ContainsFunc(held.Franchises, func(f Franchise) bool { return f.ID == team }) {
		return Refused(fmt.Sprintf("team %q is not one of this MFL league's teams", team))
	}
	if _, err := s.pools.Write().ExecContext(ctx, `INSERT INTO mfl_mirror_team(franchise,mfl_league,team) VALUES(?,?,?)
ON CONFLICT(franchise) DO UPDATE SET mfl_league=excluded.mfl_league, team=excluded.team`,
		franchise, mflLeague, team); err != nil {
		return fmt.Errorf("mflmirror: save team: %w", err)
	}
	return nil
}

// Team reads the seated franchise's chosen MFL team; ok is false when none is chosen for this MFL league.
func (s *Store) Team(ctx context.Context, franchise, mflLeague string) (string, bool, error) {
	if err := s.ensure(ctx); err != nil {
		return "", false, err
	}
	var league, team string
	found, err := s.queryOne(ctx, `SELECT mfl_league,team FROM mfl_mirror_team WHERE franchise=?`, []any{franchise}, &league, &team)
	if err != nil {
		return "", false, fmt.Errorf("mflmirror: read team: %w", err)
	}
	if !found || league != mflLeague {
		return "", false, nil
	}
	return team, true, nil
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
	needPlayers, err := s.playersStale(ctx, season, now)
	if err != nil {
		return Result{}, err
	}
	ids, franchises, data, playersRaw, err := s.fetchAll(ctx, f, mflLeague, season, needPlayers)
	if err != nil {
		return Result{}, err
	}
	if needPlayers {
		if err := s.holdPlayers(ctx, f, mflLeague, season, now, playersRaw); err != nil {
			return Result{}, err
		}
	}
	if err := s.holdRefresh(ctx, mflLeague, season, now, ids, franchises, data); err != nil {
		return Result{}, err
	}
	held, _, err = s.read(ctx, mflLeague)
	if err != nil {
		return Result{}, err
	}
	return Result{Held: held, Age: now.Sub(held.FetchedAt), Fetched: true}, nil
}

func (s *Store) holdRefresh(ctx context.Context, mflLeague string, season int, now time.Time,
	ids []string, franchises []Franchise, data LeagueData) error {
	if err := s.holdLeague(ctx, mflLeague, data); err != nil {
		return err
	}
	fbody, err := json.Marshal(franchises)
	if err != nil {
		return fmt.Errorf("mflmirror: encode franchises: %w", err)
	}
	if _, err := s.pools.Write().ExecContext(ctx, `INSERT INTO mfl_mirror_franchises(mfl_league,body) VALUES(?,?)
ON CONFLICT(mfl_league) DO UPDATE SET body=excluded.body`, mflLeague, string(fbody)); err != nil {
		return fmt.Errorf("mflmirror: hold franchises: %w", err)
	}
	body, err := json.Marshal(ids)
	if err != nil {
		return fmt.Errorf("mflmirror: encode free agents: %w", err)
	}
	if _, err := s.pools.Write().ExecContext(ctx, `INSERT INTO mfl_mirror_hold(mfl_league,season,last_fetched_at,free_agents)
VALUES(?,?,?,?) ON CONFLICT(mfl_league) DO UPDATE SET season=excluded.season,
last_fetched_at=excluded.last_fetched_at, free_agents=excluded.free_agents`,
		mflLeague, season, now.UTC().Format(time.RFC3339Nano), string(body)); err != nil {
		return fmt.Errorf("mflmirror: hold free agents: %w", err)
	}
	return nil
}

// refreshSource lets a Source share one MFL client across a refresh so `TYPE=league` is fetched once.
type refreshSource interface {
	FetchRefresh(ctx context.Context, mflLeague string, season int, needPlayers bool) ([]string, LeagueData, []byte, error)
}

// fetchAll reads everything one refresh holds from MFL: free agents, league data (including franchises)
// and, when stale, the player directory.
func (*Store) fetchAll(ctx context.Context, f Source, mflLeague string, season int,
	needPlayers bool) ([]string, []Franchise, LeagueData, []byte, error) {
	if batch, ok := f.(refreshSource); ok {
		ids, data, rawPlayers, err := batch.FetchRefresh(ctx, mflLeague, season, needPlayers)
		if err != nil {
			return nil, nil, LeagueData{}, nil, fmt.Errorf("mflmirror: refresh: %w", err)
		}
		return ids, data.Franchises, data, rawPlayers, nil
	}
	ids, err := f.FetchFreeAgents(ctx, mflLeague, season)
	if err != nil {
		return nil, nil, LeagueData{}, nil, fmt.Errorf("mflmirror: free agents: %w", err)
	}
	data, err := f.FetchLeagueData(ctx, mflLeague, season)
	if err != nil {
		return nil, nil, LeagueData{}, nil, fmt.Errorf("mflmirror: league data: %w", err)
	}
	return ids, data.Franchises, data, nil, nil
}

func (s *Store) holdLeague(ctx context.Context, mflLeague string, data LeagueData) error {
	body, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("mflmirror: encode league data: %w", err)
	}
	if _, err := s.pools.Write().ExecContext(ctx, `INSERT INTO mfl_mirror_league(mfl_league,body) VALUES(?,?)
ON CONFLICT(mfl_league) DO UPDATE SET body=excluded.body`, mflLeague, string(body)); err != nil {
		return fmt.Errorf("mflmirror: hold league data: %w", err)
	}
	return nil
}

func (s *Store) playersStale(ctx context.Context, season int, now time.Time) (bool, error) {
	var at string
	found, err := s.queryOne(ctx, `SELECT fetched_at FROM mfl_mirror_players WHERE season=?`, []any{season}, &at)
	if err != nil {
		return false, fmt.Errorf("mflmirror: read player hold: %w", err)
	}
	if found {
		if t, perr := time.Parse(time.RFC3339Nano, at); perr == nil && now.Sub(t) < PlayersTTL {
			return false, nil
		}
	}
	return true, nil
}

func (s *Store) holdPlayers(ctx context.Context, f Source, mflLeague string, season int, now time.Time, raw []byte) error {
	if raw == nil {
		var err error
		raw, err = f.FetchPlayers(ctx, mflLeague, season)
		if err != nil {
			return fmt.Errorf("mflmirror: players: %w", err)
		}
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
	var fbody string
	hasTeams, err := s.queryOne(ctx, `SELECT body FROM mfl_mirror_franchises WHERE mfl_league=?`, []any{mflLeague}, &fbody)
	if err != nil {
		return Held{}, false, fmt.Errorf("mflmirror: read franchises: %w", err)
	}
	if hasTeams {
		if err := json.Unmarshal([]byte(fbody), &h.Franchises); err != nil {
			return Held{}, false, fmt.Errorf("mflmirror: held franchises: %w", err)
		}
	}
	var lbody string
	if h.HasLeague, err = s.queryOne(ctx, `SELECT body FROM mfl_mirror_league WHERE mfl_league=?`, []any{mflLeague}, &lbody); err != nil {
		return Held{}, false, fmt.Errorf("mflmirror: read league data: %w", err)
	}
	if h.HasLeague {
		if err := json.Unmarshal([]byte(lbody), &h.League); err != nil {
			return Held{}, false, fmt.Errorf("mflmirror: held league data: %w", err)
		}
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

// Pools is the league database the mirror lives in; plans keep their rows in the same file.
func (s *Store) Pools() *db.Pools { return s.pools }
