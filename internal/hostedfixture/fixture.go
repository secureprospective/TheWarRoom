// Package hostedfixture provisions disposable hosted leagues for development and transport tests.
package hostedfixture

import (
	"context"
	"crypto/hmac"
	"crypto/sha1" //nolint:gosec // RFC 6238 authenticator compatibility.
	"encoding/base32"
	"encoding/binary"
	"fmt"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/accounts"
	"github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/governance"
	accountstore "github.com/secureprospective/TheWarRoom/internal/store/accounts"
	"github.com/secureprospective/TheWarRoom/internal/store/history"
)

// CreateLeague grants the creator authority through canonical league history, not an override.
func CreateLeague(ctx context.Context, store *history.Store, principal governance.Principal) error {
	create := commissioner.RawCommand{League: "native.hosted", Command: commissioner.Create,
		Args:   commissioner.RawArgs{Name: "Hosted test", Preset: commissioner.PresetEmpty, Season: 2026, FranchiseCount: 2},
		Reason: "Create disposable league", Request: "create", Effective: "now", Visibility: "league"}
	if _, err := store.ExecuteCommissioner(ctx, create, string(principal), nil); err != nil {
		return fmt.Errorf("create disposable league: %w", err)
	}
	return nil
}

// Code computes a disposable authenticator code without contacting the server.
func Code(seed string, at time.Time) (string, error) {
	decoded, err := base32.StdEncoding.WithPadding(base32.NoPadding).DecodeString(seed)
	if err != nil || len(decoded) != 20 || at.Unix() < 0 {
		return "", fmt.Errorf("invalid disposable authenticator seed or time")
	}
	var message [8]byte
	binary.BigEndian.PutUint64(message[:], uint64(at.Unix()/30)) //nolint:gosec // Negative time rejected above.
	mac := hmac.New(sha1.New, decoded)
	_, _ = mac.Write(message[:])
	sum := mac.Sum(nil)
	offset := sum[len(sum)-1] & 15
	return fmt.Sprintf("%06d", (binary.BigEndian.Uint32(sum[offset:offset+4])&0x7fffffff)%1000000), nil
}

// Bootstrap initializes the disposable account store and creates its sole initial account.
func Bootstrap(ctx context.Context, credentials *accountstore.Store, raw accounts.RawCredentials) (governance.Principal, error) {
	if err := credentials.Initialize(ctx); err != nil {
		return "", fmt.Errorf("initialize fixture accounts: %w", err)
	}
	account, err := accounts.New(credentials).LocalBootstrap(ctx, raw)
	if err != nil {
		return "", fmt.Errorf("bootstrap fixture account: %w", err)
	}
	principal, err := account.Principal()
	if err != nil {
		return "", fmt.Errorf("fixture account identity: %w", err)
	}
	return principal, nil
}
