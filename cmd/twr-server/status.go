package main

import (
	"context"
	"crypto/x509"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/db"
	"github.com/secureprospective/TheWarRoom/internal/leagueservice"
	"github.com/secureprospective/TheWarRoom/internal/serverstatus"
	accountstore "github.com/secureprospective/TheWarRoom/internal/store/accounts"
	"github.com/secureprospective/TheWarRoom/internal/store/history"
)

func statusSource(pools *db.Pools, league *history.Store, backups backupStatus) serverstatus.Source {
	return serverstatus.Source{Backup: backups, League: league, SignIns: accountstore.New(pools), Now: time.Now}
}

// startStatusSocket serves the private dashboard socket beside the database. It is a
// separate listener from the public HTTPS handler, so the Tunnel can never reach it.
func startStatusSocket(ctx context.Context, path string, source serverstatus.Source) (func(), error) {
	if path == "" {
		return func() {}, nil
	}
	listener, err := serverstatus.Listen(path)
	if err != nil {
		return nil, fmt.Errorf("start status dashboard: %w", err)
	}
	ctx, cancel := context.WithCancel(ctx)
	done := make(chan struct{})
	go func() {
		defer close(done)
		if err := serverstatus.Serve(ctx, listener, source); err != nil {
			log.Printf("status dashboard: %v", err)
		}
	}()
	return func() { cancel(); <-done }, nil
}

// runStatus prints the console dashboard (deploy/server/CHECKLIST.md). It only reads, and
// only contacts the private socket and loopback addresses inside this container.
func runStatus(ctx context.Context, args []string, output io.Writer) error {
	flags := flag.NewFlagSet("twr-server status", flag.ContinueOnError)
	socket := flags.String("socket", "", "the hosted server's private status socket")
	origin := flags.String("origin", "", "loopback IP:port of the league server's TLS listener")
	cert := flags.String("origin-cert", "", "the league server's private origin certificate PEM")
	tunnel := flags.String("tunnel", "", "loopback IP:port of cloudflared's metrics listener")
	state := flags.String("state", "", "folder holding the server database")
	if err := flags.Parse(args); err != nil {
		return fmt.Errorf("parse status flags: %w", err)
	}
	if flags.NArg() != 0 {
		return fmt.Errorf("unexpected status argument")
	}
	pem, err := os.ReadFile(*cert)
	if err != nil {
		return fmt.Errorf("read origin certificate: %w", err)
	}
	roots := x509.NewCertPool()
	if !roots.AppendCertsFromPEM(pem) {
		return fmt.Errorf("origin certificate file holds no certificate")
	}
	console := serverstatus.Console{Socket: *socket, Origin: *origin, OriginCA: roots, Tunnel: *tunnel, State: *state, Space: serverstatus.FreeSpace,
		Clock: leagueservice.NewHostClock().Trusted, Dial: (&net.Dialer{}).DialContext, Now: time.Now}
	checks, err := console.Checks(ctx)
	if err != nil {
		return fmt.Errorf("check server status: %w", err)
	}
	if _, err := serverstatus.Render(output, console.Now(), checks); err != nil {
		return fmt.Errorf("print server status: %w", err)
	}
	return nil
}
