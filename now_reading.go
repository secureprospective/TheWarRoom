package main

import (
	"time"

	"github.com/secureprospective/TheWarRoom/internal/app/season"

	"github.com/secureprospective/TheWarRoom/internal/app/report"

	nowapp "github.com/secureprospective/TheWarRoom/internal/app/now"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/marketfeeds"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/nflschedule"
	"github.com/secureprospective/TheWarRoom/internal/leaguefeed"
)

func (a *App) nowReadInput(week, reportWeek int, mine string) nowapp.Snapshot {
	current := a.marketWeek()
	projections := a.heldMarketForWeek(current)[marketfeeds.Projections]
	a.core.SeasonMu.Lock()
	schedules := make(map[int]nowapp.Feed[[]nflschedule.RawWeek], len(a.now.schedules))
	for k, v := range a.now.schedules {
		schedules[k] = nowFeed(v)
	}
	firsts := make(map[int]nowapp.FirstScore, len(a.now.firsts.weeks))
	for k, v := range a.now.firsts.weeks {
		firsts[k] = nowapp.FirstScore{At: v.at, Week: v.week}
	}
	raceHeld := a.raceHeld.Snapshot()
	in := nowapp.Snapshot{Race: season.Snapshot{Held: season.Held{Weeks: raceHeld.Weeks, Brackets: raceHeld.Brackets, Standings: raceHeld.Standings}}, Held: nowapp.Held{Schedules: schedules, Byes: nowFeed(a.now.byes), Firsts: nowapp.FirstScores{Weeks: firsts, Copies: a.now.firsts.copies}}, Live: nowFeed(a.core.SeasonLineups), Projections: nowFeed(projections), At: a.timeNow(), Current: current, Week: week, ReportWeek: reportWeek}
	a.core.SeasonMu.Unlock()
	a.core.LookupMu.Lock()
	in.Race.Lookup, in.Race.HasLookup = a.core.Lookup, a.core.HasLookup
	a.core.LookupMu.Unlock()
	in.Race.Config, in.Race.Mine, in.Race.Zone = a.core.Rulebook.ActiveConfig(), mine, time.Local
	return in
}
func nowFeed[T any](f heldSeasonFeed[T]) nowapp.Feed[T] {
	return nowapp.Feed[T]{Value: f.Value, FetchedAt: f.FetchedAt, RefreshError: f.RefreshError, ReadPending: f.ReadPending}
}
func wireNow(b nowapp.Board) NowBoard {
	return NowBoard{ReportWeek: b.ReportWeek, ReportWeeks: wireChoices(b.ReportWeeks), Week: b.Week, Weeks: wireChoices(b.Weeks), Matchup: wireReport(b.Matchup), Board: wireBoard(b.Board), Waiting: b.Waiting}
}
func liveWeek(l leaguefeed.Lineups, base leaguefeed.WeekResult) leaguefeed.WeekResult {
	return nowapp.LiveWeek(l, base)
}
func nflGames(w []nflschedule.RawWeek, at time.Time) map[string]nowapp.NFLGame {
	return nowapp.NFLGames(w, at)
}
func weekKickedOff(g map[string]nowapp.NFLGame, at time.Time) bool {
	return nowapp.WeekKickedOff(g, at)
}
func liveWeekUnplayed(l heldSeasonFeed[leaguefeed.Lineups], s map[int]heldSeasonFeed[[]nflschedule.RawWeek], at time.Time) bool {
	schedules := make(map[int]nowapp.Feed[[]nflschedule.RawWeek], len(s))
	for k, v := range s {
		schedules[k] = nowFeed(v)
	}
	return nowapp.LiveWeekUnplayed(nowFeed(l), schedules, at)
}
func possessive(s string) string { return nowapp.Possessive(s) }

// ReportEntity ties a literal display name to its held identity, including names in prose.
type ReportEntity struct {
	Text string `json:"text"`
	Kind string `json:"kind"`
	ID   string `json:"id"`
}

func attachReportEntities(r *Report, entities []ReportEntity) {
	s := report.Report{Columns: r.Columns, Title: r.Title, Note: r.Note, How: r.How}
	for _, row := range r.Rows {
		s.Rows = append(s.Rows, report.Row{Cells: row.Cells, Group: row.Group})
	}
	for _, row := range r.Summary {
		s.Summary = append(s.Summary, report.Row{Cells: row.Cells, Group: row.Group})
	}
	for _, e := range r.Entities {
		s.Entities = append(s.Entities, report.Entity(e))
	}
	es := make([]report.Entity, len(entities))
	for i, e := range entities {
		es[i] = report.Entity(e)
	}
	nowapp.AttachReportEntities(&s, es)
	if s.Entities == nil {
		r.Entities = nil
		return
	}
	r.Entities = make([]ReportEntity, len(s.Entities))
	for i, e := range s.Entities {
		r.Entities[i] = ReportEntity(e)
	}
}
