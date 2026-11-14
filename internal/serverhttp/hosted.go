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
	"net/netip"
	"strings"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/accounts"
	"github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/governance"
	"github.com/secureprospective/TheWarRoom/internal/leagueservice"
	"github.com/secureprospective/TheWarRoom/internal/serverhttp/landing"
	"github.com/secureprospective/TheWarRoom/internal/serverversion"
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

// Hosted has no fixture fallback. Accounts live in the sign-in file and history in the league
// file; every write that withdraws a credential reserves the league writer first (the accounts store's
// commit fence), so revocation cannot commit between transaction admission and ledger append.
type Hosted struct {
	// FootballDigest and Football are the server's pin (-football); Pins, when set, answers each league's
	// own store and pin, and an unpinned league gets the server's.
	FootballDigest string
	Football       *history.PinnedFootball
	Pins           func(ctx context.Context, league string) LeaguePin
	// WriteAdmission guards league writes and SignInAdmission guards sign-in writes: each follows its
	// own file's off-site backup, so one file falling behind pauses only its own kind of change.
	WriteAdmission  func() error
	SignInAdmission func() error
	Tokens          *accounts.TokenService
	Confirmations   *Confirmations
	Store           *history.Store
	// Leagues finds the store holding a league; league heads and the due worker read through it.
	Leagues    history.LeagueResolver
	Sessions   *accounts.SessionService
	TOTP       *accounts.TOTPService
	Enrollment *accounts.EnrollmentService
	Passwords  *accounts.Service
	Join       *accounts.JoinService
	Recovery   *accounts.RecoveryService
	Worker     *leagueservice.DueWorker
	Registry   string
	// Replies caches league-state readings per head and reading epoch; nil reads every request afresh.
	Replies *ReplyCache
	// TrustedProxy is the loopback address of a cloudflared hop; the zero address trusts none.
	TrustedProxy netip.Addr
}

func (h Hosted) Handler() http.Handler {
	return resourceLimits(versionGate(h.invalidationHandler(h.backupAdmission(http.HandlerFunc(h.serveHTTP)))), h.TrustedProxy)
}

func (h Hosted) serveHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	if h.Store == nil || h.Sessions == nil || h.TOTP == nil || h.Enrollment == nil || h.Worker == nil {
		http.Error(w, "account service unavailable", http.StatusServiceUnavailable)
		return
	}
	if r.TLS == nil || !originAdmissible(r) {
		http.Error(w, "request refused", http.StatusBadRequest)
		return
	}
	// The landing page ignores its query: shared links arrive with tracking parameters.
	if landing.Has(r.URL.Path) {
		landing.Serve(w, r)
		return
	}
	if r.URL.RawQuery != "" {
		http.Error(w, "request refused", http.StatusBadRequest)
		return
	}
	h.route(w, r)
}

func (h Hosted) route(w http.ResponseWriter, r *http.Request) {
	switch r.URL.Path {
	case RecoveryPath:
		h.recoverAccount(w, r)
	case RecoveryCodesPath:
		h.printRecoveryCodes(w, r)
	case EnrollmentPath:
		EnrollmentHandler(h.Enrollment).ServeHTTP(w, r)
	case PasswordPagePath:
		h.passwordPage(w, r)
	case PasswordChangePath:
		h.changePassword(w, r)
	case JoinPagePath:
		h.joinPage(w, r)
	case JoinFranchisesPath, JoinClaimPath:
		h.join(w, r)
	case TOTPBeginPath, TOTPConfirmPath, TOTPStepUpPath:
		h.authenticator(w, r)
	case AgentTokenPath, AgentTokenRevokePath:
		h.agentToken(w, r)
	case RevokePath:
		h.revoke(w, r)
	case LoginPath:
		h.login(w, r)
	case DispatchPath:
		// An older twr sends league changes here; the 426 envelope is what every client version shows as "update".
		writeUpdateRequired(w, serverversion.App, catalogUpdate)
	case LedgerPath, PeoplePath, PreviewPath, ExecutePath, LineupPath, PickPath, ConfirmPath, WorkflowPath:
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
		h.failure(w, h.SignInAdmission, "login refused", http.StatusUnauthorized)
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
	TokenID  string                       `json:"tokenId,omitempty"`
}

func (r RawNativeRequest) Validate() error {
	if r.Protocol != 1 {
		return fmt.Errorf("unsupported protocol")
	}
	if r.Workflow != nil {
		if r.Command != nil || r.Pick != nil || r.League+r.TokenID != "" {
			return fmt.Errorf("send only a workflow")
		}
		//nolint:wrapcheck // Preserve the catalog-owned workflow refusal text.
		return r.Workflow.Validate()
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
	if r.Command != nil || r.League != "" || r.TokenID != "" {
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
		if !sessionRefused(authenticationErr) {
			h.sessionFailure(w, authenticationErr)
			return
		}
		if h.Tokens == nil {
			h.sessionFailure(w, authenticationErr)
			return
		}
		if _, err := h.Tokens.Authenticate(r.Context(), rawSession); err != nil {
			h.sessionFailure(w, err)
			return
		}
	}
	var raw RawNativeRequest
	if !decodeNative(w, r, &raw) {
		http.Error(w, "invalid native request", http.StatusBadRequest)
		return
	}
	if authenticationErr != nil {
		h.agent(w, r, rawSession, raw)
		return
	}
	principal, err := account.Principal()
	if err != nil {
		h.writeFailure(w, "session refused", http.StatusUnauthorized)
		return
	}
	ctx := h.requestContext(r.Context(), raw, string(principal))
	r = r.WithContext(ctx)
	host := leagueservice.StoredHost{Store: h.pinFor(ctx, raw.leagueName()).Store, Context: r.Context, Principal: string(principal), Registry: h.Registry}
	gate := accountAdmission{host: host, sessions: h.Sessions.Admission(rawSession), totp: h.TOTP, proof: accounts.RecentAuth{Token: r.Header.Get("X-TWR-Recent-Auth")}}
	if r.URL.Path == ConfirmPath {
		h.confirm(w, r, gate, raw)
		return
	}
	workflowHost, err := h.workflowHost(ctx, host, raw)
	if err != nil {
		http.Error(w, "football reference unavailable", http.StatusServiceUnavailable)
		return
	}
	service := leagueservice.NewHosted(h.Worker.Host(workflowHost), gate)
	h.dispatch(w, r, service, raw)
}

// requestContext carries what the rest of the request reads: the football the workflow names, who
// asks, and the league's pin, fixed once for the whole request.
func (h Hosted) requestContext(ctx context.Context, raw RawNativeRequest, principal string) context.Context {
	if raw.Workflow != nil {
		ctx = leagueservice.WithFootballDigest(ctx, raw.Workflow.Football)
	}
	return h.withRequestPin(withReplyPrincipal(ctx, principal), raw.leagueName())
}

func decodeNative(w http.ResponseWriter, r *http.Request, raw *RawNativeRequest) bool {
	return r.Method == http.MethodPost && decodeHosted(w, r, raw) == nil && raw.Validate() == nil && raw.matchesPath(r.URL.Path)
}

// LeaguePin is the store a league is served from and the football evidence it is pinned to. Missing
// means the league is pinned to Digest but this server does not hold that evidence (a restore whose
// football reference backup was older than its league backup): the store then has no football at
// all, so readings that need football say so, and changes wait until the evidence is back.
type LeaguePin struct {
	Store    *history.Store
	Football *history.PinnedFootball
	Digest   string
	Missing  bool
}

// cacheKey tells a reading made without the league's football apart from one made with it.
func (p LeaguePin) cacheKey() string {
	if p.Missing {
		return p.Digest + "!missing"
	}
	return p.Digest
}

// pin is league's store and football pin; without Pins every league has the server's.
func (h Hosted) pin(ctx context.Context, league string) LeaguePin {
	if h.Pins != nil && league != "" {
		return h.Pins(ctx, league)
	}
	return LeaguePin{Store: h.Store, Football: h.Football, Digest: h.FootballDigest}
}

type requestPinKey struct{}

type requestPin struct {
	league string
	pin    LeaguePin
}

// withRequestPin fixes league's pin for the rest of one request, so a pin changed while it runs
// cannot give it two different football selections.
func (h Hosted) withRequestPin(ctx context.Context, league string) context.Context {
	return context.WithValue(ctx, requestPinKey{}, requestPin{league: league, pin: h.pin(ctx, league)})
}

// pinFor is league's pin as this request first read it.
func (h Hosted) pinFor(ctx context.Context, league string) LeaguePin {
	if held, ok := ctx.Value(requestPinKey{}).(requestPin); ok && held.league == league {
		return held.pin
	}
	return h.pin(ctx, league)
}

func (r RawNativeRequest) leagueName() string {
	if r.Workflow != nil {
		league, _ := r.Workflow.League()
		return league
	}
	return r.League
}

// workflowHost pins a workflow request to its league's football evidence. A league whose evidence is
// missing is served without football: its readings say what they cannot show, and workflow refuses
// its changes.
func (h Hosted) workflowHost(ctx context.Context, host leagueservice.Host, raw RawNativeRequest) (leagueservice.Host, error) {
	pin := h.pinFor(ctx, raw.leagueName())
	if raw.Workflow != nil && pin.Digest != "" && !pin.Missing {
		pinned, err := leagueservice.WithPinnedFootball(host, pin.Football)
		if err != nil {
			return nil, fmt.Errorf("select workflow evidence: %w", err)
		}
		return pinned, nil
	}
	return host, nil
}

const catalogUpdate = "League changes now go through the league catalog: update twr to the latest version, then run it again."

func (h Hosted) dispatch(w http.ResponseWriter, r *http.Request, service *leagueservice.Service, raw RawNativeRequest) {
	if raw.Workflow != nil {
		h.workflow(w, r, service, *raw.Workflow)
		return
	}
	switch r.URL.Path {
	case LineupPath, PickPath:
		h.dispatchOwner(w, r, service, raw)
	case PeoplePath:
		//nolint:contextcheck // The request-scoped StoredHost supplies r.Context to the existing reading API.
		h.cachedReading(w, r, service, raw.League, func() ([]byte, error) {
			return encodeReading(service.TargetCommissionerPeople(raw.League))
		})
	case LedgerPath:
		//nolint:contextcheck // The request-scoped StoredHost supplies r.Context to the existing reading API.
		h.cachedReading(w, r, service, raw.League, func() ([]byte, error) {
			return encodeReading(service.TargetCommissionerLedger(raw.League))
		})
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

// cachedReading answers a people or ledger reading, from the reply cache when it holds one.
func (h Hosted) cachedReading(w http.ResponseWriter, r *http.Request, service *leagueservice.Service, league string, read func() ([]byte, error)) {
	scope, err := h.replyScope(r.Context(), service, r.URL.Path, league, nil, "")
	if err != nil {
		http.Error(w, "native read refused", http.StatusForbidden)
		return
	}
	body, err := scope.reply(read)
	if err != nil {
		http.Error(w, "native read refused", http.StatusForbidden)
		return
	}
	writeReading(w, r, body)
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

func decodeHosted[T RawRecoveryRequest | RawRecoveryCodesRequest | RawRevoke | RawLogin | RawNativeRequest | RawAuthenticator | RawAgentTokenRequest | RawPasswordChangeRequest](w http.ResponseWriter, r *http.Request, raw *T) error {
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

type hostedBody interface {
	RecoveryResponse | RecoveryCodesResponse | PasswordChangeResponse | RevokeResponse | commissioner.People | commissioner.Ledger | commissioner.Preview | commissioner.Receipt | LoginResponse | AuthenticatorResponse | ConfirmationResponse | AgentTokenResponse | WorkflowResponse
}

func writeHosted[T hostedBody](w http.ResponseWriter, body T) {
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

// writeHostedReading is writeHosted for league readings, which may be compressed (writeReading).
func writeHostedReading[T commissioner.People | commissioner.Ledger | WorkflowResponse](w http.ResponseWriter, r *http.Request, body T) {
	encoded, err := json.Marshal(body)
	if err != nil {
		http.Error(w, "response unavailable", http.StatusInternalServerError)
		return
	}
	writeReading(w, r, encoded)
}

// ServeHosted never registers fixture routes or manufactures a principal.
// Exposure remains loopback-only until the deployment boundary is established.
func ServeHosted(ctx context.Context, listener net.Listener, certificate tls.Certificate, hosted Hosted) error {
	return serveFixture(ctx, listener, certificate, hosted.Handler())
}

func hostedOwner(policy governance.State, principal governance.Principal, records []commissioner.Record) bool {
	for _, id := range commissioner.LeagueFranchises(records) {
		seat, ok := policy.Franchise(governance.FranchiseID(id))
		if ok && seat.Seat == principal {
			return true
		}
	}
	return false
}

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
	return r.TokenID == "" || path == ConfirmPath
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
		if r.Method == http.MethodPost && r.TLS != nil && originAdmissible(r) && r.URL.RawQuery == "" {
			for _, admission := range h.admissions(r.URL.Path) {
				if admission == nil {
					continue
				}
				if err := admission(); err != nil {
					http.Error(w, err.Error(), http.StatusServiceUnavailable)
					return
				}
			}
		}
		next.ServeHTTP(w, r)
	})
}

// admissions are the write guards a POST to path must pass: the sign-in file's for routes that write
// only sign-ins, both for a join claim (it commits an account, then a seat), the league file's for
// everything else.
func (h Hosted) admissions(path string) []func() error {
	switch {
	case path == JoinClaimPath:
		return []func() error{h.SignInAdmission, h.WriteAdmission}
	case strings.HasPrefix(path, "/v1/auth/") || path == PasswordPagePath || path == JoinPagePath:
		return []func() error{h.SignInAdmission}
	}
	return []func() error{h.WriteAdmission}
}

func (h Hosted) writeFailure(w http.ResponseWriter, message string, status int) {
	h.failure(w, h.WriteAdmission, message, status)
}

func (h Hosted) failure(w http.ResponseWriter, admission func() error, message string, status int) {
	if admission != nil {
		if err := admission(); err != nil {
			http.Error(w, err.Error(), http.StatusServiceUnavailable)
			return
		}
	}
	http.Error(w, message, status)
}

// sessionRefused reports whether an authentication error is a real refusal (unknown, expired
// or revoked credentials) rather than the store being busy or the request running out of time.
func sessionRefused(err error) bool {
	return errors.Is(err, accounts.ErrSessionRefused) || errors.Is(err, accounts.ErrTokenRefused)
}

// sessionFailure answers an authentication error: a refusal is 401, anything else (store busy,
// timeout, context deadline) is the server being busy, so a valid owner is not told to sign in.
func (h Hosted) sessionFailure(w http.ResponseWriter, err error) {
	if sessionRefused(err) {
		h.writeFailure(w, "session refused", http.StatusUnauthorized)
		return
	}
	w.Header().Set("Retry-After", "60")
	h.writeFailure(w, BusyMessage, http.StatusServiceUnavailable)
}

// leagueHead reads a league's committed head from the store the resolver names for it.
func (h Hosted) leagueHead(ctx context.Context, league string) (int64, error) {
	if h.Leagues == nil {
		return 0, errors.New("league resolver unavailable")
	}
	store, err := h.Leagues.League(ctx, league)
	if err != nil {
		return 0, fmt.Errorf("find league store: %w", err)
	}
	return store.CommittedHead(ctx, league, commissioner.Stream) //nolint:wrapcheck // the store names its read
}
