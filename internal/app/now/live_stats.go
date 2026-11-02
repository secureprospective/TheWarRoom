package now

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/fantasyscore/source"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/livegame"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/livestats"
	"github.com/secureprospective/TheWarRoom/internal/store/history"
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

// liveFinalReread is how often a finished game's box score is read again: stat corrections after the
// final whistle still arrive, but not every minute.
const liveFinalReread = 10 * time.Minute

// LiveStatsPoller holds what the live-stats poller keeps between polls. Only the poller goroutine
// touches it.
type LiveStatsPoller struct {
	finals liveFinals // finished games' box scores
	said   string     // live-stats state last logged
}

// liveStatsHistory is the live poll's identity lookup and held-evidence store.
type liveStatsHistory interface {
	SourceIDs(context.Context, string) (map[string]string, error)
	SetLiveStats(int, int, time.Time, string, []string, map[string]source.PlayerWeek)
}

// LiveStatsRun is what one poll reads from the running app.
type LiveStatsRun struct {
	History *history.Store
	Network http.RoundTripper // nil is http.DefaultTransport
	Season  int
	Week    func() int
	Changed func(context.Context) // the moves signal; may be nil
}

// Poll reads the board once, holds any live lines, and says how long to wait before the next poll.
func (p *LiveStatsPoller) Poll(ctx context.Context, run LiveStatsRun) time.Duration {
	if run.History == nil {
		return liveStatsIdle
	}
	return p.poll(ctx, run, run.History)
}

func (p *LiveStatsPoller) poll(ctx context.Context, run LiveStatsRun, held liveStatsHistory) time.Duration {
	season, week, at := run.Season, run.Week(), time.Now()
	endpoints, err := LiveEndpoints()
	if err != nil {
		log.Printf("live stats: %v", err)
		return liveStatsIdle
	}
	fetch := livegame.HTTPFetch(&http.Client{Timeout: 10 * time.Second, Transport: run.Network}, endpoints)
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
	espn, err := held.SourceIDs(ctx, "espn")
	if err != nil {
		log.Printf("live stats: %v", err)
		return liveStatsIdle
	}
	lines, acquired := liveStatLines(ctx, adapter, games, invertIDs(espn), &p.finals, at)
	// Say so when live lines first land, stop, or move to another week; not on every poll.
	if state := fmt.Sprintf("%t/%d/%d", len(lines) > 0, season, week); state != p.said {
		p.said = state
		log.Printf("live stats: %d players from %d games, season %d week %d, %d ESPN ids held",
			len(lines), len(games), season, week, len(espn))
	}
	if acquired {
		held.SetLiveStats(season, week, at, livestats.Authority,
			livestats.Pending(), lines)
		if run.Changed != nil {
			run.Changed(ctx)
		}
	}
	if busy {
		return liveStatsBusy
	}
	return liveStatsIdle
}

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
	byESPN map[string]string, finals *liveFinals, now time.Time) (map[string]source.PlayerWeek, bool) {
	if *finals == nil {
		*finals = liveFinals{}
	}
	out := map[string]source.PlayerWeek{}
	acquired := true
	for _, g := range games {
		if g.Status == "pre" || g.SeasonType == 1 || g.SeasonType == 3 {
			continue
		}
		if g.Season == 0 || g.Week == 0 || g.SeasonType != 2 {
			acquired = false
			continue
		}
		lines, err := finals.game(ctx, adapter, g, now)
		if err != nil {
			log.Printf("live stats: game %s: %v", g.ID, err)
			acquired = false
		}

		for _, l := range lines {
			player, ok := byESPN[l.SourceID]
			if !ok {
				continue
			}
			out[player] = source.PlayerWeek{Season: g.Season, Week: g.Week, GameID: g.ID, SeasonType: "REG", Line: l.Stats}
		}
	}
	return out, acquired
}

func (f *liveFinals) game(ctx context.Context, adapter livegame.Adapter, g livegame.Game, now time.Time) ([]livestats.Line, error) {
	held, fresh := (*f)[g.ID]
	if g.Status == "post" && fresh && now.Sub(held.at) < liveFinalReread {
		return held.lines, nil
	}
	lines, err := gameLines(ctx, adapter, g.ID)
	if err != nil {
		return held.lines, err
	}
	if g.Status == "post" {
		(*f)[g.ID] = liveFinal{at: now, lines: lines}
	}
	return lines, nil
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
