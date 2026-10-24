package accounts_test

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha1" //nolint:gosec // Independently generate the RFC 6238 HMAC-SHA1 authenticator profile.
	"encoding/base32"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/accounts"
	"github.com/secureprospective/TheWarRoom/internal/db"
	"github.com/secureprospective/TheWarRoom/internal/governance"
	accountstore "github.com/secureprospective/TheWarRoom/internal/store/accounts"
)

type totpFixture struct {
	store       *accountstore.Store
	pools       *db.Pools
	service     *accounts.TOTPService
	credentials accounts.RawCredentials
	account     accounts.Account
	policy      governance.State
	now         time.Time
	key         []byte
}

func newTOTPFixture(t *testing.T) *totpFixture {
	t.Helper()
	pools, err := db.Open(t.Context(), filepath.Join(t.TempDir(), "accounts.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = pools.Close() })
	store := accountstore.New(pools)
	if err := store.Initialize(t.Context()); err != nil {
		t.Fatal(err)
	}
	f := &totpFixture{store: store, pools: pools, credentials: accounts.RawCredentials{Username: "commissioner", Password: "private-authenticator-password"}, now: time.Date(2026, 10, 24, 12, 0, 0, 0, time.UTC), key: bytes.Repeat([]byte{19}, 32)}
	store.SetClock(func() time.Time { return f.now })
	f.account, err = accounts.New(store).LocalBootstrap(t.Context(), f.credentials)
	if err != nil {
		t.Fatal(err)
	}
	f.service, err = accounts.NewTOTP(accounts.New(store), store, f.key)
	if err != nil {
		t.Fatal(err)
	}
	principal, err := f.account.Principal()
	if err != nil {
		t.Fatal(err)
	}
	f.policy, err = governance.Apply(governance.State{}, governance.LeagueCreated{Creator: principal, Franchises: []governance.FranchiseID{"one"}})
	if err != nil {
		t.Fatal(err)
	}
	return f
}

func authenticatorCode(t *testing.T, seed string, at time.Time) accounts.RawTOTP {
	t.Helper()
	decoded, err := base32.StdEncoding.WithPadding(base32.NoPadding).DecodeString(seed)
	if err != nil {
		t.Fatal(err)
	}
	var message [8]byte
	counter := at.Unix() / 30
	if counter < 0 {
		t.Fatal("negative test counter")
		return accounts.RawTOTP{}
	}
	binary.BigEndian.PutUint64(message[:], uint64(counter))
	mac := hmac.New(sha1.New, decoded)
	_, _ = mac.Write(message[:])
	sum := mac.Sum(nil)
	offset := sum[len(sum)-1] & 15
	return accounts.RawTOTP{Code: fmt.Sprintf("%06d", (binary.BigEndian.Uint32(sum[offset:offset+4])&0x7fffffff)%1000000)}
}

func TestPrivilegedTOTPEnrollmentReplayAndRecentAuth(t *testing.T) {
	f := newTOTPFixture(t)
	principal, err := f.account.Principal()
	if err != nil {
		t.Fatal(err)
	}
	if err := f.service.CheckPrivileged(t.Context(), f.policy, principal, accounts.RecentAuth{}); !errors.Is(err, accounts.ErrTOTPRequired) {
		t.Fatalf("unconfigured commissioner: %v", err)
	}
	setup, err := f.service.Begin(t.Context(), f.credentials)
	if err != nil {
		t.Fatal(err)
	}
	code := authenticatorCode(t, setup.Seed, f.now)
	if _, err := f.service.StepUp(t.Context(), f.credentials, code); !errors.Is(err, accounts.ErrTOTPRequired) {
		t.Fatalf("pending enrollment granted step-up: %v", err)
	}
	if err := f.service.Confirm(t.Context(), f.credentials, code); err != nil {
		t.Fatal(err)
	}
	if err := f.service.CheckPrivileged(t.Context(), f.policy, principal, accounts.RecentAuth{}); !errors.Is(err, accounts.ErrTOTPRequired) {
		t.Fatalf("setup granted recent auth: %v", err)
	}
	if _, err := f.service.StepUp(t.Context(), f.credentials, code); !errors.Is(err, accounts.ErrTOTPRequired) {
		t.Fatalf("confirmation replay: %v", err)
	}
	if _, err := f.service.Begin(t.Context(), f.credentials); !errors.Is(err, accounts.ErrTOTPRequired) {
		t.Fatalf("confirmed seed replaced: %v", err)
	}
	f.now = f.now.Add(30 * time.Second)
	proof, err := f.service.StepUp(t.Context(), f.credentials, authenticatorCode(t, setup.Seed, f.now))
	if err != nil {
		t.Fatal(err)
	}
	authenticated := f.now
	if err := f.service.CheckPrivileged(t.Context(), f.policy, principal, proof); err != nil {
		t.Fatal(err)
	}
	if err := f.service.CheckPrivileged(t.Context(), f.policy, principal, accounts.RecentAuth{Token: strings.Repeat("a", 43)}); !errors.Is(err, accounts.ErrTOTPRequired) {
		t.Fatalf("forged proof: %v", err)
	}
	f.now = authenticated.Add(-time.Nanosecond)
	if err := f.service.CheckPrivileged(t.Context(), f.policy, principal, proof); !errors.Is(err, accounts.ErrTOTPRequired) {
		t.Fatalf("backward clock: %v", err)
	}
	f.now = authenticated.Add(5*time.Minute - time.Nanosecond)
	if err := f.service.CheckPrivileged(t.Context(), f.policy, principal, proof); err != nil {
		t.Fatalf("before expiry: %v", err)
	}
	f.now = authenticated.Add(5 * time.Minute)
	if err := f.service.CheckPrivileged(t.Context(), f.policy, principal, proof); !errors.Is(err, accounts.ErrTOTPRequired) {
		t.Fatalf("exact expiry: %v", err)
	}
	f.now = authenticated.Add(5*time.Minute + time.Nanosecond)
	if err := f.service.CheckPrivileged(t.Context(), f.policy, principal, proof); !errors.Is(err, accounts.ErrTOTPRequired) {
		t.Fatalf("after expiry: %v", err)
	}
	assertTOTPSecretsRedacted(t, setup, code, proof)
	var sealed []byte
	var persistedProof string
	if err := f.pools.Read().QueryRowContext(t.Context(), `SELECT sealed_seed, proof_hash FROM account_totp WHERE account_id = ?`, f.account.ID).Scan(&sealed, &persistedProof); err != nil {
		t.Fatal(err)
	}
	decoded, err := base32.StdEncoding.WithPadding(base32.NoPadding).DecodeString(setup.Seed)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(sealed, decoded) || bytes.Contains(sealed, []byte(setup.Seed)) || persistedProof == proof.Token || len(persistedProof) != 64 {
		t.Fatal("secrets persisted in plaintext")
	}
}

func assertTOTPSecretsRedacted(t *testing.T, setup accounts.TOTPEnrollment, code accounts.RawTOTP, proof accounts.RecentAuth) {
	t.Helper()
	formatted := fmt.Sprintf("%v %+v %#v %v %+v %#v %v %+v %#v", setup, setup, setup, code, code, code, proof, proof, proof)
	body, err := json.Marshal(struct {
		Setup accounts.TOTPEnrollment
		Code  accounts.RawTOTP
		Proof accounts.RecentAuth
	}{setup, code, proof})
	if err != nil {
		t.Fatal(err)
	}
	for _, secret := range []string{setup.Seed, code.Code, proof.Token} {
		if secret != "" && (strings.Contains(formatted, secret) || bytes.Contains(body, []byte(secret))) {
			t.Fatal("authenticator secret in log or JSON")
		}
	}
}

func TestTOTPDeputiesAndAccountBinding(t *testing.T) {
	f := newTOTPFixture(t)
	commissioner, err := f.account.Principal()
	if err != nil {
		t.Fatal(err)
	}
	for i := range 2 {
		credential, err := f.store.Credential(t.Context(), f.account.Username)
		if err != nil {
			t.Fatal(err)
		}
		credential.ID = fmt.Sprintf("87654321-1234-4234-8234-123456789ab%d", i)
		credential.Username = fmt.Sprintf("deputy%d", i)
		if err := f.store.Insert(t.Context(), credential, false); err != nil {
			t.Fatal(err)
		}
		deputy, err := credential.Principal()
		if err != nil {
			t.Fatal(err)
		}
		f.policy, err = governance.Apply(f.policy, governance.RoleGranted{Principal: deputy, Role: governance.CoCommissioner})
		if err != nil {
			t.Fatal(err)
		}
		if err := f.service.CheckPrivileged(t.Context(), f.policy, deputy, accounts.RecentAuth{}); !errors.Is(err, accounts.ErrTOTPRequired) {
			t.Fatalf("unconfigured deputy: %v", err)
		}
		raw := f.credentials
		raw.Username = credential.Username
		setup, err := f.service.Begin(t.Context(), raw)
		if err != nil {
			t.Fatal(err)
		}
		if err := f.service.Confirm(t.Context(), raw, authenticatorCode(t, setup.Seed, f.now)); err != nil {
			t.Fatal(err)
		}
		f.now = f.now.Add(30 * time.Second)
		proof, err := f.service.StepUp(t.Context(), raw, authenticatorCode(t, setup.Seed, f.now))
		if err != nil {
			t.Fatal(err)
		}
		if err := f.service.CheckPrivileged(t.Context(), f.policy, deputy, proof); err != nil {
			t.Fatal(err)
		}
		if err := f.service.CheckPrivileged(t.Context(), f.policy, commissioner, proof); !errors.Is(err, accounts.ErrTOTPRequired) {
			t.Fatalf("cross-account proof: %v", err)
		}
	}
	if err := f.service.CheckPrivileged(t.Context(), f.policy, "ordinary.owner", accounts.RecentAuth{}); err != nil {
		t.Fatalf("TOTP imposed on unprivileged owner: %v", err)
	}
}

func TestTOTPConcurrentCodeUseAndReopen(t *testing.T) {
	f := newTOTPFixture(t)
	setup, err := f.service.Begin(t.Context(), f.credentials)
	if err != nil {
		t.Fatal(err)
	}
	if err := f.service.Confirm(t.Context(), f.credentials, authenticatorCode(t, setup.Seed, f.now)); err != nil {
		t.Fatal(err)
	}
	f.now = f.now.Add(30 * time.Second)
	code := authenticatorCode(t, setup.Seed, f.now)
	var wait sync.WaitGroup
	results := make(chan error, 4)
	for range 4 {
		wait.Go(func() { _, err := f.service.StepUp(t.Context(), f.credentials, code); results <- err })
	}
	wait.Wait()
	close(results)
	wins := 0
	for err := range results {
		if err == nil {
			wins++
		} else if !errors.Is(err, accounts.ErrTOTPRequired) {
			t.Fatal(err)
		}
	}
	if wins != 1 {
		t.Fatalf("code accepted %d times", wins)
	}
	// Reopen the same file with fresh store/service objects; replay state is durable.
	var path string
	var seq int
	var name string
	if err := f.pools.Read().QueryRowContext(t.Context(), `PRAGMA database_list`).Scan(&seq, &name, &path); err != nil {
		t.Fatal(err)
	}
	if err := f.pools.Close(); err != nil {
		t.Fatal(err)
	}
	pools, err := db.Open(t.Context(), path)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = pools.Close() }()
	store := accountstore.New(pools)
	store.SetClock(func() time.Time { return f.now })
	service, err := accounts.NewTOTP(accounts.New(store), store, f.key)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.StepUp(t.Context(), f.credentials, code); !errors.Is(err, accounts.ErrTOTPRequired) {
		t.Fatalf("replay after restart: %v", err)
	}
	f.now = f.now.Add(30 * time.Second)
	if _, err := service.StepUp(t.Context(), f.credentials, authenticatorCode(t, setup.Seed, f.now)); err != nil {
		t.Fatalf("fresh code after restart: %v", err)
	}
}
