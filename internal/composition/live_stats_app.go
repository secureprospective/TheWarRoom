package composition

import (
	"context"
	"time"

	nowapp "github.com/secureprospective/TheWarRoom/internal/app/now"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/livegame"
)

// Live stat polling lives in nowapp.LiveStatsPoller; this file only runs it on a worker once
// startup has opened the stores.
func (a *App) startLiveStats() {
	a.startWorker(func() {
		ctx := a.core.WorkCtx
		// The window can be ready before startup has opened the stores. Polling then would find no
		// history and wait the idle interval, holding live points back five minutes after launch.
		select {
		case <-ctx.Done():
			return
		case <-a.core.Started:
		}
		for {
			wait := a.pollLiveStats(ctx)
			select {
			case <-ctx.Done():
				return
			case <-time.After(wait):
			}
		}
	})
}

func (a *App) pollLiveStats(ctx context.Context) time.Duration {
	return a.liveStats.Poll(ctx, nowapp.LiveStatsRun{History: a.core.History, Network: a.core.Network,
		Season: a.core.Season, Week: a.marketWeek, Changed: a.core.MovesChanged,
		Board: func(week int, board livegame.Held) {
			a.core.LiveGameMu.Lock()
			defer a.core.LiveGameMu.Unlock()
			a.nowHeld.PublishScoreboard(week, board)
		}})
}
