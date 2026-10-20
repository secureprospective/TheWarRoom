// Package race builds League Pulse race reports from detached published inputs.
package race

import (
	"github.com/secureprospective/TheWarRoom/internal/app/report"
	"github.com/secureprospective/TheWarRoom/internal/app/season"
)

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
