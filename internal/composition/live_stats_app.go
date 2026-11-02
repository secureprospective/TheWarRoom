package composition

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	nowapp "github.com/secureprospective/TheWarRoom/internal/app/now"
	"github.com/secureprospective/TheWarRoom/internal/fantasyscore/source"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/livegame"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/livestats"
)

// Live stat polling: while any NFL game is being played, every minute the ESPN box scores of the
// games in progress (and those just finished, until the final feed lands) are read, matched to
// league players through the held id directory, and held as the newest live revision. League
// screens re-read on the moves signal, so a running score moves without a click. Between games it
// checks the scoreboard every few minutes and holds nothing.
const (
	liveStatsBusy = time.Minute
	liveStatsIdle = 5 * time.Minute
)

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
	if a.core.History == nil {
		return liveStatsIdle
	}
	endpoints, err := nowapp.LiveEndpoints()
	if err != nil {
		log.Printf("live stats: %v", err)
		return liveStatsIdle
	}
	fetch := livegame.HTTPFetch(&http.Client{Timeout: 10 * time.Second, Transport: a.core.Network}, endpoints)
	adapter := livegame.Adapter{Fetch: fetch}
	body, err := fetch(ctx, "espn", "")
	if err != nil {
		log.Printf("live stats: scoreboard: %v", err)
		return liveStatsIdle
	}
	games, err := livegame.ParseScoreboard(body)
	if err != nil {
		log.Printf("live stats: scoreboard: %v", err)
		return liveStatsIdle
	}
	busy := false
	for _, g := range games {
		if g.Status == "in" {
			busy = true
		}
	}
	espn, err := a.core.History.SourceIDs(ctx, "espn")
	if err != nil {
		log.Printf("live stats: %v", err)
		return liveStatsIdle
	}
	lines := liveStatLines(ctx, adapter, games, invertIDs(espn), a.core.Season, a.marketWeek(), &a.liveFinals, time.Now())
	// Say so when live lines first land, stop, or move to another week; not on every poll.
	if state := fmt.Sprintf("%t/%d/%d", len(lines) > 0, a.core.Season, a.marketWeek()); state != a.liveStatsSaid {
		a.liveStatsSaid = state
		log.Printf("live stats: %d players from %d games, season %d week %d, %d ESPN ids held",
			len(lines), len(games), a.core.Season, a.marketWeek(), len(espn))
	}
	if len(lines) > 0 {
		a.core.History.SetLiveStats(a.core.Season, a.marketWeek(), time.Now(), livestats.Authority,
			livestats.Pending(), lines)
		if a.core.MovesChanged != nil {
			a.core.MovesChanged(ctx)
		}
	}
	if busy {
		return liveStatsBusy
	}
	return liveStatsIdle
}

// liveFinalReread is how often a finished game's box score is read again: stat corrections after the
// final whistle still arrive, but not every minute.
const liveFinalReread = 10 * time.Minute

// liveFinals holds the box-score lines of finished games between re-reads, keyed by ESPN game id.
// Only the poller goroutine touches it.
type liveFinals map[string]liveFinal

type liveFinal struct {
	at    time.Time
	lines []livestats.Line
}

// liveStatLines reads the box score of every game in progress or finished on the board and keys
// each athlete line by league player id. A finished game's lines are reused for liveFinalReread.
// An athlete the directory cannot place is left out: a missing line stays pending, never a zero.
func liveStatLines(ctx context.Context, adapter livegame.Adapter, games []livegame.Game,
	byESPN map[string]string, season, week int, finals *liveFinals, now time.Time) map[string]source.PlayerWeek {
	if *finals == nil {
		*finals = liveFinals{}
	}
	out := map[string]source.PlayerWeek{}
	for _, g := range games {
		if g.Status == "pre" {
			continue
		}
		held, fresh := (*finals)[g.ID]
		lines := held.lines
		if g.Status != "post" || !fresh || now.Sub(held.at) >= liveFinalReread {
			read, err := gameLines(ctx, adapter, g.ID)
			if err != nil {
				log.Printf("live stats: game %s: %v", g.ID, err)
				if !fresh {
					continue
				}
			} else {
				lines = read
				if g.Status == "post" {
					(*finals)[g.ID] = liveFinal{at: now, lines: read}
				}
			}
		}
		for _, l := range lines {
			player, ok := byESPN[l.SourceID]
			if !ok {
				continue
			}
			out[player] = source.PlayerWeek{Season: season, Week: week, GameID: g.ID, SeasonType: "REG", Line: l.Stats}
		}
	}
	return out
}

func gameLines(ctx context.Context, adapter livegame.Adapter, id string) ([]livestats.Line, error) {
	body, err := adapter.Fetch(ctx, "summary", id)
	if err != nil {
		return nil, err //nolint:wrapcheck // The caller names the game.
	}
	game, err := livegame.ParseSummary(body)
	if err != nil {
		return nil, err //nolint:wrapcheck // The caller names the game.
	}
	return livestats.Lines(game) //nolint:wrapcheck // The caller names the game.
}
func invertIDs(byPlayer map[string]string) map[string]string {
	out := make(map[string]string, len(byPlayer))
	for player, id := range byPlayer {
		out[id] = player
	}
	return out
}
