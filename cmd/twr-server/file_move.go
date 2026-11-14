package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/secureprospective/TheWarRoom/internal/db"
)

// fileMove is one part of a one-file server moving into its own file beside the league file: the
// sign-in file (slice 4) and the reference file (slice 5) move through the same steps. The commit
// that drops the moved tables from the league file is the only point of no return: before it the old
// layout is intact and a half-built file is discarded; after it the proven file is put in place.
type fileMove struct {
	// what names the moved part in messages ("sign-ins", "football reference data").
	what string
	// file is the new file's name in the state folder.
	file string
	// held is a table whose presence says the league file still holds the part; record is the league
	// file's table saying the part moved.
	held, record string
	// build writes and proves the new file at pending, copying from league when held, and returns the
	// commit that drops the part from the league file and writes the record in one transaction.
	build func(ctx context.Context, league *db.Pools, pending string, held bool) (func(context.Context) error, error)
	// after runs once the move committed and the file is in place (a checkpoint or compaction).
	after func(ctx context.Context, league *db.Pools) error
}

func (m fileMove) path(leaguePath string) string {
	return filepath.Join(filepath.Dir(leaguePath), m.file)
}

func (m fileMove) pending(leaguePath string) string { return m.path(leaguePath) + ".converting" }

// conversionStep lets a test stop the process between conversion steps, as a crash would, or change
// the file being built. pending is the file under construction.
var conversionStep = func(_, _ string) {} //nolint:gochecknoglobals // a test seam: only tests replace it

// moveLayout is what the state folder holds: the part in the league file, the league file's move
// record, the new file, and a new file still being built.
type moveLayout struct{ held, moved, final, pending bool }

func (m fileMove) layout(ctx context.Context, league *db.Pools, leaguePath string) (moveLayout, error) {
	var l moveLayout
	var err error
	if l.held, err = hasTable(ctx, league, m.held); err != nil {
		return l, err
	}
	if l.moved, err = hasTable(ctx, league, m.record); err != nil {
		return l, err
	}
	l.final, l.pending = fileExists(m.path(leaguePath)), fileExists(m.pending(leaguePath))
	return l, nil
}

// settle leaves the folder in the moved layout or refuses with a plain message.
func (m fileMove) settle(ctx context.Context, league *db.Pools, leaguePath string) error {
	l, err := m.layout(ctx, league, leaguePath)
	if err != nil {
		return err
	}
	switch {
	case l.moved && l.held:
		return fmt.Errorf("the league file records that %s moved out but still holds them; nothing was changed. Restore from backup (RESTORE.md)", m.what)
	case l.moved && l.final:
		return nil
	case l.moved && l.pending:
		// The move committed, then the process stopped before the proven file took its name.
		return m.finish(ctx, league, leaguePath)
	case l.moved:
		return fmt.Errorf("the file %s is missing; the league file says %s were moved there. Nothing was changed. Restore it from backup (RESTORE.md)", m.file, m.what)
	case l.final:
		return fmt.Errorf("found %s, but the league file never moved its %s there; nothing was changed. Check that -db names this server's league file", m.file, m.what)
	}
	if err := discardPending(m.pending(leaguePath)); err != nil {
		return err
	}
	return m.convert(ctx, league, leaguePath, l.held)
}

// convert builds the new file beside the league file. With held, it copies the part, proves the copy,
// then drops the part from the league file.
func (m fileMove) convert(ctx context.Context, league *db.Pools, leaguePath string, held bool) error {
	pending := m.pending(leaguePath)
	if held {
		if _, err := league.BackupBeforeChange(ctx); err != nil {
			return fmt.Errorf("back up the league file before moving %s: %w", m.what, err)
		}
	}
	conversionStep("backup", pending)
	commit, err := m.build(ctx, league, pending, held)
	if err != nil {
		return errors.Join(fmt.Errorf("could not move %s to their own file (%w); nothing was changed and the server did not start", m.what, err), discardPending(pending))
	}
	conversionStep("proved", pending)
	if err := syncPath(pending); err != nil {
		return errors.Join(err, discardPending(pending))
	}
	conversionStep("synced", pending)
	if err := commit(ctx); err != nil {
		return errors.Join(fmt.Errorf("record the move of %s: %w; nothing was changed", m.what, err), discardPending(pending))
	}
	conversionStep("committed", pending)
	return m.finish(ctx, league, leaguePath)
}

func (m fileMove) finish(ctx context.Context, league *db.Pools, leaguePath string) error {
	pending := m.pending(leaguePath)
	if err := removeBuildBackups(pending); err != nil {
		return err
	}
	if err := os.Rename(pending, m.path(leaguePath)); err != nil {
		return fmt.Errorf("put %s in place: %w", m.file, err)
	}
	if err := syncPath(filepath.Dir(leaguePath)); err != nil {
		return err
	}
	if m.after == nil {
		return nil
	}
	return m.after(ctx, league)
}

// checkpointLeague writes the league file's WAL into the file, so pages a move freed (zeroed by
// secure_delete) reach the file itself.
func checkpointLeague(ctx context.Context, league *db.Pools) error {
	if _, err := league.Write().ExecContext(ctx, `PRAGMA wal_checkpoint(TRUNCATE)`); err != nil {
		return fmt.Errorf("flush the move into the league file: %w", err)
	}
	return nil
}

func hasTable(ctx context.Context, pools *db.Pools, name string) (bool, error) {
	return db.HasTable(ctx, pools, name) //nolint:wrapcheck // the message names the file
}

func fileExists(path string) bool {
	_, err := os.Lstat(path)
	return err == nil
}

// removeDatabase deletes a database file and its sidecars; a missing file is not an error.
func removeDatabase(path string) error {
	var errs []error
	for _, name := range []string{path, path + "-wal", path + "-shm", path + "-journal"} {
		if err := os.Remove(name); err != nil && !errors.Is(err, os.ErrNotExist) {
			errs = append(errs, fmt.Errorf("remove %s: %w", filepath.Base(name), err))
		}
	}
	return errors.Join(errs...)
}

// discardPending removes a file that was never put in place, with the copies made while building it.
func discardPending(pending string) error {
	return errors.Join(removeDatabase(pending), removeBuildBackups(pending))
}

// removeBuildBackups deletes the pre-migration copies db.Migrate made of the empty file being built.
func removeBuildBackups(pending string) error {
	matches, err := filepath.Glob(pending + ".pre-v*")
	if err != nil {
		return fmt.Errorf("find build copies: %w", err)
	}
	for _, m := range matches {
		if err := removeDatabase(m); err != nil {
			return err
		}
	}
	return nil
}
