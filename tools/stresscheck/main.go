// stresscheck is the lab-only read-only integrity and request identity reader.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/store/history"
)

func main() {
	path := flag.String("db", "", "database path (server.db also discovers split files)")
	flag.Parse()
	if err := run(context.Background(), *path, os.Stdout); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, "stresscheck:", err)
		os.Exit(1)
	}
}

func run(ctx context.Context, path string, out io.Writer) error {
	if path == "" {
		return fmt.Errorf("-db is required")
	}
	paths, err := databasePaths(path)
	if err != nil {
		return err
	}
	var report strings.Builder
	for _, file := range paths {
		if err := inspect(ctx, file, &report); err != nil {
			return err
		}
	}
	if _, err := fmt.Fprintln(out, "integrity ok"); err != nil {
		return fmt.Errorf("write integrity: %w", err)
	}
	if _, err := io.WriteString(out, report.String()); err != nil {
		return fmt.Errorf("write ledger: %w", err)
	}
	return nil
}

func inspect(ctx context.Context, path string, out io.Writer) error {
	findings, entries, err := history.InspectStressDatabase(ctx, path)
	if err != nil {
		return fmt.Errorf("inspect %s: %w", path, err)
	}
	if len(findings.Integrity) != 1 || findings.Integrity[0] != "ok" || len(findings.ForeignKeys) != 0 {
		return fmt.Errorf("database %s failed integrity: %+v", path, findings)
	}
	for i, entry := range entries {
		if entry.Stream == commissioner.Stream {
			actor, err := json.Marshal(entry.Actor)
			if err != nil {
				return fmt.Errorf("encode actor: %w", err)
			}
			command, err := json.Marshal(entry.Command)
			if err != nil {
				return fmt.Errorf("encode command: %w", err)
			}
			if _, err := fmt.Fprintf(out, "req %s %s %d\n", actor, command, entry.Revision); err != nil {
				return fmt.Errorf("write request: %w", err)
			}
		}
		if i+1 == len(entries) || entries[i+1].League != entry.League || entries[i+1].Stream != entry.Stream {
			if _, err := fmt.Fprintf(out, "head %s %s %d\n", entry.League, entry.Stream, entry.Revision); err != nil {
				return fmt.Errorf("write head: %w", err)
			}
		}
	}
	return nil
}

func databasePaths(path string) ([]string, error) {
	paths := []string{path}
	if filepath.Base(path) == "server.db" {
		split, err := filepath.Glob(filepath.Join(filepath.Dir(path), "leagues", "*", "league.db"))
		if err != nil {
			return nil, fmt.Errorf("discover league files: %w", err)
		}
		paths = append(paths, split...)
		for _, name := range []string{"accounts.db", "reference.db"} {
			file := filepath.Join(filepath.Dir(path), name)
			if _, err := os.Stat(file); err == nil {
				paths = append(paths, file)
			} else if !os.IsNotExist(err) {
				return nil, fmt.Errorf("stat split file: %w", err)
			}
		}
	}
	return paths, nil
}
