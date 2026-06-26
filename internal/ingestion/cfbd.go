package ingestion

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
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

// --- CFBD season player-stats LONG-FORMAT plumbing ---
//
// The two college-production fetchers (collegeshare, collegedefense) consume the same
// /stats/player/season endpoint, which returns one ROW PER STAT in long format. They
// shared, verbatim, the row shape, the per-category fetch+decode, the stat-cell
// parsers, the share ratio, and the drop-ambiguous gsis emit. Those are extracted here
// at the second consumer so the long-format contract lives once (Codex M17); each
// fetcher still owns its category list, its accumulator types, and how it builds its
// own RAW record.

// CFBDStatRow is the slice of a CFBD long-format season player-stats record the college
// fetchers bind. CFBD returns more fields (conference, etc.); decoding is LENIENT
// (external 3rd-party boundary). Stat is a string in the source ("10", "1.5") and is
// parsed per statType by CFBDInt / CFBDFloat.
type CFBDStatRow struct {
	PlayerID string `json:"playerId"`
	Player   string `json:"player"`
	Team     string `json:"team"`
	StatType string `json:"statType"`
	Stat     string `json:"stat"`
}

// FetchCFBDCategory GETs one season player-stats category from baseURL (year+category
// query params, no team param — one long-format response covering all FBS players) and
// leniently decodes the rows. It composes GetCFBD; the caller folds the rows.
func FetchCFBDCategory(ctx context.Context, client *http.Client, baseURL, apiKey string, year int, category string) ([]CFBDStatRow, error) {
	url := baseURL + "?year=" + strconv.Itoa(year) + "&category=" + category
	body, err := GetCFBD(ctx, client, url, apiKey, DefaultMaxCFBDBytes)
	if err != nil {
		return nil, fmt.Errorf("cfbd: %w", err)
	}
	var rows []CFBDStatRow
	if err := json.Unmarshal(body, &rows); err != nil {
		return nil, fmt.Errorf("cfbd: decode %s: %w", category, err)
	}
	return rows, nil
}

// CFBDInt parses a long-format counting-stat cell: a missing/NA cell is a legitimate 0
// (the player recorded none), a present-but-unparseable cell fails loud (corruption,
// not zero). The error names the statType and player for source-pointing diagnostics.
func CFBDInt(r CFBDStatRow) (int, error) {
	v := strings.TrimSpace(r.Stat)
	if IsMissing(v) {
		return 0, nil
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return 0, fmt.Errorf("cfbd: %s %q for player %q: %w", r.StatType, v, r.PlayerID, err)
	}
	return n, nil
}

// CFBDFloat parses a fractional cell (yardage; sacks/TFL arrive in half increments)
// with the same missing-is-zero / malformed-fails-loud rule as CFBDInt.
func CFBDFloat(r CFBDStatRow) (float64, error) {
	v := strings.TrimSpace(r.Stat)
	if IsMissing(v) {
		return 0, nil
	}
	f, err := strconv.ParseFloat(v, 64)
	if err != nil {
		return 0, fmt.Errorf("cfbd: %s %q for player %q: %w", r.StatType, v, r.PlayerID, err)
	}
	return f, nil
}

// Share returns num/denom, or 0 when denom is 0 (a player with zero of a stat — the
// numerator is then 0 too, since a player's stat is part of his team's sum). This is
// the within-team market-share ratio both college fetchers emit.
func Share(num, denom float64) float64 {
	if denom == 0 {
		return 0
	}
	return num / denom
}

// EmitDropAmbiguous resolves each accumulated player to a gsis and builds the output
// map, applying the drop-ambiguous policy in ONE place: if two distinct players resolve
// to the same gsis (the espn->gsis bridge is many-to-one in rare MFL-duplicate cases),
// that gsis is dropped from the output entirely and stays dropped — a clean miss beats
// a silently mis-attributed line (matches the crosswalk bridge precedent). A player
// whose espn id does not resolve is skipped (ordinary miss). espnOf reads the player's
// espn id; build constructs the caller's RAW record for a resolved (player, gsis).
func EmitDropAmbiguous[P, T any](players map[string]*P, espnOf func(*P) string,
	resolve func(string) (string, bool), build func(*P, string) T) map[string]T {
	out := map[string]T{}
	poisoned := map[string]bool{}
	for _, p := range players {
		gsis, ok := resolve(espnOf(p))
		if !ok || poisoned[gsis] {
			continue
		}
		if _, dup := out[gsis]; dup {
			delete(out, gsis)
			poisoned[gsis] = true
			continue
		}
		out[gsis] = build(p, gsis)
	}
	return out
}
