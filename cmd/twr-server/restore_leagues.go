package main

import (
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
	"github.com/secureprospective/TheWarRoom/internal/measures"
	"github.com/secureprospective/TheWarRoom/internal/offsite"
	"github.com/secureprospective/TheWarRoom/internal/serverversion"
	accountstore "github.com/secureprospective/TheWarRoom/internal/store/accounts"
	"github.com/secureprospective/TheWarRoom/internal/store/history"
)

// Once each league has its own file, each has its own off-site sets (leagues/<league>/…). A whole
// server comes back as every league's newest usable set, the sign-in set and the reference set nearest
// after the newest of them, and a new server.db that records the split. The staging folder is laid out
// as the server's state folder is: server.db, leagues/<league>/league.db, accounts.db, reference.db.

// stagedLeagues is every league file of a staged restore: each league's own file, or server.db for a
// set saved before leagues had their own files.
func stagedLeagues(dir string) ([]string, error) {
	leagues, err := leagueFileList(filepath.Join(dir, leaguesFolder))
	if err != nil {
		return nil, err
	}
	if len(leagues) == 0 {
		return []string{filepath.Join(dir, "server.db")}, nil
	}
	out := make([]string, len(leagues))
	for i, league := range leagues {
		out[i], _ = leagueFilePath(filepath.Join(dir, leaguesFolder), league)
	}
	return out, nil
}

// stagedSplit reports whether a staged restore holds league files (a set saved after the split).
func stagedSplit(dir string) bool { return fileExists(filepath.Join(dir, leaguesFolder)) }

// leagueSetNames lists each league's own sets, newest first, by league; before as candidateSets.
func leagueSetNames(ctx context.Context, sets recoverySets, before time.Time) (map[string][]string, error) {
	names, err := sets.KindSets(ctx, offsite.SetKind{Prefix: offsite.LeagueFilePrefix})
	if err != nil {
		return nil, fmt.Errorf("list league backups: %w", err)
	}
	out := map[string][]string{}
	for _, name := range names {
		league, _, ok := strings.Cut(strings.TrimPrefix(name, offsite.LeagueFilePrefix), "/")
		if !ok || !validLeagueID(league) {
			continue
		}
		if !before.IsZero() {
			if at, err := offsite.SetTime(name); err != nil || !at.Before(before) {
				continue
			}
		}
		out[league] = append(out[league], name)
	}
	return out, nil
}

// unlockLeagueSet decrypts one league's set into dir and checks it: from no newer release, matching the
// checkpoint saved with it, and a file of exactly that league.
func unlockLeagueSet(ctx context.Context, sets recoverySets, name, league string, identity *age.X25519Identity, dir string) (time.Time, error) {
	body, err := sets.Open(ctx, name)
	if err != nil {
		return time.Time{}, fmt.Errorf("download backup: %w", err)
	}
	defer func() { _ = body.Close() }()
	manifest, err := offsite.Restore(body, identity, dir)
	if err != nil {
		return time.Time{}, fmt.Errorf("unlock backup: %w", err)
	}
	path := filepath.Join(dir, leagueFileName)
	if !manifest.Holds(leagueFileName) {
		return time.Time{}, fmt.Errorf("it is not a league backup")
	}
	file, err := db.Inspect(ctx, path, history.LeagueSchemaVersion())
	if file.Schema > history.LeagueSchemaVersion() || file.Release > serverversion.Current().Release {
		return time.Time{}, errNewerSet
	}
	if err != nil {
		return time.Time{}, fmt.Errorf("read league file: %w", err)
	}
	if err := runCheckpoint(ctx, []string{"-db", path, "-check", filepath.Join(dir, "checkpoint.json")}, io.Discard); err != nil {
		return time.Time{}, err
	}
	pools, err := db.OpenReadOnly(ctx, path, history.LeagueSchemaVersion())
	if err != nil {
		return time.Time{}, fmt.Errorf("open league file: %w", err)
	}
	defer func() { _ = pools.Close() }()
	if err := checkLeagueFile(ctx, pools, league); err != nil {
		return time.Time{}, fmt.Errorf("league %s: %w", league, err)
	}
	return manifest.Captured, nil
}

// restoreLeagueSet puts league's newest usable set at target, keeping the set's settings folder in
// keep when keep is not empty. It returns when that set was saved.
func restoreLeagueSet(ctx context.Context, sets recoverySets, names []string, league string, identity *age.X25519Identity,
	scratch, target, keep string, output io.Writer) (time.Time, error) {
	if err := os.MkdirAll(scratch, 0o700); err != nil {
		return time.Time{}, fmt.Errorf("create restore folder: %w", err)
	}
	for i, name := range names[:min(len(names), 64)] {
		dir := filepath.Join(scratch, fmt.Sprintf("%s-%02d", league, i))
		captured, err := unlockLeagueSet(ctx, sets, name, league, identity, dir)
		if errors.Is(err, offsite.ErrRestoreKey) || errors.Is(err, errNewerSet) {
			return time.Time{}, errors.Join(err, os.RemoveAll(dir))
		}
		if err != nil {
			if _, werr := fmt.Fprintf(output, "Skipping %s: %v\n", name, err); werr != nil {
				return time.Time{}, fmt.Errorf("report restore: %w", werr)
			}
			if err := os.RemoveAll(dir); err != nil {
				return time.Time{}, fmt.Errorf("remove skipped backup: %w", err)
			}
			continue
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o700); err != nil {
			return time.Time{}, fmt.Errorf("stage league %s: %w", league, err)
		}
		if err := os.Rename(filepath.Join(dir, leagueFileName), target); err != nil {
			return time.Time{}, fmt.Errorf("stage league %s: %w", league, err)
		}
		if keep != "" && !fileExists(keep) {
			if err := os.Rename(filepath.Join(dir, "config"), keep); err != nil {
				return time.Time{}, fmt.Errorf("stage the server settings: %w", err)
			}
		}
		if err := os.RemoveAll(dir); err != nil {
			return time.Time{}, fmt.Errorf("clear the restore folder of league %s: %w", league, err)
		}
		return captured, nil
	}
	return time.Time{}, fmt.Errorf("no usable backup of league %s was found", league)
}

// restoreLeagueFiles stages a whole server from its leagues' own sets: every league's newest usable set,
// then a new server.db recording the split. It returns the newest league capture time.
func restoreLeagueFiles(ctx context.Context, sets recoverySets, byLeague map[string][]string, identity *age.X25519Identity,
	stage, dir string, output io.Writer) (time.Time, error) {
	leagues := make([]string, 0, len(byLeague))
	for league := range byLeague {
		leagues = append(leagues, league)
	}
	slices.Sort(leagues)
	scratch := filepath.Join(stage, "leagues")
	if err := os.MkdirAll(filepath.Join(dir, leaguesFolder), 0o700); err != nil {
		return time.Time{}, fmt.Errorf("create restore folder: %w", err)
	}
	var newest time.Time
	var restored []string
	for _, league := range leagues {
		target, _ := leagueFilePath(filepath.Join(dir, leaguesFolder), league)
		captured, err := restoreLeagueSet(ctx, sets, byLeague[league], league, identity, scratch, target, filepath.Join(dir, "config"), output)
		if errors.Is(err, offsite.ErrRestoreKey) || errors.Is(err, errNewerSet) {
			return time.Time{}, err
		}
		if err != nil {
			return time.Time{}, fmt.Errorf("%w; nothing was installed", err)
		}
		restored = append(restored, league)
		if captured.After(newest) {
			newest = captured
		}
	}
	if len(restored) == 0 {
		return time.Time{}, fmt.Errorf("no usable backup was found; nothing was installed")
	}
	return newest, writeSplitShared(ctx, dir, restored)
}

// writeSplitShared writes the staged server.db of a restored server: an empty shared file recording
// that its sign-ins, football reference data and leagues each have their own files, with the closed shared head the league files name.
func writeSplitShared(ctx context.Context, dir string, leagues []string) (err error) {
	first, _ := leagueFilePath(filepath.Join(dir, leaguesFolder), leagues[0])
	league, err := db.OpenReadOnly(ctx, first, history.LeagueSchemaVersion())
	if err != nil {
		return fmt.Errorf("open restored league file: %w", err)
	}
	file, err := history.ReadLeagueFile(ctx, league)
	if err = errors.Join(err, league.Close()); err != nil {
		return err
	}
	shared, err := db.Open(ctx, filepath.Join(dir, "server.db"), history.LeagueSchemaVersion())
	if err != nil {
		return fmt.Errorf("create the restored server.db: %w", err)
	}
	defer func() { err = errors.Join(err, shared.Close()) }()
	if err := history.PrepareNewLeague(ctx, shared); err != nil {
		return err //nolint:wrapcheck // names the file
	}
	// The restored server's sign-ins and football reference data live in their own files, as on the
	// server that was backed up: server.db says so, as the moves out of it did.
	if err := accountstore.CommitMove(ctx, shared, false, accountstore.MoveRecord{File: signInFileName}); err != nil {
		return err //nolint:wrapcheck // names the step
	}
	if err := history.CommitReferenceMove(ctx, shared, history.ReferenceMove{File: referenceFileName}); err != nil {
		return err //nolint:wrapcheck // names the step
	}
	if err := history.RecordLeagueSplit(ctx, shared, history.LeagueSplit{Leagues: leagues, Shared: file.Shared, At: time.Now()}); err != nil {
		return err //nolint:wrapcheck // names the step
	}
	_, err = shared.Write().ExecContext(ctx, `PRAGMA wal_checkpoint(TRUNCATE)`)
	return err //nolint:wrapcheck // rare; the restore names the step
}

// restoredReport reads what a restore installs, before and after its sign-in reset: each league file's
// chain and fingerprints (or server.db's, for a set saved before leagues had their own files).
func restoredReport(ctx context.Context, restored restoredFiles, registry *measures.Registry, security *accountstore.Store) (restoreReport, error) {
	if len(restored.leagues) == 0 {
		return readRestoreReport(ctx, history.New(restored.league, registry), security)
	}
	var r restoreReport
	var err error
	if r.security, err = security.VerifyAudit(ctx); err != nil {
		return r, fmt.Errorf("verify restored sign-in history: %w", err)
	}
	for i, pools := range restored.leagues {
		ledger := history.New(pools, registry)
		head, err := ledger.VerifyAudit(ctx)
		if err != nil {
			return r, fmt.Errorf("verify restored league %s: %w", restored.names[i], err)
		}
		r.leagues = append(r.leagues, leagueHead{league: restored.names[i], head: head})
		digests, err := ledger.RecoveryDigests(ctx)
		if err != nil {
			return r, fmt.Errorf("fingerprint restored league %s: %w", restored.names[i], err)
		}
		r.digests.Leagues = append(r.digests.Leagues, digests.Leagues...)
		r.digests.EvidenceFiles, r.digests.Evidence = digests.EvidenceFiles, digests.Evidence
	}
	return r, nil
}
