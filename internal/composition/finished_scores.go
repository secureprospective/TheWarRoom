package composition

import (
	"context"
	"strconv"
	"time"

	nowapp "github.com/secureprospective/TheWarRoom/internal/app/now"
)

func (a *App) refreshFinishedScores(ctx context.Context) {
	a.core.RefreshMu.Lock()
	defer a.core.RefreshMu.Unlock()
	defer a.sourceHealthChanged(ctx)
	week := a.marketWeek() - 1
	if week < 1 {
		return
	}
	a.refreshNowWeek(ctx, week, false)
	a.core.SeasonMu.Lock()
	research := a.snapshotResearch()
	held := research.feeds[researchKey("playerScores", strconv.Itoa(week))]
	schedule := a.now.schedules[week]
	a.core.SeasonMu.Unlock()
	if held.FetchedAt.IsZero() {
		held = a.archivedResearchFeed(ctx, "playerScores", strconv.Itoa(week))
	}
	end, finished := nowapp.FinishedScoreCutoff(nowFeed(schedule), a.timeNow())
	if !finished {
		return
	}
	a.core.SeasonMu.Lock()
	research = a.snapshotResearch()
	if research.finishedAt == nil {
		research.finishedAt = map[int]time.Time{}
	}
	if first := research.finishedAt[week]; !first.IsZero() {
		end = first
	} else {
		research.finishedAt[week] = end
	}
	a.publishResearch(research)
	a.core.SeasonMu.Unlock()
	if !held.FetchedAt.Before(end) {
		return
	}
	pause := a.core.SeasonPause
	if pause == nil {
		pause = seasonPause
	}
	for _, period := range []string{strconv.Itoa(week), "YTD"} {
		if err := pause(ctx, time.Second); err != nil {
			return
		}
		a.refreshResearchFeed(ctx, "playerScores", period, true)
	}
}
