package main

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/ingestion"
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
	return buildComms(in, raw)
}

func (a *App) refreshCommsBoards(ctx context.Context) {
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
