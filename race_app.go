package main

import (
	"context"
	"fmt"
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
	pause := a.seasonPause
	if pause == nil {
		pause = seasonPause
	}
	weeks, wf, werr := liveOrArchive(ctx, a.fallbackParent(), a.season, archivedFeed[leaguefeed.WeekResult]{
		export: weeklyresults.Export,
		fetch: func(ctx context.Context) ([]leaguefeed.WeekResult, error) {
			return weeklyresults.Fetch(ctx, a.mflClient, strconv.Itoa(a.season), ingestion.LeagueID)
		},
		parse:  weeklyresults.Parse,
		bodies: a.scopedBodies("W", "YTD"),
	}, a.fallbackLog)
	a.seasonMu.Lock()
	holdSeasonFeed(&a.race.weeks, weeks, wf, werr)
	a.seasonMu.Unlock()
	if err := pause(ctx, time.Second); err != nil {
		return
	}
	brackets, bf, berr := a.bracketsOrArchive(ctx, pause)
	a.seasonMu.Lock()
	holdSeasonFeed(&a.race.brackets, brackets, bf, berr)
	a.seasonMu.Unlock()
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
	a.seasonMu.Lock()
	defer a.seasonMu.Unlock()
	holdSeasonFeed(&a.race.standings, standings, sf, serr)
	if serr == nil {
		a.race.power = power
	}
}

// scopedBodies offers only archived bodies whose URL carries param=value, so a one-week or
// one-bracket copy never stands in for another.
func (a *App) scopedBodies(param, value string) func(context.Context, string, func(string, []byte, time.Time) bool) (bool, error) {
	return func(ctx context.Context, part string, accept func(string, []byte, time.Time) bool) (bool, error) {
		return a.history.ArchivedBodies(ctx, part, func(src string, body []byte, at time.Time) bool {
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
	year := strconv.Itoa(a.season)
	list, fresh, err := liveOrArchive(ctx, a.fallbackParent(), a.season, archivedFeed[leaguefeed.Bracket]{
		export: playoffs.ListExport,
		fetch: func(ctx context.Context) ([]leaguefeed.Bracket, error) {
			return playoffs.FetchList(ctx, a.mflClient, year, ingestion.LeagueID)
		},
		parse: playoffs.ParseList, bodies: a.history.ArchivedBodies,
	}, a.fallbackLog)
	if err != nil {
		return nil, Freshness{}, fmt.Errorf("playoff brackets: %w", err)
	}
	for i, b := range list {
		if err := pause(ctx, time.Second); err != nil {
			return nil, Freshness{}, fmt.Errorf("playoff brackets: %w", err)
		}
		games, gf, gerr := liveOrArchive(ctx, a.fallbackParent(), a.season, archivedFeed[leaguefeed.BracketGame]{
			export: playoffs.BracketExport,
			fetch: func(ctx context.Context) ([]leaguefeed.BracketGame, error) {
				return playoffs.FetchBracket(ctx, a.mflClient, year, ingestion.LeagueID, b.ID)
			},
			parse:  func(body []byte) ([]leaguefeed.BracketGame, error) { return playoffs.ParseBracket(b.ID, body) },
			bodies: a.scopedBodies("BRACKET_ID", strconv.Itoa(b.ID)),
		}, a.fallbackLog)
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
	a.seasonMu.Lock()
	held := a.race
	a.seasonMu.Unlock()
	fresh := true
	for _, f := range []struct {
		at  time.Time
		err string
	}{{held.weeks.fetchedAt, held.weeks.refreshError}, {held.brackets.fetchedAt, held.brackets.refreshError},
		{held.standings.fetchedAt, held.standings.refreshError}} {
		fresh = fresh && f.err == "" && time.Since(f.at) < poolFreshFor
	}
	if !force && fresh {
		return nil
	}
	ctx, cancel := context.WithTimeout(a.ctx, refreshTimeout)
	defer cancel()
	a.refreshMu.Lock()
	defer a.refreshMu.Unlock()
	a.refreshRace(ctx)
	return nil
}

// TargetRace reads held data only. week picks the lineups shown (0: the latest scored week), team
// the franchise the one-team reports follow and mine the GM's own ("": none chosen); Go checks all three.
func (a *App) TargetRace(week int, team, mine string) (RaceBoard, error) {
	if err := a.ready(); err != nil {
		return RaceBoard{}, fmt.Errorf("race: startup: %w", err)
	}
	a.seasonMu.Lock()
	held, live := a.race, a.seasonLineups
	a.seasonMu.Unlock()
	a.lookupMu.Lock()
	lookup, hasLookup := a.lookup, a.hasLookup
	a.lookupMu.Unlock()
	return buildRace(raceInput{held: held, live: live, config: a.rulebook.ActiveConfig(), lookup: lookup, hasLookup: hasLookup,
		week: week, team: team, mine: mine, zone: time.Local})
}
