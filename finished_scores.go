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
	games := nflGames(schedule.Value, now)
	if len(games) == 0 || !weekKickedOff(games, now) {
		return time.Time{}, false
	}
	end := time.Time{}
	for _, raw := range schedule.Value[0].Matchups {
		kickoff, _ := strconv.ParseInt(raw.Kickoff, 10, 64)
		at := time.Unix(kickoff, 0).Add(finishedGameAllowance)
		switch raw.GameSecondsRemaining {
		case "0":
			at = schedule.FetchedAt
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
	end, finished := finishedScoreCutoff(schedule, a.timeNow())
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
