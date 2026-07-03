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

// WriteTx runs fn's mutations as ONE spanning transaction: it takes the single write
// lock, opens one write-pool tx, hands fn a TxWriter bound to that tx, and on success
// commits and reloads memory ONCE. Any error from fn (or a failed step's drift guard)
// rolls the WHOLE transaction back — nothing is committed and memory is untouched, so
// a half-applied trade can never be observed. This is the sole mutation entry point on
// the Writer interface (AD-02: single writer, now single transaction). fn must not
// retain the TxWriter beyond the call — it is invalid once WriteTx returns.
func (s *Store) WriteTx(ctx context.Context, fn func(TxWriter) error) error {
	if fn == nil {
		return fmt.Errorf("state: WriteTx requires a non-nil transaction function")
	}
	s.wmu.Lock()
	defer s.wmu.Unlock()

	// A poisoned store has stale memory from a prior committed-but-not-reloaded tx;
	// refuse to mutate on top of an untrustworthy in-memory view (fail loud).
	if err := s.Err(); err != nil {
		return fmt.Errorf("state: store poisoned — memory is stale from a prior failed reload; refusing to write: %w", err)
	}

	tx, err := s.pools.Write().BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("state: begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	if err := fn(&txWriter{s: s, tx: tx}); err != nil {
		return err // deferred Rollback undoes any steps already executed in this tx
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("state: commit: %w", err)
	}

	// The tx is COMMITTED — the DB is mutated. The in-memory view must now be reloaded
	// to match, or every later read is stale. A reload failure here is NOT a rolled-back
	// transaction: retry once for a transient read hiccup, and if it still fails, POISON
	// the store so subsequent reads/writes fail loud instead of silently serving stale
	// state (GLM-B7a MAJOR — the committed-but-not-reloaded drift window).
	if err := s.reload(ctx); err != nil {
		if err2 := s.reload(ctx); err2 != nil {
			s.poison(err2)
			return fmt.Errorf("state: transaction COMMITTED but in-memory reload failed — state is now STALE, store poisoned: %w", err2)
		}
	}
	return nil
}

// The three single-op mutators remain on *Store for the direct/admin path and the
// store's own tests. Each is just a one-op WriteTx, so the mutation SQL, the exists
// guard, and the memory reload all live in exactly one place (the txWriter methods +
// WriteTx) — a single-step change and a multi-step trade share one code path.

// MovePlayer reassigns a player to another franchise as a standalone transaction.
func (s *Store) MovePlayer(ctx context.Context, mflID, toFranchiseID string) error {
	return s.WriteTx(ctx, func(w TxWriter) error { return w.MovePlayer(ctx, mflID, toFranchiseID) })
}

// SetRosterStatus changes a player's roster status as a standalone transaction.
func (s *Store) SetRosterStatus(ctx context.Context, mflID string, status domain.RosterStatus) error {
	return s.WriteTx(ctx, func(w TxWriter) error { return w.SetRosterStatus(ctx, mflID, status) })
}

// ApplyContract replaces a player's live contract terms as a standalone transaction.
func (s *Store) ApplyContract(ctx context.Context, mflID string, c ContractChange) error {
	return s.WriteTx(ctx, func(w TxWriter) error { return w.ApplyContract(ctx, mflID, c) })
}

// txWriter performs per-player mutations against ONE shared tx. It neither commits nor
// reloads — WriteTx owns the transaction lifecycle. It holds no context: each op takes
// ctx explicitly, matching the TxWriter interface.
type txWriter struct {
	s  *Store
	tx *sql.Tx
}

// MovePlayer reassigns a player to another franchise: it updates the rosters and
// contracts rows in the shared tx. Fails loud on an empty target or an unknown player.
func (w *txWriter) MovePlayer(ctx context.Context, mflID, toFranchiseID string) error {
	if strings.TrimSpace(toFranchiseID) == "" {
		return fmt.Errorf("state: MovePlayer requires a target franchise")
	}
	cur, ok := w.s.currentFranchise(mflID)
	if !ok {
		return fmt.Errorf("state: MovePlayer %q: %w", mflID, errUnknownPlayer)
	}
	if cur == toFranchiseID {
		// A move to the player's own franchise is a no-op that would still touch 1 row
		// and report success — reject it (no-silent-no-op, GLM-B7a). Genuine A↔B swaps
		// are unaffected: each player's target differs from its current franchise.
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

// SetRosterStatus changes a player's roster status (active ↔ taxi/IR) in the shared tx.
// Fails loud on an unknown player or an unrecognized status.
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

// ApplyContract replaces a player's live contract terms (a tag, restructure, extension,
// or year rollover all land here) in the shared tx. Fails loud on an unknown player, an
// invalid contract status, or a negative salary/years.
func (w *txWriter) ApplyContract(ctx context.Context, mflID string, c ContractChange) error {
	if !validContractStatus(c.ContractStatus) {
		return fmt.Errorf("state: ApplyContract: unknown contract status %q", c.ContractStatus)
	}
	if c.AnnualSalary < 0 || c.AdjustedSalary < 0 || c.ContractYears < 0 {
		return fmt.Errorf("state: ApplyContract: negative salary or years")
	}
	if !w.s.exists(mflID) {
		return fmt.Errorf("state: ApplyContract %q: %w", mflID, errUnknownPlayer)
	}
	now := time.Now().UTC().Format(time.RFC3339)
	res, err := w.tx.ExecContext(ctx, `
UPDATE contracts SET annual_salary = ?, adjusted_salary = ?, contract_years = ?,
       expiration_year = ?, contract_status = ?, is_restructured = ?, is_tagged = ?,
       last_updated = ?
WHERE league_id = ? AND season = ? AND mfl_id = ?`,
		c.AnnualSalary, c.AdjustedSalary, c.ContractYears, c.ExpirationYear,
		string(c.ContractStatus), boolToInt(c.IsRestructured), boolToInt(c.IsTagged),
		now, w.s.leagueID, w.s.season, mflID)
	if err != nil {
		return fmt.Errorf("state: apply contract: %w", err)
	}
	return requireOneRow(res, mflID)
}

// exists reports whether the player is currently in state. Caller holds wmu.
func (s *Store) exists(mflID string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	_, ok := s.byPlayer[mflID]
	return ok
}

// currentFranchise returns the franchise a player is currently on. Caller holds wmu.
func (s *Store) currentFranchise(mflID string) (string, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	fid, ok := s.byPlayer[mflID]
	return fid, ok
}

// execPlayer runs a parameterized player-scoped UPDATE on the shared tx. CONTRACT: the
// query's SET clause is filled by setArgs (SET values ONLY, in order), and the helper
// appends the fixed (league_id, season, mfl_id) WHERE tail — so mflID is bound to the
// WHERE, never a SET column, and every mutation is scoped to exactly one player in this
// league+season. A caller that slips an mflID into setArgs (or a SET value into the mflID
// arg) would compile and mis-scope the write; keep setArgs to SET values, nothing else.
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
