package main

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/accounts"
	"github.com/secureprospective/TheWarRoom/internal/db"
	"github.com/secureprospective/TheWarRoom/internal/leagueservice"
	"github.com/secureprospective/TheWarRoom/internal/serverhttp"
	accountstore "github.com/secureprospective/TheWarRoom/internal/store/accounts"
	"github.com/secureprospective/TheWarRoom/internal/store/history"
)

func assertHostedBinary(ctx context.Context, t *testing.T, binary, cert, key string, roots *x509.CertPool) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "server.db")
	pools, err := db.Open(ctx, path, history.SchemaVersion())
	if err != nil {
		t.Fatal(err)
	}
	store := accountstore.New(pools)
	if err := store.Initialize(ctx); err != nil {
		t.Fatal(err)
	}
	if _, err := accounts.New(store).LocalBootstrap(ctx, accounts.RawCredentials{Username: "local.admin", Password: "private-admin-password"}); err != nil {
		t.Fatal(err)
	}
	if err := pools.Close(); err != nil {
		t.Fatal(err)
	}
	keyFile := filepath.Join(t.TempDir(), "totp.key")
	if err := os.WriteFile(keyFile, []byte(strings.Repeat("k", 32)), 0600); err != nil {
		t.Fatal(err)
	}
	registry, err := filepath.Abs("../../docs/ui/endpoint-registry.csv")
	if err != nil {
		t.Fatal(err)
	}
	origin := startServerBinary(ctx, t, binary, "hosted", "-db", path, "-tls-cert", cert, "-tls-key", key, "-totp-key-file", keyFile, "-preview-registry", registry)
	transport := &http.Transport{TLSClientConfig: &tls.Config{MinVersion: tls.VersionTLS13, RootCAs: roots}}
	defer transport.CloseIdleConnections()
	client := &http.Client{Transport: transport}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, origin+serverhttp.LoginPath, strings.NewReader(`{"protocol":1,"username":"local.admin","password":"private-admin-password","device":"real process"}`))
	if err != nil {
		t.Fatal(err)
	}
	response, err := client.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	body, err := io.ReadAll(response.Body)
	_ = response.Body.Close()
	if err != nil || response.StatusCode != 200 {
		t.Fatalf("real hosted login: %d %s %v", response.StatusCode, body, err)
	}
	var session serverhttp.LoginResponse
	if err := json.Unmarshal(body, &session); err != nil || session.Bearer == "" {
		t.Fatalf("real session: %v", err)
	}
	for _, path := range []string{serverhttp.LedgerPath, serverhttp.PreviewPath, serverhttp.ExecutePath, serverhttp.FixturePath, serverhttp.FixtureChangePath} {
		request, err := http.NewRequestWithContext(ctx, http.MethodPost, origin+path, strings.NewReader(`{"protocol":1,"league":"native.https-fixture"}`))
		if err != nil {
			t.Fatal(err)
		}
		request.Header.Set("Authorization", "Bearer "+session.Bearer)
		response, err := client.Do(request)
		if err != nil {
			t.Fatal(err)
		}
		_ = response.Body.Close()
		if response.StatusCode < 400 {
			t.Fatalf("host bootstrap acquired fixture/league authority: %s", path)
		}
	}
}

// Hosted admission must use the kernel-backed HostClock, never a fixture or always-trusted clock.
func TestHostedUsesHostClock(t *testing.T) {
	pools, err := db.Open(t.Context(), filepath.Join(t.TempDir(), "server.db"), history.SchemaVersion())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = pools.Close() })
	handler, err := hostedServices(t.Context(), pools, []byte(strings.Repeat("k", 32)), "")
	if err != nil {
		t.Fatal(err)
	}
	clock := reflect.ValueOf(handler.Worker).Elem().FieldByName("clock")
	if clock.IsNil() || clock.Elem().Type() != reflect.TypeFor[leagueservice.HostClock]() ||
		clock.Elem().FieldByName("synchronized").IsNil() {
		t.Fatalf("hosted due worker clock is %v, not a sourced HostClock", clock)
	}
}
