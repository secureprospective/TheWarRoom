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
	"strings"
	"sync"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/accounts"
	"github.com/secureprospective/TheWarRoom/internal/serverhttp"
)

const DesktopPath = "/desktop/native"

// Desktop owns device credentials outside the webview. The webview receives neither the bearer nor
// the recent-auth proof. Its origin and league are fixed before the window opens.
type Desktop struct {
	mu          sync.Mutex
	client      *Client
	league      string
	bearer      string
	proof       string
	credentials CredentialStore
	loaded      bool
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
	d.bearer, d.proof = "", ""
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
	case "status", "enter", "forget", "revoke-device", "revoke-all":
		if r.Username != "" || r.Password != "" || r.Code != "" {
			return fmt.Errorf("unexpected credentials")
		}
		return nil
	case "recover":
		return (accounts.RawRecovery{Username: r.Username, Code: r.Code, Password: r.Password}).Validate() //nolint:wrapcheck // Local boundary returns validation only.
	case "enroll":
		return (accounts.RawEnrollment{Username: r.Username, Password: r.Password, Code: r.Code}).Validate() //nolint:wrapcheck // Local boundary returns validation only.
	case "login", "totp-begin", "totp-confirm", "totp-step-up":
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
	Server        string `json:"server"`
	League        string `json:"league"`
	Message       string `json:"message"`
	Name          string `json:"name"`
	Role          string `json:"role"`
	Franchise     string `json:"franchise"`
	FranchiseName string `json:"franchiseName"`
	Seed          string `json:"seed"`
	// Connection is shown once, right after a helper is connected; Confirmation is one change's code.
	Connection    string        `json:"connection"`
	HelperID      string        `json:"helperId"`
	Confirmation  string        `json:"confirmation"`
	Seats         []DesktopSeat `json:"seats"`
	HelperPreview string        `json:"helperPreview,omitempty"`
}

type DesktopSeat struct {
	Franchise string `json:"franchise"`
	Name      string `json:"name"`
}

// Act serializes login/forget/read so a delayed response cannot revive a forgotten session.
func (d *Desktop) Act(ctx context.Context, raw RawDesktop) (DesktopView, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	view := DesktopView{Server: d.client.base, League: d.league, Seats: []DesktopSeat{}}
	if err := raw.Validate(); err != nil {
		return view, fmt.Errorf("check desktop request: %w", err)
	}
	d.restore()
	switch raw.Action {
	case "status":
		if d.bearer != "" {
			view.Message = "Saved device session available. Open your franchise to check access."
		}
		return view, nil
	case "revoke-device", "revoke-all":
		return d.revoke(ctx, view, raw.Action == "revoke-all")
	case "forget":
		if err := d.clearCredential(); err != nil {
			return view, err
		}
		view.Message = "Signed out and removed the saved device session. Server-side device revocation is separate."
	case "enter":
		return d.enter(ctx, view)
	case "helper-issue", "helper-revoke", "helper-check", "helper-confirm":
		return d.helper(ctx, raw, view)
	case "login":
		return d.login(ctx, raw, view)
	case "recover":
		return d.recoverAccount(ctx, raw, view)
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

// post accepts only the closed set of desktop requests. Redacting domain types
// are intentionally not marshaled: they cannot carry passwords over HTTPS.
func postDesktop[T desktopPayload](ctx context.Context, d *Desktop, path string, payload T) ([]byte, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("encode native request: %w", err)
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, d.client.base+path, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create native request: %w", err)
	}
	request.Header.Set("Content-Type", "application/json")
	if d.bearer != "" {
		request.Header.Set("Authorization", "Bearer "+d.bearer)
	}
	if d.proof != "" {
		request.Header.Set("X-TWR-Recent-Auth", d.proof)
	}
	response, err := d.client.http.Do(request)
	if update, ok := updateRequired(err); ok {
		return nil, update
	}
	if err != nil {
		return nil, fmt.Errorf("cannot reach the native server securely; check the address, certificate and connection")
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK && response.StatusCode != http.StatusCreated {
		if response.StatusCode == http.StatusUnauthorized {
			_ = d.clearCredential()
		}
		return nil, desktopRefusal(path, response.StatusCode)
	}
	body, err = io.ReadAll(io.LimitReader(response.Body, (256<<10)+1))
	if err != nil || len(body) > 256<<10 || response.Header.Get("Content-Type") != "application/json" {
		return nil, fmt.Errorf("native server returned an unreadable response; try again")
	}
	return body, nil
}
