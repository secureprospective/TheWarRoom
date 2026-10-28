// Package migrationtest holds MFL leagues for proving the move from MFL: the Legacy NFL as held, and
// synthetic leagues covering every edge a league can have. It is used only by tests.
package migrationtest

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/ingestion"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/draftreports"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/league"
	"github.com/secureprospective/TheWarRoom/internal/leaguemigration"
	"github.com/secureprospective/TheWarRoom/internal/playerid"
)

// ReadAt is the instant every fixture league was read.
var ReadAt = time.Date(2026, 10, 28, 9, 0, 0, 0, time.UTC) //nolint:gochecknoglobals // a fixed instant

func repo() string {
	_, file, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(file), "..", "..", "..")
}

func read(path string) ([]byte, error) {
	body, err := os.ReadFile(filepath.Join(repo(), path))
	if err != nil {
		return nil, fmt.Errorf("read fixture: %w", err)
	}
	return body, nil
}

// Legacy is the Legacy NFL (MFL league 14432) as the app holds it: its 2026 league, rules, rosters and
// both draft reports.
func Legacy() (leaguemigration.Export, error) {
	leagueBody, err := read("internal/ingestion/league/testdata/league-2026.json")
	if err != nil {
		return leaguemigration.Export{}, err
	}
	rulesBody, err := read("internal/ingestion/league/testdata/rules-2026.json")
	if err != nil {
		return leaguemigration.Export{}, err
	}
	config, err := league.Parse(leagueBody, rulesBody)
	if err != nil {
		return leaguemigration.Export{}, fmt.Errorf("parse league: %w", err)
	}
	config.CurrentSeason = 2026
	out := leaguemigration.Export{LeagueID: "14432", Name: "Legacy NFL", Season: 2026, Config: config,
		ReadAt: ReadAt, Players: map[string]string{}}
	if out.Rosters, err = legacyRosters(); err != nil {
		return out, err
	}
	for _, r := range out.Rosters {
		out.Players[r.Player] = "Player " + r.Player
	}
	results, err := read("internal/ingestion/draftreports/testdata/results-2026.json")
	if err != nil {
		return out, err
	}
	if out.Draft, err = draftreports.ParseResults(results); err != nil {
		return out, fmt.Errorf("parse draft: %w", err)
	}
	for _, p := range out.Draft {
		if p.Player != nil {
			out.Players[p.Player.String()] = "Player " + p.Player.String()
		}
	}
	future, err := read("internal/ingestion/draftreports/testdata/future-2026.json")
	if err != nil {
		return out, err
	}
	if out.Future, err = draftreports.ParseFuture(future); err != nil {
		return out, fmt.Errorf("parse future picks: %w", err)
	}
	for _, f := range config.Franchises {
		out.Owners = append(out.Owners, leaguemigration.Owner{Franchise: f.ID, Name: "Owner " + f.ID,
			Email: "owner" + f.ID + "@example.com"})
	}
	return out, nil
}

func legacyRosters() ([]leaguemigration.Rostered, error) {
	body, err := read("internal/ingestion/testdata/comms/rosters-2026.json")
	if err != nil {
		return nil, err
	}
	var env struct {
		Rosters struct {
			Franchise ingestion.MFLList[struct {
				ID     string `json:"id"`
				Player ingestion.MFLList[struct {
					ID, Salary, ContractYear, ContractStatus, ContractInfo, Status string
				}] `json:"player"`
			}] `json:"franchise"`
		} `json:"rosters"`
	}
	if err := json.Unmarshal(body, &env); err != nil {
		return nil, fmt.Errorf("parse rosters: %w", err)
	}
	var out []leaguemigration.Rostered
	for _, f := range env.Rosters.Franchise {
		for _, p := range f.Player {
			out = append(out, leaguemigration.Rostered{Franchise: f.ID, Player: p.ID, Status: p.Status,
				Salary: p.Salary, ContractYear: p.ContractYear, ContractStatus: p.ContractStatus,
				ContractInfo: p.ContractInfo})
		}
	}
	return out, nil
}

// Source serves one fixture league as MFL would. FailAt names the part that fails, as MFL going away
// mid-read would; it never writes anything.
type Source struct {
	Export leaguemigration.Export
	FailAt string
	Reads  int
}

// ErrUnreachable is MFL not answering.
var ErrUnreachable = errors.New("MFL did not answer")

func (s *Source) fail(part string) error {
	s.Reads++
	if s.FailAt == part {
		return ErrUnreachable
	}
	return nil
}

func (s *Source) League(context.Context) (leaguemigration.LeagueRead, error) {
	if err := s.fail("league"); err != nil {
		return leaguemigration.LeagueRead{}, err
	}
	e := s.Export
	return leaguemigration.LeagueRead{ID: e.LeagueID, Name: e.Name, Config: e.Config, Owners: e.Owners,
		Champions: e.Champions}, nil
}

func (s *Source) Rosters(context.Context) ([]leaguemigration.Rostered, error) {
	return s.Export.Rosters, s.fail("rosters")
}

func (s *Source) Adjustments(context.Context) ([]leaguemigration.Adjustment, error) {
	return s.Export.Adjustments, s.fail("adjustments")
}

func (s *Source) Draft(context.Context) ([]draftreports.Pick, []draftreports.FuturePick, error) {
	return s.Export.Draft, s.Export.Future, s.fail("draft")
}

func (s *Source) Players(context.Context) (map[string]string, error) {
	return s.Export.Players, s.fail("players")
}

// pid is a typed player id for a draft pick.
func pid(raw string) *playerid.PlayerID {
	id, err := playerid.New(raw)
	if err != nil {
		panic(err)
	}
	return &id
}
