package serverhttp

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/netip"
	"strings"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/accounts"
	"github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/governance"
	"github.com/secureprospective/TheWarRoom/internal/leagueservice"
	"github.com/secureprospective/TheWarRoom/internal/store/history"
)

const (
	PeoplePath   = "/v1/native/people"
	LoginPath    = "/v1/auth/login"
	LedgerPath   = "/v1/native/ledger"
	PreviewPath  = "/v1/native/preview"
	ExecutePath  = "/v1/native/execute"
	DispatchPath = "/v1/native/dispatch"
	LineupPath   = "/v1/native/lineup/set"
	PickPath     = "/v1/native/draft/pick"
)

// Hosted has no fixture fallback. Accounts and history must share the single writer,
// so revocation cannot commit between transaction admission and ledger append.
type Hosted struct {
	FootballDigest string
	Football       *history.PinnedFootball
	WriteAdmission func() error
	Tokens         *accounts.TokenService
	Confirmations  *Confirmations
	Store          *history.Store
	Sessions       *accounts.SessionService
	TOTP           *accounts.TOTPService
	Enrollment     *accounts.EnrollmentService
	Recovery       *accounts.RecoveryService
	Worker         *leagueservice.DueWorker
	Registry       string
	// TrustedProxy is the loopback address of a cloudflared hop; the zero address trusts none.
	TrustedProxy netip.Addr
}

func (h Hosted) Handler() http.Handler {
	return resourceLimits(h.invalidationHandler(h.backupAdmission(http.HandlerFunc(h.serveHTTP))), h.TrustedProxy)
}

func (h Hosted) serveHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	if h.Store == nil || h.Sessions == nil || h.TOTP == nil || h.Enrollment == nil || h.Worker == nil {
		http.Error(w, "account service unavailable", http.StatusServiceUnavailable)
		return
	}
	if r.TLS == nil || r.Header.Get("Origin") != "" || r.URL.RawQuery != "" {
		http.Error(w, "request refused", http.StatusBadRequest)
		return
	}
	h.route(w, r)
}

func (h Hosted) route(w http.ResponseWriter, r *http.Request) {
	switch r.URL.Path {
	case RecoveryPath:
		h.recoverAccount(w, r)
	case EnrollmentPath:
		EnrollmentHandler(h.Enrollment).ServeHTTP(w, r)
	case TOTPBeginPath, TOTPConfirmPath, TOTPStepUpPath:
		h.authenticator(w, r)
	case AgentTokenPath, AgentTokenRevokePath:
		h.agentToken(w, r)
	case RevokePath:
		h.revoke(w, r)
	case LoginPath:
		h.login(w, r)
	case LedgerPath, PeoplePath, PreviewPath, ExecutePath, DispatchPath, LineupPath, PickPath, ConfirmPath, WorkflowPath:
		h.native(w, r)
	default:
		http.NotFound(w, r)
	}
}

// RawLogin deliberately decodes passwords only at this HTTPS boundary; domain credentials
// omit passwords from JSON. No account ID, principal, role or admission time is accepted.
type RawLogin struct {
	Protocol int    `json:"protocol"`
	Username string `json:"username"`
	Password string `json:"password"`
	Device   string `json:"device"`
}

func (r RawLogin) String() string   { return "[login redacted]" }
func (r RawLogin) GoString() string { return r.String() }
func (r RawLogin) Validate() error {
	if r.Protocol != 1 || !accounts.ValidDeviceName(r.Device) {
		return fmt.Errorf("invalid login")
	}
	if err := (accounts.RawCredentials{Username: r.Username, Password: r.Password}).Validate(); err != nil {
		return fmt.Errorf("login credentials: %w", err)
	}
	return nil
}

func (h Hosted) login(w http.ResponseWriter, r *http.Request) {
	var raw RawLogin
	if r.Method != http.MethodPost || decodeHosted(w, r, &raw) != nil || raw.Validate() != nil {
		http.Error(w, "invalid login request", http.StatusBadRequest)
		return
	}
	session, err := h.Sessions.Login(r.Context(), accounts.RawCredentials{Username: raw.Username, Password: raw.Password}, raw.Device)
	if passwordBusy(w, err) {
		return
	}
	if err != nil {
		h.writeFailure(w, "login refused", http.StatusUnauthorized)
		return
	}
	// The bearer crosses HTTPS only here. DeviceSession itself remains redacted.
	writeHosted(w, LoginResponse{1, session.Bearer, session})
}

type RawNativeRequest struct {
	Workflow *RawWorkflow                 `json:"workflow,omitempty"`
	Protocol int                          `json:"protocol"`
	League   string                       `json:"league,omitempty"`
	Command  *commissioner.RawCommand     `json:"command,omitempty"`
	Pick     *commissioner.DraftPickInput `json:"pick,omitempty"`
	Mode     string                       `json:"mode,omitempty"`
	TokenID  string                       `json:"tokenId,omitempty"`
}

func (r RawNativeRequest) Validate() error {
	if r.Protocol != 1 {
		return fmt.Errorf("unsupported protocol")
	}
	if r.Workflow != nil {
		if r.Command != nil || r.Pick != nil || r.League+r.Mode+r.TokenID != "" {
			return fmt.Errorf("send only a workflow")
		}
		return r.Workflow.Validate()
	}
	if !validDispatchMode(r.Mode) {
		return fmt.Errorf("choose preview or execute")
	}
	if r.Pick != nil {
		return r.validatePick()
	}
	if r.Command == nil {
		if !strings.HasPrefix(r.League, "native.") || len(r.League) <= len("native.") || len(r.League) > 128 {
			return fmt.Errorf("invalid league")
		}
		return nil
	}
	if r.League != "" {
		return fmt.Errorf("choose command or league")
	}
	if err := r.Command.ValidatePreview(); err != nil {
		return fmt.Errorf("invalid command: %w", err)
	}
	return nil
}

func (r RawNativeRequest) validatePick() error {
	if r.Command != nil || r.League != "" || r.Mode != "" || r.TokenID != "" {
		return fmt.Errorf("send only a draft pick")
	}
	if err := r.Pick.Validate(); err != nil {
		return fmt.Errorf("rookie pick input: %w", err)
	}
	return nil
}

func (h Hosted) native(w http.ResponseWriter, r *http.Request) {
	rawSession := accounts.RawSession{Bearer: strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")}
	if len(r.Header.Values("Authorization")) != 1 || !strings.HasPrefix(r.Header.Get("Authorization"), "Bearer ") {
		http.Error(w, "session required", http.StatusUnauthorized)
		return
	}
	account, authenticationErr := h.Sessions.Authenticate(r.Context(), rawSession)
	if authenticationErr != nil {
		if h.Tokens == nil {
			h.writeFailure(w, "session refused", http.StatusUnauthorized)
			return
		}
		if _, err := h.Tokens.Authenticate(r.Context(), rawSession); err != nil {
			h.writeFailure(w, "session refused", http.StatusUnauthorized)
			return
		}
	}
	var raw RawNativeRequest
	if !decodeNative(w, r, &raw) {
		http.Error(w, "invalid native request", http.StatusBadRequest)
		return
	}
	if authenticationErr != nil {
		if raw.Workflow != nil {
			http.Error(w, "sign in with your own account", http.StatusUnauthorized)
			return
		}
		h.agent(w, r, rawSession, raw)
		return
	}
	principal, err := account.Principal()
	if err != nil {
		h.writeFailure(w, "session refused", http.StatusUnauthorized)
		return
	}
	if raw.Workflow != nil {
		r = r.WithContext(leagueservice.WithFootballDigest(r.Context(), raw.Workflow.Football))
	}
	host := leagueservice.StoredHost{Store: h.Store, Context: r.Context, Principal: string(principal), Registry: h.Registry}
	gate := accountAdmission{host: host, sessions: h.Sessions.Admission(rawSession), totp: h.TOTP, proof: accounts.RecentAuth{Token: r.Header.Get("X-TWR-Recent-Auth")}}
	if r.URL.Path == ConfirmPath {
		h.confirm(w, r, gate, raw)
		return
	}
	workflowHost, err := h.workflowHost(host, raw.Workflow != nil)
	if err != nil {
		http.Error(w, "football reference unavailable", http.StatusServiceUnavailable)
		return
	}
	service := leagueservice.NewHosted(h.Worker.Host(workflowHost), gate)
	h.dispatch(w, r, service, raw)
}

func decodeNative(w http.ResponseWriter, r *http.Request, raw *RawNativeRequest) bool {
	return r.Method == http.MethodPost && decodeHosted(w, r, raw) == nil && raw.Validate() == nil && raw.matchesPath(r.URL.Path)
}

func (h Hosted) workflowHost(host leagueservice.Host, workflow bool) (leagueservice.Host, error) {
	if workflow && h.FootballDigest != "" {
		pinned, err := leagueservice.WithPinnedFootball(host, h.Football)
		if err != nil {
			return nil, fmt.Errorf("select workflow evidence: %w", err)
		}
		return pinned, nil
	}
	return host, nil
}

func (h Hosted) dispatch(w http.ResponseWriter, r *http.Request, service *leagueservice.Service, raw RawNativeRequest) {
	if raw.Workflow != nil {
		h.workflow(w, r, service, *raw.Workflow)
		return
	}
	path := r.URL.Path
	if path == DispatchPath {
		path = PreviewPath
		if raw.Mode == "execute" {
			path = ExecutePath
		}
	}
	switch path {
	case LineupPath, PickPath:
		h.dispatchOwner(w, r, service, raw)
	case PeoplePath:
		//nolint:contextcheck // The request-scoped StoredHost supplies r.Context to the existing reading API.
		reading, err := service.TargetCommissionerPeople(raw.League)
		if err != nil {
			http.Error(w, "native read refused", http.StatusForbidden)
			return
		}
		writeHosted(w, reading)
	case LedgerPath:
		//nolint:contextcheck // The request-scoped StoredHost supplies r.Context to the existing reading API.
		reading, err := service.TargetCommissionerLedger(raw.League)
		if err != nil {
			http.Error(w, "native read refused", http.StatusForbidden)
			return
		}
		writeHosted(w, reading)
	case PreviewPath:
		//nolint:contextcheck // The request-scoped StoredHost supplies r.Context to the existing preview API.
		preview, err := service.Preview(*raw.Command)
		if err != nil {
			http.Error(w, "native preview refused", http.StatusForbidden)
			return
		}
		writeHosted(w, preview)
	case ExecutePath:
		receipt, err := service.Execute(r.Context(), *raw.Command)
		if err != nil {
			h.writeFailure(w, "native change refused; reread before retrying", http.StatusConflict)
			return
		}
		writeHosted(w, receipt)
	}
}

type accountAdmission struct {
	host     leagueservice.StoredHost
	sessions commissioner.CredentialAdmission
	totp     *accounts.TOTPService
	proof    accounts.RecentAuth
}

// proofRemaining is how long this account's admission to league lasts: until its recent-auth proof
// lapses for a commissioner or deputy; limited is false for anyone whose access needs no proof.
func (a accountAdmission) proofRemaining(ctx context.Context, league string) (time.Duration, bool, error) {
	records, err := a.host.CommissionerRecords(ctx, league)
	if err != nil {
		return 0, false, fmt.Errorf("admission records: %w", err)
	}
	if len(records) == 0 {
		return 0, false, fmt.Errorf("league access unavailable")
	}
	policy, err := commissioner.GovernanceState(records)
	if err != nil {
		return 0, false, fmt.Errorf("league policy: %w", err)
	}
	principal := governance.Principal(commissioner.PrincipalFor(records, a.host.Principal))
	//nolint:wrapcheck // the account service names the refusal
	return a.totp.PrivilegedRemaining(ctx, policy, principal, a.proof)
}

func (a accountAdmission) Admit(ctx context.Context, access commissioner.Access) error {
	if a.sessions == nil || a.totp == nil {
		return accounts.ErrSessionRefused
	}
	if err := a.sessions.Admit(ctx, access); err != nil {
		return fmt.Errorf("session admission: %w", err)
	}
	records := access.Records
	if records == nil {
		var err error
		records, err = a.host.CommissionerRecords(ctx, access.League)
		if err != nil {
			return fmt.Errorf("admission records: %w", err)
		}
	}
	// League creation and bootstrap authority remain local, never implicit host grants.
	if len(records) == 0 {
		return fmt.Errorf("league access unavailable")
	}
	policy, err := commissioner.GovernanceState(records)
	if err != nil {
		return fmt.Errorf("league policy: %w", err)
	}
	principal := governance.Principal(commissioner.PrincipalFor(records, access.Principal))
	if err := a.totp.CheckPrivileged(ctx, policy, principal, a.proof); err != nil {
		return fmt.Errorf("privileged authentication: %w", err)
	}
	return admitLeagueAccess(access, policy, principal, records)
}

func admitLeagueAccess(access commissioner.Access, policy governance.State, principal governance.Principal, records []commissioner.Record) error {
	if access.Mode == "read" {
		if !hostedMember(policy, principal, records) {
			return fmt.Errorf("league membership required")
		}
		return nil
	}
	if access.DraftPick != nil {
		if !policy.HoldsSeat(governance.FranchiseID(access.DraftPick.Franchise), principal) {
			return fmt.Errorf("only a seated owner or caretaker may pick")
		}
		return nil
	}
	if access.Raw.Command == commissioner.DraftPick && access.Raw.Args.Draft != nil && access.Raw.Args.Draft.Pick != nil {
		if !policy.HoldsSeat(governance.FranchiseID(access.Raw.Args.Draft.Pick.Franchise), principal) {
			return fmt.Errorf("only a seated owner or caretaker may pick")
		}
		return nil
	}
	if _, err := commissioner.AuthorizeRecords(access.Raw, access.Principal, records); err != nil {
		return fmt.Errorf("current league permission: %w", err)
	}
	return nil
}

func decodeHosted[T RawRecoveryRequest | RawRevoke | RawLogin | RawNativeRequest | RawAuthenticator | RawAgentTokenRequest](w http.ResponseWriter, r *http.Request, raw *T) error {
	r.Body = http.MaxBytesReader(w, r.Body, 256<<10)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(raw); err != nil {
		return fmt.Errorf("decode request: %w", err)
	}
	var trailing json.RawMessage
	if err := decoder.Decode(&trailing); err != io.EOF {
		return fmt.Errorf("request must contain one JSON object")
	}
	return nil
}

// LoginResponse carries the device secret only in the successful HTTPS login reply.
type LoginResponse struct {
	Protocol int                    `json:"protocol"`
	Bearer   string                 `json:"bearer"`
	Session  accounts.DeviceSession `json:"session"`
}

func (r LoginResponse) String() string   { return "[login response redacted]" }
func (r LoginResponse) GoString() string { return r.String() }

func writeHosted[T RecoveryResponse | RevokeResponse | commissioner.People | commissioner.Ledger | commissioner.Preview | commissioner.Receipt | LoginResponse | AuthenticatorResponse | ConfirmationResponse | AgentTokenResponse | WorkflowResponse](w http.ResponseWriter, body T) {
	encoded, err := json.Marshal(body)
	if err != nil {
		http.Error(w, "response unavailable", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	if _, err := w.Write(encoded); err != nil {
		return
	}
}

// ServeHosted never registers fixture routes or manufactures a principal.
// Exposure remains loopback-only until the deployment boundary is established.
func ServeHosted(ctx context.Context, listener net.Listener, certificate tls.Certificate, hosted Hosted) error {
	return serveFixture(ctx, listener, certificate, hosted.Handler())
}

func hostedOwner(policy governance.State, principal governance.Principal, records []commissioner.Record) bool {
	for _, id := range records[0].Command.Args.Franchises {
		seat, ok := policy.Franchise(governance.FranchiseID(id))
		if ok && seat.Seat == principal {
			return true
		}
	}
	return false
}

func validDispatchMode(mode string) bool { return mode == "" || mode == "preview" || mode == "execute" }

func (r RawNativeRequest) matchesPath(path string) bool {
	if path == WorkflowPath || r.Workflow != nil {
		return path == WorkflowPath && r.Workflow != nil
	}
	if (path == LedgerPath || path == PeoplePath) != (r.Command == nil && r.Pick == nil) {
		return false
	}
	if (path == PickPath) != (r.Pick != nil) {
		return false
	}
	return (r.TokenID == "" || path == ConfirmPath) && (r.Mode == "" || path == DispatchPath)
}

func (h Hosted) dispatchOwner(w http.ResponseWriter, r *http.Request, service *leagueservice.Service, raw RawNativeRequest) {
	var receipt commissioner.Receipt
	var err error
	if r.URL.Path == PickPath {
		receipt, err = service.Pick(r.Context(), *raw.Pick)
	} else {
		receipt, err = service.SetLineup(r.Context(), *raw.Command)
	}
	if err != nil {
		h.writeFailure(w, "owner action refused", http.StatusConflict)
		return
	}
	writeHosted(w, receipt)
}

func hostedMember(policy governance.State, principal governance.Principal, records []commissioner.Record) bool {
	return policy.HasRole(principal, governance.Commissioner) || policy.HasRole(principal, governance.CoCommissioner) ||
		hostedOwner(policy, principal, records) || policy.HasRole(principal, governance.DOT)
}

func (h Hosted) backupAdmission(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost && r.TLS != nil && r.Header.Get("Origin") == "" && r.URL.RawQuery == "" && h.WriteAdmission != nil {
			if err := h.WriteAdmission(); err != nil {
				http.Error(w, err.Error(), http.StatusServiceUnavailable)
				return
			}
		}
		next.ServeHTTP(w, r)
	})
}

func (h Hosted) writeFailure(w http.ResponseWriter, message string, status int) {
	if h.WriteAdmission != nil {
		if err := h.WriteAdmission(); err != nil {
			http.Error(w, err.Error(), http.StatusServiceUnavailable)
			return
		}
	}
	http.Error(w, message, status)
}
