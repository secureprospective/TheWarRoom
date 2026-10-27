package main

import (
	"context"
	"crypto/rand"

	"fmt"
	"path/filepath"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/accounts"
	"github.com/secureprospective/TheWarRoom/internal/db"
	"github.com/secureprospective/TheWarRoom/internal/measures"
	accountstore "github.com/secureprospective/TheWarRoom/internal/store/accounts"
	"github.com/secureprospective/TheWarRoom/internal/store/history"
	"github.com/secureprospective/TheWarRoom/tools/hostedfixture"
)

func provision(ctx context.Context, dir string) (string, string, error) {
	pools, err := db.Open(ctx, filepath.Join(dir, "server.db"), history.SchemaVersion())
	if err != nil {
		return "", "", fmt.Errorf("open sandbox: %w", err)
	}
	defer func() { _ = pools.Close() }()
	registry, err := measures.Embedded()
	if err != nil {
		return "", "", fmt.Errorf("load measures: %w", err)
	}
	store := history.New(pools, registry)
	if err := store.Initialize(ctx); err != nil {
		return "", "", fmt.Errorf("initialize history: %w", err)
	}
	credentials := accountstore.New(pools)
	password := rand.Text()
	raw := accounts.RawCredentials{Username: "commissioner", Password: password}
	passwords := accounts.New(credentials)
	principal, err := hostedfixture.Bootstrap(ctx, credentials, raw)
	if err != nil {
		return "", "", fmt.Errorf("bootstrap commissioner: %w", err)
	}
	digest, err := hostedfixture.CreateWorkflowLeague(ctx, store, principal)
	if err != nil {
		return "", "", fmt.Errorf("seed owner workflows: %w", err)
	}
	for i, username := range []string{"owner", "partner"} {
		if err := hostedfixture.AddWorkflowOwner(ctx, store, credentials, principal, fmt.Sprintf("franchise.%02d", i+2), accounts.RawCredentials{Username: username, Password: password}); err != nil {
			return "", "", fmt.Errorf("seed practice owner: %w", err)
		}
	}

	if err := writePrivate(filepath.Join(dir, "football.digest"), []byte(digest)); err != nil {
		return "", "", err
	}
	return enrollAuthenticator(ctx, dir, credentials, passwords, raw)
}

func enrollAuthenticator(ctx context.Context, dir string, credentials *accountstore.Store, passwords *accounts.Service, raw accounts.RawCredentials) (string, string, error) {
	// Confirm in the preceding time step so the printed current code is immediately usable.
	now := time.Now().UTC().Add(-30 * time.Second)
	credentials.SetClock(func() time.Time { return now })
	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		return "", "", fmt.Errorf("generate encryption key: %w", err)
	}
	if err := writePrivate(filepath.Join(dir, "authenticator.key"), key); err != nil {
		return "", "", err
	}
	totp, err := accounts.NewTOTP(passwords, credentials, key)
	if err != nil {
		return "", "", fmt.Errorf("initialize authenticator: %w", err)
	}
	setup, err := totp.Begin(ctx, raw)
	if err != nil {
		return "", "", fmt.Errorf("begin authenticator: %w", err)
	}
	code, err := hostedfixture.Code(setup.Seed, now)
	if err != nil {
		return "", "", fmt.Errorf("compute confirmation: %w", err)
	}
	if err := totp.Confirm(ctx, raw, accounts.RawTOTP{Code: code}); err != nil {
		return "", "", fmt.Errorf("confirm authenticator: %w", err)
	}
	return raw.Password, setup.Seed, nil
}
