package state

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/numeric"
)

// readerView is what Store.Reader hands out: a private pointer implementing only Reader, so a
// consumer cannot type-assert it back to a Writer.
type readerView struct{ s *Store }

func (r readerView) FranchiseState(id string) (FranchiseState, bool) { return r.s.FranchiseState(id) }
func (r readerView) Roster(id string) ([]PlayerState, bool)          { return r.s.Roster(id) }
func (r readerView) CapUsed(id string) (domain.Money, bool)          { return r.s.CapUsed(id) }
func (r readerView) Player(mflID string) (PlayerState, bool)         { return r.s.Player(mflID) }
func (r readerView) Franchises() []string                            { return r.s.Franchises() }

// WriteTx runs fn's ops as one transaction under the write lock, then commits and reloads memory
// once. Any error rolls the whole thing back, so a half-applied trade is never visible. fn must not
// keep the TxWriter after it returns.
func (s *Store) WriteTx(ctx context.Context, fn func(TxWriter) error) error {
	if fn == nil {
		return fmt.Errorf("state: WriteTx requires a non-nil transaction function")
	}
	s.wmu.Lock()
	defer s.wmu.Unlock()

	// A poisoned store has stale memory: refuse to mutate on top of it.
	if err := s.Err(); err != nil {
		return fmt.Errorf("state: store poisoned — memory is stale from a prior failed reload; refusing to write: %w", err)
	}

	tx, err := s.pools.Write().BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("state: begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	if err := fn(&txWriter{s: s, tx: tx}); err != nil {
		return err // undoes any steps already run
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("state: commit: %w", err)
	}

	// The transaction is committed, so memory must be reloaded. Retry once; if it still fails, poison
	// the store so later reads and writes fail loudly instead of serving stale state.
	if err := s.reload(ctx); err != nil {
		if err2 := s.reload(ctx); err2 != nil {
			s.poison(err2)
			return fmt.Errorf("state: transaction COMMITTED but in-memory reload failed — state is now STALE, store poisoned: %w", err2)
		}
	}
	return nil
}

// The single-op mutators below are one-op WriteTx calls, so single changes and multi-step trades
// share one code path.

// MovePlayer reassigns a player to another franchise.
func (s *Store) MovePlayer(ctx context.Context, mflID, toFranchiseID string) error {
	return s.WriteTx(ctx, func(w TxWriter) error { return w.MovePlayer(ctx, mflID, toFranchiseID) })
}

// SetRosterStatus changes a player's roster status.
func (s *Store) SetRosterStatus(ctx context.Context, mflID string, status domain.RosterStatus) error {
	return s.WriteTx(ctx, func(w TxWriter) error { return w.SetRosterStatus(ctx, mflID, status) })
}

// ApplyContract replaces a player's live contract terms.
func (s *Store) ApplyContract(ctx context.Context, mflID string, c ContractChange) error {
	return s.WriteTx(ctx, func(w TxWriter) error { return w.ApplyContract(ctx, mflID, c) })
}

// txWriter runs ops against one shared transaction; WriteTx owns commit and reload.
type txWriter struct {
	s  *Store
	tx *sql.Tx
}

// MovePlayer reassigns a player in the shared transaction.
func (w *txWriter) MovePlayer(ctx context.Context, mflID, toFranchiseID string) error {
	if strings.TrimSpace(toFranchiseID) == "" {
		return fmt.Errorf("state: MovePlayer requires a target franchise")
	}
	cur, ok := w.s.currentFranchise(mflID)
	if !ok {
		return fmt.Errorf("state: MovePlayer %q: %w", mflID, errUnknownPlayer)
	}
	if cur == toFranchiseID {
		// A move to the player's own franchise would touch a row and report success: reject it.
		return fmt.Errorf("state: MovePlayer %q: already on franchise %q (no-op move rejected)", mflID, toFranchiseID)
	}
	now := time.Now().UTC().Format(time.RFC3339)
	if err := w.s.execPlayer(ctx, w.tx,
		`UPDATE rosters SET franchise_id = ?, as_of = ? WHERE league_id = ? AND season = ? AND mfl_id = ?`,
		mflID, toFranchiseID, now); err != nil {
		return err
	}
	return w.s.execPlayer(ctx, w.tx,
		`UPDATE contracts SET franchise_id = ?, last_updated = ? WHERE league_id = ? AND season = ? AND mfl_id = ?`,
		mflID, toFranchiseID, now)
}

// SetRosterStatus changes a player's roster status (active, taxi, IR).
func (w *txWriter) SetRosterStatus(ctx context.Context, mflID string, status domain.RosterStatus) error {
	if !validRosterStatus(status) {
		return fmt.Errorf("state: SetRosterStatus: unknown status %q", status)
	}
	if !w.s.exists(mflID) {
		return fmt.Errorf("state: SetRosterStatus %q: %w", mflID, errUnknownPlayer)
	}
	now := time.Now().UTC().Format(time.RFC3339)
	return w.s.execPlayer(ctx, w.tx,
		`UPDATE rosters SET roster_status = ?, as_of = ? WHERE league_id = ? AND season = ? AND mfl_id = ?`,
		mflID, string(status), now)
}

// ApplyContract replaces a player's live contract terms. Negative salary or years fail.
func (w *txWriter) ApplyContract(ctx context.Context, mflID string, c ContractChange) error {
	if !validContractStatus(c.ContractStatus) {
		return fmt.Errorf("state: ApplyContract: unknown contract status %q", c.ContractStatus)
	}
	if c.AnnualSalary < 0 || c.ContractYears < 0 {
		return fmt.Errorf("state: ApplyContract: negative salary or years")
	}
	if !w.s.exists(mflID) {
		return fmt.Errorf("state: ApplyContract %q: %w", mflID, errUnknownPlayer)
	}
	// Only the base salary is written; the cap figure lives in the ledger cells.
	now := time.Now().UTC().Format(time.RFC3339)
	res, err := w.tx.ExecContext(ctx, `
UPDATE contracts SET annual_salary_cents = ?, contract_years = ?,
       expiration_year = ?, contract_status = ?, is_restructured = ?, is_tagged = ?,
       last_updated = ?
WHERE league_id = ? AND season = ? AND mfl_id = ?`,
		c.AnnualSalary.Cents(), c.ContractYears, c.ExpirationYear,
		string(c.ContractStatus), numeric.BoolToInt(c.IsRestructured), numeric.BoolToInt(c.IsTagged),
		now, w.s.leagueID, w.s.season, mflID)
	if err != nil {
		return fmt.Errorf("state: apply contract: %w", err)
	}
	return requireOneRow(res, mflID)
}

// AddDeadCap appends one dead-cap charge, verbatim; the formula is the handler's. The UNIQUE
// (league, franchise, year, player) key rejects a double charge. That assumes one cut per player
// per franchise per year, which holds while a cut player can't be re-acquired the same season.
// When re-acquisition is modelled, key the ledger on the cut event instead.
func (w *txWriter) AddDeadCap(ctx context.Context, e DeadCapEntry) error {
	if e.DeadCap < 0 {
		return fmt.Errorf("state: AddDeadCap: negative dead cap %d", e.DeadCap)
	}
	if strings.TrimSpace(e.FranchiseID) == "" {
		return fmt.Errorf("state: AddDeadCap requires a franchise")
	}
	if strings.TrimSpace(e.MFLID) == "" {
		return fmt.Errorf("state: AddDeadCap requires a player id")
	}
	if strings.TrimSpace(e.Reason) == "" {
		return fmt.Errorf("state: AddDeadCap requires a reason")
	}
	if e.LeagueYear <= 0 {
		return fmt.Errorf("state: AddDeadCap requires an absolute league year, got %d", e.LeagueYear)
	}
	now := time.Now().UTC().Format(time.RFC3339)
	id := fmt.Sprintf("d:%s:%s:%d:%s", w.s.leagueID, e.FranchiseID, e.LeagueYear, e.MFLID)
	res, err := w.tx.ExecContext(ctx, `
INSERT INTO dead_cap_ledger (id, league_id, franchise_id, league_year, mfl_id,
       dead_cap_cents, reason, created_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		id, w.s.leagueID, e.FranchiseID, e.LeagueYear, e.MFLID, e.DeadCap.Cents(), e.Reason, now)
	if err != nil {
		return fmt.Errorf("state: add dead cap: %w", err)
	}
	return requireOneRow(res, e.MFLID)
}

// ReleasePlayer deletes a player's roster and contract rows and records his destination status,
// written last so every removal carries it. Any dead cap is a separate AddDeadCap in the same
// transaction.
func (w *txWriter) ReleasePlayer(ctx context.Context, mflID string, status domain.PlayerStatus, reason string) error {
	if !w.s.exists(mflID) {
		return fmt.Errorf("state: ReleasePlayer %q: %w", mflID, errUnknownPlayer)
	}
	if err := w.s.execPlayer(ctx, w.tx,
		`DELETE FROM contracts WHERE league_id = ? AND season = ? AND mfl_id = ?`, mflID); err != nil {
		return err
	}
	if err := w.s.execPlayer(ctx, w.tx,
		`DELETE FROM rosters WHERE league_id = ? AND season = ? AND mfl_id = ?`, mflID); err != nil {
		return err
	}
	return w.RecordStatus(ctx, mflID, status, reason)
}

// Player reads the committed snapshot from inside the transaction.
func (w *txWriter) Player(mflID string) (PlayerState, bool) { return w.s.Player(mflID) }

// Season is the absolute league year.
func (w *txWriter) Season() int { return w.s.season }

// OpCount reads the committed counter (0 if unseen). Read, check, then IncOpCount; the single
// writer means nothing can slip in between.
func (w *txWriter) OpCount(ctx context.Context, franchiseID, opKind string) (int, error) {
	var n int
	row := w.s.pools.Read().QueryRowContext(ctx, `
SELECT count FROM transaction_counts
WHERE league_id = ? AND franchise_id = ? AND season = ? AND op_kind = ?`,
		w.s.leagueID, franchiseID, w.s.season, opKind)
	switch err := row.Scan(&n); {
	case errors.Is(err, sql.ErrNoRows):
		return 0, nil
	case err != nil:
		return 0, fmt.Errorf("state: op count (%s/%s): %w", franchiseID, opKind, err)
	default:
		return n, nil
	}
}

// IncOpCount bumps the counter inside the transaction, so a rolled-back op leaves no increment.
func (w *txWriter) IncOpCount(ctx context.Context, franchiseID, opKind string) error {
	if strings.TrimSpace(franchiseID) == "" || strings.TrimSpace(opKind) == "" {
		return fmt.Errorf("state: IncOpCount requires a franchise and op kind")
	}
	_, err := w.tx.ExecContext(ctx, `
INSERT INTO transaction_counts (league_id, franchise_id, season, op_kind, count)
VALUES (?, ?, ?, ?, 1)
ON CONFLICT (league_id, franchise_id, season, op_kind)
DO UPDATE SET count = count + 1`,
		w.s.leagueID, franchiseID, w.s.season, opKind)
	if err != nil {
		return fmt.Errorf("state: inc op count (%s/%s): %w", franchiseID, opKind, err)
	}
	return nil
}

// exists reports whether the player is in state. Caller holds wmu.
func (s *Store) exists(mflID string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	_, ok := s.byPlayer[mflID]
	return ok
}

// currentFranchise returns the player's franchise. Caller holds wmu.
func (s *Store) currentFranchise(mflID string) (string, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	fid, ok := s.byPlayer[mflID]
	return fid, ok
}

// execPlayer runs a player-scoped UPDATE. setArgs fill the SET clause only; the helper appends
// the (league, season, mfl_id) WHERE, so never pass an id in setArgs.
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
