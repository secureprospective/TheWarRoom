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
	src := commsapp.Source{Log: a.fallbackLog}
	if a.history != nil {
		src.Archive = a.commsArchive
	}
	board, err := commsapp.Read(ctx, src, in, commsapp.RawCommsScope(raw))
	if err != nil {
		return CommsBoard{}, err //nolint:wrapcheck // preserve the league feed error
	}
	return wireComms(board), nil
}

func (a *App) commsInput() commsapp.Input {
	a.seasonMu.Lock()
	in := commsapp.Input{Transactions: reportHeld(a.seasonTransactions), Zone: time.Local}
	for i, feed := range a.commsBoards {
		in.Boards[i] = reportHeld(feed)
	}
	a.seasonMu.Unlock()
	a.lookupMu.Lock()
	in.Lookup = a.lookup
	a.lookupMu.Unlock()
	in.Names = a.rulebook.FranchiseNames()
	in.Mirror, _, _ = a.league.ContractsView()
	return in
}

func (a *App) commsArchive(ctx context.Context) (facts.RosterWeekColumns, facts.NflGameColumns, error) {
	a.factsMu.Lock()
	defer a.factsMu.Unlock()
	cache, _, err := a.readArchiveFacts(ctx)
	return cache.Facts.RosterWeek, cache.Facts.NflGame, err
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
	if a.mflClient != nil {
		host = a.mflClient.LeagueHost()
	}
	target, err := commsapp.URL(destination, host, a.season)
	if err != nil {
		return err //nolint:wrapcheck // the service preserves the comms hand-off error
	}
	a.openURL(a.ctx, target)
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
