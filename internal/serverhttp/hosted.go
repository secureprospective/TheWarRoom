package serverhttp

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"

	"github.com/secureprospective/TheWarRoom/internal/accounts"
	"github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/governance"
	"github.com/secureprospective/TheWarRoom/internal/leagueservice"
	"github.com/secureprospective/TheWarRoom/internal/store/history"
)

const (
	PeoplePath  = "/v1/native/people"
	LoginPath   = "/v1/auth/login"
	LedgerPath  = "/v1/native/ledger"
	PreviewPath = "/v1/native/preview"
	ExecutePath = "/v1/native/execute"
)

// Hosted has no fixture fallback. Accounts and history must share the single writer,
// so revocation cannot commit between transaction admission and ledger append.
type Hosted struct {
	Store      *history.Store
	Sessions   *accounts.SessionService
	TOTP       *accounts.TOTPService
	Enrollment *accounts.EnrollmentService
	Worker     *leagueservice.DueWorker
	Registry   string
}

func (h Hosted) Handler() http.Handler {
	return resourceLimits(h.invalidationHandler(http.HandlerFunc(h.serveHTTP)))
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
	switch r.URL.Path {
	case EnrollmentPath:
		EnrollmentHandler(h.Enrollment).ServeHTTP(w, r)
	case TOTPBeginPath, TOTPConfirmPath, TOTPStepUpPath:
		h.authenticator(w, r)
	case RevokePath:
		h.revoke(w, r)
	case LoginPath:
		h.login(w, r)
	case LedgerPath, PeoplePath, PreviewPath, ExecutePath:
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
	if errors.Is(err, accounts.ErrPasswordBusy) {
		w.Header().Set("Retry-After", "1")
		http.Error(w, "password service busy", http.StatusTooManyRequests)
		return
	}
	if err != nil {
		http.Error(w, "login refused", http.StatusUnauthorized)
		return
	}
	// The bearer crosses HTTPS only here. DeviceSession itself remains redacted.
	writeHosted(w, LoginResponse{1, session.Bearer, session})
}

type RawNativeRequest struct {
	Protocol int                      `json:"protocol"`
	League   string                   `json:"league,omitempty"`
	Command  *commissioner.RawCommand `json:"command,omitempty"`
}

func (r RawNativeRequest) Validate() error {
	if r.Protocol != 1 {
		return fmt.Errorf("unsupported protocol")
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

func (h Hosted) native(w http.ResponseWriter, r *http.Request) {
	rawSession := accounts.RawSession{Bearer: strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")}
	if len(r.Header.Values("Authorization")) != 1 || !strings.HasPrefix(r.Header.Get("Authorization"), "Bearer ") {
		http.Error(w, "session required", http.StatusUnauthorized)
		return
	}
	account, err := h.Sessions.Authenticate(r.Context(), rawSession)
	if err != nil {
		http.Error(w, "session refused", http.StatusUnauthorized)
		return
	}
	principal, err := account.Principal()
	if err != nil {
		http.Error(w, "session refused", http.StatusUnauthorized)
		return
	}
	var raw RawNativeRequest
	if r.Method != http.MethodPost || decodeHosted(w, r, &raw) != nil || raw.Validate() != nil ||
		((r.URL.Path == LedgerPath || r.URL.Path == PeoplePath) != (raw.Command == nil)) {
		http.Error(w, "invalid native request", http.StatusBadRequest)
		return
	}
	host := leagueservice.StoredHost{Store: h.Store, Context: r.Context, Principal: string(principal), Registry: h.Registry}
	gate := accountAdmission{host: host, sessions: h.Sessions.Admission(rawSession), totp: h.TOTP, proof: accounts.RecentAuth{Token: r.Header.Get("X-TWR-Recent-Auth")}}
	service := leagueservice.NewHosted(h.Worker.Host(host), gate)
	h.dispatch(w, r, service, raw)
}

func (h Hosted) dispatch(w http.ResponseWriter, r *http.Request, service *leagueservice.Service, raw RawNativeRequest) {
	switch r.URL.Path {
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
			http.Error(w, "native change refused; reread before retrying", http.StatusConflict)
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
	if access.Mode == "read" {
		if !policy.HasRole(principal, governance.Commissioner) && !policy.HasRole(principal, governance.CoCommissioner) &&
			!hostedOwner(policy, principal, records) && !policy.HasRole(principal, governance.DOT) {
			return fmt.Errorf("league membership required")
		}
		return nil
	}
	if _, err := commissioner.AuthorizeRecords(access.Raw, access.Principal, records); err != nil {
		return fmt.Errorf("current league permission: %w", err)
	}
	return nil
}

func decodeHosted[T RawRevoke | RawLogin | RawNativeRequest | RawAuthenticator](w http.ResponseWriter, r *http.Request, raw *T) error {
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

func writeHosted[T RevokeResponse | commissioner.People | commissioner.Ledger | commissioner.Preview | commissioner.Receipt | LoginResponse | AuthenticatorResponse](w http.ResponseWriter, body T) {
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
