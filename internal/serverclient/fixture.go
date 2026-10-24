// Package serverclient keeps native HTTPS transport and desktop credentials outside the webview.
package serverclient

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/serverhttp"
)

type Client struct {
	base string
	http *http.Client
}

// NewFixture requires an explicit trust root; certificate verification cannot be disabled.
func NewFixture(address string, roots *x509.CertPool) (*Client, error) {
	u, err := url.Parse(address)
	if err != nil {
		return nil, fmt.Errorf("parse fixture URL: %w", err)
	}
	ip := net.ParseIP(u.Hostname())
	if u.Scheme != "https" || ip == nil || !ip.IsLoopback() || u.User != nil ||
		u.RawQuery != "" || u.Fragment != "" || u.Path != "" || roots == nil {
		return nil, fmt.Errorf("fixture client requires a loopback HTTPS origin and trusted CA")
	}
	transport := &http.Transport{TLSClientConfig: &tls.Config{MinVersion: tls.VersionTLS13, RootCAs: roots.Clone()}}
	return &Client{base: address, http: &http.Client{Timeout: 5 * time.Second, Transport: transport,
		CheckRedirect: func(_ *http.Request, _ []*http.Request) error { return fmt.Errorf("fixture redirects are not allowed") },
	}}, nil
}

func (c *Client) Close() { c.http.CloseIdleConnections() }

func (c *Client) FixtureLedger(ctx context.Context) (commissioner.Ledger, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, c.base+serverhttp.FixturePath, nil)
	if err != nil {
		return commissioner.Ledger{}, fmt.Errorf("create fixture request: %w", err)
	}
	response, err := c.http.Do(request)
	if err != nil {
		return commissioner.Ledger{}, fmt.Errorf("request fixture ledger: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK || response.Header.Get("Content-Type") != "application/json" {
		return commissioner.Ledger{}, fmt.Errorf("fixture server refused reading (HTTP %d)", response.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(response.Body, (256<<10)+1))
	if err != nil {
		return commissioner.Ledger{}, fmt.Errorf("read fixture response: %w", err)
	}
	if len(body) > 256<<10 {
		return commissioner.Ledger{}, fmt.Errorf("fixture response too large")
	}
	var raw serverhttp.RawLedgerResponse
	if err := json.Unmarshal(body, &raw); err != nil {
		return commissioner.Ledger{}, fmt.Errorf("decode fixture response: %w", err)
	}
	if err := raw.Validate(); err != nil {
		return commissioner.Ledger{}, fmt.Errorf("validate fixture response: %w", err)
	}
	return raw.Ledger, nil
}
