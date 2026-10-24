package serverclient

import (
	"context"
	"crypto/sha256"
	"fmt"

	"github.com/secureprospective/TheWarRoom/internal/serverhttp"

	"github.com/secureprospective/TheWarRoom/internal/accounts"
	"github.com/zalando/go-keyring"
)

// CredentialStore is desktop-only secret storage. Implementations must never fall back to files.
type CredentialStore interface {
	Get() (string, error)
	Set(string) error
	Delete() error
}

type deviceKeychain struct{ account string }

func newDeviceKeychain(origin, league string) CredentialStore {
	return deviceKeychain{account: fmt.Sprintf("native-session:%x", sha256.Sum256([]byte(origin+"\x00"+league)))}
}
func (s deviceKeychain) Get() (string, error) {
	secret, err := keyring.Get("TheWarRoom.Native", s.account)
	if err != nil {
		return "", fmt.Errorf("read device keychain: %w", err)
	}
	return secret, nil
}
func (s deviceKeychain) Set(secret string) error {
	if err := keyring.Set("TheWarRoom.Native", s.account, secret); err != nil {
		return fmt.Errorf("save device keychain: %w", err)
	}
	return nil
}
func (s deviceKeychain) Delete() error {
	err := keyring.Delete("TheWarRoom.Native", s.account)
	if err == keyring.ErrNotFound {
		return nil
	}
	if err != nil {
		return fmt.Errorf("delete device keychain: %w", err)
	}
	return nil
}

func (d *Desktop) restore() {
	if d.loaded {
		return
	}
	d.loaded = true
	secret, err := d.credentials.Get()
	if err == nil && (accounts.RawSession{Bearer: secret}).Validate() == nil {
		d.bearer = secret
	}
}

func (d *Desktop) clearCredential() error {
	d.bearer, d.proof = "", ""
	if err := d.credentials.Delete(); err != nil {
		return fmt.Errorf("local keychain could not be cleared; sign in again and revoke all devices before closing the app")
	}
	return nil
}

func (d *Desktop) revoke(ctx context.Context, view DesktopView, all bool) (DesktopView, error) {
	if d.bearer == "" {
		return view, fmt.Errorf("sign in before revoking devices")
	}
	body, err := postDesktop(ctx, d, serverhttp.RevokePath, serverhttp.RawRevoke{Protocol: 1, AllDevices: all})
	if err != nil {
		return view, err
	}
	var response serverhttp.RevokeResponse
	if err := decodeDesktop(body, &response); err != nil {
		return view, err
	}
	cleanupErr := d.clearCredential()
	view.Message = "Device session revoked. Sign in again to reconnect."
	if all {
		view.Message = "All device sessions revoked. Every device must sign in again."
	}
	if cleanupErr != nil {
		view.Message += " The keychain could not be cleared, but its saved session is no longer valid."
	}
	return view, nil
}

func (d *Desktop) login(ctx context.Context, raw RawDesktop, view DesktopView) (DesktopView, error) {
	if err := d.clearCredential(); err != nil {
		view.Message = "Keychain unavailable; this sign-in will stay in memory only."
	}
	body, err := postDesktop(ctx, d, serverhttp.LoginPath, loginBody{1, raw.Username, raw.Password, "The War Room desktop"})
	if err != nil {
		return view, err
	}
	var response rawDesktopLogin
	if err := decodeDesktop(body, &response); err != nil {
		return view, err
	}
	d.bearer = response.Bearer
	view.Message = "Signed in. Device session saved in your OS keychain."
	if err := d.credentials.Set(d.bearer); err != nil {
		view.Message = "Signed in for this window only. Keychain unavailable; no plaintext file was written."
	}
	return view, nil
}
