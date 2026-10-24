package main

import (
	"context"
	"crypto/tls"
	"flag"
	"fmt"
	"io"
	"log"
	"os"

	"github.com/secureprospective/TheWarRoom/internal/accounts"
	"github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/db"
	"github.com/secureprospective/TheWarRoom/internal/leagueservice"
	"github.com/secureprospective/TheWarRoom/internal/measures"
	"github.com/secureprospective/TheWarRoom/internal/nativecheck"
	"github.com/secureprospective/TheWarRoom/internal/serverhttp"
	accountstore "github.com/secureprospective/TheWarRoom/internal/store/accounts"
	"github.com/secureprospective/TheWarRoom/internal/store/history"
)

func runHosted(ctx context.Context, args []string, output io.Writer) error {
	flags := flag.NewFlagSet("twr-server hosted", flag.ContinueOnError)
	path := flags.String("db", "", "existing server-owned history and accounts database")
	address := flags.String("listen", "127.0.0.1:0", "literal loopback IP:port; proxy deployment is separate")
	cert := flags.String("tls-cert", "", "TLS certificate PEM")
	key := flags.String("tls-key", "", "TLS private key PEM")
	totpKey := flags.String("totp-key-file", "", "owner-only 32-byte authenticator encryption key")
	registry := flags.String("preview-registry", "", "endpoint-registry.csv")
	if err := flags.Parse(args); err != nil {
		return fmt.Errorf("parse hosted flags: %w", err)
	}
	info, err := os.Stat(*path)
	if err != nil || !info.Mode().IsRegular() || flags.NArg() != 0 {
		return fmt.Errorf("hosted mode requires an existing server database")
	}
	encryptionKey, err := readOfflineKey(*totpKey)
	if err != nil {
		return fmt.Errorf("authenticator key: %w", err)
	}
	catalogue, err := readPreviewRegistry(*registry)
	if err != nil {
		return fmt.Errorf("read preview registry: %w", err)
	}
	certificate, err := tls.LoadX509KeyPair(*cert, *key)
	if err != nil {
		return fmt.Errorf("load server TLS: %w", err)
	}
	pools, err := db.Open(ctx, *path, history.SchemaVersion())
	if err != nil {
		return fmt.Errorf("open server database: %w", err)
	}
	defer func() { _ = pools.Close() }()
	handler, err := hostedServices(ctx, pools, encryptionKey, string(catalogue))
	if err != nil {
		return err
	}

	defer startHostedDueWorker(ctx, handler.Worker)()

	listener, err := serverhttp.ListenLoopback(ctx, *address)
	if err != nil {
		return fmt.Errorf("listen for account HTTPS: %w", err)
	}
	defer func() { _ = listener.Close() }()
	if _, err := fmt.Fprintf(output, "https://%s\n", listener.Addr()); err != nil {
		return fmt.Errorf("report server address: %w", err)
	}
	if err := serverhttp.ServeHosted(ctx, listener, certificate, handler); err != nil {
		return fmt.Errorf("serve account HTTPS: %w", err)
	}
	return nil
}

func startHostedDueWorker(ctx context.Context, worker *leagueservice.DueWorker) func() {
	workerCtx, stop := context.WithCancel(ctx)
	done := make(chan struct{})
	go func() {
		defer close(done)
		worker.Run(workerCtx, func(err error) { log.Printf("hosted due work: %v", err) })
	}()
	return func() { stop(); <-done }
}

func hostedServices(ctx context.Context, pools *db.Pools, encryptionKey []byte, catalogue string) (serverhttp.Hosted, error) {
	registry, err := measures.Embedded()
	if err != nil {
		return serverhttp.Hosted{}, fmt.Errorf("load server measures: %w", err)
	}
	store := history.New(pools, registry)
	if err := store.Initialize(ctx); err != nil {
		return serverhttp.Hosted{}, fmt.Errorf("initialize server history: %w", err)
	}
	credentials := accountstore.New(pools)
	if err := credentials.Initialize(ctx); err != nil {
		return serverhttp.Hosted{}, fmt.Errorf("initialize server accounts: %w", err)
	}
	work := accounts.NewPasswordWork()
	passwords := accounts.NewWithPasswordWork(credentials, work)
	totp, err := accounts.NewTOTP(passwords, credentials, encryptionKey)
	if err != nil {
		return serverhttp.Hosted{}, fmt.Errorf("initialize server authenticators: %w", err)
	}
	clock := leagueservice.NewHostClock()
	store.SetClock(clock.Now)
	worker := leagueservice.NewDueWorker(store, func() nativecheck.Facts { return nil }, clock)
	// HTTPS recovery is by printed code only; the zero digest matches no offline key, so console recovery stays offline.
	handler := serverhttp.Hosted{Store: store, Sessions: accounts.NewSessions(passwords, credentials), TOTP: totp,
		Recovery:   accounts.NewRecovery(passwords, credentials, [32]byte{}),
		Enrollment: accounts.NewEnrollmentWithPasswordWork(credentials, nil, work), Worker: worker, Registry: catalogue}
	return handler, nil
}

func readPreviewRegistry(path string) ([]byte, error) {
	catalogue, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read preview registry: %w", err)
	}
	if _, err := commissioner.ReadPreviewCatalogue(string(catalogue)); err != nil {
		return nil, fmt.Errorf("validate preview registry: %w", err)
	}
	return catalogue, nil
}
