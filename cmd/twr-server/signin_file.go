package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/secureprospective/TheWarRoom/internal/db"
	accountstore "github.com/secureprospective/TheWarRoom/internal/store/accounts"
)

// The hosted server keeps two files in its state folder: the league file (-db, server.db) and the
// sign-in file beside it. Every command finds the sign-in file through signInPath and opens it
// through openSignIns, which also moves the sign-in tables out of a one-file server on first start.
const signInFileName = "accounts.db"

// signInMoveTable is the league file's record that its sign-in tables live in the sign-in file. It is
// written in the same transaction that removes them, so the league file says which layout it is in.
const signInMoveTable = accountstore.MoveRecordTable

func signInPath(leaguePath string) string {
	return filepath.Join(filepath.Dir(leaguePath), signInFileName)
}

// pendingSignIns is where a conversion builds the sign-in file; it becomes the sign-in file only after
// the league file has committed the move.
func pendingSignIns(leaguePath string) string { return signInPath(leaguePath) + ".converting" }

// conversionStep lets a test stop the process between conversion steps, as a crash would, or change
// the file being built. pending is the sign-in file under construction.
var conversionStep = func(_, _ string) {} //nolint:gochecknoglobals // a test seam: only tests replace it

// openSignIns opens the sign-in file for the league file at leaguePath, converting a one-file server first.
func openSignIns(ctx context.Context, league *db.Pools, leaguePath string) (*db.Pools, error) {
	if err := settleSignIns(ctx, league, leaguePath); err != nil {
		return nil, err
	}
	pools, err := db.Open(ctx, signInPath(leaguePath), accountstore.ServerVersion())
	if err != nil {
		return nil, fmt.Errorf("open the sign-in file: %w", err)
	}
	return pools, nil
}

// signInLayout is what the state folder holds: sign-in tables in the league file, the league file's
// move record, the sign-in file, and a sign-in file still being built.
type signInLayout struct{ held, moved, final, pending bool }

func readLayout(ctx context.Context, league *db.Pools, leaguePath string) (signInLayout, error) {
	var l signInLayout
	var err error
	if l.held, err = hasTable(ctx, league, "accounts"); err != nil {
		return l, err
	}
	if l.moved, err = hasTable(ctx, league, signInMoveTable); err != nil {
		return l, err
	}
	l.final, l.pending = fileExists(signInPath(leaguePath)), fileExists(pendingSignIns(leaguePath))
	return l, nil
}

// settleSignIns leaves the folder in the two-file layout or refuses with a plain message. The commit
// that drops the sign-in tables from the league file is the only point of no return: before it the
// old layout is intact and a half-built sign-in file is discarded; after it the proven file is in place.
func settleSignIns(ctx context.Context, league *db.Pools, leaguePath string) error {
	l, err := readLayout(ctx, league, leaguePath)
	if err != nil {
		return err
	}
	switch {
	case l.moved && l.held:
		return fmt.Errorf("the league file records that sign-ins moved out but still holds them; nothing was changed. Restore from backup (RESTORE.md)")
	case l.moved && l.final:
		return nil
	case l.moved && l.pending:
		// The move committed, then the process stopped before the proven file took its name.
		return finishPending(leaguePath)
	case l.moved:
		return fmt.Errorf("the sign-in file %s is missing; the league file says sign-ins were moved there. Nothing was changed. Restore it from backup (RESTORE.md)", signInFileName)
	case l.final:
		return fmt.Errorf("found a sign-in file %s, but the league file never moved its sign-ins there; nothing was changed. Check that -db names this server's league file", signInFileName)
	}
	if err := discardPending(pendingSignIns(leaguePath)); err != nil {
		return err
	}
	return convertSignIns(ctx, league, leaguePath, l.held)
}

// convertSignIns builds the sign-in file beside the league file. With held, it copies the league
// file's sign-in tables and security chain, proves the copy, then drops them from the league file.
func convertSignIns(ctx context.Context, league *db.Pools, leaguePath string, held bool) error {
	pending := pendingSignIns(leaguePath)
	if held {
		if _, err := league.BackupBeforeChange(ctx); err != nil {
			return fmt.Errorf("back up the league file before moving sign-ins: %w", err)
		}
	}
	conversionStep("backup", pending)
	moved, err := buildSignIns(ctx, league, pending, held)
	if err != nil {
		return errors.Join(fmt.Errorf("could not move sign-ins to their own file (%w); nothing was changed and the server did not start", err), discardPending(pending))
	}
	conversionStep("proved", pending)
	if err := syncPath(pending); err != nil {
		return errors.Join(err, discardPending(pending))
	}
	conversionStep("synced", pending)
	if err := commitMove(ctx, league, held, moved); err != nil {
		return errors.Join(fmt.Errorf("record the sign-in move: %w; nothing was changed", err), discardPending(pending))
	}
	conversionStep("committed", pending)
	if err := finishPending(leaguePath); err != nil {
		return err
	}
	// The freed pages were zeroed in the WAL; the checkpoint writes the zeroed pages into the file.
	if _, err := league.Write().ExecContext(ctx, `PRAGMA wal_checkpoint(TRUNCATE)`); err != nil {
		return fmt.Errorf("flush the sign-in move into the league file: %w", err)
	}
	return nil
}

func finishPending(leaguePath string) error {
	if err := removeBuildBackups(pendingSignIns(leaguePath)); err != nil {
		return err
	}
	if err := os.Rename(pendingSignIns(leaguePath), signInPath(leaguePath)); err != nil {
		return fmt.Errorf("put the sign-in file in place: %w", err)
	}
	return syncPath(filepath.Dir(leaguePath))
}

// buildSignIns writes and proves the new file, then closes it so nothing of it is left in a WAL.
func buildSignIns(ctx context.Context, source *db.Pools, path string, held bool) (moved accountstore.MoveRecord, err error) {
	moved.File = signInFileName
	pools, err := db.Open(ctx, path, accountstore.ServerVersion())
	if err != nil {
		return moved, fmt.Errorf("create the sign-in file: %w", err)
	}
	defer func() { err = errors.Join(err, pools.Close()) }()
	if err := accountstore.PrepareServerFile(ctx, pools); err != nil {
		return moved, fmt.Errorf("prepare the sign-in file: %w", err)
	}
	if held {
		if err := accountstore.CopySignIns(ctx, source, pools); err != nil {
			return moved, fmt.Errorf("copy sign-ins: %w", err)
		}
		conversionStep("copied", path)
		if err := accountstore.ProveSignIns(ctx, source, pools); err != nil {
			return moved, fmt.Errorf("prove the copy: %w", err)
		}
	}
	store := accountstore.NewServerFile(pools)
	if err := store.Initialize(ctx); err != nil {
		return moved, fmt.Errorf("verify the security history in its new file: %w", err)
	}
	if moved.Security, err = store.VerifyAudit(ctx); err != nil {
		return moved, fmt.Errorf("verify the security history in its new file: %w", err)
	}
	if held {
		if err := sameSecurityHead(ctx, source, moved.Security); err != nil {
			return moved, err
		}
	}
	if moved.Accounts, err = accountstore.CountAccounts(ctx, pools); err != nil {
		return moved, fmt.Errorf("count moved accounts: %w", err)
	}
	if _, err := pools.Write().ExecContext(ctx, `PRAGMA wal_checkpoint(TRUNCATE)`); err != nil {
		return moved, fmt.Errorf("finish the sign-in file: %w", err)
	}
	return moved, nil
}

// sameSecurityHead checks the chain arrived with its hashes intact: the new head is the old head.
func sameSecurityHead(ctx context.Context, source *db.Pools, head db.AuditHead) error {
	old, found, err := accountstore.SecurityHead(ctx, source)
	if err != nil {
		return fmt.Errorf("read the security history head: %w", err)
	}
	if found && (old.Size != head.Size || old.Hash != head.Hash) {
		return fmt.Errorf("the security history head changed in the move")
	}
	return nil // not found: the league file never adopted the chain; the new file adopted it from the rows
}

func commitMove(ctx context.Context, league *db.Pools, held bool, moved accountstore.MoveRecord) error {
	return accountstore.CommitMove(ctx, league, held, moved) //nolint:wrapcheck // the caller names the step
}

func hasTable(ctx context.Context, pools *db.Pools, name string) (bool, error) {
	return accountstore.HasTable(ctx, pools, name) //nolint:wrapcheck // the message names the file
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

// discardPending removes a sign-in file that was never put in place, with the copies made while building it.
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
