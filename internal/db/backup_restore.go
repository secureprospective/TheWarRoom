package db

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// RawRestoreJournal records an offline restore in flight. Fresh marks a restore into a profile
// that held no databases: it has no safety set, and rolling it back removes what it installed.
type RawRestoreJournal struct {
	Set    string `json:"set"`
	Stage  string `json:"stage"`
	Safety string `json:"safety"`
	Step   string `json:"step"`
	Fresh  bool   `json:"fresh,omitempty"`
}

func (j RawRestoreJournal) Validate() error {
	names := []string{j.Set, j.Stage}
	if !j.Fresh {
		names = append(names, j.Safety)
	}
	for _, name := range names {
		if name == "" || strings.ContainsAny(name, `/\`) || strings.Contains(name, "..") {
			return fmt.Errorf("db: invalid restore journal path")
		}
	}
	if !strings.HasPrefix(j.Stage, ".restore-") || j.Fresh != (j.Safety == "") ||
		(!j.Fresh && !strings.HasPrefix(j.Safety, "pre-restore-")) {
		return fmt.Errorf("db: invalid restore journal layout")
	}
	switch j.Step {
	case "prepared", "moving", "installing", "complete":
		return nil
	default:
		return fmt.Errorf("db: invalid restore journal step")
	}
}

// Restore is offline: the caller holds the data-directory instance lock, and no
// application pools may be open. Recover must run on every startup, before Open. It returns the
// pre-restore safety set, or "" for a profile that held no databases yet (a new computer):
// there was nothing to keep a safety copy of. A profile holding only some databases is refused.
func (b *Backups) Restore(ctx context.Context, set string, now time.Time) (string, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if err := b.recover(ctx); err != nil {
		return "", err
	}
	set, err := filepath.Abs(set)
	if err != nil {
		return "", fmt.Errorf("db: resolve restore set: %w", err)
	}
	root, err := filepath.Abs(filepath.Join(b.dir, "backups"))
	if err != nil {
		return "", fmt.Errorf("db: resolve backup directory: %w", err)
	}
	if filepath.Dir(set) != root {
		return "", fmt.Errorf("db: restore set must be in this data directory's backups")
	}
	if _, err := b.verify(ctx, set); err != nil {
		return "", err
	}
	fresh, err := b.emptyProfile()
	if err != nil {
		return "", err
	}
	stage, err := b.stage(ctx, set)
	if err != nil {
		return "", err
	}
	// Once journaled, recovery owns both directories, even on an injected crash.
	journaled := false
	defer func() {
		if !journaled {
			_ = os.RemoveAll(stage)
		}
	}()
	j := RawRestoreJournal{Set: filepath.Base(set), Stage: filepath.Base(stage), Step: "prepared", Fresh: fresh}
	if !fresh {
		j.Safety = "pre-restore-" + now.UTC().Format("20060102T150405.000000000Z")
		if err := b.prepareSafety(ctx, filepath.Join(root, j.Safety), now); err != nil {
			return "", err
		}
	}
	if err := durableJSON(filepath.Join(b.dir, "restore.journal"), j); err != nil {
		return "", err
	}
	journaled = true
	if err := b.checkpoint("prepared"); err != nil {
		return "", err
	}
	if err := b.swap(j); err != nil {
		return "", err
	}
	if fresh {
		return "", nil
	}
	return filepath.Join(root, j.Safety), nil
}

// stage copies a verified set beside the live files, so installing it is a rename.
func (b *Backups) stage(ctx context.Context, set string) (string, error) {
	stage, err := os.MkdirTemp(b.dir, ".restore-")
	if err != nil {
		return "", fmt.Errorf("db: stage restore: %w", err)
	}
	for _, file := range append(b.fileNames(), "manifest.json") {
		if err := copyBackupFile(filepath.Join(set, file), filepath.Join(stage, file)); err != nil {
			_ = os.RemoveAll(stage)
			return "", err
		}
	}
	if _, err := b.verify(ctx, stage); err != nil {
		_ = os.RemoveAll(stage)
		return "", err
	}
	if err := syncDir(stage); err != nil {
		_ = os.RemoveAll(stage)
		return "", err
	}
	return stage, nil
}

// emptyProfile reports whether no database, nor any leftover WAL or shared-memory file, is
// present. All three databases present is a normal restore; anything between is refused, because
// a restored database must never meet another file's WAL.
func (b *Backups) emptyProfile() (bool, error) {
	databases, others := 0, 0
	for _, name := range b.fileNames() {
		for _, suffix := range []string{"", "-wal", "-shm"} {
			if _, err := os.Lstat(filepath.Join(b.dir, name+suffix)); errors.Is(err, os.ErrNotExist) {
				continue
			} else if err != nil {
				return false, fmt.Errorf("db: inspect current database: %w", err)
			}
			if suffix == "" {
				databases++
			} else {
				others++
			}
		}
	}
	switch {
	case databases == 0 && others == 0:
		return true, nil
	case databases == len(b.files):
		return false, nil
	default:
		return false, fmt.Errorf("db: this profile holds only some of its databases; restore needs all of them or none")
	}
}

func (b *Backups) fileNames() []string {
	return []string{b.files[0].Name, b.files[1].Name, b.files[2].Name}
}

func copyBackupFile(source, dest string) error {
	in, err := os.Open(source)
	if err != nil {
		return fmt.Errorf("db: open restore source: %w", err)
	}
	defer func() { _ = in.Close() }()
	out, err := os.OpenFile(dest, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		return fmt.Errorf("db: create staged file: %w", err)
	}
	defer func() { _ = out.Close() }()
	if _, err := io.Copy(out, in); err != nil {
		return fmt.Errorf("db: copy staged file: %w", err)
	}
	if err := out.Sync(); err != nil {
		return fmt.Errorf("db: sync staged file: %w", err)
	}
	return nil
}

func (b *Backups) prepareSafety(ctx context.Context, dir string, now time.Time) error {
	m := RawBackupManifest{ManifestVersion: 1, AppBuild: b.build, CapturedAt: now.UTC(), Kind: "pre-restore"}
	for _, file := range b.files {
		path := filepath.Join(b.dir, file.Name)
		if err := checkpointOffline(ctx, path); err != nil {
			return err
		}
		entry, err := backupEntry(ctx, path, file.Name)
		if err != nil {
			return err
		}
		if err := syncFile(path); err != nil {
			return err
		}
		m.Files = append(m.Files, entry)
	}
	if err := m.Validate(b.files); err != nil {
		return err
	}
	if err := os.Mkdir(dir, 0700); err != nil {
		return fmt.Errorf("db: create pre-restore set: %w", err)
	}
	if err := durableJSON(filepath.Join(dir, "manifest.json"), m); err != nil {
		return err
	}
	return syncDir(filepath.Dir(dir))
}

func checkpointOffline(ctx context.Context, path string) error {
	if _, err := os.Stat(path); err != nil {
		return fmt.Errorf("db: current restore member: %w", err)
	}
	p, err := sql.Open("sqlite", "file:"+path+"?mode=rw")
	if err != nil {
		return fmt.Errorf("db: open offline checkpoint: %w", err)
	}
	var busy, log, done int
	queryErr := p.QueryRowContext(ctx, "PRAGMA wal_checkpoint(TRUNCATE)").Scan(&busy, &log, &done)
	closeErr := p.Close()
	if queryErr != nil || closeErr != nil {
		return fmt.Errorf("db: offline checkpoint: %w", errors.Join(queryErr, closeErr))
	}
	if busy != 0 {
		return fmt.Errorf("db: offline checkpoint has an active writer")
	}
	return nil
}

func (b *Backups) checkpoint(step string) error {
	if b.crash != nil {
		return b.crash(step)
	}
	return nil
}

func (b *Backups) journalStep(j *RawRestoreJournal, step string) error {
	j.Step = step
	if err := durableJSON(filepath.Join(b.dir, "restore.journal"), *j); err != nil {
		return err
	}
	return b.checkpoint(step)
}

func (b *Backups) swap(j RawRestoreJournal) error {
	if !j.Fresh {
		if err := b.journalStep(&j, "moving"); err != nil {
			return err
		}
		for _, name := range b.fileNames() {
			if err := b.moveCurrent(j.Safety, name); err != nil {
				return err
			}
		}
	}
	if err := b.journalStep(&j, "installing"); err != nil {
		return err
	}
	for _, name := range b.fileNames() {
		if err := os.Rename(filepath.Join(b.dir, j.Stage, name), filepath.Join(b.dir, name)); err != nil {
			return fmt.Errorf("db: install restored database: %w", err)
		}
		if err := syncDir(b.dir); err != nil {
			return err
		}
		if err := syncDir(filepath.Join(b.dir, j.Stage)); err != nil {
			return err
		}
		if err := b.checkpoint("installed:" + name); err != nil {
			return err
		}
	}
	if err := b.journalStep(&j, "complete"); err != nil {
		return err
	}
	return b.finish(j)
}

func (b *Backups) finish(j RawRestoreJournal) error {
	if err := os.RemoveAll(filepath.Join(b.dir, j.Stage)); err != nil {
		return fmt.Errorf("db: clean restore staging: %w", err)
	}
	if err := os.Remove(filepath.Join(b.dir, "restore.journal")); err != nil {
		return fmt.Errorf("db: remove restore journal: %w", err)
	}
	return syncDir(b.dir)
}

func (b *Backups) Recover(ctx context.Context) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.recover(ctx)
}

func (b *Backups) recover(ctx context.Context) error {
	path := filepath.Join(b.dir, "restore.journal")
	if _, err := os.Lstat(path); errors.Is(err, os.ErrNotExist) {
		return nil
	} else if err != nil {
		return fmt.Errorf("db: inspect restore journal: %w", err)
	}
	var j RawRestoreJournal
	if err := readJSON(path, &j); err != nil {
		return err
	}
	if err := j.Validate(); err != nil {
		return err
	}
	if j.Step == "complete" {
		return b.finish(j)
	}
	if j.Fresh {
		return b.rollbackFresh(j)
	}
	safety := filepath.Join(b.dir, "backups", j.Safety)
	if err := b.validateRollback(ctx, j, safety); err != nil {
		return err
	}
	// Copy back instead of consuming the safety files: rollback itself may crash.
	for _, name := range b.fileNames() {
		for _, suffix := range []string{"", "-wal", "-shm"} {
			if err := rollbackMember(filepath.Join(safety, name+suffix), filepath.Join(b.dir, name+suffix)); err != nil {
				return err
			}
		}
	}
	if err := syncDir(b.dir); err != nil {
		return err
	}
	if err := b.finish(j); err != nil {
		return err
	}
	// A partial safety directory is not a set; it is ours, not foreign data.
	if err := os.RemoveAll(safety); err != nil {
		return fmt.Errorf("db: remove rolled-back safety set: %w", err)
	}
	return syncDir(filepath.Dir(safety))
}

// rollbackFresh undoes a restore into an empty profile: every database file present was
// installed by it, so removing them returns the profile to empty.
func (b *Backups) rollbackFresh(j RawRestoreJournal) error {
	for _, name := range b.fileNames() {
		for _, suffix := range []string{"", "-wal", "-shm"} {
			if err := removeIfPresent(filepath.Join(b.dir, name+suffix)); err != nil {
				return err
			}
		}
	}
	if err := syncDir(b.dir); err != nil {
		return err
	}
	return b.finish(j)
}

func removeIfPresent(path string) error {
	if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("db: remove restore temporary file: %w", err)
	}
	return nil
}

func (b *Backups) moveCurrent(safetyName, name string) error {
	safety := filepath.Join(b.dir, "backups", safetyName)
	for _, suffix := range []string{"", "-wal", "-shm"} {
		src := filepath.Join(b.dir, name+suffix)
		if suffix != "" {
			if _, err := os.Lstat(src); errors.Is(err, os.ErrNotExist) {
				continue
			}
		}
		if err := os.Rename(src, filepath.Join(safety, name+suffix)); err != nil {
			return fmt.Errorf("db: preserve current database: %w", err)
		}
		if err := syncDir(safety); err != nil {
			return err
		}
		if err := syncDir(b.dir); err != nil {
			return err
		}
		if err := b.checkpoint("moved:" + name + suffix); err != nil {
			return err
		}
	}
	return nil
}

func rollbackMember(src, dest string) error {
	if _, err := os.Lstat(src); errors.Is(err, os.ErrNotExist) {
		return nil
	} else if err != nil {
		return fmt.Errorf("db: inspect rollback member: %w", err)
	}
	tmp := dest + ".rollback"
	if err := removeIfPresent(tmp); err != nil {
		return err
	}
	if err := copyBackupFile(src, tmp); err != nil {
		return err
	}
	if err := os.Rename(tmp, dest); err != nil {
		return fmt.Errorf("db: roll back restore: %w", err)
	}
	return nil
}

func (b *Backups) validateRollback(ctx context.Context, j RawRestoreJournal, safety string) error {
	var m RawBackupManifest
	if err := readJSON(filepath.Join(safety, "manifest.json"), &m); err != nil {
		return err
	}
	if err := m.Validate(b.files); err != nil {
		return err
	}
	if m.Kind != "pre-restore" {
		return fmt.Errorf("db: rollback set has wrong kind")
	}
	for _, entry := range m.Files {
		path := filepath.Join(safety, entry.Name)
		if _, err := os.Lstat(path); errors.Is(err, os.ErrNotExist) {
			if j.Step == "installing" {
				return fmt.Errorf("db: missing rollback database %s", entry.Name)
			}
			if _, err := os.Lstat(filepath.Join(b.dir, entry.Name)); err != nil {
				return fmt.Errorf("db: missing current and rollback database: %w", err)
			}
			continue
		}
		got, err := backupEntry(ctx, path, entry.Name)
		if err != nil {
			return err
		}
		if got != entry {
			return fmt.Errorf("db: rollback database mismatch: %s", entry.Name)
		}
	}
	return nil
}
