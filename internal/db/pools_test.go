package db

import (
	"context"
	"path/filepath"
	"sync"
	"testing"
)

// tempPools opens a Pools against a fresh database file in a temp dir and
// registers cleanup.
func tempPools(t *testing.T) *Pools {
	t.Helper()
	path := filepath.Join(t.TempDir(), "test.db")
	pools, err := Open(context.Background(), path)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { _ = pools.Close() })
	return pools
}

func TestOpenEnablesWAL(t *testing.T) {
	t.Parallel()
	pools := tempPools(t)

	mode, err := pools.JournalMode(context.Background())
	if err != nil {
		t.Fatalf("JournalMode: %v", err)
	}
	if mode != "wal" {
		t.Errorf("journal mode = %q, want %q", mode, "wal")
	}
	if err := pools.Health(context.Background()); err != nil {
		t.Errorf("Health: %v", err)
	}
}

// TestConcurrentReadersWithWriter exercises the split-pool design under load:
// many concurrent readers against the read pool while the single writer runs
// transactions. With WAL + a one-connection write pool this must complete with
// zero SQLITE_BUSY errors. This is the concurrency check the Go overlay README
// insists on before the split-pool pattern is blessed (Section 3.4 caveat).
func TestConcurrentReadersWithWriter(t *testing.T) {
	t.Parallel()
	pools := tempPools(t)
	ctx := context.Background()

	if _, err := pools.Write().ExecContext(ctx,
		`CREATE TABLE counter (id INTEGER PRIMARY KEY, n INTEGER NOT NULL)`); err != nil {
		t.Fatalf("create table: %v", err)
	}
	if _, err := pools.Write().ExecContext(ctx,
		`INSERT INTO counter (id, n) VALUES (1, 0)`); err != nil {
		t.Fatalf("seed row: %v", err)
	}

	const writes = 50
	const readers = 16
	var wg sync.WaitGroup

	// Single writer: serialized through the one-connection write pool.
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < writes; i++ {
			if _, err := pools.Write().ExecContext(ctx,
				`UPDATE counter SET n = n + 1 WHERE id = 1`); err != nil {
				t.Errorf("write %d: %v", i, err)
				return
			}
		}
	}()

	// Many concurrent readers against the read pool.
	for r := 0; r < readers; r++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < writes; i++ {
				var n int
				if err := pools.Read().QueryRowContext(ctx,
					`SELECT n FROM counter WHERE id = 1`).Scan(&n); err != nil {
					t.Errorf("read: %v", err)
					return
				}
			}
		}()
	}

	wg.Wait()

	var final int
	if err := pools.Read().QueryRowContext(ctx,
		`SELECT n FROM counter WHERE id = 1`).Scan(&final); err != nil {
		t.Fatalf("final read: %v", err)
	}
	if final != writes {
		t.Errorf("counter = %d, want %d (writes lost or not serialized)", final, writes)
	}
}

func TestCloseIsSafe(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "close.db")
	pools, err := Open(context.Background(), path)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	if err := pools.Close(); err != nil {
		t.Errorf("Close: %v", err)
	}
}
