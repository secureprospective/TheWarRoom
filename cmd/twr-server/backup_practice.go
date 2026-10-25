//go:build practice

package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/secureprospective/TheWarRoom/internal/db"
	"github.com/secureprospective/TheWarRoom/internal/serverhttp"
)

// This file is absent from release binaries. The separately built desktop click-test
// server accepts only the launcher's private disposable layout, never R2 deployment settings.
func configureHostedBackup(_ context.Context, _ *serverhttp.Hosted, _ *db.Pools, path, temp string, files ...string) (func(), error) {
	if path != "" {
		return nil, fmt.Errorf("practice builds cannot run R2 deployment configuration; use the release server")
	}
	if err := practiceFolders(temp); err != nil {
		return nil, err
	}
	if err := practiceFiles(temp, files); err != nil {
		return nil, err
	}
	log.Printf("DISPOSABLE PRACTICE BUILD: no off-site protection; never use for real league state")
	return func() {}, nil
}

func practiceFolders(dir string) error {
	for _, name := range []string{".", "home", "config", "data", "cache", "runtime"} {
		path := filepath.Join(dir, name)
		info, err := os.Lstat(path)
		if err != nil {
			return fmt.Errorf("practice needs a complete disposable directory: %w", err)
		}
		if !info.IsDir() || info.Mode().Perm() != 0700 {
			return fmt.Errorf("practice needs private real sandbox directories, not deployment paths")
		}
	}
	return nil
}

func practiceFiles(dir string, files []string) error {
	names := []string{"authenticator.key", "server.pem", "server.key", "registry.csv"}
	if len(files) != len(names) {
		return fmt.Errorf("practice needs all disposable runtime files")
	}
	for i, path := range files {
		actual, err := filepath.Abs(path)
		if err != nil {
			return fmt.Errorf("resolve practice setting: %w", err)
		}
		expected, err := filepath.Abs(filepath.Join(dir, names[i]))
		if err != nil {
			return fmt.Errorf("resolve practice directory: %w", err)
		}
		if actual != expected {
			return fmt.Errorf("practice settings must stay inside their disposable directory")
		}
		info, err := os.Lstat(path)
		if err != nil {
			return fmt.Errorf("inspect practice setting: %w", err)
		}
		if !info.Mode().IsRegular() || (names[i] != "server.pem" && info.Mode().Perm()&0077 != 0) {
			return fmt.Errorf("practice settings must be private regular files")
		}
	}
	return nil
}
