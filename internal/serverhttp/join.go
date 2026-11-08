package serverhttp

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"github.com/secureprospective/TheWarRoom/internal/accounts"
)

// Pick-your-team sign-up. Both doors sit under /v1/auth/, so they share login's per-client
// attempt limit, in-flight cap and request budget, and (as POSTs) the backup write admission.
const (
	JoinFranchisesPath = "/v1/auth/join/franchises"
	JoinClaimPath      = "/v1/auth/join/claim"
)

// RawJoinRequest is the HTTPS envelope: the link for listing, the link plus a pick for claiming.
type RawJoinRequest struct {
	Protocol int                    `json:"protocol"`
	Join     *accounts.RawJoin      `json:"join,omitempty"`
	Claim    *accounts.RawJoinClaim `json:"claim,omitempty"`
}

func (r RawJoinRequest) String() string   { return "[join request redacted]" }
func (r RawJoinRequest) GoString() string { return r.String() }

// JoinFranchisesResponse lists franchises and which are taken; nothing else about the league.
type JoinFranchisesResponse struct {
	Protocol   int                      `json:"protocol"`
	League     string                   `json:"league"`
	Franchises []accounts.JoinFranchise `json:"franchises"`
}

// JoinClaimResponse names the new account, its franchise and the one-time recovery codes; sign in
// with /v1/auth/login next. The codes are shown once and the server keeps only their digests.
type JoinClaimResponse struct {
	Protocol      int              `json:"protocol"`
	Account       accounts.Account `json:"account"`
	Franchise     string           `json:"franchise"`
	RecoveryCodes []string         `json:"recoveryCodes,omitempty"`
}

func decodeJoin(w http.ResponseWriter, r *http.Request) (RawJoinRequest, bool) {
	var raw RawJoinRequest
	r.Body = http.MaxBytesReader(w, r.Body, 8<<10)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&raw); err != nil || raw.Protocol != 1 {
		http.Error(w, "invalid join request", http.StatusBadRequest)
		return raw, false
	}
	if err := decoder.Decode(new(json.RawMessage)); err != io.EOF {
		http.Error(w, "invalid join request", http.StatusBadRequest)
		return raw, false
	}
	if (r.URL.Path == JoinFranchisesPath) != (raw.Join != nil) || (r.URL.Path == JoinClaimPath) != (raw.Claim != nil) {
		http.Error(w, "invalid join request", http.StatusBadRequest)
		return raw, false
	}
	return raw, true
}

func joinFailure(w http.ResponseWriter, err error) {
	if passwordBusy(w, err) {
		return
	}
	switch {
	case errors.Is(err, accounts.ErrFranchiseTaken):
		http.Error(w, accounts.ErrFranchiseTaken.Error(), http.StatusConflict)
	case errors.Is(err, accounts.ErrUsernameTaken):
		http.Error(w, "that username is taken; choose another", http.StatusConflict)
	case errors.Is(err, accounts.ErrJoinRefused):
		http.Error(w, accounts.ErrJoinRefused.Error(), http.StatusForbidden)
	case errors.Is(err, accounts.ErrJoinCredentials):
		http.Error(w, accounts.ErrJoinCredentials.Error(), http.StatusBadRequest)
	default:
		http.Error(w, "join unavailable; try again", http.StatusServiceUnavailable)
	}
}

func (h Hosted) join(w http.ResponseWriter, r *http.Request) {
	if h.Join == nil {
		http.Error(w, "join links are not enabled on this server", http.StatusServiceUnavailable)
		return
	}
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", "POST")
		http.Error(w, "use POST", http.StatusMethodNotAllowed)
		return
	}
	raw, ok := decodeJoin(w, r)
	if !ok {
		return
	}
	if raw.Join != nil {
		list, err := h.Join.Franchises(r.Context(), *raw.Join)
		if err != nil {
			joinFailure(w, err)
			return
		}
		writeJoin(w, http.StatusOK, JoinFranchisesResponse{Protocol: 1, League: raw.Join.League, Franchises: list})
		return
	}
	account, err := h.Join.Claim(r.Context(), *raw.Claim)
	if err != nil {
		joinFailure(w, err)
		return
	}
	response := JoinClaimResponse{Protocol: 1, Account: account, Franchise: raw.Claim.Franchise}
	if h.Recovery != nil {
		codes, err := h.Recovery.IssueCodes(r.Context(), account)
		if err != nil {
			http.Error(w, "your account was created, but recovery codes could not be issued; sign in and print a new set", http.StatusServiceUnavailable)
			return
		}
		response.RecoveryCodes = codes.Codes[:]
	}
	writeJoin(w, http.StatusCreated, response)
}

func writeJoin[T JoinFranchisesResponse | JoinClaimResponse](w http.ResponseWriter, status int, body T) {
	encoded, err := json.Marshal(body)
	if err != nil {
		http.Error(w, "response unavailable", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_, _ = w.Write(encoded)
}
