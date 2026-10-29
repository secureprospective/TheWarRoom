package race

import (
	"context"
	"fmt"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/app/season"
	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/leagueschedule"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/leaguestandings"
	"github.com/secureprospective/TheWarRoom/internal/leaguefeed"
	"github.com/secureprospective/TheWarRoom/internal/m2service"
)

type RefreshSource struct {
	Pause            func(context.Context, time.Duration) error
	Weeks            func(context.Context) ([]leaguefeed.WeekResult, domain.Freshness, error)
	PublishWeeks     func([]leaguefeed.WeekResult, domain.Freshness, error)
	Brackets         func(context.Context) ([]leaguefeed.Bracket, domain.Freshness, error)
	PublishBrackets  func([]leaguefeed.Bracket, domain.Freshness, error)
	Standings        func(context.Context) ([]leaguestandings.RawStanding, domain.Freshness, error)
	Schedule         func(context.Context) []leagueschedule.RawScheduleWeek
	Power            func(context.Context, []leaguestandings.RawStanding, domain.Freshness, func(context.Context) []leagueschedule.RawScheduleWeek, string, string) season.PowerRankingsResult
	PublishStandings func([]leaguestandings.RawStanding, domain.Freshness, error, map[string]season.PowerRankingsResult, bool)
}

func Refresh(parent context.Context, source RefreshSource) {
	ctx, cancel := context.WithTimeout(parent, 2*time.Minute)
	defer cancel()
	weeks, wf, werr := source.Weeks(ctx)
	source.PublishWeeks(weeks, wf, werr)
	if err := source.Pause(ctx, time.Second); err != nil {
		return
	}
	brackets, bf, berr := source.Brackets(ctx)
	source.PublishBrackets(brackets, bf, berr)
	if err := source.Pause(ctx, time.Second); err != nil {
		return
	}
	standings, sf, serr := source.Standings(ctx)
	power := map[string]season.PowerRankingsResult{}
	if serr == nil {
		schedule := source.Schedule(ctx)
		cancel()
		// Archive fallback may succeed after the network budget expires. History reads
		// get their own budget, starting after network reads and the refresh lock.
		boardCtx, boardCancel := context.WithTimeout(parent, 2*time.Minute)
		defer boardCancel()
		for _, view := range []string{season.ViewSeason, season.ViewFranchise} {
			for _, mode := range []string{m2service.AggLineup, m2service.AggSum, m2service.AggTopN} {
				power[season.PowerKey(view, mode)] = source.Power(boardCtx, standings, sf, func(context.Context) []leagueschedule.RawScheduleWeek { return schedule }, mode, view)
			}
		}
	}
	source.PublishStandings(standings, sf, serr, power, serr == nil)
}

func ReadBrackets(ctx context.Context, pause func(context.Context, time.Duration) error,
	readList func(context.Context) ([]leaguefeed.Bracket, domain.Freshness, error),
	readGames func(context.Context, leaguefeed.Bracket) ([]leaguefeed.BracketGame, domain.Freshness, error)) ([]leaguefeed.Bracket, domain.Freshness, error) {
	list, fresh, err := readList(ctx)
	if err != nil {
		return nil, domain.Freshness{}, fmt.Errorf("playoff brackets: %w", err)
	}
	for i, b := range list {
		if err := pause(ctx, time.Second); err != nil {
			return nil, domain.Freshness{}, fmt.Errorf("playoff brackets: %w", err)
		}
		games, gf, gerr := readGames(ctx, b)
		if gerr != nil {
			return nil, domain.Freshness{}, fmt.Errorf("playoff bracket %s: %w", b.Name, gerr)
		}
		list[i].Games = games
		if gf.State != domain.FreshLive {
			fresh = gf
		}
	}
	return list, fresh, nil
}

func RefreshReading(force bool, ready func() error, capture func() season.Held, parent func() context.Context, run func(context.Context)) error {
	if err := ready(); err != nil {
		return fmt.Errorf("race refresh: startup: %w", err)
	}
	held := capture()
	fresh := true
	for _, f := range []struct {
		at  time.Time
		err string
	}{
		{held.Weeks.FetchedAt, held.Weeks.RefreshError},
		{held.Brackets.FetchedAt, held.Brackets.RefreshError},
		{held.Standings.FetchedAt, held.Standings.RefreshError},
	} {
		fresh = fresh && f.err == "" && time.Since(f.at) < 5*time.Minute
	}
	if !force && fresh {
		return nil
	}
	run(parent())
	return nil
}
