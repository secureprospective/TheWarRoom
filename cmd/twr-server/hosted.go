package main

import (
	"context"
	"crypto/tls"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"net/netip"
	"os"
	"path/filepath"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/accounts"
	"github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/db"
	"github.com/secureprospective/TheWarRoom/internal/leagueservice"
	"github.com/secureprospective/TheWarRoom/internal/measures"
	"github.com/secureprospective/TheWarRoom/internal/nativecheck"
	"github.com/secureprospective/TheWarRoom/internal/serverhttp"
	"github.com/secureprospective/TheWarRoom/internal/serverversion"
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

// replyCacheBytes bounds the league-state replies the server keeps in memory (serverhttp.ReplyCache).
const replyCacheBytes = 256 << 20

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
	files, err := openStateFiles(ctx, *config.path)
	if err != nil {
		return err
	}
	defer func() { _ = files.Close() }()
	handler, err := hostedServices(ctx, files, encryptionKey, string(catalogue))
	if err != nil {
		return err
	}
	if err := selectHostedFootball(ctx, &handler, signInStore(files), *config.football); err != nil {
		return err
	}
	handler.TrustedProxy = hop
	stopBackup, backups, err := configureHostedBackup(ctx, &handler, files, *config.backup, filepath.Dir(*config.path), *config.totpKey, *config.cert, *config.key, *config.registry)
	if err != nil {
		return err
	}
	defer stopBackup()
	stopStatus, err := startStatusSocket(ctx, *config.status, statusSource(files.signIns, handler.Leagues, backups))
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

// selectHostedFootball loads the server's pin and every league's own pin from the league directory.
func selectHostedFootball(ctx context.Context, handler *serverhttp.Hosted, signIns *accountstore.Store, digest string) error {
	pins, ok := handler.Leagues.(*leaguePins)
	if !ok {
		return fmt.Errorf("select hosted football: the league resolver holds no pins")
	}
	directory, err := signIns.Directory(ctx)
	if err != nil {
		return fmt.Errorf("select hosted football: %w", err)
	}
	if err := pins.selectPins(ctx, digest, directory); err != nil {
		return err
	}
	handler.Football, handler.FootballDigest = pins.server.Football, pins.server.Digest
	return nil
}

// stateFiles are the hosted server's league file, its sign-in file and its reference file. The league
// file is opened with the reference file attached, so league stores read football reference data by
// name. openStateFiles is the one way a command opens them; it moves the sign-in tables and then the
// reference tables out of a one-file server first.
type stateFiles struct{ league, signIns, reference *db.Pools }

func openStateFiles(ctx context.Context, path string) (stateFiles, error) {
	if err := servePreflight(ctx, path); err != nil {
		return stateFiles{}, fmt.Errorf("open server database: %w", err)
	}
	return openSignInFiles(ctx, path)
}

// openSignInFiles is openStateFiles for the console's incident commands, which must work on a league
// file of any release this build can open: it skips the serve preflight.
func openSignInFiles(ctx context.Context, path string) (stateFiles, error) {
	alone, err := db.Open(ctx, path, history.LeagueSchemaVersion())
	if err != nil {
		return stateFiles{}, fmt.Errorf("open server database: %w", err)
	}
	signIns, err := openSignIns(ctx, alone, path)
	if err != nil {
		return stateFiles{}, errors.Join(err, alone.Close())
	}
	files := stateFiles{signIns: signIns}
	files.reference, err = openReference(ctx, alone, path)
	if err = errors.Join(err, alone.Close()); err != nil {
		return stateFiles{}, errors.Join(err, files.Close())
	}
	if files.league, err = openLeague(ctx, path, files.reference); err != nil {
		return stateFiles{}, errors.Join(err, files.Close())
	}
	return files, nil
}

// openLeague opens a converted league file with its reference file attached.
func openLeague(ctx context.Context, path string, reference *db.Pools) (*db.Pools, error) {
	league, err := db.OpenWithReference(ctx, path, history.LeagueSchemaVersion(), reference)
	if err != nil {
		return nil, fmt.Errorf("open server database: %w", err)
	}
	return league, nil
}

func (f stateFiles) Close() error {
	var errs []error
	for _, pools := range []*db.Pools{f.signIns, f.league, f.reference} {
		if pools != nil {
			errs = append(errs, pools.Close())
		}
	}
	return errors.Join(errs...)
}

// signInStore is the sign-in store over the sign-in file, fenced by the league writer.
func signInStore(files stateFiles) *accountstore.Store {
	store := accountstore.NewServerFile(files.signIns)
	store.SetCommitFence(func(ctx context.Context) (func(), error) {
		conn, err := files.league.Write().Conn(ctx)
		if err != nil {
			return nil, fmt.Errorf("reserve the league writer: %w", err)
		}
		return func() { _ = conn.Close() }, nil
	})
	return store
}

// recordDirectory copies the league file's leagues into the sign-in file's directory. It runs after the
// league file has committed them, so a directory row never names a league that does not exist.
func recordDirectory(ctx context.Context, ledger *history.Store, signIns *accountstore.Store) error {
	states, err := ledger.LeagueStates(ctx)
	if err != nil {
		return fmt.Errorf("read the leagues on this server: %w", err)
	}
	entries := make([]accountstore.DirectoryEntry, len(states))
	for i, l := range states {
		entries[i] = accountstore.DirectoryEntry{League: l.League, State: l.State, Created: l.Created}
	}
	if err := signIns.RecordLeagues(ctx, entries); err != nil {
		return fmt.Errorf("record the leagues on this server: %w", err)
	}
	return nil
}

// servePreflight refuses, before the file is opened for writing, a league file this release may
// not serve as it is: newer, damaged, or older and not yet backed up and upgraded.
func servePreflight(ctx context.Context, path string) error {
	plan, err := serverversion.Preflight(ctx, path)
	if err != nil {
		return fmt.Errorf("check league file: %w", err)
	}
	if plan.State == serverversion.Older {
		return fmt.Errorf("check league file: %w", serverversion.ErrOlderFile)
	}
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

func hostedServices(ctx context.Context, files stateFiles, encryptionKey []byte, catalogue string) (serverhttp.Hosted, error) {
	registry, err := measures.Embedded()
	if err != nil {
		return serverhttp.Hosted{}, fmt.Errorf("load server measures: %w", err)
	}
	store := history.New(files.league, registry)
	if err := store.Initialize(ctx); err != nil {
		return serverhttp.Hosted{}, fmt.Errorf("initialize server history: %w", err)
	}
	credentials := signInStore(files)
	if err := credentials.Initialize(ctx); err != nil {
		return serverhttp.Hosted{}, fmt.Errorf("initialize server accounts: %w", err)
	}
	if err := recordDirectory(ctx, store, credentials); err != nil {
		return serverhttp.Hosted{}, err
	}
	work := accounts.NewPasswordWork()
	passwords := accounts.NewWithPasswordWork(credentials, work)
	totp, err := accounts.NewTOTP(passwords, credentials, encryptionKey)
	if err != nil {
		return serverhttp.Hosted{}, fmt.Errorf("initialize server authenticators: %w", err)
	}
	clock := leagueservice.NewHostClock()
	store.SetClock(clock.Now)
	leagues := newLeaguePins(store)
	worker := leagueservice.NewDueWorker(leagues, func() nativecheck.Facts { return nil }, clock)
	// HTTPS recovery is by printed code only; the zero digest matches no offline key, so console recovery stays offline.
	confirmations, err := serverhttp.NewConfirmations()
	if err != nil {
		return serverhttp.Hosted{}, fmt.Errorf("initialize human confirmation: %w", err)
	}
	handler := serverhttp.Hosted{Tokens: accounts.NewTokens(credentials, serverhttp.TokenPolicy{Store: store}), Confirmations: confirmations, Store: store, Leagues: leagues, Pins: leagues.pin, Sessions: accounts.NewSessions(passwords, credentials), TOTP: totp,
		Recovery:   accounts.NewRecovery(passwords, credentials, [32]byte{}),
		Enrollment: accounts.NewEnrollmentWithPasswordWork(credentials, nil, work), Passwords: passwords, Worker: worker, Registry: catalogue,
		Join:    accounts.NewJoin(credentials, leagueservice.JoinSeats{Store: store}, work),
		Replies: serverhttp.NewReplyCache(replyCacheBytes)}
	for _, pools := range []*db.Pools{files.league, files.signIns, files.reference} {
		if err := pools.StampServerRelease(ctx, serverversion.Current().Release, time.Now()); err != nil {
			return serverhttp.Hosted{}, fmt.Errorf("record server release: %w", err)
		}
	}
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
