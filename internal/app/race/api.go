// Package race builds League Pulse race reports from detached published inputs.
package race

import (
	"fmt"

	"github.com/secureprospective/TheWarRoom/internal/app/report"
	"github.com/secureprospective/TheWarRoom/internal/app/season"
	catalog "github.com/secureprospective/TheWarRoom/internal/commandcatalog"
	"github.com/secureprospective/TheWarRoom/internal/m2service"
)

// Host captures the private and shared season data together and owns refresh serialization.
type Host interface {
	Ready() error
	RaceInput(func() season.Held) season.Snapshot
	RefreshRace(bool, func() season.Held) error
}

type API struct {
	host Host
	held *Held
}

func New(host Host, held *Held) *API { return &API{host: host, held: held} }

//nolint:revive // Preserve the existing exported wire name.
type RaceBoard struct {
	Standings report.Report   `json:"standings"`
	Board     report.Board    `json:"board"`
	Week      int             `json:"week"`
	Weeks     []report.Choice `json:"weeks"`
	Team      string          `json:"team"`
	Teams     []report.Choice `json:"teams"`
	StarterN  int             `json:"starterN"`
}

func (a *API) TargetRace(week int, team, mine, aggMode string) (RaceBoard, error) {
	if err := a.host.Ready(); err != nil {
		return RaceBoard{}, fmt.Errorf("race: startup: %w", err)
	}
	in := a.host.RaceInput(a.held.Snapshot)
	in.Week, in.Team, in.Mine = week, team, mine
	for _, view := range []string{season.ViewSeason, season.ViewFranchise} {
		mode := m2service.ResolveAggMode(aggMode, season.DefaultAgg(view))
		if mode != season.DefaultAgg(view) {
			key := season.PowerKey(view, mode)
			if result, ok := in.Held.Power[key]; ok {
				in.Held.Power[view] = result
			} else {
				delete(in.Held.Power, view)
			}
		}
	}
	b, err := Build(in)
	return wireRace(b), err
}

func (a *API) TargetRaceRefresh(force bool) error {
	return a.host.RefreshRace(force, a.held.Snapshot) //nolint:wrapcheck // Preserve the existing refresh error.
}

func wireRace(b Board) RaceBoard {
	return RaceBoard(b)
}

func Commands() []catalog.Command { return catalog.DesktopCommands("race", nil) }

// RaceBoard is League Pulse › Race: the standings on top, then the reports. Week and Team are the
// choices the lineup and one-team reports follow, as Go accepted them.
type Board struct {
	Standings report.Report   `json:"standings"`
	Board     report.Board    `json:"board"`
	Week      int             `json:"week"`
	Weeks     []report.Choice `json:"weeks"`
	Team      string          `json:"team"`
	Teams     []report.Choice `json:"teams"`
	StarterN  int             `json:"starterN"`
}

func Build(in season.Snapshot) (Board, error) {
	v, err := season.NewView(in)
	if err != nil {
		return Board{}, err //nolint:wrapcheck // preserve the shared view validation text
	}
	starterN := in.Held.Power[season.PowerKey(season.ViewSeason, m2service.AggTopN)].StarterN
	board := Board{Standings: v.Standings(), Week: v.Week, Team: v.Team, StarterN: starterN,
		Weeks: v.WeekChoices(), Teams: v.TeamChoices(),
		Board: report.Board{Title: "League race reports", Loading: "Reading the league race…",
			Refresh: "Refresh from MFL", HowLabel: "How this is counted", Reports: []report.Report{}}}
	board.Board.Reports = append(board.Board.Reports, v.Power(season.ViewSeason), v.Power(season.ViewFranchise), v.Seeding(),
		v.Bubble(), v.Tiebreakers(), v.Brackets(), v.AllPlay(), v.WeeklyPoints(), v.PositionPoints(),
		v.PlayerPoints(), v.ScheduleByWeek(), v.ScheduleByTeam(), v.ScheduleGrid(), v.Lineups(), v.LineupDetail())
	return board, nil
}
