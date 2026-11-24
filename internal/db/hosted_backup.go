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
	tx, err := p.write.BeginTx(p.boundWrite(ctx), nil)
	if err != nil {
		return nil, fmt.Errorf("begin database write: %w", err)
	}
	return p.guardWrite(tx)
}

func (p *Pools) ExecWrite(ctx context.Context, execute func(*sql.Tx) (sql.Result, error)) (sql.Result, error) {
	tx, err := p.BeginWrite(ctx)
	if err != nil {
		return nil, err
	}
	return finishWrite(tx, execute)
}

// ExecWriteBounded runs a write only if the single writer is acquired within acquireTimeout.
func (p *Pools) ExecWriteBounded(ctx context.Context, acquireTimeout time.Duration, execute func(*sql.Tx) (sql.Result, error)) (sql.Result, error) {
	acquireCtx, cancel := context.WithTimeout(ctx, acquireTimeout)
	defer cancel()
	conn, err := p.write.Conn(acquireCtx)
	if err != nil {
		return nil, fmt.Errorf("acquire database writer: %w", err)
	}
	defer func() { _ = conn.Close() }()
	tx, err := conn.BeginTx(p.boundWrite(ctx), nil)
	if err != nil {
		return nil, fmt.Errorf("begin database write: %w", err)
	}
	tx, err = p.guardWrite(tx)
	if err != nil {
		return nil, err
	}
	return finishWrite(tx, execute)
}

func (p *Pools) boundWrite(ctx context.Context) context.Context {
	if p.writeGuard == nil {
		return ctx
	}
	// The returned SQL transaction rolls back automatically at its context deadline.
	// Its admission margin is one minute; no hosted transaction can outlive thirty seconds.
	bounded, cancel := context.WithTimeout(ctx, 30*time.Second)
	context.AfterFunc(bounded, cancel)
	return bounded
}

func (p *Pools) guardWrite(tx *sql.Tx) (*sql.Tx, error) {
	if p.writeGuard != nil {
		if err := p.writeGuard(); err != nil {
			_ = tx.Rollback()
			return nil, fmt.Errorf("database write paused: %w", err)
		}
	}
	return tx, nil
}

func finishWrite(tx *sql.Tx, execute func(*sql.Tx) (sql.Result, error)) (sql.Result, error) {
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
