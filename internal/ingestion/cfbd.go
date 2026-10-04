package ingestion

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Shared CollegeFootballData (CFBD) plumbing. CFBD is a bearer-authed JSON API, so it shares
// neither the CSV helpers nor the MFL client. GetCFBD returns raw bytes; each fetcher decodes
// leniently into its own concrete type.

// DefaultMaxCFBDBytes caps a CFBD response. The largest in use is about 3.5 MiB.
const DefaultMaxCFBDBytes = 32 << 20

// NewCFBDClient returns a client with HTTP/2 disabled: api.collegefootballdata.com answers
// h2 with PROTOCOL_ERROR from CT105 (verified 2026-06-21), and an empty TLSNextProto map forces
// HTTP/1.1.
func NewCFBDClient(timeout time.Duration) *http.Client {
	return &http.Client{
		Timeout: timeout,
		Transport: &http.Transport{
			TLSNextProto: map[string]func(string, *tls.Conn) http.RoundTripper{},
		},
	}
}

// GetCFBD performs a bearer-authed GET, checks the status and fails rather than truncate a
// body over maxBytes. apiKey must be trimmed: CT105's CFBD_API_KEY carries a trailing newline
// that Go's header validation rejects.
func GetCFBD(ctx context.Context, client *http.Client, url, apiKey string, maxBytes int64) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("ingestion: build CFBD request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("ingestion: fetch CFBD %s: %w", url, err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("ingestion: CFBD %s unexpected status %d", url, resp.StatusCode)
	}

	// Read one byte past the cap; consuming it means the body was over budget.
	lr := &io.LimitedReader{R: resp.Body, N: maxBytes + 1}
	body, err := io.ReadAll(lr)
	if err != nil {
		return nil, fmt.Errorf("ingestion: read CFBD %s: %w", url, err)
	}
	if lr.N == 0 {
		return nil, fmt.Errorf("ingestion: CFBD %s exceeds %d-byte cap", url, maxBytes)
	}
	return body, nil
}

// Share returns num/denom, or 0 when denom is 0: the player's within-team market share.
func Share(num, denom float64) float64 {
	if denom == 0 {
		return 0
	}
	return num / denom
}
