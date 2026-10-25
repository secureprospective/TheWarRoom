package history

import (
	"bytes"
	"compress/gzip"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/archive"
)

// ImportReceipt identifies the immutable staged copy, not permission to activate it.
type ImportReceipt struct {
	League, SourceEpoch, Content string
	RecordedAt                   time.Time
}

func addInactiveImports(ctx context.Context, tx *sql.Tx) error {
	const ddl = `
CREATE TABLE league_import (
 league TEXT PRIMARY KEY,
 source_epoch TEXT NOT NULL,
 content TEXT NOT NULL,
 recorded_at TEXT NOT NULL,
 document BLOB NOT NULL
);
CREATE TRIGGER league_import_empty BEFORE INSERT ON league_import
WHEN EXISTS(SELECT 1 FROM authority_policy WHERE league=NEW.league)
BEGIN SELECT RAISE(ABORT,'the destination already has league authority settings'); END;
CREATE TRIGGER league_import_event_guard BEFORE INSERT ON league_event
WHEN EXISTS(SELECT 1 FROM league_import WHERE league=NEW.league)
BEGIN SELECT RAISE(ABORT,'this imported league is inactive'); END;
CREATE TRIGGER league_import_authority_insert BEFORE INSERT ON authority_policy
WHEN EXISTS(SELECT 1 FROM league_import WHERE league=NEW.league)
BEGIN SELECT RAISE(ABORT,'this imported league is inactive'); END;
CREATE TRIGGER league_import_authority_update BEFORE UPDATE ON authority_policy
WHEN EXISTS(SELECT 1 FROM league_import WHERE league=NEW.league)
BEGIN SELECT RAISE(ABORT,'this imported league is inactive'); END;`
	if _, err := tx.ExecContext(ctx, ddl+appendOnlyDDL("league_import")); err != nil {
		return fmt.Errorf("prepare inactive league imports: %w", err)
	}
	return nil
}

// StageTransfer is called only after the transfer service validates and replays the document.
// The immediate transaction serializes collision checks, archive reuse, original rows and receipt
// across processes. The receipt is installed last so its trigger fences every subsequent append.
func (s *Store) StageTransfer(ctx context.Context, receipt ImportReceipt, document []byte, events []RawTransferEvent, blobs map[string][]byte) (ImportReceipt, error) {
	s.wmu.Lock()
	defer s.wmu.Unlock()
	tx, err := s.pools.Write().BeginTx(ctx, nil)
	if err != nil {
		return ImportReceipt{}, fmt.Errorf("begin inactive league import: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	prior, err := importReceipt(ctx, tx, receipt.League)
	if err == nil {
		if prior.SourceEpoch != receipt.SourceEpoch || prior.Content != receipt.Content {
			return ImportReceipt{}, fmt.Errorf("this league already has a different imported copy; keep the original file and source generation")
		}
		return prior, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return ImportReceipt{}, err
	}
	if err := checkEmptyImport(ctx, tx, receipt.League); err != nil {
		return ImportReceipt{}, err
	}
	for key, body := range blobs {
		if err := stageArchive(ctx, tx, key, body); err != nil {
			return ImportReceipt{}, err
		}
	}
	for _, event := range events {
		if event.League != receipt.League {
			return ImportReceipt{}, fmt.Errorf("the import contains another league")
		}
		if err := stageEvent(ctx, tx, event); err != nil {
			return ImportReceipt{}, err
		}
	}
	receipt.RecordedAt = s.now().UTC()
	if _, err := tx.ExecContext(ctx, `INSERT INTO league_import VALUES(?,?,?,?,?)`, receipt.League, receipt.SourceEpoch, receipt.Content, receipt.RecordedAt.Format(time.RFC3339Nano), document); err != nil {
		return ImportReceipt{}, fmt.Errorf("save inactive league receipt: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return ImportReceipt{}, fmt.Errorf("commit inactive league import: %w", err)
	}
	return receipt, nil
}

func importReceipt(ctx context.Context, tx *sql.Tx, league string) (ImportReceipt, error) {
	var r ImportReceipt
	var at string
	err := tx.QueryRowContext(ctx, `SELECT league,source_epoch,content,recorded_at FROM league_import WHERE league=?`, league).Scan(&r.League, &r.SourceEpoch, &r.Content, &at)
	if err != nil {
		return r, fmt.Errorf("read imported league receipt: %w", err)
	}
	r.RecordedAt, err = parseEventTime(at)
	return r, err
}

func checkEmptyImport(ctx context.Context, tx *sql.Tx, league string) error {
	var occupied bool
	if err := tx.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM league_event WHERE league=? UNION ALL SELECT 1 FROM authority_policy WHERE league=?)`, league, league).Scan(&occupied); err != nil {
		return fmt.Errorf("check destination league: %w", err)
	}
	if occupied {
		return fmt.Errorf("this computer already holds that league; import into a different destination")
	}
	return nil
}

func stageEvent(ctx context.Context, tx *sql.Tx, e RawTransferEvent) error {
	if err := e.Validate(); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO league_event(id,league,stream,revision,effective_at,recorded_at,actor,scope,kind,payload,input_hash,correlation_id,supersedes) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?)`, e.ID, e.League, e.Stream, e.Revision, e.EffectiveAt, e.RecordedAt, e.Actor, e.Scope, e.Kind, e.Payload, e.InputHash, e.CorrelationID, e.Supersedes); err != nil {
		return fmt.Errorf("save original league change: %w", err)
	}
	return appendLedgerAudit(ctx, tx, e.ID)
}

// Inspect the winning archive row under the writer transaction, not before INSERT OR IGNORE:
// another process may already have stored damaged bytes under this digest.
func stageArchive(ctx context.Context, tx *sql.Tx, key string, body []byte) error {
	if footballDigest(body) != key {
		return fmt.Errorf("imported football evidence has changed")
	}
	if _, err := TransferFootball(body); err != nil {
		return err
	}
	var compressed bytes.Buffer
	writer := gzip.NewWriter(&compressed)
	if _, err := writer.Write(body); err != nil {
		return fmt.Errorf("compress imported evidence: %w", err)
	}
	if err := writer.Close(); err != nil {
		return fmt.Errorf("finish imported evidence: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `INSERT OR IGNORE INTO raw_archive(sha256,size,body) VALUES(?,?,?)`, key, len(body), compressed.Bytes()); err != nil {
		return fmt.Errorf("save imported evidence: %w", err)
	}
	var held []byte
	var size int64
	if err := tx.QueryRowContext(ctx, `SELECT size,body FROM raw_archive WHERE sha256=?`, key).Scan(&size, &held); err != nil {
		return fmt.Errorf("check stored evidence: %w", err)
	}
	actual, err := archive.Gunzip(held, size)
	if err != nil {
		return fmt.Errorf("check stored evidence: %w", err)
	}
	if !bytes.Equal(actual, body) {
		return fmt.Errorf("stored football evidence conflicts with this import")
	}
	return nil
}

// ImportDocument returns the original validated artifact for comparison at a later cutover.
// Its aliases and source checkpoint remain evidence, never target account grants or audit heads.
func (s *Store) ImportDocument(ctx context.Context, league string) ([]byte, error) {
	var body []byte
	if err := s.pools.Read().QueryRowContext(ctx, `SELECT document FROM league_import WHERE league=?`, league).Scan(&body); err != nil {
		return nil, fmt.Errorf("read inactive league document: %w", err)
	}
	return body, nil
}
