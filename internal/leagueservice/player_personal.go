package leagueservice

import (
	"context"
	"fmt"
	"strings"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/governance"
	"github.com/secureprospective/TheWarRoom/internal/playerid"
	"github.com/secureprospective/TheWarRoom/internal/store/personal"
)

const (
	PlayerNoteRead = "player.note.read"
	PlayerNoteSet  = "player.note.set"
	PlayerWatchSet = "player.watch.set"
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

type PlayerNotes struct {
	League  string           `json:"league"`
	Player  string           `json:"player"`
	Notes   []personal.Entry `json:"notes"`
	Count   int              `json:"count"`
	Watched bool             `json:"watched"`
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

func (a *Service) TargetPlayerNotes(ctx context.Context, scope RawPlayerNotesScope) (PlayerNotes, error) {
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

func (a *Service) SetPlayerNote(ctx context.Context, input RawPlayerNoteSet) (PlayerPrivateChange, error) {
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

func (a *Service) SetPlayerWatch(ctx context.Context, input RawPlayerWatchSet) (PlayerPrivateChange, error) {
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
	if err := a.host.Ready(); err != nil {
		return nil, "", nil, nil, fmt.Errorf("player private data startup: %w", err)
	}
	principal, err := a.host.CommissionerPrincipal()
	if err != nil {
		return nil, "", nil, nil, fmt.Errorf("player private identity: %w", err)
	}
	ctx, cancel := context.WithTimeout(a.viewContext(requestCtx), refreshTimeout)
	fail := func(err error) (context.Context, string, *personal.Store, context.CancelFunc, error) {
		cancel()
		return nil, "", nil, nil, err
	}
	records, err := a.host.CommissionerRecords(ctx, league)
	if err != nil {
		return fail(fmt.Errorf("player private league: %w", err))
	}
	state, err := commissioner.GovernanceState(records)
	if err != nil {
		return fail(fmt.Errorf("player private seats: %w", err))
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
		return fail(fmt.Errorf("player private data requires a seated franchise owner"))
	}
	access := commissioner.Access{Principal: principal, League: league, Operation: operation, Mode: mode, Franchise: franchise}
	if err := commissioner.AdmitCredential(a.requestContext(ctx), access); err != nil {
		return fail(fmt.Errorf("admit player private data: %w", err))
	}
	entries := privateEntriesFor(a.host)
	if entries == nil {
		return fail(fmt.Errorf("player private storage unavailable"))
	}
	return ctx, franchise, entries, cancel, nil
}
