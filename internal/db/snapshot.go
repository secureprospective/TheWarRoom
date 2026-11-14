package db

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"time"

	sqlite "modernc.org/sqlite"
)

// SnapshotReport describes one consistent copy and its compacted staging file.
type SnapshotReport struct {
	CopyPages  int64 // pages in the backup-API copy; equals the source at its read snapshot
	StagePages int64 // pages after VACUUM INTO; never more than CopyPages
	PageSize   int64
}

// onlineBackup is the method modernc.org/sqlite's driver conn exposes (conn.go NewBackup,
// backup.go Backup.Step/Finish): sqlite3_backup_init/step/finish over the raw connection.
type onlineBackup interface {
	NewBackup(dstURI string) (*sqlite.Backup, error)
}

// Snapshot copies the live file with SQLite's online backup API from a read-pool connection, then
// compacts that copy with VACUUM INTO stage, then verifies both. The copy runs inside one pinned read
// transaction, so it is the source as of one commit; under WAL a reader never blocks the writer, so
// the app's single writer stays free throughout (checkpoints cannot pass the reader, so the WAL grows
// for the copy's duration). Both files reach the disk through a paced backup (stepBackup), and the
// compaction runs in memory first, so the app writer's fsync never queues behind a burst of
// snapshot writes. The in-memory stage costs about the compacted database's size in RAM while it
// lives. On any failure, cancellation included, both outputs are removed.
func (p *Pools) Snapshot(ctx context.Context, copyPath, stagePath string) (SnapshotReport, error) {
	pin, err := p.pinSnapshot(ctx)
	if err != nil {
		return SnapshotReport{}, err
	}
	return pin.copyTo(ctx, copyPath, stagePath)
}

// snapshotPin is one read transaction held open on a read-pool connection: the point a snapshot
// copies. Pinning is quick; callers that need several files or settings to match pin while holding
// the writers, release the writers, and only then copy.
type snapshotPin struct {
	conn *sql.Conn
}

func (p *Pools) pinSnapshot(ctx context.Context) (*snapshotPin, error) {
	conn, err := p.read.Conn(ctx)
	if err != nil {
		return nil, fmt.Errorf("db: reserve snapshot reader: %w", err)
	}
	// Pin one read snapshot for every step. sqlite3_backup_step leaves an open source transaction
	// alone (backup.c), so all chunks read the same commit and a write from another connection never
	// restarts the copy; without the pin each step would start a fresh read and restart on any write.
	if _, err := conn.ExecContext(ctx, "BEGIN"); err != nil {
		return nil, errors.Join(fmt.Errorf("db: begin snapshot read: %w", err), conn.Close())
	}
	var schemaRows int64
	if err := conn.QueryRowContext(ctx, "SELECT count(*) FROM sqlite_schema").Scan(&schemaRows); err != nil {
		pin := &snapshotPin{conn: conn}
		pin.release(ctx)
		return nil, fmt.Errorf("db: pin snapshot read: %w", err)
	}
	return &snapshotPin{conn: conn}, nil
}

func (s *snapshotPin) release(ctx context.Context) {
	_, _ = s.conn.ExecContext(context.WithoutCancel(ctx), "ROLLBACK")
	_ = s.conn.Close()
}

// copyTo writes the pinned point to copyPath and its compacted, verified form to stagePath, then
// releases the pin. Neither target may exist.
func (s *snapshotPin) copyTo(ctx context.Context, copyPath, stagePath string) (SnapshotReport, error) {
	defer s.release(ctx)
	for _, path := range []string{copyPath, stagePath} {
		if _, err := os.Lstat(path); !errors.Is(err, os.ErrNotExist) {
			return SnapshotReport{}, fmt.Errorf("db: snapshot target %q already exists", path)
		}
	}
	report, err := s.snapshot(ctx, copyPath, stagePath)
	if err != nil {
		return SnapshotReport{}, errors.Join(err, removeSnapshot(copyPath), removeSnapshot(stagePath))
	}
	return report, nil
}

func (s *snapshotPin) snapshot(ctx context.Context, copyPath, stagePath string) (SnapshotReport, error) {
	if err := createOwnerOnly(copyPath); err != nil {
		return SnapshotReport{}, err
	}
	sourcePages, err := s.backupTo(ctx, copyPath)
	if err != nil {
		return SnapshotReport{}, err
	}
	copied, err := sql.Open("sqlite", fileURI(copyPath)+"?_pragma=busy_timeout(5000)")
	if err != nil {
		return SnapshotReport{}, fmt.Errorf("db: open snapshot copy: %w", err)
	}
	defer func() { _ = copied.Close() }()
	var report SnapshotReport
	if err := verifySnapshotFile(ctx, copied, &report.CopyPages, &report.PageSize); err != nil {
		return SnapshotReport{}, fmt.Errorf("db: snapshot copy: %w", err)
	}
	if report.CopyPages != sourcePages {
		return SnapshotReport{}, fmt.Errorf("db: snapshot copy has %d pages, source had %d", report.CopyPages, sourcePages)
	}
	if err := createOwnerOnly(stagePath); err != nil {
		return SnapshotReport{}, err
	}
	if err := vacuumInto(ctx, copied, stagePath); err != nil {
		return SnapshotReport{}, err
	}
	staged, err := sql.Open("sqlite", fileURI(stagePath)+"?mode=ro")
	if err != nil {
		return SnapshotReport{}, fmt.Errorf("db: open staged snapshot: %w", err)
	}
	defer func() { _ = staged.Close() }()
	var stagePageSize int64
	if err := verifySnapshotFile(ctx, staged, &report.StagePages, &stagePageSize); err != nil {
		return SnapshotReport{}, fmt.Errorf("db: staged snapshot: %w", err)
	}
	if report.StagePages > report.CopyPages || stagePageSize != report.PageSize {
		return SnapshotReport{}, fmt.Errorf("db: staged snapshot has %d pages of %d B, copy %d of %d B",
			report.StagePages, stagePageSize, report.CopyPages, report.PageSize)
	}
	if err := syncFile(copyPath); err != nil {
		return SnapshotReport{}, err
	}
	return report, syncFile(stagePath)
}

// snapshotChunkPages is how many pages one backup step copies before the copy is flushed to disk.
// Flushing each chunk keeps the snapshot's dirty page cache small, so the app writer's fsync never
// queues behind more than one small chunk (2a part 2: a 470 MB dirty backlog was the 1.5-10 s stall).
const snapshotChunkPages = 64

// snapshotChunkRest is the least time one chunk may take, write-out included: 256 KiB per 10 ms caps
// the copy at 25 MiB/s (the lab disk managed ~14 MiB/s), leaving the rest of the disk to the app writer.
const snapshotChunkRest = 10 * time.Millisecond

func (s *snapshotPin) backupTo(ctx context.Context, copyPath string) (int64, error) {
	dest, err := os.OpenFile(copyPath, os.O_WRONLY, 0)
	if err != nil {
		return 0, fmt.Errorf("db: open snapshot copy for flush: %w", err)
	}
	defer func() { _ = dest.Close() }()
	var pages int64
	err = s.conn.Raw(func(driverConn any) error {
		source, ok := driverConn.(onlineBackup)
		if !ok {
			return fmt.Errorf("db: driver connection %T has no online backup", driverConn)
		}
		backup, err := source.NewBackup(fileURI(copyPath))
		if err != nil {
			return fmt.Errorf("db: start online backup: %w", err)
		}
		stepErr := stepBackup(ctx, backup, dest)
		pages = int64(backup.PageCount())
		if err := errors.Join(stepErr, backup.Finish()); err != nil {
			return fmt.Errorf("db: online backup: %w", err)
		}
		return nil
	})
	if err != nil {
		return 0, fmt.Errorf("db: snapshot reader: %w", err)
	}
	return pages, nil
}

// stepBackup copies snapshotChunkPages at a time and fsyncs each chunk to the device (portable; the
// 11-08 lab measured sync_file_range no better than fsync per chunk, candidates 1 vs 2), then
// rests as long as that chunk took, so the copy uses at most half the disk's time and an app write
// queues behind at most one chunk. It stops when ctx is done.
func stepBackup(ctx context.Context, backup *sqlite.Backup, dest *os.File) error {
	for {
		begin := time.Now()
		more, err := backup.Step(snapshotChunkPages)
		if err != nil || !more {
			return err //nolint:wrapcheck // wrapped by the caller
		}
		if err := dest.Sync(); err != nil {
			return fmt.Errorf("write out %s: %w", dest.Name(), err)
		}
		select {
		case <-ctx.Done():
			return fmt.Errorf("snapshot cancelled: %w", context.Cause(ctx))
		case <-time.After(time.Until(begin.Add(snapshotChunkRest))):
		}
	}
}

// vacuumInto compacts the copy with VACUUM INTO an in-memory database, then writes that to stage
// with the same paced backup as the copy, so compaction never bursts writes at the disk.
func vacuumInto(ctx context.Context, copied *sql.DB, stagePath string) error {
	memURI := fmt.Sprintf("file:/twr-snapshot-stage-%d?vfs=memdb", time.Now().UnixNano())
	mem, err := sql.Open("sqlite", memURI)
	if err != nil {
		return fmt.Errorf("db: open staging memory: %w", err)
	}
	defer func() { _ = mem.Close() }()
	hold, err := mem.Conn(ctx) // the memdb lives while one connection holds it
	if err != nil {
		return fmt.Errorf("db: hold staging memory: %w", err)
	}
	defer func() { _ = hold.Close() }()
	if _, err := copied.ExecContext(ctx, "VACUUM INTO ?", memURI); err != nil {
		return fmt.Errorf("db: stage snapshot: %w", err)
	}
	stage, err := os.OpenFile(stagePath, os.O_WRONLY, 0)
	if err != nil {
		return fmt.Errorf("db: open staging for flush: %w", err)
	}
	defer func() { _ = stage.Close() }()
	err = hold.Raw(func(driverConn any) error {
		source, ok := driverConn.(onlineBackup)
		if !ok {
			return fmt.Errorf("db: driver connection %T has no online backup", driverConn)
		}
		backup, err := source.NewBackup(fileURI(stagePath))
		if err != nil {
			return fmt.Errorf("db: start staging write: %w", err)
		}
		return errors.Join(stepBackup(ctx, backup, stage), backup.Finish())
	})
	if err != nil {
		return fmt.Errorf("db: write staging: %w", err)
	}
	return nil
}

func verifySnapshotFile(ctx context.Context, conn *sql.DB, pages, pageSize *int64) error {
	var result string
	if err := conn.QueryRowContext(ctx, "PRAGMA integrity_check(1)").Scan(&result); err != nil {
		return fmt.Errorf("integrity check: %w", err)
	}
	if result != "ok" {
		return fmt.Errorf("integrity check failed: %s", result)
	}
	if err := conn.QueryRowContext(ctx, "PRAGMA page_count").Scan(pages); err != nil {
		return fmt.Errorf("page count: %w", err)
	}
	if err := conn.QueryRowContext(ctx, "PRAGMA page_size").Scan(pageSize); err != nil {
		return fmt.Errorf("page size: %w", err)
	}
	if *pages <= 0 {
		return fmt.Errorf("empty database")
	}
	return nil
}

// removeSnapshot removes a failed output and the sidecars its writing or verification may have left.
func removeSnapshot(path string) error {
	var errs []error
	for _, name := range []string{path, path + "-wal", path + "-shm", path + "-journal"} {
		if err := os.Remove(name); err != nil && !errors.Is(err, os.ErrNotExist) {
			errs = append(errs, fmt.Errorf("db: remove failed snapshot %q: %w", name, err))
		}
	}
	return errors.Join(errs...)
}
