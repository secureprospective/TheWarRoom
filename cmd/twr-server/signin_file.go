package main

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"

	"github.com/secureprospective/TheWarRoom/internal/db"
	accountstore "github.com/secureprospective/TheWarRoom/internal/store/accounts"
)

// The hosted server keeps its sign-in file beside the league file (-db, server.db). Every command finds
// it through signInPath and opens it through openSignIns, which also moves the sign-in tables out of a
// one-file server on first start (fileMove).
const signInFileName = "accounts.db"

// signInMoveTable is the league file's record that its sign-in tables live in the sign-in file. It is
// written in the same transaction that removes them, so the league file says which layout it is in.
const signInMoveTable = accountstore.MoveRecordTable

func signInPath(leaguePath string) string {
	return filepath.Join(filepath.Dir(leaguePath), signInFileName)
}

// pendingSignIns is where a conversion builds the sign-in file; it becomes the sign-in file only after
// the league file has committed the move.
func pendingSignIns(leaguePath string) string { return signInMove.pending(leaguePath) }

//nolint:gochecknoglobals // fixed by the code, read only
var signInMove = fileMove{what: "sign-ins", file: signInFileName, held: "accounts", record: signInMoveTable,
	build: buildSignInMove, after: checkpointLeague}

// openSignIns opens the sign-in file for the league file at leaguePath, converting a one-file server first.
func openSignIns(ctx context.Context, league *db.Pools, leaguePath string) (*db.Pools, error) {
	if err := signInMove.settle(ctx, league, leaguePath); err != nil {
		return nil, err
	}
	pools, err := db.Open(ctx, signInPath(leaguePath), accountstore.ServerVersion())
	if err != nil {
		return nil, fmt.Errorf("open the sign-in file: %w", err)
	}
	return pools, nil
}

func buildSignInMove(ctx context.Context, league *db.Pools, pending string, held bool) (func(context.Context) error, error) {
	moved, err := buildSignIns(ctx, league, pending, held)
	if err != nil {
		return nil, err
	}
	return func(ctx context.Context) error {
		return accountstore.CommitMove(ctx, league, held, moved)
	}, nil
}

// buildSignIns writes and proves the new file, then closes it so nothing of it is left in a WAL.
func buildSignIns(ctx context.Context, source *db.Pools, path string, held bool) (moved accountstore.MoveRecord, err error) {
	moved.File = signInFileName
	pools, err := db.Open(ctx, path, accountstore.ServerVersion())
	if err != nil {
		return moved, fmt.Errorf("create the sign-in file: %w", err)
	}
	defer func() { err = errors.Join(err, pools.Close()) }()
	if err := accountstore.PrepareServerFile(ctx, pools); err != nil {
		return moved, fmt.Errorf("prepare the sign-in file: %w", err)
	}
	if held {
		if err := accountstore.CopySignIns(ctx, source, pools); err != nil {
			return moved, fmt.Errorf("copy sign-ins: %w", err)
		}
		conversionStep("copied", path)
		if err := accountstore.ProveSignIns(ctx, source, pools); err != nil {
			return moved, fmt.Errorf("prove the copy: %w", err)
		}
	}
	store := accountstore.NewServerFile(pools)
	if err := store.Initialize(ctx); err != nil {
		return moved, fmt.Errorf("verify the security history in its new file: %w", err)
	}
	if moved.Security, err = store.VerifyAudit(ctx); err != nil {
		return moved, fmt.Errorf("verify the security history in its new file: %w", err)
	}
	if held {
		if err := sameSecurityHead(ctx, source, moved.Security); err != nil {
			return moved, err
		}
	}
	if moved.Accounts, err = accountstore.CountAccounts(ctx, pools); err != nil {
		return moved, fmt.Errorf("count moved accounts: %w", err)
	}
	if _, err := pools.Write().ExecContext(ctx, `PRAGMA wal_checkpoint(TRUNCATE)`); err != nil {
		return moved, fmt.Errorf("finish the sign-in file: %w", err)
	}
	return moved, nil
}

// sameSecurityHead checks the chain arrived with its hashes intact: the new head is the old head.
func sameSecurityHead(ctx context.Context, source *db.Pools, head db.AuditHead) error {
	old, found, err := accountstore.SecurityHead(ctx, source)
	if err != nil {
		return fmt.Errorf("read the security history head: %w", err)
	}
	if found && (old.Size != head.Size || old.Hash != head.Hash) {
		return fmt.Errorf("the security history head changed in the move")
	}
	return nil // not found: the league file never adopted the chain; the new file adopted it from the rows
}
