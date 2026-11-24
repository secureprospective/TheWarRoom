package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/secureprospective/TheWarRoom/internal/db"
	"github.com/secureprospective/TheWarRoom/internal/measures"
	"github.com/secureprospective/TheWarRoom/internal/store/history"
)

// runExportModelRun writes one model run (default: the newest) of a desktop history.db to a file.
// The source is opened read-only.
func runExportModelRun(ctx context.Context, args []string, output io.Writer) (retErr error) {
	flags := flag.NewFlagSet("twr-server export-model-run", flag.ContinueOnError)
	path := flags.String("db", "", "history.db to read (opened read-only)")
	out := flags.String("out", "", "file to write")
	run := flags.Int64("run", 0, "model run id (default: newest)")
	if err := flags.Parse(args); err != nil {
		return fmt.Errorf("parse export-model-run flags: %w", err)
	}
	if flags.NArg() != 0 || *path == "" || *out == "" {
		return errors.New("export-model-run needs -db and -out")
	}
	pools, err := db.OpenReadOnly(ctx, *path, history.SchemaVersion())
	if err != nil {
		return fmt.Errorf("open history database read-only: %w", err)
	}
	defer func() { retErr = errors.Join(retErr, pools.Close()) }()
	registry, err := measures.Embedded()
	if err != nil {
		return fmt.Errorf("load measures: %w", err)
	}
	file, err := history.New(pools, registry).ExportModelRun(ctx, *run)
	if err != nil {
		return fmt.Errorf("read model run: %w", err)
	}
	body, err := json.Marshal(file)
	if err != nil {
		return fmt.Errorf("encode model run: %w", err)
	}
	if err := os.WriteFile(*out, body, 0o600); err != nil {
		return fmt.Errorf("write %s: %w", *out, err)
	}
	if _, err := fmt.Fprintf(output, "Exported model run %d (season %d, %d scores) to %s.\n", file.RunID, file.Season, len(file.Scores), *out); err != nil {
		return fmt.Errorf("report export: %w", err)
	}
	return nil
}

// runImportModelScores loads an exported model run into the server's reference file, which is what
// the computer draft picks rank by. The server must be stopped.
func runImportModelScores(ctx context.Context, args []string, output io.Writer) (retErr error) {
	flags := flag.NewFlagSet("twr-server import-model-scores", flag.ContinueOnError)
	path := flags.String("db", "", "existing server database (server stopped)")
	file := flags.String("file", "", "file written by export-model-run")
	if err := flags.Parse(args); err != nil {
		return fmt.Errorf("parse import-model-scores flags: %w", err)
	}
	if flags.NArg() != 0 || *path == "" || *file == "" {
		return errors.New("import-model-scores needs -db and -file; stop the server first")
	}
	lock, err := lockStoppedServer(*path)
	if err != nil {
		return err
	}
	defer func() { _ = lock.Close() }()
	if err := checkHostedDatabase(*path, 0); err != nil {
		return err
	}
	body, err := os.ReadFile(*file)
	if err != nil {
		return fmt.Errorf("read %s: %w", *file, err)
	}
	var run history.ModelRunFile
	if err := json.Unmarshal(body, &run); err != nil {
		return fmt.Errorf("decode %s: %w", *file, err)
	}
	files, err := openStateFiles(ctx, *path)
	if err != nil {
		return err
	}
	defer func() {
		if err := files.Close(); err != nil {
			retErr = errors.Join(retErr, fmt.Errorf("close server state: %w", err))
		}
	}()
	registry, err := measures.Embedded()
	if err != nil {
		return fmt.Errorf("load measures: %w", err)
	}
	stored, held, err := history.New(files.shared, registry).ImportModelRun(ctx, run)
	if err != nil {
		return fmt.Errorf("import model run: %w", err)
	}
	verb := "Imported"
	if held {
		verb = "Model run already held; no changes made:"
	}
	if _, err := fmt.Fprintf(output, "%s run %d, season %d, %d scores.\n", verb, stored.ID, stored.Season, len(run.Scores)); err != nil {
		return fmt.Errorf("report import: %w", err)
	}
	return nil
}
