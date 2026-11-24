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
	"slices"
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
	pregame     *pregameFetcher
	poller      now.LiveStatsPoller
	published   bool
	restoring   bool
	inPoll      bool
	notifyMu    sync.Mutex
	board       []livegame.Game
	scoreDigest string
	notified    string
	// finished is the season and week this collector holds as a finished week; zero follows the current board.
	finished [2]int
	// fetched is when a finished week was last read from the sources; tried is the last attempt.
	fetched, tried time.Time
}

// finishedRefresh is how often last week is reread before this week's first kickoff, so Monday and
// Tuesday stat corrections reach its results.
const finishedRefresh = 6 * time.Hour

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
	publish := c.pins.live.Publish
	if c.finished != [2]int{} {
		publish = c.pins.live.PublishClosed
	}
	if err := publish(candidate); err != nil {
		return fmt.Errorf("publish live overlay: %w", err)
	}
	c.published = true
	// Acquisition timestamps and packet digests change even when football facts do not.
	candidate.ID, candidate.At, candidate.Sources = "", time.Time{}, nil
	body, err := json.Marshal(candidate)
	if err != nil {
		return fmt.Errorf("fingerprint live scores: %w", err)
	}
	c.notifyMu.Lock()
	c.scoreDigest = liveDigest(body)
	c.notifyMu.Unlock()
	if !c.inPoll {
		c.notifyScores(season, week)
	}
	return nil
}

func (c *liveCollector) notifiedDigest() string {
	c.notifyMu.Lock()
	defer c.notifyMu.Unlock()
	return c.notified
}

func (c *liveCollector) boardSnapshot() []livegame.Game {
	c.notifyMu.Lock()
	defer c.notifyMu.Unlock()
	return slices.Clone(c.board)
}

func (c *liveCollector) notifyScores(season, week int) {
	c.notifyMu.Lock()
	var pregame []leagueservice.NFLGamePregame
	if c.games != nil {
		pregame = c.games.PregameSnapshot()
	}
	body, err := json.Marshal(struct {
		Games   []livegame.Game
		Scores  string
		Pregame []leagueservice.NFLGamePregame
	}{c.board, c.scoreDigest, pregame})
	if err != nil {
		c.notifyMu.Unlock()
		log.Printf("server live notification: %v", err)
		return
	}
	digest := liveDigest(body)
	if digest == c.notified {
		c.notifyMu.Unlock()
		return
	}
	c.notified = digest
	c.notifyMu.Unlock()
	if c.publisher != nil && c.pins != nil {
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
	defer c.logTick(time.Now(), c.notifiedDigest())()
	c.inPoll = true
	defer func() { c.inPoll = false }()
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
	scoreboard := ""
	if c.finished != [2]int{} {
		scoreboard = finishedBoard(endpoints.ESPN, c.finished[0], c.finished[1])
		endpoints.ESPN = scoreboard
	}
	c.packets.boardURL = endpoints.ESPN
	games, season, week, seasonType, ok := c.readBoard(look, endpoints)
	if !ok {
		return 5 * time.Minute
	}
	packet := c.packets.held.Packets[endpoints.ESPN]
	previousBoard := c.boardSnapshot()
	c.holdBoard(games, packet.At)
	if seasonType == 2 && (c.packets.held.Season != season || c.packets.held.Week != week) {
		c.packets.held = rawLiveCheckpoint{
			Version: 1, Season: season, Week: week,
			Packets: map[string]rawLivePacket{endpoints.ESPN: packet},
			IDs:     map[string]map[string]string{},
		}
	}
	c.prepareSummaries(look, endpoints, games, previousBoard)
	if seasonType == 2 {
		c.packets.prefetched = &packet
		c.poller.Poll(look, now.LiveStatsRun{
			Evidence: c, RequireComplete: true, Network: c.packets, Client: c.client,
			Season: season, Week: func() int { return week }, Scoreboard: scoreboard,
		})
	}
	c.finishTick(ctx, games, endpoints, season, week)
	return c.interval(games)
}

// readBoard reads the scoreboard and the one regular-season or playoff week it shows. A
// finished-week collector takes the board only when it is that regular-season week and every game
// on it is final.
func (c *liveCollector) readBoard(
	ctx context.Context, endpoints livegame.Endpoints,
) ([]livegame.Game, int, int, int, bool) {
	body, err := livegame.HTTPFetch(c.client, endpoints)(ctx, "espn", "")
	if err != nil {
		log.Printf("server live scoreboard: %v", err)
		return nil, 0, 0, 0, false
	}
	games, err := livegame.ParseScoreboard(body)
	if err != nil {
		log.Printf("server live scoreboard: %v", err)
		return nil, 0, 0, 0, false
	}
	if len(games) > 16 {
		return nil, 0, 0, 0, false
	}
	season, week, seasonType, ok := boardScope(games, c.finished == [2]int{})
	if !ok {
		return nil, 0, 0, 0, false
	}
	if c.finished != [2]int{} && (seasonType != 2 || c.finished != [2]int{season, week} || !allFinal(games)) {
		log.Printf("server live: week %d/%d is not finished on the board; not held",
			c.finished[0], c.finished[1])
		return nil, 0, 0, 0, false
	}
	return games, season, week, seasonType, true
}

func boardScope(games []livegame.Game, allowPlayoffs bool) (int, int, int, bool) {
	if season, week, ok := uniformBoardScope(games, 2); ok {
		return season, week, 2, true
	}
	if allowPlayoffs {
		if season, week, ok := uniformBoardScope(games, 3); ok {
			return season, week, 3, true
		}
	}
	return 0, 0, 0, false
}

func uniformBoardScope(games []livegame.Game, seasonType int) (int, int, bool) {
	season, week := 0, 0
	for _, game := range games {
		if game.SeasonType != seasonType {
			continue
		}
		if season != 0 && (season != game.Season || week != game.Week) {
			log.Print("server live: mixed scoreboard scope")
			return 0, 0, false
		}
		season, week = game.Season, game.Week
	}
	return season, week, season != 0 && week != 0
}

// finishTick holds cached game details, publishes scores, then triggers background pre-game warming.
func (c *liveCollector) finishTick(
	ctx context.Context, games []livegame.Game, endpoints livegame.Endpoints, season, week int,
) {
	if c.finished == [2]int{} {
		c.holdDetails(games, endpoints)
	}
	c.notifyScores(season, week)
	c.schedulePregame(ctx, endpoints, games, season, week)
}

func (c *liveCollector) ensureClient() {
	if c.client == nil {
		c.client = &http.Client{Timeout: 10 * time.Second, Transport: c.packets}
	}
}

func (c *liveCollector) logTick(started time.Time, previous string) func() {
	return func() {
		if previous != c.notifiedDigest() {
			log.Printf("server live: tick %d changed=1", time.Since(started).Milliseconds())
		}
	}
}

func (c *liveCollector) interval(games []livegame.Game) time.Duration {
	delay := hostedLiveInterval(games, time.Now())
	if delay == 5*time.Second && c.packets.boardInterval > delay {
		return c.packets.boardInterval
	}
	return delay
}

// hostedLiveInterval polls every five seconds while a game is on. An idle board waits no longer than the next
// kickoff, and a game past its kickoff but not yet started on ESPN is polled as live, so the first snap is
// never up to an idle wait late.
func hostedLiveInterval(games []livegame.Game, at time.Time) time.Duration {
	const busy = 5 * time.Second
	if now.LiveStatsInterval(games) == time.Minute {
		return busy
	}
	delay := now.LiveStatsInterval(games)
	for _, game := range games {
		if game.Status != "pre" || game.Kickoff.IsZero() {
			continue
		}
		delay = min(delay, max(game.Kickoff.Sub(at), busy))
	}
	return delay
}

// holdBoard keeps the parsed scoreboard for nfl.games.read, stamped with the packet's read time.
func (c *liveCollector) holdBoard(games []livegame.Game, at time.Time) {
	if at.IsZero() {
		at = time.Now()
	}
	c.notifyMu.Lock()
	c.board = games
	c.notifyMu.Unlock()
	if c.finished == [2]int{} {
		c.games.Set(games, at, "ESPN")
	}
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

func startHostedLive(
	ctx context.Context,
	pins *leaguePins,
	root string,
	publisher scoresPublisher,
	games *leagueservice.LiveGames,
) (func(), <-chan struct{}) {
	if stop, restored, ok := startPracticeLive(ctx, pins, publisher, games); ok {
		return stop, restored
	}
	scope, cancel := context.WithCancel(ctx)
	done, linked, restored := make(chan struct{}), make(chan struct{}), make(chan struct{})
	go func() {
		defer close(linked)
		keepLiveIdentities(scope, pins.files.family.Over(pins.files.reference))
	}()
	go func() {
		defer close(done)
		var restoreOnce sync.Once
		markRestored := func() { restoreOnce.Do(func() { close(restored) }) }
		defer markRestored()
		directory := filepath.Join(root, "live-scores")
		if err := os.MkdirAll(directory, 0o700); err != nil {
			log.Printf("server live: %v", err)
			return
		}
		c := &liveCollector{pins: pins, identities: pins.files.family.Over(pins.files.reference), path: filepath.Join(directory, "checkpoint.json"), publisher: publisher, games: games,
			packets: &livePackets{network: http.DefaultTransport, held: rawLiveCheckpoint{Version: 1, Packets: map[string]rawLivePacket{}, IDs: map[string]map[string]string{}}}}
		defer c.stopPregame()
		if publisher == nil {
			log.Print("server live: no scores publisher; change streams will not hear about live scores")
		}
		finished := restoreFinished(scope, c, directory)
		if err := c.restore(scope); err != nil {
			log.Printf("server live restart: %v", err)
		}
		markRestored()
		for {
			delay := c.poll(scope)
			c.keepLastWeek(scope, directory, finished)
			timer := time.NewTimer(delay)
			select {
			case <-scope.Done():
				timer.Stop()
				return
			case <-timer.C:
			}
		}
	}()
	return func() { cancel(); <-done; <-linked }, restored
}

// holdDetails reuses acquired summary packets and cached pre-game details without network I/O.
func (c *liveCollector) holdDetails(games []livegame.Game, endpoints livegame.Endpoints) {
	details := make(map[string]leagueservice.NFLGame)
	for _, board := range games {
		if board.Status == "pre" {
			if detail, ok := c.cachedPregameDetail(board, endpoints.Summary); ok {
				details[board.ID] = detail
			}
			continue
		}
		packet, ok := c.packets.held.Packets[strings.ReplaceAll(endpoints.Summary, "{id}", board.ID)]
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
			address := strings.ReplaceAll(endpoints.Summary, "{id}", board.ID)
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
		if game.Status != "in" || (game.SeasonType != 2 && game.SeasonType != 3) {
			continue
		}
		address := strings.ReplaceAll(endpoints.Summary, "{id}", game.ID)
		if !reflect.DeepEqual(old[game.ID], game) {
			delete(c.packets.nextSummary, address)
		}
	}
	for _, game := range games {
		if game.Status != "in" || (game.SeasonType != 2 && game.SeasonType != 3) {
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

// finishedBoard is ESPN's scoreboard for one named regular-season week.
func finishedBoard(board string, season, week int) string {
	return fmt.Sprintf("%s?seasontype=2&week=%d&dates=%d", board, week, season)
}

func allFinal(games []livegame.Game) bool {
	for _, game := range games {
		if game.SeasonType == 2 && game.Status != "post" {
			return false
		}
	}
	return len(games) != 0
}

func finishedPath(directory string, season, week int) string {
	return filepath.Join(directory, fmt.Sprintf("finished-%d-%02d.json", season, week))
}

// finishedCollector reads one finished week through the same acquisition as the current week, into
// its own checkpoint, and publishes it as a held week rather than the current one.
func (c *liveCollector) finishedCollector(directory string, season, week int) *liveCollector {
	return &liveCollector{pins: c.pins, identities: c.identities, path: finishedPath(directory, season, week),
		publisher: c.publisher, finished: [2]int{season, week},
		packets: &livePackets{network: c.packets.network, held: rawLiveCheckpoint{Version: 1, Packets: map[string]rawLivePacket{}, IDs: map[string]map[string]string{}}}}
}

// restoreFinished replays every finished week held beside the current checkpoint, so last week's
// results read the same after a restart, without the network.
func restoreFinished(ctx context.Context, current *liveCollector, directory string) map[[2]int]*liveCollector {
	out := map[[2]int]*liveCollector{}
	paths, err := filepath.Glob(filepath.Join(directory, "finished-*.json"))
	if err != nil {
		log.Printf("server live finished weeks: %v", err)
		return out
	}
	for _, path := range paths {
		held, err := readLiveCheckpoint(path)
		if err != nil {
			log.Printf("server live finished week %s: %v", filepath.Base(path), err)
			continue
		}
		f := current.finishedCollector(directory, held.Season, held.Week)
		if f.path != path {
			log.Printf("server live finished week %s: names a different week", filepath.Base(path))
			continue
		}
		if err := f.restore(ctx); err != nil {
			log.Printf("server live finished week %d/%d: %v", held.Season, held.Week, err)
			continue
		}
		out[f.finished] = f
	}
	return out
}

// keepLastWeek holds the week before the board's week as a finished week. The pinned football
// evidence carries no statistics, so this is what keeps last week's results once the board moves on.
// It is read once when missing and reread every finishedRefresh until this week's first kickoff.
func (c *liveCollector) keepLastWeek(ctx context.Context, directory string, finished map[[2]int]*liveCollector) {
	board := c.boardSnapshot()
	if len(board) > 0 && !slices.ContainsFunc(board, func(g livegame.Game) bool {
		return g.SeasonType == 2
	}) {
		return
	}
	season, week := c.packets.held.Season, c.packets.held.Week
	if season == 0 || week < 2 {
		return
	}
	last := [2]int{season, week - 1}
	f := finished[last]
	if f == nil {
		f = c.finishedCollector(directory, last[0], last[1])
		finished[last] = f
	}
	kicked := slices.ContainsFunc(board, func(g livegame.Game) bool {
		return g.SeasonType == 2 && g.Status != "pre"
	})
	if (c.pins.live.Closed(last[0], last[1]) && (kicked || time.Since(f.fetched) < finishedRefresh)) ||
		time.Since(f.tried) < 5*time.Minute {
		return
	}
	f.published, f.tried = false, time.Now()
	f.poll(ctx)
	if f.published {
		f.fetched = time.Now()
		log.Printf("server live: held finished week %d/%d", last[0], last[1])
	}
}
