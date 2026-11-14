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
		return hostedGuards{guard, guard}, hostedBackups{noBackups, noBackups}, func() {}, nil
	}
	league, signIns, err := newBackupManagers(files, path, temp)
	if err != nil {
		return hostedGuards{}, hostedBackups{}, nil, err
	}
	files.league.SetWriteGuard(league.CheckWrite)
	files.signIns.SetWriteGuard(signIns.CheckWrite)
	work, cancel := context.WithCancel(ctx)
	done := make(chan struct{})
	// League set first, sign-in set last in every due round: accounts commit before seats, so every
	// principal a captured league seats is already in the next sign-in set.
	go func() { defer close(done); offsite.RunInOrder(work, league, signIns) }()
	backups := hostedBackups{
		league:  func() (offsite.Status, bool) { return league.Status(), true },
		signIns: func() (offsite.Status, bool) { return signIns.Status(), true },
	}
	return hostedGuards{league.CheckWrite, signIns.CheckWrite}, backups, func() { cancel(); <-done }, nil
}

// newBackupManagers builds one off-site manager per file, each with its own revision reader.
func newBackupManagers(files stateFiles, path, temp string) (*offsite.Manager, *offsite.Manager, error) {
	config, err := offsite.ReadConfig(path)
	if err != nil {
		return nil, nil, fmt.Errorf("configure off-site backup: %w", err)
	}
	r2, err := offsite.NewR2(config)
	if err != nil {
		return nil, nil, fmt.Errorf("configure off-site backup: %w", err)
	}
	manager := func(kind offsite.SetKind, pools *db.Pools, checkpoint func(context.Context, string) ([]byte, error)) *offsite.Manager {
		return offsite.NewKindManager(kind, r2, temp, func(ctx context.Context, stage string) (string, time.Time, error) {
			return offsite.CaptureSet(ctx, kind, pools, filepath.Dir(path), stage, config.Recipient, checkpoint)
		}, pools.Revision)
	}
	if files.signIns == nil {
		return manager(offsite.LeagueSet, files.league, leagueSetCheckpoint), nil, nil
	}
	return manager(offsite.LeagueSet, files.league, leagueSetCheckpoint), manager(offsite.SignInSet, files.signIns, signInSetCheckpoint), nil
}

// upgradeOffsite uploads and reads back one matched off-site set of each file of the stopped server,
// league first, so the server as it stood before an upgrade is recoverable even if this host is lost.
func upgradeOffsite(ctx context.Context, files stateFiles, path, temp string) error {
	if filepath.Base(path) != "backup.json" {
		return fmt.Errorf("private backup settings must be named backup.json")
	}
	league, signIns, err := newBackupManagers(files, path, temp)
	if err != nil {
		return err
	}
	attempt, cancel := context.WithTimeout(ctx, 4*time.Minute)
	defer cancel()
	if err := league.Backup(attempt); err != nil {
		return fmt.Errorf("league set: %w", err)
	}
	if signIns == nil {
		return nil // a one-file server: its league set still carries its sign-ins
	}
	if err := signIns.Backup(attempt); err != nil {
		return fmt.Errorf("sign-in set: %w", err)
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
