package db

import (
	"context"
	"database/sql"
	"errors"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"
)

func TestHostedGuardRechecksQueuedWritesAndPreservesState(t *testing.T) {
	pools, err := Open(t.Context(), filepath.Join(t.TempDir(), "server.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = pools.Close() }()
	if _, err := pools.Write().ExecContext(t.Context(), `CREATE TABLE held(value TEXT); INSERT INTO held VALUES('unchanged')`); err != nil {
		t.Fatal(err)
	}
	var covered atomic.Bool
	covered.Store(true)
	refused := errors.New("off-site backup is too old")
	pools.SetWriteGuard(func() error {
		if !covered.Load() {
			return refused
		}
		return nil
	})
	blocker, err := pools.Write().BeginTx(t.Context(), nil)
	if err != nil {
		t.Fatal(err)
	}
	done := make(chan error, 1)
	go func() {
		tx, err := pools.BeginWrite(t.Context())
		if err != nil {
			done <- err
			return
		}
		defer func() { _ = tx.Rollback() }()
		if _, err := tx.ExecContext(t.Context(), `UPDATE held SET value='changed'`); err != nil {
			done <- err
			return
		}
		done <- tx.Commit()
	}()
	deadline := time.Now().Add(2 * time.Second)
	for pools.Write().Stats().WaitCount == 0 && time.Now().Before(deadline) {
		time.Sleep(time.Millisecond)
	}
	if pools.Write().Stats().WaitCount == 0 {
		t.Fatal("write never queued")
	}
	covered.Store(false)
	if err := blocker.Rollback(); err != nil {
		t.Fatal(err)
	}
	if err := <-done; !errors.Is(err, refused) {
		t.Fatalf("queued write bypassed guard: %v", err)
	}
	if _, err := pools.ExecWrite(t.Context(), func(tx *sql.Tx) (sql.Result, error) {
		return tx.ExecContext(t.Context(), `UPDATE held SET value='changed'`)
	}); !errors.Is(err, refused) {
		t.Fatalf("direct write bypassed guard: %v", err)
	}
	var value string
	if err := pools.Read().QueryRowContext(t.Context(), `SELECT value FROM held`).Scan(&value); err != nil || value != "unchanged" {
		t.Fatalf("denial changed state: %s %v", value, err)
	}
	covered.Store(true)
	tx, err := pools.BeginWrite(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := tx.ExecContext(t.Context(), `UPDATE held SET value='recovered'`); err != nil {
		t.Fatal(err)
	}
	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}
}

func TestHostedSnapshotKeepsConfigAndDatabaseBehindSameBarrier(t *testing.T) {
	pools, err := Open(t.Context(), filepath.Join(t.TempDir(), "server.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = pools.Close() }()
	if _, err := pools.Write().ExecContext(t.Context(), `CREATE TABLE held(value TEXT); INSERT INTO held VALUES('before')`); err != nil {
		t.Fatal(err)
	}
	acquired := make(chan struct{})
	release := make(chan struct{})
	captured := make(chan error, 1)
	dest := filepath.Join(t.TempDir(), "server.db")
	go func() {
		captured <- pools.CaptureHosted(t.Context(), dest, func() error { close(acquired); <-release; return nil })
	}()
	<-acquired
	wrote := make(chan error, 1)
	go func() {
		_, err := pools.ExecWrite(t.Context(), func(tx *sql.Tx) (sql.Result, error) {
			return tx.ExecContext(t.Context(), `UPDATE held SET value='after'`)
		})
		wrote <- err
	}()
	select {
	case err := <-wrote:
		t.Fatalf("writer escaped configuration barrier: %v", err)
	case <-time.After(20 * time.Millisecond):
	}
	close(release)
	if err := <-captured; err != nil {
		t.Fatal(err)
	}
	if err := <-wrote; err != nil {
		t.Fatal(err)
	}
	snapshot, err := Open(context.Background(), dest)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = snapshot.Close() }()
	var value string
	if err := snapshot.Read().QueryRowContext(t.Context(), `SELECT value FROM held`).Scan(&value); err != nil || value != "before" {
		t.Fatalf("snapshot torn: %s %v", value, err)
	}
}
