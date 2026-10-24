package accounts

import (
	"bytes"
	"errors"
	"path/filepath"
	"strings"
	"testing"
	"time"

	identity "github.com/secureprospective/TheWarRoom/internal/accounts"
)

func TestTOTPPendingExpiryReplacementAndRollback(t *testing.T) {
	store, pools := openStore(t, filepath.Join(t.TempDir(), "accounts.db"))
	credential := fixtureCredential(t)
	if err := store.Insert(t.Context(), credential, true); err != nil {
		t.Fatal(err)
	}
	issued := time.Date(2026, 10, 24, 12, 0, 0, 0, time.UTC)
	now := issued
	store.SetClock(func() time.Time { return now })
	seed := bytes.Repeat([]byte{1}, 48)
	if err := store.BeginTOTP(t.Context(), credential.ID, seed); err != nil {
		t.Fatal(err)
	}
	replacement := bytes.Repeat([]byte{2}, 48)
	if err := store.BeginTOTP(t.Context(), credential.ID, replacement); err != nil {
		t.Fatal(err)
	}
	if err := store.ConsumeTOTP(t.Context(), credential.ID, seed, now.Unix()/30, true, ""); !errors.Is(err, identity.ErrTOTPRequired) {
		t.Fatalf("stale setup: %v", err)
	}
	now = issued.Add(identity.TOTPEnrollmentLifetime)
	if err := store.ConsumeTOTP(t.Context(), credential.ID, replacement, now.Unix()/30, true, ""); !errors.Is(err, identity.ErrTOTPRequired) {
		t.Fatalf("exact setup expiry: %v", err)
	}
	state, err := store.TOTP(t.Context(), credential.ID)
	if err != nil || state.Confirmed {
		t.Fatalf("refusal confirmed factor: %v", err)
	}
	if err := store.BeginTOTP(t.Context(), credential.ID, seed); err != nil {
		t.Fatal(err)
	}
	// Inject persistence failure after verification. No code or proof may be consumed.
	if _, err := pools.Write().ExecContext(t.Context(), `CREATE TRIGGER fail_totp BEFORE UPDATE ON account_totp BEGIN SELECT RAISE(ABORT, 'fixture failure'); END;`); err != nil {
		t.Fatal(err)
	}
	if err := store.ConsumeTOTP(t.Context(), credential.ID, seed, now.Unix()/30, true, ""); err == nil {
		t.Fatal("persistence failure ignored")
	}
	var counter int64
	if err := pools.Read().QueryRowContext(t.Context(), `SELECT last_counter FROM account_totp`).Scan(&counter); err != nil || counter != -1 {
		t.Fatalf("failure consumed code: %v", err)
	}
	if _, err := pools.Write().ExecContext(t.Context(), `DROP TRIGGER fail_totp`); err != nil {
		t.Fatal(err)
	}
	if err := store.ConsumeTOTP(t.Context(), credential.ID, seed, now.Unix()/30, true, ""); err != nil {
		t.Fatal(err)
	}
	if err := store.BeginTOTP(t.Context(), credential.ID, replacement); !errors.Is(err, identity.ErrTOTPRequired) {
		t.Fatalf("confirmed seed replaced: %v", err)
	}
	now = now.Add(30 * time.Second)
	digest := strings.Repeat("a", 64)
	if err := store.ConsumeTOTP(t.Context(), credential.ID, seed, now.Unix()/30, false, digest); err != nil {
		t.Fatal(err)
	}
	if err := store.CheckRecentAuth(t.Context(), credential.ID, digest); err != nil {
		t.Fatal(err)
	}
	// Codes older than a consumed future-window code cannot be replayed on rollback.
	if err := store.ConsumeTOTP(t.Context(), credential.ID, seed, now.Unix()/30-1, false, strings.Repeat("b", 64)); !errors.Is(err, identity.ErrTOTPRequired) {
		t.Fatalf("older counter: %v", err)
	}
	if err := store.CheckRecentAuth(t.Context(), credential.ID, digest); err != nil {
		t.Fatalf("refusal replaced proof: %v", err)
	}
}

func TestTOTPQueuedVerificationUsesAdmissionClock(t *testing.T) {
	store, pools := openStore(t, filepath.Join(t.TempDir(), "accounts.db"))
	credential := fixtureCredential(t)
	if err := store.Insert(t.Context(), credential, true); err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 10, 24, 12, 0, 0, 0, time.UTC)
	store.SetClock(func() time.Time { return now })
	seed := bytes.Repeat([]byte{3}, 48)
	if err := store.BeginTOTP(t.Context(), credential.ID, seed); err != nil {
		t.Fatal(err)
	}
	counter := now.Unix() / 30
	if err := store.ConsumeTOTP(t.Context(), credential.ID, seed, counter, true, ""); err != nil {
		t.Fatal(err)
	}
	tx, err := pools.Write().BeginTx(t.Context(), nil)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = tx.Rollback() }()
	admitted := make(chan struct{})
	store.SetClock(func() time.Time { close(admitted); return now.Add(3 * time.Minute) })
	result := make(chan error, 1)
	go func() {
		result <- store.ConsumeTOTP(t.Context(), credential.ID, seed, counter+1, false, strings.Repeat("c", 64))
	}()
	deadline := time.After(5 * time.Second)
	for pools.Write().Stats().WaitCount == 0 {
		select {
		case <-admitted:
			t.Fatal("clock sampled before writer admission")
		case <-deadline:
			t.Fatal("verification did not queue")
		default:
			time.Sleep(time.Millisecond)
		}
	}
	if err := tx.Rollback(); err != nil {
		t.Fatal(err)
	}
	if err := <-result; !errors.Is(err, identity.ErrTOTPRequired) {
		t.Fatalf("stale queued code: %v", err)
	}
	var gotCounter int64
	var proof string
	if err := pools.Read().QueryRowContext(t.Context(), `SELECT last_counter, proof_hash FROM account_totp`).Scan(&gotCounter, &proof); err != nil || gotCounter != counter || proof != "" {
		t.Fatalf("late verification mutated state: %v", err)
	}
}
