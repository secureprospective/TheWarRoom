package main

import (
	"bytes"
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/db"
	"github.com/secureprospective/TheWarRoom/internal/offsite"
	"github.com/secureprospective/TheWarRoom/internal/serverhttp"
)

func runBackupCheck(args []string, output io.Writer) error {
	flags := flag.NewFlagSet("backup-check", flag.ContinueOnError)
	path := flags.String("config", "", "private R2 backup settings")
	if err := flags.Parse(args); err != nil {
		return fmt.Errorf("parse backup settings flags: %w", err)
	}
	if flags.NArg() != 0 {
		return fmt.Errorf("unexpected backup settings argument")
	}
	if _, err := offsite.ReadConfig(*path); err != nil {
		return fmt.Errorf("check backup settings: %w", err)
	}
	if _, err := fmt.Fprintln(output, "Backup settings are valid; no private restore key is needed on this server."); err != nil {
		return fmt.Errorf("report backup settings: %w", err)
	}
	return nil
}

func startHostedBackup(ctx context.Context, pools *db.Pools, path, temp string) (func() error, func(), error) {
	// Missing configuration never enables unprotected writes, including on development hosts.
	_, configErr := os.Stat(path)
	if path == "" || os.IsNotExist(configErr) {
		guard := func() error { return fmt.Errorf("changes are paused until off-site backups are configured") }
		pools.SetWriteGuard(guard)
		return guard, func() {}, nil
	}
	config, err := offsite.ReadConfig(path)
	if err != nil {
		return nil, nil, fmt.Errorf("configure off-site backup: %w", err)
	}
	r2, err := offsite.NewR2(config)
	if err != nil {
		return nil, nil, fmt.Errorf("configure off-site backup: %w", err)
	}
	manager := offsite.NewManager(r2, temp, func(ctx context.Context, stage string) (string, time.Time, error) {
		return offsite.Capture(ctx, pools, filepath.Dir(path), stage, config.Recipient, hostedCheckpoint)
	})
	pools.SetWriteGuard(manager.CheckWrite)
	work, cancel := context.WithCancel(ctx)
	done := make(chan struct{})
	go func() { defer close(done); manager.Run(work) }()
	return manager.CheckWrite, func() { cancel(); <-done }, nil
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

func checkBackupFiles(dir string, files []string) error {
	names := []string{"authenticator.key", "origin.pem", "origin.key", "endpoint-registry.csv"}
	if len(files) != len(names) {
		return fmt.Errorf("backup requires all server configuration files")
	}
	for i, file := range files {
		actual, err := filepath.Abs(file)
		if err != nil {
			return fmt.Errorf("resolve server settings: %w", err)
		}
		expected, err := filepath.Abs(filepath.Join(dir, names[i]))
		if err != nil {
			return fmt.Errorf("resolve backup settings: %w", err)
		}
		if actual != expected {
			return fmt.Errorf("hosted backups require authenticator.key, origin.pem, origin.key and endpoint-registry.csv beside backup.json")
		}
	}
	return nil
}

func hostedCheckpoint(ctx context.Context, snapshot string) ([]byte, error) {
	var checkpoint bytes.Buffer
	err := runCheckpoint(ctx, []string{"-db", snapshot}, &checkpoint)
	return checkpoint.Bytes(), err
}
