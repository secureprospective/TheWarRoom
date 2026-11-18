package leagueservice

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/commissioner/market"
	"github.com/secureprospective/TheWarRoom/internal/governance"
	"github.com/secureprospective/TheWarRoom/internal/playerid"
	"github.com/secureprospective/TheWarRoom/internal/store/personal"
)

const (
	PlayerNoteRead   = "player.note.read"
	PlayerNoteSet    = "player.note.set"
	PlayerWatchRead  = "player.watch.read"
	PlayerWatchSet   = "player.watch.set"
	PlayerWatchOrder = "player.watch.order"
	WhatIfStackRead  = "whatif.stack.read"
)

type RawPlayerNotesScope struct {
	League string `json:"league"`
	Player string `json:"player"`
}

func (r RawPlayerNotesScope) Validate() error {
	if !validPersonalLeague(r.League) {
		return fmt.Errorf("choose an app league")
	}
	if !validPersonalPlayer(r.Player) {
		return fmt.Errorf("use a full league player id")
	}
	return nil
}

type RawPlayerNoteSet struct {
	League string `json:"league"`
	Player string `json:"player"`
	Text   string `json:"text"`
	At     string `json:"at,omitempty"`
}

func (r RawPlayerNoteSet) Validate() error {
	if err := (RawPlayerNotesScope{League: r.League, Player: r.Player}).Validate(); err != nil {
		return err
	}
	action := "add"
	if r.At != "" {
		action = "remove"
	}
	if err := (personal.RawChange{Kind: "note", Entity: "player", ID: r.Player,
		Text: r.Text, Action: action, At: r.At}).ValidateFields(); err != nil {
		return fmt.Errorf("validate note fields: %w", err)
	}
	return nil
}

type RawPlayerWatchReadScope struct {
	League string `json:"league"`
}

func (r RawPlayerWatchReadScope) Validate() error {
	if !validPersonalLeague(r.League) {
		return fmt.Errorf("choose an app league")
	}
	return nil
}

type RawPlayerWatchSet struct {
	League  string `json:"league"`
	Player  string `json:"player"`
	Watched bool   `json:"watched"`
}

func (r RawPlayerWatchSet) Validate() error {
	if err := (RawPlayerNotesScope{League: r.League, Player: r.Player}).Validate(); err != nil {
		return err
	}
	action := "remove"
	if r.Watched {
		action = "add"
	}
	if err := (personal.RawChange{Kind: "watch", Entity: "player", ID: r.Player,
		Action: action}).ValidateFields(); err != nil {
		return fmt.Errorf("validate watch fields: %w", err)
	}
	return nil
}

type RawPlayerWatchOrder struct {
	League string `json:"league"`
	Player string `json:"player"`
	With   string `json:"with"`
}

func (r RawPlayerWatchOrder) Validate() error {
	if err := (RawPlayerNotesScope{League: r.League, Player: r.Player}).Validate(); err != nil {
		return err
	}
	if err := (personal.RawChange{Kind: "watch", Entity: "player", ID: r.Player,
		With: r.With, Action: "swap"}).ValidateFields(); err != nil {
		return fmt.Errorf("validate watch order: %w", err)
	}
	return nil
}

type PlayerNotes struct {
	League  string           `json:"league"`
	Player  string           `json:"player"`
	Notes   []personal.Entry `json:"notes"`
	Count   int              `json:"count"`
	Watched bool             `json:"watched"`
}

type PlayerWatchList struct {
	League  string   `json:"league"`
	Players []string `json:"players"`
}

type PlayerPrivateChange struct {
	League string `json:"league"`
	Player string `json:"player"`
}

func validPersonalLeague(league string) bool {
	return strings.HasPrefix(league, "native.") && len(league) > len("native.") &&
		len(league) <= 128 && strings.TrimSpace(league) == league
}

func validPersonalPlayer(value string) bool {
	parsed, err := playerid.New(value)
	return err == nil && parsed.String() == value
}

func (a *Service) TargetPlayerNotes(scope RawPlayerNotesScope) (PlayerNotes, error) {
	return a.TargetPlayerNotesContext(a.host.CommissionerContext(), scope)
}

func (a *Service) TargetPlayerNotesContext(ctx context.Context, scope RawPlayerNotesScope) (PlayerNotes, error) {
	if err := scope.Validate(); err != nil {
		return PlayerNotes{}, fmt.Errorf("player notes input: %w", err)
	}
	ctx, franchise, entries, cancel, err := a.playerPrivateAccess(ctx, scope.League, PlayerNoteRead, "read")
	if err != nil {
		return PlayerNotes{}, err
	}
	defer cancel()
	thread, err := entries.PlayerNotesForFranchise(ctx, franchise, scope.Player)
	if err != nil {
		return PlayerNotes{}, fmt.Errorf("read player notes: %w", err)
	}
	watched, err := entries.PlayerWatchedForFranchise(ctx, franchise, scope.Player)
	if err != nil {
		return PlayerNotes{}, fmt.Errorf("read player watch: %w", err)
	}
	return PlayerNotes{League: scope.League, Player: scope.Player, Notes: thread.Notes, Count: thread.Count, Watched: watched}, nil
}

func (a *Service) SetPlayerNote(input RawPlayerNoteSet) (PlayerPrivateChange, error) {
	return a.SetPlayerNoteContext(a.host.CommissionerContext(), input)
}

func (a *Service) SetPlayerNoteContext(ctx context.Context, input RawPlayerNoteSet) (PlayerPrivateChange, error) {
	if err := input.Validate(); err != nil {
		return PlayerPrivateChange{}, fmt.Errorf("player note input: %w", err)
	}
	ctx, franchise, entries, cancel, err := a.playerPrivateAccess(ctx, input.League, PlayerNoteSet, "execute")
	if err != nil {
		return PlayerPrivateChange{}, err
	}
	defer cancel()
	action := "add"
	if input.At != "" {
		action = "remove"
	}
	change := personal.RawChange{Kind: "note", Entity: "player", ID: input.Player, Text: input.Text, Action: action, At: input.At}
	if err := entries.ChangeForFranchise(ctx, franchise, change, a.host.CommissionerNow()); err != nil {
		return PlayerPrivateChange{}, fmt.Errorf("save player note: %w", err)
	}
	return PlayerPrivateChange{League: input.League, Player: input.Player}, nil
}

func (a *Service) TargetPlayerWatches(scope RawPlayerWatchReadScope) (PlayerWatchList, error) {
	return a.TargetPlayerWatchesContext(a.host.CommissionerContext(), scope)
}

func (a *Service) TargetPlayerWatchesContext(ctx context.Context, scope RawPlayerWatchReadScope) (PlayerWatchList, error) {
	if err := scope.Validate(); err != nil {
		return PlayerWatchList{}, fmt.Errorf("player watch input: %w", err)
	}
	ctx, franchise, entries, cancel, err := a.playerPrivateAccess(ctx, scope.League, PlayerWatchRead, "read")
	if err != nil {
		return PlayerWatchList{}, err
	}
	defer cancel()
	players, err := entries.WatchedPlayersForFranchise(ctx, franchise)
	if err != nil {
		return PlayerWatchList{}, fmt.Errorf("read player watches: %w", err)
	}
	return PlayerWatchList{League: scope.League, Players: players}, nil
}

func (a *Service) SetPlayerWatch(input RawPlayerWatchSet) (PlayerPrivateChange, error) {
	return a.SetPlayerWatchContext(a.host.CommissionerContext(), input)
}

func (a *Service) SetPlayerWatchContext(ctx context.Context, input RawPlayerWatchSet) (PlayerPrivateChange, error) {
	if err := input.Validate(); err != nil {
		return PlayerPrivateChange{}, fmt.Errorf("player watch input: %w", err)
	}
	ctx, franchise, entries, cancel, err := a.playerPrivateAccess(ctx, input.League, PlayerWatchSet, "execute")
	if err != nil {
		return PlayerPrivateChange{}, err
	}
	defer cancel()
	action := "remove"
	if input.Watched {
		action = "add"
	}
	change := personal.RawChange{Kind: "watch", Entity: "player", ID: input.Player, Action: action}
	if err := entries.ChangeForFranchise(ctx, franchise, change, a.host.CommissionerNow()); err != nil {
		return PlayerPrivateChange{}, fmt.Errorf("save player watch: %w", err)
	}
	return PlayerPrivateChange{League: input.League, Player: input.Player}, nil
}

func (a *Service) SetPlayerWatchOrder(input RawPlayerWatchOrder) (PlayerPrivateChange, error) {
	return a.SetPlayerWatchOrderContext(a.host.CommissionerContext(), input)
}

func (a *Service) SetPlayerWatchOrderContext(ctx context.Context, input RawPlayerWatchOrder) (PlayerPrivateChange, error) {
	if err := input.Validate(); err != nil {
		return PlayerPrivateChange{}, fmt.Errorf("player watch order input: %w", err)
	}
	ctx, franchise, entries, cancel, err := a.playerPrivateAccess(ctx, input.League, PlayerWatchOrder, "execute")
	if err != nil {
		return PlayerPrivateChange{}, err
	}
	defer cancel()
	change := personal.RawChange{Kind: "watch", Entity: "player", ID: input.Player,
		With: input.With, Action: "swap"}
	if err := entries.ChangeForFranchise(ctx, franchise, change, a.host.CommissionerNow()); err != nil {
		return PlayerPrivateChange{}, fmt.Errorf("save player watch order: %w", err)
	}
	return PlayerPrivateChange{League: input.League, Player: input.Player}, nil
}

type privateEntriesHost interface {
	PrivateEntries() *personal.Store
}

func privateEntriesFor(host Host) *personal.Store {
	provider, ok := host.(privateEntriesHost)
	if !ok {
		return nil
	}
	return provider.PrivateEntries()
}

func (a *Service) playerPrivateAccess(requestCtx context.Context, league, operation, mode string) (
	context.Context, string, *personal.Store, context.CancelFunc, error) {
	ctx, franchise, _, cancel, err := a.franchiseOwnerAccess(requestCtx, league, operation, mode)
	if err != nil {
		return nil, "", nil, nil, err
	}
	entries := privateEntriesFor(a.host)
	if entries == nil {
		cancel()
		return nil, "", nil, nil, fmt.Errorf("player private storage unavailable")
	}
	return ctx, franchise, entries, cancel, nil
}

func (a *Service) franchiseOwnerAccess(requestCtx context.Context, league, operation, mode string) (
	context.Context, string, []commissioner.Record, context.CancelFunc, error) {
	if err := a.host.Ready(); err != nil {
		return nil, "", nil, nil, fmt.Errorf("private owner data startup: %w", err)
	}
	principal, err := a.host.CommissionerPrincipal()
	if err != nil {
		return nil, "", nil, nil, fmt.Errorf("private owner identity: %w", err)
	}
	ctx, cancel := context.WithTimeout(a.viewContext(requestCtx), refreshTimeout)
	fail := func(err error) (context.Context, string, []commissioner.Record, context.CancelFunc, error) {
		cancel()
		return nil, "", nil, nil, err
	}
	records, err := a.host.CommissionerRecords(ctx, league)
	if err != nil {
		return fail(fmt.Errorf("private owner league: %w", err))
	}
	state, err := commissioner.GovernanceState(records)
	if err != nil {
		return fail(fmt.Errorf("private owner seats: %w", err))
	}
	owner := governance.Principal(commissioner.PrincipalFor(records, principal))
	franchise := ""
	for _, id := range commissioner.LeagueFranchises(records) {
		seat, ok := state.Franchise(governance.FranchiseID(id))
		if ok && seat.Seat == owner {
			franchise = id
			break
		}
	}
	if franchise == "" {
		return fail(fmt.Errorf("private owner data requires a seated franchise owner"))
	}
	access := commissioner.Access{Principal: principal, League: league, Operation: operation, Mode: mode, Franchise: franchise}
	if err := commissioner.AdmitCredential(a.requestContext(ctx), access); err != nil {
		return fail(fmt.Errorf("admit private owner data: %w", err))
	}
	return ctx, franchise, records, cancel, nil
}

func (a *Service) TargetNativeWhatIfStack(raw market.RawWhatIfStack) (market.WhatIfStack, error) {
	return a.TargetNativeWhatIfStackContext(a.host.CommissionerContext(), raw)
}

func (a *Service) TargetNativeWhatIfStackContext(requestCtx context.Context,
	raw market.RawWhatIfStack) (market.WhatIfStack, error) {
	if err := raw.Validate(); err != nil {
		return market.WhatIfStack{}, fmt.Errorf("native what-if stack input: %w", err)
	}
	ctx, franchise, records, cancel, err := a.franchiseOwnerAccess(requestCtx, raw.League, WhatIfStackRead, "read")
	if err != nil {
		return market.WhatIfStack{}, err
	}
	defer cancel()
	zone, err := time.LoadLocation(raw.ViewerZone)
	if err != nil {
		return market.WhatIfStack{}, fmt.Errorf("read viewer timezone: %w", err)
	}
	reading, err := market.QuoteWhatIfStack(raw, records, a.host.CommissionerNow(), zone,
		a.host.NativeFacts(), franchise)
	if err != nil {
		return market.WhatIfStack{}, fmt.Errorf("native what-if stack: %w", err)
	}
	reading.Revision = commissioner.ViewRevision(ctx, records, reading.Revision)
	return reading, nil
}
