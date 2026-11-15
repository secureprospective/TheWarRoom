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
	"time"

	"github.com/secureprospective/TheWarRoom/internal/app/now"
	"github.com/secureprospective/TheWarRoom/internal/fantasyscore/source"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/livegame"
	"github.com/secureprospective/TheWarRoom/internal/store/history"
)

// scoresPublisher is the L5 publication seam; topic counters belong to serverhttp.
type scoresPublisher interface {
	PublishScores(league string, season, week int)
}

type liveCollector struct {
	pins       *leaguePins
	identities *history.Store
	packets    *livePackets
	path       string
	publisher  scoresPublisher
	poller     now.LiveStatsPoller
	published  bool
	restoring  bool
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
	if c.publisher != nil {
		for _, league := range c.pins.files.leagues() {
			c.publisher.PublishScores(league, season, week)
		}
	}
	return nil
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
	body, err := livegame.HTTPFetch(&http.Client{Timeout: 10 * time.Second, Transport: c.packets}, endpoints)(look, "espn", "")
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
	if c.packets.held.Season != season || c.packets.held.Week != week {
		c.packets.held = rawLiveCheckpoint{Version: 1, Season: season, Week: week, Packets: map[string]rawLivePacket{endpoints.ESPN: packet}, IDs: map[string]map[string]string{}}
	}
	c.packets.prefetched = &packet
	c.poller.Poll(look, now.LiveStatsRun{Evidence: c, RequireComplete: true, Network: c.packets, Season: season, Week: func() int { return week }})
	return now.LiveStatsInterval(games)
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

func startHostedLive(ctx context.Context, pins *leaguePins, root string, publisher scoresPublisher) func() {
	scope, cancel := context.WithCancel(ctx)
	done := make(chan struct{})
	go func() {
		defer close(done)
		directory := filepath.Join(root, "live-scores")
		if err := os.MkdirAll(directory, 0o700); err != nil {
			log.Printf("server live: %v", err)
			return
		}
		c := &liveCollector{pins: pins, identities: pins.files.family.Over(pins.files.reference), path: filepath.Join(directory, "checkpoint.json"), publisher: publisher,
			packets: &livePackets{network: http.DefaultTransport, held: rawLiveCheckpoint{Version: 1, Packets: map[string]rawLivePacket{}, IDs: map[string]map[string]string{}}}}
		if publisher == nil {
			log.Print("server live: L5 scores publisher not connected; notifications unavailable")
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
	return func() { cancel(); <-done }
}
