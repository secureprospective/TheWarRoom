package main

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
	"sync"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/app/core"
	"github.com/secureprospective/TheWarRoom/internal/app/season"
	"github.com/secureprospective/TheWarRoom/internal/ingestion"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/leagueschedule"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/leaguestandings"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/playoffs"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/weeklyresults"
	"github.com/secureprospective/TheWarRoom/internal/leaguefeed"
	"github.com/secureprospective/TheWarRoom/internal/m2service"
)

func (a *App) refreshRace(ctx context.Context) {
	pause := a.core.SeasonPause
	if pause == nil {
		pause = seasonPause
	}
	weeks, wf, werr := liveOrArchive(ctx, a.fallbackParent(), a.core.Season, archivedFeed[leaguefeed.WeekResult]{
		Export: weeklyresults.Export,
		Fetch: func(ctx context.Context) ([]leaguefeed.WeekResult, error) {
			return weeklyresults.Fetch(ctx, a.core.MFLClient, strconv.Itoa(a.core.Season), ingestion.LeagueID)
		},
		Parse:  weeklyresults.Parse,
		Bodies: a.scopedBodies("W", "YTD"),
	}, a.core.FallbackLog)
	a.core.SeasonMu.Lock()
	feed := heldSeasonFeed[[]leaguefeed.WeekResult](a.raceHeld.Snapshot().Weeks)
	holdSeasonFeed(&feed, weeks, wf, werr)
	a.raceHeld.PublishWeeks(raceFeed(feed))
	a.core.SeasonMu.Unlock()
	if err := pause(ctx, time.Second); err != nil {
		return
	}
	brackets, bf, berr := a.bracketsOrArchive(ctx, pause)
	a.core.SeasonMu.Lock()
	bracketFeed := heldSeasonFeed[[]leaguefeed.Bracket](a.raceHeld.Snapshot().Brackets)
	holdSeasonFeed(&bracketFeed, brackets, bf, berr)
	a.raceHeld.PublishBrackets(raceFeed(bracketFeed))
	a.core.SeasonMu.Unlock()
	if err := pause(ctx, time.Second); err != nil {
		return
	}
	standings, sf, serr := a.standingsOrArchive(ctx)
	power := map[string]PowerRankingsResult{}
	if serr == nil {
		schedule := sync.OnceValue(func() []leagueschedule.RawScheduleWeek { return a.remainingSchedule(ctx) })
		for _, view := range []string{ViewSeason, ViewFranchise} {
			power[view] = a.powerRankings(ctx, standings, sf,
				func(context.Context) []leagueschedule.RawScheduleWeek { return schedule() },
				0, true, m2service.ResolveAggMode("", defaultAgg(view)), view)
		}
	}
	a.core.SeasonMu.Lock()
	defer a.core.SeasonMu.Unlock()
	standingFeed := heldSeasonFeed[[]leaguestandings.RawStanding](a.raceHeld.Snapshot().Standings)
	holdSeasonFeed(&standingFeed, standings, sf, serr)
	a.raceHeld.PublishStandings(raceFeed(standingFeed), racePower(power), serr == nil)
}

// scopedBodies offers only archived bodies whose URL carries param=value, so a one-week or
// one-bracket copy never stands in for another.
func (a *App) scopedBodies(param, value string) func(context.Context, string, func(string, []byte, time.Time) bool) (bool, error) {
	return func(ctx context.Context, part string, accept func(string, []byte, time.Time) bool) (bool, error) {
		return a.core.History.ArchivedBodies(ctx, part, func(src string, body []byte, at time.Time) bool {
			u, err := url.Parse(src)
			if err != nil || u.Query().Get(param) != value {
				return false
			}
			return accept(src, body, at)
		})
	}
}

// bracketsOrArchive reads the bracket list, then each bracket, each live or from the archive.
// The set is as fresh as its stalest part, and any part missing fails the whole set.
func (a *App) bracketsOrArchive(
	ctx context.Context, pause func(context.Context, time.Duration) error,
) ([]leaguefeed.Bracket, Freshness, error) {
	year := strconv.Itoa(a.core.Season)
	list, fresh, err := liveOrArchive(ctx, a.fallbackParent(), a.core.Season, archivedFeed[leaguefeed.Bracket]{
		Export: playoffs.ListExport,
		Fetch: func(ctx context.Context) ([]leaguefeed.Bracket, error) {
			return playoffs.FetchList(ctx, a.core.MFLClient, year, ingestion.LeagueID)
		},
		Parse: playoffs.ParseList, Bodies: a.core.History.ArchivedBodies,
	}, a.core.FallbackLog)
	if err != nil {
		return nil, Freshness{}, fmt.Errorf("playoff brackets: %w", err)
	}
	for i, b := range list {
		if err := pause(ctx, time.Second); err != nil {
			return nil, Freshness{}, fmt.Errorf("playoff brackets: %w", err)
		}
		games, gf, gerr := liveOrArchive(ctx, a.fallbackParent(), a.core.Season, archivedFeed[leaguefeed.BracketGame]{
			Export: playoffs.BracketExport,
			Fetch: func(ctx context.Context) ([]leaguefeed.BracketGame, error) {
				return playoffs.FetchBracket(ctx, a.core.MFLClient, year, ingestion.LeagueID, b.ID)
			},
			Parse:  func(body []byte) ([]leaguefeed.BracketGame, error) { return playoffs.ParseBracket(b.ID, body) },
			Bodies: a.scopedBodies("BRACKET_ID", strconv.Itoa(b.ID)),
		}, a.core.FallbackLog)
		if gerr != nil {
			return nil, Freshness{}, fmt.Errorf("playoff bracket %s: %w", b.Name, gerr)
		}
		list[i].Games = games
		if gf.State != FreshLive {
			fresh = gf
		}
	}
	return list, fresh, nil
}

// raceSnapshot is the joint capture; its callback only snapshots the private holder.
func (a *App) raceSnapshot(snapshot func() season.Held) season.Snapshot {
	a.core.SeasonMu.Lock()
	held := snapshot()
	live := a.core.SeasonLineups
	live.Value = core.CloneLineups(live.Value)
	unplayed := liveWeekUnplayed(live, a.now.schedules, time.Now())
	a.core.SeasonMu.Unlock()
	overlays := map[int]leaguefeed.WeekResult{}
	for _, w := range held.Weeks.Value {
		if w.Week == live.Value.Week {
			overlays[w.Week] = liveWeek(live.Value, w)
		}
	}
	return season.Snapshot{Held: held, Live: raceFeed(live), LiveUnplayed: unplayed, LiveWeeks: overlays}
}

func raceFeed[T any](h heldSeasonFeed[T]) season.Feed[T] {
	return season.Feed[T]{Value: h.Value, FetchedAt: h.FetchedAt, RefreshError: h.RefreshError, ReadPending: h.ReadPending}
}

func racePower(in map[string]PowerRankingsResult) map[string]season.PowerRankingsResult {
	out := make(map[string]season.PowerRankingsResult, len(in))
	for key, p := range in {
		rows := make([]season.PowerRow, len(p.Rows))
		for i, row := range p.Rows {
			rows[i] = season.PowerRow(row)
		}
		out[key] = season.PowerRankingsResult{OK: p.OK, Error: p.Error, View: p.View, Weight: p.Weight, AgeWeight: p.AgeWeight, ModelRunID: p.ModelRunID, AggMode: p.AggMode, StarterN: p.StarterN, Performance: p.Performance, WeeksScored: p.WeeksScored, Rows: rows}
	}
	return out
}
