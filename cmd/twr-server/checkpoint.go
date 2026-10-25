package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/db"
	"github.com/secureprospective/TheWarRoom/internal/measures"
	accountstore "github.com/secureprospective/TheWarRoom/internal/store/accounts"
	"github.com/secureprospective/TheWarRoom/internal/store/history"
)

// rawCheckpoint is the league history checkpoint kept away from the server: with each off-site
// backup and on the commissioner's desktop. A later rewrite of history and its local hash chain
// together no longer matches it.
type rawCheckpoint struct {
	Version  int          `json:"version"`
	Captured time.Time    `json:"captured"`
	Ledger   db.AuditHead `json:"ledger"`
	Security db.AuditHead `json:"security"`
}

var errCheckpointMismatch = errors.New("league history on this server no longer matches the saved checkpoint")

// runCheckpoint prints a verified checkpoint, or with -check compares the server against one.
// It only reads: no migration, no initialization, no listener.
func runCheckpoint(ctx context.Context, args []string, output io.Writer) error {
	flags := flag.NewFlagSet("twr-server checkpoint", flag.ContinueOnError)
	path := flags.String("db", "", "existing server database")
	check := flags.String("check", "", "saved checkpoint file to compare against")
	if err := flags.Parse(args); err != nil {
		return fmt.Errorf("parse checkpoint flags: %w", err)
	}
	if info, err := os.Stat(*path); err != nil || !info.Mode().IsRegular() || flags.NArg() != 0 {
		return fmt.Errorf("checkpoint needs an existing server database (-db)")
	}
	pools, err := db.Open(ctx, *path, history.SchemaVersion())
	if err != nil {
		return fmt.Errorf("open server database: %w", err)
	}
	defer func() { _ = pools.Close() }()
	registry, err := measures.Embedded()
	if err != nil {
		return fmt.Errorf("load server measures: %w", err)
	}
	ledger, security := history.New(pools, registry), accountstore.New(pools)
	if *check != "" {
		return checkCheckpoint(ctx, *check, ledger, security, output)
	}
	current := rawCheckpoint{Version: 1, Captured: time.Now().UTC()}
	if current.Ledger, err = ledger.VerifyAudit(ctx); err != nil {
		return fmt.Errorf("verify league history: %w", err)
	}
	if current.Security, err = security.VerifyAudit(ctx); err != nil {
		return fmt.Errorf("verify security history: %w", err)
	}
	if err := json.NewEncoder(output).Encode(current); err != nil {
		return fmt.Errorf("write checkpoint: %w", err)
	}
	return nil
}

func checkCheckpoint(ctx context.Context, file string, ledger *history.Store, security *accountstore.Store, output io.Writer) error {
	body, err := os.ReadFile(file)
	if err != nil {
		return fmt.Errorf("read saved checkpoint: %w", err)
	}
	var saved rawCheckpoint
	if err := json.Unmarshal(body, &saved); err != nil || saved.Version != 1 {
		return fmt.Errorf("the saved checkpoint file is not one this version can read")
	}
	if err := ledger.VerifyAuditAnchor(ctx, saved.Ledger); err != nil {
		return fmt.Errorf("%w (league history, saved %s): %w", errCheckpointMismatch, saved.Captured.Format(time.DateOnly), err)
	}
	if err := security.VerifyAuditAnchor(ctx, saved.Security); err != nil {
		return fmt.Errorf("%w (sign-in history, saved %s): %w", errCheckpointMismatch, saved.Captured.Format(time.DateOnly), err)
	}
	if _, err := fmt.Fprintf(output, "League history matches the checkpoint saved %s.\n", saved.Captured.Format(time.DateOnly)); err != nil {
		return fmt.Errorf("report checkpoint: %w", err)
	}
	return nil
}
