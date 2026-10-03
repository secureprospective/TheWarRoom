// Package state holds the league's mutable runtime state in SQLite: rosters, contracts and the
// derived cap usage for all 32 teams.
//
// Only the transaction coordinator gets the Writer; everyone else gets a Reader that cannot reach
// a mutation, and the single-connection write pool enforces it at the driver. Unlike the rulebook,
// state is never re-pulled: Initialize seeds once on a fresh DB and only loads after that. The
// store holds data and computes cap usage; rule logic belongs to the transaction handlers.
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

// errEmptySeed: seeding an empty state would silently wipe the league, so fail loudly.
var errEmptySeed = errors.New("state: seed source produced no roster state")

// errUnknownPlayer is returned by a mutation on a player not in the store.
var errUnknownPlayer = errors.New("state: player not found")

// Store is the league state store. Reads take mu; mutations serialize under wmu, so each DB
// write and its in-memory reload are one step.
type Store struct {
	pools     *db.Pools
	leagueID  string
	season    int
	src       Source
	discounts CapDiscounts // nil = no discount

	wmu sync.Mutex // serializes every mutation, end to end

	leagueView       // the loaded league; poisoned shares its mu
	poisoned   error // set when a post-commit reload fails

	// reload refreshes memory after a commit. It is a field only so a test can inject a reload
	// failure: a committed transaction with stale memory must never be served silently.
	reload func(ctx context.Context) error
}

// New constructs an unseeded store. discounts supplies the taxi/IR cap percentages (usually the
// rulebook store); nil means no discount. Call Initialize before any read.
func New(pools *db.Pools, leagueID string, season int, discounts CapDiscounts) *Store {
	s := &Store{
		pools:      pools,
		leagueID:   leagueID,
		season:     season,
		discounts:  discounts,
		leagueView: newLeagueView(),
	}
	s.reload = s.load
	return s
}

// Err reports that a transaction committed but the memory reload failed, so in-memory reads are
// stale. Cleared only by a successful Initialize.
func (s *Store) Err() error {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.poisoned
}

// poison records the stale-memory condition; the first cause wins.
func (s *Store) poison(err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.poisoned == nil {
		s.poisoned = err
	}
}

// Writer returns the mutation surface. Call it only where the coordinator is wired.
func (s *Store) Writer() Writer { return s }

// Reader returns the read-only surface. It does not embed *Store, so a type assertion cannot
// recover the writer.
func (s *Store) Reader() Reader { return readerView{s: s} }

// Initialize ensures the schema and loads state. A fresh DB seeds once from src; an existing DB
// loads as-is, with no reseed.
func (s *Store) Initialize(ctx context.Context, src Source) error {
	s.wmu.Lock()
	defer s.wmu.Unlock()
	s.src = src
	if err := s.initSchema(ctx); err != nil {
		return err
	}
	// Derive the season from the phase log before checking for state: otherwise a rolled-over DB
	// would find no rosters at the config season and reseed.
	if err := s.refreshSeason(ctx); err != nil {
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
	// Seed the genesis phase row if the phase log is empty. Idempotent.
	if err := s.seedInitialPhase(ctx); err != nil {
		return err
	}
	return s.load(ctx)
}

// hasState reports whether roster rows exist for this league and season.
func (s *Store) hasState(ctx context.Context) (bool, error) {
	n, err := s.rosterCount(ctx)
	return n > 0, err
}

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

// seed writes the normalized rosters, contracts and ledger cells in one transaction.
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

// load reads the full state into memory and swaps it under mu. It runs after every mutation:
// 32 teams is small, and a full reload keeps memory identical to the DB.
func (s *Store) load(ctx context.Context) error {
	// Re-derive the season first: after a rollover, memory must not serve the prior season's cap.
	if err := s.refreshSeason(ctx); err != nil {
		return err
	}
	rows, err := s.pools.Read().QueryContext(ctx, `
SELECT r.franchise_id, r.mfl_id, r.roster_status,
       c.annual_salary_cents, c.expiration_year,
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

	// The inner join drops a roster row with no contract. Seeding pairs them, so a shortfall is
	// drift: fail loudly.
	want, err := s.rosterCount(ctx)
	if err != nil {
		return err
	}
	if len(idx) != want {
		return fmt.Errorf("state: load matched %d of %d roster rows (contract rows missing)", len(idx), want)
	}

	// Cap usage is derived from the ledger cells, the source of truth, not the legacy salary column.
	perPlayer, perFranchise, err := loadCellCap(ctx, s.pools.Read(), s.leagueID, s.season, s.discounts)
	if err != nil {
		return err
	}
	for fid, f := range fr {
		f.CapUsed = perFranchise[fid]
		for i := range f.Players {
			p := &f.Players[i]
			cs, ok := perPlayer[p.MFLID]
			// A rostered player with a salary but no PAID current-season cell counts $0: drift. Log it
			// loudly but don't fail, because load runs on every startup and one bad row must not make the
			// app unopenable. (A VOID cell can't appear here: only a waiver voids cells, and it de-rosters
			// the player in the same transaction.)
			if !ok && p.Salary > 0 {
				log.Printf("state: load: WARNING rostered player %q has base salary %s but no PAID %d ledger cell (cell drift) — counting $0 cap until reconciled", p.MFLID, p.Salary, s.season)
			}
			p.CapSalary = cs
		}
	}

	if err := s.applyDeadCap(ctx, fr); err != nil {
		return err
	}
	if err := s.applyCapRelief(ctx, fr); err != nil {
		return err
	}

	s.mu.Lock()
	s.franchises, s.byPlayer = fr, idx
	s.mu.Unlock()
	return nil
}

// applyDeadCap adds this season's dead-cap charges and subtracts cap-relief credits. A franchise
// with dead cap but no players still appears.
func (s *Store) applyDeadCap(ctx context.Context, fr map[string]*FranchiseState) error {
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
	return nil
}

// loadDeadCap sums this season's dead-cap charges per franchise.
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
