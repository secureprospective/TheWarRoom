//go:build practice

package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/fantasyscore/source"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/livegame"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/livestats"
	"github.com/secureprospective/TheWarRoom/internal/leagueservice"
	"github.com/secureprospective/TheWarRoom/internal/store/history"
)

// Practice-only test seams for the season box (docs/architecture/season-in-a-box.md, B4). This file is absent
// from release binaries: a release server never reads these variables (backup_hosted.go). Never for production.
const (
	// practiceClockEnv names a file holding one RFC 3339 instant. The server's clock reads that instant plus the
	// wall time since the file was last written, so a test moves time by rewriting the file.
	practiceClockEnv = "TWR_PRACTICE_CLOCK_FILE"
	// practiceOfficialEnv names a directory the official-statistics job reads instead of nflverse:
	// stats_player_week_<season>.csv.gz, play_by_play_<season>.csv.gz and, optionally, players.csv.
	practiceOfficialEnv = "TWR_PRACTICE_OFFICIAL_DIR"
	// practiceLiveEnv names a directory of recorded ESPN summary JSON frames the live score job replays in
	// sorted filename order instead of polling the network. An optional "week" file overrides the summary week.
	practiceLiveEnv = "TWR_PRACTICE_LIVE_DIR"
	// practiceLiveInterval is how often the practice live replay job checks for the next frame.
	practiceLiveInterval = 200 * time.Millisecond
)

// serverClock is the hosted server's one clock: the store, the due worker and the official job read it.
func serverClock() leagueservice.DueClock {
	path := os.Getenv(practiceClockEnv)
	if path == "" {
		return leagueservice.NewHostClock()
	}
	log.Printf("DISPOSABLE PRACTICE BUILD: fake clock from %s; never use for real league state", path)
	return fileClock{path: path}
}

type fileClock struct{ path string }

// Now is the file's instant plus the wall time since it was written. An unreadable file is wall time, logged.
func (c fileClock) Now() time.Time {
	info, err := os.Stat(c.path)
	if err != nil {
		log.Printf("practice clock: %v; using wall time", err)
		return time.Now()
	}
	data, err := os.ReadFile(c.path)
	if err != nil {
		log.Printf("practice clock: %v; using wall time", err)
		return time.Now()
	}
	at, err := time.Parse(time.RFC3339Nano, strings.TrimSpace(string(data)))
	if err != nil {
		log.Printf("practice clock: %v; using wall time", err)
		return time.Now()
	}
	return at.Add(max(time.Since(info.ModTime()), 0))
}

func (fileClock) Trusted() error { return nil }

// officialSource is where the official job reads a season's files, and how soon and how often it reads them.
func officialSource(network officialFiles) (officialFiles, time.Duration, time.Duration) {
	dir := os.Getenv(practiceOfficialEnv)
	if dir == "" {
		return network, officialFirst, officialRefresh
	}
	log.Printf("DISPOSABLE PRACTICE BUILD: official statistics from %s; never use for real league state", dir)
	return directoryFiles(dir), time.Second, time.Second
}

func directoryFiles(dir string) officialFiles {
	return func(_ context.Context, season int) (seasonFiles, error) {
		var out seasonFiles
		var err error
		if out.weekly, err = os.ReadFile(filepath.Join(dir, fmt.Sprintf("stats_player_week_%d.csv.gz", season))); err != nil {
			return out, fmt.Errorf("official weekly statistics: %w", err)
		}
		if out.plays, err = os.ReadFile(filepath.Join(dir, fmt.Sprintf("play_by_play_%d.csv.gz", season))); err != nil {
			return out, fmt.Errorf("official play-by-play: %w", err)
		}
		if out.roster, err = os.ReadFile(filepath.Join(dir, "players.csv")); err != nil && !os.IsNotExist(err) {
			return out, fmt.Errorf("nflverse players: %w", err)
		}
		return out, nil
	}
}

// startPracticeLive replays recorded ESPN summary JSON frames from TWR_PRACTICE_LIVE_DIR in sorted
// filename order instead of polling ESPN, Sleeper or DynastyProcess over the network.
func startPracticeLive(
	ctx context.Context,
	pins *leaguePins,
	publisher scoresPublisher,
	games *leagueservice.LiveGames,
) (func(), <-chan struct{}, bool) {
	dir := os.Getenv(practiceLiveEnv)
	if dir == "" {
		return nil, nil, false
	}
	log.Printf("DISPOSABLE PRACTICE BUILD: live frames from %s; never use for real league state", dir)
	scope, cancel := context.WithCancel(ctx)
	done, restored := make(chan struct{}), make(chan struct{})
	identities := pins.files.family.Over(pins.files.reference)
	now := serverClock().Now
	go func() {
		defer close(done)
		close(restored)
		seen := map[string]bool{}
		for {
			publishPracticeLiveFrame(scope, dir, seen, pins, identities, publisher, games, now)
			timer := time.NewTimer(practiceLiveInterval)
			select {
			case <-scope.Done():
				timer.Stop()
				return
			case <-timer.C:
			}
		}
	}()
	return func() { cancel(); <-done }, restored, true
}

// publishPracticeLiveFrame reads the next unseen *.json ESPN summary in dir, maps athletes through the
// server's player directory, and publishes one live generation to pins.live.
func publishPracticeLiveFrame(
	ctx context.Context,
	dir string,
	seen map[string]bool,
	pins *leaguePins,
	identities *history.Store,
	publisher scoresPublisher,
	games *leagueservice.LiveGames,
	now func() time.Time,
) {
	name, body, ok := nextPracticeLiveFrame(dir, seen)
	if !ok {
		return
	}
	game, err := livegame.ParseSummary(body)
	if err != nil {
		log.Printf("practice live %s: parse summary: %v", name, err)
		return
	}
	season, week := game.Season, game.Week
	if override, has := practiceLiveWeek(dir); has {
		week = override
	}
	lines, ids, err := practiceLiveLines(ctx, identities, game, season, week)
	if err != nil {
		log.Printf("practice live %s: %v", name, err)
		return
	}
	at := now().UTC()
	candidate := history.LiveGeneration{
		ID: liveDigest(body), Season: season, Week: week, At: at,
		Authority: livestats.Authority, Missing: livestats.Pending(), Identities: ids, Lines: lines,
	}
	if err := pins.live.Publish(candidate); err != nil {
		log.Printf("practice live %s: publish: %v", name, err)
		return
	}
	seen[name] = true
	if games != nil {
		games.Set([]livegame.Game{game}, at, "ESPN")
		games.SetDetails(map[string]leagueservice.NFLGame{game.ID: {Game: &game, At: at}})
	}
	if publisher != nil {
		for _, league := range pins.files.leagues() {
			publisher.PublishScores(league, season, week)
		}
	}
	log.Printf("practice live: published %s (%d lines, season %d week %d)", name, len(lines), season, week)
}

func nextPracticeLiveFrame(dir string, seen map[string]bool) (string, []byte, bool) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		log.Printf("practice live dir: %v", err)
		return "", nil, false
	}
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !strings.HasSuffix(name, ".json") || seen[name] {
			continue
		}
		body, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			log.Printf("practice live %s: %v", name, err)
			return "", nil, false
		}
		return name, body, true
	}
	return "", nil, false
}

func practiceLiveWeek(dir string) (int, bool) {
	data, err := os.ReadFile(filepath.Join(dir, "week"))
	if err != nil {
		return 0, false
	}
	week, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil || week < 1 || week > 22 {
		return 0, false
	}
	return week, true
}

func practiceLiveLines(
	ctx context.Context, identities *history.Store, game livegame.Game, season, week int,
) (map[string]source.PlayerWeek, map[string]history.LiveIdentity, error) {
	rawLines, err := livestats.Lines(game)
	if err != nil {
		return nil, nil, fmt.Errorf("extract stat lines: %w", err)
	}
	espn, err := identities.SourceIDs(ctx, "espn")
	if err != nil {
		return nil, nil, fmt.Errorf("read espn ids: %w", err)
	}
	gsis, err := identities.SourceIDs(ctx, "gsis")
	if err != nil {
		return nil, nil, fmt.Errorf("read gsis ids: %w", err)
	}
	sleeper, err := identities.SourceIDs(ctx, "sleeper")
	if err != nil {
		return nil, nil, fmt.Errorf("read sleeper ids: %w", err)
	}
	byESPN := make(map[string]string, len(espn))
	for player, id := range espn {
		byESPN[id] = player
	}
	lines := make(map[string]source.PlayerWeek, len(rawLines))
	ids := make(map[string]history.LiveIdentity, len(rawLines))
	for _, line := range rawLines {
		player, ok := byESPN[line.SourceID]
		if !ok {
			continue
		}
		lines[player] = source.PlayerWeek{Season: season, Week: week, GameID: game.ID, SeasonType: "REG", Line: line.Stats}
		ids[player] = history.LiveIdentity{GSIS: gsis[player], Sleeper: sleeper[player]}
	}
	return lines, ids, nil
}
