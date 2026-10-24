package history

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/secureprospective/TheWarRoom/internal/db"
)

const ledgerAuditSource = "league_event"
const ledgerAuditColumns = `SELECT id,json_array(id,league,stream,revision,effective_at,recorded_at,actor,scope,kind,payload,input_hash,correlation_id,supersedes) FROM league_event`
const ledgerAuditQuery = ledgerAuditColumns + ` ORDER BY rowid`

func initializeLedgerAudit(ctx context.Context, tx *sql.Tx) error {
	if err := db.InitializeAudit(ctx, tx, ledgerAuditSource, ledgerAuditQuery); err != nil {
		return fmt.Errorf("history: initialize audit: %w", err)
	}
	return nil
}

// VerifyAudit verifies raw durable bytes, including legacy command versions, without changing receipts.
func (s *Store) VerifyAudit(ctx context.Context) (db.AuditHead, error) {
	tx, err := s.pools.Read().BeginTx(ctx, nil)
	if err != nil {
		return db.AuditHead{}, fmt.Errorf("history audit: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	head, err := db.VerifyAudit(ctx, tx, ledgerAuditSource, ledgerAuditQuery)
	if err != nil {
		return db.AuditHead{}, fmt.Errorf("history: verify audit: %w", err)
	}
	return head, nil
}

func appendLedgerAudit(ctx context.Context, tx *sql.Tx, id string) error {
	var key, payload string
	if err := tx.QueryRowContext(ctx, ledgerAuditColumns+` WHERE id=?`, id).Scan(&key, &payload); err != nil {
		return fmt.Errorf("history audit row: %w", err)
	}
	if err := db.AppendAudit(ctx, tx, ledgerAuditSource, key, payload); err != nil {
		return fmt.Errorf("history: append audit: %w", err)
	}
	return nil
}
