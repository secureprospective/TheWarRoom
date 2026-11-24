package accounts

import (
	"context"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// TestSessionTouchFailureStillAdmits: a valid session whose activity stamp cannot be written is still the
// owner's session; the stamp is simply not moved, so idle expiry is never extended by a failure.
func TestSessionTouchFailureStillAdmits(t *testing.T) {
	store, pools := openStore(t, filepath.Join(t.TempDir(), "server.db"))
	now := time.Date(2026, 11, 9, 12, 0, 0, 0, time.UTC)
	store.SetClock(func() time.Time { return now })
	credential := fixtureCredential(t)
	if err := store.Insert(t.Context(), credential, false); err != nil {
		t.Fatal(err)
	}
	digest := strings.Repeat("b", 64)
	if _, err := store.CreateSession(t.Context(), credential, digest, "phone"); err != nil {
		t.Fatal(err)
	}
	if _, err := pools.Write().ExecContext(t.Context(), `CREATE TRIGGER refuse_touch BEFORE UPDATE OF last_used_at ON account_sessions
 BEGIN SELECT RAISE(ABORT, 'writer busy'); END`); err != nil {
		t.Fatal(err)
	}
	now = now.Add(2 * sessionTouchInterval)
	account, err := store.SessionAccount(t.Context(), digest, true)
	if err != nil || account != credential.Account {
		t.Fatalf("a failed activity stamp refused a valid session: %+v %v", account, err)
	}
	var used int64
	if err := pools.Read().QueryRowContext(t.Context(), `SELECT last_used_at FROM account_sessions WHERE token_hash = ?`, digest).Scan(&used); err != nil {
		t.Fatal(err)
	}
	if !time.Unix(0, used).Equal(now.Add(-2 * sessionTouchInterval)) {
		t.Fatalf("the stamp moved despite the failed write: %v", time.Unix(0, used))
	}
}

// TestSessionTouchSkipsWhenWriterHeld: while another transaction holds the single writer,
// SessionAccount(touch=true) returns within 100ms without queuing and leaves last_used_at unchanged.
func TestSessionTouchSkipsWhenWriterHeld(t *testing.T) {
	store, pools := openStore(t, filepath.Join(t.TempDir(), "server.db"))
	now := time.Date(2026, 11, 9, 12, 0, 0, 0, time.UTC)
	store.SetClock(func() time.Time { return now })
	credential := fixtureCredential(t)
	if err := store.Insert(t.Context(), credential, false); err != nil {
		t.Fatal(err)
	}
	digest := strings.Repeat("c", 64)
	if _, err := store.CreateSession(t.Context(), credential, digest, "phone"); err != nil {
		t.Fatal(err)
	}
	held, err := pools.BeginWrite(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = held.Rollback() }()
	now = now.Add(2 * sessionTouchInterval)

	ctx, cancel := context.WithTimeout(t.Context(), 500*time.Millisecond)
	defer cancel()
	start := time.Now()
	account, err := store.SessionAccount(ctx, digest, true)
	elapsed := time.Since(start)
	if err != nil || account != credential.Account {
		t.Fatalf("busy writer refused a valid session: %+v %v", account, err)
	}
	if elapsed >= 100*time.Millisecond {
		t.Fatalf("SessionAccount waited %v while writer was held; want < 100ms", elapsed)
	}
	if err := held.Rollback(); err != nil {
		t.Fatal(err)
	}
	var used int64
	if err := pools.Read().QueryRowContext(t.Context(), `SELECT last_used_at FROM account_sessions WHERE token_hash = ?`, digest).Scan(&used); err != nil {
		t.Fatal(err)
	}
	if !time.Unix(0, used).Equal(now.Add(-2 * sessionTouchInterval)) {
		t.Fatalf("the stamp moved despite skipped touch: %v", time.Unix(0, used))
	}
}
