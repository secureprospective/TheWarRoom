package ingestion

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"net/http"
	"time"
)

// This file holds the shared CollegeFootballData (CFBD) plumbing the CFBD Layer-1
// fetchers reuse (schooltier, collegeshare, …). CFBD is an AUTHED JSON API (bearer
// token), NOT a static CSV like nflverse/dynastyprocess — so it does not use the
// extcsv.go helpers (no by-name CSV column binding). It is also NOT the MFL API — no
// host discovery, no MFL error envelope. The CFBD-specific concerns shared here are
// exactly two: an HTTP/1.1-pinned client (CT105 gets an h2 PROTOCOL_ERROR against
// api.collegefootballdata.com), and a bearer-authed, status-checked, byte-capped GET.
// Extracted from the schooltier fetcher (the first CFBD instance) before a second
// CFBD fetcher could copy-paste it (Codex M17).
//
// Each fetcher still owns its own LENIENT decode into a CONCRETE type (no
// DisallowUnknownFields — external 3rd-party boundary): GetCFBD returns the raw body
// bytes and the caller json.Unmarshal's them into its package-local shape. Returning
// bytes (not decoding here) keeps the shared helper free of an interface{} decode
// sink and leaves every fetcher's types concrete and self-documenting.

// DefaultMaxCFBDBytes bounds a CFBD JSON response so a runaway or hostile source
// cannot exhaust memory at the boundary (Codex M15). The largest CFBD response in
// use is the per-category season stats feed (all FBS players, long-format) at ~3.5
// MiB; this ceiling leaves generous headroom while still failing loud well before
// CT105's 2 GB is at risk. A fetcher expecting a larger body must raise it
// deliberately (mirroring extcsv.DefaultMaxCSVBytes).
const DefaultMaxCFBDBytes = 32 << 20

// NewCFBDClient returns an *http.Client for the CFBD API with HTTP/2 DISABLED.
// api.collegefootballdata.com returns an HTTP/2 PROTOCOL_ERROR from CT105 (verified
// live 2026-06-21: curl --http1.1 succeeds where default h2 fails); pinning the
// transport's TLSNextProto to a non-nil empty map forces HTTP/1.1.
func NewCFBDClient(timeout time.Duration) *http.Client {
	return &http.Client{
		Timeout: timeout,
		Transport: &http.Transport{
			TLSNextProto: map[string]func(string, *tls.Conn) http.RoundTripper{},
		},
	}
}

// GetCFBD performs a bearer-authed GET against a CFBD endpoint, validates the status,
// caps the body at maxBytes (failing loud over the cap rather than silently
// truncating a tail of records), and returns the response body bytes. The caller
// decodes them into its own concrete type with a lenient json.Unmarshal.
//
// apiKey must be clean: the CT105 env's CFBD_API_KEY carries trailing newlines that
// Go's strict header validation rejects (curl tolerates them), so callers TrimSpace
// it before passing it here. ctx bounds the request; the response body is always
// closed.
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

	// Read one byte past the cap: if the LimitedReader is fully consumed (N == 0) the
	// source was over budget, so we error instead of decoding a truncated body.
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
