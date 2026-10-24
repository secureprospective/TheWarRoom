package accounts

import (
	"errors"
	"path/filepath"
	"strings"
	"testing"
	"time"

	identity "github.com/secureprospective/TheWarRoom/internal/accounts"
	"github.com/secureprospective/TheWarRoom/internal/commissioner"
)

func TestAgentTokenPersistenceRecoveryAndAtomicIssuance(t *testing.T) {
	path := filepath.Join(t.TempDir(), "server.db")
	store, pools := openStore(t, path)
	now := time.Date(2026, 10, 24, 12, 0, 0, 0, time.UTC)
	store.SetClock(func() time.Time { return now })
	credential := fixtureCredential(t)
	if err := store.Insert(t.Context(), credential, false); err != nil {
		t.Fatal(err)
	}
	sessionDigest := strings.Repeat("a", 64)
	if _, err := store.CreateSession(t.Context(), credential, sessionDigest, "issuer"); err != nil {
		t.Fatal(err)
	}
	scope := identity.RawTokenScope{League: "native.tokens", Franchise: "franchise.01", Commands: []string{commissioner.NotePost}, Modes: []string{"execute"}}
	authorize := func(account identity.Account) error {
		if account != credential.Account {
			t.Fatal("wrong issuer")
		}
		return nil
	}
	id := strings.Repeat("A", 43)
	digest := strings.Repeat("b", 64)
	if _, err := pools.Write().ExecContext(t.Context(), `CREATE TRIGGER fail_token_audit BEFORE INSERT ON account_security_audit BEGIN SELECT RAISE(ABORT, 'audit unavailable'); END;`); err != nil {
		t.Fatal(err)
	}
	if _, err := store.IssueToken(t.Context(), sessionDigest, digest, id, scope, authorize); err == nil {
		t.Fatal("audit failure accepted token")
	}
	var count int
	if err := pools.Read().QueryRowContext(t.Context(), `SELECT count(*) FROM account_agent_tokens`).Scan(&count); err != nil || count != 0 {
		t.Fatalf("failed issuance persisted: %d %v", count, err)
	}
	if _, err := pools.Write().ExecContext(t.Context(), `DROP TRIGGER fail_token_audit`); err != nil {
		t.Fatal(err)
	}
	deny := func(identity.Account) error { return identity.ErrTokenRefused }
	if _, err := store.IssueToken(t.Context(), sessionDigest, digest, id, scope, deny); !errors.Is(err, identity.ErrTokenRefused) {
		t.Fatalf("denied issuance: %v", err)
	}
	token, err := store.IssueToken(t.Context(), sessionDigest, digest, id, scope, authorize)
	if err != nil {
		t.Fatal(err)
	}
	if err := pools.Close(); err != nil {
		t.Fatal(err)
	}
	store, pools = openStore(t, path)
	store.SetClock(func() time.Time { return now })
	persisted, err := store.Token(t.Context(), digest)
	if err != nil || persisted.ID != token.ID || !persisted.ExpiresAt.Equal(token.ExpiresAt) || persisted.Scope.Commands[0] != commissioner.NotePost {
		t.Fatalf("reopen token: %+v %v", persisted, err)
	}
	if _, err := pools.Write().ExecContext(t.Context(), `CREATE TRIGGER fail_revoke_audit BEFORE INSERT ON account_security_audit BEGIN SELECT RAISE(ABORT, 'audit unavailable'); END;`); err != nil {
		t.Fatal(err)
	}
	if err := store.RevokeToken(t.Context(), sessionDigest, id); err == nil {
		t.Fatal("unaudited revocation succeeded")
	}
	if _, err := store.Token(t.Context(), digest); err != nil {
		t.Fatalf("failed revocation changed token: %v", err)
	}
	if _, err := pools.Write().ExecContext(t.Context(), `DROP TRIGGER fail_revoke_audit`); err != nil {
		t.Fatal(err)
	}
	// Console recovery advances the same durable generation used by both credential types.
	if err := store.ResetAccount(t.Context(), credential.Username, credential.PasswordHash, "", "Exercise token recovery fence"); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Token(t.Context(), digest); !errors.Is(err, identity.ErrTokenRefused) {
		t.Fatalf("recovery revived delegation: %v", err)
	}
	if _, err := store.IssueToken(t.Context(), sessionDigest, strings.Repeat("c", 64), strings.Repeat("B", 42)+"A", scope, authorize); !errors.Is(err, identity.ErrSessionRefused) {
		t.Fatalf("stale issuer session: %v", err)
	}
}
