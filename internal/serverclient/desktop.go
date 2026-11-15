package serverclient

import (
	"bytes"
	"context"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/accounts"
	"github.com/secureprospective/TheWarRoom/internal/serverhttp"
	"github.com/secureprospective/TheWarRoom/internal/serverversion"
)

const DesktopPath = "/desktop/native"

// desktopReplyLimit bounds one league-server reply held in memory. A real 32-team league's
// league-wide reads (players, rosters) run near 2 MB; 16 MiB leaves room and still stops a runaway reply.
const desktopReplyLimit = 16 << 20

// Desktop owns device credentials outside the webview. The webview receives neither the bearer nor
// the recent-auth proof. Its origin and league are fixed before the window opens.
type Desktop struct {
	mu            sync.Mutex
	generation    uint64
	admission     DesktopView
	watchCancel   context.CancelFunc
	changes       DesktopChanges
	changeHistory []DesktopChanges
	changed       chan struct{}
	client        *Client
	league        string
	bearer        string
	proof         string
	credentials   CredentialStore
	loaded        bool
	required      serverversion.Side
	update        func() error
	// seats are what the last "enter" read; an AI helper acts for one of them.
	seats          []DesktopSeat
	helperCommands []string
}

func NewDesktop(address, league string, roots *x509.CertPool, storage ...CredentialStore) (*Desktop, error) {
	u, err := url.Parse(address)
	if err != nil {
		return nil, fmt.Errorf("parse native server: %w", err)
	}
	if u.Scheme != "https" || u.Hostname() == "" || u.User != nil || u.Path != "" || u.RawQuery != "" || u.Fragment != "" || u.Opaque != "" ||
		!strings.HasPrefix(league, "native.") || len(league) <= 7 || len(league) > 128 {
		return nil, fmt.Errorf("native startup requires an HTTPS origin and native league ID")
	}
	client := &Client{base: address, http: newHTTPClient(roots, 15*time.Second, "native redirects refused")}
	credentials := newDeviceKeychain(address, league)
	if len(storage) > 0 && storage[0] != nil {
		credentials = storage[0]
	}
	return &Desktop{client: client, league: league, credentials: credentials}, nil
}

func (d *Desktop) Close() {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.stopWatchLocked()
	d.bearer, d.proof = "", ""
	d.generation++
	d.client.Close()
}

// RawDesktop is the local webview boundary, not a server credential or principal.
type RawDesktop struct {
	Action    string `json:"action"`
	Username  string `json:"username"`
	Password  string `json:"password"`
	Code      string `json:"code"`
	Helper    string `json:"helper"`
	Franchise string `json:"franchise"`
	Change    string `json:"change"`
}

func (r RawDesktop) String() string   { return "[desktop credentials redacted]" }
func (r RawDesktop) GoString() string { return r.String() }
func (r RawDesktop) Validate() error {
	switch r.Action {
	case "helper-issue", "helper-revoke", "helper-check", "helper-confirm":
		return r.validateHelper()
	}
	if r.Helper+r.Franchise+r.Change != "" {
		return fmt.Errorf("unexpected helper request")
	}
	switch r.Action {
	case "retry", "update", "status", "enter", "forget", "revoke-device", "revoke-all":
		if r.Username != "" || r.Password != "" || r.Code != "" {
			return fmt.Errorf("unexpected credentials")
		}
		return nil
	case "recover":
		return (accounts.RawRecovery{Username: r.Username, Code: r.Code, Password: r.Password}).Validate() //nolint:wrapcheck // Local boundary returns validation only.
	case "enroll":
		return (accounts.RawEnrollment{Username: r.Username, Password: r.Password, Code: r.Code}).Validate() //nolint:wrapcheck // Local boundary returns validation only.
	case "login", "totp-begin", "totp-confirm", "totp-step-up", "recovery-codes":
		if err := (accounts.RawCredentials{Username: r.Username, Password: r.Password}).Validate(); err != nil {
			return fmt.Errorf("desktop credentials: %w", err)
		}
		if r.Action == "totp-confirm" || r.Action == "totp-step-up" {
			if r.Code == "" {
				return fmt.Errorf("enter the six-digit authenticator code")
			}
			return (accounts.RawTOTP{Code: r.Code}).Validate() //nolint:wrapcheck // Local boundary returns validation only.
		}
		if r.Code != "" {
			return fmt.Errorf("unexpected code")
		}
		return nil
	default:
		return fmt.Errorf("unknown desktop action")
	}
}

// DesktopView contains only presentation data, never server credentials or cursors.
type DesktopView struct {
	AuthGeneration string `json:"authGeneration"`
	Server         string `json:"server"`
	League         string `json:"league"`
	Message        string `json:"message"`
	Update         string `json:"update,omitempty"`
	Name           string `json:"name"`
	Role           string `json:"role"`
	Franchise      string `json:"franchise"`
	FranchiseName  string `json:"franchiseName"`
	Seed           string `json:"seed"`
	// Connection is shown once, right after a helper is connected; Confirmation is one change's code.
	Connection    string        `json:"connection"`
	HelperID      string        `json:"helperId"`
	Confirmation  string        `json:"confirmation"`
	Seats         []DesktopSeat `json:"seats"`
	HelperPreview string        `json:"helperPreview,omitempty"`
	// RecoveryCodes are shown once, right after they are printed; the server keeps only digests.
	RecoveryCodes []string `json:"recoveryCodes,omitempty"`
}

type DesktopSeat struct {
	Franchise string `json:"franchise"`
	Name      string `json:"name"`
}

// Act serializes login/forget/read so a delayed response cannot revive a forgotten session.
func (d *Desktop) Act(ctx context.Context, raw RawDesktop) (view DesktopView, resultErr error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	bearer, proof := d.bearer, d.proof
	defer func() { //nolint:contextcheck // The window owns the watcher, not the sign-in request.
		if d.bearer != bearer || d.proof != proof || (resultErr == nil && changesSession(raw.Action)) {
			d.generation++
		}
		view.AuthGeneration = strconv.FormatUint(d.generation, 10)
		if resultErr == nil {
			switch raw.Action {
			case "login", "enter", "totp-step-up":
				d.startWatchLocked()
			}
		}
	}()
	view = d.initialView(raw.Action)
	if err := raw.Validate(); err != nil {
		return view, fmt.Errorf("check desktop request: %w", err)
	}
	defer func() {
		if update, ok := updateRequired(resultErr); ok {
			d.required = update.Update
			view.Update = string(update.Update)
		}
	}()
	if d.required == serverversion.Server && raw.Action == "retry" {
		d.required = ""
		return view, nil
	}
	if d.required != "" {
		view.Update = string(d.required)
		if raw.Action == "update" {
			return view, d.installUpdate()
		}
		if d.required == serverversion.App || raw.Action == "status" {
			return view, nil
		}
	}
	return d.actLocked(ctx, raw, view)
}

func (d *Desktop) initialView(action string) DesktopView {
	view := DesktopView{Server: d.client.base, League: d.league, Seats: []DesktopSeat{}}
	switch action {
	case "login", "forget", "revoke-device", "revoke-all", "recover":
		return view
	}
	view.Name, view.Role = d.admission.Name, d.admission.Role
	view.Franchise, view.FranchiseName = d.admission.Franchise, d.admission.FranchiseName
	if len(d.seats) > 0 {
		view.Seats = slices.Clone(d.seats)
	}
	return view
}

func (d *Desktop) status(ctx context.Context, view DesktopView) (DesktopView, error) {
	if d.bearer != "" && d.admission.Name != "" {
		return d.enter(ctx, view)
	}
	if d.bearer != "" {
		view.Message = "Saved device session available. Open your franchise to check access."
	}
	return view, nil
}

func changesSession(action string) bool {
	switch action {
	case "login", "enter", "forget", "revoke-device", "revoke-all":
		return true
	default:
		return false
	}
}

func (d *Desktop) actLocked(ctx context.Context, raw RawDesktop, view DesktopView) (DesktopView, error) {
	d.restore()
	switch raw.Action {
	case "update", "retry":
		return view, fmt.Errorf("no desktop update is required")
	case "status":
		return d.status(ctx, view)
	case "revoke-device", "revoke-all":
		return d.revoke(ctx, view, raw.Action == "revoke-all")
	case "forget":
		if err := d.clearCredential(); err != nil {
			return view, err
		}
		view.Message = "Signed out on this computer. To sign out your other devices too, sign in and choose Revoke all devices."
	case "enter":
		return d.enter(ctx, view)
	case "helper-issue", "helper-revoke", "helper-check", "helper-confirm":
		return d.helper(ctx, raw, view)
	case "login":
		return d.login(ctx, raw, view)
	case "recover":
		return d.recoverAccount(ctx, raw, view)
	case "recovery-codes":
		return d.printRecoveryCodes(ctx, raw, view)
	case "enroll":
		body, err := postDesktop(ctx, d, serverhttp.EnrollmentPath, enrollmentBody{1, enrollmentCredentials{raw.Code, raw.Username, raw.Password}})
		if err != nil {
			return view, err
		}
		var response rawDesktopEnrollment
		if err := decodeDesktop(body, &response); err != nil {
			return view, err
		}
		view.Message = "Account created. Sign in next. Your commissioner assigns your franchise separately."
	default:
		return d.authenticator(ctx, raw, view)
	}
	return view, nil
}

func (d *Desktop) recoverAccount(ctx context.Context, raw RawDesktop, view DesktopView) (DesktopView, error) {
	body, err := postDesktop(ctx, d, serverhttp.RecoveryPath, serverhttp.RawRecoveryRequest{Protocol: 1, Username: raw.Username, Code: raw.Code, Password: raw.Password})
	if err != nil {
		return view, err
	}
	var response rawDesktopRecovery
	if err := decodeDesktop(body, &response); err != nil {
		return view, err
	}
	if err := d.clearCredential(); err != nil {
		return view, err
	}
	view.Message = "Password reset. Sign in with your new password. Set up your authenticator again if required."
	return view, nil
}

// printRecoveryCodes replaces the account's unused codes. It needs a fresh authenticator check.
func (d *Desktop) printRecoveryCodes(ctx context.Context, raw RawDesktop, view DesktopView) (DesktopView, error) {
	if d.proof == "" {
		return view, fmt.Errorf("verify a fresh authenticator code first, then print new recovery codes")
	}
	request := serverhttp.RawRecoveryCodesRequest{Protocol: 1, Username: raw.Username, Password: raw.Password}
	body, err := postDesktop(ctx, d, serverhttp.RecoveryCodesPath, request)
	if err != nil {
		return view, err
	}
	var response serverhttp.RecoveryCodesResponse
	if err := decodeDesktop(body, &response); err != nil {
		return view, err
	}
	view.RecoveryCodes = response.Codes
	view.Message = "New recovery codes. Write them down or print them now; they will not be shown again. " +
		"Each works once. Your old codes no longer work."
	return view, nil
}

type desktopSnapshot struct {
	client                         *http.Client
	address, league, bearer, proof string
	generation                     uint64
}

// snapshotLocked captures the session while d.mu is held.
func (d *Desktop) snapshotLocked() desktopSnapshot {
	return desktopSnapshot{d.client.http, d.client.base, d.league, d.bearer, d.proof, d.generation}
}

// postDesktop is used by Act, which holds d.mu throughout its session changes.
func postDesktop[T desktopPayload](ctx context.Context, d *Desktop, path string, payload T) ([]byte, error) {
	body, unauthorized, err := postDesktopSnapshot(ctx, d.snapshotLocked(), path, payload)
	if unauthorized {
		_ = d.clearCredential()
	}
	return body, err
}

// postDesktopSnapshot uses only the immutable request session, never Desktop state.
func postDesktopSnapshot[T desktopPayload](ctx context.Context, session desktopSnapshot, path string, payload T) ([]byte, bool, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, false, fmt.Errorf("encode native request: %w", err)
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, session.address+path, bytes.NewReader(body))
	if err != nil {
		return nil, false, fmt.Errorf("create native request: %w", err)
	}
	request.Header.Set("Content-Type", "application/json")
	if session.bearer != "" {
		request.Header.Set("Authorization", "Bearer "+session.bearer)
	}
	if session.proof != "" {
		request.Header.Set("X-TWR-Recent-Auth", session.proof)
	}
	response, err := session.client.Do(request)
	if update, ok := updateRequired(err); ok {
		return nil, false, update
	}
	if err != nil {
		return nil, false, fmt.Errorf("cannot reach the native server securely; check the address, certificate and connection")
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK && response.StatusCode != http.StatusCreated {
		unauthorized := response.StatusCode == http.StatusUnauthorized
		return nil, unauthorized, desktopRefusal(path, response.StatusCode)
	}
	body, err = io.ReadAll(io.LimitReader(response.Body, desktopReplyLimit+1))
	if err != nil || len(body) > desktopReplyLimit || response.Header.Get("Content-Type") != "application/json" {
		return nil, false, fmt.Errorf("native server returned an unreadable response; try again")
	}
	return body, false, nil
}
