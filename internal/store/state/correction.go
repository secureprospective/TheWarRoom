package state

import (
	"context"
	"fmt"
	"strings"
	"time"
)

// transaction_corrections is the append-only TRANSACTION-CORRECTION log — Session 2's
// retroactive-correction surface for the activity feed. A correction (clerical fix or
// reversal) NEVER updates or deletes the original ledger row it targets; it is a NEW row
// here carrying the SAME logical transaction id (tx_id) as the entry it corrects, with a
// status (CORRECTED / REVERSED). The CURRENT correction state of a logical entry is its
// latest row by MAX(seq), exactly the calendar_events / season_phases / cap_relief_ledger
// latest-row-wins idiom. Full correction history is preserved for the audit trail.
//
// It mirrors calendar_events deliberately: same shape (seq INTEGER PRIMARY KEY rowid alias,
// league_id, the stable logical id, kind, status, note, created_at), same double-immutability
// (no update/delete Go API AND BEFORE UPDATE/DELETE RAISE(ABORT) triggers), same latest-row
// read. The commissioner-facing "I don't want to see the mistake and the fix cluttering the
// view" UX is served by the reconciled read projection (Corrections, consumed by the feed IPC)
// — the WRITE path stays pure append-only, never a true delete/undo path (the panel-gated call).
//
// tx_id is the logical id shared between the original feed row and every correction of it:
// the feed composes it as Source + ":" + ID (e.g. "trade_notes:tx:…", "player_status_events:42",
// "dead_cap_ledger:d:…", "contract_year_changes:cyc:…"), so a correction row joins back to its
// original by exact string equality on tx_id. kind is the corrected entry's transaction Kind
// (TRADE/SIGN/WAIVER/…) denormalized here so the projection can style the correction without a
// join back to the source table. status is one of CORRECTED (the original's note/metadata is
// amended; its EFFECT still holds) or REVERSED (the original's EFFECT is marked undone for
// net-state purposes — the row is never deleted; the reconciled feed excludes it from net
// totals and renders it struck-through).
//
// Like every sibling audit log it is DOUBLE-immutable: no update/delete Go API AND the DB
// triggers reject a raw mutation that bypasses the Go layer. seq is a monotonic INTEGER
// PRIMARY KEY (rowid alias): inserts serialize under wmu and rows are never deleted, so seq
// strictly increases in insertion order — "latest row per tx_id" is an unambiguous MAX(seq).
const correctionDDL = `
CREATE TABLE IF NOT EXISTS transaction_corrections (
	seq          INTEGER PRIMARY KEY,
	league_id    TEXT NOT NULL,
	tx_id        TEXT NOT NULL,
	kind         TEXT NOT NULL,
	status       TEXT NOT NULL,
	commissioner TEXT NOT NULL,
	reason       TEXT NOT NULL,
	note         TEXT NOT NULL DEFAULT '',
	created_at   TEXT NOT NULL
);
CREATE TRIGGER IF NOT EXISTS transaction_corrections_no_update
BEFORE UPDATE ON transaction_corrections
BEGIN SELECT RAISE(ABORT, 'transaction_corrections is append-only'); END;
CREATE TRIGGER IF NOT EXISTS transaction_corrections_no_delete
BEFORE DELETE ON transaction_corrections
BEGIN SELECT RAISE(ABORT, 'transaction_corrections is append-only'); END;`

// Correction status values — the discriminator on each correction row. A logical entry's
// current correction state is its latest row's status (POSTED is implicit: an entry with no
// correction row is still POSTED in its source ledger).
const (
	// CorrStatusCorrected amends the original's note/metadata; the original's EFFECT still holds.
	// The reconciled feed shows the correction's note/reason alongside the original.
	CorrStatusCorrected = "CORRECTED"
	// CorrStatusReversed marks the original's EFFECT undone for net-state purposes. The original
	// ledger row is NEVER touched (append-only honored); the reconciled feed renders the original
	// struck-through and excludes it from net totals.
	CorrStatusReversed = "REVERSED"
)

// validCorrStatus reports whether s is one of the known correction statuses — the whitelist the
// append primitive enforces so a drift value never reaches the log.
func validCorrStatus(s string) bool {
	switch s {
	case CorrStatusCorrected, CorrStatusReversed:
		return true
	default:
		return false
	}
}

// CorrectionEntry is one append-only correction row as it crosses the store boundary on a WRITE.
// The caller supplies every field except CreatedAt (the store stamps it). TxID is the logical id
// shared with the corrected feed row (Source + ":" + ID); Kind is the corrected entry's
// transaction Kind; Status is CORRECTED or REVERSED. Commissioner + Reason are required for the
// audit trail (who issued the correction and why).
type CorrectionEntry struct {
	TxID         string
	Kind         string
	Status       string
	Commissioner string
	Reason       string
	Note         string
}

// CorrectionRow is one correction row as it crosses the store boundary on a READ — a
// CorrectionEntry plus the store-stamped Seq and CreatedAt. Ordered by Seq ascending from the
// store; the caller (the feed IPC) reduces to latest-per-tx_id for the reconciled projection.
type CorrectionRow struct {
	Seq int64
	CorrectionEntry
	CreatedAt string
}

// initCorrectionSchema creates the transaction_corrections table and its immutability triggers.
// Called from initSchema, mirroring initCalendarSchema / initCapReliefSchema / initSeasonPhaseSchema
// (own file, store-no-siblings + the 400-line cap). Idempotent: CREATE TABLE IF NOT EXISTS + the
// trigger's IF NOT EXISTS make it a no-op on an already-initialized DB (no migration needed — new
// append-only tables are additive, like every sibling ledger table).
func (s *Store) initCorrectionSchema(ctx context.Context) error {
	if _, err := s.pools.Write().ExecContext(ctx, correctionDDL); err != nil {
		return fmt.Errorf("state: init correction schema: %w", err)
	}
	return nil
}

// AppendCorrection appends one row to the correction log in the shared tx — Session 2's
// retroactive-correction write primitive behind the CORRECT transaction. It is APPEND-ONLY (no
// update/delete here, and the DB triggers reject a raw mutation): a second correction of the same
// entry is a NEW row carrying the same tx_id, never an edit of the prior correction. The store
// records the correction VERBATIM; whether reversing an entry should also post a compensating
// transaction is the commissioner's call, not this pure data layer's — a REVERSED marker is the
// audit signal + the display flag (the panel-gated "no true-delete/undo path"). Fails loud on a
// missing field or an unknown status.
func (w *txWriter) AppendCorrection(ctx context.Context, e CorrectionEntry) error {
	if strings.TrimSpace(e.TxID) == "" {
		return fmt.Errorf("state: AppendCorrection requires a tx id")
	}
	// The documented shape is Source + ":" + ID (e.g. "trade_notes:14432"). A tx_id without the
	// separator can never join to a real feed row — it would create a silently orphaned correction
	// (a review finding). This is a format guard only; validating Source against the actual
	// enumerated table names is the transactions package's job (it owns that vocabulary).
	if !strings.Contains(e.TxID, ":") {
		return fmt.Errorf("state: AppendCorrection: tx id %q is not in Source:ID form", e.TxID)
	}
	if strings.TrimSpace(e.Kind) == "" {
		return fmt.Errorf("state: AppendCorrection requires a kind")
	}
	if !validCorrStatus(e.Status) {
		return fmt.Errorf("state: AppendCorrection: unknown status %q", e.Status)
	}
	if strings.TrimSpace(e.Commissioner) == "" {
		return fmt.Errorf("state: AppendCorrection requires a commissioner (who issued it)")
	}
	if strings.TrimSpace(e.Reason) == "" {
		return fmt.Errorf("state: AppendCorrection requires a reason (audit trail)")
	}
	now := time.Now().UTC().Format(time.RFC3339)
	if _, err := w.tx.ExecContext(ctx, `
INSERT INTO transaction_corrections (league_id, tx_id, kind, status, commissioner, reason, note, created_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		w.s.leagueID, e.TxID, e.Kind, e.Status, e.Commissioner, e.Reason, e.Note, now); err != nil {
		return fmt.Errorf("state: append correction %q: %w", e.TxID, err)
	}
	return nil
}

// Corrections returns every correction row for this league, ordered by seq ascending — the
// reconciled/net-state read projection. The feed IPC consumes this and reduces to latest-per-tx_id
// (the current correction state of each logical entry): a CORRECTED entry shows the correction's
// note/reason alongside the original; a REVERSED entry renders the original struck-through and is
// excluded from net totals. The original ledger rows are NEVER touched (append-only honored);
// this projection is the read-side reconciliation the panel asked for. Never returns a nil slice
// on success (empty is a valid — uncorrected — league).
func (s *Store) Corrections(ctx context.Context) ([]CorrectionRow, error) {
	rows, err := s.pools.Read().QueryContext(ctx, `
SELECT seq, tx_id, kind, status, commissioner, reason, note, created_at
FROM transaction_corrections
WHERE league_id = ?
ORDER BY seq`, s.leagueID)
	if err != nil {
		return nil, fmt.Errorf("state: corrections: %w", err)
	}
	defer func() { _ = rows.Close() }()

	out := make([]CorrectionRow, 0)
	for rows.Next() {
		var r CorrectionRow
		if serr := rows.Scan(&r.Seq, &r.TxID, &r.Kind, &r.Status, &r.Commissioner, &r.Reason, &r.Note, &r.CreatedAt); serr != nil {
			return nil, fmt.Errorf("state: corrections scan: %w", serr)
		}
		out = append(out, r)
	}
	if ierr := rows.Err(); ierr != nil {
		return nil, fmt.Errorf("state: corrections iterate: %w", ierr)
	}
	return out, nil
}
