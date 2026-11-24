package mflread

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/ingestion"
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

// FetchLeagueData reads the league's public rosters, salary adjustments, league rules and the NFL
// bye weeks through the same GET-only client.
func (f Fetcher) FetchLeagueData(ctx context.Context, mflLeague string, season int) (mflmirror.LeagueData, error) {
	c, err := f.client()
	if err != nil {
		return mflmirror.LeagueData{}, err
	}
	year := strconv.Itoa(season)
	out := mflmirror.LeagueData{Byes: map[string]int{}}
	config, err := league.Fetch(ctx, c, year, mflLeague)
	if err != nil {
		return mflmirror.LeagueData{}, fmt.Errorf("fetch MFL league rules: %w", err)
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
