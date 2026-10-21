package main

import (
	"context"
	"fmt"
	"maps"
	"net/url"
	"strconv"
	"sync"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/ingestion"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/leagueschedule"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/leaguestandings"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/playoffs"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/weeklyresults"
	"github.com/secureprospective/TheWarRoom/internal/leaguefeed"
	"github.com/secureprospective/TheWarRoom/internal/m2service"
)

// raceHeld is League Pulse › Race's held MFL data: the season's weekly results, the playoff
// brackets, MFL's standings (its power rating and the M2 board read them) and the two M2 boards.
type raceHeld struct {
	weeks     heldSeasonFeed[[]leaguefeed.WeekResult]
	brackets  heldSeasonFeed[[]leaguefeed.Bracket]
	standings heldSeasonFeed[[]leaguestandings.RawStanding]
	power     map[string]PowerRankingsResult
}

func (a *App) refreshRace(ctx context.Context) {
	pause := a.core.SeasonPause
	if pause == nil {
		pause = seasonPause
	}
	weeks, wf, werr := liveOrArchive(ctx, a.fallbackParent(), a.core.Season, archivedFeed[leaguefeed.WeekResult]{
		export: weeklyresults.Export,
		fetch: func(ctx context.Context) ([]leaguefeed.WeekResult, error) {
			return weeklyresults.Fetch(ctx, a.core.MFLClient, strconv.Itoa(a.core.Season), ingestion.LeagueID)
		},
		parse:  weeklyresults.Parse,
		bodies: a.scopedBodies("W", "YTD"),
	}, a.core.FallbackLog)
	a.core.SeasonMu.Lock()
	holdSeasonFeed(&a.race.weeks, weeks, wf, werr)
	a.core.SeasonMu.Unlock()
	if err := pause(ctx, time.Second); err != nil {
		return
	}
	brackets, bf, berr := a.bracketsOrArchive(ctx, pause)
	a.core.SeasonMu.Lock()
	holdSeasonFeed(&a.race.brackets, brackets, bf, berr)
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
	holdSeasonFeed(&a.race.standings, standings, sf, serr)
	if serr == nil {
		a.race.power = power
	}
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
		export: playoffs.ListExport,
		fetch: func(ctx context.Context) ([]leaguefeed.Bracket, error) {
			return playoffs.FetchList(ctx, a.core.MFLClient, year, ingestion.LeagueID)
		},
		parse: playoffs.ParseList, bodies: a.core.History.ArchivedBodies,
	}, a.core.FallbackLog)
	if err != nil {
		return nil, Freshness{}, fmt.Errorf("playoff brackets: %w", err)
	}
	for i, b := range list {
		if err := pause(ctx, time.Second); err != nil {
			return nil, Freshness{}, fmt.Errorf("playoff brackets: %w", err)
		}
		games, gf, gerr := liveOrArchive(ctx, a.fallbackParent(), a.core.Season, archivedFeed[leaguefeed.BracketGame]{
			export: playoffs.BracketExport,
			fetch: func(ctx context.Context) ([]leaguefeed.BracketGame, error) {
				return playoffs.FetchBracket(ctx, a.core.MFLClient, year, ingestion.LeagueID, b.ID)
			},
			parse:  func(body []byte) ([]leaguefeed.BracketGame, error) { return playoffs.ParseBracket(b.ID, body) },
			bodies: a.scopedBodies("BRACKET_ID", strconv.Itoa(b.ID)),
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

// TargetRaceRefresh refreshes the race's feeds unless they are fresh; force always refreshes.
func (a *App) TargetRaceRefresh(force bool) error {
	if err := a.ready(); err != nil {
		return fmt.Errorf("race refresh: startup: %w", err)
	}
	a.core.SeasonMu.Lock()
	held := a.race
	a.core.SeasonMu.Unlock()
	fresh := true
	for _, f := range []struct {
		at  time.Time
		err string
	}{{held.weeks.FetchedAt, held.weeks.RefreshError}, {held.brackets.FetchedAt, held.brackets.RefreshError},
		{held.standings.FetchedAt, held.standings.RefreshError}} {
		fresh = fresh && f.err == "" && time.Since(f.at) < poolFreshFor
	}
	if !force && fresh {
		return nil
	}
	ctx, cancel := context.WithTimeout(a.core.Ctx, refreshTimeout)
	defer cancel()
	a.core.RefreshMu.Lock()
	defer a.core.RefreshMu.Unlock()
	a.refreshRace(ctx)
	return nil
}

// TargetRace reads held data only. week picks the lineups shown (0: the latest scored week), team
// the franchise the one-team reports follow and mine the GM's own ("": none chosen); Go checks all three.
func (a *App) TargetRace(week int, team, mine string) (RaceBoard, error) {
	if err := a.ready(); err != nil {
		return RaceBoard{}, fmt.Errorf("race: startup: %w", err)
	}
	in := a.raceReadInput()
	a.core.LookupMu.Lock()
	lookup, hasLookup := a.core.Lookup, a.core.HasLookup
	a.core.LookupMu.Unlock()
	in.config, in.lookup, in.hasLookup = a.core.Rulebook.ActiveConfig(), lookup, hasLookup
	in.week, in.team, in.mine, in.zone = week, team, mine, time.Local
	return buildRace(in)
}

// raceReadInput detaches the mutable index while holding its owner's lock. Feed payloads
// and power rows are published immutably: refresh replaces them, never edits them.
func (a *App) raceReadInput() raceInput {
	a.core.SeasonMu.Lock()
	defer a.core.SeasonMu.Unlock()
	held, live := a.race, a.core.SeasonLineups
	held.power = maps.Clone(held.power)
	return raceInput{held: held, live: live, liveUnplayed: liveWeekUnplayed(live, a.now.schedules, time.Now())}
}
