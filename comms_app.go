package main

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/ingestion"
	"github.com/secureprospective/TheWarRoom/internal/mflkey"
)

func (a *App) TargetComms(raw RawCommsScope) (CommsBoard, error) {
	if err := raw.Validate(); err != nil {
		return CommsBoard{}, err
	}
	if err := a.ready(); err != nil {
		return CommsBoard{}, fmt.Errorf("comms: startup: %w", err)
	}
	a.seasonMu.Lock()
	in := commsInput{transactions: a.seasonTransactions, boards: a.commsBoards, zone: time.Local}
	a.seasonMu.Unlock()
	a.lookupMu.Lock()
	in.lookup = a.lookup
	a.lookupMu.Unlock()
	in.names = a.rulebook.FranchiseNames()
	in.mirror, _, _ = a.league.ContractsView()
	if a.history != nil {
		ctx, cancel := context.WithTimeout(a.fallbackParent(), factsTimeout)
		defer cancel()
		a.factsMu.Lock()
		cache, _, err := a.readArchiveFacts(ctx)
		a.factsMu.Unlock()
		if err != nil {
			in.rosterNote = " Weekly contract history unavailable; dropped salaries may not be held."
			a.fallbackLog(err.Error())
		}
		in.rosters, in.rosterTimes = cache.facts.RosterWeek, rosterWeekTimes(cache.facts.NflGame)
	}
	return buildComms(in, raw)
}

// errCommsNeedsKey is the board's state when no MFL key is held: MFL serves the message board and polls only to
// a logged-in league member, so nothing is requested without one.
var errCommsNeedsKey = errors.New("MFL serves the message board and polls only to league members: connect your MFL key in Control Room › App")

// commsKey reports whether a keyed board read can go out, reading the keyring first like pending trades.
func (a *App) commsKey(ctx context.Context) error {
	a.keyMu.Lock()
	defer a.keyMu.Unlock()
	if a.keyStore == nil {
		return errors.New("MFL keyring is locked or unavailable")
	}
	_, err := a.keyStore.Get(ctx)
	switch {
	case errors.Is(err, mflkey.ErrNoKey):
		return errCommsNeedsKey
	case err != nil:
		return errors.New("MFL keyring is locked or unavailable")
	}
	return nil
}

func (a *App) refreshCommsBoards(ctx context.Context) {
	if err := a.commsKey(ctx); err != nil {
		a.seasonMu.Lock()
		for i := range a.commsBoards {
			if errors.Is(err, errCommsNeedsKey) {
				a.commsBoards[i] = heldSeasonFeed[[]ingestion.RawLeaguePost]{}
			}
			a.commsBoards[i].refreshError = err.Error()
		}
		a.seasonMu.Unlock()
		return
	}
	for i, export := range []string{"messageBoard", "polls"} {
		pause := a.seasonPause
		if pause == nil {
			pause = seasonPause
		}
		if err := pause(ctx, time.Second); err != nil {
			return
		}
		rows, fresh, err := liveOrArchive(ctx, a.fallbackParent(), a.season, archivedFeed[ingestion.RawLeaguePost]{
			export: export,
			fetch: func(ctx context.Context) ([]ingestion.RawLeaguePost, error) {
				return ingestion.FetchLeagueBoard(ctx, a.mflClient, strconv.Itoa(a.season), ingestion.LeagueID, export)
			},
			parse:  func(body []byte) ([]ingestion.RawLeaguePost, error) { return ingestion.ParseLeagueBoard(body, export) },
			bodies: a.history.ArchivedBodies,
		}, a.fallbackLog)
		a.seasonMu.Lock()
		holdSeasonFeed(&a.commsBoards[i], rows, fresh, err)
		a.seasonMu.Unlock()
	}
}

func (a *App) TargetCommsRefresh(force bool) error {
	if err := a.ready(); err != nil {
		return fmt.Errorf("comms refresh: %w", err)
	}
	a.seasonMu.Lock()
	tx := a.seasonTransactions
	boards := a.commsBoards
	a.seasonMu.Unlock()
	ctx, cancel := context.WithTimeout(a.ctx, refreshTimeout)
	defer cancel()
	a.refreshMu.Lock()
	defer a.refreshMu.Unlock()
	if force || tx.refreshError != "" || time.Since(tx.fetchedAt) >= poolFreshFor {
		a.refreshSeasonTransactions(ctx)
	}
	fresh := true
	for _, b := range boards {
		fresh = fresh && b.refreshError == "" && time.Since(b.fetchedAt) < poolFreshFor
	}
	if force || !fresh {
		a.refreshCommsBoards(ctx)
	}
	return nil
}

func commsURL(destination, host string, season int) (string, error) {
	if destination == "league-board" {
		return "https://legacynfl.boards.net/", nil
	}
	option := map[string]string{"message-board": "28", "polls": "69"}[destination]
	if option == "" {
		return "", fmt.Errorf("comms: invalid hand-off destination")
	}
	if host == "" {
		return "", fmt.Errorf("comms: MFL host not held; refresh from MFL first")
	}
	return fmt.Sprintf("https://%s/%d/options?L=%s&O=%s", host, season, url.QueryEscape(ingestion.LeagueID), option), nil
}

func (a *App) TargetOpenComms(destination string) error {
	if err := a.ready(); err != nil {
		return fmt.Errorf("open comms: %w", err)
	}
	host := ""
	if a.mflClient != nil {
		host = a.mflClient.LeagueHost()
	}
	target, err := commsURL(destination, host, a.season)
	if err != nil {
		return err
	}
	a.openURL(a.ctx, target)
	return nil
}
