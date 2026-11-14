//go:build !practice

package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"sync"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/db"
	"github.com/secureprospective/TheWarRoom/internal/offsite"
	"github.com/secureprospective/TheWarRoom/internal/serverhttp"
)

// hostedGuards are the write guards: each league's writes follow its own file's off-site backup, so a
// league whose backup is behind pauses only itself; sign-in writes follow the sign-in file's.
type hostedGuards struct {
	league  func(league string) error
	signIns func() error
}

func startHostedBackup(ctx context.Context, files stateFiles, path, temp string) (hostedGuards, hostedBackups, func(), error) {
	// Missing configuration never enables unprotected writes, including on development hosts.
	_, configErr := os.Stat(path)
	if path == "" || os.IsNotExist(configErr) {
		guard := func() error { return fmt.Errorf("changes are paused until off-site backups are configured") }
		for _, league := range files.leagues.leagues() {
			if pools, ok := files.leagues.pools(league); ok {
				pools.SetWriteGuard(guard)
			}
		}
		files.leagues.added = func(_ string, pools *db.Pools) { pools.SetWriteGuard(guard) }
		files.signIns.SetWriteGuard(guard)
		return hostedGuards{func(string) error { return guard() }, guard}, hostedBackups{league: noBackups, reference: noBackups, signIns: noBackups}, func() {}, nil
	}
	managers, err := newBackupManagers(files, path, temp)
	if err != nil {
		return hostedGuards{}, hostedBackups{}, nil, err
	}
	// Reference writes use their own pools; reference coverage pauses nothing (BACKUPS.md).
	files.leagues.added = func(league string, pools *db.Pools) { managers.addLeague(league, pools) }
	files.signIns.SetWriteGuard(managers.signIns.CheckWrite)
	work, cancel := context.WithCancel(ctx)
	done := make(chan struct{})
	go func() { defer close(done); offsite.RunEach(work, managers.inOrder) }()
	backups := hostedBackups{
		league:    managers.leagueStatus,
		reference: func() (offsite.Status, bool) { return managers.reference.Status(), true },
		signIns:   func() (offsite.Status, bool) { return managers.signIns.Status(), true },
	}
	return hostedGuards{managers.checkLeague, managers.signIns.CheckWrite}, backups, func() { cancel(); <-done }, nil
}

// backupManagers is one off-site manager per file: one per league, the reference file's and the
// sign-in file's. Leagues created while the server runs get theirs as they are made.
type backupManagers struct {
	newLeague          func(league string, pools *db.Pools) *offsite.Manager
	mu                 *sync.Mutex
	leagues            map[string]*offsite.Manager
	reference, signIns *offsite.Manager
}

// addLeague makes league's manager and puts its file's writes behind it.
func (b backupManagers) addLeague(league string, pools *db.Pools) *offsite.Manager {
	m := b.newLeague(league, pools)
	pools.SetWriteGuard(m.CheckWrite)
	b.mu.Lock()
	b.leagues[league] = m
	b.mu.Unlock()
	return m
}

func (b backupManagers) leagueManagers() []*offsite.Manager {
	b.mu.Lock()
	defer b.mu.Unlock()
	names := make([]string, 0, len(b.leagues))
	for league := range b.leagues {
		names = append(names, league)
	}
	slices.Sort(names)
	out := make([]*offsite.Manager, len(names))
	for i, league := range names {
		out[i] = b.leagues[league]
	}
	return out
}

// checkLeague is league's write guard; a league the server does not hold has nothing to pause.
func (b backupManagers) checkLeague(league string) error {
	b.mu.Lock()
	m := b.leagues[league]
	b.mu.Unlock()
	if m == nil {
		return nil
	}
	return m.CheckWrite() //nolint:wrapcheck // the manager's words are the owner's message
}

// leagueStatus is the dashboard's league backup line: every league covered, or the one furthest behind.
func (b backupManagers) leagueStatus() (offsite.Status, bool) {
	b.mu.Lock()
	defer b.mu.Unlock()
	var worst offsite.Status
	first := true
	names := make([]string, 0, len(b.leagues))
	for league := range b.leagues {
		names = append(names, league)
	}
	slices.Sort(names)
	for _, league := range names {
		s := b.leagues[league].Status()
		if s.Failure != "" {
			s.Failure = "league " + league + ": " + s.Failure
		}
		if first || worse(s, worst) {
			worst, first = s, false
		}
	}
	if first {
		now := time.Now()
		return offsite.Status{Captured: now, Verified: now, Writable: true}, true // no leagues: nothing to cover
	}
	return worst, true
}

func worse(a, b offsite.Status) bool {
	switch {
	case a.Writable != b.Writable:
		return !a.Writable
	case (a.Failure != "") != (b.Failure != ""):
		return a.Failure != ""
	}
	return a.Verified.Before(b.Verified)
}

// inOrder is the capture order of a due round: every league first, then reference, then sign-ins
// last. A league write commits its new player ids (reference) and its accounts (sign-in) before its own
// commit, so every id and account a captured league names is in the next reference and sign-in sets.
func (b backupManagers) inOrder() []*offsite.Manager {
	out := b.leagueManagers()
	for _, m := range []*offsite.Manager{b.reference, b.signIns} {
		if m != nil {
			out = append(out, m)
		}
	}
	return out
}

// newBackupManagers builds one off-site manager per file, each with its own revision reader.
func newBackupManagers(files stateFiles, path, temp string) (backupManagers, error) {
	config, err := offsite.ReadConfig(path)
	if err != nil {
		return backupManagers{}, fmt.Errorf("configure off-site backup: %w", err)
	}
	r2, err := offsite.NewR2(config)
	if err != nil {
		return backupManagers{}, fmt.Errorf("configure off-site backup: %w", err)
	}
	manager := func(kind offsite.SetKind, pools *db.Pools, checkpoint func(context.Context, string) ([]byte, error)) *offsite.Manager {
		return offsite.NewKindManager(kind, r2, temp, func(ctx context.Context, stage string) (string, time.Time, error) {
			return offsite.CaptureSet(ctx, kind, pools, filepath.Dir(path), stage, config.Recipient, checkpoint)
		}, pools.Revision)
	}
	out := backupManagers{mu: &sync.Mutex{}, leagues: map[string]*offsite.Manager{},
		newLeague: func(league string, pools *db.Pools) *offsite.Manager {
			return manager(offsite.LeagueFileSet(league), pools, leagueSetCheckpoint)
		}}
	for _, league := range files.leagues.leagues() {
		if pools, ok := files.leagues.pools(league); ok {
			out.addLeague(league, pools)
		}
	}
	if files.signIns != nil {
		out.signIns = manager(offsite.SignInSet, files.signIns, signInSetCheckpoint)
	}
	if files.reference != nil {
		out.reference = referenceManager(r2, files.reference, filepath.Dir(path), temp, config.Recipient)
	}
	return out, nil
}

// referenceManager backs up the reference file without its raw captures (captureReference).
func referenceManager(r2 *offsite.R2, pools *db.Pools, configDir, temp, recipient string) *offsite.Manager {
	evidence := offsite.NewEvidence(r2, recipient)
	return offsite.NewKindManager(offsite.ReferenceSet, r2, temp, func(ctx context.Context, stage string) (string, time.Time, error) {
		return captureReference(ctx, evidence.Put, pools, configDir, stage, recipient)
	}, pools.Revision)
}

// upgradeOffsite uploads and reads back one matched off-site set of each file of the stopped server,
// every league first, so the server as it stood before an upgrade is recoverable even if this host is lost.
func upgradeOffsite(ctx context.Context, files stateFiles, path, temp string) error {
	if filepath.Base(path) != "backup.json" {
		return fmt.Errorf("private backup settings must be named backup.json")
	}
	managers, err := newBackupManagers(files, path, temp)
	if err != nil {
		return err
	}
	attempt, cancel := context.WithTimeout(ctx, 4*time.Minute)
	defer cancel()
	for _, m := range managers.inOrder() {
		if err := m.Backup(attempt); err != nil {
			return fmt.Errorf("%s set: %w", m.Kind().Prefix, err)
		}
	}
	return nil
}

func configureHostedBackup(ctx context.Context, handler *serverhttp.Hosted, files stateFiles, path, temp string, extra ...string) (func(), hostedBackups, error) {
	if path != "" {
		if filepath.Base(path) != "backup.json" {
			return nil, hostedBackups{}, fmt.Errorf("private backup settings must be named backup.json")
		}
		if err := checkBackupFiles(filepath.Dir(path), extra); err != nil {
			return nil, hostedBackups{}, err
		}
	}
	guards, status, stop, err := startHostedBackup(ctx, files, path, temp)
	if err != nil {
		return nil, hostedBackups{}, err
	}
	handler.LeagueAdmission, handler.SignInAdmission = guards.league, guards.signIns
	return stop, status, nil
}
