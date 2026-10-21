// Package race builds League Pulse race reports from detached published inputs.
package race

import (
	"fmt"

	"github.com/secureprospective/TheWarRoom/internal/app/catalog"
	"github.com/secureprospective/TheWarRoom/internal/app/report"
	"github.com/secureprospective/TheWarRoom/internal/app/season"
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
}

func (a *API) TargetRace(week int, team, mine string) (RaceBoard, error) {
	if err := a.host.Ready(); err != nil {
		return RaceBoard{}, fmt.Errorf("race: startup: %w", err)
	}
	in := a.host.RaceInput(a.held.Snapshot)
	in.Week, in.Team, in.Mine = week, team, mine
	b, err := Build(in)
	return wireRace(b), err
}

func (a *API) TargetRaceRefresh(force bool) error {
	return a.host.RefreshRace(force, a.held.Snapshot) //nolint:wrapcheck // Preserve the existing refresh error.
}

func wireRace(b Board) RaceBoard {
	return RaceBoard(b)
}

func Commands() []catalog.Command {
	return []catalog.Command{
		{ID: catalog.Unset, Method: "TargetRace", Version: catalog.Unset, Role: "gm,commish,admin", Gravity: "G0", Availability: "both"},
		{ID: catalog.Unset, Method: "TargetRaceRefresh", Version: catalog.Unset, Role: "gm,commish,admin", Gravity: "G0", Availability: "both"},
	}
}

// RaceBoard is League Pulse › Race: the standings on top, then the reports. Week and Team are the
// choices the lineup and one-team reports follow, as Go accepted them.
type Board struct {
	Standings report.Report   `json:"standings"`
	Board     report.Board    `json:"board"`
	Week      int             `json:"week"`
	Weeks     []report.Choice `json:"weeks"`
	Team      string          `json:"team"`
	Teams     []report.Choice `json:"teams"`
}

func Build(in season.Snapshot) (Board, error) {
	v, err := season.NewView(in)
	if err != nil {
		return Board{}, err //nolint:wrapcheck // preserve the shared view validation text
	}
	board := Board{Standings: v.Standings(), Week: v.Week, Team: v.Team,
		Weeks: v.WeekChoices(), Teams: v.TeamChoices(),
		Board: report.Board{Title: "League race reports", Loading: "Reading the league race…",
			Refresh: "Refresh from MFL", HowLabel: "How this is counted", Reports: []report.Report{}}}
	board.Board.Reports = append(board.Board.Reports, v.Power(season.ViewSeason), v.Power(season.ViewFranchise), v.Seeding(),
		v.Bubble(), v.Tiebreakers(), v.Brackets(), v.AllPlay(), v.WeeklyPoints(), v.PositionPoints(),
		v.PlayerPoints(), v.ScheduleByWeek(), v.ScheduleByTeam(), v.ScheduleGrid(), v.Lineups(), v.LineupDetail())
	return board, nil
}
