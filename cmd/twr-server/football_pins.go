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
	"github.com/secureprospective/TheWarRoom/internal/serverhttp"
	"github.com/secureprospective/TheWarRoom/internal/serverstatus"
	accountstore "github.com/secureprospective/TheWarRoom/internal/store/accounts"
	"github.com/secureprospective/TheWarRoom/internal/store/history"
)

// leaguePins answers which store serves each league and which football evidence it is pinned to. Each
// league has its own file (leagueFiles) and is served by its own store over it. Its pin is its row in
// the league directory (the sign-in file); a league with none uses the server's -football pin. A
// store's football is set before it serves anything and never changed after; a new pin is a new store
// over the same league file, swapped in whole, so changing one league's pin never moves another's.
//
// A pin whose evidence this server does not hold (a restore whose football reference backup was older
// than its league backup) does not stop the server: that league is served by a store with no football,
// and the evidence is looked for again at most once a minute, so the league opens again by itself.
// A running server changes a pin through pin-football on its private socket (setPin).
type leaguePins struct {
	files  *leagueFiles
	mu     sync.RWMutex
	server string            // the server's -football digest; empty pins nothing
	chosen map[string]string // each league's own pin from the directory
	built  map[string]serverhttp.LeaguePin
	tried  map[string]time.Time
	now    func() time.Time
}

const footballRetry = time.Minute

func newLeaguePins(files *leagueFiles) *leaguePins {
	return &leaguePins{files: files, chosen: map[string]string{}, built: map[string]serverhttp.LeaguePin{},
		tried: map[string]time.Time{}, now: time.Now}
}

// selectPins records the server pin (digest, may be empty) and every league's own pin from directory,
// and builds each league's store. A league whose pin cannot be read at all (damaged evidence) stops
// the start; evidence the server does not hold only marks that league missing.
func (p *leaguePins) selectPins(ctx context.Context, digest string, directory []accountstore.DirectoryEntry) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.server = digest
	p.chosen = map[string]string{}
	p.built = map[string]serverhttp.LeaguePin{}
	for _, entry := range directory {
		if entry.Football != "" {
			p.chosen[entry.League] = entry.Football
		}
	}
	for _, league := range p.files.leagues() {
		pin := p.buildLocked(ctx, league)
		if pin.Err != nil {
			return fmt.Errorf("select football for league %s: %w", league, pin.Err)
		}
	}
	return nil
}

func (p *leaguePins) digestLocked(league string) string {
	if digest, ok := p.chosen[league]; ok {
		return digest
	}
	return p.server
}

// buildLocked makes league's store for its pin and keeps it.
func (p *leaguePins) buildLocked(ctx context.Context, league string) serverhttp.LeaguePin {
	base, ok := p.files.store(league)
	if !ok {
		return serverhttp.LeaguePin{Err: errNoLeague}
	}
	pin := p.loadPin(ctx, base, p.digestLocked(league))
	if pin.Err == nil {
		p.built[league] = pin
	}
	return pin
}

// loadPin pins a new store over base's league file to digest's evidence. Evidence the server does not
// hold gives a store with no football, marked missing; any other failure (damaged evidence) is Err.
func (p *leaguePins) loadPin(ctx context.Context, base *history.Store, digest string) serverhttp.LeaguePin {
	store := base.Sibling()
	if digest == "" {
		return serverhttp.LeaguePin{Store: store}
	}
	evidence, err := store.LoadFootball(ctx, digest)
	if errors.Is(err, history.ErrNoBody) {
		p.tried[digest] = p.now()
		return serverhttp.LeaguePin{Store: store, Digest: digest, Missing: true}
	}
	if err != nil {
		return serverhttp.LeaguePin{Err: err}
	}
	store.SetPinnedFootball(evidence)
	return serverhttp.LeaguePin{Store: store, Football: evidence, Digest: digest}
}

// pin is league's store and football; Err says the server does not hold league.
func (p *leaguePins) pin(ctx context.Context, league string) serverhttp.LeaguePin {
	p.mu.RLock()
	pin, ok := p.built[league]
	p.mu.RUnlock()
	if ok && !pin.Missing {
		return pin
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	pin, ok = p.built[league]
	switch {
	case !ok:
		// A league created after the start (league-load, a received move) gets its store on first use.
		return p.buildLocked(ctx, league)
	case pin.Missing && p.now().Sub(p.tried[pin.Digest]) >= footballRetry:
		return p.retryLocked(ctx, league, pin)
	}
	return pin
}

// retryLocked looks for league's missing evidence again (at most once a minute per version). Other
// leagues on the same version find it at their own next look.
func (p *leaguePins) retryLocked(ctx context.Context, league string, missing serverhttp.LeaguePin) serverhttp.LeaguePin {
	look, cancel := context.WithTimeout(context.WithoutCancel(ctx), 10*time.Second)
	defer cancel()
	base, ok := p.files.store(league)
	if !ok {
		return missing
	}
	found := p.loadPin(look, base, missing.Digest)
	if found.Err != nil || found.Missing {
		p.tried[missing.Digest] = p.now()
		return missing
	}
	for other, pin := range p.built {
		if pin.Missing && pin.Digest == missing.Digest && other != league {
			delete(p.tried, missing.Digest) // the others look again at once
		}
	}
	p.built[league] = found
	return found
}

// missingLeagues names each league whose football this server does not hold.
func (p *leaguePins) missingLeagues() []string {
	p.mu.RLock()
	defer p.mu.RUnlock()
	var out []string
	for league, pin := range p.built {
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
	base, ok := p.files.store(league)
	if !ok {
		return fmt.Errorf("league %s is not on this server; nothing was changed", league)
	}
	p.mu.RLock()
	effective := digest
	if effective == "" {
		effective = p.server
	}
	p.mu.RUnlock()
	next := p.loadPin(ctx, base, effective)
	if next.Err != nil || (digest != "" && next.Missing) {
		return fmt.Errorf("the reference file cannot give that football evidence; nothing was changed: %w", errors.Join(next.Err, missingErr(next)))
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	if err := signIns.PinFootball(ctx, league, digest); err != nil {
		return fmt.Errorf("pin football: %w", err)
	}
	if digest == "" {
		delete(p.chosen, league)
	} else {
		p.chosen[league] = digest
	}
	p.built[league] = next
	return nil
}

func missingErr(pin serverhttp.LeaguePin) error {
	if pin.Missing {
		return history.ErrNoBody
	}
	return nil
}

func (p *leaguePins) Leagues(ctx context.Context) ([]commissioner.NativeLeague, error) {
	var out []commissioner.NativeLeague
	for _, store := range p.files.stores() {
		leagues, err := store.NativeLeagues(ctx)
		if err != nil {
			return nil, err //nolint:wrapcheck // the resolver passes the store's words through
		}
		out = append(out, leagues...)
	}
	return out, nil
}

func (p *leaguePins) League(ctx context.Context, league string) (*history.Store, error) {
	pin := p.pin(ctx, league)
	return pin.Store, pin.Err
}

// Stores is every league's own file.
func (p *leaguePins) Stores(context.Context) ([]*history.Store, error) { return p.files.stores(), nil }

func (p *leaguePins) Commits() <-chan string { return history.NewOneFile(p.files.family).Commits() }

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
		store, err := leagueStore(ctx, files, *league)
		if err != nil {
			return err
		}
		if _, err := store.LoadFootball(ctx, *digest); err != nil {
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
