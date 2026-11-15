package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"maps"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/app/now"
	"github.com/secureprospective/TheWarRoom/internal/fantasyscore/source"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/livegame"
	"github.com/secureprospective/TheWarRoom/internal/leagueservice"
	"github.com/secureprospective/TheWarRoom/internal/serverhttp"
	"github.com/secureprospective/TheWarRoom/internal/store/history"
)

// scoresPublisher is the L5 publication seam; topic counters belong to serverhttp.
type scoresPublisher interface {
	PublishScores(league string, season, week int)
}

// The hosted server hands its one shared Publisher to the collector, so a new live generation
// reaches every open change stream as a scores topic.
var _ scoresPublisher = (*serverhttp.Publisher)(nil)

type liveCollector struct {
	pins       *leaguePins
	identities *history.Store
	packets    *livePackets
	path       string
	publisher  scoresPublisher
	// games holds the latest parsed scoreboard for nfl.games.read; nil keeps it unset.
	games       *leagueservice.LiveGames
	client      *http.Client
	poller      now.LiveStatsPoller
	published   bool
	restoring   bool
	board       []livegame.Game
	scoreDigest string
	notified    string
}

func (c *liveCollector) SourceIDs(ctx context.Context, kind string) (map[string]string, error) {
	if c.packets.replay {
		return maps.Clone(c.packets.held.IDs[kind]), nil
	}
	ids, err := c.identities.SourceIDs(ctx, kind)
	if err != nil {
		return nil, fmt.Errorf("read shared live identities: %w", err)
	}
	if err := validateLiveIDs(kind, ids); err != nil {
		return nil, err
	}
	c.packets.held.IDs[kind] = ids
	return ids, nil
}

func (c *liveCollector) SetLiveStats(ctx context.Context, season, week int, _ time.Time, authority string, missing []string, lines map[string]source.PlayerWeek) error {
	if c.packets.failed != nil {
		return fmt.Errorf("live acquisition incomplete: %w", c.packets.failed)
	}
	if err := ctx.Err(); err != nil {
		return fmt.Errorf("publish live scores: %w", err)
	}
	gsis, err := c.SourceIDs(ctx, "gsis")
	if err != nil {
		return err
	}
	candidate := history.LiveGeneration{Season: season, Week: week, Authority: authority, Missing: missing, Lines: lines, Reduced: len(c.packets.held.Reduced) > 0, Sources: map[string]time.Time{}, Identities: map[string]history.LiveIdentity{}}
	for player := range lines {
		candidate.Identities[player] = history.LiveIdentity{GSIS: gsis[player], Sleeper: c.packets.held.IDs["sleeper"][player]}
	}
	for _, packet := range c.packets.held.Packets {
		candidate.Sources[packet.URL] = packet.At
		if candidate.At.IsZero() || packet.At.Before(candidate.At) {
			candidate.At = packet.At
		}
	}
	candidate.ID = "candidate"
	if err := candidate.Validate(); err != nil {
		return fmt.Errorf("validate live generation: %w", err)
	}
	digest, err := c.checkpoint()
	if err != nil {
		return err
	}
	candidate.ID = digest
	if err := c.pins.live.Publish(candidate); err != nil {
		return fmt.Errorf("publish live overlay: %w", err)
	}
	c.published = true
	// Acquisition timestamps and packet digests change even when football facts do not.
	candidate.ID, candidate.At, candidate.Sources = "", time.Time{}, nil
	body, err := json.Marshal(candidate)
	if err != nil {
		return fmt.Errorf("fingerprint live scores: %w", err)
	}
	c.scoreDigest = liveDigest(body)
	c.notifyScores(season, week)
	return nil
}

func (c *liveCollector) notifyScores(season, week int) {
	body, err := json.Marshal(struct {
		Games  []livegame.Game
		Scores string
	}{c.board, c.scoreDigest})
	if err != nil {
		log.Printf("server live notification: %v", err)
		return
	}
	digest := liveDigest(body)
	if digest == c.notified {
		return
	}
	c.notified = digest
	if c.publisher != nil {
		for _, league := range c.pins.files.leagues() {
			c.publisher.PublishScores(league, season, week)
		}
	}
}

func (c *liveCollector) checkpoint() (string, error) {
	if !c.restoring {
		return writeLiveCheckpoint(c.path, c.packets.held)
	}
	body, err := json.Marshal(c.packets.held)
	if err != nil {
		return "", fmt.Errorf("fingerprint live checkpoint: %w", err)
	}
	return liveDigest(body), nil
}

func (c *liveCollector) poll(ctx context.Context) time.Duration {
	defer c.logTick(time.Now(), c.notified)()
	c.ensureClient()
	look, cancel := context.WithTimeout(ctx, 90*time.Second)
	defer cancel()
	c.packets.failed = nil
	if !c.packets.replay {
		c.packets.held.Reduced = nil
	}
	endpoints, err := now.LiveEndpoints()
	if err != nil {
		log.Printf("server live: %v", err)
		return 5 * time.Minute
	}
	c.packets.boardURL = endpoints.ESPN
	body, err := livegame.HTTPFetch(c.client, endpoints)(look, "espn", "")
	if err != nil {
		log.Printf("server live scoreboard: %v", err)
		return 5 * time.Minute
	}
	games, err := livegame.ParseScoreboard(body)
	if err != nil {
		log.Printf("server live scoreboard: %v", err)
		return 5 * time.Minute
	}
	season, week := 0, 0
	for _, game := range games {
		if game.SeasonType != 2 {
			continue
		}
		if season != 0 && (season != game.Season || week != game.Week) {
			log.Print("server live: mixed scoreboard scope")
			return 5 * time.Minute
		}
		season, week = game.Season, game.Week
	}
	if season == 0 || week == 0 || len(games) > 16 {
		return 5 * time.Minute
	}
	packet := c.packets.held.Packets[endpoints.ESPN]
	previousBoard := c.board
	c.holdBoard(games, packet.At)
	if c.packets.held.Season != season || c.packets.held.Week != week {
		c.packets.held = rawLiveCheckpoint{Version: 1, Season: season, Week: week, Packets: map[string]rawLivePacket{endpoints.ESPN: packet}, IDs: map[string]map[string]string{}}
	}
	c.prepareSummaries(look, endpoints, games, previousBoard)
	c.packets.prefetched = &packet
	c.poller.Poll(look, now.LiveStatsRun{Evidence: c, RequireComplete: true, Network: c.packets, Client: c.client, Season: season, Week: func() int { return week }})
	c.finishTick(games, endpoints.Summary, season, week)
	return c.interval(games)
}

// finishTick holds each live game detail for nfl.game.read, then publishes scores if anything changed.
func (c *liveCollector) finishTick(games []livegame.Game, summary string, season, week int) {
	c.holdDetails(games, summary)
	c.notifyScores(season, week)
}

func (c *liveCollector) ensureClient() {
	if c.client == nil {
		c.client = &http.Client{Timeout: 10 * time.Second, Transport: c.packets}
	}
}

func (c *liveCollector) logTick(started time.Time, previous string) func() {
	return func() {
		if previous != c.notified {
			log.Printf("server live: tick %d changed=1", time.Since(started).Milliseconds())
		}
	}
}

func (c *liveCollector) interval(games []livegame.Game) time.Duration {
	delay := hostedLiveInterval(games)
	if delay == 5*time.Second && c.packets.boardInterval > delay {
		return c.packets.boardInterval
	}
	return delay
}

func hostedLiveInterval(games []livegame.Game) time.Duration {
	if now.LiveStatsInterval(games) == time.Minute {
		return 5 * time.Second
	}
	return now.LiveStatsInterval(games)
}

// holdBoard keeps the parsed scoreboard for nfl.games.read, stamped with the packet's read time.
func (c *liveCollector) holdBoard(games []livegame.Game, at time.Time) {
	if at.IsZero() {
		at = time.Now()
	}
	c.board = games
	c.games.Set(games, at, "ESPN")
}

func (c *liveCollector) restore(ctx context.Context) error {
	checkpoint, err := readLiveCheckpoint(c.path)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	c.packets.held = checkpoint
	c.packets.replay = true
	c.restoring = true
	defer func() { c.packets.replay = false; c.restoring = false }()
	c.poll(ctx)
	if !c.published {
		return fmt.Errorf("live checkpoint could not rebuild a generation")
	}
	return nil
}

func startHostedLive(ctx context.Context, pins *leaguePins, root string, publisher scoresPublisher, games *leagueservice.LiveGames) func() {
	scope, cancel := context.WithCancel(ctx)
	done, linked := make(chan struct{}), make(chan struct{})
	go func() {
		defer close(linked)
		keepLiveIdentities(scope, pins.files.family.Over(pins.files.reference))
	}()
	go func() {
		defer close(done)
		directory := filepath.Join(root, "live-scores")
		if err := os.MkdirAll(directory, 0o700); err != nil {
			log.Printf("server live: %v", err)
			return
		}
		c := &liveCollector{pins: pins, identities: pins.files.family.Over(pins.files.reference), path: filepath.Join(directory, "checkpoint.json"), publisher: publisher, games: games,
			packets: &livePackets{network: http.DefaultTransport, held: rawLiveCheckpoint{Version: 1, Packets: map[string]rawLivePacket{}, IDs: map[string]map[string]string{}}}}
		if publisher == nil {
			log.Print("server live: no scores publisher; change streams will not hear about live scores")
		}
		if err := c.restore(scope); err != nil {
			log.Printf("server live restart: %v", err)
		}
		for {
			delay := c.poll(scope)
			timer := time.NewTimer(delay)
			select {
			case <-scope.Done():
				timer.Stop()
				return
			case <-timer.C:
			}
		}
	}()
	return func() { cancel(); <-done; <-linked }
}

// holdDetails reuses acquired summary packets; no extra requests or cadence changes.
func (c *liveCollector) holdDetails(games []livegame.Game, endpoint string) {
	details := make(map[string]leagueservice.NFLGame)
	for _, board := range games {
		packet, ok := c.packets.held.Packets[strings.ReplaceAll(endpoint, "{id}", board.ID)]
		if !ok {
			continue
		}
		game, err := livegame.ParseSummary(packet.Body)
		if err != nil {
			log.Printf("server live detail: %v", err)
			continue
		}
		if game.ID != board.ID || game.Season != board.Season || game.Week != board.Week {
			continue
		}
		var behind bool
		if game, behind = boardAhead(game, board); behind {
			// Refetch next tick instead of reusing a summary the board has already passed.
			address := strings.ReplaceAll(endpoint, "{id}", board.ID)
			c.packets.mu.Lock()
			delete(c.packets.nextSummary, address)
			c.packets.mu.Unlock()
		}
		details[game.ID] = leagueservice.NFLGame{Game: &game, At: packet.At}
	}
	c.games.SetDetails(details)
}

// ESPN's summary trails its scoreboard by a few seconds. When the board is further along, show the
// board's score and clock on the followed game so the field never lags the game list.
func boardAhead(summary, board livegame.Game) (livegame.Game, bool) {
	if gameProgress(board) <= gameProgress(summary) {
		return summary, false
	}
	summary.Home.Score, summary.Away.Score = board.Home.Score, board.Away.Score
	summary.Status, summary.Clock, summary.Period = board.Status, board.Clock, board.Period
	return summary, true
}

// gameProgress orders snapshots of one game: final last, then period, game clock (counts down), points.
func gameProgress(game livegame.Game) int {
	final := 0
	if game.Status == "post" {
		final = 1
	}
	return final<<30 + game.Period<<22 + (4095-clockSeconds(game.Clock))<<10 + min(game.Home.Score+game.Away.Score, 1023)
}

// clockSeconds reads "m:ss"; anything else counts as a full clock so it never looks further along.
func clockSeconds(clock string) int {
	minutes, seconds, ok := strings.Cut(clock, ":")
	if !ok {
		return 4095
	}
	m, err1 := strconv.Atoi(minutes)
	s, err2 := strconv.Atoi(seconds)
	if err1 != nil || err2 != nil || m < 0 || s < 0 {
		return 4095
	}
	return min(m*60+s, 4095)
}

// Warm live summaries with bounded I/O; the reducer then reads these same admitted packets.
func (c *liveCollector) prepareSummaries(ctx context.Context, endpoints livegame.Endpoints, games, previous []livegame.Game) {
	if c.packets.replay {
		return
	}
	old := make(map[string]livegame.Game, len(previous))
	for _, game := range previous {
		old[game.ID] = game
	}
	var group sync.WaitGroup
	slots := make(chan struct{}, 4)
	for _, game := range games {
		if game.Status != "in" || game.SeasonType != 2 {
			continue
		}
		address := strings.ReplaceAll(endpoints.Summary, "{id}", game.ID)
		if !reflect.DeepEqual(old[game.ID], game) {
			delete(c.packets.nextSummary, address)
		}
	}
	for _, game := range games {
		if game.Status != "in" || game.SeasonType != 2 {
			continue
		}
		address := strings.ReplaceAll(endpoints.Summary, "{id}", game.ID)
		group.Go(func() {
			slots <- struct{}{}
			defer func() { <-slots }()
			request, err := http.NewRequestWithContext(ctx, http.MethodGet, address, nil)
			if err != nil {
				return
			}
			response, err := c.client.Do(request)
			if err != nil {
				return
			}
			_ = response.Body.Close()
		})
	}
	group.Wait()
}
