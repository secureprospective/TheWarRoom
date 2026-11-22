package main

import (
	"context"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/app/now"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/livegame"
	"github.com/secureprospective/TheWarRoom/internal/leagueservice"
)

const (
	preSummaryTTL     = 8 * time.Minute
	preRetryHold      = 2 * time.Minute
	preWeatherTTL     = 30 * time.Minute
	preRequestTimeout = 3 * time.Second
)

type preSummary struct {
	body  []byte
	at    time.Time
	tried time.Time
}

type preDetail struct {
	game      livegame.Game
	at        time.Time
	weatherAt time.Time
}

type pregameJob struct {
	endpoints livegame.Endpoints
	games     []livegame.Game
}

type pregameWork struct {
	board        livegame.Game
	summary      preSummary
	fetchSummary bool
}

// pregameFetcher warms pre-game summaries and weather in one background goroutine so slow
// upstream calls never delay live score ticks.
type pregameFetcher struct {
	client  *http.Client
	weather *now.WeatherCache
	games   *leagueservice.LiveGames

	mu        sync.Mutex
	summaries map[string]preSummary
	details   map[string]preDetail
	pending   *pregameJob
	wake      chan struct{}
	cancel    context.CancelFunc
	done      chan struct{}
}

func (c *liveCollector) ensurePregame(endpoints livegame.Endpoints) *pregameFetcher {
	if c.pregame != nil {
		return c.pregame
	}
	transport := c.packets.network
	if transport == nil {
		transport = http.DefaultTransport
	}
	client := &http.Client{Timeout: preRequestTimeout, Transport: transport}
	weather, err := now.NewWeatherCache(client, endpoints.OpenMeteo)
	if err != nil {
		log.Printf("server live weather init: %v", err)
	}
	c.pregame = &pregameFetcher{
		client:    client,
		weather:   weather,
		games:     c.games,
		summaries: map[string]preSummary{},
		details:   map[string]preDetail{},
		wake:      make(chan struct{}, 1),
	}
	return c.pregame
}

func (c *liveCollector) schedulePregame(
	ctx context.Context, endpoints livegame.Endpoints, games []livegame.Game,
) {
	if c.packets == nil || c.packets.replay || c.finished != [2]int{} {
		return
	}
	pre := make([]livegame.Game, 0, len(games))
	for _, game := range games {
		if game.Status == "pre" && game.SeasonType == 2 {
			pre = append(pre, game)
		}
	}
	if len(pre) == 0 {
		if c.pregame != nil {
			c.pregame.prune(nil)
		}
		return
	}
	c.ensurePregame(endpoints).enqueue(ctx, endpoints, pre)
}

func (c *liveCollector) stopPregame() {
	if c.pregame != nil {
		c.pregame.stop()
	}
}

func (c *liveCollector) cachedPregameDetail(
	board livegame.Game, summaryEndpoint string,
) (leagueservice.NFLGame, bool) {
	if c.pregame != nil {
		if detail, ok := c.pregame.cached(board.ID); ok {
			return detail, true
		}
	}
	var (
		body []byte
		at   time.Time
	)
	if c.packets != nil {
		address := strings.ReplaceAll(summaryEndpoint, "{id}", board.ID)
		if packet, ok := c.packets.held.Packets[address]; ok {
			body, at = packet.Body, packet.At
		} else if packet, ok := c.packets.held.Packets[c.packets.boardURL]; ok {
			at = packet.At
		}
	}
	game, stamp := buildPregameGame(board, body, at)
	return leagueservice.NFLGame{Game: &game, At: stamp}, true
}

func (f *pregameFetcher) enqueue(
	ctx context.Context, endpoints livegame.Endpoints, pre []livegame.Game,
) {
	f.mu.Lock()
	f.pending = &pregameJob{endpoints: endpoints, games: pre}
	if f.cancel == nil {
		workCtx, cancel := context.WithCancel(ctx)
		done := make(chan struct{})
		f.cancel = cancel
		f.done = done
		go f.run(workCtx, done)
	}
	f.mu.Unlock()
	select {
	case f.wake <- struct{}{}:
	default:
	}
}

func (f *pregameFetcher) stop() {
	f.mu.Lock()
	cancel, done := f.cancel, f.done
	f.cancel, f.done = nil, nil
	f.mu.Unlock()
	if cancel != nil {
		cancel()
	}
	if done != nil {
		<-done
	}
}

func (f *pregameFetcher) cached(id string) (leagueservice.NFLGame, bool) {
	f.mu.Lock()
	held, ok := f.details[id]
	f.mu.Unlock()
	if !ok {
		return leagueservice.NFLGame{}, false
	}
	game := held.game
	return leagueservice.NFLGame{Game: &game, At: held.at}, true
}

func (f *pregameFetcher) prune(active map[string]bool) {
	f.mu.Lock()
	defer f.mu.Unlock()
	for id := range f.summaries {
		if !active[id] {
			delete(f.summaries, id)
		}
	}
	for id := range f.details {
		if !active[id] {
			delete(f.details, id)
		}
	}
}

func (f *pregameFetcher) takePending() *pregameJob {
	f.mu.Lock()
	job := f.pending
	f.pending = nil
	f.mu.Unlock()
	return job
}

func (f *pregameFetcher) run(ctx context.Context, done chan struct{}) {
	defer close(done)
	for {
		select {
		case <-ctx.Done():
			return
		case <-f.wake:
			if ctx.Err() != nil {
				return
			}
			if job := f.takePending(); job != nil {
				f.refresh(ctx, *job)
			}
		}
	}
}

func (f *pregameFetcher) refresh(ctx context.Context, job pregameJob) {
	due := f.collectDue(job.games)
	if len(due) == 0 {
		return
	}
	var (
		group     sync.WaitGroup
		updatedMu sync.Mutex
	)
	slots := make(chan struct{}, 4)
	updated := make(map[string]leagueservice.NFLGame, len(due))
	for _, item := range due {
		if ctx.Err() != nil {
			break
		}
		group.Go(func() {
			select {
			case slots <- struct{}{}:
			case <-ctx.Done():
				return
			}
			defer func() { <-slots }()
			if detail, ok := f.refreshOne(ctx, job.endpoints, item); ok {
				updatedMu.Lock()
				updated[item.board.ID] = detail
				updatedMu.Unlock()
			}
		})
	}
	group.Wait()
	if ctx.Err() == nil && len(updated) > 0 {
		f.games.MergePregame(updated)
	}
}

func (f *pregameFetcher) collectDue(games []livegame.Game) []pregameWork {
	active := make(map[string]bool, len(games))
	for _, game := range games {
		active[game.ID] = true
	}
	nowAt := time.Now().UTC()
	f.mu.Lock()
	defer f.mu.Unlock()
	for id := range f.summaries {
		if !active[id] {
			delete(f.summaries, id)
		}
	}
	for id := range f.details {
		if !active[id] {
			delete(f.details, id)
		}
	}
	due := make([]pregameWork, 0, len(games))
	for _, game := range games {
		sum, hasSum := f.summaries[game.ID]
		det, hasDet := f.details[game.ID]
		needSum := summaryDue(sum, hasSum, nowAt)
		needDet := detailDue(det, hasDet, nowAt)
		if !needSum && !needDet {
			continue
		}
		due = append(due, pregameWork{board: game, summary: sum, fetchSummary: needSum})
	}
	return due
}

func summaryDue(held preSummary, ok bool, nowAt time.Time) bool {
	if !ok {
		return true
	}
	if len(held.body) > 0 {
		return nowAt.Sub(held.at) >= preSummaryTTL
	}
	return nowAt.Sub(held.tried) >= preRetryHold
}

func detailDue(held preDetail, ok bool, nowAt time.Time) bool {
	if !ok {
		return true
	}
	if held.game.Weather != nil {
		return nowAt.Sub(held.weatherAt) >= preWeatherTTL
	}
	return nowAt.Sub(held.weatherAt) >= preRetryHold
}

func (f *pregameFetcher) refreshOne(
	ctx context.Context, endpoints livegame.Endpoints, item pregameWork,
) (leagueservice.NFLGame, bool) {
	sum := item.summary
	if item.fetchSummary {
		sum = f.fetchSummary(ctx, endpoints.Summary, item.board.ID, sum)
	}
	if ctx.Err() != nil {
		return leagueservice.NFLGame{}, false
	}
	game, at := buildPregameGame(item.board, sum.body, sum.at)
	if f.weather != nil {
		wCtx, cancel := context.WithTimeout(ctx, preRequestTimeout)
		f.weather.Resolve(wCtx, &game)
		cancel()
	}
	if ctx.Err() != nil {
		return leagueservice.NFLGame{}, false
	}
	f.mu.Lock()
	f.details[item.board.ID] = preDetail{game: game, at: at, weatherAt: time.Now().UTC()}
	f.mu.Unlock()
	return leagueservice.NFLGame{Game: &game, At: at}, true
}

func (f *pregameFetcher) fetchSummary(
	ctx context.Context, summaryEndpoint, id string, prior preSummary,
) preSummary {
	address := strings.ReplaceAll(summaryEndpoint, "{id}", id)
	reqCtx, cancel := context.WithTimeout(ctx, preRequestTimeout)
	body, err := livegame.FetchHTTPS(reqCtx, f.client, address)
	cancel()
	nowAt := time.Now().UTC()
	if err == nil {
		next := preSummary{body: body, at: nowAt, tried: nowAt}
		f.mu.Lock()
		f.summaries[id] = next
		f.mu.Unlock()
		return next
	}
	if ctx.Err() == nil {
		prior.tried = nowAt
		f.mu.Lock()
		f.summaries[id] = prior
		f.mu.Unlock()
	}
	return prior
}

func buildPregameGame(board livegame.Game, body []byte, at time.Time) (livegame.Game, time.Time) {
	game := board
	if len(body) > 0 {
		parsed, err := livegame.ParseSummary(body)
		if err != nil {
			log.Printf("server live pregame detail: %v", err)
		} else if parsed.ID == board.ID {
			if parsed.Season == 0 {
				parsed.Season = board.Season
				parsed.Week = board.Week
				parsed.SeasonType = board.SeasonType
			}
			game = parsed
		}
	}
	if at.IsZero() {
		at = time.Now().UTC()
	}
	return game, at
}
