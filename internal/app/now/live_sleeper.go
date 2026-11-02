package now

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/fantasyscore"
	"github.com/secureprospective/TheWarRoom/internal/fantasyscore/source"
	"github.com/secureprospective/TheWarRoom/internal/ingestion"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/livegame"
)

type rawSleeperEndpoint struct {
	Sleeper string `json:"sleeper"`
}

func (r rawSleeperEndpoint) Validate() error {
	u, err := url.Parse(r.Sleeper)
	if err != nil {
		return fmt.Errorf("sleeper endpoint: %w", err)
	}
	if u.Scheme != "https" || u.Host == "" || u.User != nil || !strings.Contains(r.Sleeper, "{season}") || !strings.Contains(r.Sleeper, "{week}") {
		return fmt.Errorf("sleeper endpoint needs HTTPS and season/week placeholders")
	}
	return nil
}

// fillLiveSleeper is optional acquisition: a failed gap source never discards ESPN evidence.
func fillLiveSleeper(ctx context.Context, run LiveStatsRun, held liveStatsHistory, games []livegame.Game, lines map[string]source.PlayerWeek) {
	scopes := map[[2]int]bool{}
	for _, game := range games {
		if game.SeasonType == 2 && game.Season > 0 && game.Week > 0 && game.Status != "pre" {
			scopes[[2]int{game.Season, game.Week}] = true
		}
	}
	if len(scopes) == 0 {
		return
	}
	ids, err := held.SourceIDs(ctx, "sleeper")
	if err != nil {
		log.Printf("live stats: Sleeper identities: %v", err)
		return
	}
	var endpoint rawSleeperEndpoint
	if err := json.Unmarshal(liveGameSources, &endpoint); err != nil {
		log.Printf("live stats: Sleeper endpoint: %v", err)
		return
	}
	if err := endpoint.Validate(); err != nil {
		log.Printf("live stats: %v", err)
		return
	}
	client := &http.Client{Timeout: 10 * time.Second, Transport: run.Network}
	for scope := range scopes {
		address := strings.NewReplacer("{season}", strconv.Itoa(scope[0]), "{week}", strconv.Itoa(scope[1])).Replace(endpoint.Sleeper)
		body, err := ingestion.Get(ctx, client, address, 4<<20)
		if err != nil {
			log.Printf("live stats: Sleeper: %v", err)
			continue
		}
		gaps, err := source.LiveSleeperLines(source.RawSleeperWeek{Season: scope[0], Week: scope[1], Body: string(body)})
		if err != nil {
			log.Printf("live stats: %v", err)
			continue
		}
		mergeLiveSleeper(lines, gaps, invertIDs(ids), scope[0], scope[1])
	}
}

func mergeLiveSleeper(lines map[string]source.PlayerWeek, gaps map[string]fantasyscore.StatLine, bySleeper map[string]string, season, week int) {
	for id, gap := range gaps {
		player, known := bySleeper[id]
		if !known {
			continue
		}
		live, exists := lines[player]
		if exists && (live.Season != season || live.Week != week) {
			continue
		}
		inputs := map[string]fantasyscore.StatLine{source.Sleeper: gap}
		if exists {
			inputs[source.ESPNLive] = live.Line
		} else {
			live = source.PlayerWeek{Season: season, Week: week, SeasonType: "REG"}
		}
		line, owners := source.MergeStatSources(inputs)
		live.Line, live.Provenance = line, source.LiveAuthority(owners)
		lines[player] = live
	}
}
