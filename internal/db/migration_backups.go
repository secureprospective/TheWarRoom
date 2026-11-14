package db

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"modernc.org/sqlite"
	sqlite3 "modernc.org/sqlite/lib"
)

const migrationBackupLimit = 5

func (p *Pools) backupMigration(ctx context.Context, version int) (string, error) {
	backups, err := p.migrationBackupCandidates(ctx)
	if err != nil {
		return "", err
	}
	dest, err := p.captureMigrationBackup(ctx, version)
	if err != nil {
		return dest, err
	}
	retained, err := retainMatchingMigrationBackup(ctx, dest, backups)
	if err != nil {
		return dest, err
	}
	if err := p.pruneMigrationBackups(retained); err != nil {
		return retained, err
	}
	return retained, syncDir(filepath.Dir(p.path))
}

func (p *Pools) captureMigrationBackup(ctx context.Context, version int) (string, error) {
	f, err := os.CreateTemp(filepath.Dir(p.path), fmt.Sprintf("%s.pre-v%d-*.bak", filepath.Base(p.path), version))
	if err != nil {
		return "", fmt.Errorf("db: pre-migration backup: %w", err)
	}
	dest := f.Name()
	if err := f.Close(); err != nil {
		return "", fmt.Errorf("db: close pre-migration backup: %w", errors.Join(err, os.Remove(dest)))
	}
	// The exclusively created empty file is 0600; VACUUM preserves its mode.
	if _, err := p.write.ExecContext(ctx, "VACUUM INTO ?", dest); err != nil {
		// Cancellation can race a successful write; never assume the file is incomplete.
		valid, checkErr := verifyMigrationBackup(ctx, dest)
		if checkErr == nil && !valid {
			checkErr = removeIncompleteMigrationBackup(dest)
		}
		return "", fmt.Errorf("db: pre-migration backup: %w", errors.Join(err, checkErr))
	}
	if err := syncFile(dest); err != nil {
		return dest, err
	}
	valid, err := verifyMigrationBackup(ctx, dest)
	if err != nil {
		return dest, err
	}
	if !valid {
		return dest, fmt.Errorf("db: incomplete pre-migration backup %s", filepath.Base(dest))
	}
	return dest, syncDir(filepath.Dir(p.path))
}

// Immutable probes must not recover a killed VACUUM's journal and mistake it for a backup.
// CHECK/FK violations in the source are migration findings, not incomplete copies.
func verifyMigrationBackup(ctx context.Context, path string) (bool, error) {
	info, err := os.Lstat(path)
	if err != nil {
		return false, fmt.Errorf("db: inspect pre-migration backup: %w", err)
	}
	if info.Size() == 0 {
		return false, nil // SQLite otherwise accepts an empty file as an empty database.
	}
	probe, err := sql.Open("sqlite", fileURI(path)+"?mode=ro&immutable=1&_pragma=ignore_check_constraints(ON)")
	if err != nil {
		return false, fmt.Errorf("db: open pre-migration probe: %w", err)
	}
	defer func() { _ = probe.Close() }()
	var finding string
	err = probe.QueryRowContext(ctx, "PRAGMA integrity_check").Scan(&finding)
	var sqliteErr *sqlite.Error
	if errors.As(err, &sqliteErr) && (sqliteErr.Code()&0xff == sqlite3.SQLITE_CORRUPT || sqliteErr.Code()&0xff == sqlite3.SQLITE_NOTADB) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("db: verify pre-migration backup: %w", err)
	}
	return finding == "ok", nil
}

func (p *Pools) migrationBackupCandidates(ctx context.Context) ([]string, error) {
	dir, base := filepath.Dir(p.path), filepath.Base(p.path)
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("db: list pre-migration backups: %w", err)
	}
	var complete []string
	for _, entry := range entries {
		if !entry.Type().IsRegular() || !isMigrationBackup(entry.Name(), base) {
			continue
		}
		path := filepath.Join(dir, entry.Name())
		valid, err := verifyMigrationBackup(ctx, path)
		if err != nil {
			return nil, err
		}
		if valid {
			complete = append(complete, path)
		} else if strings.Contains(strings.TrimPrefix(entry.Name(), base+".pre-v"), "-") {
			// Only the random-suffix names were created by this VACUUM helper.
			if err := removeIncompleteMigrationBackup(path); err != nil {
				return nil, err
			}
		}
	}
	return complete, nil
}

func removeIncompleteMigrationBackup(path string) error {
	for _, name := range []string{path + "-journal", path} {
		info, err := os.Lstat(name)
		if errors.Is(err, os.ErrNotExist) {
			continue
		}
		if err != nil {
			return fmt.Errorf("db: inspect incomplete backup: %w", err)
		}
		if !info.Mode().IsRegular() {
			return fmt.Errorf("db: incomplete backup is not a regular file: %s", filepath.Base(name))
		}
		if err := os.Remove(name); err != nil {
			return fmt.Errorf("db: remove incomplete backup: %w", err)
		}
	}
	return nil
}

// VACUUM produces a compact, self-contained image including uncheckpointed WAL writes.
// Compare that verified image, not the live main file or just its schema version.
func retainMatchingMigrationBackup(ctx context.Context, current string, backups []string) (string, error) {
	want, err := backupEntry(ctx, current, "")
	if err != nil {
		return current, err
	}
	retained := current
	for _, path := range backups {
		got, err := backupEntry(ctx, path, "")
		if err != nil {
			return retained, err
		}
		if got.Size != want.Size || got.SHA256 != want.SHA256 {
			continue
		}
		if err := syncFile(path); err != nil {
			return retained, err
		}
		if err := os.Remove(retained); err != nil {
			return retained, fmt.Errorf("db: remove redundant pre-migration backup: %w", err)
		}
		retained = path
	}
	return retained, nil
}

// Include fixed-name backups from older builds, but never unrelated files or symlinks.
func isMigrationBackup(name, base string) bool {
	rest, ok := strings.CutPrefix(name, base+".pre-v")
	if !ok || !strings.HasSuffix(rest, ".bak") {
		return false
	}
	parts := strings.Split(strings.TrimSuffix(rest, ".bak"), "-")
	if len(parts) > 2 {
		return false
	}
	for _, part := range parts {
		if _, err := strconv.ParseUint(part, 10, 64); err != nil {
			return false
		}
	}
	return true
}

func (p *Pools) pruneMigrationBackups(current string) error {
	dir := filepath.Dir(p.path)
	entries, err := os.ReadDir(dir)
	if err != nil {
		return fmt.Errorf("db: list migration backups: %w", err)
	}
	var backups []os.FileInfo
	for _, entry := range entries {
		if !entry.Type().IsRegular() || !isMigrationBackup(entry.Name(), filepath.Base(p.path)) || filepath.Join(dir, entry.Name()) == current {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			return fmt.Errorf("db: inspect migration backup: %w", err)
		}
		backups = append(backups, info)
	}
	sort.Slice(backups, func(i, j int) bool {
		if backups[i].ModTime().Equal(backups[j].ModTime()) {
			return backups[i].Name() > backups[j].Name()
		}
		return backups[i].ModTime().After(backups[j].ModTime())
	})
	for i := migrationBackupLimit - 1; i < len(backups); i++ {
		if err := os.Remove(filepath.Join(dir, backups[i].Name())); err != nil {
			return fmt.Errorf("db: prune migration backup: %w", err)
		}
	}
	return nil
}

// BackupBeforeChange takes the same pre-migration backup Migrate takes, for a one-time change to the
// file that is not a schema step (moving sign-in tables to their own file).
func (p *Pools) BackupBeforeChange(ctx context.Context) (string, error) {
	p.migrationMu.Lock()
	defer p.migrationMu.Unlock()
	var version int
	if err := p.read.QueryRowContext(ctx, "PRAGMA user_version").Scan(&version); err != nil {
		return "", fmt.Errorf("db: backup version: %w", err)
	}
	return p.backupMigration(ctx, version)
}
