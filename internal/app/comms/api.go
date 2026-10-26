package comms

import (
	"context"
	"fmt"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/app/catalog"
	"github.com/secureprospective/TheWarRoom/internal/app/report"
	"github.com/secureprospective/TheWarRoom/internal/facts"
	"github.com/secureprospective/TheWarRoom/internal/ingestion"
	"github.com/secureprospective/TheWarRoom/internal/leaguefeed"
	"github.com/secureprospective/TheWarRoom/internal/normalize"
	"github.com/secureprospective/TheWarRoom/internal/store/state"
)

// Host supplies detached readings and serializes publication; no lock or transport enters the API.
type Host interface {
	Ready() error
	CommsContext() context.Context
	CommsFeeds() (report.Held[leaguefeed.Transaction], [2]report.Held[ingestion.RawLeaguePost], *time.Location)
	CommsLeague() (normalize.Lookup, map[string]string, state.MirrorSnapshot)
	CommsArchive(context.Context) (facts.RosterWeekColumns, facts.NflGameColumns, error)
	LogFallback(string)
	RefreshComms(bool) error
	CommsDestination() (string, int)
	OpenBrowser(string)
}

type API struct{ host Host }

func New(host Host) *API { return &API{host: host} }

func (a *API) TargetComms(raw RawCommsScope) (CommsBoard, error) {
	if err := raw.Validate(); err != nil {
		return CommsBoard{}, err
	}
	if err := a.host.Ready(); err != nil {
		return CommsBoard{}, fmt.Errorf("comms: startup: %w", err)
	}
	var in Input
	in.Transactions, in.Boards, in.Zone = a.host.CommsFeeds()
	in.Lookup, in.Names, in.Mirror = a.host.CommsLeague()
	ctx, cancel := context.WithTimeout(a.host.CommsContext(), 5*time.Minute)
	defer cancel()
	board, err := Read(ctx, Source{Archive: a.host.CommsArchive, Log: a.host.LogFallback}, in, raw)
	if err != nil {
		return CommsBoard{}, err
	}
	return wireComms(board), nil
}

func (a *API) TargetCommsRefresh(force bool) error {
	return a.host.RefreshComms(force) //nolint:wrapcheck // The host preserves the existing refresh error.
}

func (a *API) TargetOpenComms(destination string) error {
	if err := a.host.Ready(); err != nil {
		return fmt.Errorf("open comms: %w", err)
	}
	host, season := a.host.CommsDestination()
	target, err := URL(destination, host, season)
	if err != nil {
		return err
	}
	a.host.OpenBrowser(target)
	return nil
}

//nolint:revive // Preserve the existing exported wire name.
type CommsEvent struct {
	Sentence string `json:"sentence"`
	Date     string `json:"date"`
}

//nolint:revive // Preserve the existing exported wire name.
type CommsBoard struct {
	Feed         report.Report   `json:"feed"`
	Transactions report.Report   `json:"transactions"`
	Events       []CommsEvent    `json:"events"`
	BoardNote    string          `json:"boardNote"`
	LeagueTalk   string          `json:"leagueTalk"`
	BoardPosts   []report.Report `json:"boardPosts"`
	Team         string          `json:"team"`
	Kind         string          `json:"kind"`
	Teams        []report.Choice `json:"teams"`
	Kinds        []report.Choice `json:"kinds"`
}

func wireComms(b Board) CommsBoard {
	out := CommsBoard{Feed: b.Feed, Transactions: b.Transactions, BoardNote: b.BoardNote,
		LeagueTalk: b.LeagueTalk, BoardPosts: b.BoardPosts, Team: b.Team, Kind: b.Kind, Teams: b.Teams, Kinds: b.Kinds}
	if b.Events != nil {
		out.Events = make([]CommsEvent, len(b.Events))
		for i, event := range b.Events {
			out.Events[i] = CommsEvent(event)
		}
	}
	return out
}

// Source reads archive columns through the caller's lock and held snapshot.
// A nil Archive means archive history is not configured.
type Source struct {
	Archive func(context.Context) (facts.RosterWeekColumns, facts.NflGameColumns, error)
	Log     func(string)
}

func Read(ctx context.Context, src Source, in Input, scope RawCommsScope) (Board, error) {
	if src.Archive != nil {
		rosters, games, err := src.Archive(ctx)
		if err != nil {
			in.RosterNote = " Weekly contract history unavailable; dropped salaries may not be held."
			src.Log(err.Error())
		}
		in.Rosters, in.RosterTimes = rosters, RosterWeekTimes(games)
	}
	return Build(in, scope)
}

// Commands is data, not an API method: Wails must not bind the catalog itself.
func Commands() []catalog.Command { return catalog.DesktopCommands("comms", nil) }
