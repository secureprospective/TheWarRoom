package now

import (
	raceapp "github.com/secureprospective/TheWarRoom/internal/app/race"
	"github.com/secureprospective/TheWarRoom/internal/app/report"
	"github.com/secureprospective/TheWarRoom/internal/engine/race"
	"github.com/secureprospective/TheWarRoom/internal/leaguefeed"
	"github.com/secureprospective/TheWarRoom/internal/playerid"
)

// raceView keeps Now's existing season-view seam while construction belongs to Race.
type raceView struct {
	names   map[string]string
	service raceapp.View
	in      raceapp.Snapshot
	table   race.Table
	weeks   []leaguefeed.WeekResult
	week    int
	team    string
}

func newRaceView(in raceapp.Snapshot) (raceView, error) {
	v, err := raceapp.NewView(in)
	return raceView{names: v.Names, service: v, in: in, table: v.Table, weeks: v.Weeks, week: v.Week, team: v.Team}, err //nolint:wrapcheck // preserve race validation text
}
func (v raceView) view() raceapp.View          { s := v.service; s.Week = v.week; s.Team = v.team; return s }
func weekScored(w leaguefeed.WeekResult) bool  { return raceapp.WeekScored(w) }
func weekLabel(w leaguefeed.WeekResult) string { return raceapp.WeekLabel(w) }
func findGame(w leaguefeed.WeekResult, team string) (leaguefeed.Game, bool) {
	return raceapp.FindGame(w, team)
}
func scoreText(p leaguefeed.Points) string { return report.ScoreText(p) }
func (v raceView) name(id string) string   { return v.view().Name(id) }
func (v raceView) row(id string, cells ...string) report.Row {
	return v.view().Row(id, cells...)
}
func (v raceView) weeksReport(id, title, how string, columns ...string) report.Report {
	return v.view().WeeksReport(id, title, how, columns...)
}
func (v raceView) gameResult(g leaguefeed.Game) string                { return v.view().GameResult(g) }
func (v raceView) player(id playerid.PlayerID) (string, string)       { return v.view().Player(id) }
func (v raceView) positions(seen map[string]bool) []string            { return v.view().Positions(seen) }
func (v raceView) starters(s leaguefeed.Side) []leaguefeed.WeekPlayer { return v.view().Starters(s) }
func (v raceView) directoryNote(r *report.Report) {
	s := report.Report{Note: r.Note}
	v.view().DirectoryNote(&s)
	r.Note = s.Note
}
