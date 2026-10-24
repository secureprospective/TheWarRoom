package db

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// Backups coordinates matched snapshots and offline recovery. Call Recover before opening
// application pools. Restore requires the same instance lock as startup and no open pools.
type Backups struct {
	mu         sync.Mutex
	dir, build string
	files      [3]BackupFile
	crash      func(string) error
}

func NewBackups(dir, build string, files [3]BackupFile) (*Backups, error) {
	for i, base := range []string{"thewarroom", "history", "whatif"} {
		if (files[i].Name != base+".db" && files[i].Name != base+"-dev.db") || files[i].MaxVersion < 0 {
			return nil, fmt.Errorf("db: invalid backup database configuration")
		}
	}
	if build == "" {
		return nil, fmt.Errorf("db: missing backup build")
	}
	return &Backups{dir: dir, build: build, files: files}, nil
}

func fileUserVersion(ctx context.Context, path string) (int, error) {
	return probeVersion(ctx, path, "?mode=ro")
}

func probeVersion(ctx context.Context, path, options string) (int, error) {
	p, err := sql.Open("sqlite", "file:"+path+options)
	if err != nil {
		return 0, fmt.Errorf("db: open backup probe: %w", err)
	}
	defer func() { _ = p.Close() }()
	var version int
	if err := p.QueryRowContext(ctx, "PRAGMA user_version").Scan(&version); err != nil {
		return 0, fmt.Errorf("db: backup version: %w", err)
	}
	return version, nil
}

// Capture reserves the single writer connection in main, history, what-if order;
// reservations are the write locks shared by every store, released in reverse order.
// No VACUUM starts until every writer is stopped. Callers must not hold a writer.
func (b *Backups) Capture(ctx context.Context, pools [3]*Pools, kind string, now time.Time) (string, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if kind != "daily" {
		return "", fmt.Errorf("db: online capture must be daily")
	}
	if due, err := b.dailyDue(ctx, now); err != nil || !due {
		return "", err
	}
	var conns [3]*sql.Conn
	defer func() {
		for i := 2; i >= 0; i-- {
			if conns[i] != nil {
				_ = conns[i].Close()
			}
		}
	}()
	for i, p := range pools {
		if p == nil || filepath.Clean(p.path) != filepath.Join(b.dir, b.files[i].Name) {
			return "", fmt.Errorf("db: backup pool does not match %s", b.files[i].Name)
		}
		conn, err := p.write.Conn(ctx)
		if err != nil {
			return "", fmt.Errorf("db: acquire backup writer: %w", err)
		}
		conns[i] = conn
	}
	return b.capture(ctx, kind, now, func(ctx context.Context, i int, dest string) error {
		if _, err := conns[i].ExecContext(ctx, "VACUUM INTO ?", dest); err != nil {
			return fmt.Errorf("db: capture %s: %w", b.files[i].Name, err)
		}
		return nil
	})
}

// BeforeMigrations runs before any application pool opens; a new empty home has no
// prior set. A partially present home is refused rather than silently losing a member.
func (b *Backups) BeforeMigrations(ctx context.Context, now time.Time) (string, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	present, pending := 0, false
	for _, file := range b.files {
		path := filepath.Join(b.dir, file.Name)
		if _, err := os.Stat(path); errors.Is(err, os.ErrNotExist) {
			continue
		} else if err != nil {
			return "", fmt.Errorf("db: inspect startup database: %w", err)
		}
		version, err := fileUserVersion(ctx, path)
		if err != nil {
			return "", err
		}
		if err := refuseNewer(version, file.MaxVersion); err != nil {
			return "", err
		}
		present++
		pending = pending || version < file.MaxVersion
	}
	if present == 0 || !pending {
		return "", nil
	}
	if present != 3 {
		return "", fmt.Errorf("db: pre-migration backup requires all three databases")
	}
	return b.capture(ctx, "pre-migration", now, func(ctx context.Context, i int, dest string) error {
		p, err := sql.Open("sqlite", "file:"+filepath.Join(b.dir, b.files[i].Name)+"?mode=ro")
		if err != nil {
			return fmt.Errorf("db: open startup snapshot: %w", err)
		}
		defer func() { _ = p.Close() }()
		if _, err := p.ExecContext(ctx, "VACUUM INTO ?", dest); err != nil {
			return fmt.Errorf("db: startup snapshot: %w", err)
		}
		return nil
	})
}

func (b *Backups) capture(ctx context.Context, kind string, now time.Time, vacuum func(context.Context, int, string) error) (string, error) {
	root := filepath.Join(b.dir, "backups")
	if err := os.MkdirAll(root, 0700); err != nil {
		return "", fmt.Errorf("db: create backups directory: %w", err)
	}
	stage, err := os.MkdirTemp(root, ".capture-")
	if err != nil {
		return "", fmt.Errorf("db: stage backup: %w", err)
	}
	defer func() { _ = os.RemoveAll(stage) }()
	m := RawBackupManifest{ManifestVersion: 1, AppBuild: b.build, CapturedAt: now.UTC(), Kind: kind}
	for i, file := range b.files {
		path := filepath.Join(stage, file.Name)
		if err := createOwnerOnly(path); err != nil {
			return "", err
		}
		if err := vacuum(ctx, i, path); err != nil {
			return "", err
		}
		entry, err := backupEntry(ctx, path, file.Name)
		if err != nil {
			return "", err
		}
		if err := syncFile(path); err != nil {
			return "", err
		}
		m.Files = append(m.Files, entry)
	}
	if err := m.Validate(b.files); err != nil {
		return "", err
	}
	if err := durableJSON(filepath.Join(stage, "manifest.json"), m); err != nil {
		return "", err
	}
	dest := filepath.Join(root, kind+"-"+now.UTC().Format("20060102T150405.000000000Z"))
	if _, err := os.Lstat(dest); !errors.Is(err, os.ErrNotExist) {
		return "", fmt.Errorf("db: backup destination exists or is inaccessible: %s", dest)
	}
	if err := os.Rename(stage, dest); err != nil {
		return "", fmt.Errorf("db: publish backup set: %w", err)
	}
	if err := syncDir(root); err != nil {
		return "", err
	}
	return dest, nil
}

func (b *Backups) dailyDue(ctx context.Context, now time.Time) (bool, error) {
	sets, _, err := b.sets(ctx)
	if err != nil {
		return false, err
	}
	for _, set := range sets {
		if set.manifest.Kind == "daily" && set.manifest.CapturedAt.UTC().Format(time.DateOnly) == now.UTC().Format(time.DateOnly) {
			return false, nil
		}
	}
	return true, nil
}
