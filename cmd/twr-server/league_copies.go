package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/secureprospective/TheWarRoom/internal/db"
	"github.com/secureprospective/TheWarRoom/internal/store/history"
)

// leagueFileList is every league file under root, by league, in order. Only reads the folder.
func leagueFileList(root string) ([]string, error) {
	entries, err := os.ReadDir(root)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read the league files: %w", err)
	}
	var out []string
	for _, entry := range entries {
		path, err := leagueFilePath(root, entry.Name())
		if err == nil && entry.IsDir() && fileExists(path) {
			out = append(out, entry.Name())
		}
	}
	return out, nil
}

// openLeagueCopies opens every league file under root as it is, changing nothing (no migration), for
// copying and off-site capture before an upgrade.
func openLeagueCopies(ctx context.Context, root string) (*leagueFiles, error) {
	files := &leagueFiles{root: root, open: map[string]*leagueFile{}}
	leagues, err := leagueFileList(root)
	if err != nil {
		return nil, err
	}
	for _, league := range leagues {
		path, _ := leagueFilePath(root, league)
		pools, err := db.Open(ctx, path, history.LeagueSchemaVersion())
		if err != nil {
			return nil, errors.Join(fmt.Errorf("open league %s: %w", league, err), files.Close())
		}
		files.open[league] = &leagueFile{pools: pools}
	}
	return files, nil
}

// saveLeagueCopies writes each league file into save/leagues/<league>/league.db.
func saveLeagueCopies(ctx context.Context, leagues *leagueFiles, savedServer string) error {
	root := leaguesRoot(savedServer)
	for _, league := range leagues.leagues() {
		pools, _ := leagues.pools(league)
		target, err := leagueFilePath(root, league)
		if err != nil {
			return err
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o700); err != nil {
			return fmt.Errorf("make the pre-upgrade league folder: %w", err)
		}
		if err := pools.CaptureHosted(ctx, target, nil); err != nil {
			return fmt.Errorf("save league %s's pre-upgrade copy: %w", league, err)
		}
	}
	return nil
}

// putBackLeagues puts each saved league file back. A copy saved before the split holds no league
// files: then the failed upgrade's new league folder goes, and server.db (put back) holds every league.
func putBackLeagues(path, saved string) error {
	root, savedRoot := leaguesRoot(path), leaguesRoot(saved)
	if filepath.Base(root) != leaguesFolder || !strings.HasPrefix(root, filepath.Dir(path)) {
		return fmt.Errorf("refusing to touch %q: it is not this server's league folder", root)
	}
	leagues, err := leagueFileList(savedRoot)
	if err != nil {
		return err
	}
	if !fileExists(savedRoot) {
		if err := os.RemoveAll(root); err != nil {
			return fmt.Errorf("remove the failed upgrade's league files: %w", err)
		}
		return nil
	}
	for _, league := range leagues {
		final, _ := leagueFilePath(root, league)
		copyOf, _ := leagueFilePath(savedRoot, league)
		if err := os.MkdirAll(filepath.Dir(final), 0o700); err != nil {
			return fmt.Errorf("make league %s's folder: %w", league, err)
		}
		if err := putBackFile(final, copyOf); err != nil {
			return err
		}
		if err := syncPath(filepath.Dir(final)); err != nil {
			return err
		}
	}
	return nil
}
