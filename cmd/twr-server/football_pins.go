package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net"
	"net/http"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/measures"
	"github.com/secureprospective/TheWarRoom/internal/serverhttp"
	"github.com/secureprospective/TheWarRoom/internal/serverstatus"
	accountstore "github.com/secureprospective/TheWarRoom/internal/store/accounts"
	"github.com/secureprospective/TheWarRoom/internal/store/history"
)

// leaguePins answers which store serves each league and which football evidence it is pinned to. Each
// league's pin is its row in the league directory (the sign-in file); a league with none uses the
// server's -football pin, as every league did before. Every distinct pin gets its own store over the
// same league file, so changing one league's pin never moves another's. A store's football is set
// before it serves anything and never changed after; a new pin is a new store, swapped in whole.
//
// A pin whose evidence this server does not hold (a restore whose football reference backup was older
// than its league backup) does not stop the server: that league is served by a store with no football,
// and the evidence is looked for again at most once a minute, so the league opens again by itself.
// A running server changes a pin through pin-football on its private socket (setPin).
type leaguePins struct {
	base     *history.Store
	mu       sync.RWMutex
	server   serverhttp.LeaguePin
	byLeague map[string]serverhttp.LeaguePin
	tried    map[string]time.Time
	now      func() time.Time
}

const footballRetry = time.Minute

func newLeaguePins(base *history.Store) *leaguePins {
	return &leaguePins{base: base, server: serverhttp.LeaguePin{Store: base}, byLeague: map[string]serverhttp.LeaguePin{},
		tried: map[string]time.Time{}, now: time.Now}
}

// selectPins loads the server pin (digest, may be empty) and every league's own pin from directory.
func (p *leaguePins) selectPins(ctx context.Context, digest string, directory []accountstore.DirectoryEntry) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if digest != "" {
		pin, err := p.loadPin(ctx, digest, p.base)
		if err != nil {
			return fmt.Errorf("select hosted football: %w", err)
		}
		p.server = pin
	}
	stores := map[string]serverhttp.LeaguePin{digest: p.server}
	for _, entry := range directory {
		if entry.Football == "" {
			continue
		}
		pin, ok := stores[entry.Football]
		if !ok {
			var err error
			if pin, err = p.loadPin(ctx, entry.Football, p.base.Sibling()); err != nil {
				return fmt.Errorf("select football for league %s: %w", entry.League, err)
			}
			stores[entry.Football] = pin
		}
		p.byLeague[entry.League] = pin
	}
	return nil
}

// loadPin pins store to digest's evidence. Evidence the server does not hold gives a store with no
// football, marked missing; any other failure (damaged evidence) is an error.
func (p *leaguePins) loadPin(ctx context.Context, digest string, store *history.Store) (serverhttp.LeaguePin, error) {
	evidence, err := store.LoadFootball(ctx, digest)
	if errors.Is(err, history.ErrNoBody) {
		p.tried[digest] = p.now()
		return serverhttp.LeaguePin{Store: p.base.Sibling(), Digest: digest, Missing: true}, nil
	}
	if err != nil {
		return serverhttp.LeaguePin{}, err //nolint:wrapcheck // the caller names the league
	}
	store.SetPinnedFootball(evidence)
	return serverhttp.LeaguePin{Store: store, Football: evidence, Digest: digest}, nil
}

// pin is league's store and football; an unpinned league gets the server's.
func (p *leaguePins) pin(ctx context.Context, league string) serverhttp.LeaguePin {
	p.mu.RLock()
	pin, ok := p.byLeague[league]
	if !ok {
		pin = p.server
	}
	p.mu.RUnlock()
	if pin.Missing {
		return p.retry(ctx, pin)
	}
	return pin
}

// retry looks for missing evidence again, at most once a minute per version. When it is found, every
// league pinned to it (and the server pin) gets a new store with it.
func (p *leaguePins) retry(ctx context.Context, missing serverhttp.LeaguePin) serverhttp.LeaguePin {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.now().Sub(p.tried[missing.Digest]) < footballRetry {
		return missing
	}
	ctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 10*time.Second)
	defer cancel()
	found, err := p.loadPin(ctx, missing.Digest, p.base.Sibling())
	if err != nil || found.Missing {
		p.tried[missing.Digest] = p.now()
		return missing
	}
	delete(p.tried, missing.Digest)
	for league, pin := range p.byLeague {
		if pin.Digest == missing.Digest {
			p.byLeague[league] = found
		}
	}
	if p.server.Digest == missing.Digest {
		p.server = found
	}
	return found
}

// missingLeagues names each league (and the server pin) whose football this server does not hold.
func (p *leaguePins) missingLeagues() []string {
	p.mu.RLock()
	defer p.mu.RUnlock()
	var out []string
	if p.server.Missing {
		out = append(out, "the server's football pin "+p.server.Digest)
	}
	for league, pin := range p.byLeague {
		if pin.Missing {
			out = append(out, league+" (football version "+pin.Digest+")")
		}
	}
	slices.Sort(out)
	return out
}

// setPin pins league to digest's evidence (empty: back to the server's pin) on a running server. The
// directory row is written first, through the sign-in file's guarded writer, so a change the backups
// cannot yet cover is refused; only then is the league's next request served with the new pin.
// Evidence the reference file does not hold is refused and nothing changes.
func (p *leaguePins) setPin(ctx context.Context, signIns *accountstore.Store, league, digest string) error {
	var next serverhttp.LeaguePin
	if digest != "" {
		store := p.base.Sibling()
		evidence, err := store.LoadFootball(ctx, digest)
		if err != nil {
			return fmt.Errorf("the reference file cannot give that football evidence; nothing was changed: %w", err)
		}
		store.SetPinnedFootball(evidence)
		next = serverhttp.LeaguePin{Store: store, Football: evidence, Digest: digest}
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	if err := signIns.PinFootball(ctx, league, digest); err != nil {
		return fmt.Errorf("pin football: %w", err)
	}
	if digest == "" {
		delete(p.byLeague, league)
	} else {
		p.byLeague[league] = next
	}
	return nil
}

func (p *leaguePins) Leagues(ctx context.Context) ([]commissioner.NativeLeague, error) {
	return p.base.NativeLeagues(ctx) //nolint:wrapcheck // the resolver passes the store's words through
}

func (p *leaguePins) League(ctx context.Context, league string) (*history.Store, error) {
	return p.pin(ctx, league).Store, nil
}

// Stores is every file holding league history: one league file, whichever pins its leagues use.
func (p *leaguePins) Stores(context.Context) ([]*history.Store, error) {
	return []*history.Store{p.base}, nil
}

func (p *leaguePins) Commits() <-chan string { return history.NewOneFile(p.base).Commits() }

// pinFootballPath is the private-socket route a running server changes a league's pin on.
const pinFootballPath = "/pin-football"

type pinRequest struct {
	League   string `json:"league"`
	Football string `json:"football"`
}

// pinRoute serves pin-football on the owner-only status socket.
func pinRoute(pins *leaguePins, signIns *accountstore.Store) serverstatus.Route {
	return serverstatus.Route{Pattern: "POST " + pinFootballPath, Handler: func(w http.ResponseWriter, r *http.Request) {
		var request pinRequest
		if err := json.NewDecoder(io.LimitReader(r.Body, 4096)).Decode(&request); err != nil || request.League == "" {
			http.Error(w, "pin-football needs a league", http.StatusBadRequest)
			return
		}
		if err := pins.setPin(r.Context(), signIns, request.League, request.Football); err != nil {
			http.Error(w, err.Error(), http.StatusConflict)
			return
		}
		_, _ = io.WriteString(w, pinWords(request.League, request.Football, true))
	}}
}

func pinWords(league, digest string, live bool) string {
	words := "now uses the server's football pin"
	if digest != "" {
		words = "is pinned to football evidence " + digest
	}
	when := "Start the server for it to take effect."
	if live {
		when = "The running server uses it from the league's next request."
	}
	return fmt.Sprintf("League %s %s. Other leagues are unchanged. %s\n", league, words, when)
}

// runPinFootball is the console command that pins one league to football evidence the reference file
// holds (or, with an empty -football, returns it to the server's pin). With -status-socket it asks the
// running server, which applies the change at once; with the server stopped it writes the league's
// directory row itself, read on the next start. Only that league's directory row changes.
func runPinFootball(ctx context.Context, args []string, output io.Writer) error {
	flags := flag.NewFlagSet("twr-server pin-football", flag.ContinueOnError)
	path := flags.String("db", "", "existing server database (server stopped)")
	socket := flags.String("status-socket", "", "the running server's status socket (server running)")
	league := flags.String("league", "", "league ID (native.…)")
	digest := flags.String("football", "", "football evidence digest the reference file holds; empty uses the server's pin")
	if err := flags.Parse(args); err != nil {
		return fmt.Errorf("parse pin-football flags: %w", err)
	}
	if flags.NArg() != 0 || *league == "" || (*path == "") == (*socket == "") {
		return fmt.Errorf("pin-football needs -league and one of -status-socket (server running) or -db (server stopped)")
	}
	if *socket != "" {
		return pinRunning(ctx, *socket, pinRequest{League: *league, Football: *digest}, output)
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
	if _, err := io.WriteString(output, pinWords(*league, *digest, false)); err != nil {
		return fmt.Errorf("report pin: %w", err)
	}
	return nil
}

func pinRunning(ctx context.Context, socket string, request pinRequest, output io.Writer) error {
	body, err := json.Marshal(request)
	if err != nil {
		return fmt.Errorf("encode pin: %w", err)
	}
	client := &http.Client{Timeout: time.Minute, Transport: &http.Transport{DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
		return (&net.Dialer{}).DialContext(ctx, "unix", socket)
	}}}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "http://twr-server"+pinFootballPath, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("pin football: %w", err)
	}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("reach the running server on %s (is it running with -status-socket?): %w", socket, err)
	}
	defer func() { _ = resp.Body.Close() }()
	words, err := io.ReadAll(io.LimitReader(resp.Body, 4096))
	if err != nil {
		return fmt.Errorf("read the server's answer: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return errors.New(strings.TrimSpace(string(words)))
	}
	if _, err := output.Write(words); err != nil {
		return fmt.Errorf("report pin: %w", err)
	}
	return nil
}
