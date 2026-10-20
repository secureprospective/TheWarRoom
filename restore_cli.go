package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"time"
)

// listBackups prints the backup sets in the data directory, oldest first: each one a directory
// under backups/ that holds a manifest. It reads names only; Restore verifies the set it is given.
func listBackups(stdout, stderr io.Writer) int {
	dir, err := configDir()
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "thewarroom: %s\n", err)
		return 1
	}
	entries, err := os.ReadDir(filepath.Join(dir, "backups"))
	if err != nil && !os.IsNotExist(err) {
		_, _ = fmt.Fprintf(stderr, "thewarroom: read backups: %s\n", err)
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
// kept as a pre-restore set beside the others.
func restoreBackup(set string, stdout, stderr io.Writer) int {
	dir, err := configDir()
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "thewarroom: %s\n", err)
		return 1
	}
	lock, err := acquireInstanceLock(filepath.Join(dir, dbFileName("thewarroom", isDevBuild())))
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "thewarroom: restore needs the app closed: %s\n", err)
		return 1
	}
	defer releaseInstanceLock(lock)
	backups, err := newDatabaseBackups(dir)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "thewarroom: %s\n", err)
		return 1
	}
	if err := backups.Restore(context.Background(), filepath.Join(dir, "backups", filepath.Base(set)), time.Now().UTC()); err != nil {
		_, _ = fmt.Fprintf(stderr, "thewarroom: restore %s: %s\n", set, err)
		return 1
	}
	_, _ = fmt.Fprintf(stdout, "restored %s; the replaced files are kept as a pre-restore set in %s\n", filepath.Base(set), filepath.Join(dir, "backups"))
	return 0
}
