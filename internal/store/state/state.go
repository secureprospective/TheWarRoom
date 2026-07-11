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
	"log"
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
	poisoned   error                      // sticky: set if a post-commit reload fails (memory is stale)

	// reload refreshes in-memory state from the DB after a committed transaction. It
	// is a field (defaulting to load) ONLY so a test can inject a post-commit reload
	// failure — the GLM-B7a MAJOR: a committed tx whose reload fails leaves memory
	// stale, and serving that silently would corrupt a cap engine.
	reload func(ctx context.Context) error
}

// New constructs an unseeded store for one league + season over the given pools.
// Call Initialize before any read.
func New(pools *db.Pools, leagueID string, season int) *Store {
	s := &Store{
		pools:      pools,
		leagueID:   leagueID,
		season:     season,
		franchises: map[string]*FranchiseState{},
		byPlayer:   map[string]string{},
	}
	s.reload = s.load
	return s
}

// Err returns the store's poison error, if any. A non-nil result means a transaction
// committed to the DB but the follow-up in-memory reload failed — so in-memory reads are
// STALE and must not be trusted. Read consumers on the mutation path check this before
// serving state. Cleared only by a fresh, successful Initialize.
func (s *Store) Err() error {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.poisoned
}

// poison records the sticky stale-memory condition (first cause wins).
func (s *Store) poison(err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.poisoned == nil {
		s.poisoned = err
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
	// Seed the genesis season-phase row (→ OFFSEASON) if the phase log is empty. Independent
	// of the roster seed so a DB that pre-dates the phase feature also gets it on first startup;
	// idempotent, so an existing transition history is never clobbered.
	if err := s.seedInitialPhase(ctx); err != nil {
		return err
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
func (s *Store) CapUsed(franchiseID string) (domain.Money, bool) {
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

// --- Persistence: presence, seed, load -------------------------------------

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
			// Ship 1: flat-fill the per-year ledger alongside the legacy contract row, in
			// the SAME tx. Additive — nothing reads these cells yet, so the live cap path
			// is untouched and main stays green.
			if err := seedLedgerPlayer(ctx, tx, s.leagueID, s.season, now, p); err != nil {
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
       c.annual_salary_cents, c.contract_years, c.expiration_year,
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

	// Ship 3 read-flip: cap usage is DERIVED from the ledger cells (the KING), not the frozen
	// legacy salary columns. loadCellCap gives each player's raw cap-counting salary (CapSalary)
	// and each franchise's cell cap ($10k-snapped per cell). Set both here — scanState no longer
	// touches money beyond the base Salary column.
	perPlayer, perFranchise, err := loadCellCap(ctx, s.pools.Read(), s.leagueID, s.season)
	if err != nil {
		return err
	}
	for fid, f := range fr {
		f.CapUsed = perFranchise[fid]
		for i := range f.Players {
			p := &f.Players[i]
			cs, ok := perPlayer[p.MFLID]
			// M1 carry-forward (DeepSeek): with cap now cell-derived, a rostered player who
			// carries a base salary but has NO PAID current-season cell will count $0 against
			// the cap — a ledger drift the read-flip can no longer absorb. We SURFACE it but do
			// NOT hard-fail: load() runs on every startup, and a single drift row must never
			// render the app unopenable (GLM-5.2 Ship-4 review — blast radius). A genuinely $0
			// base salary needs no cell, so the check gates on Salary > 0. A VOID cell counts as
			// absent here (perPlayer holds only PAID); that is CORRECT, not a false positive:
			// VoidCells runs ONLY inside deadcap.Waive, which de-rosters the player in the SAME
			// tx, so a still-rostered player's current cell is PAID or absent, never VOID. Real
			// drift is logged loudly (visible in the terminal) and the player counts $0 until
			// reconciled — degraded but open, not bricked.
			if !ok && p.Salary > 0 {
				log.Printf("state: load: WARNING rostered player %q has base salary %s but no PAID %d ledger cell (cell drift) — counting $0 cap until reconciled", p.MFLID, p.Salary, s.season)
			}
			p.CapSalary = cs
		}
	}

	// CapUsed then adds this season's dead-cap charges (B7b): the §8 waiver penalty counts
	// against the cap for its absolute year. A franchise that carries dead cap but no current
	// players still appears here (its cap is non-zero) — a deliberate extension of the
	// player-derived identity rule.
	dc, err := s.loadDeadCap(ctx)
	if err != nil {
		return err
	}
	for fid, amt := range dc {
		f, ok := fr[fid]
		if !ok {
			f = &FranchiseState{FranchiseID: fid}
			fr[fid] = f
		}
		f.CapUsed += amt
	}

	s.mu.Lock()
	s.franchises, s.byPlayer = fr, idx
	s.mu.Unlock()
	return nil
}

// loadDeadCap sums the dead-cap ledger charges per franchise for THIS store's season
// (league_year == season — dead cap is keyed to an absolute year, and CapUsed is a
// current-season figure). Returns cents-typed Money keyed by franchise id.
func (s *Store) loadDeadCap(ctx context.Context) (map[string]domain.Money, error) {
	rows, err := s.pools.Read().QueryContext(ctx, `
SELECT franchise_id, COALESCE(SUM(dead_cap_cents), 0)
FROM dead_cap_ledger
WHERE league_id = ? AND league_year = ?
GROUP BY franchise_id`, s.leagueID, s.season)
	if err != nil {
		return nil, fmt.Errorf("state: load dead cap: %w", err)
	}
	defer func() { _ = rows.Close() }()

	out := map[string]domain.Money{}
	for rows.Next() {
		var fid string
		var cents int64
		if err := rows.Scan(&fid, &cents); err != nil {
			return nil, fmt.Errorf("state: dead cap scan: %w", err)
		}
		out[fid] = domain.Money(cents)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("state: dead cap iterate: %w", err)
	}
	return out, nil
}
