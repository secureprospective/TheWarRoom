package main

import (
	"maps"
	"time"

	nowapp "github.com/secureprospective/TheWarRoom/internal/app/now"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/marketfeeds"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/nflschedule"
	"github.com/secureprospective/TheWarRoom/internal/leaguefeed"
)

type nowInput struct {
	race        raceInput
	held        nowHeld
	live        heldSeasonFeed[leaguefeed.Lineups]
	projections heldSeasonFeed[[]marketfeeds.Feed]
	at          time.Time
	reportWeek  int
	current     int
	week        int
}

func (a *App) nowReadInput(week, reportWeek int, mine string) nowapp.Snapshot {
	current := a.marketWeek()
	projections := a.heldMarketForWeek(current)[marketfeeds.Projections]
	a.seasonMu.Lock()
	held, now, live := a.race, a.now, a.seasonLineups
	held.power = nil // Now does not read power reports.
	now.schedules = maps.Clone(now.schedules)
	now.firsts.weeks = maps.Clone(now.firsts.weeks)
	a.seasonMu.Unlock()
	a.lookupMu.Lock()
	lookup, hasLookup := a.lookup, a.hasLookup
	a.lookupMu.Unlock()
	return (nowInput{race: raceInput{held: held, config: a.rulebook.ActiveConfig(), lookup: lookup, hasLookup: hasLookup, mine: mine, zone: time.Local}, held: now, live: live, projections: projections, at: a.timeNow(), current: current, week: week, reportWeek: reportWeek}).snapshot()
}
func nowFeed[T any](f heldSeasonFeed[T]) nowapp.Feed[T] {
	return nowapp.Feed[T]{Value: f.value, FetchedAt: f.fetchedAt, RefreshError: f.refreshError, ReadPending: f.readPending}
}
func (in nowInput) snapshot() nowapp.Snapshot {
	schedules := make(map[int]nowapp.Feed[[]nflschedule.RawWeek], len(in.held.schedules))
	for k, v := range in.held.schedules {
		schedules[k] = nowFeed(v)
	}
	firsts := make(map[int]nowapp.FirstScore, len(in.held.firsts.weeks))
	for k, v := range in.held.firsts.weeks {
		firsts[k] = nowapp.FirstScore{At: v.at, Week: v.week}
	}
	return nowapp.Snapshot{Race: in.race.snapshot(), Held: nowapp.Held{Schedules: schedules, Byes: nowFeed(in.held.byes), Firsts: nowapp.FirstScores{Weeks: firsts, Copies: in.held.firsts.copies}}, Live: nowFeed(in.live), Projections: nowFeed(in.projections), At: in.at, ReportWeek: in.reportWeek, Current: in.current, Week: in.week}
}
func buildNow(in nowInput) (NowBoard, error) {
	b, e := nowapp.Build(in.snapshot())
	return wireNow(b), e
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
