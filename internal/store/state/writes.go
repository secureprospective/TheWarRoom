package state

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/domain"
)

// readerView is the concrete type Store.Reader hands out. It embeds NO *Store, only a
// private pointer, and implements ONLY Reader by delegation — so a consumer
// cannot type-assert it back to Writer. This is the compile/runtime enforcement
// of the single-writer law (proven by a planted test).
type readerView struct{ s *Store }

func (r readerView) FranchiseState(id string) (FranchiseState, bool) { return r.s.FranchiseState(id) }
func (r readerView) Roster(id string) ([]PlayerState, bool)          { return r.s.Roster(id) }
func (r readerView) CapUsed(id string) (float64, bool)               { return r.s.CapUsed(id) }
func (r readerView) Player(mflID string) (PlayerState, bool)         { return r.s.Player(mflID) }
func (r readerView) Franchises() []string                            { return r.s.Franchises() }

// MovePlayer reassigns a player to another franchise (the roster-move primitive B7
// builds acquisitions and trades on). It updates the rosters and contracts rows in
// one transaction, then reloads memory. Fails loud on an unknown player.
func (s *Store) MovePlayer(ctx context.Context, mflID, toFranchiseID string) error {
	if strings.TrimSpace(toFranchiseID) == "" {
		return fmt.Errorf("state: MovePlayer requires a target franchise")
	}
	s.wmu.Lock()
	defer s.wmu.Unlock()
	if !s.exists(mflID) {
		return fmt.Errorf("state: MovePlayer %q: %w", mflID, errUnknownPlayer)
	}
	now := time.Now().UTC().Format(time.RFC3339)
	err := s.inTx(ctx, func(tx *sql.Tx) error {
		if err := s.execPlayer(ctx, tx,
			`UPDATE rosters SET franchise_id = ?, as_of = ? WHERE league_id = ? AND season = ? AND mfl_id = ?`,
			mflID, toFranchiseID, now); err != nil {
			return err
		}
		return s.execPlayer(ctx, tx,
			`UPDATE contracts SET franchise_id = ?, last_updated = ? WHERE league_id = ? AND season = ? AND mfl_id = ?`,
			mflID, toFranchiseID, now)
	})
	if err != nil {
		return err
	}
	return s.load(ctx)
}

// SetRosterStatus changes a player's roster status (active ↔ taxi/IR). Fails loud on
// an unknown player or an unrecognized status.
func (s *Store) SetRosterStatus(ctx context.Context, mflID string, status domain.RosterStatus) error {
	if !validRosterStatus(status) {
		return fmt.Errorf("state: SetRosterStatus: unknown status %q", status)
	}
	s.wmu.Lock()
	defer s.wmu.Unlock()
	if !s.exists(mflID) {
		return fmt.Errorf("state: SetRosterStatus %q: %w", mflID, errUnknownPlayer)
	}
	now := time.Now().UTC().Format(time.RFC3339)
	err := s.inTx(ctx, func(tx *sql.Tx) error {
		return s.execPlayer(ctx, tx,
			`UPDATE rosters SET roster_status = ?, as_of = ? WHERE league_id = ? AND season = ? AND mfl_id = ?`,
			mflID, string(status), now)
	})
	if err != nil {
		return err
	}
	return s.load(ctx)
}

// ApplyContract replaces a player's live contract terms (a tag, restructure,
// extension, or year rollover all land here). Fails loud on an unknown player or an
// invalid contract status.
func (s *Store) ApplyContract(ctx context.Context, mflID string, c ContractChange) error {
	if !validContractStatus(c.ContractStatus) {
		return fmt.Errorf("state: ApplyContract: unknown contract status %q", c.ContractStatus)
	}
	if c.AnnualSalary < 0 || c.AdjustedSalary < 0 || c.ContractYears < 0 {
		return fmt.Errorf("state: ApplyContract: negative salary or years")
	}
	s.wmu.Lock()
	defer s.wmu.Unlock()
	if !s.exists(mflID) {
		return fmt.Errorf("state: ApplyContract %q: %w", mflID, errUnknownPlayer)
	}
	now := time.Now().UTC().Format(time.RFC3339)
	err := s.inTx(ctx, func(tx *sql.Tx) error {
		res, err := tx.ExecContext(ctx, `
UPDATE contracts SET annual_salary = ?, adjusted_salary = ?, contract_years = ?,
       expiration_year = ?, contract_status = ?, is_restructured = ?, is_tagged = ?,
       last_updated = ?
WHERE league_id = ? AND season = ? AND mfl_id = ?`,
			c.AnnualSalary, c.AdjustedSalary, c.ContractYears, c.ExpirationYear,
			string(c.ContractStatus), boolToInt(c.IsRestructured), boolToInt(c.IsTagged),
			now, s.leagueID, s.season, mflID)
		if err != nil {
			return fmt.Errorf("state: apply contract: %w", err)
		}
		return requireOneRow(res, mflID)
	})
	if err != nil {
		return err
	}
	return s.load(ctx)
}

// exists reports whether the player is currently in state. Caller holds wmu.
func (s *Store) exists(mflID string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	_, ok := s.byPlayer[mflID]
	return ok
}

// inTx runs fn inside a single write-pool transaction with rollback-on-error.
func (s *Store) inTx(ctx context.Context, fn func(*sql.Tx) error) error {
	tx, err := s.pools.Write().BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("state: begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	if err := fn(tx); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("state: commit: %w", err)
	}
	return nil
}

// execPlayer runs a parameterized player-scoped UPDATE. setArgs fill the SET-clause
// placeholders; the helper appends the fixed (league_id, season, mfl_id) WHERE tail,
// so every mutation is scoped to exactly one player in this league+season.
func (s *Store) execPlayer(ctx context.Context, tx *sql.Tx, query, mflID string, setArgs ...any) error {
	args := make([]any, 0, len(setArgs)+3)
	args = append(args, setArgs...)
	args = append(args, s.leagueID, s.season, mflID)
	res, err := tx.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("state: player write: %w", err)
	}
	return requireOneRow(res, mflID)
}
