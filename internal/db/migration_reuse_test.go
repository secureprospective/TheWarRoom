package db

import (
	"context"
	"path/filepath"
	"testing"
	"time"
)

func TestMigrationReinitializationDoesNotWaitForWriter(t *testing.T) {
	pools, err := Open(t.Context(), filepath.Join(t.TempDir(), "main.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = pools.Close() }()
	if err := pools.InitializeMain(t.Context()); err != nil {
		t.Fatal(err)
	}
	tx, err := pools.Write().BeginTx(t.Context(), nil)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = tx.Rollback() }()
	ctx, cancel := context.WithTimeout(t.Context(), 200*time.Millisecond)
	defer cancel()
	if err := pools.InitializeMain(ctx); err != nil {
		t.Fatalf("reinitialization waited for writer instead of reusing migration checks: %v", err)
	}
	if len(pools.MigrationFindings().After.Integrity) != 1 {
		t.Fatal("reinitialization lost migration findings")
	}
}

func TestMigrationFindingsOwnTheirSlices(t *testing.T) {
	pools, err := Open(t.Context(), filepath.Join(t.TempDir(), "main.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = pools.Close() }()
	findings, err := pools.Migrate(t.Context(), mainMigrations()...)
	if err != nil {
		t.Fatal(err)
	}
	findings.After.Integrity[0] = "changed by caller"
	if got := pools.MigrationFindings().After.Integrity[0]; got != "ok" {
		t.Fatalf("caller mutated held findings: %s", got)
	}
}
