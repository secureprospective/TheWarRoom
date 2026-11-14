package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// A deleted league is not gone at once (Christopher, 2026-11-14): its folder moves to archive/ beside
// server.db, named for the league and the moment, and the server itself deletes it once 30 days have
// passed. Nothing else on the server changes: other leagues keep their files untouched.
const (
	archiveFolder = "archive"
	archiveDays   = 30
	archiveStamp  = "20060102T150405Z"
)

func archiveRoot(root string) string { return filepath.Join(filepath.Dir(root), archiveFolder) }

func archiveName(league string, at time.Time) string {
	return league + "@" + at.UTC().Format(archiveStamp)
}

// archivedAt reads an archive entry's league and moment; ok is false for anything else.
func archivedAt(name string) (string, time.Time, bool) {
	league, stamp, found := strings.Cut(name, "@")
	if !found || !validLeagueID(league) {
		return "", time.Time{}, false
	}
	at, err := time.Parse(archiveStamp, stamp)
	return league, at, err == nil
}

func leagueArchived(root, league string) bool {
	entries, err := os.ReadDir(archiveRoot(root))
	if err != nil {
		return false
	}
	for _, entry := range entries {
		if name, _, ok := archivedAt(entry.Name()); ok && name == league {
			return true
		}
	}
	return false
}

// archiveLeague moves league's folder into the archive. The caller has closed its file.
func archiveLeague(root, league string, now time.Time) (string, error) {
	final, err := leagueFilePath(root, league)
	if err != nil {
		return "", err
	}
	archive := archiveRoot(root)
	if err := os.MkdirAll(archive, 0o700); err != nil {
		return "", fmt.Errorf("make the archive folder: %w", err)
	}
	target := filepath.Join(archive, archiveName(league, now))
	if err := os.Rename(filepath.Dir(final), target); err != nil {
		return "", fmt.Errorf("archive league %s: %w", league, err)
	}
	if err := errors.Join(syncPath(archive), syncPath(root)); err != nil {
		return "", err
	}
	return target, nil
}

// purgeArchive deletes every archived league older than 30 days, and nothing else.
func purgeArchive(root string, now time.Time) ([]string, error) {
	archive := archiveRoot(root)
	entries, err := os.ReadDir(archive)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read the archive: %w", err)
	}
	var gone []string
	for _, entry := range entries {
		league, at, ok := archivedAt(entry.Name())
		if !ok || !entry.IsDir() || now.Sub(at) < archiveDays*24*time.Hour {
			continue
		}
		if err := os.RemoveAll(filepath.Join(archive, entry.Name())); err != nil {
			return gone, fmt.Errorf("delete archived league %s: %w", league, err)
		}
		gone = append(gone, league)
	}
	if len(gone) > 0 {
		return gone, syncPath(archive)
	}
	return gone, nil
}

// startArchivePurge purges once at start and then daily while the server runs.
func startArchivePurge(ctx context.Context, root string, now func() time.Time, report func(string)) func() {
	purge := func() {
		gone, err := purgeArchive(root, now())
		if err != nil {
			report(fmt.Sprintf("archived leagues: %v", err))
		}
		for _, league := range gone {
			report(fmt.Sprintf("deleted archived league %s: its 30 days had passed", league))
		}
	}
	purge()
	ctx, cancel := context.WithCancel(ctx)
	done := make(chan struct{})
	go func() {
		defer close(done)
		ticker := time.NewTicker(24 * time.Hour)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				purge()
			}
		}
	}()
	return func() { cancel(); <-done }
}

// runDeleteLeague is the console command that deletes one league from a stopped server: its file
// moves to the archive for 30 days and its directory row and join link go. Other leagues are untouched.
func runDeleteLeague(ctx context.Context, args []string, output io.Writer) error {
	flags := flag.NewFlagSet("twr-server delete-league", flag.ContinueOnError)
	path := flags.String("db", "", "existing server database (server stopped)")
	league := flags.String("league", "", "league ID (native.…)")
	if err := flags.Parse(args); err != nil {
		return fmt.Errorf("parse delete-league flags: %w", err)
	}
	if flags.NArg() != 0 || *league == "" || *path == "" {
		return fmt.Errorf("delete-league needs -db and -league, with the server stopped")
	}
	if err := checkHostedDatabase(*path, 0); err != nil {
		return err
	}
	files, err := openStateFiles(ctx, *path)
	if err != nil {
		return err
	}
	defer func() { _ = files.Close() }()
	if _, ok := files.leagues.store(*league); !ok {
		return fmt.Errorf("league %s is not on this server; nothing was changed", *league)
	}
	if err := files.leagues.remove(*league); err != nil {
		return err
	}
	now := time.Now()
	target, err := archiveLeague(files.leagues.root, *league, now)
	if err != nil {
		return err
	}
	if err := signInStore(files).ForgetLeague(ctx, *league); err != nil {
		return fmt.Errorf("league %s is archived at %s, but its directory row could not be removed: %w", *league, target, err)
	}
	until := now.Add(archiveDays * 24 * time.Hour)
	_, err = fmt.Fprintf(output, "League %s is deleted from this server. Its file is kept in %s until %s, then the server deletes it. Other leagues are unchanged.\n",
		*league, target, until.Format("2006-01-02"))
	if err != nil {
		return fmt.Errorf("report delete (the league is already archived): %w", err)
	}
	return nil
}
