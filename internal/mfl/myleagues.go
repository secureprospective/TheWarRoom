package mfl

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
)

// MyFranchise resolves the saved login's team without using a discovered league host.
func (c *Client) MyFranchise(ctx context.Context, year, league string) (string, error) {
	resp, err := c.doMyLeagues(ctx, year)
	if err != nil {
		return "", err
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("mfl: my leagues returned HTTP %d", resp.StatusCode)
	}
	var raw RawMyLeagues
	if err := json.Unmarshal(resp.Body, &raw); err != nil {
		return "", fmt.Errorf("mfl: decode my leagues: %w", err)
	}
	if err := raw.Validate(); err != nil {
		return "", err
	}
	for _, row := range raw.Leagues.League {
		if row.LeagueID == league {
			return row.FranchiseID, nil
		}
	}
	return "", errors.New("mfl: login has no franchise in this league")
}

// RawMyLeagues accepts MFL's singleton and list encodings at the wire boundary.
type RawMyLeagues struct {
	Leagues struct {
		League rawLoginLeagues `json:"league"`
	} `json:"leagues"`
}

type rawLoginLeague struct {
	LeagueID    string `json:"league_id"`
	FranchiseID string `json:"franchise_id"`
}
type rawLoginLeagues []rawLoginLeague

func (r *rawLoginLeagues) UnmarshalJSON(data []byte) error {
	if strings.HasPrefix(strings.TrimSpace(string(data)), "{") {
		var row rawLoginLeague
		if err := json.Unmarshal(data, &row); err != nil {
			return fmt.Errorf("decode login league: %w", err)
		}
		*r = []rawLoginLeague{row}
		return nil
	}
	var rows []rawLoginLeague
	if err := json.Unmarshal(data, &rows); err != nil {
		return fmt.Errorf("decode login leagues: %w", err)
	}
	*r = rows
	return nil
}

func (r RawMyLeagues) Validate() error {
	seen := make(map[string]bool)
	for _, row := range r.Leagues.League {
		if !decimalID(row.LeagueID) || !decimalID(row.FranchiseID) || seen[row.LeagueID] {
			return errors.New("mfl: invalid login league or franchise")
		}
		seen[row.LeagueID] = true
	}
	return nil
}

func decimalID(s string) bool {
	if s == "" {
		return false
	}
	for _, c := range s {
		if c < '0' || c > '9' {
			return false
		}
	}
	return true
}

func (c *Client) doMyLeagues(ctx context.Context, year string) (out Response, err error) {
	if c.keySource == nil {
		return out, errors.New("mfl: keyed request has no key source")
	}
	key, err := c.keySource(ctx)
	secret := key.Reveal()
	cleanURL := c.buildURL("api", year, "myleagues", nil)
	defer func() { err = scrubError(err, cleanURL, secret) }()
	if err != nil {
		return out, fmt.Errorf("mfl: retrieve key: %w", err)
	}
	if secret == "" {
		return out, errors.New("mfl: my leagues requires a key")
	}
	return c.executeKeyed(ctx, cleanURL, secret)
}
