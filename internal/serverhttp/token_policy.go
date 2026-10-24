package serverhttp

import (
	"context"
	"fmt"
	"net/http"

	"github.com/secureprospective/TheWarRoom/internal/accounts"
	"github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/governance"
	"github.com/secureprospective/TheWarRoom/internal/leagueservice"
	"github.com/secureprospective/TheWarRoom/internal/store/history"
)

// TokenPolicy uses server-owned league history, with the transaction snapshot on append.
type TokenPolicy struct{ Store *history.Store }

func (p TokenPolicy) CheckTokenScope(ctx context.Context, account accounts.Account, scope accounts.RawTokenScope) error {
	principal, err := account.Principal()
	if err != nil {
		return fmt.Errorf("token issuer: %w", err)
	}
	host := leagueservice.StoredHost{Store: p.Store}
	records, err := host.CommissionerRecords(ctx, scope.League)
	if err != nil {
		return fmt.Errorf("token league: %w", err)
	}
	if err := commissioner.TokenScopePermission(records, string(principal), scope.Franchise); err != nil {
		return fmt.Errorf("token issuer rights: %w", err)
	}
	return tokenSeatScope(records, string(principal), scope)
}

func (p TokenPolicy) CheckTokenAccess(ctx context.Context, account accounts.Account, franchise string, access commissioner.Access) error {
	records := access.Records
	if records == nil {
		var err error
		records, err = (leagueservice.StoredHost{Store: p.Store}).CommissionerRecords(ctx, access.League)
		if err != nil {
			return fmt.Errorf("token current records: %w", err)
		}
	}
	principal, err := account.Principal()
	if err != nil {
		return fmt.Errorf("token issuer: %w", err)
	}
	if err := commissioner.TokenScopePermission(records, string(principal), franchise); err != nil {
		return fmt.Errorf("token current issuer rights: %w", err)
	}
	access.Records = records
	if err := commissioner.CheckTokenFranchise(access, franchise); err != nil {
		return fmt.Errorf("token franchise: %w", err)
	}
	if access.Mode == "read" {
		return nil
	}
	if access.DraftPick != nil || access.Operation == commissioner.DraftPick {
		state, err := commissioner.GovernanceState(records)
		if err != nil {
			return fmt.Errorf("token seats: %w", err)
		}
		if !state.HoldsSeat(governance.FranchiseID(franchise), governance.Principal(commissioner.PrincipalFor(records, string(principal)))) {
			return fmt.Errorf("take a vacant franchise's caretaker seat before picking")
		}
		return nil
	}
	if _, err := commissioner.AuthorizeRecords(access.Raw, string(principal), records); err != nil {
		return fmt.Errorf("token command permission: %w", err)
	}
	return nil
}

type agentAdmission struct {
	token         commissioner.CredentialAdmission
	confirmations *Confirmations
	confirmation  string
	auditID       string
}

func (a agentAdmission) Admit(ctx context.Context, access commissioner.Access) error {
	if err := a.token.Admit(ctx, access); err != nil {
		return fmt.Errorf("agent admission: %w", err)
	}
	if access.Mode != "execute" || access.Operation == commissioner.DraftPick {
		return nil
	}
	definition, err := access.Raw.EffectiveDefinition()
	if err != nil {
		return fmt.Errorf("agent command: %w", err)
	}
	if definition.Gravity == "G2" || definition.Gravity == "G3" {
		if err := a.confirmations.check(a.confirmation, a.auditID, access.Principal, access.Raw); err != nil {
			return fmt.Errorf("human confirmation: %w", err)
		}
	}
	return nil
}

func (h Hosted) agent(w http.ResponseWriter, r *http.Request, session accounts.RawSession, raw RawNativeRequest) {
	if h.Tokens == nil || r.URL.Path == ConfirmPath {
		http.Error(w, "session refused", http.StatusUnauthorized)
		return
	}
	token, err := h.Tokens.Authenticate(r.Context(), session)
	if err != nil {
		http.Error(w, "session refused", http.StatusUnauthorized)
		return
	}
	principal, err := token.Account.Principal()
	if err != nil {
		http.Error(w, "session refused", http.StatusUnauthorized)
		return
	}
	ctx := commissioner.WithTokenAudit(r.Context(), token.ID)
	host := leagueservice.StoredHost{Store: h.Store, Context: func() context.Context { return ctx }, Principal: string(principal), Registry: h.Registry}
	gate := agentAdmission{token: h.Tokens.Admission(session), confirmations: h.Confirmations, confirmation: r.Header.Get("X-TWR-Human-Confirmation"), auditID: token.ID}
	service := leagueservice.NewHosted(h.Worker.Host(host), gate)
	h.dispatch(w, r.WithContext(ctx), service, raw)
}

func tokenSeatScope(records []commissioner.Record, principal string, scope accounts.RawTokenScope) error {
	state, err := commissioner.GovernanceState(records)
	if err != nil {
		return fmt.Errorf("token seats: %w", err)
	}
	p := governance.Principal(commissioner.PrincipalFor(records, principal))
	seat, _ := state.Franchise(governance.FranchiseID(scope.Franchise))
	for _, command := range scope.Commands {
		if (command == commissioner.LineupSet || command == commissioner.DraftPick) && !state.HoldsSeat(governance.FranchiseID(scope.Franchise), p) {
			return fmt.Errorf("take a vacant franchise's caretaker seat before delegating lineups or picks")
		}
		if (commissioner.OwnerBidAction(command) != "" || commissioner.OwnerWaiverAction(command) != "" || commissioner.OwnerTradeAction(command) != "") && seat.Seat != p {
			return fmt.Errorf("delegate owner actions only for your own franchise")
		}
	}
	return nil
}
