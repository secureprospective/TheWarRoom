package accounts

import (
	"errors"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	identity "github.com/secureprospective/TheWarRoom/internal/accounts"
)

func TestEnrollmentConcurrentRedemptionAndReopen(t *testing.T) {
	path := filepath.Join(t.TempDir(), "server.db")
	store, pools := openStore(t, path)
	issuer := fixtureCredential(t)
	if err := store.Insert(t.Context(), issuer, true); err != nil {
		t.Fatal(err)
	}
	principal, err := issuer.Principal()
	if err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 10, 24, 12, 0, 0, 123, time.UTC)
	store.SetClock(func() time.Time { return now })
	digest := strings.Repeat("a", 64)
	expires, err := store.IssueEnrollment(t.Context(), digest, "new.owner", "native.test", principal)
	if err != nil || !expires.Equal(now.Add(48*time.Hour)) {
		t.Fatalf("issue: %v %v", expires, err)
	}
	credential := issuer
	credential.ID = "87654321-1234-4234-8234-123456789abc"
	credential.Username = "new.owner"
	var wait sync.WaitGroup
	results := make(chan error, 8)
	for range 8 {
		wait.Go(func() { results <- store.RedeemEnrollment(t.Context(), digest, credential) })
	}
	wait.Wait()
	close(results)
	wins := 0
	for err := range results {
		if err == nil {
			wins++
		} else if !errors.Is(err, identity.ErrEnrollmentRefused) {
			t.Fatal(err)
		}
	}
	if wins != 1 {
		t.Fatalf("redemption winners: %d", wins)
	}
	if err := pools.Close(); err != nil {
		t.Fatal(err)
	}
	store, _ = openStore(t, path)
	store.SetClock(func() time.Time { return now })
	if err := store.RedeemEnrollment(t.Context(), digest, credential); !errors.Is(err, identity.ErrEnrollmentRefused) {
		t.Fatalf("replayed after reopen: %v", err)
	}
	got, err := store.Credential(t.Context(), credential.Username)
	if err != nil || got != credential {
		t.Fatalf("persisted account: %v", err)
	}
}

func TestEnrollmentExpiryIdentityConflictAndRollback(t *testing.T) {
	store, pools := openStore(t, filepath.Join(t.TempDir(), "server.db"))
	issuer := fixtureCredential(t)
	if err := store.Insert(t.Context(), issuer, true); err != nil {
		t.Fatal(err)
	}
	principal, err := issuer.Principal()
	if err != nil {
		t.Fatal(err)
	}
	issued := time.Date(2026, 10, 24, 12, 0, 0, 123, time.UTC)
	now := issued
	store.SetClock(func() time.Time { return now })
	next := issuer
	next.ID = "87654321-1234-4234-8234-123456789abc"
	next.Username = "new.owner"
	for i, offset := range []time.Duration{-time.Nanosecond, 0, time.Nanosecond} {
		now = issued
		digest := strings.Repeat(string(rune('a'+i)), 64)
		expires, err := store.IssueEnrollment(t.Context(), digest, next.Username, "native.test", principal)
		if err != nil {
			t.Fatal(err)
		}
		now = expires.Add(offset)
		err = store.RedeemEnrollment(t.Context(), digest, next)
		if offset < 0 && err != nil {
			t.Fatalf("before expiry: %v", err)
		}
		if offset >= 0 && !errors.Is(err, identity.ErrEnrollmentRefused) {
			t.Fatalf("at/after expiry: %v", err)
		}
		if offset < 0 {
			next.Username = "later.owner"
			next.ID = "87654321-1234-4234-8234-123456789abd"
		}
	}
	now = issued
	digest := strings.Repeat("d", 64)
	if _, err := store.IssueEnrollment(t.Context(), digest, next.Username, "native.test", principal); err != nil {
		t.Fatal(err)
	}
	wrong := next
	wrong.Username = "wrong.owner"
	if err := store.RedeemEnrollment(t.Context(), digest, wrong); !errors.Is(err, identity.ErrEnrollmentRefused) {
		t.Fatalf("wrong identity: %v", err)
	}
	if _, err := store.Credential(t.Context(), wrong.Username); !errors.Is(err, identity.ErrNotFound) {
		t.Fatal("wrong identity created account")
	}
	// A persistence failure after insertion must restore both the code and account.
	if _, err := pools.Write().ExecContext(t.Context(), `CREATE TRIGGER reject_consumption BEFORE UPDATE ON account_enrollments BEGIN SELECT RAISE(ABORT, 'injected failure'); END`); err != nil {
		t.Fatal(err)
	}
	if err := store.RedeemEnrollment(t.Context(), digest, next); err == nil {
		t.Fatal("injected failure accepted")
	}
	if _, err := store.Credential(t.Context(), next.Username); !errors.Is(err, identity.ErrNotFound) {
		t.Fatal("failed consumption left account")
	}
	if _, err := pools.Write().ExecContext(t.Context(), `DROP TRIGGER reject_consumption`); err != nil {
		t.Fatal(err)
	}
	if err := store.Insert(t.Context(), next, false); err != nil {
		t.Fatal(err)
	}
	replacement := next
	replacement.ID = "87654321-1234-4234-8234-123456789abe"
	if err := store.RedeemEnrollment(t.Context(), digest, replacement); !errors.Is(err, identity.ErrEnrollmentRefused) {
		t.Fatalf("occupied username: %v", err)
	}
	got, err := store.Credential(t.Context(), next.Username)
	if err != nil || got != next {
		t.Fatal("enrollment replaced an existing identity")
	}
	var consumed bool
	if err := pools.Read().QueryRowContext(t.Context(), `SELECT redeemed_account IS NOT NULL FROM account_enrollments WHERE code_hash = ?`, digest).Scan(&consumed); err != nil || consumed {
		t.Fatalf("refusal consumed code: %v", err)
	}
}

func TestEnrollmentQueuedPastExpiry(t *testing.T) {
	store, pools := openStore(t, filepath.Join(t.TempDir(), "server.db"))
	issuer := fixtureCredential(t)
	if err := store.Insert(t.Context(), issuer, true); err != nil {
		t.Fatal(err)
	}
	principal, err := issuer.Principal()
	if err != nil {
		t.Fatal(err)
	}
	issued := time.Date(2026, 10, 24, 12, 0, 0, 0, time.UTC)
	store.SetClock(func() time.Time { return issued })
	digest := strings.Repeat("e", 64)
	expires, err := store.IssueEnrollment(t.Context(), digest, "queued.owner", "native.test", principal)
	if err != nil {
		t.Fatal(err)
	}
	tx, err := pools.Write().BeginTx(t.Context(), nil)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = tx.Rollback() }()
	admitted := make(chan struct{})
	store.SetClock(func() time.Time { close(admitted); return expires })
	credential := issuer
	credential.Username = "queued.owner"
	credential.ID = "87654321-1234-4234-8234-123456789abc"
	result := make(chan error, 1)
	go func() { result <- store.RedeemEnrollment(t.Context(), digest, credential) }()
	// Wait until the writer acquisition is demonstrably queued, not merely scheduled.
	deadline := time.After(5 * time.Second)
	for pools.Write().Stats().WaitCount == 0 {
		select {
		case <-admitted:
			t.Fatal("sampled clock before writer admission")
		case <-deadline:
			t.Fatal("redemption did not queue")
		default:
			time.Sleep(time.Millisecond)
		}
	}
	if err := tx.Rollback(); err != nil {
		t.Fatal(err)
	}
	if err := <-result; !errors.Is(err, identity.ErrEnrollmentRefused) {
		t.Fatalf("late queued redemption: %v", err)
	}
	if _, err := store.Credential(t.Context(), credential.Username); !errors.Is(err, identity.ErrNotFound) {
		t.Fatal("late redemption created account")
	}
}
