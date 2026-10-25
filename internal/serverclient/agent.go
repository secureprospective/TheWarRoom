package serverclient

import (
	"bytes"
	"context"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/serverhttp"
	"github.com/zalando/go-keyring"
)

// ErrRefused is a change the league would not record. Bigger changes (G2/G3) also need a person
// to confirm that exact change in The War Room; preview says what is blocking it.
var ErrRefused = errors.New("the league did not record this change. Preview it to see why; bigger changes also need the commissioner to confirm them in The War Room")

// Agent is the twr CLI's connection: an agent token issued by a signed-in commissioner, used
// over the same HTTPS dispatch as the desktop. It never logs in, mints tokens or confirms.
type Agent struct {
	client *Client
	league string
	bearer string
}

// NewAgent checks the origin and league the same way the desktop does. nil roots uses the
// system's trusted certificates, which is right for a public league hostname.
func NewAgent(address, league, bearer string, roots *x509.CertPool) (*Agent, error) {
	u, err := url.Parse(address)
	if err != nil || u.Scheme != "https" || u.Hostname() == "" || u.User != nil || (u.Path != "" && u.Path != "/") ||
		u.RawQuery != "" || u.Fragment != "" || !strings.HasPrefix(league, "native.") || len(league) <= 7 || len(league) > 128 {
		return nil, fmt.Errorf("use the league's https:// address and its league ID (native.…)")
	}
	if bearer == "" || len(bearer) > 256 {
		return nil, fmt.Errorf("paste the whole connection token from The War Room")
	}
	transport := &http.Transport{TLSClientConfig: &tls.Config{MinVersion: tls.VersionTLS13, RootCAs: roots}}
	client := &Client{base: strings.TrimSuffix(address, "/"), http: &http.Client{Timeout: 15 * time.Second, Transport: transport,
		CheckRedirect: func(*http.Request, []*http.Request) error { return fmt.Errorf("redirects refused") }}}
	return &Agent{client: client, league: league, bearer: bearer}, nil
}

func (a *Agent) Ledger(ctx context.Context) (commissioner.Ledger, error) {
	var ledger commissioner.Ledger
	body, err := a.post(ctx, serverhttp.LedgerPath, serverhttp.RawNativeRequest{Protocol: 1, League: a.league}, "")
	if err != nil {
		return ledger, err
	}
	if err := json.Unmarshal(body, &ledger); err != nil {
		return ledger, fmt.Errorf("the league server sent an unreadable ledger; try again")
	}
	return ledger, nil
}

// Dispatch previews by default; execute needs mode "execute", and a G2/G3 change also needs the
// person's confirmation code. The expected revision is read fresh when the caller leaves it 0.
func (a *Agent) Dispatch(ctx context.Context, raw commissioner.RawCommand, mode, confirmation string) (json.RawMessage, error) {
	raw.League = a.league
	if raw.Expected == 0 {
		ledger, err := a.Ledger(ctx)
		if err != nil {
			return nil, err
		}
		raw.Expected = ledger.Revision
	}
	if mode == "" {
		mode = "preview"
	}
	body, err := a.post(ctx, serverhttp.DispatchPath, serverhttp.RawNativeRequest{Protocol: 1, Command: &raw, Mode: mode}, confirmation)
	if err != nil {
		return nil, err
	}
	return json.RawMessage(body), nil
}

func (a *Agent) post(ctx context.Context, path string, payload serverhttp.RawNativeRequest, confirmation string) ([]byte, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("encode league request: %w", err)
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, a.client.base+path, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create league request: %w", err)
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Authorization", "Bearer "+a.bearer)
	if confirmation != "" {
		request.Header.Set("X-TWR-Human-Confirmation", confirmation)
	}
	response, err := a.client.http.Do(request)
	if err != nil {
		return nil, fmt.Errorf("can't reach the league server securely; check the address and your internet connection")
	}
	defer func() { _ = response.Body.Close() }()
	switch response.StatusCode {
	case http.StatusOK:
	case http.StatusUnauthorized:
		return nil, fmt.Errorf("the connection token was refused: it may have expired or been revoked. Ask the commissioner for a new one, then run twr connect")
	case http.StatusConflict:
		return nil, ErrRefused
	case http.StatusServiceUnavailable:
		return nil, fmt.Errorf("the league server is busy; try again in a minute")
	default:
		return nil, fmt.Errorf("the league refused this request (%d); this token may not cover it", response.StatusCode)
	}
	body, err = io.ReadAll(io.LimitReader(response.Body, (1<<20)+1))
	if err != nil || len(body) > 1<<20 || response.Header.Get("Content-Type") != "application/json" {
		return nil, fmt.Errorf("the league server sent an unreadable reply; try again")
	}
	return body, nil
}

// AgentKeychain keeps the CLI's token in the system keychain, never in a file.
type AgentKeychain struct{ account string }

func NewAgentKeychain(origin, league string) AgentKeychain {
	return AgentKeychain{account: fmt.Sprintf("agent-token:%x", sha256.Sum256([]byte(origin+"\x00"+league)))}
}

func (s AgentKeychain) Get() (string, error) {
	secret, err := keyring.Get("TheWarRoom.Agent", s.account)
	if err != nil {
		return "", fmt.Errorf("no saved connection for this league; run twr connect first")
	}
	return secret, nil
}

func (s AgentKeychain) Set(secret string) error {
	if err := keyring.Set("TheWarRoom.Agent", s.account, secret); err != nil {
		return fmt.Errorf("save the connection token in this computer's keychain: %w", err)
	}
	return nil
}

func (s AgentKeychain) Delete() error {
	if err := keyring.Delete("TheWarRoom.Agent", s.account); err != nil && !errors.Is(err, keyring.ErrNotFound) {
		return fmt.Errorf("remove the saved connection token: %w", err)
	}
	return nil
}
