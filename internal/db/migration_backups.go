package db

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

const migrationBackupLimit = 5

func (p *Pools) backupMigration(ctx context.Context, version int) (string, error) {
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
		return "", fmt.Errorf("db: pre-migration backup: %w", errors.Join(err, os.Remove(dest)))
	}
	if err := syncFile(dest); err != nil {
		return dest, err
	}
	if err := p.pruneMigrationBackups(dest); err != nil {
		return dest, err
	}
	return dest, syncDir(filepath.Dir(p.path))
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
