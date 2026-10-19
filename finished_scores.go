package main

import (
	"context"
	"strconv"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/ingestion/nflschedule"
)

// finishedGameAllowance allows four hours from kickoff when MFL holds no final flag.
const finishedGameAllowance = 4 * time.Hour

func finishedScoreCutoff(schedule heldSeasonFeed[[]nflschedule.RawWeek], now time.Time) (time.Time, bool) {
	games := nflGames(schedule.value, now)
	if len(games) == 0 || !weekKickedOff(games, now) {
		return time.Time{}, false
	}
	end := time.Time{}
	for _, raw := range schedule.value[0].Matchups {
		kickoff, _ := strconv.ParseInt(raw.Kickoff, 10, 64)
		at := time.Unix(kickoff, 0).Add(finishedGameAllowance)
		switch raw.GameSecondsRemaining {
		case "0":
			at = schedule.fetchedAt
		case "":
		default:
			return time.Time{}, false
		}
		if now.Before(at) {
			return time.Time{}, false
		}
		if at.After(end) {
			end = at
		}
	}
	return end, !end.IsZero()
}

func (a *App) refreshFinishedScores(ctx context.Context) {
	a.refreshMu.Lock()
	defer a.refreshMu.Unlock()
	defer a.sourceHealthChanged(ctx)
	week := a.marketWeek() - 1
	if week < 1 {
		return
	}
	a.refreshNowWeek(ctx, week, false)
	a.seasonMu.Lock()
	held := a.research.feeds[researchKey("playerScores", strconv.Itoa(week))]
	schedule := a.now.schedules[week]
	a.seasonMu.Unlock()
	if held.fetchedAt.IsZero() {
		held = a.archivedResearchFeed(ctx, "playerScores", strconv.Itoa(week))
	}
	end, finished := finishedScoreCutoff(schedule, a.timeNow())
	if !finished {
		return
	}
	a.seasonMu.Lock()
	if a.research.finishedAt == nil {
		a.research.finishedAt = map[int]time.Time{}
	}
	if first := a.research.finishedAt[week]; !first.IsZero() {
		end = first
	} else {
		a.research.finishedAt[week] = end
	}
	a.seasonMu.Unlock()
	if !held.fetchedAt.Before(end) {
		return
	}
	pause := a.seasonPause
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
