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

const preSummaryTTL = 8 * time.Minute

type preSummary struct {
	body []byte
	at   time.Time
}

func (c *liveCollector) ensurePreClient() {
	if c.preClient != nil || c.packets == nil || c.packets.replay {
		return
	}
	transport := c.packets.network
	if transport == nil {
		transport = http.DefaultTransport
	}
	c.preClient = &http.Client{Timeout: 10 * time.Second, Transport: transport}
}

func (c *liveCollector) ensureWeather(endpoints livegame.Endpoints) *now.WeatherCache {
	if c.weather != nil {
		return c.weather
	}
	var transport livegame.Transport
	if c.packets == nil || !c.packets.replay {
		transport = c.preClient
	}
	cache, err := now.NewWeatherCache(transport, endpoints.NWS)
	if err != nil {
		log.Printf("server live weather init: %v", err)
		return nil
	}
	c.weather = cache
	return cache
}

// preparePreSummaries warms ESPN summaries for upcoming games outside the live-scoring checkpoint,
// so a pre-game fetch failure never blocks live score publication or exhausts checkpoint slots.
func (c *liveCollector) preparePreSummaries(
	ctx context.Context, endpoints livegame.Endpoints, games []livegame.Game,
) {
	if c.packets.replay || c.preClient == nil || c.finished != [2]int{} {
		return
	}
	if c.preSummaries == nil {
		c.preSummaries = map[string]preSummary{}
	}
	active := map[string]bool{}
	due := make([]livegame.Game, 0, len(games))
	nowAt := time.Now().UTC()
	for _, game := range games {
		if game.Status != "pre" || game.SeasonType != 2 {
			continue
		}
		active[game.ID] = true
		if held, ok := c.preSummaries[game.ID]; ok && nowAt.Sub(held.at) < preSummaryTTL {
			continue
		}
		due = append(due, game)
	}
	for id := range c.preSummaries {
		if !active[id] {
			delete(c.preSummaries, id)
		}
	}
	if len(due) == 0 {
		return
	}
	var (
		mu    sync.Mutex
		group sync.WaitGroup
	)
	slots := make(chan struct{}, 4)
	for _, game := range due {
		address := strings.ReplaceAll(endpoints.Summary, "{id}", game.ID)
		id := game.ID
		group.Go(func() {
			slots <- struct{}{}
			defer func() { <-slots }()
			body, err := livegame.FetchHTTPS(ctx, c.preClient, address)
			if err != nil {
				return
			}
			mu.Lock()
			c.preSummaries[id] = preSummary{body: body, at: time.Now().UTC()}
			mu.Unlock()
		})
	}
	group.Wait()
}

func (c *liveCollector) pregameDetail(
	ctx context.Context, board livegame.Game, summaryEndpoint string, weather *now.WeatherCache,
) (leagueservice.NFLGame, bool) {
	var (
		body []byte
		at   time.Time
	)
	if held, ok := c.preSummaries[board.ID]; ok {
		body, at = held.body, held.at
	} else if packet, ok := c.packets.held.Packets[strings.ReplaceAll(summaryEndpoint, "{id}", board.ID)]; ok {
		body, at = packet.Body, packet.At
	}
	game := board
	if len(body) > 0 {
		parsed, err := livegame.ParseSummary(body)
		if err != nil {
			log.Printf("server live pregame detail: %v", err)
		} else if parsed.ID == board.ID {
			if parsed.Season == 0 {
				parsed.Season, parsed.Week, parsed.SeasonType = board.Season, board.Week, board.SeasonType
			}
			game = parsed
		}
	}
	if at.IsZero() {
		if packet, ok := c.packets.held.Packets[c.packets.boardURL]; ok {
			at = packet.At
		} else {
			at = time.Now().UTC()
		}
	}
	if weather != nil {
		weather.Resolve(ctx, &game)
	}
	return leagueservice.NFLGame{Game: &game, At: at}, true
}
