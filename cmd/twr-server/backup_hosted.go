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

func startHostedBackup(ctx context.Context, pools *db.Pools, path, temp string) (func() error, func(), error) {
	// Missing configuration never enables unprotected writes, including on development hosts.
	_, configErr := os.Stat(path)
	if path == "" || os.IsNotExist(configErr) {
		guard := func() error { return fmt.Errorf("changes are paused until off-site backups are configured") }
		pools.SetWriteGuard(guard)
		return guard, func() {}, nil
	}
	manager, err := newBackupManager(pools, path, temp)
	if err != nil {
		return nil, nil, err
	}
	pools.SetWriteGuard(manager.CheckWrite)
	work, cancel := context.WithCancel(ctx)
	done := make(chan struct{})
	go func() { defer close(done); manager.Run(work) }()
	return manager.CheckWrite, func() { cancel(); <-done }, nil
}

func newBackupManager(pools *db.Pools, path, temp string) (*offsite.Manager, error) {
	config, err := offsite.ReadConfig(path)
	if err != nil {
		return nil, fmt.Errorf("configure off-site backup: %w", err)
	}
	r2, err := offsite.NewR2(config)
	if err != nil {
		return nil, fmt.Errorf("configure off-site backup: %w", err)
	}
	return offsite.NewManager(r2, temp, func(ctx context.Context, stage string) (string, time.Time, error) {
		return offsite.Capture(ctx, pools, filepath.Dir(path), stage, config.Recipient, hostedCheckpoint)
	}), nil
}

// upgradeOffsite uploads and reads back one matched off-site set of the stopped server, so the
// league as it stood before an upgrade is recoverable even if this host is lost.
func upgradeOffsite(ctx context.Context, pools *db.Pools, path, temp string) error {
	if filepath.Base(path) != "backup.json" {
		return fmt.Errorf("private backup settings must be named backup.json")
	}
	manager, err := newBackupManager(pools, path, temp)
	if err != nil {
		return err
	}
	attempt, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()
	//nolint:wrapcheck // The upgrade names the step that failed.
	return manager.Backup(attempt)
}

func configureHostedBackup(ctx context.Context, handler *serverhttp.Hosted, pools *db.Pools, path, temp string, files ...string) (func(), error) {
	if path != "" {
		if filepath.Base(path) != "backup.json" {
			return nil, fmt.Errorf("private backup settings must be named backup.json")
		}
		if err := checkBackupFiles(filepath.Dir(path), files); err != nil {
			return nil, err
		}
	}
	admission, stop, err := startHostedBackup(ctx, pools, path, temp)
	if err != nil {
		return nil, err
	}
	handler.WriteAdmission = admission
	return stop, nil
}
