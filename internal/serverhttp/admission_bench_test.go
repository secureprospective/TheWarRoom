package serverhttp

import (
	"bytes"
	"context"
	"path/filepath"
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/accounts"
	"github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/db"
	"github.com/secureprospective/TheWarRoom/internal/leagueservice"
	"github.com/secureprospective/TheWarRoom/internal/measures"
	accountstore "github.com/secureprospective/TheWarRoom/internal/store/accounts"
	"github.com/secureprospective/TheWarRoom/internal/store/history"
	"github.com/secureprospective/TheWarRoom/tools/hostedfixture"
)

func BenchmarkHostedReadCredentialAdmission(b *testing.B) {
	ctx := b.Context()
	pools, err := db.Open(ctx, filepath.Join(b.TempDir(), "server.db"), history.SchemaVersion())
	if err != nil {
		b.Fatal(err)
	}
	defer func() { _ = pools.Close() }()
	store := history.New(pools, &measures.Registry{})
	if err := store.Initialize(ctx); err != nil {
		b.Fatal(err)
	}
	credentials := accountstore.New(pools)
	raw := accounts.RawCredentials{Username: "commissioner", Password: "private-password"}
	principal, err := hostedfixture.Bootstrap(ctx, credentials, raw)
	if err != nil {
		b.Fatal(err)
	}
	if err := hostedfixture.CreateLeague(ctx, store, principal); err != nil {
		b.Fatal(err)
	}
	passwords := accounts.New(credentials)
	sessions := accounts.NewSessions(passwords, credentials)
	session, err := sessions.Login(ctx, raw, "benchmark")
	if err != nil {
		b.Fatal(err)
	}
	totp, err := accounts.NewTOTP(passwords, credentials, bytes.Repeat([]byte{19}, 32))
	if err != nil {
		b.Fatal(err)
	}
	host := leagueservice.StoredHost{Store: store, Context: func() context.Context { return ctx }, Principal: string(principal)}
	gate := accountAdmission{host: host, sessions: sessions.Admission(session.RawSession), accounts: sessions, totp: totp}
	ctx = commissioner.WithCredentialAdmission(ctx, gate)
	access := commissioner.Access{Principal: string(principal), League: "native.hosted", Operation: "native.assets.read", Mode: "read"}
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		if err := commissioner.AdmitCredential(ctx, access); err != nil {
			b.Fatal(err)
		}
	}
}
