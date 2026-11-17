package leagueservice

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/governance"
	"github.com/secureprospective/TheWarRoom/internal/store/personal"
)

type RawDraftQueueScope struct {
	League string `json:"league"`
}

func (r RawDraftQueueScope) Validate() error {
	if !validDraftQueueLeague(r.League) {
		return fmt.Errorf("choose an app league")
	}
	return nil
}

type RawDraftQueueSet struct {
	League  string   `json:"league"`
	Players []string `json:"players"`
}

func (r RawDraftQueueSet) Validate() error {
	if !validDraftQueueLeague(r.League) {
		return fmt.Errorf("choose an app league")
	}
	if r.Players == nil {
		return fmt.Errorf("provide the complete draft queue")
	}
	if err := personal.ValidateDraftQueue(r.Players); err != nil {
		return fmt.Errorf("draft queue: %w", err)
	}
	return nil
}

type DraftQueue struct {
	Players []string `json:"players"`
}

// WithPrivateEntries supplies the hosted league file's private entry store.
func (a *Service) WithPrivateEntries(store *personal.Store) *Service {
	a.privateEntries = store
	return a
}

func validDraftQueueLeague(league string) bool {
	return strings.HasPrefix(league, "native.") && league != "native." && strings.TrimSpace(league) == league && len(league) <= 128
}

// TargetDraftQueue reads only the queue belonging to the authenticated seated owner.
func (a *Service) TargetDraftQueue(ctx context.Context, raw RawDraftQueueScope) (DraftQueue, error) {
	if err := raw.Validate(); err != nil {
		return DraftQueue{}, fmt.Errorf("draft queue scope: %w", err)
	}
	ctx, store, franchise, err := a.draftQueueAccess(ctx, raw.League, "draft.queue.read", "read")
	if err != nil {
		return DraftQueue{}, err
	}
	entries, err := store.ReadFranchise(ctx, franchise)
	if err != nil {
		return DraftQueue{}, fmt.Errorf("read private draft queue: %w", err)
	}
	players := make([]string, 0)
	for _, entry := range entries {
		if entry.Kind == "draft" && entry.Entity == "player" {
			players = append(players, entry.ID)
		}
	}
	return DraftQueue{Players: players}, nil
}

// SetDraftQueue replaces the authenticated owner's complete ordered draft queue.
func (a *Service) SetDraftQueue(ctx context.Context, raw RawDraftQueueSet) (DraftQueue, error) {
	if err := raw.Validate(); err != nil {
		return DraftQueue{}, fmt.Errorf("draft queue: %w", err)
	}
	ctx, store, franchise, err := a.draftQueueAccess(ctx, raw.League, "draft.queue.set", "execute")
	if err != nil {
		return DraftQueue{}, err
	}
	if err := store.ReplaceDraftQueue(ctx, franchise, raw.Players, a.host.CommissionerNow()); err != nil {
		return DraftQueue{}, fmt.Errorf("replace private draft queue: %w", err)
	}
	return DraftQueue{Players: slices.Clone(raw.Players)}, nil
}

func (a *Service) draftQueueAccess(ctx context.Context, league, operation, mode string) (context.Context,
	*personal.Store, string, error) {
	if !a.hosted {
		return nil, nil, "", fmt.Errorf("draft queue requires the hosted league")
	}
	if err := a.host.Ready(); err != nil {
		return nil, nil, "", fmt.Errorf("draft queue startup: %w", err)
	}
	if a.privateEntries == nil {
		return nil, nil, "", fmt.Errorf("private draft queue storage is unavailable")
	}
	principal, err := a.host.CommissionerPrincipal()
	if err != nil {
		return nil, nil, "", fmt.Errorf("draft queue identity: %w", err)
	}
	ctx = a.viewContext(ctx)
	records, err := a.host.CommissionerRecords(ctx, league)
	if err != nil {
		return nil, nil, "", fmt.Errorf("draft queue seats: %w", err)
	}
	franchise, err := seatedFranchise(records, principal)
	if err != nil {
		return nil, nil, "", err
	}
	access := commissioner.Access{Principal: principal, League: league, Operation: operation, Mode: mode, Franchise: franchise}
	if err := commissioner.AdmitCredential(ctx, access); err != nil {
		return nil, nil, "", fmt.Errorf("admit draft queue: %w", err)
	}
	return ctx, a.privateEntries, franchise, nil
}

func seatedFranchise(records []commissioner.Record, principal string) (string, error) {
	state, err := commissioner.GovernanceState(records)
	if err != nil {
		return "", fmt.Errorf("draft queue policy: %w", err)
	}
	principal = commissioner.PrincipalFor(records, principal)
	franchise := ""
	for _, id := range commissioner.LeagueFranchises(records) {
		seat, ok := state.Franchise(governance.FranchiseID(id))
		if ok && seat.Seat == governance.Principal(principal) {
			if franchise != "" {
				return "", fmt.Errorf("draft queue requires one seated franchise")
			}
			franchise = id
		}
	}
	if franchise == "" {
		return "", fmt.Errorf("only a seated owner may use a private draft queue")
	}
	return franchise, nil
}
