// Package mflsource reads one MFL league for the move into the app. It only calls MFL's read exports;
// it never submits anything to MFL.
package mflsource

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/ingestion"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/draftreports"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/league"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/rosters"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/salaryadjustments"
	"github.com/secureprospective/TheWarRoom/internal/leaguemigration"
	"github.com/secureprospective/TheWarRoom/internal/mfl"
)

// Source reads one MFL league. Players come from the player list the app already holds: MFL's players
// export allows one call a day, and the list is the same for every league.
type Source struct {
	Client *mfl.Client
	ID     string
	Now    func() time.Time
	Held   func() map[string]string
	season int
}

// owners is the part of MFL's league export the settings reader does not keep: the league's name and
// each franchise's owner (MFL sends owner names and emails to the commissioner's key only).
type owners struct {
	League struct {
		Name       string `json:"name"`
		Franchises struct {
			Franchise ingestion.MFLList[struct {
				ID        string `json:"id"`
				OwnerName string `json:"owner_name"`
				Email     string `json:"email"`
			}] `json:"franchise"`
		} `json:"franchises"`
	} `json:"league"`
}

func (s *Source) year() string { return strconv.Itoa(s.season) }

// League reads the settings, scoring, franchises and divisions, then the name and owners.
func (s *Source) League(ctx context.Context) (leaguemigration.LeagueRead, error) {
	config, err := league.Discover(ctx, s.Client, s.ID, s.Now().Year())
	if err != nil {
		return leaguemigration.LeagueRead{}, fmt.Errorf("read league: %w", err)
	}
	s.season = config.CurrentSeason
	env, err := ingestion.FetchLeagueExport[owners](ctx, s.Client, "league", s.year(), s.ID, nil)
	if err != nil {
		return leaguemigration.LeagueRead{}, fmt.Errorf("read league owners: %w", err)
	}
	out := leaguemigration.LeagueRead{ID: s.ID, Name: env.League.Name, Config: config}
	for _, f := range env.League.Franchises.Franchise {
		out.Owners = append(out.Owners, leaguemigration.Owner{Franchise: f.ID, Name: f.OwnerName, Email: f.Email})
	}
	return out, nil
}

func (s *Source) Rosters(ctx context.Context) ([]leaguemigration.Rostered, error) {
	raw, err := rosters.Fetch(ctx, s.Client, s.year(), s.ID)
	if err != nil {
		return nil, fmt.Errorf("read rosters: %w", err)
	}
	out := make([]leaguemigration.Rostered, 0, len(raw))
	for _, r := range raw {
		out = append(out, leaguemigration.Rostered{Franchise: r.FranchiseID, Player: r.PlayerID, Status: r.Status,
			Salary: r.Salary, ContractYear: r.ContractYear, ContractStatus: r.ContractStatus, ContractInfo: r.ContractInfo})
	}
	return out, nil
}

func (s *Source) Adjustments(ctx context.Context) ([]leaguemigration.Adjustment, error) {
	raw, err := salaryadjustments.Fetch(ctx, s.Client, s.year(), s.ID)
	if err != nil {
		return nil, fmt.Errorf("read salary adjustments: %w", err)
	}
	out := make([]leaguemigration.Adjustment, 0, len(raw))
	for _, a := range raw {
		out = append(out, leaguemigration.Adjustment{ID: a.ID, Franchise: a.FranchiseID, Amount: a.Amount,
			Description: a.Description})
	}
	return out, nil
}

func (s *Source) Draft(ctx context.Context) ([]draftreports.Pick, []draftreports.FuturePick, error) {
	results, err := draftreports.FetchResults(ctx, s.Client, s.year(), s.ID)
	if err != nil {
		return nil, nil, fmt.Errorf("read draft results: %w", err)
	}
	future, err := draftreports.FetchFuture(ctx, s.Client, s.year(), s.ID)
	if err != nil {
		return nil, nil, fmt.Errorf("read future picks: %w", err)
	}
	return results, future, nil
}

func (s *Source) Players(context.Context) (map[string]string, error) {
	players := s.Held()
	if len(players) == 0 {
		return nil, fmt.Errorf("the app holds no player list yet; refresh it, then import again")
	}
	return players, nil
}
