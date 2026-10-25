package main

import (
	"bytes"
	"context"
	"flag"
	"fmt"
	"io"
	"path/filepath"

	"github.com/secureprospective/TheWarRoom/internal/offsite"
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
