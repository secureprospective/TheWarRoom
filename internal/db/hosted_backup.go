package db

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

// SetWriteGuard installs hosted durability admission before serving. Desktop pools leave it nil.
// The guard is checked after acquiring the single writer, including queued requests.
func (p *Pools) SetWriteGuard(guard func() error) { p.writeGuard = guard }

func (p *Pools) BeginWrite(ctx context.Context) (*sql.Tx, error) {
	if p.writeGuard != nil {
		// The returned SQL transaction rolls back automatically at its context deadline.
		// Its admission margin is one minute; no hosted transaction can outlive thirty seconds.
		bounded, cancel := context.WithTimeout(ctx, 30*time.Second)
		context.AfterFunc(bounded, cancel)
		ctx = bounded
	}
	tx, err := p.write.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("begin database write: %w", err)
	}
	if p.writeGuard != nil {
		if err := p.writeGuard(); err != nil {
			_ = tx.Rollback()
			return nil, fmt.Errorf("database write paused: %w", err)
		}
	}
	return tx, nil
}

func (p *Pools) ExecWrite(ctx context.Context, execute func(*sql.Tx) (sql.Result, error)) (sql.Result, error) {
	tx, err := p.BeginWrite(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback() }()
	result, err := execute(tx)
	if err != nil {
		return nil, fmt.Errorf("execute database write: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit database write: %w", err)
	}
	return result, nil
}

// CaptureHosted writes a compacted, verified snapshot of the database to dest (which must not
// exist). It reserves the same writer as history, evidence and accounts only long enough to pin the
// snapshot's read point and run config, so config and database belong to one point; the copy itself
// runs after the writer is released, and writes commit while it runs. A nil config is allowed. Never
// copy a live WAL file. The intermediate full copy is dest+".copy", removed before returning.
func (p *Pools) CaptureHosted(ctx context.Context, dest string, config func() error) error {
	pin, err := p.pinUnderWriter(ctx, config)
	if err != nil {
		return err
	}
	copyPath := dest + ".copy"
	_, err = pin.copyTo(ctx, copyPath, dest)
	if err == nil {
		err = removeSnapshot(copyPath)
	}
	if err != nil {
		return fmt.Errorf("snapshot hosted database: %w", err)
	}
	return nil
}

// pinUnderWriter waits for the writer (so no write transaction is open), pins a snapshot read point,
// runs barrier while the writer is still held, then releases the writer.
func (p *Pools) pinUnderWriter(ctx context.Context, barrier func() error) (*snapshotPin, error) {
	conn, err := p.write.Conn(ctx)
	if err != nil {
		return nil, fmt.Errorf("reserve backup writer: %w", err)
	}
	defer func() { _ = conn.Close() }()
	pin, err := p.pinSnapshot(ctx)
	if err != nil {
		return nil, err
	}
	if barrier != nil {
		if err := barrier(); err != nil {
			pin.release(ctx)
			return nil, err
		}
	}
	return pin, nil
}
