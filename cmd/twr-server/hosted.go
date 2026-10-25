package main

import (
	"context"
	"crypto/tls"
	"flag"
	"fmt"
	"io"
	"log"
	"net/netip"
	"os"
	"path/filepath"

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

type hostedFlags struct {
	flags    *flag.FlagSet
	path     *string
	address  *string
	cert     *string
	key      *string
	totpKey  *string
	registry *string
	football *string
	backup   *string
	trusted  *string
	status   *string
}

func parseHostedFlags(args []string) (hostedFlags, error) {
	flags := flag.NewFlagSet("twr-server hosted", flag.ContinueOnError)
	path := flags.String("db", "", "existing server-owned history and accounts database")
	address := flags.String("listen", "127.0.0.1:0", "literal loopback IP:port; proxy deployment is separate")
	cert := flags.String("tls-cert", "", "TLS certificate PEM")
	key := flags.String("tls-key", "", "TLS private key PEM")
	totpKey := flags.String("totp-key-file", "", "owner-only 32-byte authenticator encryption key")
	registry := flags.String("preview-registry", "", "endpoint-registry.csv")
	football := flags.String("football", "", "pinned server-held football evidence digest for owner workflows")
	backup := flags.String("backup-config", "", "private R2 backup settings; required for hosted writes")
	trusted := flags.String("trusted-proxy", "", "loopback IP of a cloudflared hop whose CF-Connecting-IP names each client; empty trusts no proxy")
	status := flags.String("status-socket", "", "owner-only Unix socket for the console status dashboard; empty serves none")
	if err := flags.Parse(args); err != nil {
		return hostedFlags{}, fmt.Errorf("parse hosted flags: %w", err)
	}
	return hostedFlags{flags: flags, path: path, address: address, cert: cert, key: key, totpKey: totpKey, registry: registry, football: football, backup: backup, trusted: trusted, status: status}, nil
}

func runHosted(ctx context.Context, args []string, output io.Writer) error {
	config, err := parseHostedFlags(args)
	if err != nil {
		return err
	}
	if err := checkHostedDatabase(*config.path, config.flags.NArg()); err != nil {
		return err
	}
	encryptionKey, catalogue, certificate, err := hostedFiles(*config.totpKey, *config.registry, *config.cert, *config.key)
	if err != nil {
		return err
	}
	hop, err := trustedProxy(*config.trusted)
	if err != nil {
		return err
	}
	pools, err := db.Open(ctx, *config.path, history.SchemaVersion())
	if err != nil {
		return fmt.Errorf("open server database: %w", err)
	}
	defer func() { _ = pools.Close() }()
	handler, err := hostedServices(ctx, pools, encryptionKey, string(catalogue))
	if err != nil {
		return err
	}
	if err := selectHostedFootball(ctx, &handler, *config.football); err != nil {
		return err
	}
	handler.TrustedProxy = hop
	stopBackup, backups, err := configureHostedBackup(ctx, &handler, pools, *config.backup, filepath.Dir(*config.path), *config.totpKey, *config.cert, *config.key, *config.registry)
	if err != nil {
		return err
	}
	defer stopBackup()
	stopStatus, err := startStatusSocket(ctx, *config.status, statusSource(pools, handler.Store, backups))
	if err != nil {
		return err
	}
	defer stopStatus()

	defer startHostedDueWorker(ctx, handler.Worker)()

	return serveHosted(ctx, *config.address, certificate, handler, output)
}

func checkHostedDatabase(path string, extraArgs int) error {
	info, err := os.Stat(path)
	if err != nil || !info.Mode().IsRegular() || extraArgs != 0 {
		return fmt.Errorf("hosted mode requires an existing server database")
	}
	return nil
}

func selectHostedFootball(ctx context.Context, handler *serverhttp.Hosted, digest string) error {
	if digest == "" {
		return nil
	}
	evidence, err := handler.Store.LoadFootball(ctx, digest)
	if err != nil {
		return fmt.Errorf("select hosted football: %w", err)
	}
	handler.Football, handler.FootballDigest = evidence, digest
	handler.Store.SetPinnedFootball(evidence)
	return nil
}

// hostedFiles reads the authenticator key, preview registry and TLS pair the server is started with.
func hostedFiles(totpKey, registry, cert, key string) ([]byte, []byte, tls.Certificate, error) {
	encryptionKey, err := readOfflineKey(totpKey)
	if err != nil {
		return nil, nil, tls.Certificate{}, fmt.Errorf("authenticator key: %w", err)
	}
	catalogue, err := readPreviewRegistry(registry)
	if err != nil {
		return nil, nil, tls.Certificate{}, fmt.Errorf("read preview registry: %w", err)
	}
	certificate, err := tls.LoadX509KeyPair(cert, key)
	if err != nil {
		return nil, nil, tls.Certificate{}, fmt.Errorf("load server TLS: %w", err)
	}
	return encryptionKey, catalogue, certificate, nil
}

// trustedProxy accepts only a literal loopback address: the tunnel hop must run on this host.
func trustedProxy(flag string) (netip.Addr, error) {
	if flag == "" {
		return netip.Addr{}, nil
	}
	hop, err := netip.ParseAddr(flag)
	if err != nil || !hop.IsLoopback() || hop.Zone() != "" {
		return netip.Addr{}, fmt.Errorf("trusted proxy must be a literal loopback IP")
	}
	return hop, nil
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
	confirmations, err := serverhttp.NewConfirmations()
	if err != nil {
		return serverhttp.Hosted{}, fmt.Errorf("initialize human confirmation: %w", err)
	}
	handler := serverhttp.Hosted{Tokens: accounts.NewTokens(credentials, serverhttp.TokenPolicy{Store: store}), Confirmations: confirmations, Store: store, Sessions: accounts.NewSessions(passwords, credentials), TOTP: totp,
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

func serveHosted(ctx context.Context, address string, certificate tls.Certificate, handler serverhttp.Hosted, output io.Writer) error {
	listener, err := serverhttp.ListenLoopback(ctx, address)
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
