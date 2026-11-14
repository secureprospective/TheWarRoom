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
	"github.com/secureprospective/TheWarRoom/internal/serverstatus"
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
	if err := recoverPlayers(ctx, files, output); err != nil {
		return err
	}
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
	var routes []serverstatus.Route
	if pins, ok := handler.Leagues.(*leaguePins); ok {
		routes = append(routes, pinRoute(pins, signInStore(files)))
	}
	stopStatus, err := startStatusSocket(ctx, *config.status, statusSource(files.signIns, handler.Leagues, backups), routes...)
	if err != nil {
		return err
	}
	defer stopStatus()

	defer startHostedDueWorker(ctx, handler.Worker)()
	defer startArchivePurge(ctx, files.leagues.root, time.Now, func(line string) { log.Print(line) })()

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
	handler.FootballDigest = pins.server
	return nil
}

// stateFiles are the hosted server's files: server.db (shared, which after the split keeps only the
// record of it), the sign-in file, the football reference file, and one file per league (leagues).
// Each league file is opened with the reference file attached, so league stores read football reference
// data by name. openStateFiles is the one way a command opens them; on an older server it first moves
// the sign-in tables, then the reference tables, then each league, out of server.db.
type stateFiles struct {
	shared, signIns, reference *db.Pools
	leagues                    *leagueFiles
}

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
	if files.shared, err = openLeague(ctx, path, files.reference); err != nil {
		return stateFiles{}, errors.Join(err, files.Close())
	}
	head, err := settleLeagueSplit(ctx, files.shared, files.reference, path)
	if err != nil {
		return stateFiles{}, errors.Join(err, files.Close())
	}
	registry, err := measures.Embedded()
	if err != nil {
		return stateFiles{}, errors.Join(fmt.Errorf("load server measures: %w", err), files.Close())
	}
	if files.leagues, err = openLeagueFiles(ctx, leaguesRoot(path), files.reference, history.New(nil, registry), head); err != nil {
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
	if f.leagues != nil {
		errs = append(errs, f.leagues.Close())
	}
	for _, pools := range []*db.Pools{f.signIns, f.shared, f.reference} {
		if pools != nil {
			errs = append(errs, pools.Close())
		}
	}
	return errors.Join(errs...)
}

// signInStore is the sign-in store over the sign-in file, fenced by every league file's writer.
func signInStore(files stateFiles) *accountstore.Store {
	store := accountstore.NewServerFile(files.signIns)
	store.SetCommitFence(func(ctx context.Context) (func(), error) {
		if files.leagues == nil {
			return func() {}, nil
		}
		return files.leagues.fence(ctx)
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

// leagueStore is league's own store, ready for a console command.
func leagueStore(ctx context.Context, files stateFiles, league string) (*history.Store, error) {
	store, ok := files.leagues.store(league)
	if !ok {
		return nil, fmt.Errorf("league %s is not on this server; nothing was changed", league)
	}
	if err := store.Initialize(ctx); err != nil {
		return nil, fmt.Errorf("initialize league %s: %w", league, err)
	}
	return store, nil
}

// recordLeagues records every league file's leagues in the directory.
func recordLeagues(ctx context.Context, leagues *leagueFiles, signIns *accountstore.Store) error {
	for _, store := range leagues.stores() {
		if err := recordDirectory(ctx, store, signIns); err != nil {
			return err
		}
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
	family := files.leagues.family
	for _, store := range files.leagues.stores() {
		if err := store.Initialize(ctx); err != nil {
			return serverhttp.Hosted{}, fmt.Errorf("initialize server history: %w", err)
		}
	}
	credentials := signInStore(files)
	if err := credentials.Initialize(ctx); err != nil {
		return serverhttp.Hosted{}, fmt.Errorf("initialize server accounts: %w", err)
	}
	if err := recordLeagues(ctx, files.leagues, credentials); err != nil {
		return serverhttp.Hosted{}, err
	}
	work := accounts.NewPasswordWork()
	passwords := accounts.NewWithPasswordWork(credentials, work)
	totp, err := accounts.NewTOTP(passwords, credentials, encryptionKey)
	if err != nil {
		return serverhttp.Hosted{}, fmt.Errorf("initialize server authenticators: %w", err)
	}
	clock := leagueservice.NewHostClock()
	family.SetClock(clock.Now)
	leagues := newLeaguePins(files.leagues)
	worker := leagueservice.NewDueWorker(leagues, func() nativecheck.Facts { return nil }, clock)
	// HTTPS recovery is by printed code only; the zero digest matches no offline key, so console recovery stays offline.
	confirmations, err := serverhttp.NewConfirmations()
	if err != nil {
		return serverhttp.Hosted{}, fmt.Errorf("initialize human confirmation: %w", err)
	}
	// Store is the family: it gives the clock; every league's data comes from its own store (Pins).
	handler := serverhttp.Hosted{Tokens: accounts.NewTokens(credentials, serverhttp.TokenPolicy{Leagues: leagues}), Confirmations: confirmations, Store: family, Leagues: leagues, Pins: leagues.pin, Sessions: accounts.NewSessions(passwords, credentials), TOTP: totp,
		Recovery:   accounts.NewRecovery(passwords, credentials, [32]byte{}),
		Enrollment: accounts.NewEnrollmentWithPasswordWork(credentials, nil, work), Passwords: passwords, Worker: worker, Registry: catalogue,
		Join:    accounts.NewJoin(credentials, leagueservice.JoinSeats{Leagues: leagues}, work),
		Replies: serverhttp.NewReplyCache(replyCacheBytes)}
	if err := stampRelease(ctx, files); err != nil {
		return serverhttp.Hosted{}, err
	}
	return handler, nil
}

// stampRelease records this release in every file the server holds.
func stampRelease(ctx context.Context, files stateFiles) error {
	pools := []*db.Pools{files.shared, files.signIns, files.reference}
	for _, league := range files.leagues.leagues() {
		if p, ok := files.leagues.pools(league); ok {
			pools = append(pools, p)
		}
	}
	for _, p := range pools {
		if err := p.StampServerRelease(ctx, serverversion.Current().Release, time.Now()); err != nil {
			return fmt.Errorf("record server release: %w", err)
		}
	}
	return nil
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

// recoverPlayers puts back, before anything is served, any player a league owns that the football
// reference file lacks, from the league's own restore points. It is a no-op on a healthy server; after a
// restore whose football reference backup was older than its league backup it keeps every owned player's
// permanent id, so nothing later mints a second id for the same player.
func recoverPlayers(ctx context.Context, files stateFiles, output io.Writer) error {
	if files.reference == nil {
		return nil
	}
	for _, league := range files.leagues.leagues() {
		pools, ok := files.leagues.pools(league)
		if !ok {
			continue
		}
		recovered, err := history.RecoverLeagueIdentities(ctx, pools, files.reference)
		if err != nil {
			return fmt.Errorf("put back league %s's players: %w", league, err)
		}
		if recovered.Installed > 0 {
			if _, err := fmt.Fprintf(output, "Put back %d player(s) the football reference file lacked, from league %s's own records.\n", recovered.Installed, league); err != nil {
				return fmt.Errorf("report players: %w", err)
			}
		}
		for _, conflict := range recovered.Conflicts {
			if _, err := fmt.Fprintf(output, "Left out player %s of league %s: another player already holds that outside id. Call Christopher.\n", conflict, league); err != nil {
				return fmt.Errorf("report players: %w", err)
			}
		}
	}
	return nil
}
