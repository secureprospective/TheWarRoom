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

// CFBD /stats/player/season returns one row per stat. collegeshare and collegedefense share
// this row shape, fetch and parsing; each owns its categories and output records.

// CFBDStatRow is the part of a long-format stats row the college fetchers read. Stat is a
// string ("10", "1.5") parsed by CFBDInt or CFBDFloat.
type CFBDStatRow struct {
	PlayerID string `json:"playerId"`
	Player   string `json:"player"`
	Team     string `json:"team"`
	StatType string `json:"statType"`
	Stat     string `json:"stat"`
}

// FetchCFBDCategory fetches one stats category for all FBS players in a season.
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

// CFBDInt parses a counting stat: missing is 0, unparseable is an error.
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

// CFBDFloat is CFBDInt for fractional stats (yards, half sacks).
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

// Share returns num/denom, or 0 when denom is 0: the player's within-team market share.
func Share(num, denom float64) float64 {
	if denom == 0 {
		return 0
	}
	return num / denom
}

// EmitDropAmbiguous maps each player to a gsis id and builds the output. When two players
// resolve to one gsis, that gsis is dropped entirely: a clean miss beats a mis-attributed line.
// An unresolved espn id is skipped.
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
