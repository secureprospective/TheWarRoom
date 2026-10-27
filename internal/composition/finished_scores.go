package composition

import (
	"context"
	"strconv"
	"time"

	nowapp "github.com/secureprospective/TheWarRoom/internal/app/now"
	researchapp "github.com/secureprospective/TheWarRoom/internal/app/research"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/nflschedule"
)

func (a *App) refreshFinishedScores(ctx context.Context) {
	a.core.RefreshMu.Lock()
	defer a.core.RefreshMu.Unlock()
	defer a.sourceHealthChanged(ctx)
	var schedule heldSeasonFeed[[]nflschedule.RawWeek]
	(researchapp.FinishedScoreRefresh{
		Week: a.marketWeek, Schedule: func(ctx context.Context, week int) { a.refreshNowWeek(ctx, week, false) },
		Held: func(week int) time.Time {
			a.core.SeasonMu.Lock()
			defer a.core.SeasonMu.Unlock()
			schedule = a.now.schedules[week]
			return a.snapshotResearch().feeds[researchKey("playerScores", strconv.Itoa(week))].FetchedAt
		},
		Archive: func(ctx context.Context, week int) time.Time {
			return a.archivedResearchFeed(ctx, "playerScores", strconv.Itoa(week)).FetchedAt
		},
		Cutoff: func(_ int) (time.Time, bool) {
			return nowapp.FinishedScoreCutoff(nowFeed(schedule), a.timeNow())
		},
		Pin: func(week int, end time.Time) time.Time {
			a.core.SeasonMu.Lock()
			defer a.core.SeasonMu.Unlock()
			values := a.researchHeld.Snapshot()
			end = researchapp.PinFinished(&values, week, end)
			a.researchHeld.Publish(values)
			return end
		},
		Pause:   a.seasonPauseFunc,
		Refresh: func(ctx context.Context, period string) { a.refreshResearchFeed(ctx, "playerScores", period, true) },
	}).RefreshDue(ctx)
}
