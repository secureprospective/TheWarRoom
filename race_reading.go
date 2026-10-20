package main

import (
	"time"

	raceapp "github.com/secureprospective/TheWarRoom/internal/app/race"
	"github.com/secureprospective/TheWarRoom/internal/app/season"
	"github.com/secureprospective/TheWarRoom/internal/playerid"

	"github.com/secureprospective/TheWarRoom/internal/app/report"
	"github.com/secureprospective/TheWarRoom/internal/engine/race"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/league"
	"github.com/secureprospective/TheWarRoom/internal/leaguefeed"
	"github.com/secureprospective/TheWarRoom/internal/normalize"
)

// RaceBoard is League Pulse › Race: the standings on top, then the reports. Week and Team are the
// choices the lineup and one-team reports follow, as Go accepted them.
type RaceBoard struct {
	Standings Report       `json:"standings"`
	Board     ReportBoard  `json:"board"`
	Week      int          `json:"week"`
	Weeks     []RaceChoice `json:"weeks"`
	Team      string       `json:"team"`
	Teams     []RaceChoice `json:"teams"`
}

type RaceChoice struct {
	Value string `json:"value"`
	Label string `json:"label"`
}

type raceInput struct {
	live heldSeasonFeed[leaguefeed.Lineups]
	// liveUnplayed: no NFL game of the live week has kicked off, so its 0.00 scores are not a game in progress.
	liveUnplayed bool
	held         raceHeld
	config       league.RawConfig
	lookup       normalize.Lookup
	week         int
	team         string
	mine         string
	zone         *time.Location
	hasLookup    bool
}

func (in raceInput) snapshot() raceapp.Snapshot {
	power := make(map[string]season.PowerRankingsResult, len(in.held.power))
	for key, p := range in.held.power {
		rows := make([]season.PowerRow, len(p.Rows))
		for i, r := range p.Rows {
			rows[i] = season.PowerRow(r)
		}
		power[key] = season.PowerRankingsResult{OK: p.OK, Error: p.Error, View: p.View, Weight: p.Weight, AgeWeight: p.AgeWeight, ModelRunID: p.ModelRunID, AggMode: p.AggMode, StarterN: p.StarterN, Performance: p.Performance, WeeksScored: p.WeeksScored, Rows: rows}
	}
	liveWeeks := map[int]leaguefeed.WeekResult{}
	for _, w := range in.held.weeks.value {
		if w.Week == in.live.value.Week {
			liveWeeks[w.Week] = liveWeek(in.live.value, w)
		}
	}
	return raceapp.Snapshot{LiveWeeks: liveWeeks, Held: raceapp.Held{Weeks: raceFeed(in.held.weeks), Brackets: raceFeed(in.held.brackets), Standings: raceFeed(in.held.standings), Power: power}, Live: raceFeed(in.live), LiveUnplayed: in.liveUnplayed, Config: in.config, Lookup: in.lookup, HasLookup: in.hasLookup, Week: in.week, Team: in.team, Mine: in.mine, Zone: in.zone}
}
func raceFeed[T any](h heldSeasonFeed[T]) raceapp.Feed[T] {
	return raceapp.Feed[T]{Value: h.value, FetchedAt: h.fetchedAt, RefreshError: h.refreshError, ReadPending: h.readPending}
}
func buildRace(in raceInput) (RaceBoard, error) {
	b, err := raceapp.Build(in.snapshot())
	return wireRace(b), err
}
func wireRace(b raceapp.Board) RaceBoard {
	return RaceBoard{Standings: wireReport(b.Standings), Board: wireBoard(b.Board), Week: b.Week, Weeks: wireChoices(b.Weeks), Team: b.Team, Teams: wireChoices(b.Teams)}
}

// raceView keeps Now's existing season-view seam while construction belongs to Race.
type raceView struct {
	names   map[string]string
	service raceapp.View
	in      raceInput
	table   race.Table
	weeks   []leaguefeed.WeekResult
	week    int
	team    string
}

func newRaceView(in raceInput) (raceView, error) {
	v, err := raceapp.NewView(in.snapshot())
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
func (v raceView) row(id string, cells ...string) ReportRow {
	return wireRows([]report.Row{v.view().Row(id, cells...)})[0]
}
func (v raceView) weeksReport(id, title, how string, columns ...string) Report {
	return wireReport(v.view().WeeksReport(id, title, how, columns...))
}
func (v raceView) gameResult(g leaguefeed.Game) string                { return v.view().GameResult(g) }
func (v raceView) player(id playerid.PlayerID) (string, string)       { return v.view().Player(id) }
func (v raceView) positions(seen map[string]bool) []string            { return v.view().Positions(seen) }
func (v raceView) starters(s leaguefeed.Side) []leaguefeed.WeekPlayer { return v.view().Starters(s) }
func (v raceView) directoryNote(r *Report) {
	s := report.Report{Note: r.Note}
	v.view().DirectoryNote(&s)
	r.Note = s.Note
}
func (v raceView) scheduleByTeam() Report { return wireReport(v.view().ScheduleByTeam()) }
func (v raceView) bubble() Report         { return wireReport(v.view().Bubble()) }
