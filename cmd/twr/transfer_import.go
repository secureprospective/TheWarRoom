package main

import (
	"context"
	"fmt"
	"os"

	"github.com/secureprospective/TheWarRoom/internal/db"
	"github.com/secureprospective/TheWarRoom/internal/leaguetransfer"
	"github.com/secureprospective/TheWarRoom/internal/measures"
	"github.com/secureprospective/TheWarRoom/internal/store/history"
	"github.com/spf13/cobra"
)

func (e env) importCommand() *cobra.Command {
	var database, path, keyPath string
	command := &cobra.Command{Use: "league-import", Short: "Copy an encrypted league here without activating it", Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error { return e.importLeague(cmd, database, path, keyPath) }}
	command.Flags().StringVar(&database, "db", "", "existing destination league database")
	command.Flags().StringVar(&path, "file", "", "encrypted league filename")
	command.Flags().StringVar(&keyPath, "key-file", "", "private export key file")
	return command
}

func (e env) importLeague(cmd *cobra.Command, database, path, keyPath string) error {
	key, err := readExportKey(keyPath)
	if err != nil {
		return err
	}
	// Read only through this descriptor. Replacing the pathname cannot change the authenticated
	// bytes later used by the import, and no temporary plaintext archive is created.
	file, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("open encrypted league: %w", err)
	}
	defer func() { _ = file.Close() }()
	info, err := file.Stat()
	if err != nil {
		return fmt.Errorf("check encrypted league file: %w", err)
	}
	if !info.Mode().IsRegular() {
		return fmt.Errorf("choose a regular encrypted league file")
	}
	info, err = os.Lstat(database)
	if err != nil || !info.Mode().IsRegular() {
		return fmt.Errorf("choose an existing destination league database, not a link or missing file")
	}
	pools, err := db.Open(cmd.Context(), database, history.SchemaVersion())
	if err != nil {
		return fmt.Errorf("open import destination: %w", err)
	}
	defer func() { _ = pools.Close() }()
	registry, err := measures.Embedded()
	if err != nil {
		return fmt.Errorf("load league measures: %w", err)
	}
	store := history.New(pools, registry)
	receipt, err := leaguetransfer.Import(cmd.Context(), store, file, key)
	if err != nil {
		return fmt.Errorf("copy league to this computer: %w", err)
	}
	return e.reportImport(cmd.Context(), store, receipt)
}

func (e env) reportImport(ctx context.Context, store *history.Store, receipt history.ImportReceipt) error {
	status, err := store.TransferStatus(ctx, receipt.League)
	if err != nil {
		return fmt.Errorf("read imported league location: %w", err)
	}
	state := "was copied and is inactive"
	if status.Writable {
		state = "is already active here"
	}
	if _, err := fmt.Fprintf(e.out, "League %s %s. %s. Import receipt: %s / %s / %s.\n", receipt.League, state, status.Label, receipt.SourceEpoch, receipt.Content, receipt.RecordedAt.Format("2006-01-02T15:04:05.999999999Z07:00")); err != nil {
		return fmt.Errorf("report league import: %w", err)
	}
	return nil
}
