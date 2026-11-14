//go:build !practice

package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/db"
	"github.com/secureprospective/TheWarRoom/internal/offsite"
	"github.com/secureprospective/TheWarRoom/internal/serverhttp"
)

// hostedGuards are the two write guards: league writes follow the league file's off-site backup and
// sign-in writes follow the sign-in file's.
type hostedGuards struct{ league, signIns func() error }

func startHostedBackup(ctx context.Context, files stateFiles, path, temp string) (hostedGuards, hostedBackups, func(), error) {
	// Missing configuration never enables unprotected writes, including on development hosts.
	_, configErr := os.Stat(path)
	if path == "" || os.IsNotExist(configErr) {
		guard := func() error { return fmt.Errorf("changes are paused until off-site backups are configured") }
		files.league.SetWriteGuard(guard)
		files.signIns.SetWriteGuard(guard)
		return hostedGuards{guard, guard}, hostedBackups{league: noBackups, reference: noBackups, signIns: noBackups}, func() {}, nil
	}
	managers, err := newBackupManagers(files, path, temp)
	if err != nil {
		return hostedGuards{}, hostedBackups{}, nil, err
	}
	// The reference file is written only through the league file's attached writer, so the league
	// guard admits reference writes too. Reference coverage pauses nothing (BACKUPS.md).
	files.league.SetWriteGuard(managers.league.CheckWrite)
	files.signIns.SetWriteGuard(managers.signIns.CheckWrite)
	work, cancel := context.WithCancel(ctx)
	done := make(chan struct{})
	go func() { defer close(done); offsite.RunInOrder(work, managers.inOrder()...) }()
	backups := hostedBackups{
		league:    func() (offsite.Status, bool) { return managers.league.Status(), true },
		reference: func() (offsite.Status, bool) { return managers.reference.Status(), true },
		signIns:   func() (offsite.Status, bool) { return managers.signIns.Status(), true },
	}
	return hostedGuards{managers.league.CheckWrite, managers.signIns.CheckWrite}, backups, func() { cancel(); <-done }, nil
}

// backupManagers is one off-site manager per file. A one-file server (upgrade only) has only league.
type backupManagers struct{ league, reference, signIns *offsite.Manager }

// inOrder is the capture order of a due round: league first, then reference, then sign-ins last. A
// league write commits its new player ids (reference) and its accounts (sign-in) before its own
// commit, so every id and account a captured league names is in the next reference and sign-in sets.
func (b backupManagers) inOrder() []*offsite.Manager {
	out := []*offsite.Manager{b.league}
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
	out := backupManagers{league: manager(offsite.LeagueSet, files.league, leagueSetCheckpoint)}
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
// league first, so the server as it stood before an upgrade is recoverable even if this host is lost.
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
	// A one-file server has only its league set, which still carries its sign-ins and reference data.
	for _, m := range managers.inOrder() {
		if err := m.Backup(attempt); err != nil {
			return fmt.Errorf("%s set: %w", m.Kind().File, err)
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
	handler.WriteAdmission, handler.SignInAdmission = guards.league, guards.signIns
	return stop, status, nil
}
