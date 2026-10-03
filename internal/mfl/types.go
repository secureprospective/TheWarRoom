// Package mfl is the MFL API transport: rate limiting, host discovery and 429 backoff. It
// knows no domain types.
package mfl

// Request represents a transport-level MFL API request.
type Request struct {
	Type   string            // The endpoint TYPE (e.g. "league", "rosters", "players")
	Year   string            // The season year (e.g. "2026")
	Params map[string]string // Additional query parameters (e.g. L, W)
}

// Response represents a transport-level MFL API response.
type Response struct {
	StatusCode int    // HTTP status code
	Body       []byte // Unprocessed JSON response body
}

// leagueResponse is the part of the league endpoint used for host discovery.
type leagueResponse struct {
	League struct {
		BaseURL string `json:"baseURL"`
	} `json:"league"`
}
