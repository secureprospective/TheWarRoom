package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/db"
	"github.com/secureprospective/TheWarRoom/internal/store/history"
)

// settleLeagueSplit leaves the state folder with every league in its own file, or refuses with a plain
// message. A server from before slice 6 holds all its leagues in server.db; on its first start each
// league is copied into leagues/<league>/league.db.converting and proved, then one commit in server.db
// drops its league tables and records the split (the point of no return), then each proved file takes
// its name. A stop before the commit leaves server.db as it was; a stop after it is finished on the
// next start. A new server goes through the same steps with no leagues. It returns the closed shared
// audit chain's head, which every league file names.
func settleLeagueSplit(ctx context.Context, shared, reference *db.Pools, serverPath string) (db.AuditHead, error) {
	root := leaguesRoot(serverPath)
	split, done, err := history.ReadLeagueSplit(ctx, shared)
	if err != nil {
		return db.AuditHead{}, fmt.Errorf("read server.db's league split: %w", err)
	}
	if done {
		return split.Shared, errors.Join(finishSplit(root, split.Leagues), history.MigrateLeague(ctx, shared))
	}
	leagues, head, err := splitStart(ctx, shared, root)
	if err != nil {
		return db.AuditHead{}, err
	}
	if len(leagues) > 0 {
		if _, err := shared.BackupBeforeChange(ctx); err != nil {
			return db.AuditHead{}, fmt.Errorf("back up server.db before giving each league its own file: %w", err)
		}
	}
	conversionStep("split-backup", root)
	for _, league := range leagues {
		if err := buildLeagueFile(ctx, shared, reference, root, league, head); err != nil {
			return db.AuditHead{}, errors.Join(fmt.Errorf("could not give league %s its own file (%w); nothing was changed and the server did not start", league, err), discardSplit(root, leagues))
		}
	}
	conversionStep("split-proved", root)
	if err := history.CommitLeagueSplit(ctx, shared, history.LeagueSplit{Leagues: leagues, Shared: head, At: time.Now()}); err != nil {
		return db.AuditHead{}, errors.Join(fmt.Errorf("record the league split: %w; nothing was changed", err), discardSplit(root, leagues))
	}
	conversionStep("split-committed", root)
	return head, settleSharedAfterSplit(ctx, shared, root, leagues)
}

// splitStart lists server.db's leagues and its audit head, refusing a split that could not finish.
func splitStart(ctx context.Context, shared *db.Pools, root string) ([]string, db.AuditHead, error) {
	leagues, err := history.SharedLeagues(ctx, shared)
	if err != nil {
		return nil, db.AuditHead{}, fmt.Errorf("list server.db's leagues: %w", err)
	}
	for _, league := range leagues {
		if _, err := leagueFilePath(root, league); err != nil {
			return nil, db.AuditHead{}, fmt.Errorf("server.db holds league %q, which cannot have its own file; nothing was changed. Call Christopher", league)
		}
	}
	if err := refuseStrayLeagueFiles(root); err != nil {
		return nil, db.AuditHead{}, err
	}
	head, err := history.SharedAuditHead(ctx, shared)
	if err != nil {
		return nil, db.AuditHead{}, err //nolint:wrapcheck // names the chain
	}
	return leagues, head, nil
}

// settleSharedAfterSplit puts the committed league files in place and leaves server.db, which now holds
// no league tables, at this release's league version and compacted.
func settleSharedAfterSplit(ctx context.Context, shared *db.Pools, root string, leagues []string) error {
	if err := finishSplit(root, leagues); err != nil {
		return err
	}
	if err := history.MigrateLeague(ctx, shared); err != nil {
		return err //nolint:wrapcheck // names the file
	}
	if len(leagues) == 0 {
		return nil
	}
	return history.CompactLeague(ctx, shared) //nolint:wrapcheck // names the file
}

func pendingLeague(final string) string { return final + ".converting" }

// refuseStrayLeagueFiles stops a split that would sit beside league files server.db never made.
func refuseStrayLeagueFiles(root string) error {
	entries, err := os.ReadDir(root)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("read the league files: %w", err)
	}
	for _, entry := range entries {
		if fileExists(filepath.Join(root, entry.Name(), leagueFileName)) {
			return fmt.Errorf("found a league file for %s, but server.db never gave its leagues their own files; nothing was changed. Check that -db names this server's database", entry.Name())
		}
	}
	return nil
}

// buildLeagueFile copies league into its pending file and proves the copy, then closes the file.
func buildLeagueFile(ctx context.Context, shared, reference *db.Pools, root, league string, head db.AuditHead) (err error) {
	final, err := leagueFilePath(root, league)
	if err != nil {
		return err
	}
	pending := pendingLeague(final)
	if err := discardPending(pending); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(final), 0o700); err != nil {
		return fmt.Errorf("make the league folder: %w", err)
	}
	pools, err := db.OpenWithReference(ctx, pending, history.LeagueSchemaVersion(), reference)
	if err != nil {
		return fmt.Errorf("create the league file: %w", err)
	}
	defer func() { err = errors.Join(err, pools.Close()) }()
	if err := history.PrepareNewLeague(ctx, pools); err != nil {
		return err //nolint:wrapcheck // names the file
	}
	if err := history.CopyLeague(ctx, shared, pools, history.LeagueFile{League: league, Shared: head}); err != nil {
		return err //nolint:wrapcheck // names the league
	}
	conversionStep("split-copied", pending)
	if err := history.ProveLeague(ctx, shared, pools, league); err != nil {
		return err //nolint:wrapcheck // names the league and table
	}
	if _, err := pools.Write().ExecContext(ctx, `PRAGMA wal_checkpoint(TRUNCATE)`); err != nil {
		return fmt.Errorf("flush the league file: %w", err)
	}
	return syncPath(pending)
}

// discardSplit removes every pending league file of a split that did not commit.
func discardSplit(root string, leagues []string) error {
	var errs []error
	for _, league := range leagues {
		final, err := leagueFilePath(root, league)
		if err != nil {
			continue
		}
		errs = append(errs, discardPending(pendingLeague(final)))
	}
	return errors.Join(errs...)
}

// finishSplit puts each committed league's proved file in place. A league whose file is already in
// place (or was archived since) needs nothing; one with neither file nor pending file is lost.
func finishSplit(root string, leagues []string) error {
	for _, league := range leagues {
		final, err := leagueFilePath(root, league)
		if err != nil {
			return err
		}
		pending := pendingLeague(final)
		switch {
		case fileExists(final):
			if err := discardPending(pending); err != nil {
				return err
			}
		case fileExists(pending):
			if err := removeBuildBackups(pending); err != nil {
				return err
			}
			if err := os.Rename(pending, final); err != nil {
				return fmt.Errorf("put league %s's file in place: %w", league, err)
			}
			if err := syncPath(filepath.Dir(final)); err != nil {
				return err
			}
		case leagueArchived(root, league):
		default:
			return fmt.Errorf("league %s's file is missing; server.db says it was given its own file there. Nothing was changed. Restore it from backup (RESTORE.md)", league)
		}
	}
	return nil
}
