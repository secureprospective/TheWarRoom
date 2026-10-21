package main

import (
	"time"

	raceapp "github.com/secureprospective/TheWarRoom/internal/app/race"
	"github.com/secureprospective/TheWarRoom/internal/app/season"

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

func (in raceInput) snapshot() season.Snapshot {
	power := make(map[string]season.PowerRankingsResult, len(in.held.power))
	for key, p := range in.held.power {
		rows := make([]season.PowerRow, len(p.Rows))
		for i, r := range p.Rows {
			rows[i] = season.PowerRow(r)
		}
		power[key] = season.PowerRankingsResult{OK: p.OK, Error: p.Error, View: p.View, Weight: p.Weight, AgeWeight: p.AgeWeight, ModelRunID: p.ModelRunID, AggMode: p.AggMode, StarterN: p.StarterN, Performance: p.Performance, WeeksScored: p.WeeksScored, Rows: rows}
	}
	liveWeeks := map[int]leaguefeed.WeekResult{}
	for _, w := range in.held.weeks.Value {
		if w.Week == in.live.Value.Week {
			liveWeeks[w.Week] = liveWeek(in.live.Value, w)
		}
	}
	return season.Snapshot{LiveWeeks: liveWeeks, Held: season.Held{Weeks: raceFeed(in.held.weeks), Brackets: raceFeed(in.held.brackets), Standings: raceFeed(in.held.standings), Power: power}, Live: raceFeed(in.live), LiveUnplayed: in.liveUnplayed, Config: in.config, Lookup: in.lookup, HasLookup: in.hasLookup, Week: in.week, Team: in.team, Mine: in.mine, Zone: in.zone}
}
func raceFeed[T any](h heldSeasonFeed[T]) season.Feed[T] {
	return season.Feed[T]{Value: h.Value, FetchedAt: h.FetchedAt, RefreshError: h.RefreshError, ReadPending: h.ReadPending}
}
func buildRace(in raceInput) (RaceBoard, error) {
	b, err := raceapp.Build(in.snapshot())
	return wireRace(b), err
}
func wireRace(b raceapp.Board) RaceBoard {
	return RaceBoard{Standings: wireReport(b.Standings), Board: wireBoard(b.Board), Week: b.Week, Weeks: wireChoices(b.Weeks), Team: b.Team, Teams: wireChoices(b.Teams)}
}

// raceView keeps Now's existing season-view seam while construction belongs to season.
type raceView struct {
	names   map[string]string
	service season.View
	in      raceInput
	table   race.Table
	weeks   []leaguefeed.WeekResult
	week    int
	team    string
}

func newRaceView(in raceInput) (raceView, error) {
	v, err := season.NewView(in.snapshot())
	return raceView{names: v.Names, service: v, in: in, table: v.Table, weeks: v.Weeks, week: v.Week, team: v.Team}, err //nolint:wrapcheck // preserve race validation text
}
func (v raceView) view() season.View          { s := v.service; s.Week = v.week; s.Team = v.team; return s }
func weekScored(w leaguefeed.WeekResult) bool { return season.WeekScored(w) }
func findGame(w leaguefeed.WeekResult, team string) (leaguefeed.Game, bool) {
	return season.FindGame(w, team)
}
func (v raceView) name(id string) string  { return v.view().Name(id) }
func (v raceView) scheduleByTeam() Report { return wireReport(v.view().ScheduleByTeam()) }
func (v raceView) bubble() Report         { return wireReport(v.view().Bubble()) }
