package composition

import (
	"context"
	"fmt"
	"log"
	"slices"
	"strconv"
	"time"

	controlapp "github.com/secureprospective/TheWarRoom/internal/app/control"

	"github.com/secureprospective/TheWarRoom/internal/ingestion/nflschedule"
	"github.com/secureprospective/TheWarRoom/internal/leagueclock"
	"github.com/secureprospective/TheWarRoom/internal/leagueweek"
)

func (a *App) refreshWeeksInBackground(parent context.Context) {
	ticker := time.NewTicker(time.Hour)
	defer ticker.Stop()
	(controlapp.WeekWorker{Timeout: refreshTimeout, Week: a.refreshWeek, Season: a.refreshSeason,
		Finished: a.refreshFinishedScores, Engine: a.refreshEngine, Changed: a.core.ClockChanged,
		Launch: a.launchRefresh, LastRegular: a.lastRegularWeek,
		Schedule: func(ctx context.Context, week int) { a.refreshNowWeek(ctx, week, false) },
	}).Run(parent, ticker.C)
}

func (a *App) refreshWeek(ctx context.Context) {
	a.core.RefreshMu.Lock()
	defer a.core.RefreshMu.Unlock()
	(controlapp.WeekRefresh{Now: a.timeNow, Read: a.fetchLineupWeek,
		Publish: func(w leagueweek.Week, fetched time.Time, err error) {
			a.core.WeekMu.Lock()
			defer a.core.WeekMu.Unlock()
			controlapp.HoldWeek(&a.core.Week, &a.core.WeekFetchedAt, &a.core.WeekRefreshError, w, fetched, err)
		}, Derive: a.deriveWeekPhase, Revalidate: a.revalidateReleaseAfterRefresh, Changed: a.core.ClockChanged,
		Log: func(text string) { log.Print(text) },
	}).Run(ctx)
}

func (a *App) fetchLineupWeek(ctx context.Context, now time.Time) (leagueweek.Week, error) {
	year := strconv.Itoa(a.core.League.Season())
	return controlapp.LineupWeekFrom(now, func(requested int) (nflschedule.RawWeek, error) { //nolint:wrapcheck // The service retains the week refresh context.
		return nflschedule.Fetch(ctx, a.core.MFLClient, year, requested)
	})
}

// seedCurrentWeek applies the live refresh's finality rule to held schedules, without fetching.
func (a *App) seedCurrentWeek(ctx context.Context) {
	controlapp.SeedWeek(a.timeNow(), func(requested int) ([]nflschedule.RawWeek, time.Time, bool, error) {
		return readArchivedFeed(ctx, a.core.Season, archivedFeed[nflschedule.RawWeek]{
			Export: nflschedule.Export, Leagueless: true,
			Parse: func(body []byte) ([]nflschedule.RawWeek, error) {
				raw, err := nflschedule.Parse(body)
				if err != nil {
					return nil, fmt.Errorf("held NFL week: %w", err)
				}
				return []nflschedule.RawWeek{raw}, nil
			},
			Bodies: a.scopedBodies("W", controlapp.LineupWeekParam(requested)),
		})
	}, func(w leagueweek.Week, fetched time.Time) {
		a.core.WeekMu.Lock()
		defer a.core.WeekMu.Unlock()
		a.core.Week, a.core.WeekFetchedAt, a.core.WeekRefreshError = w, fetched, ""
	}, a.core.FallbackLog)
}

func (a *App) deriveWeekPhase(ctx context.Context, w leagueweek.Week, fetched time.Time) {
	controlapp.DeriveWeekPhase(ctx, w, fetched, a.core.Rulebook.ActiveConfig(), a.core.Whatif.CurrentPhase, a.core.Coordinator.Execute, func(text string) { log.Print(text) })
}

func (a *App) weekSnapshot() controlapp.WeekSnapshot {
	a.core.WeekMu.Lock()
	defer a.core.WeekMu.Unlock()
	w := a.core.Week
	w.Games = slices.Clone(w.Games)
	return controlapp.WeekSnapshot{Week: w, FetchedAt: a.core.WeekFetchedAt, RefreshError: a.core.WeekRefreshError}
}

func (a *App) weekClock(now time.Time) (*leagueclock.LineupLock, Freshness) {
	return controlapp.WeekClock(a.weekSnapshot(), now)
}

func (a *App) lastRegularWeek() int {
	return controlapp.LastRegularWeek(a.core.Rulebook.ActiveConfig().LastRegularSeasonWeek)
}

func (a *App) windowSnapshot() controlapp.WindowSnapshot {
	last := a.lastRegularWeek()
	a.core.SeasonMu.Lock()
	defer a.core.SeasonMu.Unlock()
	return controlapp.CaptureWindows(last, a.now.schedules)
}

func (a *App) windowInputs(now time.Time) leagueclock.WindowInputs {
	return controlapp.WindowInputs(a.windowSnapshot(), now)
}

// timeNow is the league clock's current time: the wall clock unless a test pins it.
func (a *App) timeNow() time.Time {
	return controlapp.TimeNow(a.core.ClockNow)
}
