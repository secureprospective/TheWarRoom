package livegame

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

type Endpoints struct {
	ESPN    string `json:"espn"`
	Yahoo   string `json:"yahoo"`
	Summary string `json:"summary"`
	Plays   string `json:"plays"`
}
type Transport interface {
	Do(*http.Request) (*http.Response, error)
}

func (r Endpoints) Validate() error {
	for name, s := range map[string]string{"espn": r.ESPN, "yahoo": r.Yahoo, "summary": r.Summary, "plays": r.Plays} {
		u, err := url.Parse(strings.ReplaceAll(s, "{id}", "1"))
		if err != nil {
			return fmt.Errorf("%s endpoint: %w", name, err)
		}
		if u.Scheme != "https" || u.Host == "" || u.User != nil {
			return fmt.Errorf("%s endpoint: requires HTTPS without credentials", name)
		}
	}
	return nil
}
func HTTPFetch(transport Transport, endpoints Endpoints) Fetch {
	return func(ctx context.Context, kind, id string) (result []byte, failure error) {
		if err := endpoints.Validate(); err != nil {
			return nil, err
		}
		if id != "" && !digits(id) {
			return nil, fmt.Errorf("event id: invalid")
		}
		address := map[string]string{"espn": endpoints.ESPN, "yahoo": endpoints.Yahoo, "summary": endpoints.Summary, "plays": endpoints.Plays}[kind]
		if address == "" {
			return nil, fmt.Errorf("source kind: unknown")
		}
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, strings.ReplaceAll(address, "{id}", id), nil)
		if err != nil {
			return nil, fmt.Errorf("livegame request: %w", err)
		}
		resp, err := transport.Do(req)
		if err != nil {
			return nil, fmt.Errorf("livegame transport: %w", err)
		}
		defer func() {
			if err := resp.Body.Close(); err != nil && failure == nil {
				failure = fmt.Errorf("livegame close: %w", err)
			}
		}()
		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("livegame status: HTTP %d", resp.StatusCode)
		}
		const limit = 4 << 20
		body, err := io.ReadAll(io.LimitReader(resp.Body, limit+1))
		if err != nil {
			return nil, fmt.Errorf("livegame body: %w", err)
		}
		if len(body) > limit {
			return nil, fmt.Errorf("livegame body: over %d bytes", limit)
		}
		return body, nil
	}
}
