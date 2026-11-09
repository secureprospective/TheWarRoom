package composition

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	commissionerapp "github.com/secureprospective/TheWarRoom/internal/app/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/db"
	"github.com/secureprospective/TheWarRoom/internal/store/history"
)

func identityFile(dir string) string { return filepath.Join(dir, "local-identity") }

func persistedIdentity(dir string) (string, error) {
	if dir == "" {
		return "", fmt.Errorf("local identity is unavailable")
	}
	body, err := os.ReadFile(identityFile(dir))
	if err == nil {
		principal := string(body)
		if !strings.HasPrefix(principal, "local.") || len(principal) != 70 {
			return "", fmt.Errorf("invalid local identity file")
		}
		if _, err := hex.DecodeString(principal[6:]); err != nil || strings.ToLower(principal) != principal {
			return "", fmt.Errorf("invalid local identity file")
		}
		return principal, nil
	}
	if !errors.Is(err, os.ErrNotExist) {
		return "", fmt.Errorf("read local identity: %w", err)
	}
	principal, err := commissionerapp.LocalPrincipal(dir, os.Getuid())
	if err != nil {
		return "", fmt.Errorf("compute local identity: %w", err)
	}
	f, err := os.OpenFile(identityFile(dir), os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if errors.Is(err, os.ErrExist) {
		return persistedIdentity(dir)
	}
	if err != nil {
		return "", fmt.Errorf("create local identity: %w", err)
	}
	_, writeErr := f.WriteString(principal)
	closeErr := f.Close()
	if writeErr != nil || closeErr != nil {
		return "", fmt.Errorf("write local identity: %w", errors.Join(writeErr, closeErr))
	}
	return principal, nil
}

func relinkIdentity(original string, stdout, stderr io.Writer) int {
	if err := relink(original); err != nil {
		_, _ = fmt.Fprintf(stderr, "Identity not re-linked: %v\n", err)
		return 1
	}
	_, _ = fmt.Fprintln(stdout, "Local identity re-linked.")
	return 0
}

func relink(original string) error {
	principal, err := commissionerapp.LocalPrincipal(original, os.Getuid())
	if err != nil || !filepath.IsAbs(original) {
		return fmt.Errorf("give the original absolute config directory path")
	}
	cfg, err := os.UserConfigDir()
	if err != nil {
		return fmt.Errorf("find profile: %w", err)
	}
	dir := filepath.Join(cfg, "TheWarRoom")
	ctx := context.Background()
	pools, err := db.OpenReadOnly(ctx, filepath.Join(dir, dbFileName("history", isDevBuild())), history.SchemaVersion())
	if err != nil {
		return fmt.Errorf("open profile ledger: %w", err)
	}
	defer func() { _ = pools.Close() }()
	store := history.New(pools, nil)
	leagues, err := store.NativeLeagues(ctx)
	if err != nil {
		return fmt.Errorf("read profile leagues: %w", err)
	}
	for _, league := range leagues {
		records, err := store.HeldCommissionerRecords(ctx, league.ID)
		if err != nil {
			return fmt.Errorf("read profile ledger: %w", err)
		}
		people, err := commissioner.PeopleReading(league.ID, principal, records)
		if err != nil {
			return fmt.Errorf("read profile seats: %w", err)
		}
		match := commissioner.Replay(records).Creator == principal
		for _, seat := range people.Seats {
			match = match || seat.Owner == principal
		}
		if match {
			return writeRelinkedIdentity(dir, principal)
		}
	}
	return fmt.Errorf("the original path does not match a league creator or seat holder in this profile")
}

func writeRelinkedIdentity(dir, principal string) error {
	f, err := os.CreateTemp(dir, ".local-identity-*")
	if err != nil {
		return fmt.Errorf("create local identity: %w", err)
	}
	defer func() { _ = os.Remove(f.Name()) }()
	_, writeErr := f.WriteString(principal)
	closeErr := f.Close()
	if writeErr != nil || closeErr != nil {
		return fmt.Errorf("write local identity: %w", errors.Join(writeErr, closeErr))
	}
	if err := os.Rename(f.Name(), identityFile(dir)); err != nil {
		return fmt.Errorf("install local identity: %w", err)
	}
	return nil
}
