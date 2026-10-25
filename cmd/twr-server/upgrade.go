package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/db"
	"github.com/secureprospective/TheWarRoom/internal/measures"
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
	current := serverversion.Current()
	return report(output, fmt.Sprintf("Upgraded the league file to version %d (release %d). The league as it was before is saved in %s.", current.Schema, current.Release, save))
}

// saveMatchedCopy writes the pre-upgrade league into save and, with backup settings, uploads and
// reads back an off-site set of it. Neither step changes the league file.
func saveMatchedCopy(ctx context.Context, path, save, backup string) (string, error) {
	if err := os.MkdirAll(filepath.Dir(save), 0o700); err != nil {
		return "", fmt.Errorf("create the upgrade folder: %w", err)
	}
	if err := os.Mkdir(save, 0o700); err != nil {
		return "", fmt.Errorf("create the pre-upgrade folder (it must be new): %w", err)
	}
	pools, err := db.Open(ctx, path, history.SchemaVersion())
	if err != nil {
		return "", fmt.Errorf("open server database: %w", err)
	}
	defer func() { _ = pools.Close() }()
	saved := filepath.Join(save, "server.db")
	if err := pools.CaptureHosted(ctx, saved, func() error { return syncPath(saved) }); err != nil {
		return "", fmt.Errorf("save the pre-upgrade copy: %w", err)
	}
	if backup != "" {
		if err := upgradeOffsite(ctx, pools, backup, filepath.Dir(path)); err != nil {
			return "", fmt.Errorf("save the off-site backup before upgrading: %w", err)
		}
	}
	return saved, syncPath(save)
}

// migrateServer brings history and accounts to this release and stamps it, so an older binary
// put back over this file refuses it.
func migrateServer(ctx context.Context, path string) (err error) {
	pools, err := db.Open(ctx, path, history.SchemaVersion())
	if err != nil {
		return fmt.Errorf("open server database: %w", err)
	}
	defer func() { err = errors.Join(err, pools.Close()) }()
	registry, err := measures.Embedded()
	if err != nil {
		return fmt.Errorf("load server measures: %w", err)
	}
	if err := history.New(pools, registry).Initialize(ctx); err != nil {
		return fmt.Errorf("upgrade league history: %w", err)
	}
	if err := accountstore.New(pools).Initialize(ctx); err != nil {
		return fmt.Errorf("upgrade accounts: %w", err)
	}
	if err := pools.StampServerRelease(ctx, serverversion.Current().Release, time.Now()); err != nil {
		return fmt.Errorf("record server release: %w", err)
	}
	return nil
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

// putBack restores the pre-upgrade copy over a failed upgrade. Its sidecars belong to the failed
// attempt and go first; the copy replaces the file in one rename.
func putBack(path, saved string, cause error) error {
	restoring := path + ".restoring"
	err := copyFile(saved, restoring)
	if err == nil {
		err = removeSidecars(path)
	}
	if err == nil {
		err = os.Rename(restoring, path)
	}
	if err == nil {
		err = syncPath(filepath.Dir(path))
	}
	if err != nil {
		return fmt.Errorf("the upgrade failed (%w) and the pre-upgrade copy in %s could not be put back (%w); do not start the server, and follow UPGRADE.md", cause, filepath.Dir(saved), err)
	}
	return fmt.Errorf("the upgrade failed (%w); the league file was put back exactly as it was, so the current server can keep running", cause)
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
	out, err := os.OpenFile(dest, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o600)
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
