package serverhttp

import (
	"context"
	"crypto/sha256"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/accounts"
	"github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/governance"
	"github.com/secureprospective/TheWarRoom/internal/leagueservice"
)

const browserSessionDevice = "The War Room browser"

// RawBrowserLogin carries only sign-in credentials; device identity is server-assigned.
type RawBrowserLogin struct {
	Protocol int    `json:"protocol"`
	Username string `json:"username"`
	Password string `json:"password"`
	Code     string `json:"code,omitempty"`
}

func (r RawBrowserLogin) String() string   { return "[browser login redacted]" }
func (r RawBrowserLogin) GoString() string { return r.String() }
func (r RawBrowserLogin) Validate() error {
	if r.Protocol != 1 {
		return fmt.Errorf("unsupported browser login protocol")
	}
	if err := (accounts.RawCredentials{Username: r.Username, Password: r.Password}).Validate(); err != nil {
		return fmt.Errorf("browser login credentials: %w", err)
	}
	if r.Code != "" {
		if err := (accounts.RawTOTP{Code: r.Code}).Validate(); err != nil {
			return fmt.Errorf("browser authenticator code: %w", err)
		}
	}
	return nil
}

type BrowserSeat struct {
	Franchise string `json:"franchise"`
	Name      string `json:"name"`
}

type BrowserAdmission struct {
	League        string        `json:"league"`
	Name          string        `json:"name"`
	Franchise     string        `json:"franchise"`
	FranchiseName string        `json:"franchiseName"`
	Role          string        `json:"role"`
	Seats         []BrowserSeat `json:"seats"`
}

type BrowserLeague struct {
	ID            string `json:"id"`
	Name          string `json:"name"`
	Role          string `json:"role"`
	Franchise     string `json:"franchise"`
	FranchiseName string `json:"franchiseName"`
}

type BrowserSessionResponse struct {
	Protocol  int              `json:"protocol"`
	Admission BrowserAdmission `json:"admission"`
	Leagues   []BrowserLeague  `json:"leagues"`
}

type browserLeagueAdmission struct {
	view      BrowserAdmission
	policy    governance.State
	principal governance.Principal
}

type browserProof struct {
	token     string
	expiresAt time.Time
}

type browserSessionAuth struct {
	mu     sync.Mutex
	proofs map[[sha256.Size]byte]browserProof
}

func newBrowserSessionAuth() *browserSessionAuth {
	return &browserSessionAuth{proofs: make(map[[sha256.Size]byte]browserProof)}
}

func (s *browserSessionAuth) remember(session, proof string) {
	if proof == "" {
		return
	}
	key := sha256.Sum256([]byte(session))
	now := time.Now()
	s.mu.Lock()
	defer s.mu.Unlock()
	for digest, entry := range s.proofs {
		if !now.Before(entry.expiresAt) {
			delete(s.proofs, digest)
		}
	}
	if len(s.proofs) >= 1024 {
		var oldest [sha256.Size]byte
		var expires time.Time
		for digest, entry := range s.proofs {
			if expires.IsZero() || entry.expiresAt.Before(expires) {
				oldest, expires = digest, entry.expiresAt
			}
		}
		delete(s.proofs, oldest)
	}
	s.proofs[key] = browserProof{token: proof, expiresAt: now.Add(accounts.RecentAuthLifetime)}
}

func (s *browserSessionAuth) recent(session string) string {
	key := sha256.Sum256([]byte(session))
	now := time.Now()
	s.mu.Lock()
	defer s.mu.Unlock()
	entry, ok := s.proofs[key]
	if !ok {
		return ""
	}
	if !now.Before(entry.expiresAt) {
		delete(s.proofs, key)
		return ""
	}
	return entry.token
}

func (s *browserSessionAuth) forget(session string) {
	key := sha256.Sum256([]byte(session))
	s.mu.Lock()
	delete(s.proofs, key)
	s.mu.Unlock()
}

func browserSessionCookie(r *http.Request) (string, bool, bool) {
	var value string
	count := 0
	for _, cookie := range r.Cookies() {
		if cookie.Name == appSessionCookie {
			value = cookie.Value
			count++
		}
	}
	return value, count > 0, count == 1 && value != ""
}

func hasBrowserSessionCookie(r *http.Request) bool {
	_, found, _ := browserSessionCookie(r)
	return found
}

func setBrowserSessionCookie(w http.ResponseWriter, session string, expires time.Time) {
	http.SetCookie(w, &http.Cookie{Name: appSessionCookie, Value: session, Path: "/", Expires: expires,
		Secure: true, HttpOnly: true, SameSite: http.SameSiteStrictMode})
}

func clearBrowserSessionCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{Name: appSessionCookie, Value: "", Path: "/", Expires: time.Unix(1, 0),
		MaxAge: -1, Secure: true, HttpOnly: true, SameSite: http.SameSiteStrictMode})
}

func (h Hosted) appSession(w http.ResponseWriter, r *http.Request) {
	if !h.ownBrowserOrigin(r) || len(r.Header.Values("Authorization")) != 0 {
		http.Error(w, "browser session refused", http.StatusForbidden)
		return
	}
	switch r.Method {
	case http.MethodPost:
		h.createBrowserSession(w, r)
	case http.MethodGet:
		h.readBrowserSession(w, r)
	case http.MethodDelete:
		h.revokeBrowserSession(w, r)
	default:
		w.Header().Set("Allow", "GET, POST, DELETE")
		http.Error(w, "use GET, POST or DELETE", http.StatusMethodNotAllowed)
	}
}

func (h Hosted) createBrowserSession(w http.ResponseWriter, r *http.Request) {
	var raw RawBrowserLogin
	if decodeHosted(w, r, &raw) != nil || raw.Validate() != nil {
		http.Error(w, "invalid browser login request", http.StatusBadRequest)
		return
	}
	credentials := accounts.RawCredentials{Username: raw.Username, Password: raw.Password}
	proof, ok := h.browserLoginProof(w, r, raw, credentials)
	if !ok {
		return
	}
	session, err := h.Sessions.Login(r.Context(), credentials, browserSessionDevice)
	if passwordBusy(w, r, err) {
		return
	}
	if err != nil {
		h.failure(w, h.SignInAdmission, "sign-in refused", http.StatusUnauthorized)
		return
	}
	h.finishBrowserSession(w, r, session, proof)
}

func (h Hosted) browserLoginProof(w http.ResponseWriter, r *http.Request, raw RawBrowserLogin,
	credentials accounts.RawCredentials) (accounts.RecentAuth, bool) {
	if raw.Code == "" {
		return accounts.RecentAuth{}, true
	}
	proof, err := h.TOTP.StepUp(r.Context(), credentials, accounts.RawTOTP{Code: raw.Code})
	if passwordBusy(w, r, err) {
		return accounts.RecentAuth{}, false
	}
	if err != nil {
		http.Error(w, "sign-in refused", http.StatusUnauthorized)
		return accounts.RecentAuth{}, false
	}
	return proof, true
}

func (h Hosted) finishBrowserSession(w http.ResponseWriter, r *http.Request, session accounts.DeviceSession,
	proof accounts.RecentAuth) {
	admissions, err := h.browserAdmissions(r.Context(), session.Account)
	if err != nil {
		h.rejectNewBrowserSession(w, r, session.RawSession, http.StatusServiceUnavailable, "league access unavailable")
		return
	}
	if len(admissions) == 0 {
		h.rejectNewBrowserSession(w, r, session.RawSession, http.StatusForbidden, "league access unavailable")
		return
	}
	if h.browserPrivilegeProof(r, admissions, proof) != nil {
		h.rejectNewBrowserSession(w, r, session.RawSession, http.StatusForbidden, "sign-in refused")
		return
	}
	if proof.Token != "" {
		h.appAuth.remember(session.Bearer, proof.Token)
	}
	setBrowserSessionCookie(w, session.Bearer, session.ExpiresAt)
	writeHosted(w, browserSessionResponse(admissions))
}

func (h Hosted) browserPrivilegeProof(r *http.Request, admissions []browserLeagueAdmission,
	proof accounts.RecentAuth) error {
	for _, admission := range admissions {
		if _, _, err := h.TOTP.PrivilegedRemaining(r.Context(), admission.policy, admission.principal, proof); err != nil {
			return fmt.Errorf("check browser privileged access: %w", err)
		}
	}
	return nil
}

func (h Hosted) rejectNewBrowserSession(w http.ResponseWriter, r *http.Request, session accounts.RawSession,
	status int, message string) {
	if err := h.revokeNewBrowserSession(r, session); err != nil {
		writeUnavailable(w, "browser sign-in could not be completed")
		return
	}
	if status == http.StatusServiceUnavailable {
		writeUnavailable(w, message)
		return
	}
	http.Error(w, message, status)
}

func (h Hosted) readBrowserSession(w http.ResponseWriter, r *http.Request) {
	raw, found, valid := browserSessionCookie(r)
	if !found || !valid {
		http.Error(w, "session required", http.StatusUnauthorized)
		return
	}
	session := accounts.RawSession{Bearer: raw}
	account, err := h.Sessions.Authenticate(r.Context(), session)
	if err != nil {
		h.sessionFailure(w, r, err)
		return
	}
	admissions, err := h.browserAdmissions(r.Context(), account)
	if err != nil {
		writeUnavailable(w, "league access unavailable")
		return
	}
	if len(admissions) == 0 {
		http.Error(w, "league access unavailable", http.StatusForbidden)
		return
	}
	writeHosted(w, browserSessionResponse(admissions))
}

func (h Hosted) revokeBrowserSession(w http.ResponseWriter, r *http.Request) {
	raw, found, valid := browserSessionCookie(r)
	if !found || !valid {
		clearBrowserSessionCookie(w)
		http.Error(w, "session required", http.StatusUnauthorized)
		return
	}
	session := accounts.RawSession{Bearer: raw}
	if err := h.Sessions.Revoke(r.Context(), session, false); err != nil {
		clearBrowserSessionCookie(w)
		if sessionRefused(err) {
			http.Error(w, "session required", http.StatusUnauthorized)
		} else {
			writeUnavailable(w, "sign-out unavailable")
		}
		return
	}
	h.appAuth.forget(session.Bearer)
	clearBrowserSessionCookie(w)
	w.WriteHeader(http.StatusNoContent)
}

func (h Hosted) revokeNewBrowserSession(r *http.Request, session accounts.RawSession) error {
	if err := h.Sessions.Revoke(r.Context(), session, false); err != nil {
		return fmt.Errorf("revoke unreturned browser session: %w", err)
	}
	h.appAuth.forget(session.Bearer)
	return nil
}

func (h Hosted) browserAdmissions(ctx context.Context, account accounts.Account) ([]browserLeagueAdmission, error) {
	if h.Leagues == nil {
		return nil, fmt.Errorf("server league directory unavailable")
	}
	leagues, err := h.Leagues.Leagues(ctx)
	if err != nil {
		return nil, fmt.Errorf("read server league directory: %w", err)
	}
	principal, err := account.Principal()
	if err != nil {
		return nil, fmt.Errorf("browser account identity: %w", err)
	}
	admissions := make([]browserLeagueAdmission, 0, len(leagues))
	for _, league := range leagues {
		if !strings.HasPrefix(league.ID, "native.") || len(league.ID) <= len("native.") {
			return nil, fmt.Errorf("invalid server league identity")
		}
		pin := h.pin(ctx, league.ID)
		if pin.Err != nil || pin.Store == nil {
			return nil, fmt.Errorf("server league store unavailable")
		}
		host := leagueservice.StoredHost{Store: pin.Store, Context: func() context.Context { return ctx }, Principal: string(principal)}
		records, err := host.CommissionerRecords(ctx, league.ID)
		if err != nil {
			return nil, fmt.Errorf("read server league access: %w", err)
		}
		policy, err := commissioner.GovernanceState(records)
		if err != nil {
			return nil, fmt.Errorf("read server league policy: %w", err)
		}
		ownPrincipal := governance.Principal(commissioner.PrincipalFor(records, string(principal)))
		if !hostedMember(policy, ownPrincipal, records) {
			continue
		}
		people, err := commissioner.PeopleReading(league.ID, string(principal), records)
		if err != nil {
			return nil, fmt.Errorf("read admitted league details: %w", err)
		}
		admissions = append(admissions, browserLeagueAdmission{
			view: browserAdmissionResponse(league.ID, people), policy: policy, principal: ownPrincipal,
		})
	}
	return admissions, nil
}

func browserSessionResponse(admissions []browserLeagueAdmission) BrowserSessionResponse {
	leagues := make([]BrowserLeague, 0, len(admissions))
	for _, admission := range admissions {
		leagues = append(leagues, BrowserLeague{ID: admission.view.League, Name: admission.view.Name,
			Role: admission.view.Role, Franchise: admission.view.Franchise,
			FranchiseName: admission.view.FranchiseName})
	}
	return BrowserSessionResponse{Protocol: 1, Admission: admissions[0].view, Leagues: leagues}
}

func browserAdmissionResponse(league string, people commissioner.People) BrowserAdmission {
	seats := make([]BrowserSeat, 0, len(people.Seats))
	franchiseName := ""
	for _, seat := range people.Seats {
		seats = append(seats, BrowserSeat{Franchise: seat.Franchise, Name: seat.Name})
		if seat.Franchise == people.YourFranchise {
			franchiseName = seat.Name
		}
	}
	return BrowserAdmission{League: league, Name: people.Name, Franchise: people.YourFranchise,
		FranchiseName: franchiseName, Role: people.YourRole, Seats: seats}
}
