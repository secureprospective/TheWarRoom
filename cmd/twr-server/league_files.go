package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"

	"github.com/secureprospective/TheWarRoom/internal/db"
	"github.com/secureprospective/TheWarRoom/internal/store/history"
)

// Every league on a hosted server has its own file, leagues/<league>/league.db beside server.db, with
// its own writer, audit chain, off-site backup set and write guard (Christopher, 2026-11-14: "Each league
// needs its own dedicated, segregated database that doesn't mix from one league to the next league").
// Each is opened with the shared football reference file attached read-only. server.db itself keeps
// only the record that its leagues were split out (leagueSplit).
const (
	leaguesFolder  = "leagues"
	leagueFileName = "league.db"
	// leagueReaders caps each league file's readers: a server holds many small files, not one big one.
	leagueReaders = 4
)

var errNoLeague = errors.New("league access unavailable")

// validLeagueID is the league ids a file may be named for: the hosted request check's shape, limited to
// characters that are safe as one folder name.
func validLeagueID(id string) bool {
	if !strings.HasPrefix(id, "native.") || len(id) <= len("native.") || len(id) > 128 {
		return false
	}
	return !strings.ContainsFunc(id, func(r rune) bool {
		return (r < 'a' || r > 'z') && (r < 'A' || r > 'Z') && (r < '0' || r > '9') && r != '.' && r != '_' && r != '-'
	})
}

func leaguesRoot(serverPath string) string {
	return filepath.Join(filepath.Dir(serverPath), leaguesFolder)
}

func leagueFilePath(root, league string) (string, error) {
	if !validLeagueID(league) {
		return "", fmt.Errorf("%q cannot be a league on this server", league)
	}
	return filepath.Join(root, league, leagueFileName), nil
}

// leagueFiles is the server's open league files. All are opened at start and stay open: a server holds
// a few dozen leagues at most, each capped at leagueReaders readers.
type leagueFiles struct {
	root      string
	reference *db.Pools
	// family gives every league's store the same registry, clock and commit signal.
	family *history.Store
	// shared is the closed shared chain's head, named in every league file created here.
	shared db.AuditHead
	mu     sync.RWMutex
	open   map[string]*leagueFile
	// added runs for a league created while the server runs (its backup manager and write guard).
	added func(league string, pools *db.Pools)
}

type leagueFile struct {
	pools *db.Pools
	store *history.Store
	// moving is set while the league is on its way to another server (league_move.go): it reads, but
	// does not change.
	moving bool
}

// openLeagueFiles opens every league file under root. A folder that is not a valid league, or a file
// that names another league, stops the start: nothing is guessed.
func openLeagueFiles(ctx context.Context, root string, reference *db.Pools, family *history.Store, shared db.AuditHead) (*leagueFiles, error) {
	files := &leagueFiles{root: root, reference: reference, family: family, shared: shared, open: map[string]*leagueFile{}}
	entries, err := os.ReadDir(root)
	if errors.Is(err, os.ErrNotExist) {
		return files, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read the league files: %w", err)
	}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		path, err := leagueFilePath(root, entry.Name())
		if err != nil {
			return nil, errors.Join(fmt.Errorf("the leagues folder holds %s, which is not a league; nothing was changed", entry.Name()), files.Close())
		}
		if !fileExists(path) {
			continue // an archived or half-made league folder holds no league.db
		}
		file, err := files.openFile(ctx, entry.Name(), path, false)
		if err != nil {
			return nil, errors.Join(err, files.Close())
		}
		files.open[entry.Name()] = file
	}
	return files, nil
}

func (f *leagueFiles) openFile(ctx context.Context, league, path string, create bool) (*leagueFile, error) {
	pools, err := db.OpenWithReference(ctx, path, history.LeagueSchemaVersion(), f.reference)
	if err != nil {
		return nil, fmt.Errorf("open league %s: %w", league, err)
	}
	pools.Read().SetMaxOpenConns(leagueReaders)
	pools.Read().SetMaxIdleConns(leagueReaders)
	if create {
		err = history.CreateLeagueFile(ctx, pools, history.LeagueFile{League: league, Shared: f.shared})
	} else {
		err = history.MigrateLeague(ctx, pools)
	}
	if err == nil {
		err = checkLeagueFile(ctx, pools, league)
	}
	store := f.family.Over(pools)
	if err == nil {
		_, err = store.VerifyAudit(ctx)
	}
	if err != nil {
		return nil, errors.Join(fmt.Errorf("league %s: %w", league, err), pools.Close())
	}
	return &leagueFile{pools: pools, store: store, moving: fileExists(filepath.Join(filepath.Dir(path), movingFile))}, nil
}

func checkLeagueFile(ctx context.Context, pools *db.Pools, league string) error {
	file, err := history.ReadLeagueFile(ctx, pools)
	if err != nil {
		return err //nolint:wrapcheck // the caller names the league
	}
	if file.League != league {
		return fmt.Errorf("its file belongs to league %s; nothing was changed", file.League)
	}
	return nil
}

// create makes a new league's file. The league must not be on this server yet.
func (f *leagueFiles) create(ctx context.Context, league string) (*history.Store, error) {
	path, err := leagueFilePath(f.root, league)
	if err != nil {
		return nil, err
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.open[league] != nil || fileExists(path) {
		return nil, fmt.Errorf("league %s is already on this server", league)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return nil, fmt.Errorf("make league %s's folder: %w", league, err)
	}
	file, err := f.openFile(ctx, league, path, true)
	if err != nil {
		return nil, errors.Join(err, removeDatabase(path))
	}
	if err := syncPath(filepath.Dir(path)); err != nil {
		return nil, errors.Join(err, file.pools.Close())
	}
	f.open[league] = file
	if f.added != nil {
		f.added(league, file.pools)
	}
	return file.store, nil
}

// adopt opens a league file that was put in place under root (a restored league) and serves it.
func (f *leagueFiles) adopt(ctx context.Context, league string) (*history.Store, error) {
	path, err := leagueFilePath(f.root, league)
	if err != nil {
		return nil, err
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.open[league] != nil {
		return nil, fmt.Errorf("league %s is already open", league)
	}
	file, err := f.openFile(ctx, league, path, false)
	if err != nil {
		return nil, err
	}
	f.open[league] = file
	if f.added != nil {
		f.added(league, file.pools)
	}
	return file.store, nil
}

// store is league's own store; ok is false for a league this server does not hold.
func (f *leagueFiles) store(league string) (*history.Store, bool) {
	f.mu.RLock()
	defer f.mu.RUnlock()
	file := f.open[league]
	if file == nil {
		return nil, false
	}
	return file.store, true
}

func (f *leagueFiles) pools(league string) (*db.Pools, bool) {
	f.mu.RLock()
	defer f.mu.RUnlock()
	file := f.open[league]
	if file == nil {
		return nil, false
	}
	return file.pools, true
}

// leagues is every league this server holds, in order.
func (f *leagueFiles) leagues() []string {
	f.mu.RLock()
	defer f.mu.RUnlock()
	out := make([]string, 0, len(f.open))
	for league := range f.open {
		out = append(out, league)
	}
	slices.Sort(out)
	return out
}

func (f *leagueFiles) stores() []*history.Store {
	var out []*history.Store
	for _, league := range f.leagues() {
		if store, ok := f.store(league); ok {
			out = append(out, store)
		}
	}
	return out
}

// fence reserves every league file's writer, in league order, so no credential-withdrawing sign-in
// write commits between a league write's credential recheck and its commit.
func (f *leagueFiles) fence(ctx context.Context) (func(), error) {
	var conns []interface{ Close() error }
	release := func() {
		for i := len(conns) - 1; i >= 0; i-- {
			_ = conns[i].Close()
		}
	}
	for _, league := range f.leagues() {
		pools, ok := f.pools(league)
		if !ok {
			continue
		}
		conn, err := pools.Write().Conn(ctx)
		if err != nil {
			release()
			return nil, fmt.Errorf("reserve league %s's writer: %w", league, err)
		}
		conns = append(conns, conn)
	}
	return release, nil
}

// remove closes league's file and forgets it (the file itself is moved by the caller).
func (f *leagueFiles) remove(league string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	file := f.open[league]
	if file == nil {
		return fmt.Errorf("league %s is not on this server", league)
	}
	delete(f.open, league)
	return file.pools.Close() //nolint:wrapcheck // db names the file
}

// discard closes and deletes a league file nothing names yet (a load that failed); "" is a no-op.
func (f *leagueFiles) discard(league string) error {
	if league == "" {
		return nil
	}
	path, err := leagueFilePath(f.root, league)
	if err != nil {
		return err
	}
	return errors.Join(f.remove(league), removeDatabase(path), os.Remove(filepath.Dir(path)))
}

func (f *leagueFiles) Close() error {
	f.mu.Lock()
	defer f.mu.Unlock()
	var errs []error
	for league, file := range f.open {
		if err := file.pools.Close(); err != nil {
			errs = append(errs, fmt.Errorf("close league %s: %w", league, err))
		}
	}
	f.open = map[string]*leagueFile{}
	return errors.Join(errs...)
}
