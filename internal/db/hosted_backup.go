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

// CaptureHosted reserves the same writer as history, evidence and accounts. The caller
// copies immutable server configuration while this barrier is held. Never copy a live WAL file.
func (p *Pools) CaptureHosted(ctx context.Context, dest string, config func() error) error {
	conn, err := p.write.Conn(ctx)
	if err != nil {
		return fmt.Errorf("reserve backup writer: %w", err)
	}
	defer func() { _ = conn.Close() }()
	if err := createOwnerOnly(dest); err != nil {
		return err
	}
	if _, err := conn.ExecContext(ctx, "VACUUM INTO ?", dest); err != nil {
		return fmt.Errorf("snapshot hosted database: %w", err)
	}
	return config()
}
