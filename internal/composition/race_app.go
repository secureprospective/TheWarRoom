package composition

import (
	"context"
	"strconv"
	"time"

	nowapp "github.com/secureprospective/TheWarRoom/internal/app/now"
	raceapp "github.com/secureprospective/TheWarRoom/internal/app/race"
	"github.com/secureprospective/TheWarRoom/internal/app/report"
	scoringapp "github.com/secureprospective/TheWarRoom/internal/app/scoring"

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
	raceapp.Refresh(ctx, raceapp.RefreshSource{
		Pause: pause,
		Weeks: func(ctx context.Context) ([]leaguefeed.WeekResult, Freshness, error) {
			return liveOrArchive(ctx, a.fallbackParent(), a.core.Season, archivedFeed[leaguefeed.WeekResult]{
				Export: weeklyresults.Export,
				Fetch: func(ctx context.Context) ([]leaguefeed.WeekResult, error) {
					return weeklyresults.Fetch(ctx, a.core.MFLClient, strconv.Itoa(a.core.Season), ingestion.LeagueID)
				},
				Parse:  weeklyresults.Parse,
				Bodies: a.scopedBodies("W", "YTD"),
			}, a.core.FallbackLog)
		},
		PublishWeeks: func(rows []leaguefeed.WeekResult, fresh Freshness, err error) {
			a.core.SeasonMu.Lock()
			defer a.core.SeasonMu.Unlock()
			feed := heldSeasonFeed[[]leaguefeed.WeekResult](a.raceHeld.Snapshot().Weeks)
			holdSeasonFeed(&feed, rows, fresh, err)
			a.raceHeld.PublishWeeks(raceFeed(feed))
		},
		Brackets: func(ctx context.Context) ([]leaguefeed.Bracket, Freshness, error) {
			return a.bracketsOrArchive(ctx, pause)
		},
		PublishBrackets: func(rows []leaguefeed.Bracket, fresh Freshness, err error) {
			a.core.SeasonMu.Lock()
			defer a.core.SeasonMu.Unlock()
			feed := heldSeasonFeed[[]leaguefeed.Bracket](a.raceHeld.Snapshot().Brackets)
			holdSeasonFeed(&feed, rows, fresh, err)
			a.raceHeld.PublishBrackets(raceFeed(feed))
		},
		Standings: a.standingsOrArchive, Schedule: func(ctx context.Context) []leagueschedule.RawScheduleWeek {
			return scoringapp.RemainingSchedule(ctx, a.leagueScheduleOrArchive)
		},
		Power: func(ctx context.Context, standings []leaguestandings.RawStanding, fresh Freshness, schedule func(context.Context) []leagueschedule.RawScheduleWeek, view string) season.PowerRankingsResult {
			p := a.powerRankings(ctx, standings, fresh, schedule, 0, true, m2service.ResolveAggMode("", defaultAgg(view)), view)
			return racePowerResult(p)
		},
		PublishStandings: func(rows []leaguestandings.RawStanding, fresh Freshness, err error, power map[string]season.PowerRankingsResult, replace bool) {
			a.core.SeasonMu.Lock()
			defer a.core.SeasonMu.Unlock()
			feed := heldSeasonFeed[[]leaguestandings.RawStanding](a.raceHeld.Snapshot().Standings)
			holdSeasonFeed(&feed, rows, fresh, err)
			a.raceHeld.PublishStandings(raceFeed(feed), power, replace)
		},
	})
}

// scopedBodies offers only archived bodies whose URL carries param=value, so a one-week or
// one-bracket copy never stands in for another.
func (a *App) scopedBodies(param, value string) func(context.Context, string, func(string, []byte, time.Time) bool) (bool, error) {
	return report.ScopedBodies(a.core.History.ArchivedBodies, param, value)
}

// bracketsOrArchive reads the bracket list, then each bracket, each live or from the archive.
// The set is as fresh as its stalest part, and any part missing fails the whole set.
func (a *App) bracketsOrArchive(
	ctx context.Context, pause func(context.Context, time.Duration) error,
) ([]leaguefeed.Bracket, Freshness, error) {
	year := strconv.Itoa(a.core.Season)
	//nolint:wrapcheck // Preserve bracket read refusal text.
	return raceapp.ReadBrackets(ctx, pause,
		func(ctx context.Context) ([]leaguefeed.Bracket, Freshness, error) {
			return liveOrArchive(ctx, a.fallbackParent(), a.core.Season, archivedFeed[leaguefeed.Bracket]{
				Export: playoffs.ListExport,
				Fetch: func(ctx context.Context) ([]leaguefeed.Bracket, error) {
					return playoffs.FetchList(ctx, a.core.MFLClient, year, ingestion.LeagueID)
				},
				Parse: playoffs.ParseList, Bodies: a.core.History.ArchivedBodies,
			}, a.core.FallbackLog)
		}, func(ctx context.Context, bracket leaguefeed.Bracket) ([]leaguefeed.BracketGame, Freshness, error) {
			return liveOrArchive(ctx, a.fallbackParent(), a.core.Season, archivedFeed[leaguefeed.BracketGame]{
				Export: playoffs.BracketExport,
				Fetch: func(ctx context.Context) ([]leaguefeed.BracketGame, error) {
					return playoffs.FetchBracket(ctx, a.core.MFLClient, year, ingestion.LeagueID, bracket.ID)
				},
				Parse:  func(body []byte) ([]leaguefeed.BracketGame, error) { return playoffs.ParseBracket(bracket.ID, body) },
				Bodies: a.scopedBodies("BRACKET_ID", strconv.Itoa(bracket.ID)),
			}, a.core.FallbackLog)
		})
}

// raceSnapshot is the joint capture; its callback only snapshots the private holder.
func (a *App) raceSnapshot(snapshot func() season.Held) season.Snapshot {
	a.core.SeasonMu.Lock()
	held := snapshot()
	live := a.core.SeasonLineups
	live.Value = core.CloneLineups(live.Value)
	unplayed := liveWeekUnplayed(live, a.now.schedules, time.Now())
	a.core.SeasonMu.Unlock()
	return nowapp.RaceSnapshot(held, nowFeed(live), unplayed)
}

func raceFeed[T any](h heldSeasonFeed[T]) season.Feed[T] {
	return season.Feed[T]{Value: h.Value, FetchedAt: h.FetchedAt, RefreshError: h.RefreshError, ReadPending: h.ReadPending}
}

func racePowerResult(p PowerRankingsResult) season.PowerRankingsResult {
	rows := make([]season.PowerRow, len(p.Rows))
	for i, row := range p.Rows {
		rows[i] = season.PowerRow(row)
	}
	return season.PowerRankingsResult{OK: p.OK, Error: p.Error, View: p.View, Weight: p.Weight, AgeWeight: p.AgeWeight, PickWeight: p.PickWeight, ModelRunID: p.ModelRunID, AggMode: p.AggMode, StarterN: p.StarterN, Performance: p.Performance, WeeksScored: p.WeeksScored, Rows: rows}
}

type raceHost struct{ *core.State }

func (h raceHost) RefreshRace(force bool, snapshot func() season.Held) error {
	//nolint:wrapcheck // Preserve Race readiness errors.
	return raceapp.RefreshReading(force, h.Ready, func() season.Held { return h.RaceRefreshState(snapshot) }, func() context.Context { return h.Ctx }, func(ctx context.Context) {
		h.RefreshMu.Lock()
		defer h.RefreshMu.Unlock()
		h.RefreshRaceFeeds(ctx)
	})
}
