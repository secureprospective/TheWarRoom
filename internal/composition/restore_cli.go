package composition

import (
	"context"
	"errors"
	"log"

	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/db"
)

// listBackups prints the backup sets in the data directory, oldest first: each one a directory
// under backups/ that holds a manifest. It reads names only; Restore verifies the set it is given.
func listBackups(stdout, stderr io.Writer) int {
	dir, err := configDir()
	if err != nil {
		restoreFailure(stderr, err, "The app could not open its data folder. Check that your account can open and write to its settings folder, then try again.")
		return 1
	}
	entries, err := os.ReadDir(filepath.Join(dir, "backups"))
	if err != nil && !os.IsNotExist(err) {
		restoreFailure(stderr, err, "The saved backups could not be listed. Check that your account can read the backups folder, then try again.")
		return 1
	}
	var sets []string
	for _, e := range entries {
		if _, err := os.Stat(filepath.Join(dir, "backups", e.Name(), "manifest.json")); e.IsDir() && err == nil {
			sets = append(sets, e.Name())
		}
	}
	sort.Strings(sets)
	for _, set := range sets {
		_, _ = fmt.Fprintln(stdout, set)
	}
	if len(sets) == 0 {
		_, _ = fmt.Fprintln(stdout, "no backup sets in "+filepath.Join(dir, "backups"))
	}
	return 0
}

// restoreBackup replaces all three databases with the named set, offline. It takes the same
// instance lock the app does, so it refuses while the app is open, and the files it replaces are
// kept as a pre-restore set beside the others. A profile never opened (a new computer) has no
// files to keep, and the result says so.
func restoreBackup(set string, stdout, stderr io.Writer) int {
	dir, err := configDir()
	if err != nil {
		restoreFailure(stderr, err, "The app could not open its data folder. Check that your account can open and write to its settings folder, then try again.")
		return 1
	}
	lock, err := acquireInstanceLock(filepath.Join(dir, dbFileName("thewarroom", isDevBuild())))
	if err != nil {
		restoreFailure(stderr, err, "Restore needs the app closed and access to its data folder. Close every copy of The War Room, check that your account can write to its data folder, then try again.")
		return 1
	}
	defer releaseInstanceLock(lock)
	backups, err := newDatabaseBackups(dir)
	if err != nil {
		restoreFailure(stderr, err, "Restore could not prepare the backup folder. Check that your account can read and write to the data folder, then try again.")
		return 1
	}
	safety, err := backups.Restore(context.Background(), filepath.Join(dir, "backups", filepath.Base(set)), time.Now().UTC())
	if err != nil {
		message := "The backup could not be restored. Keep the backup and the data folder as they are. Check the backup name with -backups and make sure there is enough free disk space, then try again. If it still fails, ask for help and include the app log."
		if errors.Is(err, db.ErrRestoreJournalNewer) {
			message = err.Error()
		}
		restoreFailure(stderr, err, message)
		return 1
	}
	if safety == "" {
		_, _ = fmt.Fprintf(stdout, "restored %s; this computer had no league data yet, so there was nothing to keep a safety copy of\n", filepath.Base(set))
		return 0
	}
	_, _ = fmt.Fprintf(stdout, "restored %s; the replaced files are kept as the safety copy %s\n", filepath.Base(set), filepath.Base(safety))
	return 0
}

// CLI logs must not tee technical details back to the terminal.
func restoreFailure(stderr io.Writer, cause error, message string) {
	cfg, err := os.UserConfigDir()
	var file *os.File
	if err == nil {
		dir := filepath.Join(cfg, "TheWarRoom", "logs")
		if err = os.MkdirAll(dir, 0o700); err == nil {
			file, err = os.CreateTemp(dir, "restore-*.log")
		}
	}
	if err != nil {
		file, err = os.CreateTemp("", "thewarroom-restore-*.log")
	}
	_, _ = fmt.Fprintln(stderr, message)
	if err != nil {
		_, _ = fmt.Fprintln(stderr, "The detail log could not be saved. Check that your account can write to its folders before trying again.")
		return
	}
	log.New(file, "", log.LstdFlags).Printf("the war room: restore: %v", cause)
	_, _ = fmt.Fprintln(stderr, "Details saved in "+file.Name())
	_ = file.Close()
}
