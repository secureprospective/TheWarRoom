package mflread

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/ingestion"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/freeagents"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/players"
	"github.com/secureprospective/TheWarRoom/internal/mfl"
	"github.com/secureprospective/TheWarRoom/internal/mflmirror"
	"github.com/secureprospective/TheWarRoom/internal/normalize"
)

const defaultRPS = 2

// Fetcher reads public exports through a GET-only client with no key source. It implements
// mflmirror.Source.
type Fetcher struct {
	Transport http.RoundTripper // nil uses http.DefaultTransport
	Now       func() time.Time
	RPS       float64
}

func (f Fetcher) client() (*mfl.Client, error) {
	rps := f.RPS
	if rps == 0 {
		rps = defaultRPS
	}
	c, err := mfl.New("api", rps, mfl.WithTransport(ReadOnly(f.Transport)))
	if err != nil {
		return nil, fmt.Errorf("open the MFL reader: %w", err)
	}
	return c, nil
}

// FetchFreeAgents reads the league's free-agent list.
func (f Fetcher) FetchFreeAgents(ctx context.Context, mflLeague string, season int) ([]string, error) {
	c, err := f.client()
	if err != nil {
		return nil, err
	}
	raw, err := freeagents.Fetch(ctx, c, strconv.Itoa(season), mflLeague)
	if err != nil {
		return nil, fmt.Errorf("fetch MFL free agents: %w", err)
	}
	ids := make([]string, 0, len(raw))
	for _, r := range raw {
		ids = append(ids, r.ID)
	}
	return ids, nil
}

// FetchFranchises reads the league's public franchise list from the `league` export.
func (f Fetcher) FetchFranchises(ctx context.Context, mflLeague string, season int) ([]mflmirror.Franchise, error) {
	c, err := f.client()
	if err != nil {
		return nil, err
	}
	env, err := ingestion.FetchLeagueExport[struct {
		League struct {
			Franchises struct {
				Franchise ingestion.MFLList[struct {
					ID   string `json:"id"`
					Name string `json:"name"`
				}] `json:"franchise"`
			} `json:"franchises"`
		} `json:"league"`
	}](ctx, c, "league", strconv.Itoa(season), mflLeague, nil)
	if err != nil {
		return nil, fmt.Errorf("fetch MFL franchises: %w", err)
	}
	out := []mflmirror.Franchise{}
	for _, row := range env.League.Franchises.Franchise {
		if row.ID != "" {
			out = append(out, mflmirror.Franchise{ID: row.ID, Name: row.Name})
		}
	}
	return out, nil
}

// FetchPlayers reads the player directory. MFL allows this export about once a day, so the
// store holds it and the caller asks only when the held copy is old.
func (f Fetcher) FetchPlayers(ctx context.Context, mflLeague string, season int) ([]byte, error) {
	c, err := f.client()
	if err != nil {
		return nil, err
	}
	raw, err := players.Fetch(ctx, c, strconv.Itoa(season), mflLeague)
	if err != nil {
		return nil, fmt.Errorf("fetch MFL player directory: %w", err)
	}
	body, err := json.Marshal(raw)
	if err != nil {
		return nil, fmt.Errorf("encode MFL player directory: %w", err)
	}
	return body, nil
}

// Directory reads a held player directory into the app's player facts.
func (Fetcher) Directory(playersJSON []byte) (mflmirror.Directory, error) {
	lookup, err := normalize.NewLookupJSON(playersJSON)
	if err != nil {
		return nil, fmt.Errorf("read held MFL player directory: %w", err)
	}
	return lookup, nil
}
