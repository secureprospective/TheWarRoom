package accounts_test

import (
	"bytes"
	"errors"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/accounts"
)

func TestTOTPCredentialsCiphertextIntegrityAndPendingExpiry(t *testing.T) {
	f := newTOTPFixture(t)
	wrong := f.credentials
	wrong.Password = "not-the-account-password"
	if _, err := f.service.Begin(t.Context(), wrong); !errors.Is(err, accounts.ErrInvalidCredentials) {
		t.Fatalf("wrong password setup: %v", err)
	}
	var count int
	if err := f.pools.Read().QueryRowContext(t.Context(), `SELECT count(*) FROM account_totp`).Scan(&count); err != nil || count != 0 {
		t.Fatalf("bad password created setup: %v", err)
	}
	setup, err := f.service.Begin(t.Context(), f.credentials)
	if err != nil {
		t.Fatal(err)
	}
	code := authenticatorCode(t, setup.Seed, f.now)
	badCode := code
	previous := authenticatorCode(t, setup.Seed, f.now.Add(-30*time.Second))
	next := authenticatorCode(t, setup.Seed, f.now.Add(30*time.Second))
	for badCode.Code == code.Code || badCode.Code == previous.Code || badCode.Code == next.Code {
		badCode.Code = string('0'+(badCode.Code[0]-'0'+1)%10) + badCode.Code[1:]
	}
	if err := f.service.Confirm(t.Context(), f.credentials, badCode); !errors.Is(err, accounts.ErrTOTPRequired) {
		t.Fatalf("incorrect code: %v", err)
	}
	if err := f.service.Confirm(t.Context(), wrong, code); !errors.Is(err, accounts.ErrInvalidCredentials) {
		t.Fatalf("wrong password confirm: %v", err)
	}
	state, err := f.store.TOTP(t.Context(), f.account.ID)
	if err != nil || state.Confirmed {
		t.Fatalf("failed verification changed setup: %v", err)
	}
	wrongKey, err := accounts.NewTOTP(accounts.New(f.store), f.store, bytes.Repeat([]byte{20}, 32))
	if err != nil {
		t.Fatal(err)
	}
	if err := wrongKey.Confirm(t.Context(), f.credentials, code); !errors.Is(err, accounts.ErrTOTPRequired) {
		t.Fatalf("wrong encryption key: %v", err)
	}
	corrupted := bytes.Clone(state.SealedSeed)
	corrupted[len(corrupted)-1] ^= 1
	if _, err := f.pools.Write().ExecContext(t.Context(), `UPDATE account_totp SET sealed_seed = ? WHERE account_id = ?`, corrupted, f.account.ID); err != nil {
		t.Fatal(err)
	}
	if err := f.service.Confirm(t.Context(), f.credentials, code); !errors.Is(err, accounts.ErrTOTPRequired) {
		t.Fatalf("tampered ciphertext: %v", err)
	}
	if _, err := f.pools.Write().ExecContext(t.Context(), `UPDATE account_totp SET sealed_seed = ? WHERE account_id = ?`, state.SealedSeed, f.account.ID); err != nil {
		t.Fatal(err)
	}
	f.now = f.now.Add(accounts.TOTPEnrollmentLifetime)
	if err := f.service.Confirm(t.Context(), f.credentials, authenticatorCode(t, setup.Seed, f.now)); !errors.Is(err, accounts.ErrTOTPRequired) {
		t.Fatalf("expired setup: %v", err)
	}
	// Expired and failed verification paths leave enrollment unconfirmed.
	after, err := f.store.TOTP(t.Context(), f.account.ID)
	if err != nil || after.Confirmed || !bytes.Equal(after.SealedSeed, state.SealedSeed) {
		t.Fatalf("refusal mutated enrollment: %v", err)
	}
}

func TestTOTPSeedEncryptionBindsAccountAndFreshProofReplacesOld(t *testing.T) {
	f := newTOTPFixture(t)
	setup, err := f.service.Begin(t.Context(), f.credentials)
	if err != nil {
		t.Fatal(err)
	}
	original, err := f.store.TOTP(t.Context(), f.account.ID)
	if err != nil {
		t.Fatal(err)
	}
	other, err := f.store.Credential(t.Context(), f.account.Username)
	if err != nil {
		t.Fatal(err)
	}
	other.ID = "87654321-1234-4234-8234-123456789abc"
	other.Username = "another.account"
	if err := f.store.Insert(t.Context(), other, false); err != nil {
		t.Fatal(err)
	}
	if err := f.store.BeginTOTP(t.Context(), other.ID, original.SealedSeed); err != nil {
		t.Fatal(err)
	}
	raw := f.credentials
	raw.Username = other.Username
	if err := f.service.Confirm(t.Context(), raw, authenticatorCode(t, setup.Seed, f.now)); !errors.Is(err, accounts.ErrTOTPRequired) {
		t.Fatalf("cross-account ciphertext: %v", err)
	}
	if err := f.service.Confirm(t.Context(), f.credentials, authenticatorCode(t, setup.Seed, f.now)); err != nil {
		t.Fatal(err)
	}
	f.now = f.now.Add(30 * time.Second)
	first, err := f.service.StepUp(t.Context(), f.credentials, authenticatorCode(t, setup.Seed, f.now))
	if err != nil {
		t.Fatal(err)
	}
	f.now = f.now.Add(30 * time.Second)
	second, err := f.service.StepUp(t.Context(), f.credentials, authenticatorCode(t, setup.Seed, f.now))
	if err != nil {
		t.Fatal(err)
	}
	principal, err := f.account.Principal()
	if err != nil {
		t.Fatal(err)
	}
	if err := f.service.CheckPrivileged(t.Context(), f.policy, principal, first); !errors.Is(err, accounts.ErrTOTPRequired) {
		t.Fatalf("superseded proof: %v", err)
	}
	if err := f.service.CheckPrivileged(t.Context(), f.policy, principal, second); err != nil {
		t.Fatal(err)
	}
}
