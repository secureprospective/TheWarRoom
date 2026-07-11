package state

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/domain"
)

// cap_relief_ledger is the append-only COMMISSIONER CAP-RELIEF log (§13 Cap Relief Appeal).
// Each row is one relief credit against a franchise's cap for an ABSOLUTE league year — the
// commissioner reducing a cap hit for a career-ending injury, recurring injury, or behavioral
// suspension. It is a FIRST-CLASS cap component in its own ledger, deliberately NOT a negative
// dead_cap_ledger row: dead cap carries a CHECK(>= 0) non-negativity invariant (rollover is off,
// §1), and a relief is a CREDIT, not a charge — mixing signed values into the dead-cap ledger
// would corrupt "dead cap" semantics. CapUsed subtracts this ledger: it is derived as
// Σ(contract cells) + Σ(dead_cap) − Σ(cap_relief), floored at 0 (a franchise can't use negative cap).
//
// Like season_phases / dead_cap_ledger it is DOUBLE-immutable — no update/delete Go API AND
// BEFORE UPDATE/DELETE RAISE(ABORT) triggers, so a raw mutation that bypasses the Go layer still
// aborts (the audit-log idiom). seq is a monotonic INTEGER PRIMARY KEY (rowid alias); a franchise
// may receive MANY reliefs in a season (different players/appeals), so there is no UNIQUE key —
// rows accumulate and are summed. relief_cents is exact cents with a CHECK(> 0): a $0 relief is a
// no-op and is rejected at the Go layer before it reaches here.
const capReliefDDL = `
CREATE TABLE IF NOT EXISTS cap_relief_ledger (
	seq          INTEGER PRIMARY KEY,
	league_id    TEXT NOT NULL,
	franchise_id TEXT NOT NULL,
	league_year  INTEGER NOT NULL,
	relief_cents INTEGER NOT NULL CHECK (relief_cents > 0),
	reason       TEXT NOT NULL,
	created_at   TEXT NOT NULL
);
CREATE TRIGGER IF NOT EXISTS cap_relief_ledger_no_update
BEFORE UPDATE ON cap_relief_ledger
BEGIN SELECT RAISE(ABORT, 'cap_relief_ledger is append-only'); END;
CREATE TRIGGER IF NOT EXISTS cap_relief_ledger_no_delete
BEFORE DELETE ON cap_relief_ledger
BEGIN SELECT RAISE(ABORT, 'cap_relief_ledger is append-only'); END;`

// initCapReliefSchema creates the cap_relief_ledger table and its immutability triggers.
// Called from initSchema, mirroring initSeasonPhaseSchema / initLedgerSchema (own file,
// store-no-siblings + the 400-line cap).
func (s *Store) initCapReliefSchema(ctx context.Context) error {
	if _, err := s.pools.Write().ExecContext(ctx, capReliefDDL); err != nil {
		return fmt.Errorf("state: init cap-relief schema: %w", err)
	}
	return nil
}

// AddCapRelief appends one commissioner cap-relief credit to the ledger in the shared tx — the
// §13 Cap Relief Appeal write primitive. It is APPEND-ONLY (no update/delete here, and the DB
// triggers reject a raw mutation). The store records the credit VERBATIM; whether the appeal is
// justified is the commissioner's judgment, not this pure data layer's. Fails loud on a
// non-positive amount (a $0 relief is a no-op), a missing field, or a non-absolute league year.
func (w *txWriter) AddCapRelief(ctx context.Context, e CapReliefEntry) error {
	if e.Amount <= 0 {
		return fmt.Errorf("state: AddCapRelief: relief must be positive, got %d", e.Amount)
	}
	if strings.TrimSpace(e.FranchiseID) == "" {
		return fmt.Errorf("state: AddCapRelief requires a franchise")
	}
	if strings.TrimSpace(e.Reason) == "" {
		return fmt.Errorf("state: AddCapRelief requires a reason")
	}
	if e.LeagueYear <= 0 {
		return fmt.Errorf("state: AddCapRelief requires an absolute league year, got %d", e.LeagueYear)
	}
	now := time.Now().UTC().Format(time.RFC3339)
	if _, err := w.tx.ExecContext(ctx, `
INSERT INTO cap_relief_ledger (league_id, franchise_id, league_year, relief_cents, reason, created_at)
VALUES (?, ?, ?, ?, ?, ?)`,
		w.s.leagueID, e.FranchiseID, e.LeagueYear, e.Amount.Cents(), e.Reason, now); err != nil {
		return fmt.Errorf("state: add cap relief: %w", err)
	}
	return nil
}

// applyCapRelief SUBTRACTS this season's commissioner cap-relief credits (§13) from the derived
// per-franchise CapUsed. It is floored at 0 — a franchise can't use negative cap, so an
// over-generous relief (more than the current hit) just zeroes it. A relief for a franchise with
// no current cap component is ignored (nothing to reduce): it would floor to 0 anyway, so we do
// not materialize a phantom franchise for relief alone.
func (s *Store) applyCapRelief(ctx context.Context, fr map[string]*FranchiseState) error {
	cr, err := s.loadCapRelief(ctx)
	if err != nil {
		return err
	}
	for fid, amt := range cr {
		f, ok := fr[fid]
		if !ok {
			continue
		}
		if amt >= f.CapUsed {
			f.CapUsed = 0
		} else {
			f.CapUsed -= amt
		}
	}
	return nil
}

// loadCapRelief sums the cap-relief credits per franchise for THIS store's season (league_year ==
// season — relief is keyed to an absolute year, and CapUsed is a current-season figure). Returns
// cents-typed Money keyed by franchise id.
func (s *Store) loadCapRelief(ctx context.Context) (map[string]domain.Money, error) {
	rows, err := s.pools.Read().QueryContext(ctx, `
SELECT franchise_id, COALESCE(SUM(relief_cents), 0)
FROM cap_relief_ledger
WHERE league_id = ? AND league_year = ?
GROUP BY franchise_id`, s.leagueID, s.season)
	if err != nil {
		return nil, fmt.Errorf("state: load cap relief: %w", err)
	}
	defer func() { _ = rows.Close() }()

	out := map[string]domain.Money{}
	for rows.Next() {
		var fid string
		var cents int64
		if err := rows.Scan(&fid, &cents); err != nil {
			return nil, fmt.Errorf("state: cap relief scan: %w", err)
		}
		out[fid] = domain.Money(cents)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("state: cap relief iterate: %w", err)
	}
	return out, nil
}
