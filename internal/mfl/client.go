// internal/mfl/client.go
package mfl

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

// Client is the MFL HTTP transport client.
type Client struct {
	http    *http.Client
	limiter *rate.Limiter // token bucket; back off on 429, do NOT retry-storm
	mu      sync.RWMutex
	host    string // discovered league host (e.g. www47), cached
}

// New creates and initializes a new Client.
func New(host string, rps float64) *Client {
	return &Client{
		http:    &http.Client{Timeout: 15 * time.Second},
		limiter: rate.NewLimiter(rate.Limit(rps), 1),
		host:    host,
	}
}

// Do is the ONLY exported surface. Transport in, transport out. No Player, no Schedule.
func (c *Client) Do(ctx context.Context, req Request) (Response, error) {
	if err := c.limiter.Wait(ctx); err != nil {
		return Response{}, fmt.Errorf("rate limiter wait failed: %w", err)
	}

	c.mu.RLock()
	host := c.host
	c.mu.RUnlock()

	switch {
	case req.Type == "league":
		// Host discovery (and any other "league" lookup) always targets the
		// canonical api host, regardless of c.host — a stale or down cached
		// host must not block re-discovery.
		host = "api"
	case req.Params == nil || req.Params["L"] == "":
		host = "api"
	case host == "":
		host = "api"
	}

	urlStr := c.buildURL(host, req.Year, req.Type, req.Params)

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, urlStr, nil)
	if err != nil {
		return Response{}, fmt.Errorf("failed to create http request: %w", err)
	}

	resp, err := c.executeWithRetry(ctx, httpReq)
	if err != nil {
		return Response{}, fmt.Errorf("failed to execute request: %w", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return Response{}, fmt.Errorf("failed to read response body: %w", err)
	}

	return Response{
		StatusCode: resp.StatusCode,
		Body:       body,
	}, nil
}

// DiscoverHost queries the MFL league endpoint to discover and cache the league's active host server.
// If the discovery call fails, c.host remains unchanged (or defaults to the initial host).
func (c *Client) DiscoverHost(ctx context.Context, year string, leagueID string) error {
	req := Request{
		Type: "league",
		Year: year,
		Params: map[string]string{
			"L": leagueID,
		},
	}

	resp, err := c.Do(ctx, req)
	if err != nil {
		return fmt.Errorf("failed to execute discovery request: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("discovery request returned unexpected status code: %d", resp.StatusCode)
	}

	var lr leagueResponse
	if err := json.Unmarshal(resp.Body, &lr); err != nil {
		return fmt.Errorf("failed to parse discovery response JSON: %w", err)
	}

	baseURL := lr.League.BaseURL
	sub, err := extractSubdomain(baseURL)
	if err != nil {
		return fmt.Errorf("failed to extract subdomain from base URL %q: %w", baseURL, err)
	}

	c.mu.Lock()
	c.host = sub
	c.mu.Unlock()
	return nil
}

// buildURL constructs the MFL API URL for the given host, year, type, and params.
func (c *Client) buildURL(host, year, endpoint string, params map[string]string) string {
	h := host
	if h == "" {
		h = "api"
	}

	var domain string
	if !strings.Contains(h, ".") {
		domain = h + ".myfantasyleague.com"
	} else {
		domain = h
	}

	u := url.URL{
		Scheme: "https",
		Host:   domain,
		Path:   fmt.Sprintf("/%s/export", year),
	}
	q := u.Query()
	for k, v := range params {
		q.Set(k, v)
	}
	// TYPE and JSON are transport-mandated; set last so caller-supplied
	// params can never override them.
	q.Set("TYPE", endpoint)
	q.Set("JSON", "1")
	u.RawQuery = q.Encode()
	return u.String()
}

// executeWithRetry executes the HTTP request, with exponential backoff on HTTP 429.
func (c *Client) executeWithRetry(ctx context.Context, httpReq *http.Request) (*http.Response, error) {
	backoffs := []time.Duration{
		1 * time.Second,
		2 * time.Second,
		4 * time.Second,
		8 * time.Second,
		16 * time.Second,
		32 * time.Second,
		60 * time.Second,
	}

	attempts := 0
	for {
		var reqToRun *http.Request
		if attempts == 0 {
			reqToRun = httpReq
		} else {
			reqToRun = httpReq.Clone(ctx)
		}

		resp, err := c.http.Do(reqToRun)
		if err != nil {
			return nil, fmt.Errorf("failed to execute HTTP request: %w", err)
		}

		if resp.StatusCode == http.StatusTooManyRequests {
			_ = resp.Body.Close()

			if attempts >= len(backoffs) {
				return nil, fmt.Errorf("rate limited by MFL (429) and exhausted all %d retry attempts", len(backoffs))
			}

			dur := backoffs[attempts]
			attempts++

			select {
			case <-ctx.Done():
				return nil, fmt.Errorf("request cancelled during backoff: %w", ctx.Err())
			case <-time.After(dur):
			}
			continue
		}

		return resp, nil
	}
}

// extractSubdomain extracts the host subdomain from the MFL base URL.
func extractSubdomain(baseURL string) (string, error) {
	if baseURL == "" {
		return "", fmt.Errorf("empty baseURL")
	}
	parsed, err := url.Parse(baseURL)
	if err != nil {
		return "", fmt.Errorf("failed to parse baseURL %q: %w", baseURL, err)
	}
	host := parsed.Host
	if host == "" {
		return "", fmt.Errorf("empty host in baseURL %q", baseURL)
	}

	if strings.Contains(host, ":") {
		h, _, err := net.SplitHostPort(host)
		if err == nil {
			host = h
		}
	}

	const suffix = ".myfantasyleague.com"
	if strings.HasSuffix(strings.ToLower(host), suffix) {
		sub := host[:len(host)-len(suffix)]
		if sub != "" {
			return sub, nil
		}
	}
	return host, nil
}
