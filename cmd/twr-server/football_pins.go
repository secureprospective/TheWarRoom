package main

import (
	"context"
	"flag"
	"fmt"
	"io"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/measures"
	"github.com/secureprospective/TheWarRoom/internal/serverhttp"
	accountstore "github.com/secureprospective/TheWarRoom/internal/store/accounts"
	"github.com/secureprospective/TheWarRoom/internal/store/history"
)

// leaguePins answers which store serves each league and which football evidence it is pinned to. Each
// league's pin is its row in the league directory (the sign-in file); a league with none uses the
// server's -football pin, as every league did before. Every distinct pin gets its own store over the
// same league file, so changing one league's pin never moves another's. Pins are read at start, before
// any request or due work; a pin is changed with the console's pin-football command and a restart.
type leaguePins struct {
	base     *history.Store
	server   serverhttp.LeaguePin
	byLeague map[string]serverhttp.LeaguePin
}

func newLeaguePins(base *history.Store) *leaguePins {
	return &leaguePins{base: base, server: serverhttp.LeaguePin{Store: base}, byLeague: map[string]serverhttp.LeaguePin{}}
}

// selectPins loads the server pin (digest, may be empty) and every league's own pin from directory.
func (p *leaguePins) selectPins(ctx context.Context, digest string, directory []accountstore.DirectoryEntry) error {
	if digest != "" {
		evidence, err := p.base.LoadFootball(ctx, digest)
		if err != nil {
			return fmt.Errorf("select hosted football: %w", err)
		}
		p.base.SetPinnedFootball(evidence)
		p.server = serverhttp.LeaguePin{Store: p.base, Football: evidence, Digest: digest}
	}
	stores := map[string]serverhttp.LeaguePin{digest: p.server}
	for _, entry := range directory {
		if entry.Football == "" {
			continue
		}
		pin, ok := stores[entry.Football]
		if !ok {
			store := p.base.Sibling()
			evidence, err := store.LoadFootball(ctx, entry.Football)
			if err != nil {
				return fmt.Errorf("select football for league %s: %w", entry.League, err)
			}
			store.SetPinnedFootball(evidence)
			pin = serverhttp.LeaguePin{Store: store, Football: evidence, Digest: entry.Football}
			stores[entry.Football] = pin
		}
		p.byLeague[entry.League] = pin
	}
	return nil
}

// pin is league's store and football; an unpinned league gets the server's.
func (p *leaguePins) pin(league string) serverhttp.LeaguePin {
	if pin, ok := p.byLeague[league]; ok {
		return pin
	}
	return p.server
}

func (p *leaguePins) Leagues(ctx context.Context) ([]commissioner.NativeLeague, error) {
	return p.base.NativeLeagues(ctx) //nolint:wrapcheck // the resolver passes the store's words through
}

func (p *leaguePins) League(_ context.Context, league string) (*history.Store, error) {
	return p.pin(league).Store, nil
}

// Stores is every file holding league history: one league file, whichever pins its leagues use.
func (p *leaguePins) Stores(context.Context) ([]*history.Store, error) {
	return []*history.Store{p.base}, nil
}

func (p *leaguePins) Commits() <-chan string { return history.NewOneFile(p.base).Commits() }

// runPinFootball is the console command that pins one league to football evidence the reference file
// holds (or, with an empty -football, returns it to the server's pin). Stop the service first; the new
// pin is read on the next start. Only that league's directory row changes.
func runPinFootball(ctx context.Context, args []string, output io.Writer) error {
	flags := flag.NewFlagSet("twr-server pin-football", flag.ContinueOnError)
	path := flags.String("db", "", "existing server database")
	league := flags.String("league", "", "league ID (native.…)")
	digest := flags.String("football", "", "football evidence digest the reference file holds; empty uses the server's pin")
	if err := flags.Parse(args); err != nil {
		return fmt.Errorf("parse pin-football flags: %w", err)
	}
	if flags.NArg() != 0 || *league == "" {
		return fmt.Errorf("pin-football needs -db and -league (and -football, or empty for the server's pin)")
	}
	if err := checkHostedDatabase(*path, 0); err != nil {
		return err
	}
	files, err := openStateFiles(ctx, *path)
	if err != nil {
		return err
	}
	defer func() { _ = files.Close() }()
	if *digest != "" {
		registry, err := measures.Embedded()
		if err != nil {
			return fmt.Errorf("load server measures: %w", err)
		}
		if _, err := history.New(files.league, registry).LoadFootball(ctx, *digest); err != nil {
			return fmt.Errorf("the reference file cannot give that football evidence; nothing was changed: %w", err)
		}
	}
	if err := signInStore(files).PinFootball(ctx, *league, *digest); err != nil {
		return fmt.Errorf("pin football: %w", err)
	}
	words := "now uses the server's football pin"
	if *digest != "" {
		words = "is pinned to football evidence " + *digest
	}
	if _, err := fmt.Fprintf(output, "League %s %s. Other leagues are unchanged. Start the server for it to take effect.\n", *league, words); err != nil {
		return fmt.Errorf("report pin: %w", err)
	}
	return nil
}
