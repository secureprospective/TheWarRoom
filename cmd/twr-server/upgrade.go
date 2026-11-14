package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/secureprospective/TheWarRoom/internal/db"
	"github.com/secureprospective/TheWarRoom/internal/releasetrust"
	"github.com/secureprospective/TheWarRoom/internal/serverversion"
	accountstore "github.com/secureprospective/TheWarRoom/internal/store/accounts"
	"github.com/secureprospective/TheWarRoom/internal/store/history"
)

// runUpgrade is the server-first upgrade step. The NEW binary runs it as the service user while
// the old server is stopped (deploy/server/upgrade.sh): read-only preflight, a matched copy of the
// league (and a verified off-site set when backups are set up), migration, then a readiness check.
// Any failure after the copy puts the copy back. -check only reads and may run while serving.
func runUpgrade(ctx context.Context, args []string, output io.Writer) error {
	flags := flag.NewFlagSet("twr-server upgrade", flag.ContinueOnError)
	path := flags.String("db", "", "the installed server database")
	check := flags.Bool("check", false, "only read the league file and report; change nothing")
	save := flags.String("save", "", "new private folder for the pre-upgrade copy")
	backup := flags.String("backup-config", "", "private R2 backup settings; when given, a verified off-site set is saved first")
	if err := flags.Parse(args); err != nil {
		return fmt.Errorf("parse upgrade flags: %w", err)
	}
	if flags.NArg() != 0 || *path == "" {
		return fmt.Errorf("upgrade needs the installed server database (-db)")
	}
	plan, err := serverversion.Preflight(ctx, *path)
	if err != nil {
		return fmt.Errorf("check league file: %w", err)
	}
	if *check || plan.State != serverversion.Older {
		return report(output, plan.Describe())
	}
	if *save == "" {
		return fmt.Errorf("upgrade needs -save, a new folder for the pre-upgrade copy")
	}
	return upgradeFile(ctx, *path, *save, *backup, output)
}

func upgradeFile(ctx context.Context, path, save, backup string, output io.Writer) error {
	saved, err := saveMatchedCopy(ctx, path, save, backup)
	if err != nil {
		return fmt.Errorf("%w; nothing was changed", err)
	}
	if err := migrateServer(ctx, path); err != nil {
		return putBack(path, saved, err)
	}
	if err := upgradeReady(ctx, path); err != nil {
		return putBack(path, saved, err)
	}
	if err := runCheckpoint(ctx, []string{"-db", path, "-check", filepath.Join(save, "checkpoint.json")}, io.Discard); err != nil {
		return putBack(path, saved, err)
	}
	current := serverversion.Current()
	return report(output, fmt.Sprintf("Upgraded the league file to version %d (release %d). The league as it was before is saved in %s.", current.Schema, current.Release, save))
}

// saveMatchedCopy writes the pre-upgrade league file, and the sign-in file when the server has one,
// into save and, with backup settings, uploads and reads back an off-site set of each. Nothing here
// changes either file: a one-file server is copied as it is and converted only by the migration.
func saveMatchedCopy(ctx context.Context, path, save, backup string) (string, error) {
	if err := os.MkdirAll(filepath.Dir(save), 0o700); err != nil {
		return "", fmt.Errorf("create the upgrade folder: %w", err)
	}
	if err := os.Mkdir(save, 0o700); err != nil {
		return "", fmt.Errorf("create the pre-upgrade folder (it must be new): %w", err)
	}
	files, err := openForCopy(ctx, path)
	if err != nil {
		return "", err
	}
	defer func() { _ = files.Close() }()
	saved := filepath.Join(save, "server.db")
	if err := files.shared.CaptureHosted(ctx, saved, nil); err != nil {
		return "", fmt.Errorf("save the pre-upgrade copy: %w", err)
	}
	if files.signIns != nil {
		if err := files.signIns.CaptureHosted(ctx, signInPath(saved), nil); err != nil {
			return "", fmt.Errorf("save the pre-upgrade sign-in copy: %w", err)
		}
	}
	if files.reference != nil {
		if err := files.reference.CaptureHosted(ctx, referencePath(saved), nil); err != nil {
			return "", fmt.Errorf("save the pre-upgrade football reference copy: %w", err)
		}
	}
	if err := saveLeagueCopies(ctx, files.leagues, saved); err != nil {
		return "", err
	}
	checkpoint, err := hostedCheckpoint(ctx, saved)
	if err != nil {
		return "", fmt.Errorf("verify the pre-upgrade copy: %w", err)
	}
	if err := os.WriteFile(filepath.Join(save, "checkpoint.json"), checkpoint, 0o600); err != nil {
		return "", fmt.Errorf("save pre-upgrade checkpoint: %w", err)
	}
	if err := syncPath(filepath.Join(save, "checkpoint.json")); err != nil {
		return "", err
	}
	if backup != "" {
		if err := upgradeOffsite(ctx, files, backup, filepath.Dir(path)); err != nil {
			return "", fmt.Errorf("save the off-site backup before upgrading: %w", err)
		}
	}
	return saved, syncPath(save)
}

// openForCopy opens the league file and, if the server already has one, its sign-in file, changing
// neither. signIns is nil for a one-file server.
func openForCopy(ctx context.Context, path string) (stateFiles, error) {
	source, err := locateSignIns(ctx, path)
	if err != nil {
		return stateFiles{}, err
	}
	league, err := db.Open(ctx, path, history.LeagueSchemaVersion())
	if err != nil {
		return stateFiles{}, fmt.Errorf("open server database: %w", err)
	}
	files := stateFiles{shared: league, leagues: &leagueFiles{open: map[string]*leagueFile{}}}
	if source.path != signInPath(path) {
		return files, nil
	}
	if files.signIns, err = db.Open(ctx, source.path, source.version); err != nil {
		return stateFiles{}, errors.Join(fmt.Errorf("open sign-in file: %w", err), league.Close())
	}
	if fileExists(referencePath(path)) {
		if files.reference, err = db.Open(ctx, referencePath(path), history.ReferenceSchemaVersion()); err != nil {
			return stateFiles{}, errors.Join(fmt.Errorf("open football reference file: %w", err), files.Close())
		}
	}
	if files.leagues, err = openLeagueCopies(ctx, leaguesRoot(path)); err != nil {
		return stateFiles{}, errors.Join(err, files.Close())
	}
	return files, nil
}

// migrateServer brings history and accounts to this release and stamps every file, so an older binary
// put back over them refuses them. A one-file server's sign-ins and then its football reference data
// move to their own files here, after the league file's own migration.
func migrateServer(ctx context.Context, path string) (err error) {
	alone, err := db.Open(ctx, path, history.LeagueSchemaVersion())
	if err != nil {
		return fmt.Errorf("open server database: %w", err)
	}
	if err := history.MigrateLeague(ctx, alone); err != nil {
		return errors.Join(fmt.Errorf("upgrade league history: %w", err), alone.Close())
	}
	if err := alone.Close(); err != nil {
		return fmt.Errorf("close server database: %w", err)
	}
	files, err := openSignInFiles(ctx, path)
	if err != nil {
		return err
	}
	defer func() { err = errors.Join(err, files.Close()) }()
	for _, store := range files.leagues.stores() {
		if err := store.Initialize(ctx); err != nil {
			return fmt.Errorf("upgrade league history: %w", err)
		}
	}
	if err := accountstore.NewServerFile(files.signIns).Initialize(ctx); err != nil {
		return fmt.Errorf("upgrade accounts: %w", err)
	}
	return stampRelease(ctx, files)
}

// upgradeReady is the readiness check: the file now matches this release exactly and both
// history chains still verify.
func upgradeReady(ctx context.Context, path string) error {
	plan, err := serverversion.Preflight(ctx, path)
	if err != nil {
		return fmt.Errorf("check upgraded league file: %w", err)
	}
	if plan.State != serverversion.Ready || plan.File.Release != serverversion.Current().Release {
		return fmt.Errorf("the upgraded league file does not match this release")
	}
	if _, err := hostedCheckpoint(ctx, path); err != nil {
		return fmt.Errorf("verify upgraded league history: %w", err)
	}
	return nil
}

// putBack restores the pre-upgrade copies over a failed upgrade. Their sidecars belong to the failed
// attempt and go first; each copy replaces its file in one rename. A server saved before it had a
// sign-in or reference file gets none back: the old binary reads them from the league file.
func putBack(path, saved string, cause error) error {
	err := putBackFile(path, saved)
	for _, move := range []fileMove{signInMove, referenceMove} {
		if err != nil {
			break
		}
		if fileExists(move.path(saved)) {
			err = putBackFile(move.path(path), move.path(saved))
		} else {
			err = errors.Join(removeDatabase(move.path(path)), discardPending(move.pending(path)))
		}
	}
	if err == nil {
		err = putBackLeagues(path, saved)
	}
	if err == nil {
		err = syncPath(filepath.Dir(path))
	}
	if err != nil {
		return fmt.Errorf("the upgrade failed (%w) and the pre-upgrade copy in %s could not be put back (%w); do not start the server, and follow UPGRADE.md", cause, filepath.Dir(saved), err)
	}
	return fmt.Errorf("the upgrade failed (%w); the league file was put back exactly as it was, so the current server can keep running", cause)
}

func putBackFile(path, saved string) error {
	restoring := path + ".restoring"
	err := copyFile(saved, restoring)
	if err == nil {
		err = removeSidecars(path)
	}
	if err != nil {
		return err
	}
	if err := os.Rename(restoring, path); err != nil {
		return fmt.Errorf("put back %s: %w", filepath.Base(path), err)
	}
	return nil
}

func removeSidecars(path string) error {
	for _, side := range []string{path + "-wal", path + "-shm"} {
		if err := os.Remove(side); err != nil && !errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("remove the failed upgrade's %s: %w", filepath.Base(side), err)
		}
	}
	return nil
}

func copyFile(source, dest string) (err error) {
	in, err := os.Open(source)
	if err != nil {
		return fmt.Errorf("open pre-upgrade copy: %w", err)
	}
	defer func() { _ = in.Close() }()
	out, err := os.OpenFile(dest, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
	if err != nil {
		return fmt.Errorf("create restored file: %w", err)
	}
	defer func() { err = errors.Join(err, out.Close()) }()
	if _, err := io.Copy(out, in); err != nil {
		return fmt.Errorf("copy pre-upgrade copy: %w", err)
	}
	if err := out.Sync(); err != nil {
		return fmt.Errorf("flush restored file: %w", err)
	}
	return nil
}

func syncPath(path string) error {
	f, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("open %s to flush it: %w", filepath.Base(path), err)
	}
	defer func() { _ = f.Close() }()
	if err := f.Sync(); err != nil {
		return fmt.Errorf("flush %s: %w", filepath.Base(path), err)
	}
	return nil
}

// runVersion prints what this binary serves, for the operator comparing bundles.
func runVersion(args []string, output io.Writer) error {
	if len(args) != 0 {
		return fmt.Errorf("version takes no arguments")
	}
	current := serverversion.Current()
	oldest := max(current.ProtocolMinor-1, current.SecurityFloor)
	return report(output, fmt.Sprintf("League server release %d: league file version %d; serves The War Room protocol %d.%d to %d.%d; durable commands %d.",
		current.Release, current.Schema, serverversion.ProtocolMajor, oldest, serverversion.ProtocolMajor, current.ProtocolMinor, current.Commands))
}

func report(output io.Writer, line string) error {
	if _, err := fmt.Fprintln(output, line); err != nil {
		return fmt.Errorf("report: %w", err)
	}
	return nil
}

func runVerifyRelease(args []string, output io.Writer) error {
	flags := flag.NewFlagSet("twr-server verify-release", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	root := flags.String("root", "/etc/twr-release/root.json", "installed, independently trusted public root")
	cache := flags.String("cache", "/var/lib/twr-release", "private persistent trust cache; use one verifier at a time; never clear it to retry")
	directory := flags.String("metadata", "", "folder containing delivered TUF metadata")
	bundle := flags.String("bundle", "", "server bundle to authenticate, never run or unpack")
	installed := flags.Int("installed-release", 0, "installed server's monotonic release number (required)")
	if err := flags.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			flags.SetOutput(output)
			flags.PrintDefaults()
			return nil
		}
		return fmt.Errorf("release not checked; run twr-server verify-release -help and correct the options")
	}
	if flags.NArg() != 0 || *directory == "" || *bundle == "" || *installed < 1 {
		return fmt.Errorf("release not checked; provide -metadata, -bundle and -installed-release, then try again")
	}
	release, digest, err := releasetrust.VerifyServerBundle(*root, *cache, *directory, *bundle, *installed)
	if err != nil {
		return fmt.Errorf("release refused; keep the installed server and ask your release provider for a fresh signed bundle and metadata; do not clear the trust cache")
	}
	if _, err := fmt.Fprintf(output, "verified release %d sha256 %s\n", release, digest); err != nil {
		return fmt.Errorf("report verified release: %w", err)
	}
	return nil
}
