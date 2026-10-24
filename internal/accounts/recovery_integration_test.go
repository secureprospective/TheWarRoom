package accounts_test

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/accounts"
)

func recoveryFixture(t *testing.T) (*totpFixture, *accounts.RecoveryService, accounts.RecoveryCodes, accounts.RecentAuth) {
	t.Helper()
	f := newTOTPFixture(t)
	setup, err := f.service.Begin(t.Context(), f.credentials)
	if err != nil {
		t.Fatal(err)
	}
	if err := f.service.Confirm(t.Context(), f.credentials, authenticatorCode(t, setup.Seed, f.now)); err != nil {
		t.Fatal(err)
	}
	f.now = f.now.Add(30 * time.Second)
	proof, err := f.service.StepUp(t.Context(), f.credentials, authenticatorCode(t, setup.Seed, f.now))
	if err != nil {
		t.Fatal(err)
	}
	recovery := accounts.NewRecovery(accounts.New(f.store), f.store, sha256.Sum256(bytes.Repeat([]byte{42}, 32)))
	codes, err := recovery.PrintCodes(t.Context(), f.credentials, proof)
	if err != nil {
		t.Fatal(err)
	}
	return f, recovery, codes, proof
}

func TestRecoveryConcurrentUseRevokesCredentials(t *testing.T) {
	f, recovery, codes, proof := recoveryFixture(t)
	seen := map[string]bool{}
	for _, code := range codes.Codes {
		if len(code) != 43 || seen[code] {
			t.Fatal("codes not eight independent secrets")
		}
		seen[code] = true
	}
	generation, err := f.store.AuthGeneration(t.Context(), f.account.ID)
	if err != nil {
		t.Fatal(err)
	}
	raw := accounts.RawRecovery{Username: f.account.Username, Code: codes.Codes[0], Password: "replacement-password"}
	var wg sync.WaitGroup
	results := make(chan error, 2)
	for range 2 {
		wg.Go(func() { results <- recovery.Recover(t.Context(), raw) })
	}
	wg.Wait()
	close(results)
	successes := 0
	for err := range results {
		if err == nil {
			successes++
		} else if !errors.Is(err, accounts.ErrRecoveryRefused) {
			t.Fatal(err)
		}
	}
	if successes != 1 {
		t.Fatalf("got %d recovery winners", successes)
	}
	after, err := f.store.AuthGeneration(t.Context(), f.account.ID)
	if err != nil || after != generation+1 {
		t.Fatalf("delegation generation not revoked: %d %v", after, err)
	}
	if err := f.store.CheckRecentAuth(t.Context(), f.account.ID, digestToken(proof.Token)); !errors.Is(err, accounts.ErrTOTPRequired) {
		t.Fatalf("recent proof survived: %v", err)
	}
	if _, err := accounts.New(f.store).Authenticate(t.Context(), f.credentials); !errors.Is(err, accounts.ErrInvalidCredentials) {
		t.Fatalf("old password survived: %v", err)
	}
	if _, err := accounts.New(f.store).Authenticate(t.Context(), accounts.RawCredentials{Username: f.account.Username, Password: raw.Password}); err != nil {
		t.Fatal(err)
	}
	principal, err := f.account.Principal()
	if err != nil {
		t.Fatal(err)
	}
	if err := f.service.CheckPrivileged(t.Context(), f.policy, principal, proof); !errors.Is(err, accounts.ErrTOTPRequired) {
		t.Fatalf("reset privileged account admitted without setup: %v", err)
	}
	raw.Code = codes.Codes[1]
	if err := recovery.Recover(t.Context(), raw); !errors.Is(err, accounts.ErrRecoveryRefused) {
		t.Fatalf("other old codes survived: %v", err)
	}
	var count int
	if err := f.pools.Read().QueryRowContext(t.Context(), `SELECT count(*) FROM account_security_audit WHERE operation = 'recovery_code_reset'`).Scan(&count); err != nil || count != 1 {
		t.Fatalf("audit: %d %v", count, err)
	}
}

func digestToken(token string) string { return fmt.Sprintf("%x", sha256.Sum256([]byte(token))) }

func TestRecoveryAuditFailureRollsBackAndConsoleRequiresOfflineKey(t *testing.T) {
	f, recovery, codes, proof := recoveryFixture(t)
	raw := accounts.RawRecovery{Username: f.account.Username, Code: codes.Codes[0], Password: "replacement-password"}
	if _, err := f.pools.Write().ExecContext(t.Context(), `CREATE TRIGGER fail_recovery_audit BEFORE INSERT ON account_security_audit BEGIN SELECT RAISE(ABORT, 'fixture failure'); END;`); err != nil {
		t.Fatal(err)
	}
	if err := recovery.Recover(t.Context(), raw); err == nil {
		t.Fatal("audit failure ignored")
	}
	if _, err := accounts.New(f.store).Authenticate(t.Context(), f.credentials); err != nil {
		t.Fatalf("failed recovery changed password: %v", err)
	}
	if err := f.store.CheckRecentAuth(t.Context(), f.account.ID, digestToken(proof.Token)); err != nil {
		t.Fatalf("failed recovery removed proof: %v", err)
	}
	generation, err := f.store.AuthGeneration(t.Context(), f.account.ID)
	if err != nil || generation != 0 {
		t.Fatalf("failed recovery revoked credentials: %d %v", generation, err)
	}
	if _, err := f.pools.Write().ExecContext(t.Context(), `DROP TRIGGER fail_recovery_audit`); err != nil {
		t.Fatal(err)
	}
	console := accounts.RawConsoleRecovery{Username: f.account.Username, Password: raw.Password, Reason: "Lost printed codes and authenticator"}
	if err := recovery.ConsoleRecover(t.Context(), console, bytes.Repeat([]byte{1}, 32)); !errors.Is(err, accounts.ErrRecoveryRefused) {
		t.Fatalf("wrong offline key: %v", err)
	}
	// A wrong account cannot be reset with another account's recovery code, no matter
	// what league role or seat the caller has. No policy input exists at this door.
	other := raw
	other.Username = "another-owner"
	if err := recovery.Recover(t.Context(), other); !errors.Is(err, accounts.ErrRecoveryRefused) {
		t.Fatalf("cross-account code: %v", err)
	}
	if err := recovery.ConsoleRecover(t.Context(), console, bytes.Repeat([]byte{42}, 32)); err != nil {
		t.Fatal(err)
	}
	if err := recovery.Recover(t.Context(), raw); !errors.Is(err, accounts.ErrRecoveryRefused) {
		t.Fatalf("console retained old recovery code: %v", err)
	}
	var reason string
	if err := f.pools.Read().QueryRowContext(t.Context(), `SELECT reason FROM account_security_audit WHERE operation = 'offline_key_reset'`).Scan(&reason); err != nil || reason != console.Reason {
		t.Fatalf("console audit: %q %v", reason, err)
	}
	encoded, err := json.Marshal(raw)
	if err != nil {
		t.Fatal(err)
	}
	for _, value := range []string{string(encoded), fmt.Sprintf("%+v %#v %v", raw, console, codes)} {
		if strings.Contains(value, raw.Password) || strings.Contains(value, raw.Code) {
			t.Fatal("secret leaked")
		}
	}
}

func TestRecoveryPrintingRequiresFreshAccountBoundProof(t *testing.T) {
	f, recovery, _, proof := recoveryFixture(t)
	if _, err := recovery.PrintCodes(t.Context(), f.credentials, accounts.RecentAuth{Token: strings.Repeat("a", 43)}); !errors.Is(err, accounts.ErrRecoveryRefused) {
		t.Fatalf("forged proof: %v", err)
	}
	f.now = f.now.Add(accounts.RecentAuthLifetime)
	if _, err := recovery.PrintCodes(t.Context(), f.credentials, proof); !errors.Is(err, accounts.ErrRecoveryRefused) {
		t.Fatalf("expired proof: %v", err)
	}
}
