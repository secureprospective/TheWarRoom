package main

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"filippo.io/age"
	"github.com/secureprospective/TheWarRoom/internal/db"
	"github.com/secureprospective/TheWarRoom/internal/offsite"
	"github.com/secureprospective/TheWarRoom/internal/serverversion"
	accountstore "github.com/secureprospective/TheWarRoom/internal/store/accounts"
	"github.com/secureprospective/TheWarRoom/internal/store/history"
)

// A server that moved its sign-ins is backed up as two kinds of set: league sets and sign-in sets.
// Restore unlocks the chosen league set, then the sign-in set nearest after it (the sign-in set is
// captured last in every round, so it holds every account the league names), and refuses a pair in
// which the league names an account the sign-in set lacks.

// targetHolds reports whether the freshly installed server holds a league or accounts in either file.
func targetHolds(ctx context.Context, target string) (bool, error) {
	if leagues, err := leagueFileList(leaguesRoot(target)); err != nil || len(leagues) > 0 {
		return len(leagues) > 0, err
	}
	held, err := holdsLeague(ctx, target)
	if err != nil || held || !fileExists(signInPath(target)) {
		return held, err
	}
	return holdsLeague(ctx, signInPath(target))
}

// attachSignIns puts the matching sign-in file beside a restored league file that needs one.
func attachSignIns(ctx context.Context, sets recoverySets, identity *age.X25519Identity, dir string, captured time.Time, output io.Writer) error {
	league := filepath.Join(dir, "server.db")
	moved := stagedSplit(dir)
	if !moved {
		var err error
		if moved, err = leagueHas(ctx, league, signInMoveTable); err != nil || !moved {
			return err // a set saved before sign-ins moved carries them in its league file
		}
	}
	names, err := sets.KindSets(ctx, offsite.SignInSet)
	if err != nil {
		return fmt.Errorf("list sign-in backups: %w", err)
	}
	named, err := stagedPrincipals(ctx, dir)
	if err != nil {
		return err
	}
	for i, name := range nearestAfter(names, captured)[:min(len(names), 64)] {
		sub := filepath.Join(filepath.Dir(dir), fmt.Sprintf("%s-signins-%02d", filepath.Base(dir), i))
		missing, err := unlockSignIns(ctx, sets, name, identity, sub, named)
		if errors.Is(err, offsite.ErrRestoreKey) || errors.Is(err, errNewerSet) {
			return err
		}
		if err == nil && len(missing) == 0 {
			err = os.Rename(filepath.Join(sub, signInFileName), filepath.Join(dir, signInFileName))
			return errors.Join(err, os.RemoveAll(sub))
		}
		if err == nil {
			err = fmt.Errorf("the league names %d account(s) it does not hold: %s", len(missing), strings.Join(missing, ", "))
		}
		if _, werr := fmt.Fprintf(output, "Skipping sign-in backup %s: %v\n", name, err); werr != nil {
			return fmt.Errorf("report restore: %w", werr)
		}
		if err := os.RemoveAll(sub); err != nil {
			return fmt.Errorf("remove skipped backup: %w", err)
		}
	}
	return fmt.Errorf("no sign-in backup matches the league backup saved %s (every account the league names must be in it); nothing was installed",
		captured.UTC().Format(time.RFC3339))
}

// nearestAfter orders sign-in sets: those saved at or after the league set, oldest first, then the
// earlier ones, newest first. Unreadable names go last.
func nearestAfter(names []string, captured time.Time) []string {
	key := func(name string) (int, time.Duration) {
		at, err := offsite.SetTime(name)
		switch {
		case err != nil:
			return 2, 0
		case !at.Before(captured):
			return 0, at.Sub(captured)
		default:
			return 1, captured.Sub(at)
		}
	}
	out := slices.Clone(names)
	slices.SortStableFunc(out, func(a, b string) int {
		ga, da := key(a)
		gb, db := key(b)
		return cmp.Or(cmp.Compare(ga, gb), cmp.Compare(da, db))
	})
	return out
}

// unlockSignIns decrypts one sign-in set, checks it against the checkpoint saved inside it, and
// returns the principals the league names that it lacks.
func unlockSignIns(ctx context.Context, sets recoverySets, name string, identity *age.X25519Identity, dir string, named []string) ([]string, error) {
	body, err := sets.Open(ctx, name)
	if err != nil {
		return nil, fmt.Errorf("download backup: %w", err)
	}
	defer func() { _ = body.Close() }()
	manifest, err := offsite.Restore(body, identity, dir)
	if err != nil {
		return nil, fmt.Errorf("unlock backup: %w", err)
	}
	if !manifest.Holds(offsite.SignInSet.File) {
		return nil, fmt.Errorf("it is not a sign-in backup")
	}
	path := filepath.Join(dir, signInFileName)
	file, err := db.Inspect(ctx, path, accountstore.ServerVersion())
	if file.Schema > accountstore.ServerVersion() || file.Release > serverversion.Current().Release {
		return nil, errNewerSet
	}
	if err != nil {
		return nil, fmt.Errorf("read sign-in file: %w", err)
	}
	saved, err := readCheckpoint(filepath.Join(dir, "checkpoint.json"))
	if err != nil {
		return nil, err
	}
	if saved.Security == nil {
		return nil, fmt.Errorf("its checkpoint names no sign-in history")
	}
	if err := checkAnchors(ctx, rawCheckpoint{Captured: saved.Captured, Security: saved.Security}, "",
		signInSource{path: path, version: accountstore.ServerVersion()}); err != nil {
		return nil, err
	}
	return missingAccounts(ctx, path, named)
}

// stagedPrincipals is every account any staged league file names.
func stagedPrincipals(ctx context.Context, dir string) ([]string, error) {
	files, err := stagedLeagues(dir)
	if err != nil {
		return nil, err
	}
	var out []string
	for _, file := range files {
		named, err := leaguePrincipals(ctx, file)
		if err != nil {
			return nil, err
		}
		out = append(out, named...)
	}
	slices.Sort(out)
	return slices.Compact(out), nil
}

func leaguePrincipals(ctx context.Context, league string) ([]string, error) {
	pools, err := db.OpenReadOnly(ctx, league, history.LeagueSchemaVersion())
	if err != nil {
		return nil, fmt.Errorf("open restored database: %w", err)
	}
	defer func() { _ = pools.Close() }()
	return history.NamedPrincipals(ctx, pools) //nolint:wrapcheck // the message names what was read
}

func missingAccounts(ctx context.Context, path string, named []string) ([]string, error) {
	pools, err := db.OpenReadOnly(ctx, path, accountstore.ServerVersion())
	if err != nil {
		return nil, fmt.Errorf("open sign-in file: %w", err)
	}
	defer func() { _ = pools.Close() }()
	return accountstore.MissingAccounts(ctx, pools, named) //nolint:wrapcheck // the message names what was read
}

// restoredFiles are the staged league file and, for a server that moved them out, its sign-in file and
// its reference file (attached to the league file, as on the running server).
type restoredFiles struct {
	league, signIns, reference *db.Pools
	// leagues are each league's own file, for a server whose leagues have them (league is then server.db).
	leagues []*db.Pools
	names   []string
}

func openRestored(ctx context.Context, league string) (restoredFiles, error) {
	var pools, reference *db.Pools
	var err error
	if fileExists(referencePath(league)) {
		reference, err = db.Open(ctx, referencePath(league), history.ReferenceSchemaVersion())
		if err == nil {
			pools, err = db.OpenWithReference(ctx, league, history.LeagueSchemaVersion(), reference)
		}
	} else {
		pools, err = db.Open(ctx, league, history.LeagueSchemaVersion())
	}
	if err != nil {
		if reference != nil {
			err = errors.Join(err, reference.Close())
		}
		return restoredFiles{}, fmt.Errorf("open restored database: %w", err)
	}
	f := restoredFiles{league: pools, reference: reference}
	dir := filepath.Dir(league)
	if stagedSplit(dir) {
		names, err := leagueFileList(filepath.Join(dir, leaguesFolder))
		if err != nil {
			return restoredFiles{}, errors.Join(err, f.Close())
		}
		for _, name := range names {
			path, _ := leagueFilePath(filepath.Join(dir, leaguesFolder), name)
			open, err := db.OpenWithReference(ctx, path, history.LeagueSchemaVersion(), reference)
			if err != nil {
				return restoredFiles{}, errors.Join(fmt.Errorf("open restored league %s: %w", name, err), f.Close())
			}
			f.leagues, f.names = append(f.leagues, open), append(f.names, name)
		}
	}
	if !fileExists(signInPath(league)) {
		return f, nil
	}
	if f.signIns, err = db.Open(ctx, signInPath(league), accountstore.ServerVersion()); err != nil {
		return restoredFiles{}, errors.Join(fmt.Errorf("open restored sign-in file: %w", err), f.Close())
	}
	return f, nil
}

// store is where the restored sign-ins live: the sign-in file, or a one-file set's league file
// (which then moves its sign-ins out on the restored server's first start).
func (f restoredFiles) store() *accountstore.Store {
	if f.signIns == nil {
		return accountstore.New(f.league)
	}
	return accountstore.NewServerFile(f.signIns)
}

// finish flushes each staged file's WAL and closes it, so each is one complete file.
func (f restoredFiles) finish(ctx context.Context) error {
	for _, pools := range append(append([]*db.Pools{}, f.leagues...), f.league, f.signIns, f.reference) {
		if pools == nil {
			continue
		}
		if _, err := pools.Write().ExecContext(ctx, `PRAGMA wal_checkpoint(TRUNCATE)`); err != nil {
			return fmt.Errorf("finish restored database: %w", err)
		}
		if err := pools.Close(); err != nil {
			return fmt.Errorf("close restored database: %w", err)
		}
	}
	return nil
}

func (f restoredFiles) Close() error {
	var errs []error
	for _, pools := range append(append([]*db.Pools{}, f.leagues...), f.signIns, f.league, f.reference) {
		if pools != nil {
			errs = append(errs, pools.Close())
		}
	}
	return errors.Join(errs...)
}
