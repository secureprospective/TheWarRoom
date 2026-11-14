package serverhttp

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/accounts"
	"github.com/secureprospective/TheWarRoom/internal/leagueservice"
)

// A signed-in human, with recent authenticator proof for the league, connects an AI helper or
// script by issuing it a scoped token; only the reply carries the token's secret.
const (
	AgentTokenPath       = "/v1/auth/agent-token"        //nolint:gosec // a URL path, not a credential
	AgentTokenRevokePath = "/v1/auth/agent-token/revoke" //nolint:gosec // a URL path, not a credential
)

type RawAgentTokenRequest struct {
	Protocol int                     `json:"protocol"`
	Scope    *accounts.RawTokenScope `json:"scope,omitempty"`
	ID       string                  `json:"id,omitempty"`
}

func (r RawAgentTokenRequest) validate(path string) error {
	if r.Protocol != 1 || (path == AgentTokenPath) != (r.Scope != nil) || (path == AgentTokenRevokePath) != (r.ID != "") {
		return fmt.Errorf("invalid agent token request")
	}
	if r.Scope != nil {
		if err := r.Scope.Validate(); err != nil {
			return fmt.Errorf("agent token scope: %w", err)
		}
	}
	return nil
}

type AgentTokenResponse struct {
	Protocol  int                     `json:"protocol"`
	ID        string                  `json:"id,omitempty"`
	Bearer    string                  `json:"bearer,omitempty"`
	Scope     *accounts.RawTokenScope `json:"scope,omitempty"`
	ExpiresAt *time.Time              `json:"expiresAt,omitempty"`
}

func (AgentTokenResponse) String() string     { return "[agent token response redacted]" }
func (r AgentTokenResponse) GoString() string { return r.String() }

// deviceSession is the request's owner session; agent tokens are never device sessions, so they cannot
// mint or revoke credentials. !ok means the refusal was written.
func (h Hosted) deviceSession(w http.ResponseWriter, r *http.Request) (accounts.RawSession, accounts.Account, bool) {
	authorization := r.Header.Get("Authorization")
	if len(r.Header.Values("Authorization")) != 1 || !strings.HasPrefix(authorization, "Bearer ") {
		http.Error(w, "session required", http.StatusUnauthorized)
		return accounts.RawSession{}, accounts.Account{}, false
	}
	session := accounts.RawSession{Bearer: strings.TrimPrefix(authorization, "Bearer ")}
	account, err := h.Sessions.Authenticate(r.Context(), session)
	if err != nil {
		h.sessionFailure(w, err)
		return session, account, false
	}
	if h.Tokens == nil {
		http.Error(w, "session refused", http.StatusUnauthorized)
		return session, account, false
	}
	return session, account, true
}

func (h Hosted) agentToken(w http.ResponseWriter, r *http.Request) {
	session, account, ok := h.deviceSession(w, r)
	if !ok {
		return
	}
	var raw RawAgentTokenRequest
	if r.Method != http.MethodPost || decodeHosted(w, r, &raw) != nil || raw.validate(r.URL.Path) != nil {
		http.Error(w, "invalid agent token request", http.StatusBadRequest)
		return
	}
	if raw.Scope == nil {
		if err := h.Tokens.Revoke(r.Context(), session, raw.ID); err != nil {
			http.Error(w, "agent token refused", http.StatusForbidden)
			return
		}
		writeHosted(w, AgentTokenResponse{Protocol: 1, ID: raw.ID})
		return
	}
	principal, err := account.Principal()
	if err != nil {
		http.Error(w, "session refused", http.StatusUnauthorized)
		return
	}
	pin := h.pin(r.Context(), raw.Scope.League)
	if pin.Err != nil {
		http.Error(w, "agent token refused", http.StatusForbidden)
		return
	}
	host := leagueservice.StoredHost{Store: pin.Store, Context: r.Context, Principal: string(principal), Registry: h.Registry}
	gate := accountAdmission{host: host, sessions: h.Sessions.Admission(session), totp: h.TOTP, proof: accounts.RecentAuth{Token: r.Header.Get("X-TWR-Recent-Auth")}}
	if remaining, limited, err := gate.proofRemaining(r.Context(), raw.Scope.League); err != nil || !limited || remaining <= 0 {
		http.Error(w, "confirm with your authenticator first", http.StatusForbidden)
		return
	}
	token, err := h.Tokens.Issue(r.Context(), session, *raw.Scope)
	if err != nil {
		http.Error(w, "agent token refused", http.StatusForbidden)
		return
	}
	writeHosted(w, AgentTokenResponse{Protocol: 1, ID: token.ID, Bearer: token.Bearer, Scope: &token.Scope, ExpiresAt: &token.ExpiresAt})
}
