package serverclient

import (
	"bytes"
	"context"
	"crypto/tls"
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

// Desktop owns memory-only credentials. The webview receives neither the bearer nor
// the recent-auth proof. Its origin and league are fixed before the window opens.
type Desktop struct {
	mu     sync.Mutex
	client *Client
	league string
	bearer string
	proof  string
}

func NewDesktop(address, league string, roots *x509.CertPool) (*Desktop, error) {
	u, err := url.Parse(address)
	if err != nil {
		return nil, fmt.Errorf("parse native server: %w", err)
	}
	if u.Scheme != "https" || u.Hostname() == "" || u.User != nil || u.Path != "" || u.RawQuery != "" || u.Fragment != "" || u.Opaque != "" ||
		!strings.HasPrefix(league, "native.") || len(league) <= 7 || len(league) > 128 {
		return nil, fmt.Errorf("native startup requires an HTTPS origin and native league ID")
	}
	transport := &http.Transport{TLSClientConfig: &tls.Config{MinVersion: tls.VersionTLS13, RootCAs: roots}}
	client := &Client{base: address, http: &http.Client{Timeout: 15 * time.Second, Transport: transport,
		CheckRedirect: func(_ *http.Request, _ []*http.Request) error { return fmt.Errorf("native redirects refused") }}}
	return &Desktop{client: client, league: league}, nil
}

func (d *Desktop) Close() {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.bearer, d.proof = "", ""
	d.client.Close()
}

// RawDesktop is the local webview boundary, not a server credential or principal.
type RawDesktop struct {
	Action   string `json:"action"`
	Username string `json:"username"`
	Password string `json:"password"`
	Code     string `json:"code"`
}

func (r RawDesktop) String() string   { return "[desktop credentials redacted]" }
func (r RawDesktop) GoString() string { return r.String() }
func (r RawDesktop) Validate() error {
	switch r.Action {
	case "status", "enter", "forget":
		if r.Username != "" || r.Password != "" || r.Code != "" {
			return fmt.Errorf("unexpected credentials")
		}
		return nil
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
}

// Act serializes login/forget/read so a delayed response cannot revive a forgotten session.
func (d *Desktop) Act(ctx context.Context, raw RawDesktop) (DesktopView, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	view := DesktopView{Server: d.client.base, League: d.league}
	if err := raw.Validate(); err != nil {
		return view, fmt.Errorf("check desktop request: %w", err)
	}
	switch raw.Action {
	case "status":
		return view, nil
	case "forget":
		d.bearer, d.proof = "", ""
		view.Message = "Signed out on this window. Server-side device revocation is separate."
	case "enter":
		return d.enter(ctx, view)
	case "login":
		d.bearer, d.proof = "", ""
		body, err := postDesktop(ctx, d, serverhttp.LoginPath, loginBody{1, raw.Username, raw.Password, "The War Room desktop"})
		if err != nil {
			return view, err
		}
		var response rawDesktopLogin
		if err := decodeDesktop(body, &response); err != nil {
			return view, err
		}
		d.bearer = response.Bearer
		view.Message = "Signed in. Open your franchise, or verify your authenticator if you are a commissioner."
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
	if err != nil {
		return nil, fmt.Errorf("cannot reach the native server securely; check the address, certificate and connection")
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK && response.StatusCode != http.StatusCreated {
		if response.StatusCode == http.StatusUnauthorized {
			d.bearer, d.proof = "", ""
		}
		return nil, desktopRefusal(path, response.StatusCode)
	}
	body, err = io.ReadAll(io.LimitReader(response.Body, (256<<10)+1))
	if err != nil || len(body) > 256<<10 || response.Header.Get("Content-Type") != "application/json" {
		return nil, fmt.Errorf("native server returned an unreadable response; try again")
	}
	return body, nil
}
