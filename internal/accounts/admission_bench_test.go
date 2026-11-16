package accounts_test

import (
	"path/filepath"
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/accounts"
	"github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/db"
	accountstore "github.com/secureprospective/TheWarRoom/internal/store/accounts"
)

func BenchmarkReadSessionAdmission(b *testing.B) {
	ctx := b.Context()
	pools, err := db.Open(ctx, filepath.Join(b.TempDir(), "accounts.db"))
	if err != nil {
		b.Fatal(err)
	}
	defer func() { _ = pools.Close() }()
	store := accountstore.New(pools)
	if err := store.Initialize(ctx); err != nil {
		b.Fatal(err)
	}
	credentials := accounts.RawCredentials{Username: "commissioner", Password: "private-authenticator-password"}
	account, err := accounts.New(store).LocalBootstrap(ctx, credentials)
	if err != nil {
		b.Fatal(err)
	}
	principal, err := account.Principal()
	if err != nil {
		b.Fatal(err)
	}
	sessions := accounts.NewSessions(accounts.New(store), store)
	session, err := sessions.Login(ctx, credentials, "benchmark")
	if err != nil {
		b.Fatal(err)
	}
	gate := sessions.Admission(session.RawSession)
	access := commissioner.Access{Principal: string(principal), Mode: "read"}
	ctx = commissioner.WithCredentialAdmission(ctx, gate)
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		if err := commissioner.AdmitCredential(ctx, access); err != nil {
			b.Fatal(err)
		}
	}
}
