package accounts

import (
	"path/filepath"
	"strconv"
	"testing"
	"time"

	identity "github.com/secureprospective/TheWarRoom/internal/accounts"
)

// SignInFacts reports a confirmed authenticator only, the latest session activity, and repeats the
// account's facts under each linked identity.
func TestSignInFacts(t *testing.T) {
	const id = "12345678-1234-4234-8234-123456789abc"
	seen := time.Date(2026, 10, 30, 12, 0, 0, 0, time.UTC)
	tests := []struct {
		name  string
		setup []string
		want  []identity.SignInFact
	}{
		{"no sign-in yet", nil, []identity.SignInFact{{Principal: "acct." + id}, {Principal: "local.a"}}},
		{"pending authenticator is not on", []string{
			`INSERT INTO account_totp(account_id, sealed_seed, pending_until, confirmed) VALUES ('` + id + `', zeroblob(48), 1, 0)`},
			[]identity.SignInFact{{Principal: "acct." + id}, {Principal: "local.a"}}},
		{"confirmed authenticator and latest session", []string{
			`INSERT INTO account_totp(account_id, sealed_seed, pending_until, confirmed) VALUES ('` + id + `', zeroblob(48), 1, 1)`,
			`INSERT INTO account_sessions VALUES ('` + hex64('a') + `', '` + id + `', 'd', 0, 'e', 1, 5, 0)`,
			`INSERT INTO account_sessions VALUES ('` + hex64('b') + `', '` + id + `', 'd', 0, 'e', 1, ` + itoa(seen.UnixNano()) + `, 1)`},
			[]identity.SignInFact{{Principal: "acct." + id, Authenticator: true, LastSeen: "2026-10-30T12:00:00Z"},
				{Principal: "local.a", Authenticator: true, LastSeen: "2026-10-30T12:00:00Z"}}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			store, pools := openStore(t, filepath.Join(t.TempDir(), "server.db"))
			if err := store.Insert(t.Context(), fixtureCredential(t), false); err != nil {
				t.Fatal(err)
			}
			if err := store.BindAlias(t.Context(), id, "local.a"); err != nil {
				t.Fatal(err)
			}
			for _, q := range tc.setup {
				if _, err := pools.Write().ExecContext(t.Context(), q); err != nil {
					t.Fatal(err)
				}
			}
			got, err := store.SignInFacts(t.Context())
			if err != nil {
				t.Fatal(err)
			}
			if len(got) != len(tc.want) {
				t.Fatalf("got %+v, want %+v", got, tc.want)
			}
			for i := range got {
				if got[i] != tc.want[i] {
					t.Fatalf("row %d: got %+v, want %+v", i, got[i], tc.want[i])
				}
			}
		})
	}
}

func hex64(c byte) string {
	b := make([]byte, 64)
	for i := range b {
		b[i] = c
	}
	return string(b)
}

func itoa(n int64) string { return strconv.FormatInt(n, 10) }
