package main

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/secureprospective/TheWarRoom/internal/accounts"
	"github.com/secureprospective/TheWarRoom/internal/db"
	accountstore "github.com/secureprospective/TheWarRoom/internal/store/accounts"
)

// This command never starts a listener. The key digest must come from trusted
// operator configuration; neither a league role nor stdin can select it.
func runRecovery(ctx context.Context, args []string, input io.Reader, output io.Writer) error {
	flags := flag.NewFlagSet("twr-server recover", flag.ContinueOnError)
	path := flags.String("accounts-db", "", "existing server database")
	keyPath := flags.String("offline-key-file", "", "mounted offline 32-byte key; remove after use")
	pinned := flags.String("offline-key-sha256", "", "trusted configured SHA-256 of offline key (hex)")
	if err := flags.Parse(args); err != nil {
		return fmt.Errorf("parse recovery flags: %w", err)
	}
	digest, err := hex.DecodeString(*pinned)
	if err != nil || len(digest) != 32 || flags.NArg() != 0 {
		return fmt.Errorf("recovery requires a configured offline key digest")
	}
	info, err := os.Stat(*path)
	if err != nil || !info.Mode().IsRegular() {
		return fmt.Errorf("recovery requires an existing server database")
	}
	key, err := readOfflineKey(*keyPath)
	if err != nil {
		return err
	}
	raw, err := readConsoleRecovery(input)
	if err != nil {
		return err
	}
	pools, err := db.Open(ctx, *path)
	if err != nil {
		return fmt.Errorf("open recovery database: %w", err)
	}
	defer func() { _ = pools.Close() }()
	store := accountstore.New(pools)
	if err := store.Initialize(ctx); err != nil {
		return fmt.Errorf("initialize recovery storage: %w", err)
	}
	service := accounts.NewRecovery(accounts.New(store), store, [32]byte(digest))
	if err := service.ConsoleRecover(ctx, raw, key); err != nil {
		return fmt.Errorf("console recovery: %w", err)
	}
	if _, err := fmt.Fprintln(output, "Account recovered. Existing credentials are revoked; set up the authenticator again."); err != nil {
		return fmt.Errorf("report recovery: %w", err)
	}
	return nil
}

func readOfflineKey(path string) ([]byte, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open offline key: %w", err)
	}
	defer func() { _ = file.Close() }()
	info, err := file.Stat()
	if err != nil || !info.Mode().IsRegular() || info.Mode().Perm()&0077 != 0 {
		return nil, fmt.Errorf("offline key must be an owner-only regular file")
	}
	key, err := io.ReadAll(io.LimitReader(file, 33))
	if err != nil {
		return nil, fmt.Errorf("read offline key: %w", err)
	}
	if len(key) != 32 {
		return nil, fmt.Errorf("offline key must contain exactly 32 bytes")
	}
	return key, nil
}

func readConsoleRecovery(input io.Reader) (accounts.RawConsoleRecovery, error) {
	var raw accounts.RawConsoleRecovery
	body, err := io.ReadAll(io.LimitReader(input, 4097))
	if err != nil || len(body) > 4096 {
		return accounts.RawConsoleRecovery{}, fmt.Errorf("recovery input exceeds 4096 bytes or cannot be read")
	}
	decoder := json.NewDecoder(bytes.NewReader(body))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&raw); err != nil {
		return accounts.RawConsoleRecovery{}, fmt.Errorf("recovery requires username, password and reason as JSON on stdin")
	}
	var trailing json.RawMessage
	if err := decoder.Decode(&trailing); err != io.EOF {
		return accounts.RawConsoleRecovery{}, fmt.Errorf("recovery requires exactly one JSON object")
	}
	if err := raw.Validate(); err != nil {
		return accounts.RawConsoleRecovery{}, accounts.ErrRecoveryRefused
	}

	return raw, nil
}
