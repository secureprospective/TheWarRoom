package db

import (
	"bytes"
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"strings"
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

	mode, err := journalMode(context.Background(), pools.Read())
	if err != nil {
		t.Fatalf("journalMode: %v", err)
	}
	if mode != "wal" {
		t.Errorf("journal mode seen by the read pool = %q, want %q", mode, "wal")
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

func TestEveryPooledConnectionEnforcesPragmas(t *testing.T) {
	t.Parallel()
	pools := tempPools(t)
	for _, pool := range []struct {
		name string
		db   *sql.DB
	}{{"read", pools.Read()}, {"write", pools.Write()}} {
		t.Run(pool.name, func(t *testing.T) {
			// Force replacement of the connection Open already checked.
			pool.db.SetMaxIdleConns(0)
			pool.db.SetMaxIdleConns(10)
			count := 1
			if pool.name == "read" {
				count = 3
			}
			for range count {
				conn, err := pool.db.Conn(t.Context())
				if err != nil {
					t.Fatal(err)
				}
				t.Cleanup(func() { _ = conn.Close() })
				for _, pragma := range []struct {
					query string
					want  int
				}{{"PRAGMA foreign_keys", 1}, {"PRAGMA recursive_triggers", 1}, {"PRAGMA synchronous", 2}} {
					var got int
					if err := conn.QueryRowContext(t.Context(), pragma.query).Scan(&got); err != nil {
						t.Fatal(err)
					}
					if got != pragma.want {
						t.Errorf("%s = %d, want %d", pragma.query, got, pragma.want)
					}
				}
			}
		})
	}
}

func TestOpenRefusesNewerVersionWithoutWriting(t *testing.T) {
	t.Parallel()
	for _, name := range []string{"thewarroom.db", "history.db", "whatif.db"} {
		t.Run(name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), name)
			fixture, err := sql.Open("sqlite", path)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := fixture.ExecContext(t.Context(), `PRAGMA user_version=99; CREATE TABLE future_only (id TEXT); INSERT INTO future_only VALUES ('retained')`); err != nil {
				t.Fatal(err)
			}
			if err := fixture.Close(); err != nil {
				t.Fatal(err)
			}
			before, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			pools, err := Open(t.Context(), path)
			if pools != nil {
				_ = pools.Close()
			}
			if err == nil || !strings.Contains(err.Error(), "NEWER version") {
				t.Fatalf("Open downgrade refusal = %v", err)
			}
			after, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(before, after) {
				t.Fatal("newer database changed on refusal")
			}
			for _, suffix := range []string{"-wal", "-shm"} {
				if _, err := os.Stat(path + suffix); !os.IsNotExist(err) {
					t.Fatalf("unexpected sidecar %s: %v", suffix, err)
				}
			}
		})
	}
}
