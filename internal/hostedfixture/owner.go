package hostedfixture

import (
	"context"
	"fmt"

	"github.com/secureprospective/TheWarRoom/internal/accounts"
	"github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/governance"
	accountstore "github.com/secureprospective/TheWarRoom/internal/store/accounts"
	"github.com/secureprospective/TheWarRoom/internal/store/history"
)

type ownerEnrollment struct {
	store  *history.Store
	issuer governance.Principal
}

func (a ownerEnrollment) EnrollmentPolicy(ctx context.Context, league string) (governance.State, governance.Principal, error) {
	events, err := a.store.Events(ctx, league, commissioner.Stream, 0)
	if err != nil {
		return governance.State{}, "", fmt.Errorf("read practice policy: %w", err)
	}
	records := make([]commissioner.Record, 0, len(events))
	for _, event := range events {
		records = append(records, *event.Payload.Commissioner)
	}
	policy, err := commissioner.GovernanceState(records)
	if err != nil {
		return governance.State{}, "", fmt.Errorf("practice policy: %w", err)
	}
	return policy, a.issuer, nil
}

// AddWorkflowOwner enrolls an ordinary second owner through the real enrollment and seat APIs.
func AddWorkflowOwner(ctx context.Context, store *history.Store, credentials *accountstore.Store, issuer governance.Principal, franchise string, raw accounts.RawCredentials) error {
	service := accounts.NewEnrollment(credentials, ownerEnrollment{store, issuer})
	code, err := service.Issue(ctx, "native.hosted", raw.Username)
	if err != nil {
		return fmt.Errorf("invite practice owner: %w", err)
	}
	account, err := service.Redeem(ctx, accounts.RawEnrollment{Code: code.Code, Username: raw.Username, Password: raw.Password})
	if err != nil {
		return fmt.Errorf("enroll practice owner: %w", err)
	}
	principal, err := account.Principal()
	if err != nil {
		return fmt.Errorf("practice owner identity: %w", err)
	}
	assets, err := store.NativeAssets(ctx, "native.hosted")
	if err != nil {
		return fmt.Errorf("read practice seats: %w", err)
	}
	command := commissioner.RawCommand{League: "native.hosted", Command: "commish.seat.replace", Args: commissioner.RawArgs{People: &commissioner.RawPeople{Franchise: franchise, Principal: string(principal)}}, Expected: assets.Revision, Request: "practice.owner." + franchise, Reason: "Seat disposable second owner", Effective: "now", Visibility: "league"}
	if _, err := store.ExecuteCommissioner(ctx, command, string(issuer), nil); err != nil {
		return fmt.Errorf("seat practice owner: %w", err)
	}
	return nil
}
