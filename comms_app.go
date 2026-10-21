package main

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	commsapp "github.com/secureprospective/TheWarRoom/internal/app/comms"
	"github.com/secureprospective/TheWarRoom/internal/app/report"
	"github.com/secureprospective/TheWarRoom/internal/facts"
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
	in := a.commsInput()
	ctx, cancel := context.WithTimeout(a.fallbackParent(), factsTimeout)
	defer cancel()
	src := commsapp.Source{Log: a.core.FallbackLog}
	if a.core.History != nil {
		src.Archive = a.commsArchive
	}
	board, err := commsapp.Read(ctx, src, in, commsapp.RawCommsScope(raw))
	if err != nil {
		return CommsBoard{}, err //nolint:wrapcheck // preserve the league feed error
	}
	return wireComms(board), nil
}

func (a *App) commsInput() commsapp.Input {
	a.core.SeasonMu.Lock()
	in := commsapp.Input{Transactions: reportHeld(a.core.SeasonTransactions), Zone: time.Local}
	for i, feed := range a.core.CommsBoards {
		in.Boards[i] = reportHeld(feed)
	}
	a.core.SeasonMu.Unlock()
	a.core.LookupMu.Lock()
	in.Lookup = a.core.Lookup
	a.core.LookupMu.Unlock()
	in.Names = a.core.Rulebook.FranchiseNames()
	in.Mirror, _, _ = a.core.League.ContractsView()
	return in
}

func (a *App) commsArchive(ctx context.Context) (facts.RosterWeekColumns, facts.NflGameColumns, error) {
	a.core.FactsMu.Lock()
	defer a.core.FactsMu.Unlock()
	cache, _, err := a.readArchiveFacts(ctx)
	return cache.Facts.RosterWeek, cache.Facts.NflGame, err
}

// errCommsNeedsKey is the board's state when no MFL key is held: MFL serves the message board and polls only to
// a logged-in league member, so nothing is requested without one.
var errCommsNeedsKey = errors.New("MFL serves the message board and polls only to league members: connect your MFL key in Control Room › App")

// commsKey reports whether a keyed board read can go out, reading the keyring first like pending trades.
func (a *App) commsKey(ctx context.Context) error {
	a.core.KeyMu.Lock()
	defer a.core.KeyMu.Unlock()
	if a.core.KeyStore == nil {
		return errors.New("MFL keyring is locked or unavailable")
	}
	_, err := a.core.KeyStore.Get(ctx)
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
		a.core.SeasonMu.Lock()
		for i := range a.core.CommsBoards {
			if errors.Is(err, errCommsNeedsKey) {
				a.core.CommsBoards[i] = heldSeasonFeed[[]ingestion.RawLeaguePost]{}
			}
			a.core.CommsBoards[i].RefreshError = err.Error()
		}
		a.core.SeasonMu.Unlock()
		return
	}
	for i, export := range []string{"messageBoard", "polls"} {
		pause := a.core.SeasonPause
		if pause == nil {
			pause = seasonPause
		}
		if err := pause(ctx, time.Second); err != nil {
			return
		}
		rows, fresh, err := liveOrArchive(ctx, a.fallbackParent(), a.core.Season, archivedFeed[ingestion.RawLeaguePost]{
			export: export,
			fetch: func(ctx context.Context) ([]ingestion.RawLeaguePost, error) {
				return ingestion.FetchLeagueBoard(ctx, a.core.MFLClient, strconv.Itoa(a.core.Season), ingestion.LeagueID, export)
			},
			parse:  func(body []byte) ([]ingestion.RawLeaguePost, error) { return ingestion.ParseLeagueBoard(body, export) },
			bodies: a.core.History.ArchivedBodies,
		}, a.core.FallbackLog)
		a.core.SeasonMu.Lock()
		holdSeasonFeed(&a.core.CommsBoards[i], rows, fresh, err)
		a.core.SeasonMu.Unlock()
	}
}

func (a *App) TargetCommsRefresh(force bool) error {
	if err := a.ready(); err != nil {
		return fmt.Errorf("comms refresh: %w", err)
	}
	a.core.SeasonMu.Lock()
	tx := a.core.SeasonTransactions
	boards := a.core.CommsBoards
	a.core.SeasonMu.Unlock()
	ctx, cancel := context.WithTimeout(a.core.Ctx, refreshTimeout)
	defer cancel()
	a.core.RefreshMu.Lock()
	defer a.core.RefreshMu.Unlock()
	feeds := [2]report.Held[ingestion.RawLeaguePost]{}
	for i, feed := range boards {
		feeds[i] = reportHeld(feed)
	}
	commsapp.Refresh(ctx, commsapp.RefreshSource{Transactions: a.refreshSeasonTransactions,
		Boards: a.refreshCommsBoards, FreshFor: poolFreshFor}, reportHeld(tx), feeds, force)
	return nil
}

func (a *App) TargetOpenComms(destination string) error {
	if err := a.ready(); err != nil {
		return fmt.Errorf("open comms: %w", err)
	}
	host := ""
	if a.core.MFLClient != nil {
		host = a.core.MFLClient.LeagueHost()
	}
	target, err := commsapp.URL(destination, host, a.core.Season)
	if err != nil {
		return err //nolint:wrapcheck // the service preserves the comms hand-off error
	}
	a.core.OpenURL(a.core.Ctx, target)
	return nil
}

type RawCommsScope struct {
	Team string `json:"team"`
	Kind string `json:"kind"`
}

func (r RawCommsScope) Validate() error {
	return commsapp.RawCommsScope(r).Validate() //nolint:wrapcheck // preserve the league feed validation error
}

type CommsEvent struct {
	Sentence string `json:"sentence"`
	Date     string `json:"date"`
}

type CommsBoard struct {
	Feed         Report       `json:"feed"`
	Transactions Report       `json:"transactions"`
	Events       []CommsEvent `json:"events"`
	BoardNote    string       `json:"boardNote"`
	LeagueTalk   string       `json:"leagueTalk"`
	BoardPosts   []Report     `json:"boardPosts"`
	Team         string       `json:"team"`
	Kind         string       `json:"kind"`
	Teams        []RaceChoice `json:"teams"`
	Kinds        []RaceChoice `json:"kinds"`
}

func proseName(name string) string { return report.ProseName(name) }

func wireComms(b commsapp.Board) CommsBoard {
	out := CommsBoard{Feed: wireReport(b.Feed), Transactions: wireReport(b.Transactions),
		BoardNote: b.BoardNote, LeagueTalk: b.LeagueTalk, Team: b.Team, Kind: b.Kind,
		Teams: wireChoices(b.Teams), Kinds: wireChoices(b.Kinds)}
	if b.Events != nil {
		out.Events = make([]CommsEvent, len(b.Events))
		for i, event := range b.Events {
			out.Events[i] = CommsEvent(event)
		}
	}
	if b.BoardPosts != nil {
		out.BoardPosts = make([]Report, len(b.BoardPosts))
		for i, post := range b.BoardPosts {
			out.BoardPosts[i] = wireReport(post)
		}
	}
	return out
}
