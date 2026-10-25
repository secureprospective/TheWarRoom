// Command twr-server serves account-authenticated leagues or isolated development fixtures.
package main

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/db"
	"github.com/secureprospective/TheWarRoom/internal/leagueservice"
	"github.com/secureprospective/TheWarRoom/internal/measures"
	"github.com/secureprospective/TheWarRoom/internal/nativecheck"
	"github.com/secureprospective/TheWarRoom/internal/serverclient"
	"github.com/secureprospective/TheWarRoom/internal/serverhttp"
	"github.com/secureprospective/TheWarRoom/internal/store/history"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	err := run(ctx, os.Args[1:], os.Stdout)
	stop()
	if err != nil {
		log.Fatal(err)
	}
}

// subcommand runs an operator mode and reports whether name is one. Console commands read private input from stdin.
func subcommand(ctx context.Context, name string, args []string, output io.Writer) (bool, error) {
	switch name {
	case "backup-check":
		return true, runBackupCheck(args, output)
	case "hosted":
		return true, runHosted(ctx, args, output)
	case "checkpoint":
		return true, runCheckpoint(ctx, args, output)
	case "recover":
		return true, runRecovery(ctx, args, os.Stdin, output)
	case "restore":
		return true, runRestore(ctx, args, os.Stdin, output)
	case "reset-sign-ins":
		return true, runResetSignIns(ctx, args, os.Stdin, output)
	case "upgrade":
		return true, runUpgrade(ctx, args, output)
	case "version":
		return true, runVersion(args, output)
	case "status":
		return true, runStatus(ctx, args, output)
	}
	return false, nil
}

func run(ctx context.Context, args []string, output io.Writer) error {
	if len(args) > 0 {
		if ok, err := subcommand(ctx, args[0], args[1:], output); ok {
			return err
		}
	}
	flags := flag.NewFlagSet("twr-server", flag.ContinueOnError)
	fixture := flags.Bool("fixture", false, "required: use disposable built-in fixture only")
	changes := flags.Bool("fixture-changes", false, "enable one fixture mutation in a fresh disposable database")
	address := flags.String("listen", "127.0.0.1:0", "literal loopback IP:port")
	cert := flags.String("tls-cert", "", "TLS certificate PEM")
	key := flags.String("tls-key", "", "TLS private key PEM")
	read := flags.String("read", "", "read fixture from this loopback HTTPS origin instead of serving")
	ca := flags.String("ca", "", "trusted CA PEM for -read")
	if err := flags.Parse(args); err != nil {
		return fmt.Errorf("parse flags: %w", err)
	}
	if !*fixture || flags.NArg() != 0 {
		return fmt.Errorf("explicit -fixture mode is required; no live league access exists")
	}
	if *read != "" {
		return readFixture(ctx, *read, *ca, output)
	}
	certificate, err := tls.LoadX509KeyPair(*cert, *key)
	if err != nil {
		return fmt.Errorf("load fixture TLS certificate: %w", err)
	}
	listener, err := serverhttp.ListenLoopback(ctx, *address)
	if err != nil {
		return fmt.Errorf("fixture operation: %w", err)
	}
	defer listener.Close()
	if _, err := fmt.Fprintf(output, "https://%s\n", listener.Addr()); err != nil {
		return fmt.Errorf("report fixture address: %w", err)
	}
	if *changes {
		return serveChanges(ctx, listener, certificate)
	}
	if err := serverhttp.ServeFixture(ctx, listener, certificate); err != nil {
		return fmt.Errorf("run fixture server: %w", err)
	}
	return nil
}

func readFixture(ctx context.Context, address, caPath string, output io.Writer) error {
	pem, err := os.ReadFile(caPath)
	if err != nil {
		return fmt.Errorf("read trusted CA: %w", err)
	}
	roots := x509.NewCertPool()
	if !roots.AppendCertsFromPEM(pem) {
		return fmt.Errorf("trusted CA file contains no certificates")
	}
	client, err := serverclient.NewFixture(address, roots)
	if err != nil {
		return fmt.Errorf("fixture operation: %w", err)
	}
	defer client.Close()
	ledger, err := client.FixtureLedger(ctx)
	if err != nil {
		return fmt.Errorf("fixture operation: %w", err)
	}
	if err := json.NewEncoder(output).Encode(ledger); err != nil {
		return fmt.Errorf("print fixture ledger: %w", err)
	}
	return nil
}

func serveChanges(ctx context.Context, listener net.Listener, certificate tls.Certificate) error {
	dir, err := os.MkdirTemp("", "twr-loopback-fixture-")
	if err != nil {
		return fmt.Errorf("create fixture state: %w", err)
	}
	defer func() {
		if err := os.RemoveAll(dir); err != nil {
			log.Printf("remove fixture state: %v", err)
		}
	}()
	pools, err := db.Open(ctx, filepath.Join(dir, "history.db"), history.SchemaVersion())
	if err != nil {
		return fmt.Errorf("open fixture state: %w", err)
	}
	defer func() {
		if err := pools.Close(); err != nil {
			log.Printf("close fixture state: %v", err)
		}
	}()
	store := history.New(pools, &measures.Registry{})
	if err := store.Initialize(ctx); err != nil {
		return fmt.Errorf("initialize fixture state: %w", err)
	}
	records, err := leagueservice.FixtureRecords()
	if err != nil {
		return fmt.Errorf("prepare fixture state: %w", err)
	}
	if _, err := store.ExecuteCommissioner(ctx, records[0].Command, "fixture.creator", nil); err != nil {
		return fmt.Errorf("seed fixture state: %w", err)
	}
	if err := serveChangesWorker(ctx, listener, certificate, store); err != nil {
		return fmt.Errorf("serve fixture changes: %w", err)
	}
	return nil
}

// The disposable fixture deliberately trusts its wall clock. Deployable mode must
// supply real clock-health evidence; fixture trust is not a production health check.
type fixtureDueClock struct{}

func (fixtureDueClock) Now() time.Time { return time.Now() }
func (fixtureDueClock) Trusted() error { return nil }

func serveChangesWorker(ctx context.Context, listener net.Listener, certificate tls.Certificate, store *history.Store) error {
	worker := leagueservice.NewDueWorker(store, func() nativecheck.Facts { return nil }, fixtureDueClock{})
	ctx, cancel := context.WithCancel(ctx)
	done := make(chan struct{})
	go func() {
		defer close(done)
		worker.Run(ctx, func(err error) { log.Printf("fixture due worker: %v", err) })
	}()
	defer func() { cancel(); <-done }()
	//nolint:wrapcheck // The caller adds the fixture server operation context.
	return serverhttp.ServeFixtureChanges(ctx, listener, certificate, leagueservice.NewFixtureChangesWithWorker(store, worker))
}
