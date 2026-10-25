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

func startHostedBackup(ctx context.Context, pools *db.Pools, path, temp string) (func() error, backupStatus, func(), error) {
	// Missing configuration never enables unprotected writes, including on development hosts.
	_, configErr := os.Stat(path)
	if path == "" || os.IsNotExist(configErr) {
		guard := func() error { return fmt.Errorf("changes are paused until off-site backups are configured") }
		pools.SetWriteGuard(guard)
		return guard, noBackups, func() {}, nil
	}
	config, err := offsite.ReadConfig(path)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("configure off-site backup: %w", err)
	}
	r2, err := offsite.NewR2(config)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("configure off-site backup: %w", err)
	}
	manager := offsite.NewManager(r2, temp, func(ctx context.Context, stage string) (string, time.Time, error) {
		return offsite.Capture(ctx, pools, filepath.Dir(path), stage, config.Recipient, hostedCheckpoint)
	})
	pools.SetWriteGuard(manager.CheckWrite)
	work, cancel := context.WithCancel(ctx)
	done := make(chan struct{})
	go func() { defer close(done); manager.Run(work) }()
	return manager.CheckWrite, func() (offsite.Status, bool) { return manager.Status(), true }, func() { cancel(); <-done }, nil
}

func configureHostedBackup(ctx context.Context, handler *serverhttp.Hosted, pools *db.Pools, path, temp string, files ...string) (func(), backupStatus, error) {
	if path != "" {
		if filepath.Base(path) != "backup.json" {
			return nil, nil, fmt.Errorf("private backup settings must be named backup.json")
		}
		if err := checkBackupFiles(filepath.Dir(path), files); err != nil {
			return nil, nil, err
		}
	}
	admission, status, stop, err := startHostedBackup(ctx, pools, path, temp)
	if err != nil {
		return nil, nil, err
	}
	handler.WriteAdmission = admission
	return stop, status, nil
}
