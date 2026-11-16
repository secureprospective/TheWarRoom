package main

import (
	"context"
	"fmt"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/app/sources"
	"github.com/secureprospective/TheWarRoom/internal/archive"
	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/ingestion"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/players"
	"github.com/secureprospective/TheWarRoom/internal/leaguemigration/mflsource"
	"github.com/secureprospective/TheWarRoom/internal/mfl"
	"github.com/secureprospective/TheWarRoom/internal/normalize"
)

const playersRetry = time.Hour

// playerListings reads each seeded league's reference scope, never inferring it from its native ID.
// The reference file owns one player record, so all scopes contribute to a single listing.
func (k *signalsKeeper) playerListings(ctx context.Context) serverListings {
	var listings serverListings
	for _, name := range k.values.pins.files.leagues() {
		records, err := k.values.leagueRecords(ctx, name)
		if err != nil {
			k.logf("server values: players for %s: %v", name, err)
			continue
		}
		league, season, _, ok := mflsource.ReferenceLeague(records)
		if !ok {
			continue
		}
		listing, held, err := k.players(ctx, league, season)
		if err != nil {
			k.logf("server values: players for %s: %v; keeping the last listing", name, err)
		}
		if held {
			listings = append(listings, listing)
		}
	}
	return listings
}

// players replays the last valid body before considering a network read. Both daily freshness
// and failed-attempt backoff live in the archive, so restarting cannot spend the quota again.
func (k *signalsKeeper) players(ctx context.Context, league string, season int) (normalize.Lookup, bool, error) {
	held, at, found, err := k.archivedPlayers(ctx, league, season)
	if err != nil {
		return held, found, err
	}
	if found && k.now().Before(at.Add(signalsRefresh)) {
		return held, true, nil
	}
	attempt, err := k.playersAttempt(ctx, league, season)
	if err != nil || k.now().Before(attempt.Add(playersRetry)) {
		return held, found, err
	}
	if err := k.openPlayers(); err != nil {
		return held, found, err
	}
	lookup, err := sources.Directory(ctx, normalize.Lookup{}, false,
		func(ctx context.Context) ([]players.RawPlayer, error) {
			return players.Fetch(ctx, k.listing, strconv.Itoa(season), league)
		}, func(normalize.Lookup, time.Time) {})
	if err != nil {
		return held, found, fmt.Errorf("read MFL players: %w", err)
	}
	return lookup, true, nil
}

func (k *signalsKeeper) openPlayers() error {
	if k.listing != nil {
		return nil
	}
	client, err := mfl.New("", 2, mfl.WithoutRetries(), mfl.WithTransport(&archive.Transport{
		Base: k.client.Transport, Sink: k.values.reference, Judge: ingestion.MFLRefusal,
	}))
	if err != nil {
		return fmt.Errorf("open MFL players reader: %w", err)
	}
	k.listing = client
	return nil
}

func (k *signalsKeeper) archivedPlayers(ctx context.Context, league string, season int) (normalize.Lookup, time.Time, bool, error) {
	var lookup normalize.Lookup
	var at time.Time
	found, err := k.values.reference.ArchivedBodies(ctx, "TYPE=players", func(src string, body []byte, fetched time.Time) bool {
		if !playersScope(src, league, season, "players") {
			return false
		}
		raw, err := players.Parse(ctx, body)
		if err != nil {
			return false
		}
		lookup, err = normalize.NewLookup(raw)
		if err != nil {
			return false
		}
		at = fetched
		return true
	})
	if err != nil {
		return lookup, at, found, fmt.Errorf("read archived MFL players: %w", err)
	}
	return lookup, at, found, nil
}

// Discovery failures count too: they must not become a restart-driven request loop.
func (k *signalsKeeper) playersAttempt(ctx context.Context, league string, season int) (time.Time, error) {
	fetches, err := k.values.reference.SourceFetches(ctx, "mfl")
	if err != nil {
		return time.Time{}, fmt.Errorf("read MFL players attempts: %w", err)
	}
	var last time.Time
	for _, fetch := range fetches {
		if playersScope(fetch.URL, league, season, "players") || playersScope(fetch.URL, league, season, "league") {
			if fetch.FetchedAt.After(last) {
				last = fetch.FetchedAt
			}
		}
	}
	return last, nil
}

func playersScope(raw, league string, season int, export string) bool {
	u, err := url.Parse(raw)
	if err != nil || u.Scheme != "https" || !strings.HasSuffix(u.Hostname(), ".myfantasyleague.com") {
		return false
	}
	q := u.Query()
	return u.Path == "/"+strconv.Itoa(season)+"/export" && q.Get("L") == league && q.Get("TYPE") == export &&
		(export != "players" || q.Get("DETAILS") == "1")
}

type serverListings []normalize.Lookup

func (listings serverListings) Facts(id string) (domain.PlayerFacts, bool) {
	for _, listing := range listings {
		if facts, ok := listing.Facts(id); ok {
			return facts, true
		}
	}
	return domain.PlayerFacts{}, false
}

func (listings serverListings) IDs() []string {
	var ids []string
	for _, listing := range listings {
		ids = append(ids, listing.IDs()...)
	}
	slices.Sort(ids)
	return slices.Compact(ids)
}
