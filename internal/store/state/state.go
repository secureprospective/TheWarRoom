// Package state is the B3c League State Store: the second Layer-2 store and the
// 32-team MUTABLE runtime state (live rosters + contracts + derived cap usage),
// SQLite-backed. It clones B3b's store template (file shape, parameterized SQL,
// db.Pools read/write split, injected seed source, concurrency idioms) but DIVERGES
// in two deliberate ways:
//
//   - Single-writer law (AD-02/AD-05): the mutation surface (Writer) is handed
//     ONLY to B7a; every other consumer gets a read-only Reader that cannot
//     reach a mutation even by type assertion. The single-connection write pool is
//     the driver-level enforcement.
//   - NO Reload/versioning/overrides. Config can be re-pulled from MFL (B3b); runtime
//     state CANNOT — B7's validated mutations are the source of truth at runtime.
//     Initialize seeds ONCE from normalize; on an existing DB it loads, never reseeds.
//
// It is PURE DATA ACCESS: it stores and serves state and computes cap USAGE, but runs
// no rule logic (tag prices, dead cap, eligibility) — that is engine/B7 work.
package state

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/db"
	"github.com/secureprospective/TheWarRoom/internal/domain"
)

// errEmptySeed is returned when the seed source yields no franchise state. Seeding an
// empty state would silently wipe the league (the rosters errEmpty lesson) — fail loud.
var errEmptySeed = errors.New("state: seed source produced no roster state")

// errUnknownPlayer is returned by a mutation targeting a player not in the store.
var errUnknownPlayer = errors.New("state: player not found")

// Store is the league state store. Construct with New; seed with Initialize. It is
// safe for concurrent reads via mu (the reader/snapshot lock); admin/B7 mutations
// serialize under wmu (the outer write lock), so each DB write and its in-memory
// reload are one atomic step — identical to B3b's two-lock pattern.
type Store struct {
	pools    *db.Pools
	leagueID string
	season   int
	src      Source

	wmu sync.Mutex // serializes mutations (seed + every Writer call) end to end

	mu         sync.RWMutex
	franchises map[string]*FranchiseState // keyed franchise id; values owned by store
	byPlayer   map[string]string          // mflID -> franchiseID index
}

// New constructs an unseeded store for one league + season over the given pools.
// Call Initialize before any read.
func New(pools *db.Pools, leagueID string, season int) *Store {
	return &Store{
		pools:      pools,
		leagueID:   leagueID,
		season:     season,
		franchises: map[string]*FranchiseState{},
		byPlayer:   map[string]string{},
	}
}

// Writer returns the mutation surface. It is intended for B7a's dependency injection
// ONLY — call it exactly where the single writer is wired, nowhere else.
func (s *Store) Writer() Writer { return s }

// Reader returns the read-only surface for the engine, modules, and handlers. The
// returned value does NOT embed *Store, so a consumer cannot recover the writer by
// type assertion — the boundary is real, not just a naming convention.
func (s *Store) Reader() Reader { return readerView{s: s} }

// Initialize ensures the schema, remembers the seed source, and loads state into
// memory. On a fresh database (no rows for this league+season) it seeds ONCE from
// src; on an existing database it loads what is there WITHOUT reseeding — runtime
// state is never blindly re-pulled (the B3c divergence from rulebook's Reload).
func (s *Store) Initialize(ctx context.Context, src Source) error {
	s.wmu.Lock()
	defer s.wmu.Unlock()
	s.src = src
	if err := s.initSchema(ctx); err != nil {
		return err
	}
	has, err := s.hasState(ctx)
	if err != nil {
		return err
	}
	if !has {
		rosters, ferr := src.Rosters(ctx)
		if ferr != nil {
			return fmt.Errorf("state: seed fetch: %w", ferr)
		}
		if seedPlayerCount(rosters) == 0 {
			return errEmptySeed
		}
		if err := s.seed(ctx, rosters); err != nil {
			return err
		}
	}
	return s.load(ctx)
}

// --- Reads (the Reader implementation on *Store) ----------------------

// FranchiseState returns a deep copy of one franchise's state. ok is false for an
// unknown franchise.
func (s *Store) FranchiseState(franchiseID string) (FranchiseState, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	fs, ok := s.franchises[franchiseID]
	if !ok {
		return FranchiseState{}, false
	}
	return cloneFranchise(fs), true
}

// Roster returns a deep copy of one franchise's players. ok is false for an unknown
// franchise.
func (s *Store) Roster(franchiseID string) ([]PlayerState, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	fs, ok := s.franchises[franchiseID]
	if !ok {
		return nil, false
	}
	return clonePlayers(fs.Players), true
}

// CapUsed returns one franchise's derived cap usage. ok is false for an unknown
// franchise.
func (s *Store) CapUsed(franchiseID string) (float64, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	fs, ok := s.franchises[franchiseID]
	if !ok {
		return 0, false
	}
	return fs.CapUsed, true
}

// Player returns a copy of one player's state by id. ok is false if unrostered.
func (s *Store) Player(mflID string) (PlayerState, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	fid, ok := s.byPlayer[mflID]
	if !ok {
		return PlayerState{}, false
	}
	for _, p := range s.franchises[fid].Players {
		if p.MFLID == mflID {
			return p, true
		}
	}
	return PlayerState{}, false
}

// Franchises lists the franchise ids currently in state, sorted. NOTE (v1.0):
// franchise identity is PLAYER-DERIVED — a franchise appears only while it holds at
// least one player. A franchise emptied of all players is absent here and returns
// ok=false from FranchiseState/CapUsed. The canonical 32-team list is owned
// elsewhere (a future franchise registry); this store does not assert "always 32".
func (s *Store) Franchises() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return sortedKeys(s.franchises)
}

// --- Persistence: schema, presence, seed, load -----------------------------

// initSchema creates the rosters and contracts tables if absent (Backend_Architecture
// §8). B3c OWNS these two tables; waiver_order / dead_cap_ledger / league_state are
// out of B3c v1.0 scope (B7 / B6 own them).
func (s *Store) initSchema(ctx context.Context) error {
	const ddl = `
CREATE TABLE IF NOT EXISTS rosters (
	id            TEXT PRIMARY KEY,
	league_id     TEXT NOT NULL,
	mfl_id        TEXT NOT NULL,
	franchise_id  TEXT NOT NULL,
	roster_status TEXT NOT NULL,
	season        INTEGER NOT NULL,
	as_of         TEXT NOT NULL,
	UNIQUE (league_id, season, mfl_id)
);
CREATE TABLE IF NOT EXISTS contracts (
	id              TEXT PRIMARY KEY,
	league_id       TEXT NOT NULL,
	mfl_id          TEXT NOT NULL,
	franchise_id    TEXT NOT NULL,
	annual_salary   REAL NOT NULL DEFAULT 0,
	adjusted_salary REAL NOT NULL DEFAULT 0,
	contract_years  INTEGER NOT NULL DEFAULT 0,
	expiration_year INTEGER NOT NULL DEFAULT 0,
	contract_status TEXT NOT NULL DEFAULT '',
	is_restructured INTEGER NOT NULL DEFAULT 0,
	is_tagged       INTEGER NOT NULL DEFAULT 0,
	season          INTEGER NOT NULL,
	last_updated    TEXT NOT NULL,
	UNIQUE (league_id, season, mfl_id)
);`
	if _, err := s.pools.Write().ExecContext(ctx, ddl); err != nil {
		return fmt.Errorf("state: init schema: %w", err)
	}
	return nil
}

// hasState reports whether any roster rows already exist for this league + season.
func (s *Store) hasState(ctx context.Context) (bool, error) {
	n, err := s.rosterCount(ctx)
	return n > 0, err
}

// rosterCount returns the number of roster rows for this league + season.
func (s *Store) rosterCount(ctx context.Context) (int, error) {
	var n int
	row := s.pools.Read().QueryRowContext(ctx,
		`SELECT COUNT(1) FROM rosters WHERE league_id = ? AND season = ?`,
		s.leagueID, s.season)
	if err := row.Scan(&n); err != nil {
		return 0, fmt.Errorf("state: roster count: %w", err)
	}
	return n, nil
}

// seed writes the normalized rosters into the rosters + contracts tables in ONE
// transaction (atomic — a partial seed never lands). Only fields normalize provides
// are seeded; adjustment/years-remaining/tag flags stay at their zero values until
// B7 sets them.
func (s *Store) seed(ctx context.Context, rosters []domain.Roster) error {
	now := time.Now().UTC().Format(time.RFC3339)
	tx, err := s.pools.Write().BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("state: seed begin: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	for _, r := range rosters {
		for _, p := range r.Players {
			if err := seedPlayer(ctx, tx, s.leagueID, s.season, now, r.FranchiseID, p); err != nil {
				return err
			}
		}
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("state: seed commit: %w", err)
	}
	return nil
}

// load reads the full rosters⋈contracts state into memory and swaps it under mu, so
// concurrent readers always see a consistent snapshot. Called after seed and after
// every mutation (32 teams is small; a full reload keeps memory and DB identical).
func (s *Store) load(ctx context.Context) error {
	rows, err := s.pools.Read().QueryContext(ctx, `
SELECT r.franchise_id, r.mfl_id, r.roster_status,
       c.annual_salary, c.adjusted_salary, c.contract_years, c.expiration_year,
       c.contract_status, c.is_restructured, c.is_tagged
FROM rosters r
JOIN contracts c
  ON c.league_id = r.league_id AND c.season = r.season AND c.mfl_id = r.mfl_id
WHERE r.league_id = ? AND r.season = ?
ORDER BY r.franchise_id, r.mfl_id`, s.leagueID, s.season)
	if err != nil {
		return fmt.Errorf("state: load: %w", err)
	}
	defer func() { _ = rows.Close() }()

	fr, idx, err := scanState(rows)
	if err != nil {
		return err
	}

	// Reconcile: the rosters⋈contracts inner join silently drops any roster row
	// whose contract mate is missing. The seed pairs them in one transaction, so a
	// shortfall means drift — fail loud rather than serve a player short.
	want, err := s.rosterCount(ctx)
	if err != nil {
		return err
	}
	if len(idx) != want {
		return fmt.Errorf("state: load matched %d of %d roster rows (contract rows missing)", len(idx), want)
	}

	s.mu.Lock()
	s.franchises, s.byPlayer = fr, idx
	s.mu.Unlock()
	return nil
}
