package state

import (
	"context"
	"fmt"
	"strings"
	"time"
)

// transaction_corrections is the append-only correction log. A correction never touches the
// original row: it is a new row with the same tx_id (the feed's Source + ":" + ID) and a status.
//
//	CORRECTED  the original's note is amended; its effect stands
//	REVERSED   the original's effect is marked undone; the row is never deleted
//
// The latest row per tx_id (by seq) is the current state. kind is copied from the original so the
// feed can style a correction without a join. Triggers abort any update or delete.
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

// Correction statuses. An entry with no correction row is POSTED.
const (
	CorrStatusCorrected = "CORRECTED"
	CorrStatusReversed  = "REVERSED"
)

// validCorrStatus guards the append, so an unknown status never reaches the log.
func validCorrStatus(s string) bool {
	switch s {
	case CorrStatusCorrected, CorrStatusReversed:
		return true
	default:
		return false
	}
}

// CorrectionEntry is one correction to append. CreatedAt is stamped by the store; Commissioner
// and Reason are required.
type CorrectionEntry struct {
	TxID         string
	Kind         string
	Status       string
	Commissioner string
	Reason       string
	Note         string
}

// CorrectionRow is a stored correction, with Seq and CreatedAt.
type CorrectionRow struct {
	Seq int64
	CorrectionEntry
	CreatedAt string
}

// initCorrectionSchema creates the table and its immutability triggers. Idempotent.
func (s *Store) initCorrectionSchema(ctx context.Context) error {
	if _, err := s.pools.Write().ExecContext(ctx, correctionDDL); err != nil {
		return fmt.Errorf("state: init correction schema: %w", err)
	}
	return nil
}

// AppendCorrection appends one correction. Correcting an entry twice adds a second row. Whether a
// reversal should also post a compensating transaction is the commissioner's call; this records
// the marker only.
func (w *txWriter) AppendCorrection(ctx context.Context, e CorrectionEntry) error {
	if strings.TrimSpace(e.TxID) == "" {
		return fmt.Errorf("state: AppendCorrection requires a tx id")
	}
	// A tx_id without ":" can never join to a feed row and would orphan the correction.
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

// Corrections returns every correction for the league in seq order; the caller reduces to the
// latest per tx_id. Never nil on success.
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
