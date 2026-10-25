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

type moveOptions struct {
	source, target, file, key, principal string
	confirm                              bool
}

func (e env) moveCommand() *cobra.Command {
	var o moveOptions
	cmd := &cobra.Command{Use: "league-move seal|activate|abort", Short: "Confirm one step of moving a league between existing databases", Args: cobra.ExactArgs(1), RunE: func(cmd *cobra.Command, args []string) error { return e.moveLeague(cmd, args[0], o) }}
	cmd.Flags().StringVar(&o.source, "source-db", "", "existing current authority database")
	cmd.Flags().StringVar(&o.target, "target-db", "", "existing destination database")
	cmd.Flags().StringVar(&o.file, "file", "", "encrypted league export")
	cmd.Flags().StringVar(&o.key, "key-file", "", "private export key file")
	cmd.Flags().StringVar(&o.principal, "commissioner", "", "current league commissioner principal; local filesystem custody required")
	cmd.Flags().BoolVar(&o.confirm, "confirm", false, "explicitly confirm this action as commissioner")
	return cmd
}
func (e env) moveLeague(cmd *cobra.Command, action string, o moveOptions) error {
	key, err := readExportKey(o.key)
	if err != nil {
		return err
	}
	file, err := os.Open(o.file)
	if err != nil {
		return fmt.Errorf("open league move file: %w", err)
	}
	defer func() { _ = file.Close() }()
	info, err := file.Stat()
	if err != nil {
		return fmt.Errorf("check move file: %w", err)
	}
	if !info.Mode().IsRegular() {
		return fmt.Errorf("choose a regular encrypted league file")
	}
	raw, err := leaguetransfer.Decrypt(cmd.Context(), file, key)
	if err != nil {
		return fmt.Errorf("check move export: %w", err)
	}
	source, closeSource, err := openMoveStore(cmd.Context(), o.source)
	if err != nil {
		return err
	}
	defer closeSource()
	target, closeTarget, err := openMoveStore(cmd.Context(), o.target)
	if err != nil {
		return err
	}
	defer closeTarget()
	if err := leaguetransfer.Move(cmd.Context(), source, target, raw, action, o.principal, o.confirm); err != nil {
		return fmt.Errorf("move league: %w", err)
	}
	status, err := source.TransferStatus(cmd.Context(), raw.League)
	if err != nil {
		return fmt.Errorf("read league location: %w", err)
	}
	if _, err := fmt.Fprintln(e.out, status.Label); err != nil {
		return fmt.Errorf("report league location: %w", err)
	}
	return nil
}
func openMoveStore(ctx context.Context, path string) (*history.Store, func(), error) {
	info, err := os.Lstat(path)
	if err != nil || !info.Mode().IsRegular() {
		return nil, nil, fmt.Errorf("choose an existing league database, not a link or missing file")
	}
	pools, err := db.Open(ctx, path, history.SchemaVersion())
	if err != nil {
		return nil, nil, fmt.Errorf("open league move database: %w", err)
	}
	closeStore := func() { _ = pools.Close() }
	registry, err := measures.Embedded()
	if err != nil {
		closeStore()
		return nil, nil, fmt.Errorf("load league measures: %w", err)
	}
	return history.New(pools, registry), closeStore, nil
}
