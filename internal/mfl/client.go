package mfl

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/mflkey"
	"golang.org/x/time/rate"
)

// Client is the MFL HTTP transport client.
type Client struct {
	http    *http.Client
	limiter *rate.Limiter
	*hostState
	keySource func(context.Context) (mflkey.Key, error)
	keyedHTTP *http.Client
	backoffs  []time.Duration
}

type hostState struct {
	mu      sync.RWMutex
	host    string    // discovered league host (e.g. www47), cached
	hostFor string    // the year/league the host was discovered for
	hostAt  time.Time // when; a discovery older than hostTTL is redone
}

// ErrNonPositiveRate rejects a rate that is not positive and finite. rate.Limit(0) allows
// nothing, and NaN slips past rps <= 0 and makes Wait block forever.
var ErrNonPositiveRate = errors.New("mfl: requests-per-second must be a positive, real number")

// Option configures a Client at construction.
type Option func(*Client)

// WithTransport sends every request through rt: the app's archive recorder, or a test double
// serving canned MFL responses.
func WithTransport(rt http.RoundTripper) Option {
	return func(c *Client) { c.http.Transport = rt }
}

// New creates a Client at rps requests per second. An empty host means the canonical api
// host until DiscoverHost sets the league's.
func New(host string, rps float64, opts ...Option) (*Client, error) {
	if rps <= 0 || math.IsNaN(rps) {
		return nil, fmt.Errorf("%w: got %g", ErrNonPositiveRate, rps)
	}
	c := &Client{
		http:      &http.Client{Timeout: 15 * time.Second},
		limiter:   rate.NewLimiter(rate.Limit(rps), 1),
		hostState: &hostState{host: host},
		// 1s doubling to 60s, then return the error.
		backoffs: []time.Duration{
			1 * time.Second, 2 * time.Second, 4 * time.Second, 8 * time.Second,
			16 * time.Second, 32 * time.Second, 60 * time.Second,
		},
	}
	for _, o := range opts {
		o(c)
	}
	c.keyedHTTP = &http.Client{
		Transport:     c.http.Transport,
		Timeout:       c.http.Timeout,
		CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
	}
	return c, nil
}

// Do is the transport primitive: wait on the rate limit, execute, back off on 429. It and
// DiscoverHost are the only exported request methods.
func (c *Client) Do(ctx context.Context, req Request) (Response, error) {
	if req.Keyed {
		return c.doKeyed(ctx, req)
	}
	c.mu.RLock()
	host := c.host
	c.mu.RUnlock()

	switch {
	case req.Type == "league":
		// League lookups always use the api host, so a stale cached host cannot block rediscovery.
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

	resp, err := c.executeWithRetry(ctx, httpReq, c.http)
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

// hostTTL is how long a discovered host is trusted. MFL can move a league between servers, so a
// discovery is redone after it; within it, a refresh's several fetches share one discovery.
const hostTTL = 15 * time.Minute

// DiscoverHost looks up and caches the league's host server for a year and league, for hostTTL.
// On failure the host is unchanged.
func (c *Client) DiscoverHost(ctx context.Context, year string, leagueID string) error {
	key := year + "/" + leagueID
	c.mu.RLock()
	known := c.hostFor == key && time.Since(c.hostAt) < hostTTL
	c.mu.RUnlock()
	if known {
		return nil
	}
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
	c.host, c.hostFor, c.hostAt = sub, key, time.Now()
	c.mu.Unlock()
	return nil
}

// buildURL builds the API URL for a host, year, endpoint and params.
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
		// TYPE and JSON belong to the transport; a caller key matching either in any case is
		// dropped so it cannot add a second value.
		if strings.EqualFold(k, "TYPE") || strings.EqualFold(k, "JSON") ||
			strings.EqualFold(k, "APIKEY") {
			continue
		}
		q.Set(k, v)
	}
	q.Set("TYPE", endpoint)
	q.Set("JSON", "1")
	u.RawQuery = q.Encode()
	return u.String()
}

// executeWithRetry runs the request, backing off on HTTP 429.
func (c *Client) executeWithRetry(
	ctx context.Context, httpReq *http.Request, client *http.Client,
) (*http.Response, error) {
	backoffs := c.backoffs

	attempts := 0
	for {
		// Every attempt waits on the limiter, retries included; otherwise concurrent fetchers would
		// all wake from backoff and fire at once.
		if err := c.limiter.Wait(ctx); err != nil {
			return nil, fmt.Errorf("rate limiter wait failed: %w", err)
		}

		var reqToRun *http.Request
		if attempts == 0 {
			reqToRun = httpReq
		} else {
			reqToRun = httpReq.Clone(ctx)
		}

		resp, err := client.Do(reqToRun)
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

			// NewTimer + Stop, so a cancelled ctx does not leak the timer as time.After would.
			timer := time.NewTimer(dur)
			select {
			case <-ctx.Done():
				timer.Stop()
				return nil, fmt.Errorf("request cancelled during backoff: %w", ctx.Err())
			case <-timer.C:
			}
			continue
		}

		return resp, nil
	}
}

// extractSubdomain returns the host subdomain of an MFL base URL.
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
