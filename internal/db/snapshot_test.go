package db

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// snapshotFixture holds accounts whose balances always sum to snapshotTotal; every writer
// transaction moves value between two of them, so a torn copy shows as a wrong sum.
const snapshotTotal = 4000 * 100

func snapshotFixture(t *testing.T, path string) *Pools {
	t.Helper()
	pools, err := Open(t.Context(), path)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = pools.Close() })
	if _, err := pools.Write().ExecContext(t.Context(), `
		CREATE TABLE account(id INTEGER PRIMARY KEY, balance INTEGER NOT NULL, pad BLOB NOT NULL);
		WITH RECURSIVE n(i) AS (SELECT 1 UNION ALL SELECT i+1 FROM n WHERE i < 4000)
		INSERT INTO account SELECT i, 100, randomblob(8192) FROM n;`); err != nil {
		t.Fatal(err)
	}
	return pools
}

// stallWriter commits small transfers through the app's writer until stop closes and
// returns each write's latency.
func stallWriter(ctx context.Context, pools *Pools, stop <-chan struct{}, commits *atomic.Int64) ([]time.Duration, error) {
	var latencies []time.Duration
	for i := 0; ; i++ {
		select {
		case <-stop:
			return latencies, nil
		default:
		}
		from, to := i%4000+1, (i*7+3)%4000+1
		start := time.Now()
		_, err := pools.ExecWrite(ctx, func(tx *sql.Tx) (sql.Result, error) {
			if _, err := tx.ExecContext(ctx, `UPDATE account SET balance = balance - 1 WHERE id = ?`, from); err != nil {
				return nil, err
			}
			return tx.ExecContext(ctx, `UPDATE account SET balance = balance + 1 WHERE id = ?`, to)
		})
		if err != nil {
			return latencies, err
		}
		latencies = append(latencies, time.Since(start))
		commits.Add(1)
		time.Sleep(time.Millisecond)
	}
}

func percentile(sorted []time.Duration, p float64) time.Duration {
	if len(sorted) == 0 {
		return 0
	}
	return sorted[int(float64(len(sorted)-1)*p)]
}

func TestSnapshot(t *testing.T) {
	tests := []struct {
		name     string
		prepare  func(t *testing.T, dir string) (*Pools, string, string)
		check    func(t *testing.T, pools *Pools, report SnapshotReport, err error, copyPath, stagePath string)
		deadline time.Duration
	}{
		{
			name: "copy is consistent while the writer commits and the writer is not held",
			prepare: func(t *testing.T, dir string) (*Pools, string, string) {
				return snapshotFixture(t, filepath.Join(dir, "live.db")), filepath.Join(dir, "copy.db"), filepath.Join(dir, "stage.db")
			},
		},
		{
			name: "free pages are dropped by staging",
			prepare: func(t *testing.T, dir string) (*Pools, string, string) {
				pools := snapshotFixture(t, filepath.Join(dir, "live.db"))
				if _, err := pools.Write().ExecContext(t.Context(), `CREATE TABLE junk AS SELECT randomblob(8192) FROM account; DROP TABLE junk`); err != nil {
					t.Fatal(err)
				}
				return pools, filepath.Join(dir, "copy.db"), filepath.Join(dir, "stage.db")
			},
			check: func(t *testing.T, _ *Pools, report SnapshotReport, err error, _, _ string) {
				if err != nil {
					t.Fatal(err)
				}
				if report.StagePages >= report.CopyPages {
					t.Fatalf("staging kept free pages: copy %d stage %d", report.CopyPages, report.StagePages)
				}
			},
		},
		{
			name: "corrupted source is refused and leaves no output",
			prepare: func(t *testing.T, dir string) (*Pools, string, string) {
				live := filepath.Join(dir, "live.db")
				pools := snapshotFixture(t, live)
				if _, err := pools.Write().ExecContext(t.Context(), `PRAGMA wal_checkpoint(TRUNCATE)`); err != nil {
					t.Fatal(err)
				}
				var pageSize int64
				if err := pools.Read().QueryRowContext(t.Context(), `PRAGMA page_size`).Scan(&pageSize); err != nil {
					t.Fatal(err)
				}
				f, err := os.OpenFile(live, os.O_WRONLY, 0)
				if err != nil {
					t.Fatal(err)
				}
				// Page 2 is the account table's root, an interior b-tree page.
				if _, err := f.WriteAt([]byte(strings.Repeat("\xff", 512)), pageSize+64); err != nil {
					t.Fatal(err)
				}
				if err := f.Close(); err != nil {
					t.Fatal(err)
				}
				return pools, filepath.Join(dir, "copy.db"), filepath.Join(dir, "stage.db")
			},
			check: func(t *testing.T, _ *Pools, _ SnapshotReport, err error, copyPath, stagePath string) {
				if err == nil {
					t.Fatal("corrupted source was accepted")
				}
				for _, path := range []string{copyPath, stagePath, copyPath + "-wal", copyPath + "-shm"} {
					if _, statErr := os.Stat(path); !errors.Is(statErr, os.ErrNotExist) {
						t.Fatalf("%s left behind after %v", path, err)
					}
				}
			},
		},
		{
			name: "snapshot completes while a write transaction holds the writer",
			prepare: func(t *testing.T, dir string) (*Pools, string, string) {
				pools := snapshotFixture(t, filepath.Join(dir, "live.db"))
				held, err := pools.Write().BeginTx(t.Context(), nil)
				if err != nil {
					t.Fatal(err)
				}
				t.Cleanup(func() { _ = held.Rollback() })
				if _, err := held.ExecContext(t.Context(), `UPDATE account SET balance = balance WHERE id = 1`); err != nil {
					t.Fatal(err)
				}
				return pools, filepath.Join(dir, "copy.db"), filepath.Join(dir, "stage.db")
			},
			deadline: 30 * time.Second,
			check: func(t *testing.T, _ *Pools, report SnapshotReport, err error, _, _ string) {
				if err != nil {
					t.Fatalf("snapshot waited on the writer: %v", err)
				}
				if report.CopyPages == 0 {
					t.Fatal("empty copy")
				}
			},
		},
		{
			name: "a cancelled context stops the paced copy and leaves no output",
			prepare: func(t *testing.T, dir string) (*Pools, string, string) {
				return snapshotFixture(t, filepath.Join(dir, "live.db")), filepath.Join(dir, "copy.db"), filepath.Join(dir, "stage.db")
			},
			// The fixture's ~8,500 pages take over a second at the paced rate; stop well before.
			deadline: 100 * time.Millisecond,
			check: func(t *testing.T, _ *Pools, _ SnapshotReport, err error, copyPath, stagePath string) {
				if !errors.Is(err, context.DeadlineExceeded) {
					t.Fatalf("snapshot ignored its context: %v", err)
				}
				for _, path := range []string{copyPath, stagePath, copyPath + "-wal", copyPath + "-shm", copyPath + "-journal", stagePath + "-journal"} {
					if _, statErr := os.Stat(path); !errors.Is(statErr, os.ErrNotExist) {
						t.Fatalf("%s left behind after cancel", path)
					}
				}
			},
		},
		{
			name: "existing target is refused and kept",
			prepare: func(t *testing.T, dir string) (*Pools, string, string) {
				pools := snapshotFixture(t, filepath.Join(dir, "live.db"))
				stage := filepath.Join(dir, "stage.db")
				if err := os.WriteFile(stage, []byte("keep"), 0o600); err != nil {
					t.Fatal(err)
				}
				return pools, filepath.Join(dir, "copy.db"), stage
			},
			check: func(t *testing.T, _ *Pools, _ SnapshotReport, err error, copyPath, stagePath string) {
				if err == nil {
					t.Fatal("existing target was overwritten")
				}
				if body, readErr := os.ReadFile(stagePath); readErr != nil || string(body) != "keep" {
					t.Fatalf("existing target changed: %q %v", body, readErr)
				}
				if _, statErr := os.Stat(copyPath); !errors.Is(statErr, os.ErrNotExist) {
					t.Fatal("copy created despite refusal")
				}
			},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			pools, copyPath, stagePath := tc.prepare(t, dir)
			if tc.check != nil {
				ctx := t.Context()
				if tc.deadline > 0 {
					var cancel context.CancelFunc
					ctx, cancel = context.WithTimeout(ctx, tc.deadline)
					defer cancel()
				}
				report, err := pools.Snapshot(ctx, copyPath, stagePath)
				tc.check(t, pools, report, err, copyPath, stagePath)
				return
			}
			checkConsistentUnderWrites(t, pools, func(ctx context.Context) ([]string, error) {
				_, err := pools.Snapshot(ctx, copyPath, stagePath)
				return []string{copyPath, stagePath}, err
			})
		})
	}
}

// TestCaptureHostedPinsConfigWithTheDatabase proves the hosted capture holds the writer only for the
// pin and the config copy: config runs with no write open, and writes commit during the copy.
func TestCaptureHostedPinsConfigWithTheDatabase(t *testing.T) {
	dir := t.TempDir()
	pools := snapshotFixture(t, filepath.Join(dir, "live.db"))
	dest := filepath.Join(dir, "server.db")
	var configCalls int
	checkConsistentUnderWrites(t, pools, func(ctx context.Context) ([]string, error) {
		return []string{dest}, pools.CaptureHosted(ctx, dest, func() error {
			configCalls++
			// The writer is reserved here: a write from another goroutine must wait for config.
			blocked, cancel := context.WithTimeout(ctx, 50*time.Millisecond)
			defer cancel()
			if _, err := pools.ExecWrite(blocked, func(tx *sql.Tx) (sql.Result, error) {
				return tx.ExecContext(blocked, `UPDATE account SET balance = balance WHERE id = 1`)
			}); err == nil {
				return errors.New("a write committed while config was being copied")
			}
			return nil
		})
	})
	if configCalls != 1 {
		t.Fatalf("config ran %d times", configCalls)
	}
	if _, err := os.Stat(dest + ".copy"); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("intermediate copy left behind")
	}
}

// writerStallBound is the stop-rule ceiling for one small write during a snapshot.
const writerStallBound = time.Second

func checkConsistentUnderWrites(t *testing.T, pools *Pools, capture func(context.Context) ([]string, error)) {
	stop := make(chan struct{})
	var latencies []time.Duration
	var writeErr error
	var commits atomic.Int64
	var wg sync.WaitGroup
	wg.Go(func() { latencies, writeErr = stallWriter(t.Context(), pools, stop, &commits) })
	time.Sleep(20 * time.Millisecond)
	startWrites := commits.Load()
	start := time.Now()
	paths, err := capture(t.Context())
	copyTime := time.Since(start)
	during := commits.Load()
	time.Sleep(20 * time.Millisecond)
	close(stop)
	wg.Wait()
	if err != nil || writeErr != nil {
		t.Fatal(err, writeErr)
	}
	sorted := append([]time.Duration(nil), latencies...)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i] < sorted[j] })
	maxStall := sorted[len(sorted)-1]
	t.Logf("snapshot %v; %d writes, p99 %v max %v", copyTime, len(sorted), percentile(sorted, 0.99), maxStall)
	if maxStall > writerStallBound || maxStall > copyTime/2+50*time.Millisecond {
		t.Fatalf("writer stalled %v during a %v snapshot", maxStall, copyTime)
	}
	if during-startWrites < 2 {
		t.Fatalf("only %d writes committed during the %v snapshot: the writer was held", during-startWrites, copyTime)
	}
	var committed int
	if err := pools.Read().QueryRowContext(t.Context(), `SELECT count(*) FROM account WHERE balance != 100`).Scan(&committed); err != nil || committed == 0 {
		t.Fatalf("writer made no visible change: %d %v", committed, err)
	}
	for _, path := range paths {
		snap, err := sql.Open("sqlite", fileURI(path)+"?mode=ro")
		if err != nil {
			t.Fatal(err)
		}
		var check string
		var sum, rows int64
		err = errors.Join(snap.QueryRowContext(t.Context(), `PRAGMA integrity_check`).Scan(&check),
			snap.QueryRowContext(t.Context(), `SELECT sum(balance), count(*) FROM account`).Scan(&sum, &rows))
		_ = snap.Close()
		if err != nil || check != "ok" || sum != snapshotTotal || rows != 4000 {
			t.Fatalf("%s torn: integrity %q sum %d rows %d err %v", filepath.Base(path), check, sum, rows, err)
		}
	}
}
