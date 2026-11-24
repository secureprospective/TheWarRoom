package mflread

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/freeagents"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/league"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/nflschedule"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/players"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/rosters"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/salaryadjustments"
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
	return f.clientWith(f.Transport)
}

func (f Fetcher) clientWith(transport http.RoundTripper) (*mfl.Client, error) {
	rps := f.RPS
	if rps == 0 {
		rps = defaultRPS
	}
	c, err := mfl.New("api", rps, mfl.WithTransport(ReadOnly(transport)))
	if err != nil {
		return nil, fmt.Errorf("open the MFL reader: %w", err)
	}
	return c, nil
}

// onceLeagueTransport caches the one GET TYPE=league 200 reply for a single refresh so host
// discovery, franchises and league rules all read from one MFL call.
type onceLeagueTransport struct {
	base   http.RoundTripper
	mu     sync.Mutex
	url    string
	body   []byte
	header http.Header
}

func (t *onceLeagueTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	base := t.base
	if base == nil {
		base = http.DefaultTransport
	}
	if req.URL == nil || req.URL.Query().Get("TYPE") != "league" {
		return base.RoundTrip(req) //nolint:wrapcheck // transport errors pass through unchanged.
	}
	key := req.URL.String()
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.url == key && t.body != nil {
		return &http.Response{StatusCode: http.StatusOK, Status: "200 OK", Header: t.header.Clone(),
			Body: io.NopCloser(bytes.NewReader(t.body)), Request: req}, nil
	}
	resp, err := base.RoundTrip(req)
	if err != nil || resp.StatusCode != http.StatusOK {
		return resp, err //nolint:wrapcheck // transport errors pass through unchanged.
	}
	body, rerr := io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	if rerr != nil {
		return nil, fmt.Errorf("read MFL league export: %w", rerr)
	}
	t.url, t.body, t.header = key, body, resp.Header.Clone()
	resp.Body = io.NopCloser(bytes.NewReader(body))
	return resp, nil
}

// FetchRefresh reads free agents, all league data and (when needPlayers is true) the player
// directory through one MFL client so TYPE=league is fetched once per refresh.
func (f Fetcher) FetchRefresh(ctx context.Context, mflLeague string, season int,
	needPlayers bool) ([]string, mflmirror.LeagueData, []byte, error) {
	c, err := f.clientWith(&onceLeagueTransport{base: f.Transport})
	if err != nil {
		return nil, mflmirror.LeagueData{}, nil, err
	}
	year := strconv.Itoa(season)
	data, err := fetchLeagueData(ctx, c, year, mflLeague)
	if err != nil {
		return nil, mflmirror.LeagueData{}, nil, err
	}
	raw, err := freeagents.Fetch(ctx, c, year, mflLeague)
	if err != nil {
		return nil, mflmirror.LeagueData{}, nil, fmt.Errorf("fetch MFL free agents: %w", err)
	}
	ids := make([]string, 0, len(raw))
	for _, r := range raw {
		ids = append(ids, r.ID)
	}
	var playersBody []byte
	if needPlayers {
		rawPlayers, perr := players.Fetch(ctx, c, year, mflLeague)
		if perr != nil {
			return nil, mflmirror.LeagueData{}, nil, fmt.Errorf("fetch MFL player directory: %w", perr)
		}
		if playersBody, err = json.Marshal(rawPlayers); err != nil {
			return nil, mflmirror.LeagueData{}, nil, fmt.Errorf("encode MFL player directory: %w", err)
		}
	}
	return ids, data, playersBody, nil
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

// FetchLeagueData reads the league's public franchises, rosters, salary adjustments, league rules
// and the NFL bye weeks through the same GET-only client.
func (f Fetcher) FetchLeagueData(ctx context.Context, mflLeague string, season int) (mflmirror.LeagueData, error) {
	c, err := f.clientWith(&onceLeagueTransport{base: f.Transport})
	if err != nil {
		return mflmirror.LeagueData{}, err
	}
	return fetchLeagueData(ctx, c, strconv.Itoa(season), mflLeague)
}

func fetchLeagueData(ctx context.Context, c *mfl.Client, year, mflLeague string) (mflmirror.LeagueData, error) {
	out := mflmirror.LeagueData{Byes: map[string]int{}}
	config, err := league.Fetch(ctx, c, year, mflLeague)
	if err != nil {
		return mflmirror.LeagueData{}, fmt.Errorf("fetch MFL league rules: %w", err)
	}
	out.Franchises = make([]mflmirror.Franchise, 0, len(config.Franchises))
	for _, fr := range config.Franchises {
		out.Franchises = append(out.Franchises, mflmirror.Franchise{ID: fr.ID, Name: fr.Name})
	}
	if out.Config, err = json.Marshal(config); err != nil {
		return mflmirror.LeagueData{}, fmt.Errorf("hold MFL league rules: %w", err)
	}
	raw, err := rosters.Fetch(ctx, c, year, mflLeague)
	if err != nil {
		return mflmirror.LeagueData{}, fmt.Errorf("fetch MFL rosters: %w", err)
	}
	for _, r := range raw {
		salary, perr := domain.ParseMoneyMillions(r.Salary)
		if perr != nil {
			return mflmirror.LeagueData{}, fmt.Errorf("MFL roster salary %q: %w", r.Salary, perr)
		}
		contractYear, _ := strconv.Atoi(r.ContractYear)
		out.Rosters = append(out.Rosters, mflmirror.RosterRow{Franchise: r.FranchiseID, Player: r.PlayerID,
			Salary: salary.Cents(), ContractYear: contractYear, Status: r.Status})
	}
	adjustments, err := salaryadjustments.Fetch(ctx, c, year, mflLeague)
	if err != nil {
		return mflmirror.LeagueData{}, fmt.Errorf("fetch MFL salary adjustments: %w", err)
	}
	for _, a := range adjustments {
		amount, perr := domain.ParseSignedMoneyMillions(a.Amount)
		if perr != nil {
			return mflmirror.LeagueData{}, fmt.Errorf("MFL salary adjustment %q: %w", a.Amount, perr)
		}
		out.Adjustments = append(out.Adjustments, mflmirror.Adjustment{Franchise: a.FranchiseID, Cents: amount.Cents()})
	}
	byes, err := nflschedule.FetchByes(ctx, c, year)
	if err != nil {
		return mflmirror.LeagueData{}, fmt.Errorf("fetch NFL bye weeks: %w", err)
	}
	for _, b := range byes {
		if week, perr := strconv.Atoi(b.Week); perr == nil {
			out.Byes[b.Team] = week
		}
	}
	return out, nil
}
